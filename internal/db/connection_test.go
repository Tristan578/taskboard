package db

import (
	"strings"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

const connectionTestID = "connect-00000000-0000-4000-8000-000000000001"

func TestConnectionPilotRequiresMatchingReportAndSavedResult(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	pilot, err := s.CreateConnectionPilot(connectionTestID, "Prepare for an interview")
	if err != nil {
		t.Fatal(err)
	}
	if pilot.State != "waiting" || pilot.Reported || !strings.Contains(pilot.Prompt, pilot.Task.ID) || !strings.Contains(pilot.Prompt, connectionTestID) {
		t.Fatal(pilot)
	}
	retry, err := s.CreateConnectionPilot(connectionTestID, "A changed draft after a lost response")
	if err != nil || retry.Task.ID != pilot.Task.ID {
		t.Fatal("retry duplicated task", err)
	}
	_, count, _ := s.ListProjects("")
	if count != 1 {
		t.Fatal("retry duplicated board")
	}
	_, err = s.ReportActivity(pilot.Task.ID, models.ActivityRequest{Agent: "Other", SessionID: "wrong-session", Summary: "Old report"})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := s.GetConnectionPilot(connectionTestID)
	if updated.Reported {
		t.Fatal("unrelated session accepted")
	}
	_, err = s.db.Exec("UPDATE work_state SET heartbeat_at=? WHERE ticket_id=?", time.Now().Add(-time.Hour), pilot.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ReportActivity(pilot.Task.ID, models.ActivityRequest{Agent: "Test agent", SessionID: connectionTestID, Summary: "Planning"})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ = s.GetConnectionPilot(connectionTestID)
	if !updated.Reported || updated.State != "working" {
		t.Fatal("matching report not observed", updated)
	}
	_, err = s.MoveTicket(pilot.Task.ID, models.MoveTicketRequest{Status: "review"})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ = s.GetConnectionPilot(connectionTestID)
	if updated.State != "working" {
		t.Fatal("Review without a result accepted")
	}
	result := "Prepare for an interview\n1. Review the role\n2. Prepare examples\n3. Practice"
	_, err = s.UpdateTicket(pilot.Task.ID, models.UpdateTicketRequest{Description: &result})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ = s.GetConnectionPilot(connectionTestID)
	if updated.State != "needs_review" {
		t.Fatal("result not observed", updated.State)
	}
	_, err = s.MoveTicket(pilot.Task.ID, models.MoveTicketRequest{Status: "done", ExpectedRevision: &updated.Task.Revision})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ = s.GetConnectionPilot(connectionTestID)
	if updated.State != "completed" {
		t.Fatal(updated.State)
	}
	if err = s.DeleteTicket(pilot.Task.ID, updated.Task.Revision); err != nil {
		t.Fatal(err)
	}
	if gone, err := s.GetConnectionPilot(connectionTestID); err != nil || gone != nil {
		t.Fatal("deleted task reported as connected", err)
	}
}

func TestInvalidConnectionPilotDoesNotCreateBoards(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	for _, tc := range []struct{ id, goal string }{{"bad", "Goal"}, {connectionTestID, " "}, {connectionTestID, strings.Repeat("a", 4001)}} {
		if _, err := s.CreateConnectionPilot(tc.id, tc.goal); err == nil {
			t.Fatal("invalid pilot accepted")
		}
	}
	if _, err := s.GetConnectionPilot("bad"); err == nil {
		t.Fatal("invalid lookup accepted")
	}
	if missing, err := s.GetConnectionPilot(connectionTestID); err != nil || missing != nil {
		t.Fatal("missing pilot", err)
	}
	_, count, _ := s.ListProjects("")
	if count != 0 {
		t.Fatal("invalid request created a board")
	}
}
