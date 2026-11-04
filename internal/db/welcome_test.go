package db

import (
	"github.com/Tristan578/taskboard/internal/models"
	"path/filepath"
	"testing"
)

func TestGuidedWorkspaceRetryAndCompletion(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	r := WorkspaceRequest{RequestID: "unique-request-12345", Name: "Home base", Goal: "A calmer week", CompletionPolicy: "manual", Tasks: []StarterTask{{Title: "Plan meals"}, {Title: "Check calendar"}}}
	p, err := s.CreateWorkspace(r)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CreateWorkspace(r)
	if err != nil || again.ID != p.ID {
		t.Fatalf("retry: %v %+v", err, again)
	}
	board, err := s.GetBoard(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var id string
	for _, c := range board.Columns {
		for _, task := range c.Tickets {
			count++
			id = task.ID
		}
	}
	if count != 2 {
		t.Fatalf("duplicate or missing tasks: %d", count)
	}
	if _, err = s.MoveTicket(id, models.MoveTicketRequest{Status: "done"}); err != nil {
		t.Fatal(err)
	}
	repo := "owner/repo"
	if _, err = s.UpdateProject(p.ID, models.UpdateProjectRequest{GitHubRepo: &repo}); err == nil {
		t.Fatal("everyday board silently became GitHub board")
	}
	r.RequestID = "software-request-12345"
	r.CompletionPolicy = "verified"
	r.Name = "Ship it"
	p, err = s.CreateWorkspace(r)
	if err != nil {
		t.Fatal(err)
	}
	board, _ = s.GetBoard(p.ID)
	id = board.Columns[0].Tickets[0].ID
	if _, err = s.MoveTicket(id, models.MoveTicketRequest{Status: "done"}); err == nil {
		t.Fatal("software bypassed verification")
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err = s.Backup(backup); err != nil {
		t.Fatal(err)
	}
	snapshot, err := OpenAt(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()
	projects, total, err := NewStore(snapshot).ListProjects("")
	if err != nil || total != 2 || len(projects) != 2 {
		t.Fatalf("backup missing data: %d %v", total, err)
	}
}
func TestGuidedWorkspaceValidationIsAtomic(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	cases := []WorkspaceRequest{
		{RequestID: "short", Name: "Home", CompletionPolicy: "manual"},
		{RequestID: "long-enough-request", Name: " ", CompletionPolicy: "manual"},
		{RequestID: "long-enough-request", Name: "Home", CompletionPolicy: "other"},
		{RequestID: "long-enough-request", Name: "Home", CompletionPolicy: "manual", Tasks: []StarterTask{{Title: "Valid"}, {Title: " "}}},
	}
	for _, r := range cases {
		if _, err := s.CreateWorkspace(r); err == nil {
			t.Fatalf("accepted invalid workspace: %+v", r)
		}
	}
	_, total, err := s.ListProjects("")
	if err != nil || total != 0 {
		t.Fatalf("partial board: %d %v", total, err)
	}
}
