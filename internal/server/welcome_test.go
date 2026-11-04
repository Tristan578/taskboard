package server

import (
	"encoding/json"
	"github.com/Tristan578/taskboard/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWelcomeAndBackupHTTP(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()
	payload := `{"requestId":"http-request-123456","name":"My week","goal":"Keep it simple","completionPolicy":"manual","tasks":[{"title":"Plan meals","description":"Start small"}]}`
	var id string
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("POST", "http://localhost/api/welcome", strings.NewReader(payload)))
		if w.Code != 201 {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
		var p models.Project
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		if i == 1 && id != p.ID {
			t.Fatal("duplicate request created a second board")
		}
		id = p.ID
	}
	_, total, _ := store.ListProjects("")
	if total != 1 {
		t.Fatal("duplicate board")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/api/backup", nil))
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), "SQLite format 3") || !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("invalid backup: %d", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("POST", "http://localhost/api/welcome", strings.NewReader(`{"unknown":true}`)))
	if w.Code != 400 {
		t.Fatal("accepted malformed plan")
	}
}

func TestConnectionSettingsUseThisDatabase(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/api/connections", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var config struct {
		Servers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	p, err := store.DatabasePath()
	if err != nil {
		t.Fatal(err)
	}
	c := config.Servers["player2"]
	if c.Command == "" || len(c.Args) != 3 || c.Args[1] != p || c.Args[2] != "mcp" {
		t.Fatalf("invalid settings: %+v", c)
	}
}
