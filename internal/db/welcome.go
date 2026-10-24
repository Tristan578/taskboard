package db

import (
	"database/sql"
	"fmt"
	"github.com/Tristan578/taskboard/internal/models"
	"strings"
	"time"
)

type StarterTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
type WorkspaceRequest struct {
	RequestID        string        `json:"requestId"`
	Name             string        `json:"name"`
	Goal             string        `json:"goal"`
	CompletionPolicy string        `json:"completionPolicy"`
	Tasks            []StarterTask `json:"tasks"`
}

// CreateWorkspace commits the board and its tasks together. Retrying a request
// after a lost response returns the original board rather than duplicating it.
func (s *Store) CreateWorkspace(r WorkspaceRequest) (*models.Project, error) {
	r.Name = strings.TrimSpace(r.Name)
	if len(r.RequestID) < 16 || len(r.RequestID) > 100 {
		return nil, invalid("request ID is required")
	}
	if r.Name == "" || len(r.Name) > 120 {
		return nil, invalid("use a board name between 1 and 120 characters")
	}
	if len(r.Goal) > 4000 || len(r.Tasks) > 50 {
		return nil, invalid("use a shorter goal and no more than 50 tasks")
	}
	if r.CompletionPolicy != "manual" && r.CompletionPolicy != "verified" {
		return nil, invalid("choose an everyday or software board")
	}
	for i := range r.Tasks {
		r.Tasks[i].Title = strings.TrimSpace(r.Tasks[i].Title)
		if r.Tasks[i].Title == "" || len(r.Tasks[i].Title) > 240 || len(r.Tasks[i].Description) > 10000 {
			return nil, invalid("each task needs a short title and a description under 10000 characters")
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var id string
	err = tx.QueryRow("SELECT project_id FROM workspace_requests WHERE request_id=?", r.RequestID).Scan(&id)
	if err == nil {
		_ = tx.Rollback()
		return s.GetProject(id)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	id = newID()
	now := time.Now()
	_, err = tx.Exec("INSERT INTO projects (id,name,prefix,description,icon,color,status,completion_policy,created_at,updated_at) VALUES (?,?,?,?,?,?,'active',?,?,?)", id, r.Name, "P"+id[18:], r.Goal, "✨", "#14b8a6", r.CompletionPolicy, now, now)
	if err != nil {
		return nil, err
	}
	for i, t := range r.Tasks {
		_, err = tx.Exec("INSERT INTO tickets (id,project_id,number,title,description,status,priority,position,lexo_rank,created_at,updated_at) VALUES (?,?,?,?,?,'todo','medium',?,?,?,?)", newID(), id, i+1, t.Title, t.Description, i, fmt.Sprintf("%08d", i), now, now)
		if err != nil {
			return nil, err
		}
	}
	if _, err = tx.Exec("INSERT INTO workspace_requests(request_id,project_id) VALUES (?,?)", r.RequestID, id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetProject(id)
}

// Backup uses SQLite's consistent online snapshot, including committed WAL data.
func (s *Store) Backup(path string) error { _, err := s.db.Exec("VACUUM INTO ?", path); return err }

func (s *Store) DatabasePath() (string, error) {
	var sequence int
	var name, path string
	err := s.db.QueryRow("PRAGMA database_list").Scan(&sequence, &name, &path)
	return path, err
}
