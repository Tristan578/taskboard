package server

import (
	"github.com/Tristan578/taskboard/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalBoundary(t *testing.T) {
	handler := localBoundary(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, tc := range []struct {
		host, origin string
		want         int
	}{
		{"localhost:3010", "http://localhost:3010", 204}, {"127.0.0.1:3010", "", 204},
		{"evil.example:3010", "", 403}, {"localhost:3010", "https://evil.example", 403},
		{"localhost:3010", "null", 403}, {"localhost:3010", "http://localhost:3011", 403},
		{"[::1]:3010", "http://[::1]:3010", 204},
	} {
		t.Run(tc.host+tc.origin, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/terminal/ws", nil)
			r.Host = tc.host
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("got %d", w.Code)
			}
		})
	}
}

func TestWorkflowEndpoints(t *testing.T) {
	server, store, cleanup := setupTestServer(t)
	defer cleanup()
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Task"})
	t.Setenv("GITHUB_TOKEN", "")
	for _, tc := range []struct {
		path, body string
		want       int
	}{
		{"/api/tickets/" + ticket.ID + "/activity", `{"agent":"Claude","sessionId":"run","summary":"Working"}`, 200},
		{"/api/tickets/" + ticket.ID + "/activity", `{}`, 409},
		{"/api/tickets/" + ticket.ID + "/complete", `{"evidence":"https://github.com/o/r/pull/1"}`, 409},
		{"/api/tickets/" + ticket.ID + "/resolve", `{"choice":"remote"}`, 409},
		{"/api/sync/retry", `{}`, 200},
	} {
		req := localRequest("POST", tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s: %d %s", tc.path, rec.Code, rec.Body.String())
		}
	}
}
