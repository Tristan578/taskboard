package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT, prefix TEXT UNIQUE, description TEXT, icon TEXT, color TEXT, status TEXT, github_repo TEXT, github_last_synced DATETIME, strict BOOLEAN DEFAULT 0, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE tickets (id TEXT PRIMARY KEY, project_id TEXT REFERENCES projects(id) ON DELETE CASCADE, team_id TEXT, number INTEGER, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME, position REAL, lexo_rank TEXT, github_issue_number INTEGER, github_last_synced_at DATETIME, github_last_synced_sha TEXT DEFAULT '', user_story TEXT DEFAULT '', acceptance_criteria TEXT DEFAULT '', technical_details TEXT DEFAULT '', testing_details TEXT DEFAULT '', is_draft BOOLEAN DEFAULT 0, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE sync_jobs (id TEXT PRIMARY KEY, project_id TEXT, ticket_id TEXT, action TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE teams (id TEXT PRIMARY KEY, name TEXT, color TEXT, created_at DATETIME);
		CREATE TABLE labels (id TEXT PRIMARY KEY, name TEXT, color TEXT);
		CREATE TABLE subtasks (id TEXT PRIMARY KEY, ticket_id TEXT REFERENCES tickets(id) ON DELETE CASCADE, title TEXT, completed BOOLEAN DEFAULT 0, position INTEGER);
		CREATE TABLE ticket_labels (ticket_id TEXT, label_id TEXT, PRIMARY KEY (ticket_id, label_id));
		CREATE TABLE ticket_dependencies (ticket_id TEXT, blocked_by_id TEXT, PRIMARY KEY (ticket_id, blocked_by_id));
		CREATE TABLE schema_migrations (version TEXT PRIMARY KEY);
	`)
	if err != nil {
		t.Fatal(err)
	}

	return NewStore(db), func() { _ = db.Close() }
}

func TestDB_Lifecycle(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "db-lifecycle")
	defer os.RemoveAll(tempDir)

	path := filepath.Join(tempDir, "test.db")
	dbConn, err := OpenAt(path)
	if err != nil { t.Fatalf("OpenAt failed: %v", err) }
	_ = dbConn.Close()

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)
	
	db2, err := Open()
	if err == nil { _ = db2.Close() }
}

func TestDB_OpenAt_Error(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "db-err")
	defer os.RemoveAll(tempDir)
	_, err := OpenAt(tempDir)
	if err == nil { t.Errorf("expected error") }
}

func TestStore_Exhaustive_CRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	
	newName := "P2"
	newPrefix := "P2X"
	newDesc := "D2"
	newIcon := "I2"
	newColor := "C2"
	newStatus := "archived"
	newRepo := "owner/repo2"
	newStrict := true
	_, _ = s.UpdateProject(p.ID, models.UpdateProjectRequest{
		Name: &newName, Prefix: &newPrefix, Description: &newDesc,
		Icon: &newIcon, Color: &newColor, Status: &newStatus,
		GitHubRepo: &newRepo, Strict: &newStrict,
	})
	_, _ = s.GetProject(p.ID)
	_, _, _ = s.ListProjects("archived")
	_, _, _ = s.ListProjects("")

	tm, _ := s.CreateTeam(models.CreateTeamRequest{Name: "T1"})
	newTeamName := "T2"
	newTeamColor := "blue"
	_, _ = s.UpdateTeam(tm.ID, models.UpdateTeamRequest{Name: &newTeamName, Color: &newTeamColor})
	_, _ = s.GetTeam(tm.ID)
	_, _, _ = s.ListTeams()

	tick, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	_, _ = s.UpdateTicket(tick.ID, models.UpdateTicketRequest{Title: &newName})
	_, _ = s.GetTicket(tick.ID)
	// Filter permutations
	urgent := "urgent"
	_, _, _ = s.ListTickets(models.TicketFilter{ProjectID: p.ID, TeamID: tm.ID, Status: "archived", Priority: urgent})
	_, _ = s.MoveTicket(tick.ID, models.MoveTicketRequest{Status: "done"})

	l, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L1", Color: "red"})
	newLabelName := "L2"
	newLabelColor := "blue"
	_, _ = s.UpdateLabel(l.ID, models.UpdateLabelRequest{Name: &newLabelName, Color: &newLabelColor})
	_, _ = s.GetLabel(l.ID)
	_, _, _ = s.ListLabels()

	st, _ := s.AddSubtask(tick.ID, models.CreateSubtaskRequest{Title: "S1"})
	_, _ = s.GetSubtask(st.ID)
	// This should hit getTicketSubtasks via GetTicket
	_, _ = s.GetTicket(tick.ID)
	_, _ = s.ToggleSubtask(st.ID)
	_ = s.DeleteSubtask(st.ID)

	t2, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T2", Labels: []string{l.ID}, BlockedBy: []string{tick.ID}})
	
	ghn := 1
	sha := "sha"
	us := "us"
	ac := "ac"
	td := "td"
	ts := "ts"
	dr := true
	pos := 500.0
	lr := "rank"
	due := "2026-01-01"
	teamID := tm.ID

	_, _ = s.UpdateTicket(t2.ID, models.UpdateTicketRequest{
		Title: &newName, Description: &newName, Status: &newName, Priority: &newName,
		Position: &pos, LexoRank: &lr, GitHubIssueNumber: &ghn,
		GitHubLastSyncedSHA: &sha, UserStory: &us, AcceptanceCriteria: &ac,
		TechnicalDetails: &td, TestingDetails: &ts, IsDraft: &dr, DueDate: &due,
		TeamID: &teamID, Labels: []string{l.ID}, BlockedBy: []string{tick.ID},
	})

	_, _ = s.GetBoard(p.ID)
	_, _ = s.GetBoard("")

	_ = s.QueueSyncJob(p.ID, tick.ID, "sync", nil)
	jobs, _ := s.GetPendingSyncJobs()
	_ = s.UpdateSyncJobStatus(jobs[0].ID, "failed", 1, "err")
	_, _ = s.GetPendingSyncJobs()

	// Ensure ListDeletedTickets hits its loop
	_ = s.DeleteTicket(tick.ID)
	deleted, _ := s.ListDeletedTickets(p.ID)
	if len(deleted) == 0 { t.Error("expected deleted ticket") }
	_ = s.PurgeDeletedTickets(p.ID)
	_ = s.DeleteTeam(tm.ID)
	_ = s.DeleteLabel(l.ID)
	_ = s.DeleteProject(p.ID)
	_ = s.ClearData()
}

func TestStore_Errors(t *testing.T) {
	dbConn, _ := sql.Open("sqlite", ":memory:")
	s := NewStore(dbConn)
	_ = s.Close() 

	if _, err := s.CreateProject(models.CreateProjectRequest{}); err == nil { t.Errorf("1") }
	if _, err := s.GetProject("1"); err == nil { t.Errorf("2") }
	if _, err := s.UpdateProject("1", models.UpdateProjectRequest{}); err == nil { t.Errorf("3") }
	if err := s.DeleteProject("1"); err == nil { t.Errorf("4") }
	if _, err := s.CreateTeam(models.CreateTeamRequest{}); err == nil { t.Errorf("5") }
	if _, err := s.GetTeam("1"); err == nil { t.Errorf("6") }
	if _, err := s.UpdateTeam("1", models.UpdateTeamRequest{}); err == nil { t.Errorf("7") }
	if err := s.DeleteTeam("1"); err == nil { t.Errorf("8") }
	if _, err := s.CreateTicket(models.CreateTicketRequest{}); err == nil { t.Errorf("9") }
	if _, err := s.GetTicket("1"); err == nil { t.Errorf("10") }
	if _, err := s.UpdateTicket("1", models.UpdateTicketRequest{}); err == nil { t.Errorf("11") }
	if _, err := s.MoveTicket("1", models.MoveTicketRequest{}); err == nil { t.Errorf("12") }
	if err := s.DeleteTicket("1"); err == nil { t.Errorf("13") }
	if _, err := s.CreateLabel(models.CreateLabelRequest{}); err == nil { t.Errorf("14") }
	if _, err := s.GetLabel("1"); err == nil { t.Errorf("15") }
	if _, err := s.UpdateLabel("1", models.UpdateLabelRequest{}); err == nil { t.Errorf("16") }
	if err := s.DeleteLabel("1"); err == nil { t.Errorf("17") }
	if _, err := s.AddSubtask("1", models.CreateSubtaskRequest{}); err == nil { t.Errorf("18") }
	if _, err := s.ToggleSubtask("1"); err == nil { t.Errorf("19") }
	if err := s.DeleteSubtask("1"); err == nil { t.Errorf("20") }
	if err := s.ClearData(); err == nil { t.Errorf("21") }
	if _, err := s.GetBoard("1"); err == nil { t.Errorf("22") }
	if _, err := s.GetPendingSyncJobs(); err == nil { t.Errorf("23") }
	if err := s.UpdateSyncJobStatus("1", "done", 1, ""); err == nil { t.Errorf("24") }
	if _, _, err := s.ListProjects(""); err == nil { t.Errorf("25") }
	if _, _, err := s.ListTeams(); err == nil { t.Errorf("26") }
	if _, _, err := s.ListTickets(models.TicketFilter{}); err == nil { t.Errorf("27") }
	if _, err := s.ListDeletedTickets("1"); err == nil { t.Errorf("28") }
	if _, _, err := s.ListLabels(); err == nil { t.Errorf("29") }
	if _, err := s.GetSubtask("1"); err == nil { t.Errorf("30") }
	if err := s.PurgeDeletedTickets("1"); err == nil { t.Errorf("31") }
}

func TestStore_ClearData_Jobs(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	_ = s.QueueSyncJob("p1", "t1", "sync", nil)
	_ = s.ClearData()
}

func TestStore_Get_NotFound(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	if p, _ := s.GetProject("none"); p != nil { t.Errorf("1") }
	if tick, _ := s.GetTicket("none"); tick != nil { t.Errorf("2") }
	if st, _ := s.GetSubtask("none"); st != nil { t.Errorf("3") }
}

func TestDB_DefaultDBPath_Fallback(t *testing.T) {
	// On Windows, unsetting APPDATA should trigger the fallback
	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", "")
	defer os.Setenv("APPDATA", oldApp)

	path, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestDB_OpenAt_MkdirError(t *testing.T) {
	// Use a path that's a file, so MkdirAll fails
	tempFile, _ := os.CreateTemp("", "mkdir-err")
	defer os.Remove(tempFile.Name())
	
	path := filepath.Join(tempFile.Name(), "db.sqlite")
	_, err := OpenAt(path)
	if err == nil {
		t.Error("expected error for MkdirAll on a file path")
	}
}

func TestDB_runMigrations_Error(t *testing.T) {
	dbConn, _ := sql.Open("sqlite", ":memory:")
	_ = dbConn.Close() // Close it to make Exec fail
	
	err := runMigrations(dbConn)
	if err == nil {
		t.Error("expected error from runMigrations on closed DB")
	}
}

func TestDB_runMigrations_Idempotent(t *testing.T) {
	dbConn, _ := sql.Open("sqlite", ":memory:")
	defer dbConn.Close()
	
	err := runMigrations(dbConn)
	if err != nil { t.Fatalf("first migration: %v", err) }
	
	err = runMigrations(dbConn)
	if err != nil { t.Fatalf("second migration: %v", err) }
}

func TestStore_MoveTicket_Position(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	tick, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	
	pos := 500.0
	_, err := s.MoveTicket(tick.ID, models.MoveTicketRequest{Status: "in_progress", Position: &pos})
	if err != nil { t.Fatalf("MoveTicket failed: %v", err) }
}

func TestStore_InvalidDate(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	badDate := "not-a-date"
	tick, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", DueDate: &badDate})
	if tick.DueDate != nil { t.Errorf("1") }
}

func TestStore_Pagination_Tickets(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})

	// Create 10 tickets
	for i := 0; i < 10; i++ {
		_, _ = s.CreateTicket(models.CreateTicketRequest{
			ProjectID: p.ID, Title: fmt.Sprintf("T%d", i),
		})
	}

	// Paginate with limit=3
	tickets, total, err := s.ListTickets(models.TicketFilter{
		ProjectID: p.ID, Limit: 3, Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if total != 10 {
		t.Errorf("expected total=10, got %d", total)
	}
	if len(tickets) != 3 {
		t.Errorf("expected 3 tickets, got %d", len(tickets))
	}

	// Second page
	tickets2, total2, _ := s.ListTickets(models.TicketFilter{
		ProjectID: p.ID, Limit: 3, Offset: 3,
	})
	if total2 != 10 {
		t.Errorf("expected total=10, got %d", total2)
	}
	if len(tickets2) != 3 {
		t.Errorf("expected 3 tickets, got %d", len(tickets2))
	}

	// No pagination (all tickets)
	all, allTotal, _ := s.ListTickets(models.TicketFilter{ProjectID: p.ID})
	if len(all) != 10 || allTotal != 10 {
		t.Errorf("expected 10 tickets without pagination, got %d (total %d)", len(all), allTotal)
	}
}

func TestStore_Pagination_Projects(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		_, _ = s.CreateProject(models.CreateProjectRequest{
			Name: fmt.Sprintf("P%d", i), Prefix: fmt.Sprintf("P%d", i),
		})
	}

	projects, total, err := s.ListProjects("", 2, 0)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestStore_Pagination_Teams(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		_, _ = s.CreateTeam(models.CreateTeamRequest{Name: fmt.Sprintf("T%d", i)})
	}

	teams, total, err := s.ListTeams(2, 0)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(teams) != 2 {
		t.Errorf("expected 2 teams, got %d", len(teams))
	}
}

func TestStore_Pagination_Labels(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		_, _ = s.CreateLabel(models.CreateLabelRequest{Name: fmt.Sprintf("L%d", i), Color: "red"})
	}

	labels, total, err := s.ListLabels(2, 0)
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}
}

func TestStore_BatchLoad(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	l, _ := s.CreateLabel(models.CreateLabelRequest{Name: "Bug", Color: "red"})
	t1, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", Labels: []string{l.ID}})
	_, _ = s.AddSubtask(t1.ID, models.CreateSubtaskRequest{Title: "S1"})

	t2, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T2", BlockedBy: []string{t1.ID}})

	// ListTickets uses batch loading now
	tickets, _, err := s.ListTickets(models.TicketFilter{ProjectID: p.ID})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}

	// Find t1 and t2 in the results
	for _, tick := range tickets {
		if tick.ID == t1.ID {
			if len(tick.Labels) != 1 || tick.Labels[0].ID != l.ID {
				t.Errorf("expected label on t1, got %+v", tick.Labels)
			}
			if len(tick.Subtasks) != 1 {
				t.Errorf("expected 1 subtask on t1, got %d", len(tick.Subtasks))
			}
		}
		if tick.ID == t2.ID {
			if len(tick.BlockedBy) != 1 || tick.BlockedBy[0] != t1.ID {
				t.Errorf("expected blockedBy on t2, got %+v", tick.BlockedBy)
			}
		}
	}
}

func TestStore_NormalizePagination(t *testing.T) {
	l, o := normalizePagination(0, 0)
	if l != 50 || o != 0 { t.Errorf("default: got %d, %d", l, o) }

	l, o = normalizePagination(300, -5)
	if l != 200 || o != 0 { t.Errorf("cap: got %d, %d", l, o) }

	l, o = normalizePagination(10, 5)
	if l != 10 || o != 5 { t.Errorf("normal: got %d, %d", l, o) }
}

func TestStore_Ping(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	if err := s.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestStore_SyncJobRetry(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	_ = s.QueueSyncJob("p1", "t1", "sync", nil)
	jobs, _ := s.GetPendingSyncJobs()
	if len(jobs) == 0 {
		t.Fatal("expected pending job")
	}

	// Set retry far in the future
	futureRetry := time.Now().Add(1 * time.Hour)
	err := s.UpdateSyncJobRetry(jobs[0].ID, 1, "temporary error", futureRetry)
	if err != nil {
		t.Fatalf("UpdateSyncJobRetry: %v", err)
	}

	// Job should NOT appear since next_retry_at is in the future
	pending, _ := s.GetPendingSyncJobs()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending jobs (retry in future), got %d", len(pending))
	}

	// Set retry in the past
	pastRetry := time.Now().Add(-1 * time.Minute)
	_ = s.UpdateSyncJobRetry(jobs[0].ID, 1, "retry now", pastRetry)

	// Job should now appear
	pending2, _ := s.GetPendingSyncJobs()
	if len(pending2) != 1 {
		t.Errorf("expected 1 pending job (retry in past), got %d", len(pending2))
	}
}
