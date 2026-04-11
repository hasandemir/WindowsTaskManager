package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html style.css app.js
var staticFiles embed.FS

// FS returns the embedded static UI filesystem.
func FS() fs.FS {
	return staticFiles
}
