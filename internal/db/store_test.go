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
			lexo_rank TEXT,
			github_issue_number INTEGER,
			github_last_synced_at DATETIME,
			github_last_synced_sha TEXT DEFAULT '',
			user_story TEXT DEFAULT '',
			acceptance_criteria TEXT DEFAULT '',
			technical_details TEXT DEFAULT '',
			testing_details TEXT DEFAULT '',
			is_draft BOOLEAN DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE sync_jobs (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			ticket_id TEXT,
			action TEXT,
			payload TEXT,
			status TEXT DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			last_error TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE teams (
			id TEXT PRIMARY KEY,
			name TEXT,
			color TEXT,
			created_at DATETIME
		);
		CREATE TABLE labels (
			id TEXT PRIMARY KEY,
			name TEXT,
			color TEXT
		);
		CREATE TABLE subtasks (
			id TEXT PRIMARY KEY,
			ticket_id TEXT,
			title TEXT,
			completed BOOLEAN DEFAULT 0,
			position INTEGER
		);
		CREATE TABLE ticket_labels (
			ticket_id TEXT,
			label_id TEXT,
			PRIMARY KEY (ticket_id, label_id)
		);
		CREATE TABLE ticket_dependencies (
			ticket_id TEXT,
			blocked_by_id TEXT,
			PRIMARY KEY (ticket_id, blocked_by_id)
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

func TestStore_Teams(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	// Create
	team, err := s.CreateTeam(models.CreateTeamRequest{Name: "Engineering", Color: "#FF0000"})
	if err != nil {
		t.Fatal(err)
	}

	// List
	teams, _ := s.ListTeams()
	if len(teams) != 1 {
		t.Errorf("Expected 1 team, got %d", len(teams))
	}

	// Update
	newName := "Product"
	_, _ = s.UpdateTeam(team.ID, models.UpdateTeamRequest{Name: &newName})
	
	t2, _ := s.GetTeam(team.ID)
	if t2.Name != "Product" {
		t.Errorf("Team name update failed")
	}

	// Delete
	_ = s.DeleteTeam(team.ID)
	teams, _ = s.ListTeams()
	if len(teams) != 0 {
		t.Errorf("Delete team failed")
	}
}

func TestStore_Labels(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	label, _ := s.CreateLabel(models.CreateLabelRequest{Name: "bug", Color: "red"})
	labels, _ := s.ListLabels()
	if len(labels) != 1 {
		t.Errorf("Expected 1 label")
	}

	_ = s.DeleteLabel(label.ID)
}

func TestStore_Subtasks(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	ticket, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	st, _ := s.AddSubtask(ticket.ID, models.CreateSubtaskRequest{Title: "Task 1"})
	if st.Title != "Task 1" {
		t.Errorf("Subtask creation failed")
	}

	st2, _ := s.ToggleSubtask(st.ID)
	if !st2.Completed {
		t.Errorf("Subtask toggle failed")
	}

	_ = s.DeleteSubtask(st.ID)
}

func TestStore_MoveTicket(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	_, _ = s.MoveTicket(t1.ID, models.MoveTicketRequest{Status: "in_progress"})
	
	t2, _ := s.GetTicket(t1.ID)
	if t2.Status != "in_progress" {
		t.Errorf("Move ticket failed")
	}
}
