package db

import (
	"database/sql"
	"testing"

	"github.com/Tristan578/taskboard/internal/models"
	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Manual table creation for speed in unit tests
	_, err = db.Exec(`
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			name TEXT,
			prefix TEXT,
			description TEXT,
			icon TEXT,
			color TEXT,
			status TEXT,
			github_repo TEXT,
			github_last_synced DATETIME,
			strict BOOLEAN DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE tickets (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			team_id TEXT,
			number INTEGER,
			title TEXT,
			description TEXT,
			status TEXT,
			priority TEXT,
			due_date DATETIME,
			position REAL,
			github_issue_number INTEGER,
			github_last_synced_at DATETIME,
			github_last_synced_sha TEXT DEFAULT '',
			user_story TEXT DEFAULT '',
			acceptance_criteria TEXT DEFAULT '',
			technical_details TEXT DEFAULT '',
			testing_details TEXT DEFAULT '',
			created_at DATETIME,
			updated_at DATETIME
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	return NewStore(db), func() { db.Close() }
}

func TestStore_CreateProject(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	req := models.CreateProjectRequest{
		Name:       "Test Project",
		Prefix:     "TEST",
		GitHubRepo: "owner/repo",
		Strict:     true,
	}

	p, err := s.CreateProject(req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	if p.Name != req.Name || p.Prefix != req.Prefix || p.GitHubRepo != req.GitHubRepo || p.Strict != true {
		t.Errorf("Project fields mismatch: %+v", p)
	}

	// Verify GetProject
	p2, err := s.GetProject(p.ID)
	if err != nil || p2 == nil {
		t.Fatalf("Failed to get project: %v", err)
	}

	if p2.Strict != true {
		t.Errorf("Expected project to be strict")
	}
}

func TestStore_CreateTicket(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})

	req := models.CreateTicketRequest{
		ProjectID:          p.ID,
		Title:              "Strict Ticket",
		UserStory:          "As a user...",
		AcceptanceCriteria: "Given...",
	}

	ticket, err := s.CreateTicket(req)
	if err != nil {
		t.Fatalf("Failed to create ticket: %v", err)
	}

	if ticket.UserStory != req.UserStory || ticket.AcceptanceCriteria != req.AcceptanceCriteria {
		t.Errorf("Ticket fields mismatch")
	}
}
