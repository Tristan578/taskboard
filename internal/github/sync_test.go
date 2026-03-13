package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	for _, t := range m.tickets {
		if t.ID == id { return &t, nil }
	}
	return nil, nil
}
func (m *mockStore) UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error) { return nil, nil }
func (m *mockStore) CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error) { 
	if m.project != nil && m.project.ID == "create_err" { return nil, fmt.Errorf("err") }
	return &models.Ticket{ID: "new"}, nil 
}
func (m *mockStore) UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error) { return nil, nil }
func (m *mockStore) ListDeletedTickets(id string) ([]models.Ticket, error) { 
	if m.project != nil && m.project.ID == "p_err" { return nil, fmt.Errorf("err") }
	return m.deleted, nil 
}
func (m *mockStore) PurgeDeletedTickets(id string) error { return nil }
func (m *mockStore) GetPendingSyncJobs() ([]models.SyncJob, error) {
	if m.project != nil && m.project.ID == "job_err" { return nil, fmt.Errorf("err") }
	return []models.SyncJob{{ID: "j1", ProjectID: "p1", Action: "full_sync"}}, nil
}
func (m *mockStore) UpdateSyncJobStatus(id, status string, attempts int, lastError string) error { return nil }
func (m *mockStore) ListLabels() ([]models.Label, error) { return nil, nil }
func (m *mockStore) GetLabel(id string) (*models.Label, error) { return nil, nil }
func (m *mockStore) CreateLabel(req models.CreateLabelRequest) (*models.Label, error) { return nil, nil }
func (m *mockStore) UpdateLabel(id string, req models.UpdateLabelRequest) (*models.Label, error) { return nil, nil }
func (m *mockStore) DeleteLabel(id string) error { return nil }
func (m *mockStore) GetSubtask(id string) (*models.Subtask, error) { return nil, nil }
func (m *mockStore) AddSubtask(ticketID string, req models.CreateSubtaskRequest) (*models.Subtask, error) { return nil, nil }
func (m *mockStore) ToggleSubtask(id string) (*models.Subtask, error) { return nil, nil }
func (m *mockStore) DeleteSubtask(id string) error { return nil }
func (m *mockStore) GetBoard(projectID string) (*models.Board, error) { return nil, nil }
func (m *mockStore) ListTeams() ([]models.Team, error) { return nil, nil }
func (m *mockStore) GetTeam(id string) (*models.Team, error) { return nil, nil }
func (m *mockStore) CreateTeam(req models.CreateTeamRequest) (*models.Team, error) { return nil, nil }
func (m *mockStore) UpdateTeam(id string, req models.UpdateTeamRequest) (*models.Team, error) { return nil, nil }
func (m *mockStore) DeleteTeam(id string) error { return nil }
func (m *mockStore) ClearData() error { return nil }
func (m *mockStore) Close() error { return nil }

func TestSyncProject_Full(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"issues": map[string]interface{}{
							"nodes": []interface{}{
								map[string]interface{}{
									"number": 1, "title": "GH Newer", "body": "B", "state": "OPEN",
									"labels": map[string]interface{}{"nodes": []interface{}{}},
									"updatedAt": "2027-01-01T00:00:00Z", 
								},
								map[string]interface{}{
									"number": 2, "title": "Local Newer", "body": "B", "state": "OPEN",
									"labels": map[string]interface{}{"nodes": []interface{}{}},
									"updatedAt": "2020-01-01T00:00:00Z", 
								},
								map[string]interface{}{
									"number": 3, "title": "Only on GH", "body": "B", "state": "OPEN",
									"labels": map[string]interface{}{"nodes": []interface{}{}},
									"updatedAt": "2025-01-01T00:00:00Z", 
								},
							},
							"pageInfo": map[string]interface{}{"hasNextPage": false},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{"number": 4})
		}
	}))
	defer ts.Close()

	n1, n2 := 1, 2
	store := &mockStore{
		project: &models.Project{ID: "p1", GitHubRepo: "o/r"},
		tickets: []models.Ticket{
			{ID: "t1", GitHubIssueNumber: &n1, UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, 
			{ID: "t2", GitHubIssueNumber: &n2, UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{ID: "t4", GitHubIssueNumber: nil, UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		deleted: []models.Ticket{{ID: "d1", GitHubIssueNumber: intPtr(123)}},
	}
	
	client := NewClientWithURLs(context.Background(), "fake", ts.URL, ts.URL)
	_ = SyncProject(context.Background(), client, store, "p1")
}

func TestSyncProject_Errors(t *testing.T) {
	s := &mockStore{project: &models.Project{ID: "p1", GitHubRepo: "o/r"}}
	client := NewClient(context.Background(), "f")
	if SyncProject(context.Background(), nil, s, "p1") == nil { t.Errorf("1") }
	if SyncProject(context.Background(), client, s, "none") == nil { t.Errorf("2") }
	s.project.GitHubRepo = ""
	if SyncProject(context.Background(), client, s, "p1") != nil { t.Errorf("3") }
	s.project.GitHubRepo = "invalid"
	if SyncProject(context.Background(), client, s, "p1") == nil { t.Errorf("4") }
}

func TestSyncProject_REST_Failures(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			n1 := 1
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"issues": map[string]interface{}{
							"nodes": []interface{}{
								map[string]interface{}{
									"number": n1, "title": "T", "body": "B", "state": "OPEN",
									"labels": map[string]interface{}{"nodes": []interface{}{}},
									"updatedAt": "2020-01-01T00:00:00Z", 
								},
							},
							"pageInfo": map[string]interface{}{"hasNextPage": false},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer ts.Close()

	n1 := 1
	client := NewClientWithURLs(context.Background(), "f", ts.URL, ts.URL)

	t.Run("UpdateFail", func(t *testing.T) {
		s := &mockStore{
			project: &models.Project{ID: "p1", GitHubRepo: "o/r"},
			tickets: []models.Ticket{{ID: "t1", GitHubIssueNumber: &n1, UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		}
		_ = SyncProject(context.Background(), client, s, "p1")
	})

	t.Run("CreateFail", func(t *testing.T) {
		s := &mockStore{
			project: &models.Project{ID: "p1", GitHubRepo: "o/r"},
			tickets: []models.Ticket{{ID: "t2", GitHubIssueNumber: nil}},
		}
		_ = SyncProject(context.Background(), client, s, "p1")
	})

	t.Run("DeleteFail", func(t *testing.T) {
		s := &mockStore{
			project: &models.Project{ID: "p1", GitHubRepo: "o/r"},
			deleted: []models.Ticket{{ID: "d1", GitHubIssueNumber: intPtr(123)}},
		}
		_ = SyncProject(context.Background(), client, s, "p1")
	})
}

func TestSyncProject_MissingAndCreateFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		n1, n2 := 1, 2
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"number": n1, "title": "T1", "body": "B", "state": "OPEN",
								"labels": map[string]interface{}{"nodes": []interface{}{}},
								"updatedAt": "2020-01-01T00:00:00Z", 
							},
							map[string]interface{}{
								"number": n2, "title": "T2", "body": "B", "state": "OPEN",
								"labels": map[string]interface{}{"nodes": []interface{}{}},
								"updatedAt": "2020-01-01T00:00:00Z", 
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	n1 := 1
	store := &mockStore{
		project: &models.Project{ID: "create_err", GitHubRepo: "o/r"},
		tickets: []models.Ticket{
			{ID: "t1", GitHubIssueNumber: &n1},
		},
	}
	client := NewClientWithURLs(context.Background(), "f", ts.URL, ts.URL)
	_ = SyncProject(context.Background(), client, store, "create_err")
}

func TestWorker_Full(t *testing.T) {
	s := &mockStore{project: &models.Project{ID: "p1", GitHubRepo: "o/r"}}
	client := NewClient(context.Background(), "f")
	worker := NewWorker(s, client)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker.Start(ctx) 

	worker.processJobs(context.Background())
	_ = worker.executeAction(context.Background(), models.SyncJob{Action: "full_sync", ProjectID: "p1"})
	_ = worker.executeAction(context.Background(), models.SyncJob{Action: "push_ticket", ProjectID: "p1"})
	_ = worker.executeAction(context.Background(), models.SyncJob{Action: "unknown", ProjectID: "p1"})

	s.project = &models.Project{ID: "job_err"}
	worker.processJobs(context.Background())
}

func TestParseRepo_Errors(t *testing.T) {
	_, _, err := ParseRepo("not-a-repo")
	if err == nil { t.Errorf("1") }
}

func TestFormatIssueBody(t *testing.T) {
	ticket := &models.Ticket{UserStory: "US", LexoRank: "LR"}
	body := FormatIssueBody("desc", ticket)
	if find(body, "player2-metadata") == -1 { t.Errorf("meta missing") }
}

func TestParseIssueBody(t *testing.T) {
	body := "desc\n\n<!-- player2-metadata:eyJ1cyI6IlVTIiwiYWMiOiJBQyJ9 -->"
	d, m := ParseIssueBody(body)
	if d != "desc" || m.UserStory != "US" { t.Errorf("parse fail") }

	legacy := "---\nplayer2:\n  us: US\n---\ndesc"
	d, _ = ParseIssueBody(legacy)
	if d != "desc" { t.Errorf("legacy fail") }
}

func TestMappings(t *testing.T) {
	if mapGHStateToStatus("CLOSED", nil) != "done" { t.Errorf("1") }
	if mapGHStateToStatus("OPEN", []string{"in-progress"}) != "in_progress" { t.Errorf("2") }
	if mapGHStateToStatus("OPEN", nil) != "todo" { t.Errorf("3") }
	if mapStatusToGHState("done") != "closed" { t.Errorf("4") }
	if mapStatusToGHState("todo") != "open" { t.Errorf("5") }
}

func TestStripMetadata(t *testing.T) {
	if stripMetadata("h") != "h" { t.Errorf("1") }
	if stripMetadata("---\nplayer2:\n  us: 1\n---\nd") != "d" { t.Errorf("2") }
	if stripMetadata("d\n\n<!-- player2-metadata:a -->") != "d" { t.Errorf("3") }
}

func intPtr(i int) *int { return &i }
