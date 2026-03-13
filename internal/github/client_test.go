package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestClient_REST_ErrorScenario(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithURLs(context.Background(), "fake-token", server.URL, "")
	_, err := client.CreateIssue(context.Background(), "o", "r", "T", "B")
	if err == nil {
		t.Errorf("Expected error for 500 response")
	}
}

func TestClient_REST_Methods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/issues" && r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"number": 123})
			return
		}
		if r.URL.Path == "/repos/owner/repo/issues/123" && r.Method == "PATCH" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"number": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClientWithURLs(context.Background(), "fake-token", server.URL, "")

	num, err := client.CreateIssue(context.Background(), "owner", "repo", "Title", "Body")
	if err != nil || num != 123 {
		t.Errorf("CreateIssue failed: %v", err)
	}

	err = client.UpdateIssue(context.Background(), "owner", "repo", 123, "New Title", "New Body", "open")
	if err != nil {
		t.Errorf("UpdateIssue failed: %v", err)
	}

	_, err = client.CreateIssue(context.Background(), "wrong", "repo", "T", "B")
	if err == nil {
		t.Errorf("Expected error for 404 response")
	}
}

func TestClient_NewClient(t *testing.T) {
	_ = NewClient(context.Background(), "token")
	
	oldToken := os.Getenv("GITHUB_TOKEN")
	os.Setenv("GITHUB_TOKEN", "fake")
	_ = NewClient(context.Background(), "")
	os.Setenv("GITHUB_TOKEN", oldToken)
}

func TestClient_Getters(t *testing.T) {
	c := NewClient(context.Background(), "token")
	if c.REST() == nil { t.Errorf("REST() returned nil") }
	if c.GQL() == nil { t.Errorf("GQL() returned nil") }
}

func TestWorker_Start(t *testing.T) {
	s := &mockStore{}
	w := NewWorker(s, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	w.Start(ctx)
}
