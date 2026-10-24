package workflow

import (
	"context"
	"github.com/Tristan578/taskboard/internal/github"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyEvidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/o/r/pulls/1":
			w.Write([]byte(`{"merged":true,"merged_at":"2026-01-01T00:00:00Z"}`))
		case "/repos/o/r/pulls/2":
			w.Write([]byte(`{"merged":false}`))
		case "/repos/o/r/deployments/1/statuses":
			w.Write([]byte(`[{"state":"success"}]`))
		case "/repos/o/r/deployments/2/statuses":
			w.Write([]byte(`[{"state":"failure"},{"state":"success"}]`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	c := github.NewClientWithURLs(context.Background(), "test", srv.URL, srv.URL)
	for _, tc := range []struct {
		url string
		ok  bool
	}{
		{"https://github.com/o/r/pull/1", true}, {"https://github.com/o/r/pull/2", false},
		{"https://github.com/other/r/pull/1", false}, {"http://github.com/o/r/pull/1", false},
		{"https://github.com/o/r/pull/no", false}, {"https://evil.example/o/r/pull/1", false},
		{"https://api.github.com/repos/o/r/deployments/1", true}, {"https://api.github.com/repos/o/r/deployments/2", false},
		{"https://github.com/o/r/pull/99", false}, {"https://github.com/o/r/pull/1?q=x", false},
	} {
		t.Run(tc.url, func(t *testing.T) {
			err := VerifyEvidence(context.Background(), c, "o", "r", tc.url)
			if (err == nil) != tc.ok {
				t.Fatalf("result %v", err)
			}
		})
	}
}
