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
	}
	desc := "This is the description."
	body := FormatIssueBody(desc, ticket)

	if !contains(body, "player2:") {
		t.Errorf("Body missing player2 header")
	}
	if !contains(body, "user_story: As a user...") {
		t.Errorf("Body missing user story")
	}
	if !contains(body, desc) {
		t.Errorf("Body missing description")
	}
}

func TestParseIssueBody(t *testing.T) {
	body := `---
player2:
  user_story: "As a user..."
  acceptance_criteria: "Given..."
---

My Description`

	desc, meta := ParseIssueBody(body)
	if desc != "My Description" {
		t.Errorf("Expected description 'My Description', got '%s'", desc)
	}
	if meta.UserStory != "As a user..." {
		t.Errorf("Expected user story 'As a user...', got '%s'", meta.UserStory)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || find(s, substr) >= 0)
}

func find(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
