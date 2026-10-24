package server

import (
	"github.com/Tristan578/taskboard/internal/models"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrowserRevisionPreconditions(t *testing.T) {
	s, store, cleanup := setupTestServer(t)
	defer cleanup()
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Home", Prefix: "H", CompletionPolicy: "manual"})
	task, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Old"})
	title := "New"
	_, err := store.UpdateTicket(task.ID, models.UpdateTicketRequest{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, path, body string }{{"PUT", "/api/tickets/" + task.ID, `{"title":"Overwrite","expectedRevision":0}`}, {"POST", "/api/tickets/" + task.ID + "/move", `{"status":"done","expectedRevision":0}`}, {"DELETE", "/api/tickets/" + task.ID + "?revision=0", ""}} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, localRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if w.Code != 409 {
			t.Fatalf("%s: %d %s", tc.method, w.Code, w.Body.String())
		}
	}
	saved, _ := store.GetTicket(task.ID)
	if saved.Title != "New" || saved.Status != "todo" {
		t.Fatal("stale browser replaced newer work")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, localRequest("DELETE", "/api/tickets/"+task.ID+"?revision=invalid", nil))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
