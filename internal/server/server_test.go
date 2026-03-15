package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
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

// ============================================================
// Gap 1: Strict Mode Edge Cases
// ============================================================

func TestServer_StrictMode_CreateNonDraftWithoutUSAC(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "STR", Strict: true})

	body, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: false})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for non-draft ticket without US/AC in strict project, got %d", w.Code)
	}
}

func TestServer_StrictMode_CreateDraftWithoutUSAC(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "STR", Strict: true})

	body, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: true})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201 for draft ticket without US/AC in strict project, got %d", w.Code)
	}
}

func TestServer_StrictMode_CreateNonDraftWithUSAC(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "STR", Strict: true})

	body, _ := json.Marshal(models.CreateTicketRequest{
		ProjectID:          p.ID,
		Title:              "T1",
		IsDraft:            false,
		UserStory:          "As a user...",
		AcceptanceCriteria: "Given... When... Then...",
	})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201 for non-draft ticket with US/AC in strict project, got %d", w.Code)
	}
}

func TestServer_StrictMode_UpdateDraftToNonDraftWithUSAC(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "STR", Strict: true})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: true})

	// First add US/AC
	us := "As a user..."
	ac := "Given... When... Then..."
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{UserStory: &us, AcceptanceCriteria: &ac})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 when adding US/AC to draft, got %d", w.Code)
	}

	// Now convert to non-draft
	isDraft := false
	updateBody2, _ := json.Marshal(models.UpdateTicketRequest{IsDraft: &isDraft})
	req = httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody2))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 when converting draft to non-draft with US/AC, got %d", w.Code)
	}
}

func TestServer_StrictMode_ClearUSWhileNonDraft(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "STR", Strict: true})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{
		ProjectID:          p.ID,
		Title:              "T1",
		IsDraft:            false,
		UserStory:          "As a user...",
		AcceptanceCriteria: "Given...",
	})

	emptyUS := ""
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{UserStory: &emptyUS})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when clearing US on non-draft in strict project, got %d", w.Code)
	}
}

// ============================================================
// Gap 2: Sync Job Queueing
// ============================================================

func TestServer_SyncJob_CreateNonDraftInGithubProject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "GH", Prefix: "GH", GitHubRepo: "owner/repo",
	})

	body, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	jobs, err := store.GetPendingSyncJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) == 0 {
		t.Error("Expected sync job to be queued for non-draft ticket in GitHub-linked project")
	}
}

func TestServer_SyncJob_CreateDraftInGithubProject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "GH", Prefix: "GH", GitHubRepo: "owner/repo",
	})

	body, _ := json.Marshal(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: true})
	req := httptest.NewRequest("POST", "/api/tickets", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	jobs, err := store.GetPendingSyncJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Error("Expected NO sync job for draft ticket in GitHub-linked project")
	}
}

func TestServer_SyncJob_UpdateNonDraftInGithubProject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "GH", Prefix: "GH", GitHubRepo: "owner/repo",
	})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	// Drain any existing sync jobs
	_, _ = store.GetPendingSyncJobs()

	newTitle := "Updated"
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{Title: &newTitle})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	jobs, err := store.GetPendingSyncJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) == 0 {
		t.Error("Expected sync job to be queued for update of non-draft ticket in GitHub-linked project")
	}
}

func TestServer_SyncJob_MoveDraftInGithubProject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "GH", Prefix: "GH", GitHubRepo: "owner/repo",
	})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: true})

	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "todo"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	jobs, err := store.GetPendingSyncJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) == 0 {
		t.Error("Expected sync job to be queued when moving draft ticket in GitHub-linked project")
	}
}

// ============================================================
// Gap 3: Ticket Filter Combinations
// ============================================================

func TestServer_TicketFilters(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	_, _ = store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", Status: "todo", Priority: "high"})
	_, _ = store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T2", Status: "done", Priority: "low"})

	tests := []struct {
		name string
		url  string
	}{
		{"projectId only", "/api/tickets?projectId=" + p.ID},
		{"status only", "/api/tickets?status=todo"},
		{"priority only", "/api/tickets?priority=high"},
		{"projectId+status", "/api/tickets?projectId=" + p.ID + "&status=todo"},
		{"projectId+status+priority", "/api/tickets?projectId=" + p.ID + "&status=todo&priority=high"},
		{"empty results", "/api/tickets?status=nonexistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Expected 200 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// ============================================================
// Gap 5: Validation Error Paths
// ============================================================

func TestServer_ValidationErrors(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	tests := []struct {
		name   string
		method string
		url    string
		body   string
		code   int
	}{
		// Create project missing name
		{"project missing name", "POST", "/api/projects", `{"prefix":"X"}`, http.StatusBadRequest},
		// Create project missing prefix
		{"project missing prefix", "POST", "/api/projects", `{"name":"X"}`, http.StatusBadRequest},
		// Create label missing name
		{"label missing name", "POST", "/api/labels", `{"color":"red"}`, http.StatusBadRequest},
		// Create label missing color
		{"label missing color", "POST", "/api/labels", `{"name":"L"}`, http.StatusBadRequest},
		// Move ticket missing status
		{"move missing status", "POST", "/api/tickets/" + ticket.ID + "/move", `{}`, http.StatusBadRequest},
		// Bad JSON for various entity types
		{"create ticket bad json", "POST", "/api/tickets", `{bad`, http.StatusBadRequest},
		{"update ticket bad json", "PUT", "/api/tickets/" + ticket.ID, `{bad`, http.StatusBadRequest},
		{"move ticket bad json", "POST", "/api/tickets/" + ticket.ID + "/move", `{bad`, http.StatusBadRequest},
		{"add subtask bad json", "POST", "/api/tickets/" + ticket.ID + "/subtasks", `{bad`, http.StatusBadRequest},
		{"create label bad json", "POST", "/api/labels", `{bad`, http.StatusBadRequest},
		{"update label bad json", "PUT", "/api/labels/someid", `{bad`, http.StatusBadRequest},
		{"update project bad json", "PUT", "/api/projects/" + p.ID, `{bad`, http.StatusBadRequest},
		{"create team bad json", "POST", "/api/teams", `{bad`, http.StatusBadRequest},
		{"update team bad json", "PUT", "/api/teams/someid", `{bad`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			if w.Code != tt.code {
				t.Errorf("%s: expected %d, got %d", tt.name, tt.code, w.Code)
			}
		})
	}
}

// ============================================================
// Gap 6: WebSocket - Binary Message
// ============================================================

func TestServer_TerminalWS_BinaryMessage(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()
	ts := httptest.NewServer(s)
	defer ts.Close()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/terminal/ws"
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return // PTY may not be available in CI
	}

	// Send a binary message (written directly to PTY)
	_ = ws.WriteMessage(websocket.BinaryMessage, []byte("echo hello\n"))

	_ = ws.Close()
}

// ============================================================
// Gap 7: Board without projectId
// ============================================================

func TestServer_BoardWithoutProjectId(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})
	_, _ = store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", Status: "todo"})

	req := httptest.NewRequest("GET", "/api/board", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for board without projectId, got %d", w.Code)
	}
}

// ============================================================
// Gap 8: ListProjects with status filter
// ============================================================

func TestServer_ListProjectsWithStatusFilter(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	_, _ = store.CreateProject(models.CreateProjectRequest{Name: "P1", Prefix: "P1"})

	req := httptest.NewRequest("GET", "/api/projects?status=active", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for projects with status filter, got %d", w.Code)
	}
}

// ============================================================
// Gap 9: Web FS (SPA serving)
// ============================================================

func TestServer_WebFS_SPA(t *testing.T) {
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
	webFS := newMockFS()
	srv := New(store, webFS)

	// Request for known static file (style.css)
	req := httptest.NewRequest("GET", "/style.css", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for /style.css, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "body{}") {
		t.Errorf("Expected style.css content, got %q", w.Body.String())
	}

	// Unknown path should fall back to index.html (SPA routing)
	// The server rewrites URL to "/" which serves index.html via the file server
	req = httptest.NewRequest("GET", "/some/unknown/path", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	// SPA fallback: path is rewritten to "/" which serves index.html
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for SPA fallback, got %d", w.Code)
	}
}

// ============================================================
// Gap 4: Delete internal errors with closed DB
// ============================================================

func TestServer_DeleteInternalErrors(t *testing.T) {
	s, store, cleanup := setupTestServer(t)

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "DEL"})
	team, _ := store.CreateTeam(models.CreateTeamRequest{Name: "T"})
	tick, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	label, _ := store.CreateLabel(models.CreateLabelRequest{Name: "L", Color: "C"})
	st, _ := store.AddSubtask(tick.ID, models.CreateSubtaskRequest{Title: "S"})

	_ = store.Close()

	tests := []struct {
		method string
		url    string
	}{
		{"DELETE", "/api/projects/" + p.ID},
		{"DELETE", "/api/teams/" + team.ID},
		{"DELETE", "/api/tickets/" + tick.ID},
		{"DELETE", "/api/labels/" + label.ID},
		{"DELETE", "/api/subtasks/" + st.ID},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.url, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("%s %s expected 500, got %d", tt.method, tt.url, w.Code)
		}
	}
	cleanup()
}

// ============================================================
// Mock FS for SPA tests
// ============================================================

type mockFS struct {
	files map[string]string
}

func newMockFS() *mockFS {
	return &mockFS{
		files: map[string]string{
			"index.html": "<html><body>Hello</body></html>",
			"style.css":  "body{}",
		},
	}
}

func (m *mockFS) Open(name string) (fs.File, error) {
	// Handle root directory request
	if name == "." || name == "" {
		entries := make([]fs.DirEntry, 0, len(m.files))
		for fname := range m.files {
			entries = append(entries, &mockDirEntry{name: fname, size: int64(len(m.files[fname]))})
		}
		return &mockDir{name: ".", entries: entries}, nil
	}
	content, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &mockFile{name: name, content: []byte(content)}, nil
}

type mockFile struct {
	name    string
	content []byte
	offset  int
}

func (f *mockFile) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{name: f.name, size: int64(len(f.content)), isDir: false}, nil
}

func (f *mockFile) Read(b []byte) (int, error) {
	if f.offset >= len(f.content) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(b, f.content[f.offset:])
	f.offset += n
	return n, nil
}

func (f *mockFile) Close() error { return nil }

type mockDir struct {
	name    string
	entries []fs.DirEntry
	offset  int
}

func (d *mockDir) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{name: d.name, size: 0, isDir: true}, nil
}

func (d *mockDir) Read([]byte) (int, error) {
	return 0, fmt.Errorf("is a directory")
}

func (d *mockDir) Close() error { return nil }

func (d *mockDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.offset >= len(d.entries) {
		if n <= 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("EOF")
	}
	if n <= 0 || n > len(d.entries)-d.offset {
		n = len(d.entries) - d.offset
	}
	entries := d.entries[d.offset : d.offset+n]
	d.offset += n
	return entries, nil
}

type mockDirEntry struct {
	name string
	size int64
}

func (e *mockDirEntry) Name() string               { return e.name }
func (e *mockDirEntry) IsDir() bool                 { return false }
func (e *mockDirEntry) Type() fs.FileMode           { return 0 }
func (e *mockDirEntry) Info() (fs.FileInfo, error)   { return &mockFileInfo{name: e.name, size: e.size, isDir: false}, nil }

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fi *mockFileInfo) Name() string        { return fi.name }
func (fi *mockFileInfo) Size() int64         { return fi.size }
func (fi *mockFileInfo) Mode() fs.FileMode   { if fi.isDir { return fs.ModeDir | 0555 }; return 0444 }
func (fi *mockFileInfo) ModTime() time.Time  { return time.Time{} }
func (fi *mockFileInfo) IsDir() bool         { return fi.isDir }
func (fi *mockFileInfo) Sys() interface{}    { return nil }

// ============================================================
// Additional coverage: nil-to-empty-slice branches for list endpoints
// ============================================================

func TestServer_EmptyLists(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	// List projects on empty DB (covers nil -> []models.Project{} branch)
	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for empty projects list, got %d", w.Code)
	}

	// List teams on empty DB (covers nil -> []models.Team{} branch)
	req = httptest.NewRequest("GET", "/api/teams", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for empty teams list, got %d", w.Code)
	}

	// List labels on empty DB (covers nil -> []models.Label{} branch)
	req = httptest.NewRequest("GET", "/api/labels", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for empty labels list, got %d", w.Code)
	}

	// List tickets on empty DB (covers nil -> []models.Ticket{} branch)
	req = httptest.NewRequest("GET", "/api/tickets", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for empty tickets list, got %d", w.Code)
	}
}

// ============================================================
// Additional coverage: move non-draft ticket in GitHub-linked project
// (covers the else branch at line 452 where existing.IsDraft is false)
// ============================================================

func TestServer_SyncJob_MoveNonDraftInGithubProject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "GH", Prefix: "GH2", GitHubRepo: "owner/repo",
	})
	// Create a non-draft ticket
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: false})

	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "done"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// ============================================================
// Additional coverage: strict mode on move with US/AC present (passes validation)
// ============================================================

func TestServer_StrictMode_MoveDraftWithUSAC(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "SM2", Strict: true})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{
		ProjectID:          p.ID,
		Title:              "T1",
		IsDraft:            true,
		UserStory:          "As a user...",
		AcceptanceCriteria: "Given...",
	})

	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "todo"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 when moving draft with US/AC, got %d", w.Code)
	}
}

// ============================================================
// Additional coverage: update ticket in non-strict project (skips strict validation)
// ============================================================

func TestServer_UpdateTicket_NonStrictProject(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Relaxed", Prefix: "REL", Strict: false})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1"})

	isDraft := false
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{IsDraft: &isDraft})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for non-strict update, got %d", w.Code)
	}
}

// ============================================================
// Additional coverage: move non-draft in strict project (skips isDraft branch)
// ============================================================

func TestServer_StrictMode_MoveNonDraft(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Strict", Prefix: "SM3", Strict: true})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{
		ProjectID:          p.ID,
		Title:              "T1",
		IsDraft:            false,
		UserStory:          "As a user...",
		AcceptanceCriteria: "Given...",
	})

	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "done"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 when moving non-draft in strict project, got %d", w.Code)
	}
}

// ============================================================
// Additional coverage: delete error paths (Get succeeds, Delete fails)
// We achieve this by dropping tables after data is created,
// so the in-memory cache in GetX returns data but DeleteX fails
// on the actual DB operation.
// ============================================================

func setupTestServerWithDB(t *testing.T) (*Server, *db.Store, *sql.DB, func()) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

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
	return s, store, database, func() { _ = database.Close() }
}

func TestServer_DeleteProject_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "DP"})
	// Drop the projects table so Delete fails but rename won't affect Get
	// Actually, we need Get to succeed and Delete to fail.
	// We'll rename the table after Get by using a trigger or simply
	// by making the delete fail due to a constraint. Let's drop and recreate
	// with a trigger that fails on delete.
	_, _ = database.Exec("CREATE TRIGGER prevent_project_delete BEFORE DELETE ON projects BEGIN SELECT RAISE(ABORT, 'delete blocked'); END;")

	req := httptest.NewRequest("DELETE", "/api/projects/"+p.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when DeleteProject fails, got %d", w.Code)
	}
}

func TestServer_DeleteTeam_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	team, _ := store.CreateTeam(models.CreateTeamRequest{Name: "T"})
	_, _ = database.Exec("CREATE TRIGGER prevent_team_delete BEFORE DELETE ON teams BEGIN SELECT RAISE(ABORT, 'delete blocked'); END;")

	req := httptest.NewRequest("DELETE", "/api/teams/"+team.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when DeleteTeam fails, got %d", w.Code)
	}
}

func TestServer_DeleteTicket_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "DT"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	// Soft delete uses UPDATE, not DELETE, so we need to block updates
	_, _ = database.Exec("CREATE TRIGGER prevent_ticket_update BEFORE UPDATE ON tickets BEGIN SELECT RAISE(ABORT, 'update blocked'); END;")

	req := httptest.NewRequest("DELETE", "/api/tickets/"+ticket.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when DeleteTicket fails, got %d", w.Code)
	}
}

func TestServer_DeleteLabel_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	label, _ := store.CreateLabel(models.CreateLabelRequest{Name: "L", Color: "C"})
	_, _ = database.Exec("CREATE TRIGGER prevent_label_delete BEFORE DELETE ON labels BEGIN SELECT RAISE(ABORT, 'delete blocked'); END;")

	req := httptest.NewRequest("DELETE", "/api/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when DeleteLabel fails, got %d", w.Code)
	}
}

func TestServer_DeleteSubtask_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "DS"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	st, _ := store.AddSubtask(ticket.ID, models.CreateSubtaskRequest{Title: "S"})
	_, _ = database.Exec("CREATE TRIGGER prevent_subtask_delete BEFORE DELETE ON subtasks BEGIN SELECT RAISE(ABORT, 'delete blocked'); END;")

	req := httptest.NewRequest("DELETE", "/api/subtasks/"+st.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when DeleteSubtask fails, got %d", w.Code)
	}
}

// ============================================================
// Additional coverage: updateTicket second nil check
// and moveTicket MoveTicket error path
// ============================================================

func TestServer_MoveTicket_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "MT"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	// Block updates so MoveTicket fails
	_, _ = database.Exec("CREATE TRIGGER prevent_ticket_update BEFORE UPDATE ON tickets BEGIN SELECT RAISE(ABORT, 'update blocked'); END;")

	moveBody, _ := json.Marshal(models.MoveTicketRequest{Status: "done"})
	req := httptest.NewRequest("POST", "/api/tickets/"+ticket.ID+"/move", bytes.NewBuffer(moveBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when MoveTicket fails, got %d", w.Code)
	}
}

func TestServer_UpdateDraftTicket_NoSyncJob(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{
		Name: "GH", Prefix: "GH3", GitHubRepo: "owner/repo",
	})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T1", IsDraft: true})

	newTitle := "Updated Draft"
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{Title: &newTitle})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Verify no sync job was queued for a draft ticket update
	jobs, err := store.GetPendingSyncJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Error("Expected no sync job for draft ticket update in GitHub project")
	}
}

func TestServer_UpdateTicket_InternalError(t *testing.T) {
	s, store, database, cleanup := setupTestServerWithDB(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "UT"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	// Block updates so UpdateTicket fails
	_, _ = database.Exec("CREATE TRIGGER prevent_ticket_update BEFORE UPDATE ON tickets BEGIN SELECT RAISE(ABORT, 'update blocked'); END;")

	newTitle := "Updated"
	updateBody, _ := json.Marshal(models.UpdateTicketRequest{Title: &newTitle})
	req := httptest.NewRequest("PUT", "/api/tickets/"+ticket.ID, bytes.NewBuffer(updateBody))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when UpdateTicket fails, got %d", w.Code)
	}
}

// ============================================================
// Phase 1: ListenAndServe coverage
// ============================================================

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

// ============================================================
// Phase 1: updateTicket strict-mode reject
// ============================================================

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

// ============================================================
// Phase 1: handleTerminalWS coverage (PTY failure on Windows)
// ============================================================

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
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
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

// ============================================================
// Phase 2: Pagination tests
// ============================================================

func TestServer_ListTickets_Pagination(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "PG"})
	for i := 0; i < 5; i++ {
		_, _ = store.CreateTicket(models.CreateTicketRequest{
			ProjectID: p.ID, Title: fmt.Sprintf("T%d", i),
		})
	}

	// Without pagination - returns flat array
	req := httptest.NewRequest("GET", "/api/tickets?projectId="+p.ID, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// With pagination - returns PaginatedResult
	req = httptest.NewRequest("GET", "/api/tickets?projectId="+p.ID+"&limit=2&offset=0", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var result models.PaginatedResult[models.Ticket]
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(result.Data))
	}
	if !result.HasMore {
		t.Error("expected hasMore=true")
	}
	if result.Limit != 2 || result.Offset != 0 {
		t.Errorf("unexpected limit/offset: %d/%d", result.Limit, result.Offset)
	}
}

func TestServer_ListProjects_Pagination(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()

	for i := 0; i < 4; i++ {
		_, _ = store.CreateProject(models.CreateProjectRequest{
			Name: fmt.Sprintf("P%d", i), Prefix: fmt.Sprintf("PP%d", i),
		})
	}

	req := httptest.NewRequest("GET", "/api/projects?limit=2&offset=0", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var result models.PaginatedResult[models.Project]
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 4 {
		t.Errorf("expected total=4, got %d", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 projects, got %d", len(result.Data))
	}
}

// ============================================================
