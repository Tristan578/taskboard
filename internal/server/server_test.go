package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"database/sql"
	_ "modernc.org/sqlite"
)

func setupTestServer(t *testing.T) (*Server, *db.Store, func()) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Simple schema for server tests
	_, _ = database.Exec(`
		CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT, prefix TEXT, description TEXT, icon TEXT, color TEXT, status TEXT, github_repo TEXT, github_last_synced DATETIME, strict BOOLEAN DEFAULT 0, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE tickets (id TEXT PRIMARY KEY, project_id TEXT, team_id TEXT, number INTEGER, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME, position REAL, lexo_rank TEXT, github_issue_number INTEGER, github_last_synced_at DATETIME, github_last_synced_sha TEXT DEFAULT '', user_story TEXT DEFAULT '', acceptance_criteria TEXT DEFAULT '', technical_details TEXT DEFAULT '', testing_details TEXT DEFAULT '', is_draft BOOLEAN DEFAULT 0, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE sync_jobs (id TEXT PRIMARY KEY, project_id TEXT, ticket_id TEXT, action TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0, last_error TEXT, created_at DATETIME, updated_at DATETIME);
	`)

	store := db.NewStore(database)
	s := New(store, nil)
	return s, store, func() { database.Close() }
}

func TestServer_GetBoard(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	_, _ = store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	req := httptest.NewRequest("GET", "/api/board?projectId="+p.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestServer_DraftMode(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "Strict Project",
		Prefix: "STRICT",
		Strict: true,
	})

	// 1. Create a draft (should succeed even without UserStory/AC)
	reqBody, _ := json.Marshal(models.CreateTicketRequest{
		ProjectID: p.ID,
		Title: "Draft Ticket",
		IsDraft: true,
	})
	
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for draft, got %d", w.Code)
	}

	var ticket models.Ticket
	json.Unmarshal(w.Body.Bytes(), &ticket)

	// 2. Attempt to move draft to 'todo' (should fail)
	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "todo"})
	req = httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 when moving draft without ACs, got %d", w.Code)
	}
}
