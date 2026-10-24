package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) connectionSettings(w http.ResponseWriter, r *http.Request) {
	executable, database, err := s.connectionPaths()
	if err != nil {
		writeError(w, 500, "Could not locate this installation and database.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"mcpServers": map[string]any{"player2": map[string]any{"command": executable, "args": []string{"--db", database, "mcp"}}}})
}

func (s *Server) connectionPaths() (string, string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	if strings.EqualFold(filepath.Base(executable), "Player2.exe") {
		candidate := filepath.Join(filepath.Dir(executable), "player2-kanban.exe")
		if _, err := os.Stat(candidate); err == nil {
			executable = candidate
		}
	}
	database, err := s.store.DatabasePath()
	return executable, database, err
}
