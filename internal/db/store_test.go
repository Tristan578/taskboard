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

func TestStore_SyncJobs(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	err := s.QueueSyncJob("p1", "t1", "full_sync", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatal(err)
	}

	// Check if it exists at all
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM sync_jobs").Scan(&count)
	if count != 1 {
		t.Fatalf("Job not inserted into database")
	}

	jobs, err := s.GetPendingSyncJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		var status, attempts string
		s.db.QueryRow("SELECT status, attempts FROM sync_jobs").Scan(&status, &attempts)
		t.Errorf("Expected 1 job, got %d. Status: %s, Attempts: %s", len(jobs), status, attempts)
		return
	}

	job := jobs[0]
	if job.ProjectID != "p1" || job.Action != "full_sync" {
		t.Errorf("Job data mismatch")
	}

	_ = s.UpdateSyncJobStatus(job.ID, "completed", 1, "")
	jobs, _ = s.GetPendingSyncJobs()
	if len(jobs) != 0 {
		t.Errorf("Expected 0 pending jobs")
	}
}

func TestStore_DeletedTickets(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	_ = s.DeleteTicket(t1.ID)
	
	active, _ := s.ListTickets(models.TicketFilter{ProjectID: p.ID})
	if len(active) != 0 {
		t.Errorf("Deleted ticket still in active list")
	}

	deleted, _ := s.ListDeletedTickets(p.ID)
	if len(deleted) != 1 {
		t.Errorf("Ticket not in deleted list")
	}

	_ = s.PurgeDeletedTickets(p.ID)
	deleted, _ = s.ListDeletedTickets(p.ID)
	if len(deleted) != 0 {
		t.Errorf("Purge failed")
	}
}

func TestStore_ClearData(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	_ = s.ClearData()
	
	projects, _ := s.ListProjects("")
	if len(projects) != 0 {
		t.Errorf("ClearData failed")
	}
}

func TestStore_UpdateProject(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Old Name", Prefix: "OLD"})
	
	newName := "New Name"
	newPrefix := "NEW"
	newDesc := "Desc"
	newIcon := "🚀"
	newColor := "#000000"
	newStatus := "archived"
	newRepo := "owner/repo"
	newStrict := false

	updated, err := s.UpdateProject(p.ID, models.UpdateProjectRequest{
		Name: &newName, Prefix: &newPrefix, Description: &newDesc, 
		Icon: &newIcon, Color: &newColor, Status: &newStatus, 
		GitHubRepo: &newRepo, Strict: &newStrict,
	})
	if err != nil || updated.Name != newName || updated.Prefix != newPrefix || updated.Status != "archived" {
		t.Errorf("UpdateProject failed: %+v", updated)
	}

	_ = s.DeleteProject(p.ID)
	p2, _ := s.GetProject(p.ID)
	if p2 != nil {
		t.Errorf("DeleteProject failed")
	}
}

func TestStore_UpdateTicket(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	ticket, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	newTitle := "Updated Title"
	newDesc := "Updated Desc"
	newStatus := "done"
	newPriority := "urgent"
	newUS := "As a..."
	newAC := "Given..."
	newTech := "Tech..."
	newTest := "Test..."
	newDraft := false

	updated, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{
		Title: &newTitle, Description: &newDesc, Status: &newStatus, 
		Priority: &newPriority, UserStory: &newUS, AcceptanceCriteria: &newAC,
		TechnicalDetails: &newTech, TestingDetails: &newTest, IsDraft: &newDraft,
	})
	if err != nil || updated.Title != newTitle || updated.Status != "done" {
		t.Errorf("UpdateTicket failed: %+v", updated)
	}
}

func TestStore_TicketLabels(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	l1, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L1"})
	
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", Labels: []string{l1.ID}})
	if len(t1.Labels) != 1 {
		t.Errorf("Labels during creation failed")
	}

	l2, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L2"})
	updated, _ := s.UpdateTicket(t1.ID, models.UpdateTicketRequest{Labels: []string{l2.ID}})
	if len(updated.Labels) != 1 || updated.Labels[0].ID != l2.ID {
		t.Errorf("Label update failed")
	}
}

func TestStore_MoveTicket(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	// Move to in_progress
	_, _ = s.MoveTicket(t1.ID, models.MoveTicketRequest{Status: "in_progress"})
	t2, _ := s.GetTicket(t1.ID)
	if t2.Status != "in_progress" {
		t.Errorf("Move ticket to in_progress failed")
	}

	// Move with position
	pos := 500.0
	_, _ = s.MoveTicket(t1.ID, models.MoveTicketRequest{Status: "todo", Position: &pos})
	t3, _ := s.GetTicket(t1.ID)
	if t3.Status != "todo" || t3.Position != pos {
		t.Errorf("Move ticket with position failed")
	}
}

func TestStore_Filters(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p1, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	p2, _ := s.CreateProject(models.CreateProjectRequest{Name: "P2", Prefix: "P2"})
	team, _ := s.CreateTeam(models.CreateTeamRequest{Name: "T1"})

	_, _ = s.CreateTicket(models.CreateTicketRequest{ProjectID: p1.ID, TeamID: &team.ID, Title: "T1", Status: "todo", Priority: "high"})
	_, _ = s.CreateTicket(models.CreateTicketRequest{ProjectID: p2.ID, Title: "T2", Status: "in_progress", Priority: "low"})

	// Filter by Project
	list, _ := s.ListTickets(models.TicketFilter{ProjectID: p1.ID})
	if len(list) != 1 { t.Errorf("Filter Project failed") }

	// Filter by Team
	list, _ = s.ListTickets(models.TicketFilter{TeamID: team.ID})
	if len(list) != 1 { t.Errorf("Filter Team failed") }

	// Filter by Status
	list, _ = s.ListTickets(models.TicketFilter{Status: "in_progress"})
	if len(list) != 1 { t.Errorf("Filter Status failed") }

	// Filter by Priority
	list, _ = s.ListTickets(models.TicketFilter{Priority: "high"})
	if len(list) != 1 { t.Errorf("Filter Priority failed") }
}
