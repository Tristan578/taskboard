package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/Tristan578/taskboard/internal/cli"
)

//go:embed web/dist
var webEmbed embed.FS

func init() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	case "info", "":
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

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
