package server

import (
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) downloadBackup(w http.ResponseWriter, r *http.Request) {
	dir, err := os.MkdirTemp("", "player2-backup-")
	if err != nil {
		writeError(w, 500, "Could not prepare a backup. Please try again.")
		return
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "player2-backup.db")
	if err = s.store.Backup(path); err != nil {
		writeError(w, 500, "Could not create a backup. Please try again.")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="player2-backup.db"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
}
