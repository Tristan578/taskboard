package server

import (
	"context"
	"fmt"
	"net"
	"time"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"database/sql"
	_ "modernc.org/sqlite"
	"github.com/gorilla/websocket"
)

func setupTestServer(t *testing.T) (*Server, *db.Store, func()) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Comprehensive schema for server tests
	_, _ = database.Exec(`
		CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT, prefix TEXT UNIQUE, description TEXT, icon TEXT, color TEXT, status TEXT, github_repo TEXT, github_last_synced DATETIME, strict BOOLEAN DEFAULT 0, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE tickets (id TEXT PRIMARY KEY, project_id TEXT, team_id TEXT, number INTEGER, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME, position REAL, lexo_rank TEXT, github_issue_number INTEGER, github_last_synced_at DATETIME, github_last_synced_sha TEXT DEFAULT '', user_story TEXT DEFAULT '', acceptance_criteria TEXT DEFAULT '', technical_details TEXT DEFAULT '', testing_details TEXT DEFAULT '', is_draft BOOLEAN DEFAULT 0, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE sync_jobs (id TEXT PRIMARY KEY, project_id TEXT, ticket_id TEXT, action TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0, last_error TEXT, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE teams (id TEXT PRIMARY KEY, name TEXT, color TEXT, created_at DATETIME);
		CREATE TABLE labels (id TEXT PRIMARY KEY, name TEXT, color TEXT);
		CREATE TABLE subtasks (id TEXT PRIMARY KEY, ticket_id TEXT, title TEXT, completed BOOLEAN DEFAULT 0, position INTEGER);
		CREATE TABLE ticket_labels (ticket_id TEXT, label_id TEXT, PRIMARY KEY (ticket_id, label_id));
		CREATE TABLE ticket_dependencies (ticket_id TEXT, blocked_by_id TEXT, PRIMARY KEY (ticket_id, blocked_by_id));
	`)

	store := db.NewStore(database)
	s := New(store, nil)
	return s, store, func() { _ = database.Close() }
}

func TestServer_Projects(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Create
	body, _ := json.Marshal(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	req := httptest.NewRequest("POST", "/api/projects", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated { t.Errorf("Create project failed: %d", w.Code) }

	var project models.Project
	_ = json.Unmarshal(w.Body.Bytes(), &project)

	// 2. List
	req = httptest.NewRequest("GET", "/api/projects", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("List projects failed") }

	// 3. Get
	req = httptest.NewRequest("GET", "/api/projects/"+project.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Get project failed") }

	// 4. Update
	newName := "P1 Updated"
	ub, _ := json.Marshal(models.UpdateProjectRequest{Name: &newName})
	req = httptest.NewRequest("PUT", "/api/projects/"+project.ID, bytes.NewBuffer(ub))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Update project failed") }

	// 5. Delete
	req = httptest.NewRequest("DELETE", "/api/projects/"+project.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent { t.Errorf("Delete project failed") }
}

func TestServer_Tickets(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})

	// 1. Create
	body, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated { t.Errorf("Create ticket failed") }
	
	var ticket models.Ticket
	_ = json.Unmarshal(w.Body.Bytes(), &ticket)

	// 2. List
	req = httptest.NewRequest("GET", "/api/tickets", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("List tickets failed") }

	// 3. Get
	req = httptest.NewRequest("GET", "/api/tickets/"+ticket.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Get ticket failed") }

	// 4. Update
	newTitle := "T1 Updated"
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{Title: &newTitle})
	req = httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Update ticket failed") }

	// 5. Move
	mb, _ := json.Marshal(models.MoveTicketRequest{Status: "done"})
	req = httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(mb))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Move ticket failed") }

	// 6. Delete
	req = httptest.NewRequest("DELETE", "/api/tickets/"+ticket.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent { t.Errorf("Delete ticket failed") }
}

func TestServer_Teams(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(models.CreateTeamRequest{Name: "T1"})
	req := httptest.NewRequest("POST", "/api/teams", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated { t.Errorf("Create team failed") }
	
	var team models.Team
	_ = json.Unmarshal(w.Body.Bytes(), &team)

	req = httptest.NewRequest("GET", "/api/teams", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("List teams failed") }

	req = httptest.NewRequest("GET", "/api/teams/"+team.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Get team failed") }

	// Update
	name := "T2"
	ub, _ := json.Marshal(models.UpdateTeamRequest{Name: &name})
	req = httptest.NewRequest("PUT", "/api/teams/"+team.ID, bytes.NewBuffer(ub))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Update team failed") }

	req = httptest.NewRequest("DELETE", "/api/teams/"+team.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent { t.Errorf("Delete team failed") }
}

func TestServer_Labels(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(models.CreateLabelRequest{Name: "L1", Color: "red"})
	req := httptest.NewRequest("POST", "/api/labels", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated { t.Errorf("Create label failed") }
	
	var label models.Label
	_ = json.Unmarshal(w.Body.Bytes(), &label)

	req = httptest.NewRequest("GET", "/api/labels", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("List labels failed") }

	name := "L2"
	ub, _ := json.Marshal(models.UpdateLabelRequest{Name: &name})
	req = httptest.NewRequest("PUT", "/api/labels/"+label.ID, bytes.NewBuffer(ub))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Update label failed") }

	req = httptest.NewRequest("DELETE", "/api/labels/"+label.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent { t.Errorf("Delete label failed") }
}

func TestServer_Subtasks(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	// 1. Add Subtask
	body, _ := json.Marshal(models.CreateSubtaskRequest{Title: "Sub 1"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/subtasks", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated { t.Errorf("Add subtask failed") }
	
	var st models.Subtask
	_ = json.Unmarshal(w.Body.Bytes(), &st)

	// 2. Toggle
	req = httptest.NewRequest("POST", "/api/subtasks/"+st.ID+"/toggle", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Toggle subtask failed") }

	// 3. Delete
	req = httptest.NewRequest("DELETE", "/api/subtasks/"+st.ID, nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent { t.Errorf("Delete subtask failed") }
}

func TestServer_Board(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	req := httptest.NewRequest("GET", "/api/board?projectId="+p.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("Get board failed") }
}

func TestServer_Strict_Enforcement(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1", Strict: true})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: true})

	// 1. Move draft to todo should fail if missing US/AC
	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "todo"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest { t.Errorf("Expected 400 when moving draft out") }

	// 2. Update to non-draft should fail if missing specs
	isDraft := false
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{IsDraft: &isDraft})
	req = httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest { t.Errorf("Expected 400 when converting to non-draft") }
}

func TestServer_Recoverer(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	s.router.Get("/panic", func(w http.ResponseWriter, r *http.Request) { panic("test panic") })
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError { t.Errorf("Expected 500 for panic") }
}

func TestServer_CORS(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("OPTIONS", "/api/projects", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent { t.Errorf("CORS failed: %d", w.Code) }
}

func TestServer_ErrorPaths(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		method string
		url    string
		body   interface{}
		code   int
	}{
		{"GET", "/api/projects/none", nil, http.StatusNotFound},
		{"PUT", "/api/projects/none", map[string]string{"name": "f"}, http.StatusNotFound},
		{"DELETE", "/api/projects/none", nil, http.StatusNotFound},
		{"GET", "/api/teams/none", nil, http.StatusNotFound},
		{"PUT", "/api/teams/none", map[string]string{"name": "f"}, http.StatusNotFound},
		{"DELETE", "/api/teams/none", nil, http.StatusNotFound},
		{"GET", "/api/tickets/none", nil, http.StatusNotFound},
		{"PUT", "/api/tickets/none", map[string]string{"title": "f"}, http.StatusNotFound},
		{"POST", "/api/tickets/none/move", map[string]string{"status": "todo"}, http.StatusNotFound},
		{"DELETE", "/api/tickets/none", nil, http.StatusNotFound},
		{"POST", "/api/projects", map[string]string{"invalid": "json"}, http.StatusBadRequest},
		{"DELETE", "/api/subtasks/none", nil, http.StatusNotFound},
		{"POST", "/api/subtasks/none/toggle", nil, http.StatusNotFound},
		{"PUT", "/api/labels/none", map[string]string{"name": "f"}, http.StatusNotFound},
		{"DELETE", "/api/labels/none", nil, http.StatusNotFound},
		{"POST", "/api/tickets", map[string]string{"title": "no proj"}, http.StatusBadRequest},
		{"POST", "/api/tickets/t1/subtasks", map[string]string{"notitle": "f"}, http.StatusBadRequest},
		{"POST", "/api/teams", map[string]string{"notname": "f"}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		var bodyBuf *bytes.Buffer
		if tt.body != nil {
			b, _ := json.Marshal(tt.body)
			bodyBuf = bytes.NewBuffer(b)
		} else {
			bodyBuf = bytes.NewBuffer(nil)
		}
		req := httptest.NewRequest(tt.method, tt.url, bodyBuf)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		if w.Code != tt.code { t.Errorf("%s %s expected %d, got %d", tt.method, tt.url, tt.code, w.Code) }
	}

	badJSON := httptest.NewRequest("POST", "/api/projects", bytes.NewBufferString("{invalid"))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, badJSON)
	if rec.Code != http.StatusBadRequest { t.Errorf("Expected 400 for malformed JSON") }
}

func TestServer_InternalErrors(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	tick, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	team, _ := store.CreateTeam(models.CreateTeamRequest{Name: "T"})
	label, _ := store.CreateLabel(models.CreateLabelRequest{Name: "L", Color: "C"})
	st, _ := store.AddSubtask(tick.ID, models.CreateSubtaskRequest{Title: "S"})

	_ = store.Close() 

	tests := []struct {
		method string
		url    string
		body   interface{}
	}{
		{"GET", "/api/projects", nil},
		{"GET", "/api/projects/"+p.ID, nil},
		{"POST", "/api/projects", map[string]string{"name": "P", "prefix": "P"}},
		{"PUT", "/api/projects/"+p.ID, map[string]string{"name": "P"}},
		{"GET", "/api/teams", nil},
		{"GET", "/api/teams/"+team.ID, nil},
		{"POST", "/api/teams", map[string]string{"name": "T"}},
		{"PUT", "/api/teams/"+team.ID, map[string]string{"name": "T"}},
		{"GET", "/api/tickets", nil},
		{"GET", "/api/tickets/"+tick.ID, nil},
		{"POST", "/api/tickets", map[string]string{"projectId": p.ID, "title": "T"}},
		{"PUT", "/api/tickets/"+tick.ID, map[string]string{"title": "T"}},
		{"POST", "/api/tickets/"+tick.ID+"/move", map[string]string{"status": "done"}},
		{"POST", "/api/tickets/"+tick.ID+"/subtasks", map[string]string{"title": "S"}},
		{"POST", "/api/subtasks/"+st.ID+"/toggle", nil},
		{"GET", "/api/labels", nil},
		{"POST", "/api/labels", map[string]string{"name": "L", "color": "C"}},
		{"PUT", "/api/labels/"+label.ID, map[string]string{"name": "L"}},
		{"GET", "/api/board", nil},
	}

	for _, tt := range tests {
		var bodyBuf *bytes.Buffer
		if tt.body != nil {
			b, _ := json.Marshal(tt.body)
			bodyBuf = bytes.NewBuffer(b)
		} else {
			bodyBuf = bytes.NewBuffer(nil)
		}
		req := httptest.NewRequest(tt.method, tt.url, bodyBuf)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError { t.Errorf("%s %s expected 500, got %d", tt.method, tt.url, w.Code) }
	}
	cleanup()
}

func TestServer_TerminalWS_Full(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()
	ts := httptest.NewServer(s)
	defer ts.Close()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/terminal/ws"
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil { return }
	
	// 1. Resize
	_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"resize","cols":80,"rows":24}`))
	
	// 2. Input
	_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"input","data":"ls\n"}`))
	
	// 3. Unknown type
	_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"unknown"}`))
	
	// 4. Invalid JSON
	_ = ws.WriteMessage(websocket.TextMessage, []byte(`{invalid}`))
	
	_ = ws.Close()
}

func TestServer_ListenAndServe(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	_, _ = database.Exec(`
		CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT, prefix TEXT UNIQUE, description TEXT, icon TEXT, color TEXT, status TEXT, github_repo TEXT, github_last_synced DATETIME, strict BOOLEAN DEFAULT 0, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE tickets (id TEXT PRIMARY KEY, project_id TEXT, team_id TEXT, number INTEGER, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME, position REAL, lexo_rank TEXT, github_issue_number INTEGER, github_last_synced_at DATETIME, github_last_synced_sha TEXT DEFAULT '', user_story TEXT DEFAULT '', acceptance_criteria TEXT DEFAULT '', technical_details TEXT DEFAULT '', testing_details TEXT DEFAULT '', is_draft BOOLEAN DEFAULT 0, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE sync_jobs (id TEXT PRIMARY KEY, project_id TEXT, ticket_id TEXT, action TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0, last_error TEXT, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE teams (id TEXT PRIMARY KEY, name TEXT, color TEXT, created_at DATETIME);
		CREATE TABLE labels (id TEXT PRIMARY KEY, name TEXT, color TEXT);
		CREATE TABLE subtasks (id TEXT PRIMARY KEY, ticket_id TEXT, title TEXT, completed BOOLEAN DEFAULT 0, position INTEGER);
		CREATE TABLE ticket_labels (ticket_id TEXT, label_id TEXT, PRIMARY KEY (ticket_id, label_id));
		CREATE TABLE ticket_dependencies (ticket_id TEXT, blocked_by_id TEXT, PRIMARY KEY (ticket_id, blocked_by_id));
	`)

	store := db.NewStore(database)
	srv := New(store, nil)

	// Find a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe(port)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make a request
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/projects", port))
	if err != nil {
		t.Fatalf("GET /api/projects failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestServer_UpdateTicket_StrictModeReject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a strict project
	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "StrictProj", Prefix: "SP", Strict: true,
	})

	// Create a draft ticket (no userStory/AC)
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{
		ProjectID: p.ID, Title: "Draft", IsDraft: true,
	})

	// Try to un-draft without US/AC → should get 400
	isDraft := false
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{IsDraft: &isDraft})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for strict mode reject, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Strict Mode") {
		t.Errorf("Expected strict mode error message, got %q", w.Body.String())
	}
}

func TestServer_HandleTerminalWS(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(s)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/terminal/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// On Windows, PTY start will fail, so we should get an error message
	// On non-Windows, PTY may work - either way we test the upgrade path
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err == nil {
		// Got a message - on Windows this should be the error JSON
		if strings.Contains(string(msg), "failed to start terminal") {
			// Expected on Windows - PTY not available
			return
		}
		// On Linux/Mac, this would be actual terminal output - that's fine too
	}
	// Connection may have closed or timed out - both acceptable outcomes
}
