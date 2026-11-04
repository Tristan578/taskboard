package server

import (
	"encoding/json"
	"errors"
	"github.com/Tristan578/taskboard/internal/db"
	"net/http"
)

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req db.WorkspaceRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, 400, "We could not read this plan. Check the text and try again.")
		return
	}
	p, err := s.store.CreateWorkspace(req)
	if err != nil {
		var validation *db.ValidationError
		if errors.As(err, &validation) {
			writeError(w, 400, err.Error())
		} else {
			writeError(w, 500, "Your board could not be saved. Your plan is still here; please try again.")
		}
		return
	}
	writeJSON(w, 201, p)
}
