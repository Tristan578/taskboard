package cli

import (
	"context"
	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"io"
	"testing"
	"time"
)

func TestTrackedCommandLifecycle(t *testing.T) {
	database, err := db.OpenAt(t.TempDir() + "/run.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	store := db.NewStore(database)
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	for _, tc := range []struct {
		name   string
		args   []string
		status string
		fail   bool
	}{
		{"success", []string{"go", "version"}, "review", false},
		{"missing executable", []string{"player2-nonexistent-command"}, "blocked", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: tc.name})
			err := runTracked(context.Background(), store, ticket.ID, "Codex", tc.args, nil, io.Discard, io.Discard, time.Millisecond)
			if (err != nil) != tc.fail {
				t.Fatalf("result %v", err)
			}
			updated, _ := store.GetTicket(ticket.ID)
			if updated.Status != tc.status || updated.Work == nil || updated.Work.Agent != "Codex" {
				t.Fatalf("tracking failed: %+v", updated)
			}
		})
	}
}
