package github

import (
	"context"
	"testing"

	"github.com/Tristan578/taskboard/internal/models"
)

func contains(s, substr string) bool {
	return find(s, substr) >= 0
}

func find(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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

	if !contains(body, "<!-- player2-metadata:") {
		t.Errorf("Body missing hidden metadata header")
	}
	if !contains(body, desc) {
		t.Errorf("Body missing original description")
	}
}

func TestParseIssueBody(t *testing.T) {
	// New Format
	body := "My Description\n\n<!-- player2-metadata:eyJ1cyI6IkFzIGEgdXNlci4uLiIsImFjIjoiR2l2ZW4uLi4ifQ== -->"

	desc, meta := ParseIssueBody(body)
	if desc != "My Description" {
		t.Errorf("Expected description 'My Description', got '%s'", desc)
	}
	if meta.UserStory != "As a user..." {
		t.Errorf("Expected user story 'As a user...', got '%s'", meta.UserStory)
	}
}

func TestParseRepo(t *testing.T) {
	tests := []struct {
		url   string
		owner string
		repo  string
	}{
		{"https://github.com/owner/repo", "owner", "repo"},
		{"https://github.com/owner/repo.git", "owner", "repo"},
		{"git@github.com:owner/repo.git", "owner", "repo"},
		{"owner/repo", "owner", "repo"},
	}

	for _, tt := range tests {
		o, r, err := ParseRepo(tt.url)
		if err != nil || o != tt.owner || r != tt.repo {
			t.Errorf("ParseRepo(%s) = (%s, %s), want (%s, %s)", tt.url, o, r, tt.owner, tt.repo)
		}
	}
}

func TestMapGHStateToStatus(t *testing.T) {
	if mapGHStateToStatus("CLOSED", nil) != "done" {
		t.Errorf("CLOSED should be done")
	}
	if mapGHStateToStatus("OPEN", []string{"in-progress"}) != "in_progress" {
		t.Errorf("OPEN with in-progress label should be in_progress")
	}
	if mapGHStateToStatus("OPEN", nil) != "todo" {
		t.Errorf("OPEN should be todo")
	}
}

func TestMapStatusToGHState(t *testing.T) {
	if mapStatusToGHState("done") != "closed" {
		t.Errorf("done should be closed")
	}
	if mapStatusToGHState("todo") != "open" {
		t.Errorf("todo should be open")
	}
}

type mockStore struct {
	project *models.Project
	tickets []models.Ticket
}

func (m *mockStore) GetProject(id string) (*models.Project, error) { return m.project, nil }
func (m *mockStore) ListTickets(filter models.TicketFilter) ([]models.Ticket, error) { return m.tickets, nil }
func (m *mockStore) GetTicket(id string) (*models.Ticket, error) { return nil, nil }
func (m *mockStore) UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error) { return nil, nil }
func (m *mockStore) CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error) { return &models.Ticket{ID: "new"}, nil }
func (m *mockStore) UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error) { return nil, nil }
func (m *mockStore) ListDeletedTickets(id string) ([]models.Ticket, error) { return nil, nil }
func (m *mockStore) PurgeDeletedTickets(id string) error { return nil }

func TestWorker_ProcessJob(t *testing.T) {
	store := &mockStore{
		project: &models.Project{ID: "p1", GitHubRepo: "owner/repo"},
	}
	worker := NewWorker(store, nil) // SyncProject will fail but we want to hit UpdateSyncJobStatus code path

	ctx := context.Background()
	job := models.SyncJob{ID: "j1", ProjectID: "p1", Action: "full_sync"}
	
	worker.processJob(ctx, job)
	// mockStore.UpdateSyncJobStatus should have been called (implied by execution)
}

func (m *mockStore) GetPendingSyncJobs() ([]models.SyncJob, error) {
	return []models.SyncJob{{ID: "j1", ProjectID: "p1", Action: "full_sync"}}, nil
}

func (m *mockStore) UpdateSyncJobStatus(id, status string, attempts int, lastError string) error {
	return nil
}

func TestWorker_ProcessJobs(t *testing.T) {
	store := &mockStore{
		project: &models.Project{ID: "p1", GitHubRepo: "owner/repo"},
	}
	worker := NewWorker(store, nil)
	worker.processJobs(context.Background())
}
