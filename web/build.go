// Package web embeds the built single-page management UI (the Vite bundle in
// dist/) into the harbrr binary. The production bundle is committed so Go
// applications embedding harbrr do not need Bun in their own build pipeline.
package web

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist returns the embedded dist directory with the "dist/" prefix stripped,
// so files resolve as "index.html", "assets/…".
func Dist() (fs.FS, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("web: embedded dist: %w", err)
	}
	return sub, nil
}
