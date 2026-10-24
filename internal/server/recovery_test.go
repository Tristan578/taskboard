package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
)

func TestRecoveryHTTPReviewRestoreAndUndo(t *testing.T) {
	server, store, cleanup := setupTestServer(t)
	defer cleanup()
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Home", Prefix: "H", CompletionPolicy: "manual"})
	task, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Keep"})
	call := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		server.ServeHTTP(w, localRequest(method, path, strings.NewReader(body)))
		return w
	}
	for _, path := range []string{"/api/recovery", "/api/trash"} {
		if w := call("GET", path, ""); w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	w := call("POST", "/api/recovery/backups", "")
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	var backup struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &backup)
	w = call("GET", "/api/recovery/backups/"+backup.ID, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	data := w.Body.Bytes()
	previewResponse := httptest.NewRecorder()
	server.ServeHTTP(previewResponse, localRequest("POST", "/api/recovery/preview", bytes.NewReader(data)))
	if previewResponse.Code != 200 {
		t.Fatal(previewResponse.Body.String())
	}
	var preview db.RecoveryPreview
	_ = json.Unmarshal(previewResponse.Body.Bytes(), &preview)
	if w = call("POST", "/api/recovery/restore", `{"id":"`+preview.ID+`"}`); w.Code != 400 {
		t.Fatal("confirmation bypassed")
	}
	payload := `{"id":"` + preview.ID + `","confirm":true}`
	if w = call("POST", "/api/recovery/restore", payload); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = call("POST", "/api/recovery/restore", payload); w.Code != 200 {
		t.Fatal("retry failed")
	}
	if w = call("POST", "/api/recovery/preview", "bad file"); w.Code != 400 {
		t.Fatal("bad file accepted")
	}
	_ = store.DeleteTicket(task.ID, task.Revision)
	w = call("POST", "/api/trash/"+task.ID+"/restore", `{"revision":1}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	req := localRequest("GET", "/api/recovery", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("cross-site backup list exposed")
	}
}
