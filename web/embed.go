package web

// embed.go bundles the built admin SPA into the Go binary.
//
// Why: the global `kiroproxy` install has no repo checkout, so the binary must
// carry its own web assets — serving them from a relative "web/dist" path only
// works when the process happens to run from the repository root.

import (
	"embed"
	"io/fs"
)

// distFS holds the Vite build output (web/frontend build.outDir = web/dist).
//
// The all: prefix matters: Vite emits chunks and public files whose names may
// start with "_" or ".", which a plain //go:embed silently skips.
//
//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded dist tree rooted at web/dist (so callers open
// "index.html", not "dist/index.html"). The second return value is false when
// the build had no dist directory — callers then fall back to reading from disk.
func DistFS() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	// A dist/ that only holds the committed placeholder means the frontend was
	// never built into this binary; disk fallback is more useful than serving
	// the placeholder.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
