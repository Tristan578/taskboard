package cli

import (
	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivityCLIAndDiscovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	database, err := db.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := db.NewStore(database)
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Task"})
	for _, args := range [][]string{
		{"--db", path, "activity", ticket.ID, "--agent", "Codex", "--session", "run", "--summary", "Working"},
		{"--db", path, "status", "--port", "1"},
	} {
		root := NewRootCmd(nil)
		root.SetOut(io.Discard)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	current, _ := store.GetTicket(ticket.ID)
	if current.Work == nil || current.Work.Agent != "Codex" {
		t.Fatal("activity not persisted")
	}
	root := NewRootCmd(nil)
	root.SetArgs([]string{"--db", path, "ticket", "complete", ticket.ID, "--evidence", "https://github.com/o/r/pull/1"})
	t.Setenv("GITHUB_TOKEN", "")
	if err := root.Execute(); err == nil {
		t.Fatal("completion accepted without verification")
	}
}
func TestAgentInstructionsPreserveUserContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	original := "# My rules\nNever remove these.\n# Player2 Kanban: custom notes\nKeep these too.\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	target := agentTarget{path: path}
	if err := installAgentConfig("codex", target, []byte("Workflow v1")); err != nil {
		t.Fatal(err)
	}
	if err := installAgentConfig("codex", target, []byte("Workflow v2")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), original) || strings.Count(string(data), "<!-- player2:start -->") != 1 || strings.Contains(string(data), "Workflow v1") {
		t.Fatal(string(data))
	}
}
