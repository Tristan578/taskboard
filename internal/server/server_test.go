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
		CREATE TABLE tickets (id TEXT PRIMARY KEY, project_id TEXT, team_id TEXT, number INTEGER, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME, position REAL, github_issue_number INTEGER, github_last_synced_at DATETIME, github_last_synced_sha TEXT DEFAULT '', user_story TEXT DEFAULT '', acceptance_criteria TEXT DEFAULT '', technical_details TEXT DEFAULT '', testing_details TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME);
	`)

	store := db.NewStore(database)
	s := New(store, nil)
	return s, store, func() { database.Close() }
}

func TestServer_CreateTicket_Strict(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Create a Strict Project
	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "Strict Project",
		Prefix: "STRICT",
		Strict: true,
	})

	// 2. Attempt to create a ticket without required fields
	reqBody, _ := json.Marshal(models.CreateTicketRequest{
		ProjectID: p.ID,
		Title: "Failing Ticket",
	})
	
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing strict fields, got %d", w.Code)
	}

	// 3. Create a ticket with required fields
	reqBody, _ = json.Marshal(models.CreateTicketRequest{
		ProjectID: p.ID,
		Title: "Passing Ticket",
		UserStory: "As a user...",
		AcceptanceCriteria: "Given...",
	})
	
	req = httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(reqBody))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for valid strict fields, got %d. Body: %s", w.Code, w.Body.String())
	}
}
