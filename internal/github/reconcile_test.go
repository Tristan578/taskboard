package github

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReconciliationLifecycle(t *testing.T) {
	database, err := db.OpenAt(t.TempDir() + "/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := db.NewStore(database)
	project, err := store.CreateProject(models.CreateProjectRequest{Name: "Sync", Prefix: "SYNC", GitHubRepo: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := store.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Initial"})
	if err != nil {
		t.Fatal(err)
	}
	issues := map[int]Issue{}
	creates := 0
	writes := 0
	failWrites := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/graphql" {
			nodes := []map[string]any{}
			for _, i := range issues {
				nodes = append(nodes, map[string]any{"number": i.Number, "title": i.Title, "body": i.Body, "state": strings.ToUpper(i.State), "updatedAt": "2026-01-01T00:00:00Z", "labels": map[string]any{"nodes": []any{}}})
			}
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": map[string]any{"issues": map[string]any{"nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false}}}}})
			return
		}
		if failWrites {
			w.WriteHeader(503)
			return
		}
		var input struct{ Title, Body, State string }
		json.NewDecoder(r.Body).Decode(&input)
		n := 1
		if r.Method == "POST" {
			creates++
		} else {
			fmt.Sscanf(r.URL.Path, "/repos/o/r/issues/%d", &n)
		}
		if input.State == "" {
			input.State = "open"
		}
		issues[n] = Issue{Number: n, Title: input.Title, Body: input.Body, State: input.State}
		writes++
		json.NewEncoder(w).Encode(map[string]any{"number": n})
	}))
	defer srv.Close()
	client := NewClientWithURLs(context.Background(), "token", srv.URL, srv.URL+"/graphql")
	sync := func() error { return SyncProject(context.Background(), client, store, project.ID) }
	if err := sync(); err != nil {
		t.Fatal(err)
	}
	if creates != 1 {
		t.Fatalf("creates: %d", creates)
	}
	oldWrites := writes
	if err := sync(); err != nil {
		t.Fatal(err)
	}
	if writes != oldWrites {
		t.Fatal("unchanged sync wrote remotely")
	}
	ticket, _ = store.MoveTicket(ticket.ID, models.MoveTicketRequest{Status: "in_progress"})
	if err := sync(); err != nil {
		t.Fatal(err)
	}
	_, meta := ParseIssueBody(issues[1].Body)
	if meta.Status != "in_progress" {
		t.Fatal("workflow did not round-trip")
	}
	title := "Local change"
	ticket, err = store.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	remote := issues[1]
	remote.Title = "Remote change"
	issues[1] = remote
	oldWrites = writes
	if err := sync(); err == nil {
		t.Fatal("concurrent changes silently overwrote")
	}
	ticket, _ = store.GetTicket(ticket.ID)
	if ticket.Title != title || ticket.Work.Conflict == "" || writes != oldWrites {
		t.Fatal("conflict lost a version")
	}
	if err := ResolveConflict(context.Background(), client, store, ticket.ID, "remote"); err != nil {
		t.Fatal(err)
	}
	ticket, _ = store.GetTicket(ticket.ID)
	if ticket.Title != "Remote change" || ticket.Work.Conflict != "" {
		t.Fatal("resolution failed")
	}
	// Failed deletion remains durable and cannot be reimported.
	if err := store.DeleteTicket(ticket.ID); err != nil {
		t.Fatal(err)
	}
	failWrites = true
	if err := sync(); err == nil {
		t.Fatal("partial failure reported success")
	}
	deleted, _ := store.ListDeletedTickets(project.ID)
	if len(deleted) != 1 {
		t.Fatal("lost tombstone")
	}
	active, _, _ := store.ListTickets(models.TicketFilter{ProjectID: project.ID})
	if len(active) != 0 {
		t.Fatal("deleted task resurrected")
	}
	failWrites = false
	if err := sync(); err != nil {
		t.Fatal(err)
	}
	deleted, _ = store.ListDeletedTickets(project.ID)
	if len(deleted) != 1 || deleted[0].GitHubLastSyncedSHA != "deleted-acknowledged" {
		t.Fatal("deletion not acknowledged")
	}
	if err := sync(); err != nil {
		t.Fatal(err)
	}
	active, _, _ = store.ListTickets(models.TicketFilter{ProjectID: project.ID})
	if len(active) != 0 {
		t.Fatal("acknowledged deleted ticket resurrected on next sync")
	}
}

func TestMetadataPreservesSurroundingText(t *testing.T) {
	tkt := models.Ticket{ID: "stable", Status: "review", UserStory: "story"}
	body := FormatIssueBody("before", &tkt) + "\nHuman text after metadata"
	desc, m := ParseIssueBody(body)
	if !strings.Contains(desc, "Human text after metadata") || m.TicketID != "stable" {
		t.Fatal(desc, m)
	}
	rewritten := FormatIssueBody(body, &tkt)
	if strings.Count(rewritten, metadataPrefix) != 1 || !strings.Contains(rewritten, "Human text after metadata") {
		t.Fatal(rewritten)
	}
}

func TestCreateSuccessAtLowRateLimitIsAcknowledged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", "1")
		w.Write([]byte(`{"number":42}`))
	}))
	defer srv.Close()
	c := NewClientWithURLs(context.Background(), "token", srv.URL, srv.URL)
	n, err := c.CreateIssue(context.Background(), "o", "r", "task", "body")
	if err != nil || n != 42 {
		t.Fatalf("successful create must not be retried: %d %v", n, err)
	}
}

func TestRetryAdoptsRemoteIssueByStableIdentity(t *testing.T) {
	database, err := db.OpenAt(t.TempDir() + "/retry.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := db.NewStore(database)
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P", GitHubRepo: "o/r"})
	ticket, _ := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Task"})
	body := FormatIssueBody("", ticket)
	creates := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/graphql" {
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": map[string]any{"issues": map[string]any{"nodes": []any{map[string]any{"number": 9, "title": "Task", "body": body, "state": "OPEN", "updatedAt": "2026-01-01T00:00:00Z", "labels": map[string]any{"nodes": []any{}}}}, "pageInfo": map[string]any{"hasNextPage": false}}}}})
			return
		}
		if r.Method == "POST" {
			creates++
		}
		w.Write([]byte(`{"number":9}`))
	}))
	defer srv.Close()
	c := NewClientWithURLs(context.Background(), "token", srv.URL, srv.URL+"/graphql")
	if err := SyncProject(context.Background(), c, store, p.ID); err != nil {
		t.Fatal(err)
	}
	tickets, _, _ := store.ListTickets(models.TicketFilter{ProjectID: p.ID})
	if creates != 0 || len(tickets) != 1 || tickets[0].GitHubIssueNumber == nil || *tickets[0].GitHubIssueNumber != 9 {
		t.Fatalf("duplicate issue or ticket: creates=%d tickets=%v", creates, tickets)
	}
}
