package proxy

// static.go serves the admin web assets under /admin/ with gzip compression and
// cache headers. It replaces the bare http.ServeFile calls so text assets are
// compressed on the wire and the browser can cache immutable-ish files.
//
// Design notes:
//   - Assets come from an fs.FS, not the filesystem directly. In a release build
//     that FS is the SPA embedded into the binary (see web/embed.go), so /admin
//     works no matter which directory the process was started from — required
//     for the global `kiroproxy` install. Tests and `go run` from the repo fall
//     back to reading web/dist off disk.
//   - Compression is done by gzipping the file into an in-memory buffer and
//     handing that to http.ServeContent via a bytes.Reader. This keeps the
//     Content-Length correct (the gzipped size) and preserves ServeContent's
//     Last-Modified / If-Modified-Since (304) handling for free — which a
//     naive gzip.Writer wrapper around ServeFile would break.
//   - Only compressible text types are gzipped; woff2/ico/png are already
//     compressed and pass through untouched.
//   - Cache-Control is short and paired with the ?v=<timestamp> cache-busting
//     the HTML already appends, so a redeploy still busts caches.

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"kiro-go/config"
)

// staticContentTypes maps file extensions to their Content-Type. Extensions not
// listed fall back to Go's sniffing inside http.ServeContent.
var staticContentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".png":   "image/png",
	".woff2": "font/woff2",
	".woff":  "font/woff",
}

// staticCompressible is the set of extensions worth gzipping. Binary assets
// (woff2/ico/png) are already compressed and are intentionally excluded.
var staticCompressible = map[string]bool{
	".html": true,
	".css":  true,
	".js":   true,
	".mjs":  true,
	".json": true,
	".svg":  true,
}

// staticCacheControl returns the Cache-Control value for a given extension.
// HTML is revalidated every load (it is the cache-bust entry point); other
// assets get a short max-age since the HTML appends ?v=<timestamp> to them.
func staticCacheControl(ext string) string {
	switch ext {
	case ".html":
		return "no-cache"
	case ".woff2", ".woff", ".ico", ".png":
		return "public, max-age=604800" // 7 days — fonts/icons rarely change
	default:
		return "public, max-age=3600" // 1 hour — busted by ?v= on redeploy
	}
}

// diskDistRoot is where the Vite build lands in a repo checkout. Used only as
// the fallback FS when no embedded assets were installed via SetAssetFS.
const diskDistRoot = "web/dist"

// assetFS is the tree /admin is served from, guarded because main installs the
// embedded build at startup while tests may swap it while a server is live.
//
// embeddedAssets records whether the active FS is the embedded build: it has
// zero mtimes, so ServeContent cannot do If-Modified-Since and we substitute a
// content-derived ETag in that case only.
var (
	assetMu        sync.RWMutex
	assetFS        fs.FS = os.DirFS(diskDistRoot)
	embeddedAssets bool
)

// SetAssetFS installs the admin asset tree, rooted so that "index.html" resolves
// to the SPA entry point. main() calls this with the embedded build; passing nil
// or never calling it leaves the on-disk web/dist fallback in place.
func SetAssetFS(f fs.FS) {
	if f == nil {
		return
	}
	assetMu.Lock()
	defer assetMu.Unlock()
	assetFS = f
	embeddedAssets = true
}

func currentAssetFS() (fs.FS, bool) {
	assetMu.RLock()
	defer assetMu.RUnlock()
	return assetFS, embeddedAssets
}

func (h *Handler) serveAdminPage(w http.ResponseWriter, r *http.Request) {
	h.serveStatic(w, r, "index.html")
}

// cleanAssetName turns a request-relative path into a name safe to open in an
// fs.FS, or "" when it cannot be one. Rooting the path first means path.Clean
// collapses every ".." against "/" rather than escaping upward, and the leading
// "/" is then stripped because fs.FS rejects rooted names.
func cleanAssetName(rel string) string {
	clean := strings.TrimPrefix(path.Clean("/"+rel), "/")
	if clean == "" || !fs.ValidPath(clean) {
		return ""
	}
	return clean
}

func (h *Handler) serveStaticFile(w http.ResponseWriter, r *http.Request) {
	clean := cleanAssetName(strings.TrimPrefix(r.URL.Path, "/admin/"))
	if clean == "" {
		http.NotFound(w, r)
		return
	}

	// SPA fallback: a path with no file extension is a client-side React Router
	// route (e.g. /admin/accounts), not a real asset — serve index.html so the
	// SPA can boot and route. Real missing assets (with an extension) still 404
	// via serveStatic.
	if path.Ext(clean) == "" {
		fsys, _ := currentAssetFS()
		if _, err := fs.Stat(fsys, clean); err != nil {
			h.serveStatic(w, r, "index.html")
			return
		}
	}
	h.serveStatic(w, r, clean)
}

// serveStatic writes the asset at name (a slash-separated path relative to the
// dist root) with the right Content-Type, Cache-Control, and gzip when the
// client accepts it and the type is compressible.
func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request, name string) {
	fsys, embedded := currentAssetFS()
	if fsys == nil || !fs.ValidPath(name) {
		http.NotFound(w, r)
		return
	}

	f, err := fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}

	ext := strings.ToLower(path.Ext(name))
	if ct := staticContentTypes[ext]; ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", staticCacheControl(ext))
	// Embedded files carry no mtime, so ServeContent would skip Last-Modified and
	// every reload would be a full 200. A version+path ETag restores 304s: the
	// asset names are content-hashed by Vite, and index.html changes with the
	// binary version.
	if embedded {
		w.Header().Set("ETag", staticETag(name))
	}

	if staticCompressible[ext] && clientAcceptsGzip(r) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := io.Copy(gz, f); err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		if err := gz.Close(); err != nil {
			http.Error(w, "compress error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		// ServeContent sets Content-Length from the reader and handles 304 via modtime.
		http.ServeContent(w, r, name, stat.ModTime(), bytes.NewReader(buf.Bytes()))
		return
	}

	rs, ok := f.(io.ReadSeeker)
	if !ok {
		// fs.File is not required to be seekable; ServeContent needs that for
		// range requests, so buffer the (small) asset instead.
		body, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, name, stat.ModTime(), bytes.NewReader(body))
		return
	}
	http.ServeContent(w, r, name, stat.ModTime(), rs)
}

// staticETag derives a stable validator from the binary version and the asset
// path. Vite content-hashes asset filenames, so path+version changing is exactly
// when the bytes change.
func staticETag(name string) string {
	sum := sha256.Sum256([]byte(config.Version + "\x00" + name))
	return `"` + hex.EncodeToString(sum[:12]) + `"`
}

// clientAcceptsGzip reports whether the request's Accept-Encoding allows gzip.
func clientAcceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]), "gzip") {
			return true
		}
	}
	return false
}
