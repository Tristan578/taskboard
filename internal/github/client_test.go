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

func TestClient_GetIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"number": 1,
								"title": "T1",
								"body": "B1",
								"state": "OPEN",
								"labels": map[string]interface{}{
									"nodes": []interface{}{
										map[string]interface{}{"name": "bug"},
									},
								},
								"updatedAt": "2026-01-01T00:00:00Z",
							},
						},
						"pageInfo": map[string]interface{}{
							"endCursor": "abc",
							"hasNextPage": false,
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithURLs(context.Background(), "fake", "", server.URL)
	issues, err := client.GetIssues(context.Background(), "owner", "repo")
	if err != nil || len(issues) != 1 {
		t.Errorf("GetIssues failed: %v", err)
	}
	if issues[0].Title != "T1" {
		t.Errorf("Unexpected issue title")
	}
}

func TestClient_GetIssues_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClientWithURLs(context.Background(), "fake", "", server.URL)
	_, err := client.GetIssues(context.Background(), "o", "r")
	if err == nil {
		t.Errorf("Expected error for unauthorized graphql")
	}
}

func TestClient_Getters(t *testing.T) {
	c := NewClient(context.Background(), "token")
	if c.REST() == nil { t.Errorf("REST() returned nil") }
	if c.GQL() == nil { t.Errorf("GQL() returned nil") }
}

func TestClient_GetIssues_Pagination(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		hasNext := false
		if count == 0 { hasNext = true }
		count++
		
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{"number": count, "title": "T", "body": "B", "state": "OPEN", "labels": map[string]interface{}{"nodes": []interface{}{}}, "updatedAt": "2026-01-01T00:00:00Z"},
						},
						"pageInfo": map[string]interface{}{
							"endCursor": "abc",
							"hasNextPage": hasNext,
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithURLs(context.Background(), "fake", "", server.URL)
	issues, _ := client.GetIssues(context.Background(), "owner", "repo")
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues from pagination, got %d", len(issues))
	}
}

func TestWorker_Start(t *testing.T) {
	s := &mockStore{}
	w := NewWorker(s, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	w.Start(ctx)
}
