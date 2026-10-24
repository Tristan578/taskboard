package db

import (
	"github.com/Tristan578/taskboard/internal/models"
	"testing"
	"time"
)

func TestSharedWorkflowRules(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, err := s.CreateProject(models.CreateProjectRequest{Name: "Test", Prefix: "T", Strict: true, GitHubRepo: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Missing requirements"}); err == nil {
		t.Fatal("strict create bypassed")
	}
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Idea", IsDraft: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.MoveTicket(ticket.ID, models.MoveTicketRequest{Status: "in_progress"}); err == nil {
		t.Fatal("draft gate bypassed")
	}
	us, ac := "As a person", "Given work When it completes Then review it"
	if _, err = s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{UserStory: &us, AcceptanceCriteria: &ac}); err != nil {
		t.Fatal(err)
	}
	ticket, err = s.MoveTicket(ticket.ID, models.MoveTicketRequest{Status: "in_progress"})
	if err != nil {
		t.Fatal(err)
	}
	if ticket.IsDraft {
		t.Fatal("not published")
	}
	for _, status := range []string{"done", "unknown"} {
		if _, err = s.MoveTicket(ticket.ID, models.MoveTicketRequest{Status: status}); err == nil {
			t.Fatalf("accepted %s", status)
		}
	}
	if _, err = s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Fake done", Status: "done", UserStory: us, AcceptanceCriteria: ac}); err == nil {
		t.Fatal("created done without evidence")
	}
	jobs, err := s.GetPendingSyncJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("publication not queued: %v %v", jobs, err)
	}
	if _, err = s.CompleteVerified(ticket.ID, "https://github.com/o/r/pull/1"); err != nil {
		t.Fatal(err)
	}
	ticket, err = s.GetTicket(ticket.ID)
	if err != nil || ticket.Status != "done" || ticket.Work.CompletionURL == "" {
		t.Fatal("completion evidence lost", err)
	}
}

func TestActivityLeaseAndStaleness(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Test", Prefix: "T"})
	ticket, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Work"})
	a := models.ActivityRequest{Agent: "Codex", SessionID: "one", Summary: "Investigating"}
	if _, err := s.ReportActivity(ticket.ID, models.ActivityRequest{}); err == nil {
		t.Fatal("empty heartbeat accepted")
	}
	if _, err := s.ReportActivity("missing", a); err == nil {
		t.Fatal("missing ticket accepted")
	}
	if _, err := s.ReportActivity(ticket.ID, a); err != nil {
		t.Fatal(err)
	}
	other := a
	other.SessionID = "two"
	if _, err := s.ReportActivity(ticket.ID, other); err == nil {
		t.Fatal("stole active session")
	}
	if _, err := s.ReportActivity(ticket.ID, a); err != nil {
		t.Fatal("owner heartbeat rejected", err)
	}
	if _, err := s.db.Exec("UPDATE work_state SET heartbeat_at=? WHERE ticket_id=?", time.Now().Add(-16*time.Minute), ticket.ID); err != nil {
		t.Fatal(err)
	}
	state, err := s.GetWorkState(ticket.ID)
	if err != nil || !state.Stale {
		t.Fatal("stale not detected", err)
	}
	if _, err := s.ReportActivity(ticket.ID, other); err != nil {
		t.Fatal("stale lease cannot be reclaimed", err)
	}
	if err := s.DeleteTicket(ticket.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReportActivity(ticket.ID, a); err == nil {
		t.Fatal("deleted ticket accepted")
	}
}

func TestSyncLockAndRetry(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	if ok, err := s.AcquireSyncLock("p", "one"); err != nil || !ok {
		t.Fatal(err)
	}
	if ok, err := s.AcquireSyncLock("p", "two"); err != nil || ok {
		t.Fatal("lock stolen", err)
	}
	if err := s.ReleaseSyncLock("p", "two"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.AcquireSyncLock("p", "two"); ok {
		t.Fatal("nonowner unlocked")
	}
	if err := s.ReleaseSyncLock("p", "one"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.AcquireSyncLock("p", "two"); !ok {
		t.Fatal("lock not released")
	}
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Test", Prefix: "T"})
	if err := s.QueueSyncJob(p.ID, "", "full_sync", nil); err != nil {
		t.Fatal(err)
	}
	jobs, _ := s.GetPendingSyncJobs()
	if len(jobs) != 1 {
		t.Fatal(jobs)
	}
	if err := s.UpdateSyncJobRetry(jobs[0].ID, 5, "offline", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.GetPendingSyncJobs(); len(jobs) != 0 {
		t.Fatal("exhausted retry ran")
	}
	if err := s.RetryFailedSyncJobs(); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.GetPendingSyncJobs(); len(jobs) != 1 {
		t.Fatal("manual retry not scheduled")
	}
}

func TestStaleSyncRevisionDoesNotOverwriteUserEdit(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	ticket, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Original"})
	revision := ticket.Revision
	userTitle := "User change"
	remoteTitle := "Stale remote change"
	if _, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Title: &userTitle}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Title: &remoteTitle, ExpectedRevision: &revision}); err == nil {
		t.Fatal("stale snapshot overwrote edit")
	}
	current, _ := s.GetTicket(ticket.ID)
	if current.Title != userTitle {
		t.Fatal("user edit lost")
	}
}
