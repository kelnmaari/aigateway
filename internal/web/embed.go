package web

import (
	"embed"
	"io/fs"
)

// LegacyFiles contains the legacy HTML/JS frontend
// Files are in internal/web/static/ (copied from web/ during build)
//
//go:embed all:static
var LegacyFiles embed.FS

// SvelteFiles contains the new Svelte frontend (when built)
// Files are in internal/web/svelte-build/ (copied from web-svelte/build/ during build)
// Note: This will be empty if Svelte is not built
//
//go:embed all:svelte-build
var SvelteFiles embed.FS

// GetStaticFS returns the appropriate static files based on version
// version: "legacy" for HTML/JS frontend, "svelte" for Svelte frontend
func GetStaticFS(version string) (fs.FS, error) {
	switch version {
	case "svelte":
		return fs.Sub(SvelteFiles, "svelte-build")
	case "legacy":
		fallthrough
	default:
		return fs.Sub(LegacyFiles, "static")
	}
}

// HasSvelteUI checks if Svelte UI is available
func HasSvelteUI() bool {
	entries, err := fs.ReadDir(SvelteFiles, "svelte-build")
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// HasLegacyUI checks if Legacy UI is available
func HasLegacyUI() bool {
	entries, err := fs.ReadDir(LegacyFiles, "static")
	if err != nil {
		return false
	}
	return len(entries) > 0
}
