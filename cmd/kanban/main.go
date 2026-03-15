package main

import (
	"embed"
	"io/fs"

	"github.com/Tristan578/taskboard/internal/cli"
)

//go:embed web/dist
var webEmbed embed.FS

// webFS extracts the web/dist sub-filesystem from the embedded FS.
// Returns nil if the sub-filesystem cannot be created.
func webFS() fs.FS {
	sub, err := fs.Sub(webEmbed, "web/dist")
	if err != nil {
		return nil
	}
	return sub
}

func main() {
	cli.Execute(webFS())
}
