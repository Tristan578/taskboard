package mcp

import (
	"encoding/json"
	"errors"
	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"testing"
)

func TestAgentWorkflowTools(t *testing.T) {
	server, store, cleanup := setupTestMCP(t)
	defer cleanup()
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "S", Strict: true})
	bad, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "Missing acceptance"})
	if _, err := server.callTool("create_ticket", bad); err == nil {
		t.Fatal("MCP bypassed strict mode")
	}
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Task", UserStory: "As a user", AcceptanceCriteria: "Given work"})
	args, _ := json.Marshal(map[string]string{"id": ticket.ID, "agent": "Claude", "sessionId": "run", "summary": "Investigating"})
	if _, err := server.callTool("report_activity", args); err != nil {
		t.Fatal(err)
	}
	if _, err := server.callTool("report_activity", json.RawMessage(`invalid`)); err == nil {
		t.Fatal("invalid input accepted")
	}
	args, _ = json.Marshal(map[string]string{"id": ticket.ID, "status": "done"})
	if _, err := server.callTool("move_ticket", args); err == nil {
		t.Fatal("MCP completed without evidence")
	}
	if _, err := server.callTool("player2_status", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range server.toolDefinitions() {
		names[tool.Name] = true
	}
	for _, name := range []string{"player2_status", "report_activity", "complete_ticket"} {
		if !names[name] {
			t.Fatal("tool undiscoverable", name)
		}
	}
}

func TestMCPStaleEditorsCannotOverwriteWork(t *testing.T) {
	server, store, cleanup := setupTestMCP(t)
	defer cleanup()
	t.Setenv("GITHUB_TOKEN", "test-token-no-network")
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Work", Prefix: "W", CompletionPolicy: "manual"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Original"})
	title := "Newer work"
	if _, err := store.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Title: &title}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"update_ticket", "move_ticket", "delete_ticket", "complete_ticket"} {
		args, _ := json.Marshal(map[string]any{"id": ticket.ID, "expectedRevision": ticket.Revision, "title": "Stale edit", "status": "done", "evidence": "https://github.com/o/r/pull/1"})
		var conflict *db.ConflictError
		if _, err := server.callTool(name, args); !errors.As(err, &conflict) {
			t.Fatalf("%s did not reject stale input: %v", name, err)
		}
		if _, err := server.callTool(name, json.RawMessage(`invalid`)); err == nil {
			t.Fatalf("%s accepted invalid JSON", name)
		}
		for _, tool := range server.toolDefinitions() {
			if tool.Name == name && tool.InputSchema.Properties["expectedRevision"].Type != "integer" {
				t.Fatalf("%s does not advertise revision checking", name)
			}
		}
	}
	saved, _ := store.GetTicket(ticket.ID)
	if saved == nil || saved.Title != title || saved.Status != "todo" {
		t.Fatal("agent overwrote newer work")
	}
}
