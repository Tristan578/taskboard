package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tristan578/taskboard/internal/models"
)

func find(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

type mockStore struct {
	project *models.Project
	tickets []models.Ticket
	deleted []models.Ticket
}

func (m *mockStore) GetProject(id string) (*models.Project, error) { return m.project, nil }
func (m *mockStore) ListTickets(filter models.TicketFilter) ([]models.Ticket, error) { return m.tickets, nil }
func (m *mockStore) GetTicket(id string) (*models.Ticket, error) {
	if len(m.tickets) > 0 { return &m.tickets[0], nil }
	return nil, nil
}
func (m *mockStore) UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error) { return nil, nil }
func (m *mockStore) CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error) { return &models.Ticket{ID: "new"}, nil }
func (m *mockStore) UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error) { return nil, nil }
func (m *mockStore) ListDeletedTickets(id string) ([]models.Ticket, error) { return m.deleted, nil }
func (m *mockStore) PurgeDeletedTickets(id string) error { return nil }
func (m *mockStore) GetPendingSyncJobs() ([]models.SyncJob, error) {
	return []models.SyncJob{{ID: "j1", ProjectID: "p1", Action: "full_sync"}}, nil
}
func (m *mockStore) UpdateSyncJobStatus(id, status string, attempts int, lastError string) error {
	return nil
}

func TestFormatIssueBody(t *testing.T) {
	ticket := &models.Ticket{
		UserStory:          "As a user...",
		AcceptanceCriteria: "Given... When... Then...",
		TechnicalDetails:   "Change X and Y",
		TestingDetails:     "Write test Z",
		LexoRank:           "0000001000",
	}
	desc := "This is the description."
	body := FormatIssueBody(desc, ticket)

	if find(body, "<!-- player2-metadata:") == -1 {
		t.Errorf("Body missing hidden metadata header")
	}
}

func TestParseIssueBody(t *testing.T) {
	body := "My Description\n\n<!-- player2-metadata:eyJ1cyI6IkFzIGEgdXNlci4uLiIsImFjIjoiR2l2ZW4uLi4ifQ== -->"
	desc, meta := ParseIssueBody(body)
	if desc != "My Description" {
		t.Errorf("Expected description 'My Description', got '%s'", desc)
	}
	if meta.UserStory != "As a user..." {
		t.Errorf("Expected user story 'As a user...', got '%s'", meta.UserStory)
	}
}

func TestWorker_ProcessJob(t *testing.T) {
	store := &mockStore{
		project: &models.Project{ID: "p1", GitHubRepo: "owner/repo"},
	}
	worker := NewWorker(store, nil)
	worker.processJob(context.Background(), models.SyncJob{ID: "j1", ProjectID: "p1", Action: "full_sync"})
}

func TestWorker_ProcessJobs(t *testing.T) {
	store := &mockStore{
		project: &models.Project{ID: "p1", GitHubRepo: "owner/repo"},
	}
	worker := NewWorker(store, nil)
	worker.processJobs(context.Background())
}

func TestSyncProject_Errors(t *testing.T) {
	store := &mockStore{}
	err := SyncProject(context.Background(), nil, store, "p1")
	if err == nil { t.Errorf("Expected error for nil client") }

	store.project = nil
	err = SyncProject(context.Background(), &Client{}, store, "none")
	if err == nil { t.Errorf("Expected error for missing project") }

	store.project = &models.Project{ID: "p1", GitHubRepo: ""}
	err = SyncProject(context.Background(), &Client{}, store, "p1")
	if err != nil { t.Errorf("SyncProject should return nil if not linked") }
}

func TestSyncProject_Linked(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"repository": nil}})
	}))
	defer ts.Close()

	store := &mockStore{
		project: &models.Project{ID: "p1", GitHubRepo: "owner/repo"},
		tickets: []models.Ticket{{ID: "t1", Title: "T1"}},
		deleted: []models.Ticket{{ID: "d1", GitHubIssueNumber: intPtr(123)}},
	}
	
	client := NewClientWithURLs(context.Background(), "fake", ts.URL, ts.URL)
	_ = SyncProject(context.Background(), client, store, "p1")
}

func TestMappings(t *testing.T) {
	if mapGHStateToStatus("CLOSED", nil) != "done" { t.Errorf("done fail") }
	if mapGHStateToStatus("OPEN", []string{"in-progress"}) != "in_progress" { t.Errorf("in_progress fail") }
	if mapGHStateToStatus("OPEN", nil) != "todo" { t.Errorf("todo fail") }

	if mapStatusToGHState("done") != "closed" { t.Errorf("closed fail") }
	if mapStatusToGHState("todo") != "open" { t.Errorf("open fail") }
}

func TestStripMetadata(t *testing.T) {
	body := "Desc\n\n<!-- player2-metadata:abc -->"
	if stripMetadata(body) != "Desc" {
		t.Errorf("stripMetadata failed")
	}
	
	body2 := "---\nplayer2:\n  us: 1\n---\nDesc2"
	if stripMetadata(body2) != "Desc2" {
		t.Errorf("stripMetadata legacy failed")
	}
}

func TestParseRepo_Extra(t *testing.T) {
	_, _, err := ParseRepo("invalid")
	if err == nil { t.Errorf("expected error") }
}

func intPtr(i int) *int { return &i }
