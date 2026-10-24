package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

func prepareRepeat(t *models.Ticket, resetAnchor bool) error {
	switch t.RepeatRule {
	case "":
		t.RepeatAnchor = 0
		return nil
	case "daily", "weekly", "monthly":
	default:
		return invalid("choose daily, weekly, monthly, or no repeat")
	}
	if t.DueDate == nil {
		return invalid("choose a due date for a repeating task")
	}
	if resetAnchor || t.RepeatAnchor < 1 {
		t.RepeatAnchor = t.DueDate.Day()
	}
	return nil
}

// Calendar arithmetic avoids daylight-saving offsets and preserves a monthly
// anchor (January 31 -> February 28 -> March 31). Missed periods are skipped.
func nextRepeatDate(t *models.Ticket, now time.Time) time.Time {
	due := *t.DueDate
	today, _ := time.Parse("2006-01-02", now.Format("2006-01-02"))
	for {
		switch t.RepeatRule {
		case "daily":
			due = due.AddDate(0, 0, 1)
		case "weekly":
			due = due.AddDate(0, 0, 7)
		case "monthly":
			first := time.Date(due.Year(), due.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			last := first.AddDate(0, 1, -1).Day()
			day := t.RepeatAnchor
			if day > last {
				day = last
			}
			due = first.AddDate(0, 0, day-1)
		}
		if due.After(today) {
			return due
		}
	}
}

func createNextOccurrence(tx *sql.Tx, t *models.Ticket, now time.Time) error {
	if t.RepeatRule == "" {
		return nil
	}
	if err := prepareRepeat(t, false); err != nil {
		return err
	}
	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM tickets WHERE repeat_source_id=?", t.ID).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	id := newID()
	var number int
	if err := tx.QueryRow("SELECT COALESCE(MAX(number),0)+1 FROM tickets WHERE project_id=?", t.ProjectID).Scan(&number); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT INTO tickets(id,project_id,team_id,number,title,description,status,priority,due_date,position,lexo_rank,user_story,acceptance_criteria,technical_details,testing_details,is_draft,repeat_rule,repeat_anchor,repeat_source_id,created_at,updated_at)
 VALUES(?,?,?,?,?,?,'todo',?,?,?,?,?,?,?,?,0,?,?,?,?,?)`, id, t.ProjectID, t.TeamID, number, t.Title, t.Description, t.Priority, nextRepeatDate(t, now), float64(number)*1000, fmt.Sprintf("%010d", number*1000), t.UserStory, t.AcceptanceCriteria, t.TechnicalDetails, t.TestingDetails, t.RepeatRule, t.RepeatAnchor, t.ID, now, now)
	if err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO ticket_labels(ticket_id,label_id) SELECT ?,label_id FROM ticket_labels WHERE ticket_id=?", id, t.ID); err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO subtasks(id,ticket_id,title,completed,position) SELECT lower(hex(randomblob(16))),?,title,0,position FROM subtasks WHERE ticket_id=?", id, t.ID)
	return err
}
