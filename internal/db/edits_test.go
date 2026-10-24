package db

import (
	"errors"
	"github.com/Tristan578/taskboard/internal/models"
	"testing"
)

func TestStaleEditsAndCompletionPreserveNewerWork(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Work", Prefix: "W", CompletionPolicy: "manual"})
	task, _ := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Original"})
	agentTitle := "Agent's newer work"
	updated, err := s.UpdateTicket(task.ID, models.UpdateTicketRequest{Title: &agentTitle})
	if err != nil {
		t.Fatal(err)
	}
	staleTitle := "Old browser draft"
	cases := []func() error{
		func() error {
			_, e := s.UpdateTicket(task.ID, models.UpdateTicketRequest{Title: &staleTitle, ExpectedRevision: &task.Revision})
			return e
		},
		func() error {
			_, e := s.MoveTicket(task.ID, models.MoveTicketRequest{Status: "done", ExpectedRevision: &task.Revision})
			return e
		},
		func() error { return s.DeleteTicket(task.ID, task.Revision) },
		func() error {
			_, e := s.CompleteVerified(task.ID, "https://github.com/o/r/pull/1", task.Revision)
			return e
		},
	}
	for _, run := range cases {
		var conflict *ConflictError
		if err := run(); !errors.As(err, &conflict) {
			t.Fatalf("expected conflict, got %v", err)
		}
	}
	saved, _ := s.GetTicket(task.ID)
	if saved.Title != agentTitle || saved.Status != "todo" || saved.Revision != updated.Revision {
		t.Fatalf("newer work lost: %+v", saved)
	}
	completed, err := s.CompleteVerified(task.ID, "https://github.com/o/r/pull/1", saved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Revision <= saved.Revision {
		t.Fatal("completion did not invalidate stale editors")
	}
}
func TestEditFieldsAndRelationshipsCommitTogether(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	p, _ := s.CreateProject(models.CreateProjectRequest{Name: "Work", Prefix: "W"})
	team, _ := s.CreateTeam(models.CreateTeamRequest{Name: "Team"})
	due := "2026-10-23"
	task, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Task", DueDate: &due, TeamID: &team.ID})
	if err != nil {
		t.Fatal(err)
	}
	empty := ""
	saved, err := s.UpdateTicket(task.ID, models.UpdateTicketRequest{DueDate: &empty, TeamID: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if saved.DueDate != nil || saved.TeamID != nil {
		t.Fatal("could not clear optional fields")
	}
	title := "Partial write must not survive"
	if _, err = s.UpdateTicket(task.ID, models.UpdateTicketRequest{Title: &title, Labels: []string{"nonexistent-label"}}); err == nil {
		t.Fatal("missing label accepted")
	}
	unchanged, _ := s.GetTicket(task.ID)
	if unchanged.Title != task.Title || unchanged.Revision != saved.Revision {
		t.Fatal("partial update survived rollback")
	}
	invalidDate := "2026-02-30"
	if _, err = s.UpdateTicket(task.ID, models.UpdateTicketRequest{DueDate: &invalidDate}); err == nil {
		t.Fatal("invalid date accepted")
	}
}
