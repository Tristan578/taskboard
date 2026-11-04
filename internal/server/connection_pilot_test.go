package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tristan578/taskboard/internal/db"
)

func TestConnectionPilotHTTPAndOriginBoundary(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()
	call := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, localRequest(method, path, strings.NewReader(body)))
		return w
	}
	if w := call("GET", "/api/connections/setup", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "codex mcp add") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	id := "connect-00000000-0000-4000-8000-000000000001"
	body := `{"requestId":"` + id + `","goal":"Plan a week"}`
	foreign := localRequest("POST", "/api/connections/pilots", strings.NewReader(body))
	foreign.Header.Set("Origin", "https://foreign.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, foreign)
	if w.Code != 403 {
		t.Fatal("cross-origin pilot created")
	}
	w = call("POST", "/api/connections/pilots", body)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var pilot db.ConnectionPilot
	if err := json.Unmarshal(w.Body.Bytes(), &pilot); err != nil {
		t.Fatal(err)
	}
	if pilot.Reported || pilot.State != "waiting" {
		t.Fatal("setup claimed a connection")
	}
	if w = call("GET", "/api/connections/pilots/"+id, ""); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = call("POST", "/api/connections/pilots", `{}`); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/api/connections/pilots/connect-00000000-0000-4000-8000-000000000002", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
	_, count, _ := store.ListProjects("")
	if count != 1 {
		t.Fatal("unexpected boards", count)
	}
}

func TestSetupCommandsQuotePathsAndSeparateDatabases(t *testing.T) {
	value := "A path/with 'quotes' and $symbols`"
	if got := shellQuote(value, "windows"); got != "'A path/with ''quotes'' and $symbols`'" {
		t.Fatal(got)
	}
	if got := shellQuote(value, "linux"); got != "'A path/with '\"'\"'quotes'\"'\"' and $symbols`'" {
		t.Fatal(got)
	}
	for _, platform := range []string{"windows", "darwin", "linux"} {
		plan := setupCommands(value, "/a board.db", platform)
		other := setupCommands(value, "/another board.db", platform)
		if plan["name"] == other["name"] || !strings.Contains(plan["codex"].(string), shellQuote(value, platform)) || !strings.Contains(plan["claude"].(string), "--scope user") {
			t.Fatal(plan)
		}
	}
}
