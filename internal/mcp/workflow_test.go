package mcp

import (
	"encoding/json"
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
