package db

import (
	"fmt"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

type DeletedTask struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Board     string    `json:"board"`
	DeletedAt time.Time `json:"deletedAt"`
	Revision  int       `json:"revision"`
	Linked    bool      `json:"linked"`
}

func (s *Store) Trash() ([]DeletedTask, error) {
	rows, err := s.db.Query(`SELECT t.id,t.title,p.name,t.deleted_at,t.revision,t.github_issue_number IS NOT NULL FROM tickets t JOIN projects p ON t.project_id=p.id WHERE t.deleted_at IS NOT NULL ORDER BY t.deleted_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []DeletedTask{}
	for rows.Next() {
		var t DeletedTask
		if err = rows.Scan(&t.ID, &t.Title, &t.Board, &t.DeletedAt, &t.Revision, &t.Linked); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (s *Store) RestoreTicket(id string, revision int) (*models.Ticket, error) {
	var projectID string
	if err := s.db.QueryRow("SELECT project_id FROM tickets WHERE id=?", id).Scan(&projectID); err != nil {
		return nil, fmt.Errorf("deleted task was not found")
	}
	owner := newID()
	ok, err := s.AcquireSyncLock(projectID, owner)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, invalid("GitHub is syncing this board. Try restoring the task in a moment.")
	}
	defer func() { _ = s.ReleaseSyncLock(projectID, owner) }()
	result, err := s.db.Exec(`UPDATE tickets SET deleted_at=NULL,updated_at=?,revision=revision+1,github_last_synced_sha='' WHERE id=? AND revision=? AND deleted_at IS NOT NULL`, time.Now(), id, revision)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		t, e := s.GetTicket(id)
		if e == nil && t != nil && t.Revision == revision+1 {
			return t, nil
		} // retry after a lost response
		return nil, &ConflictError{}
	}
	return s.GetTicket(id)
}
