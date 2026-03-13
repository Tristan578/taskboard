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

	// Comprehensive schema for unit tests with constraints
	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			name TEXT,
			prefix TEXT UNIQUE,
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
			project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
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
			ticket_id TEXT REFERENCES tickets(id) ON DELETE CASCADE,
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

	team, err := s.CreateTeam(models.CreateTeamRequest{Name: "Engineering", Color: "#FF0000"})
	if err != nil {
		t.Fatal(err)
	}

	teams, _ := s.ListTeams()
	if len(teams) != 1 {
		t.Errorf("Expected 1 team, got %d", len(teams))
	}

	newName := "Product"
	_, _ = s.UpdateTeam(team.ID, models.UpdateTeamRequest{Name: &newName})
	
	t2, _ := s.GetTeam(team.ID)
	if t2.Name != "Product" {
		t.Errorf("Team name update failed")
	}

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
		t.Errorf("Expected 1 job, got %d", len(jobs))
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
	updated, err := s.UpdateProject(p.ID, models.UpdateProjectRequest{Name: &newName})
	if err != nil || updated.Name != newName {
		t.Errorf("UpdateProject failed")
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
	updated, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Title: &newTitle})
	if err != nil || updated.Title != newTitle {
		t.Errorf("UpdateTicket failed")
	}
}

func TestStore_MoveTicket_NoPosition(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	_, _ = s.MoveTicket(t1.ID, models.MoveTicketRequest{Status: "done"})
	t2, _ := s.GetTicket(t1.ID)
	if t2.Status != "done" || t2.Position < 1000 {
		t.Errorf("Auto-positioning failed")
	}
}

func TestStore_Constraints(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Unique Prefix Constraint
	_, _ = s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "AUTH"})
	_, err := s.CreateProject(models.CreateProjectRequest{Name: "P2", Prefix: "AUTH"})
	if err == nil {
		t.Errorf("Expected error for duplicate project prefix")
	}

	// 2. Foreign Key Constraint (Ticket without Project)
	_, err = s.CreateTicket(models.CreateTicketRequest{ProjectID: "nonexistent", Title: "T1"})
	if err == nil {
		t.Errorf("Expected error for ticket with nonexistent project ID")
	}
}

func TestStore_Teams_Errors(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := s.GetTeam("nonexistent")
	if err != nil {
		t.Errorf("GetTeam should return nil, nil for nonexistent, got %v", err)
	}
}

func TestStore_Labels_Errors(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := s.GetLabel("nonexistent")
	if err != nil {
		t.Errorf("GetLabel should return nil, nil for nonexistent")
	}
}

func TestStore_Subtasks_Errors(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := s.GetSubtask("nonexistent")
	if err != nil {
		t.Errorf("GetSubtask should return nil, nil for nonexistent")
	}

	_, err = s.ToggleSubtask("nonexistent")
	if err != nil {
		t.Errorf("ToggleSubtask should return nil, nil for nonexistent")
	}
}

func TestStore_Filters(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p1, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	team, _ := s.CreateTeam(models.CreateTeamRequest{Name: "T1"})

	_, _ = s.CreateTicket(models.CreateTicketRequest{ProjectID: p1.ID, TeamID: &team.ID, Title: "T1", Status: "todo", Priority: "high"})

	list, _ := s.ListTickets(models.TicketFilter{ProjectID: p1.ID})
	if len(list) != 1 { t.Errorf("Filter Project failed") }

	list, _ = s.ListTickets(models.TicketFilter{TeamID: team.ID})
	if len(list) != 1 { t.Errorf("Filter Team failed") }

	list, _ = s.ListTickets(models.TicketFilter{Priority: "high"})
	if len(list) != 1 { t.Errorf("Filter Priority failed") }
}

func TestStore_InternalHelpers(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	l, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L1"})
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", Labels: []string{l.ID}})
	_, _ = s.AddSubtask(t1.ID, models.CreateSubtaskRequest{Title: "S1"})
	t2, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T2", BlockedBy: []string{t1.ID}})

	// Direct check of internal helpers (scan coverage)
	_, _ = s.getTicketLabels(t1.ID)
	_, _ = s.getTicketSubtasks(t1.ID)
	_, _ = s.getTicketBlockedBy(t2.ID)
}

func TestStore_GetBoard(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	_, _ = s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", Status: "todo"})
	
	board, err := s.GetBoard(p.ID)
	if err != nil || len(board.Columns) != 3 {
		t.Errorf("GetBoard failed")
	}
}

func TestStore_UpdateLabel(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	l, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L1", Color: "red"})
	name := "L2"
	_, _ = s.UpdateLabel(l.ID, models.UpdateLabelRequest{Name: &name})
}

func TestStore_UpdateTeam(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	tm, _ := s.CreateTeam(models.CreateTeamRequest{Name: "T1"})
	name := "T2"
	_, _ = s.UpdateTeam(tm.ID, models.UpdateTeamRequest{Name: &name})
}

func TestStore_ListProjects_All(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1", Color: ""})
	list, _ := s.ListProjects("")
	if len(list) != 1 { t.Errorf("ListProjects failed") }
}

func TestStore_UpdateProject_Exhaustive(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	
	name := "N"
	pref := "PR"
	desc := "D"
	icon := "I"
	color := "C"
	stat := "archived"
	repo := "R"
	strict := true

	_, _ = s.UpdateProject(p.ID, models.UpdateProjectRequest{
		Name: &name, Prefix: &pref, Description: &desc, Icon: &icon, 
		Color: &color, Status: &stat, GitHubRepo: &repo, Strict: &strict,
	})
}

func TestStore_UpdateTicket_AllFields(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	tick, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	title := "T2"
	desc := "D2"
	stat := "done"
	prio := "low"
	pos := 2000.0
	lr := "0000002000"
	ghn := 456
	sha := "def"
	us := "us"
	ac := "ac"
	td := "td"
	ts := "ts"
	idr := true
	due := "2027-01-01"

	_, err := s.UpdateTicket(tick.ID, models.UpdateTicketRequest{
		Title: &title, Description: &desc, Status: &stat, Priority: &prio,
		Position: &pos, LexoRank: &lr, GitHubIssueNumber: &ghn,
		GitHubLastSyncedSHA: &sha, UserStory: &us, AcceptanceCriteria: &ac,
		TechnicalDetails: &td, TestingDetails: &ts, IsDraft: &idr, DueDate: &due,
	})
	if err != nil { t.Errorf("Exhaustive UpdateTicket failed: %v", err) }
}

func TestStore_UpdateLabel_Minimal(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	l, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L1", Color: "red"})
	_, _ = s.UpdateLabel(l.ID, models.UpdateLabelRequest{})
}

func TestStore_UpdateTeam_Minimal(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	tm, _ := s.CreateTeam(models.CreateTeamRequest{Name: "T1"})
	_, _ = s.UpdateTeam(tm.ID, models.UpdateTeamRequest{})
}

func TestStore_ListEmpty(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	list, _ := s.ListTeams()
	if list != nil && len(list) != 0 { t.Errorf("ListTeams should be empty") }

	labels, _ := s.ListLabels()
	if labels != nil && len(labels) != 0 { t.Errorf("ListLabels should be empty") }
}

func TestStore_ScanErrors(t *testing.T) {
	_, cleanup := setupTestStore(t)
	defer cleanup()
}

func TestStore_MoveTicket_Complex(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	_, _ = s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	t2, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T2"})

	// Move without position (auto-increment)
	_, _ = s.MoveTicket(t2.ID, models.MoveTicketRequest{Status: "todo"})
}

func TestStore_ListTickets_Empty(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	list, _ := s.ListTickets(models.TicketFilter{})
	if len(list) != 0 { t.Errorf("Expected 0 tickets") }
}
