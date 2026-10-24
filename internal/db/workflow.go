package db

import (
	"database/sql"
	"fmt"
	"github.com/Tristan578/taskboard/internal/models"
	"strings"
	"time"
)

type ValidationError struct{ Message string }

// ConflictError preserves the newer version when an editor or integration is stale.
type ConflictError struct{}

func (*ConflictError) Error() string {
	return "This task changed since you last read it. Review the latest version before trying again."
}

func (e *ValidationError) Error() string       { return e.Message }
func invalid(format string, args ...any) error { return &ValidationError{fmt.Sprintf(format, args...)} }

const StaleAfter = 15 * time.Minute

func (s *Store) validateTicket(t *models.Ticket, previousStatus string) error {
	if strings.TrimSpace(t.Title) == "" {
		return invalid("title is required")
	}
	switch t.Status {
	case "todo", "in_progress", "review", "blocked", "done":
	default:
		return invalid("invalid status: %s", t.Status)
	}
	switch t.Priority {
	case "low", "medium", "high", "urgent":
	default:
		return invalid("invalid priority: %s", t.Priority)
	}
	p, err := s.GetProject(t.ProjectID)
	if err != nil {
		return err
	}
	if p == nil {
		return invalid("project not found")
	}
	if p.Strict && !t.IsDraft && (strings.TrimSpace(t.UserStory) == "" || strings.TrimSpace(t.AcceptanceCriteria) == "") {
		return invalid("Strict Mode requires a user story and acceptance criteria")
	}
	if t.IsDraft && t.Status != "todo" {
		return invalid("drafts must remain in Todo; publish the ticket before starting work")
	}
	if t.Status == "done" && previousStatus != "done" && p.CompletionPolicy != "manual" {
		return invalid("Done requires verified merge or deployment evidence; use complete_ticket or ticket complete")
	}
	return nil
}

func (s *Store) GetWorkState(id string) (*models.WorkState, error) {
	var w models.WorkState
	err := s.db.QueryRow("SELECT agent,session_id,heartbeat_at,summary,completion_url,completed_at,conflict FROM work_state WHERE ticket_id=?", id).Scan(&w.Agent, &w.SessionID, &w.HeartbeatAt, &w.Summary, &w.CompletionURL, &w.CompletedAt, &w.Conflict)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.Stale = w.HeartbeatAt != nil && time.Since(*w.HeartbeatAt) > StaleAfter
	return &w, nil
}

// ReportActivity uses a session lease so competing agents cannot silently take over.
func (s *Store) ReportActivity(id string, a models.ActivityRequest) (*models.Ticket, error) {
	if strings.TrimSpace(a.Agent) == "" || strings.TrimSpace(a.SessionID) == "" || strings.TrimSpace(a.Summary) == "" {
		return nil, fmt.Errorf("agent, sessionId and summary are required")
	}
	t, err := s.GetTicket(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("ticket not found")
	}
	if t.Status == "done" {
		return nil, fmt.Errorf("reopen completed work before reporting activity")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var session string
	var heartbeat sql.NullTime
	err = tx.QueryRow("SELECT session_id,heartbeat_at FROM work_state WHERE ticket_id=?", id).Scan(&session, &heartbeat)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && session != a.SessionID && heartbeat.Valid && time.Since(heartbeat.Time) <= StaleAfter {
		return nil, fmt.Errorf("ticket has an active session owned by another agent")
	}
	_, err = tx.Exec(`INSERT INTO work_state(ticket_id,agent,session_id,heartbeat_at,summary) VALUES(?,?,?,?,?) ON CONFLICT(ticket_id) DO UPDATE SET agent=excluded.agent,session_id=excluded.session_id,heartbeat_at=excluded.heartbeat_at,summary=excluded.summary`, id, a.Agent, a.SessionID, time.Now().UTC(), a.Summary)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetTicket(id)
}

// CompleteVerified is called only after the integration has verified remote evidence.
func (s *Store) CompleteVerified(id, url string, expected ...int) (*models.Ticket, error) {
	t, err := s.GetTicket(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("ticket not found")
	}
	if len(expected) > 0 && t.Revision != expected[0] {
		return nil, &ConflictError{}
	}
	if t.IsDraft {
		return nil, fmt.Errorf("publish a draft before completing it")
	}
	if err = s.validateTicket(t, t.Status); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC()
	if _, err = tx.Exec(`INSERT INTO work_state(ticket_id,completion_url,completed_at) VALUES(?,?,?) ON CONFLICT(ticket_id) DO UPDATE SET completion_url=excluded.completion_url,completed_at=excluded.completed_at`, id, url, now); err != nil {
		return nil, err
	}
	result, err := tx.Exec("UPDATE tickets SET status='done',updated_at=?,revision=revision+1 WHERE id=? AND revision=? AND deleted_at IS NULL", now, id, t.Revision)
	if err != nil {
		return nil, err
	}
	if n, e := result.RowsAffected(); e != nil {
		return nil, e
	} else if n != 1 {
		return nil, &ConflictError{}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetTicket(id)
}

func (s *Store) SetConflict(id, detail string) error {
	_, err := s.db.Exec(`INSERT INTO work_state(ticket_id,conflict) VALUES(?,?) ON CONFLICT(ticket_id) DO UPDATE SET conflict=excluded.conflict`, id, detail)
	return err
}
func (s *Store) SaveSyncBaseline(id, baseline string) error {
	_, err := s.db.Exec("UPDATE tickets SET github_last_synced_sha=?,github_last_synced_at=? WHERE id=?", baseline, time.Now().UTC(), id)
	return err
}
func (s *Store) MarkProjectSynced(id string) error {
	_, err := s.db.Exec("UPDATE projects SET github_last_synced=? WHERE id=?", time.Now().UTC(), id)
	return err
}
func (s *Store) AcknowledgeDeletedTicket(id string) error {
	_, err := s.db.Exec("UPDATE tickets SET github_last_synced_sha='deleted-acknowledged' WHERE id=? AND deleted_at IS NOT NULL", id)
	return err
}
func (s *Store) AcquireSyncLock(projectID, owner string) (bool, error) {
	now := time.Now().UTC()
	r, err := s.db.Exec(`INSERT INTO sync_locks(project_id,owner,expires_at) VALUES(?,?,?) ON CONFLICT(project_id) DO UPDATE SET owner=excluded.owner,expires_at=excluded.expires_at WHERE sync_locks.expires_at < ?`, projectID, owner, now.Add(5*time.Minute), now)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n > 0, err
}
func (s *Store) ReleaseSyncLock(projectID, owner string) error {
	_, err := s.db.Exec("DELETE FROM sync_locks WHERE project_id=? AND owner=?", projectID, owner)
	return err
}

func (s *Store) RetryFailedSyncJobs() error {
	_, err := s.db.Exec("UPDATE sync_jobs SET status='pending',attempts=0,next_retry_at=NULL WHERE status='failed'")
	return err
}
