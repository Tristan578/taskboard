package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"golang.org/x/oauth2"
)

// Defense in depth: this opt-in live test also uses an Actions token with only
// read permissions. Mutations cannot escape even if reconciliation regresses.
type readOnlyGitHub struct{ calls int }

func (g *readOnlyGitHub) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "api.github.com" {
		return nil, fmt.Errorf("unexpected live-test host")
	}
	if r.Method != "GET" {
		if r.Method != "POST" || r.URL.Path != "/graphql" {
			return nil, fmt.Errorf("live test refused a GitHub write")
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(data))
		var body struct {
			Query string `json:"query"`
		}
		if json.Unmarshal(data, &body) != nil || !strings.HasPrefix(strings.TrimSpace(body.Query), "query") {
			return nil, fmt.Errorf("live test refused a GraphQL mutation")
		}
	}
	g.calls++
	return http.DefaultTransport.RoundTrip(r)
}
func TestLiveGitHubReadOnlyImport(t *testing.T) {
	repo := os.Getenv("PLAYER2_LIVE_GITHUB_REPO")
	if repo == "" {
		t.Skip("opt-in live check runs in CI with a read-only token")
	}
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Fatal("live check needs the CI read-only token")
	}
	if _, _, err := ParseRepo(repo); err != nil {
		t.Fatal(err)
	}
	guard := &readOnlyGitHub{}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Transport: guard})
	client := NewClient(ctx, os.Getenv("GITHUB_TOKEN"))
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "live-import.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := db.NewStore(database)
	project, err := store.CreateProject(models.CreateProjectRequest{Name: "Read-only live validation", Prefix: "LIVE", GitHubRepo: repo})
	if err != nil {
		t.Fatal(err)
	}
	if err = SyncProject(ctx, client, store, project.ID); err != nil {
		t.Fatal(err)
	}
	tickets, _, err := store.ListTickets(models.TicketFilter{ProjectID: project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if guard.calls == 0 {
		t.Fatal("live API was never called")
	}
	for _, ticket := range tickets {
		if ticket.GitHubIssueNumber == nil || ticket.GitHubLastSyncedSHA == "" || ticket.Status == "done" {
			t.Fatal("import lost identity, baseline, or completion policy")
		}
	}
	t.Logf("Live read-only GitHub import passed: %d API requests, %d issues imported; no writes permitted.", guard.calls, len(tickets))
}
