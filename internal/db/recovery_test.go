package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

func TestPreviousBackupVersionMigratesOnlyTheStagingCopy(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "previous.db")
	database, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`CREATE TABLE schema_migrations (
 version TEXT PRIMARY KEY,
 applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
 )`)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ := migrationsFS.ReadDir(migrations)
	for _, entry := range entries {
		if entry.Name() == "012_recovery_recurrence.sql" {
			break
		}
		content, _ := migrationsFS.ReadFile("migrations/" + entry.Name())
		if _, err = database.Exec(string(content)); err != nil {
			t.Fatal(err)
		}
		if _, err = database.Exec("INSERT INTO schema_migrations(version) VALUES(?)", entry.Name()); err != nil {
			t.Fatal(err)
		}
	}
	_ = database.Close()
	snapshot, err := ReadBackup(filename)
	if err != nil || len(snapshot.Projects) != 0 {
		t.Fatal("previous version unsupported", err)
	}
}

func TestRepeatFailureRollsBackCompletionAndMonthlyEditsKeepAnchor(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Home", Prefix: "H", CompletionPolicy: "manual"})
	due := "2028-01-31"
	task, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Monthly", DueDate: &due, RepeatRule: "monthly"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("CREATE TRIGGER fail_repeat BEFORE INSERT ON tickets WHEN NEW.repeat_source_id IS NOT NULL BEGIN SELECT RAISE(ABORT,'disk failure'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.MoveTicket(task.ID, models.MoveTicketRequest{Status: "done"}); err == nil {
		t.Fatal("failed repeat marked parent done")
	}
	unchanged, _ := s.GetTicket(task.ID)
	if unchanged.Status != "todo" || unchanged.Revision != task.Revision {
		t.Fatal("partial completion survived rollback")
	}
	_, _ = s.db.Exec("DROP TRIGGER fail_repeat")
	if _, err = s.MoveTicket(task.ID, models.MoveTicketRequest{Status: "done"}); err != nil {
		t.Fatal(err)
	}
	tickets, _, _ := s.ListTickets(models.TicketFilter{Status: "todo"})
	next := tickets[0]
	feb := next.DueDate.Format("2006-01-02")
	monthly := "monthly"
	note := "Edited notes"
	nextTask, err := s.UpdateTicket(next.ID, models.UpdateTicketRequest{Description: &note, DueDate: &feb, RepeatRule: &monthly})
	if err != nil {
		t.Fatal(err)
	}
	if nextTask.RepeatAnchor != 31 {
		t.Fatal("editing February's notes changed the monthly anchor")
	}
}

func TestRecoveryPreservesExistingContentAndRelationships(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Home", Prefix: "H", CompletionPolicy: "manual"})
	team, _ := s.CreateTeam(models.CreateTeamRequest{Name: "Household"})
	label, _ := s.CreateLabel(models.CreateLabelRequest{Name: "Weekly", Color: "blue"})
	due := "2026-01-31"
	task, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, TeamID: &team.ID, Title: "Plan meals", Description: "Original notes", DueDate: &due, RepeatRule: "monthly", Labels: []string{label.ID}})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.AddSubtask(task.ID, models.CreateSubtaskRequest{Title: "Check pantry"})
	filename := filepath.Join(t.TempDir(), "backup.db")
	if err = s.Backup(filename); err != nil {
		t.Fatal(err)
	}
	snapshot, err := ReadBackup(filename)
	if err != nil {
		t.Fatal(err)
	}
	title := "New work since backup"
	_, _ = s.UpdateTicket(task.ID, models.UpdateTicketRequest{Title: &title})
	ids, err := s.RestoreBoards("request-1", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := s.RestoreBoards("request-1", snapshot)
	if err != nil || len(retry) != 1 || retry[0] != ids[0] {
		t.Fatal("restore retry duplicated boards", err)
	}
	original, _ := s.GetTicket(task.ID)
	if original.Title != title {
		t.Fatal("restore overwrote current task")
	}
	restored, _, err := s.ListTickets(models.TicketFilter{ProjectID: ids[0]})
	if err != nil || len(restored) != 1 {
		t.Fatal(err)
	}
	copy := restored[0]
	if copy.ID == task.ID || copy.Title != "Plan meals" || copy.TeamID == nil || *copy.TeamID == team.ID || len(copy.Labels) != 1 || len(copy.Subtasks) != 1 || copy.RepeatRule != "monthly" {
		t.Fatalf("lost recovery data: %+v", copy)
	}
	// A failed relationship insert must not leave even an empty recovered board.
	snapshot.Tickets[0].Labels = append(snapshot.Tickets[0].Labels, models.Label{ID: "missing"})
	if _, err = s.RestoreBoards("broken", snapshot); err == nil {
		t.Fatal("invalid relationship accepted")
	}
	_, count, _ := s.ListProjects("")
	if count != 2 {
		t.Fatal("partial recovery survived rollback", count)
	}
}

func TestBackupRejectsUntrustedSchemaAndBrokenFiles(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	filename := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(filename, []byte("not a database"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBackup(filename); err == nil {
		t.Fatal("bad file accepted")
	}
	_ = os.Remove(filename)
	if err := s.Backup(filename); err != nil {
		t.Fatal(err)
	}
	other, err := OpenAt(filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = other.Exec("CREATE TRIGGER unexpected AFTER INSERT ON teams BEGIN DELETE FROM tickets; END")
	if err != nil {
		t.Fatal(err)
	}
	_ = other.Close()
	if _, err = ReadBackup(filename); err == nil {
		t.Fatal("untrusted trigger accepted")
	}
}

func TestDeletedTaskRestorePreservesNotesAndRejectsStaleUndo(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Work", Prefix: "W", GitHubRepo: "o/r"})
	task, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Recover", Description: "Keep these notes"})
	_, _ = s.AddSubtask(task.ID, models.CreateSubtaskRequest{Title: "Keep checklist"})
	if err := s.DeleteTicket(task.ID, task.Revision); err != nil {
		t.Fatal(err)
	}
	trash, err := s.Trash()
	if err != nil || len(trash) != 1 {
		t.Fatal(err)
	}
	_, _ = s.AcquireSyncLock(p.ID, "worker")
	if _, err = s.RestoreTicket(task.ID, trash[0].Revision); err == nil {
		t.Fatal("restored during sync")
	}
	_ = s.ReleaseSyncLock(p.ID, "worker")
	recovered, err := s.RestoreTicket(task.ID, trash[0].Revision)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Description != "Keep these notes" || len(recovered.Subtasks) != 1 {
		t.Fatal("undo lost content")
	}
	if _, err = s.RestoreTicket(task.ID, trash[0].Revision); err != nil {
		t.Fatal("retry failed", err)
	}
	_ = s.DeleteTicket(task.ID, recovered.Revision)
	if _, err = s.RestoreTicket(task.ID, trash[0].Revision); err == nil {
		t.Fatal("old Undo restored a newer deletion")
	}
}

func TestRepeatCompletionExactlyOnceAndCalendarAnchors(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Home", Prefix: "H", CompletionPolicy: "manual"})
	due := time.Now().Format("2006-01-02")
	task, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Check budget", DueDate: &due, RepeatRule: "weekly"})
	if err != nil {
		t.Fatal(err)
	}
	sub, _ := s.AddSubtask(task.ID, models.CreateSubtaskRequest{Title: "Check account"})
	_, _ = s.ToggleSubtask(sub.ID)
	done, err := s.MoveTicket(task.ID, models.MoveTicketRequest{Status: "done", ExpectedRevision: &task.Revision})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.MoveTicket(task.ID, models.MoveTicketRequest{Status: "review", ExpectedRevision: &done.Revision})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.MoveTicket(task.ID, models.MoveTicketRequest{Status: "done"})
	if err != nil {
		t.Fatal(err)
	}
	tasks, _, _ := s.ListTickets(models.TicketFilter{})
	if len(tasks) != 2 {
		t.Fatal("duplicate recurring task", len(tasks))
	}
	for _, next := range tasks {
		if next.ID != task.ID {
			if next.Status != "todo" || len(next.Subtasks) != 1 || next.Subtasks[0].Completed || next.Work != nil || next.DueDate == nil || next.RepeatSourceID == nil {
				t.Fatalf("invalid next occurrence: %+v", next)
			}
		}
	}
	jan, _ := time.Parse("2006-01-02", "2028-01-31")
	sample := models.Ticket{RepeatRule: "monthly", RepeatAnchor: 31, DueDate: &jan}
	feb := nextRepeatDate(&sample, jan)
	if feb.Format("2006-01-02") != "2028-02-29" {
		t.Fatal(feb)
	}
	sample.DueDate = &feb
	mar := nextRepeatDate(&sample, feb)
	if mar.Format("2006-01-02") != "2028-03-31" {
		t.Fatal(mar)
	}
	late, _ := time.Parse("2006-01-02", "2028-05-06")
	if nextRepeatDate(&sample, late).Format("2006-01-02") != "2028-05-31" {
		t.Fatal("late completion flooded missed periods")
	}
	empty := ""
	if _, err = s.UpdateTicket(task.ID, models.UpdateTicketRequest{DueDate: &empty}); err == nil {
		t.Fatal("repeating task lost its due date")
	}
}
