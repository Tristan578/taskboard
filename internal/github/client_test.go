package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetIssues(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"number":    1,
								"title":     "Issue 1",
								"body":      "Body 1",
								"state":     "OPEN",
								"labels":    map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"name": "bug"}}},
								"updatedAt": "2024-03-12T10:00:00Z",
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   nil,
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClientWithURLs(context.Background(), "token", ts.URL, ts.URL)
	issues, err := client.GetIssues(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetIssues failed: %v", err)
	}

	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
}

func TestClient_CreateIssue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 123})
	}))
	defer ts.Close()

	client := NewClientWithURLs(context.Background(), "token", ts.URL, ts.URL)
	num, err := client.CreateIssue(context.Background(), "owner", "repo", "Title", "Body")
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	if num != 123 {
		t.Errorf("Expected issue number 123, got %d", num)
	}
}

func TestClient_UpdateIssue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 123})
	}))
	defer ts.Close()

	client := NewClientWithURLs(context.Background(), "token", ts.URL, ts.URL)
	err := client.UpdateIssue(context.Background(), "owner", "repo", 123, "Title", "Body", "closed")
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}
}

func TestClient_REST_GQL_Accessors(t *testing.T) {
	client := NewClient(context.Background(), "token")
	if client.REST() == nil {
		t.Error("REST() returned nil")
	}
	if client.GQL() == nil {
		t.Error("GQL() returned nil")
	}
}
