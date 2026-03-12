package github

import (
	"testing"

	"github.com/Tristan578/taskboard/internal/models"
)

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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (find(s, substr) >= 0)
}

func find(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
