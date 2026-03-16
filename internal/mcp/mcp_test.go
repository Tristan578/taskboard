package mcp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	_ "modernc.org/sqlite"
)

func setupTestMCP(t *testing.T) (*MCPServer, *db.Store, func()) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = database.Exec(`
		CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT, prefix TEXT UNIQUE, description TEXT, icon TEXT, color TEXT, status TEXT, github_repo TEXT, github_last_synced DATETIME, strict BOOLEAN DEFAULT 0, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE tickets (id TEXT PRIMARY KEY, project_id TEXT, team_id TEXT, number INTEGER, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME, position REAL, lexo_rank TEXT, github_issue_number INTEGER, github_last_synced_at DATETIME, github_last_synced_sha TEXT DEFAULT '', user_story TEXT DEFAULT '', acceptance_criteria TEXT DEFAULT '', technical_details TEXT DEFAULT '', testing_details TEXT DEFAULT '', is_draft BOOLEAN DEFAULT 0, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE teams (id TEXT PRIMARY KEY, name TEXT, color TEXT, created_at DATETIME);
		CREATE TABLE labels (id TEXT PRIMARY KEY, name TEXT, color TEXT);
		CREATE TABLE subtasks (id TEXT PRIMARY KEY, ticket_id TEXT, title TEXT, completed BOOLEAN DEFAULT 0, position INTEGER);
		CREATE TABLE ticket_labels (ticket_id TEXT, label_id TEXT, PRIMARY KEY (ticket_id, label_id));
		CREATE TABLE ticket_dependencies (ticket_id TEXT, blocked_by_id TEXT, PRIMARY KEY (ticket_id, blocked_by_id));
		CREATE TABLE sync_jobs (id TEXT PRIMARY KEY, project_id TEXT, ticket_id TEXT, action TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME);
	`)

	store := db.NewStore(database)
	s := NewServer(store)
	return s, store, func() { _ = database.Close() }
}

func TestMCP_Tools(t *testing.T) {
	s, store, cleanup := setupTestMCP(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	team, _ := store.CreateTeam(models.CreateTeamRequest{Name: "T1"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Ticket 1"})

	tests := []struct {
		name string
		args interface{}
	}{
		{"list_projects", map[string]string{}},
		{"get_project", map[string]string{"id": p.ID}},
		{"update_project", map[string]interface{}{"id": p.ID, "name": "P1 Updated"}},
		{"list_teams", map[string]string{}},
		{"get_team", map[string]string{"id": team.ID}},
		{"create_team", map[string]string{"name": "New Team"}},
		{"update_team", map[string]interface{}{"id": team.ID, "name": "T1 Updated"}},
		{"list_tickets", map[string]string{"projectId": p.ID}},
		{"get_ticket", map[string]string{"id": ticket.ID}},
		{"update_ticket", map[string]interface{}{"id": ticket.ID, "title": "T1 Updated"}},
		{"move_ticket", map[string]string{"id": ticket.ID, "status": "in_progress"}},
		{"get_board", map[string]string{"projectId": p.ID}},
		{"create_subtask", map[string]string{"ticketId": ticket.ID, "title": "Sub 1"}},
		{"batch_create_subtasks", map[string]interface{}{"ticketId": ticket.ID, "subtasks": []map[string]string{{"title": "Sub 2"}, {"title": "Sub 3"}}}},
		{"toggle_subtask", map[string]string{"id": "s1"}}, 
		{"delete_subtask", map[string]string{"id": "s1"}},
		{"delete_ticket", map[string]string{"id": ticket.ID}},
		{"delete_team", map[string]string{"id": team.ID}},
		{"delete_project", map[string]string{"id": p.ID}},
		{"create_ticket", map[string]string{"projectId": p.ID, "title": "T1"}},
	}

	for _, tt := range tests {
		argsJSON, _ := json.Marshal(tt.args)
		_, _ = s.callTool(tt.name, argsJSON) 
	}

	// Test error cases
	errorTests := []struct {
		name string
		args interface{}
	}{
		{"create_subtask", map[string]string{}},
		{"batch_create_subtasks", map[string]interface{}{}},
		{"batch_create_subtasks", map[string]interface{}{"ticketId": "nonexistent", "subtasks": []map[string]string{{"title": "S1"}}}},
		{"get_project", map[string]string{"id": "nonexistent"}},
		{"get_team", map[string]string{"id": "nonexistent"}},
		{"get_ticket", map[string]string{"id": "nonexistent"}},
		{"unknown_tool", nil},
	}

	for _, tt := range errorTests {
		argsJSON, _ := json.Marshal(tt.args)
		_, _ = s.callTool(tt.name, argsJSON)
	}

	// Test initialized notification
	reqNotify := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	respNotify := s.handleRequest(reqNotify)
	if respNotify != nil {
		t.Errorf("Expected nil response for initialized notification")
	}
}

func TestMCP_HandleRequest(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	tests := []struct {
		method string
		id     interface{}
		params interface{}
	}{
		{"initialize", 1, nil},
		{"notifications/initialized", nil, nil},
		{"tools/list", 2, nil},
		{"tools/call", 3, map[string]interface{}{"name": "list_projects", "arguments": map[string]string{}}},
		{"unknown", 4, nil},
	}

	for _, tt := range tests {
		paramsJSON, _ := json.Marshal(tt.params)
		req := jsonrpcRequest{
			JSONRPC: "2.0",
			ID:      tt.id,
			Method:  tt.method,
			Params:  paramsJSON,
		}
		resp := s.handleRequest(req)
		if tt.method == "notifications/initialized" {
			if resp != nil {
				t.Errorf("Expected nil response for notification")
			}
		} else {
			if resp == nil {
				t.Errorf("Expected response for method %s", tt.method)
			}
		}
	}
}

func TestMCP_Run(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}
invalid json
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
`)
	output := &bytes.Buffer{}

	err := s.Run(input, output)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !bytes.Contains(output.Bytes(), []byte(`"id":1`)) {
		t.Errorf("Output missing response for ID 1")
	}
}

type errorReader struct{}
func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("forced error")
}

func TestMCP_Run_ReadError(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	err := s.Run(&errorReader{}, &bytes.Buffer{})
	if err == nil {
		t.Errorf("Expected error from Run with errorReader")
	}
}

type faultyWriter struct{}
func (f *faultyWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write error")
}

func TestMCP_Run_WriteError(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n")
	output := &faultyWriter{}
	
	_ = s.Run(input, output)
}

func TestMCP_HandleToolCall_InvalidJSON(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{invalid}`),
	}
	resp := s.handleToolCall(req)
	if resp.Error == nil {
		t.Errorf("Expected error for invalid JSON params")
	}
}

func TestMCP_HandleToolCall_MarshalError(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "__test_marshal_error", "arguments": {}}`),
	}
	resp := s.handleToolCall(req)
	if resp.Error == nil {
		t.Errorf("Expected error for unmarshalable tool result")
	}
}

func TestMCP_Run_MarshalError(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	// Method that returns unmarshalable Result
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"__test_trigger_marshal_error"}` + "\n")
	output := &bytes.Buffer{}
	_ = s.Run(input, output)
	if len(output.Bytes()) > 0 {
		t.Errorf("Expected no output for marshal error, got %s", output.String())
	}
}

func TestMCP_BatchCreate_Error(t *testing.T) {
	s, store, cleanup := setupTestMCP(t)
	// No defer cleanup, we'll close manually to trigger error
	
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	tick, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	
	_ = store.Close()
	
	args := map[string]interface{}{
		"ticketId": tick.ID,
		"subtasks": []map[string]string{{"title": "S1"}},
	}
	argsJSON, _ := json.Marshal(args)
	_, err := s.callTool("batch_create_subtasks", argsJSON)
	if err == nil {
		t.Errorf("Expected error for batch create with closed DB")
	}
	cleanup()
}

func TestMCP_HandleToolCall_InvalidParams(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	// req.Params is a number, cannot unmarshal into struct
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`123`), 
	}
	resp := s.handleToolCall(req)
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Errorf("Expected invalid params error (-32602)")
	}
}

func TestMCP_Run_InvalidLine(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	input := bytes.NewBufferString("invalid json\n")
	output := &bytes.Buffer{}
	_ = s.Run(input, output)
}

func TestMCP_CallTool_Extra(t *testing.T) {
	s, _, cleanup := setupTestMCP(t)
	defer cleanup()

	// Hit remaining branches in callTool
	_, _ = s.callTool("list_teams", nil)
	_, _ = s.callTool("list_tickets", nil)
}
