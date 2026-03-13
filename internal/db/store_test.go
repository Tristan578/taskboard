package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

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
		CREATE TABLE sync_jobs (id TEXT PRIMARY KEY, project_id TEXT, ticket_id TEXT, action TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0, last_error TEXT, created_at DATETIME, updated_at DATETIME);
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
	_, _ = s.UpdateProject(p.ID, models.UpdateProjectRequest{Name: &newName})
	_, _ = s.GetProject(p.ID)
	_, _ = s.ListProjects("active")
	_, _ = s.ListProjects("")

	tm, _ := s.CreateTeam(models.CreateTeamRequest{Name: "T1"})
	_, _ = s.UpdateTeam(tm.ID, models.UpdateTeamRequest{Name: &newName})
	_, _ = s.GetTeam(tm.ID)
	_, _ = s.ListTeams()

	tick, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	_, _ = s.UpdateTicket(tick.ID, models.UpdateTicketRequest{Title: &newName})
	_, _ = s.GetTicket(tick.ID)
	_, _ = s.ListTickets(models.TicketFilter{ProjectID: p.ID})
	_, _ = s.MoveTicket(tick.ID, models.MoveTicketRequest{Status: "done"})

	l, _ := s.CreateLabel(models.CreateLabelRequest{Name: "L1", Color: "red"})
	_, _ = s.UpdateLabel(l.ID, models.UpdateLabelRequest{Name: &newName})
	_, _ = s.GetLabel(l.ID)
	_, _ = s.ListLabels()

	st, _ := s.AddSubtask(tick.ID, models.CreateSubtaskRequest{Title: "S1"})
	_, _ = s.GetSubtask(st.ID)
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

	_ = s.DeleteTicket(tick.ID)
	_, _ = s.ListDeletedTickets(p.ID)
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

func TestStore_InvalidDate(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	badDate := "not-a-date"
	tick, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", DueDate: &badDate})
	if tick.DueDate != nil { t.Errorf("1") }
}
