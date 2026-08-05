package proxy

// Tests for the /admin static handler after it moved from direct filesystem
// reads to an fs.FS. The interesting properties are the ones that only break
// once assets live inside the binary: relative names must resolve without a
// web/dist directory on disk, traversal must not escape the asset tree, and
// embedded files (which have no mtime) must still be cacheable.

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// withAssetFS installs fsys for the duration of a test and restores whatever was
// there before, so tests do not leak an asset tree into each other.
func withAssetFS(t *testing.T, fsys fs.FS) {
	t.Helper()
	assetMu.Lock()
	prevFS, prevEmbedded := assetFS, embeddedAssets
	assetMu.Unlock()
	t.Cleanup(func() {
		assetMu.Lock()
		assetFS, embeddedAssets = prevFS, prevEmbedded
		assetMu.Unlock()
	})
	SetAssetFS(fsys)
}

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html":            {Data: []byte("<!doctype html><title>spa</title>")},
		"check.html":            {Data: []byte("<!doctype html><title>check</title>")},
		"kiro.svg":              {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		"assets/main-abc123.js": {Data: []byte("console.log('x')")},
	}
}

func TestServeStaticFromAssetFS(t *testing.T) {
	withAssetFS(t, testAssets())
	h := &Handler{}

	for _, tc := range []struct {
		path     string
		wantCode int
		wantType string
	}{
		{"/admin/", http.StatusOK, "text/html; charset=utf-8"},
		{"/admin/kiro.svg", http.StatusOK, "image/svg+xml"},
		{"/admin/assets/main-abc123.js", http.StatusOK, "text/javascript; charset=utf-8"},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if tc.path == "/admin/" {
			h.serveAdminPage(w, r)
		} else {
			h.serveStaticFile(w, r)
		}
		if w.Code != tc.wantCode {
			t.Errorf("%s: code = %d, want %d", tc.path, w.Code, tc.wantCode)
		}
		if got := w.Header().Get("Content-Type"); got != tc.wantType {
			t.Errorf("%s: content-type = %q, want %q", tc.path, got, tc.wantType)
		}
	}
}

// A route with no file extension is a client-side React Router path, so the SPA
// entry point must be served instead of a 404 — otherwise a page reload on
// /admin/accounts breaks.
func TestServeStaticFileSPAFallback(t *testing.T) {
	withAssetFS(t, testAssets())
	h := &Handler{}

	w := httptest.NewRecorder()
	h.serveStaticFile(w, httptest.NewRequest(http.MethodGet, "/admin/accounts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "<!doctype html><title>spa</title>" {
		t.Fatalf("body = %q, want index.html", body)
	}
}

// A missing asset that looks like a real file must 404 rather than silently
// returning the SPA shell, or a broken script tag turns into an HTML parse error.
func TestServeStaticFileMissingAsset404(t *testing.T) {
	withAssetFS(t, testAssets())
	h := &Handler{}

	w := httptest.NewRecorder()
	h.serveStaticFile(w, httptest.NewRequest(http.MethodGet, "/admin/nope.js", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", w.Code)
	}
}

// Traversal attempts must never read outside the asset tree. Before the move to
// fs.FS these concatenated straight into an os.Open, so "../../etc/passwd" was a
// real file read; now the cleaned name is resolved inside the FS, which cannot
// represent anything above its root.
func TestServeStaticFileRejectsTraversal(t *testing.T) {
	withAssetFS(t, testAssets())
	h := &Handler{}

	for _, p := range []string{
		"/admin/../../etc/passwd",
		"/admin/..%2f..%2fetc%2fpasswd",
		"/admin/../../../../../../etc/hosts",
		"/admin//etc/passwd",
	} {
		w := httptest.NewRecorder()
		h.serveStaticFile(w, httptest.NewRequest(http.MethodGet, p, nil))
		// Nothing at these names exists inside the asset FS, so the only correct
		// outcomes are a 404 or the SPA shell (extension-less paths). Any other
		// body means a real host file was read.
		if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
			t.Errorf("%s: unexpected code %d", p, w.Code)
		}
		if body := w.Body.String(); body != "" &&
			body != "<!doctype html><title>spa</title>" &&
			w.Code == http.StatusOK {
			t.Errorf("%s: served unexpected content %q", p, body)
		}
	}

	// The name-cleaning contract that makes the above safe: whatever survives
	// must be a valid, non-rooted fs.FS path.
	for _, rel := range []string{"../../etc/passwd", "/etc/passwd", "a/../../b"} {
		clean := cleanAssetName(rel)
		if clean != "" && !fs.ValidPath(clean) {
			t.Errorf("cleanAssetName(%q) = %q, not a valid fs path", rel, clean)
		}
	}
}

// Embedded assets report a zero mtime, so http.ServeContent cannot emit
// Last-Modified and every reload would be a full 200. The ETag is what restores
// conditional requests.
func TestServeStaticEmbeddedSetsETagAndRevalidates(t *testing.T) {
	withAssetFS(t, testAssets())
	h := &Handler{}

	w := httptest.NewRecorder()
	h.serveStaticFile(w, httptest.NewRequest(http.MethodGet, "/admin/kiro.svg", nil))
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on embedded asset")
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/admin/kiro.svg", nil)
	r2.Header.Set("If-None-Match", etag)
	h.serveStaticFile(w2, r2)
	if w2.Code != http.StatusNotModified {
		t.Fatalf("revalidate code = %d, want 304", w2.Code)
	}
}

// Text assets are gzipped when the client allows it; already-compressed types
// are not, since re-compressing them only burns CPU.
func TestServeStaticGzipsOnlyCompressibleTypes(t *testing.T) {
	assets := testAssets()
	assets["icon.png"] = &fstest.MapFile{Data: []byte("\x89PNG\r\n\x1a\nnot-really-a-png-but-long-enough-to-compress")}
	withAssetFS(t, assets)
	h := &Handler{}

	for path, wantGzip := range map[string]bool{
		"/admin/assets/main-abc123.js": true,
		"/admin/kiro.svg":              true,
		"/admin/icon.png":              false,
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("Accept-Encoding", "gzip")
		h.serveStaticFile(w, r)
		gotGzip := w.Header().Get("Content-Encoding") == "gzip"
		if gotGzip != wantGzip {
			t.Errorf("%s: gzip = %v, want %v", path, gotGzip, wantGzip)
		}
	}
}

// The public key-check page lives outside /admin but is served from the same
// asset tree, so its name must be relative to the dist root too.
func TestServeCheckPageFromAssetFS(t *testing.T) {
	withAssetFS(t, testAssets())
	h := &Handler{}

	w := httptest.NewRecorder()
	h.serveStatic(w, httptest.NewRequest(http.MethodGet, "/check/key", nil), "check.html")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "<!doctype html><title>check</title>" {
		t.Fatalf("body = %q, want check.html", body)
	}
}

// SetAssetFS(nil) must be a no-op rather than blanking the tree: main() calls it
// unconditionally and a binary built without the frontend must keep its disk
// fallback.
func TestSetAssetFSIgnoresNil(t *testing.T) {
	withAssetFS(t, testAssets())
	SetAssetFS(nil)

	h := &Handler{}
	w := httptest.NewRecorder()
	h.serveAdminPage(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d after SetAssetFS(nil), want 200", w.Code)
	}
}
