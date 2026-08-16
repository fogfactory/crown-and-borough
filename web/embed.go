package webassets

import (
	"embed"
	"io/fs"
)

// dist contains the Vite output. The Docker build creates it before the Go
// compiler packages this directory.
//
//go:embed all:dist
var embeddedDist embed.FS

// EmbeddedFS returns the frontend files without the dist/ prefix.
func EmbeddedFS() fs.FS {
	files, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		panic("webassets: embedded dist directory is missing")
	}
	return files
}
