package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/server"
)

func TestDiscoveryRequiresHealthyMatchingDatabase(t *testing.T) {
	directory := t.TempDir()
	expected := filepath.Join(directory, "agent.db")
	other := filepath.Join(directory, "other.db")
	for _, filename := range []string{expected, other} {
		if err := os.WriteFile(filename, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, application, database, status string
		code                                int
		want                                bool
	}{
		{"matching", "player2-kanban", expected, "ok", 200, true},
		{"other database", "player2-kanban", other, "ok", 200, false},
		{"legacy identity missing", "player2-kanban", "", "ok", 200, false},
		{"other service", "another-app", expected, "ok", 200, false},
		{"degraded", "player2-kanban", expected, "degraded", 503, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_ = json.NewEncoder(w).Encode(map[string]any{"name": tc.application, "pid": os.Getpid(), "status": tc.status, "databasePath": tc.database})
			}))
			defer srv.Close()
			probe := probeBoard(context.Background(), srv.URL, expected, time.Second)
			if probe.Running != tc.want || probe.Instruction == "" {
				t.Fatalf("unexpected discovery: %+v", probe)
			}
		})
	}
}

func TestDiscoveryRejectsRedirectsAndOversizedResponses(t *testing.T) {
	requests := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++ }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	if probeBoard(context.Background(), redirect.URL, "", time.Second).Running || requests != 0 {
		t.Fatal("discovery followed a redirect")
	}
	large := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"padding":"` + strings.Repeat("a", 20<<10) + `","name":"player2-kanban","pid":1,"status":"ok"}`))
	}))
	defer large.Close()
	if probeBoard(context.Background(), large.URL, "", time.Second).Detected {
		t.Fatal("oversized response accepted")
	}
}

func TestStatusAndOpenIdentifyTheActualBoard(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "visible.db")
	database, err := db.OpenAt(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	srv := httptest.NewServer(server.New(db.NewStore(database), nil))
	defer srv.Close()
	address, _ := url.Parse(srv.URL)
	for _, tc := range []struct {
		path string
		want bool
	}{{filepath.Join(directory, "another.db"), false}, {filename, true}} {
		var output bytes.Buffer
		root := NewRootCmd(nil)
		root.SetOut(&output)
		root.SetArgs([]string{"--db", tc.path, "status", "--port", address.Port()})
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		var result struct {
			ServerRunning   bool `json:"serverRunning"`
			DatabaseMatches bool `json:"databaseMatches"`
		}
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.ServerRunning != tc.want || result.DatabaseMatches != tc.want {
			t.Fatalf("misleading status: %s", output.String())
		}
		if err := verifyRunningBoard(srv.URL, os.Getpid()); (err == nil) != tc.want {
			t.Fatalf("open reused incorrect board: want success %v, err %v", tc.want, err)
		}
	}
	if err := verifyRunningBoard(srv.URL, os.Getpid()+1); err == nil {
		t.Fatal("a different process was accepted")
	}
}
