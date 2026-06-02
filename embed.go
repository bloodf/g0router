package g0router

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed ui/dist
var uiDist embed.FS

func UI() (fs.FS, error) {
	ui, err := fs.Sub(uiDist, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("open embedded ui: %w", err)
	}
	return ui, nil
}
