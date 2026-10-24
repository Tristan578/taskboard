package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) StartMaintenance(ctx context.Context) { go s.recovery.Run(ctx) }
func (s *Server) cancelPreview(w http.ResponseWriter, r *http.Request) {
	if err := s.recovery.CancelPreview(chi.URLParam(r, "id")); err != nil {
		writeError(w, 400, "Could not cancel this preview.")
		return
	}
	w.WriteHeader(204)
}
func (s *Server) recoveryStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.recovery.Status()
	if err != nil {
		writeError(w, 500, "Could not read saved backups. Check the data folder and try again.")
		return
	}
	writeJSON(w, 200, status)
}
func (s *Server) createBackup(w http.ResponseWriter, r *http.Request) {
	b, err := s.recovery.Create()
	if err != nil {
		writeError(w, 500, "A backup could not be saved. Check disk space and folder permissions.")
		return
	}
	writeJSON(w, 201, b)
}
func (s *Server) savedBackup(w http.ResponseWriter, r *http.Request) {
	path, err := s.recovery.BackupPath(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "Backup not found")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="player2-backup.db"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	// #nosec G703 -- BackupPath accepts only a ULID plus an allowed backup suffix, in the private backup directory; it rejects symlinks and non-regular files.
	http.ServeFile(w, r, path)
}
func (s *Server) previewBackup(w http.ResponseWriter, r *http.Request) {
	preview, err := s.recovery.Preview(r.Body)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, preview)
}
func (s *Server) previewSavedBackup(w http.ResponseWriter, r *http.Request) {
	preview, err := s.recovery.PreviewSaved(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, preview)
}
func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request) {
	var a struct {
		ID      string `json:"id"`
		Confirm bool   `json:"confirm"`
	}
	if err := decodeJSON(r, &a); err != nil || !a.Confirm {
		writeError(w, 400, "Review the preview and confirm restoration first.")
		return
	}
	ids, err := s.recovery.Restore(a.ID)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"projectIds": ids})
}
func (s *Server) listTrash(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.Trash()
	if err != nil {
		writeError(w, 500, "Could not read deleted tasks.")
		return
	}
	writeJSON(w, 200, tasks)
}
func (s *Server) restoreTask(w http.ResponseWriter, r *http.Request) {
	var a struct {
		Revision *int `json:"revision"`
	}
	if err := decodeJSON(r, &a); err != nil || a.Revision == nil {
		writeError(w, 400, "Refresh deleted tasks before restoring.")
		return
	}
	task, err := s.store.RestoreTicket(chi.URLParam(r, "id"), *a.Revision)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, task)
}
