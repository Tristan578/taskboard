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
		CREATE TABLE teams (id TEXT PRIMARY KEY, name TEXT, color TEXT, created_at DATETIME);
		CREATE TABLE labels (id TEXT PRIMARY KEY, name TEXT, color TEXT);
		CREATE TABLE subtasks (id TEXT PRIMARY KEY, ticket_id TEXT, title TEXT, completed BOOLEAN DEFAULT 0, position INTEGER);
		CREATE TABLE ticket_labels (ticket_id TEXT, label_id TEXT, PRIMARY KEY (ticket_id, label_id));
		CREATE TABLE ticket_dependencies (ticket_id TEXT, blocked_by_id TEXT, PRIMARY KEY (ticket_id, blocked_by_id));
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

func TestServer_Teams(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Create Team
	body, _ := json.Marshal(models.CreateTeamRequest{Name: "Devs", Color: "blue"})
	req := httptest.NewRequest("POST", "/api/teams", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Team creation failed")
	}

	// 2. List Teams
	req = httptest.NewRequest("GET", "/api/teams", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Teams listing failed")
	}
}

func TestServer_Labels(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(models.CreateLabelRequest{Name: "bug", Color: "red"})
	req := httptest.NewRequest("POST", "/api/labels", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Label creation failed")
	}
}

func TestServer_Projects(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(models.CreateProjectRequest{Name: "New Project", Prefix: "NEW"})
	req := httptest.NewRequest("POST", "/api/projects", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Project creation failed")
	}

	var project models.Project
	json.Unmarshal(w.Body.Bytes(), &project)

	// Update
	newName := "Updated Project"
	updateBody, _ := json.Marshal(models.UpdateProjectRequest{Name: &newName})
	req = httptest.NewRequest("PUT", "/api/projects/"+project.ID, bytes.NewBuffer(updateBody))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Project update failed")
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/projects/"+project.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("Project deletion failed")
	}
}

func TestServer_Tickets(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	
	// Create
	body, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	
	var ticket models.Ticket
	json.Unmarshal(w.Body.Bytes(), &ticket)

	// Update
	newTitle := "T1 Updated"
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{Title: &newTitle})
	req = httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Ticket update failed")
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/tickets/"+ticket.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("Ticket deletion failed")
	}
}
