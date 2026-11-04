package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5"
)

func shellQuote(value, platform string) string {
	if platform == "windows" {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func setupCommands(executable, database, platform string) map[string]any {
	digest := sha256.Sum256([]byte(database))
	name := fmt.Sprintf("player2-%x", digest[:6])
	launch := shellQuote(executable, platform) + " --db " + shellQuote(database, platform) + " mcp"
	shell := "Terminal (bash or zsh)"
	if platform == "windows" {
		shell = "PowerShell"
	}
	return map[string]any{
		"name": name, "shell": shell, "platform": platform,
		"codex":  "codex mcp add " + name + " -- " + launch,
		"claude": "claude mcp add --transport stdio --scope user " + name + " -- " + launch,
		"config": map[string]any{"mcpServers": map[string]any{name: map[string]any{"command": executable, "args": []string{"--db", database, "mcp"}}}},
	}
}

func (s *Server) connectionSetup(w http.ResponseWriter, r *http.Request) {
	executable, database, err := s.connectionPaths()
	if err != nil {
		writeError(w, 500, "Could not prepare this installation's connection settings. Reopen Player2 and try again.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, setupCommands(executable, database, runtime.GOOS))
}

func (s *Server) createConnectionPilot(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RequestID string `json:"requestId"`
		Goal      string `json:"goal"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, 400, "Choose a short goal for your first task.")
		return
	}
	pilot, err := s.store.CreateConnectionPilot(request.RequestID, request.Goal)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 201, pilot)
}

func (s *Server) connectionPilot(w http.ResponseWriter, r *http.Request) {
	pilot, err := s.store.GetConnectionPilot(chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if pilot == nil {
		writeError(w, 404, "This first task was removed or belongs to another installation. Start a new connection check.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, pilot)
}
