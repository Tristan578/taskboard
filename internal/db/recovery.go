package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

type RecoverySnapshot struct {
	Projects []models.Project
	Teams    []models.Team
	Labels   []models.Label
	Tickets  []models.Ticket
}
type RecoveryPreview struct {
	ID     string   `json:"id"`
	Boards []string `json:"boards"`
	Tasks  int      `json:"tasks"`
}

// ReadBackup accepts only schemas produced by the bundled migrations. No SQL,
// triggers or views supplied by an uploaded file are executed during migration.
// The caller owns an isolated staging copy, never the live database.
func ReadBackup(filename string) (*RecoverySnapshot, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(absolute)
	// SQLite file URIs require forward slashes and / before a Windows drive
	// letter. Escape the path separately so # and ? cannot become URI options.
	if len(uriPath) >= 2 && uriPath[1] == ':' {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	raw, err := sql.Open("sqlite", u.String()+"?mode=ro&_pragma=trusted_schema(0)&_pragma=query_only(1)")
	if err != nil {
		return nil, err
	}
	raw.SetMaxOpenConns(1)
	if err = validateBackupSchema(raw); err != nil {
		_ = raw.Close()
		return nil, invalid("This is not a supported Player2 backup: %v", err)
	}
	if err = raw.Close(); err != nil {
		return nil, err
	}
	database, err := OpenAt(filename)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	source := NewStore(database)
	result := &RecoverySnapshot{}
	if result.Projects, _, err = source.ListProjects(""); err != nil {
		return nil, err
	}
	if result.Teams, _, err = source.ListTeams(); err != nil {
		return nil, err
	}
	if result.Labels, _, err = source.ListLabels(); err != nil {
		return nil, err
	}
	if result.Tickets, _, err = source.ListTickets(models.TicketFilter{}); err != nil {
		return nil, err
	}
	if len(result.Projects) > 500 || len(result.Tickets) > 20000 {
		return nil, invalid("This backup is too large for guided recovery")
	}
	return result, nil
}

func validateBackupSchema(raw *sql.DB) error {
	// Inspect schema text before selecting any values from an untrusted table;
	// a generated column or view must not be evaluated as migration history.
	var historySQL string
	if err := raw.QueryRow("SELECT sql FROM sqlite_schema WHERE type='table' AND name='schema_migrations'").Scan(&historySQL); err != nil {
		return fmt.Errorf("missing migration history")
	}
	knownHistory := "CREATE TABLE schema_migrations ( version TEXT PRIMARY KEY, applied_at DATETIME DEFAULT CURRENT_TIMESTAMP )"
	if strings.Join(strings.Fields(historySQL), " ") != knownHistory {
		return fmt.Errorf("unexpected migration history structure")
	}
	rows, err := raw.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("missing migration history")
	}
	versions := []string{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		versions = append(versions, name)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	entries, err := migrationsFS.ReadDir(migrations)
	if err != nil {
		return err
	}
	if len(versions) == 0 || len(versions) > len(entries) {
		return fmt.Errorf("unsupported backup version")
	}
	reference, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer reference.Close()
	reference.SetMaxOpenConns(1)
	if _, err = reference.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
 )`); err != nil {
		return err
	}
	for i, name := range versions {
		if entries[i].Name() != name {
			return fmt.Errorf("incomplete or newer migration history")
		}
		contents, e := migrationsFS.ReadFile(path.Join(migrations, name))
		if e != nil {
			return e
		}
		if _, e = reference.Exec(string(contents)); e != nil {
			return e
		}
	}
	actual, err := schemaDescription(raw)
	if err != nil {
		return err
	}
	expected, err := schemaDescription(reference)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("unexpected database structure")
	}
	var check string
	if err = raw.QueryRow("PRAGMA quick_check").Scan(&check); err != nil || check != "ok" {
		return fmt.Errorf("backup integrity check failed")
	}
	rows, err = raw.Query("PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("backup contains broken task references")
	}
	return rows.Err()
}

func schemaDescription(database *sql.DB) ([]string, error) {
	rows, err := database.Query("SELECT type,name,tbl_name,COALESCE(sql,'') FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var kind, name, table, query string
		if err = rows.Scan(&kind, &name, &table, &query); err != nil {
			return nil, err
		}
		result = append(result, kind+"|"+name+"|"+table+"|"+strings.Join(strings.Fields(query), " "))
	}
	return result, rows.Err()
}

func (snapshot *RecoverySnapshot) Preview(id string) RecoveryPreview {
	p := RecoveryPreview{ID: id, Boards: []string{}, Tasks: len(snapshot.Tickets)}
	for _, board := range snapshot.Projects {
		p.Boards = append(p.Boards, board.Name)
	}
	return p
}

func (s *Store) RecoveryReceipt(requestID string) ([]string, error) {
	var raw string
	err := s.db.QueryRow("SELECT project_ids FROM restore_receipts WHERE request_id=?", requestID).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	err = json.Unmarshal([]byte(raw), &ids)
	return ids, err
}

// RestoreBoards copies content into new IDs in one transaction. No current
// content, GitHub links, sync jobs or active agent sessions are replaced.
func (s *Store) RestoreBoards(requestID string, snapshot *RecoverySnapshot) ([]string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var receipt string
	err = tx.QueryRow("SELECT project_ids FROM restore_receipts WHERE request_id=?", requestID).Scan(&receipt)
	if err == nil {
		var ids []string
		err = json.Unmarshal([]byte(receipt), &ids)
		return ids, err
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	projects, teams, labels, tickets := map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}
	ids := []string{}
	now := time.Now()
	for _, p := range snapshot.Projects {
		id := newID()
		projects[p.ID] = id
		ids = append(ids, id)
		_, err = tx.Exec(`INSERT INTO projects(id,name,prefix,description,icon,color,status,strict,completion_policy,created_at,updated_at) VALUES(?,?,?,?,?,?,'active',?,?,?,?)`, id, p.Name+" (restored)", "R"+id[18:], p.Description, p.Icon, p.Color, p.Strict, p.CompletionPolicy, now, now)
		if err != nil {
			return nil, err
		}
	}
	for _, t := range snapshot.Teams {
		id := newID()
		teams[t.ID] = id
		if _, err = tx.Exec("INSERT INTO teams(id,name,color,created_at) VALUES(?,?,?,?)", id, t.Name, t.Color, now); err != nil {
			return nil, err
		}
	}
	for _, l := range snapshot.Labels {
		id := newID()
		labels[l.ID] = id
		if _, err = tx.Exec("INSERT INTO labels(id,name,color) VALUES(?,?,?)", id, l.Name, l.Color); err != nil {
			return nil, err
		}
	}
	for _, t := range snapshot.Tickets {
		tickets[t.ID] = newID()
	}
	for _, t := range snapshot.Tickets {
		id := tickets[t.ID]
		var team any
		if t.TeamID != nil {
			team = teams[*t.TeamID]
		}
		status := t.Status
		if status == "in_progress" {
			status = "todo"
		}
		for _, p := range snapshot.Projects {
			if p.ID == t.ProjectID && p.CompletionPolicy == "verified" && status == "done" {
				status = "review"
			}
		}
		if err = prepareRepeat(&t, false); err != nil {
			return nil, err
		}
		_, err = tx.Exec(`INSERT INTO tickets(id,project_id,team_id,number,title,description,status,priority,due_date,position,lexo_rank,user_story,acceptance_criteria,technical_details,testing_details,is_draft,repeat_rule,repeat_anchor,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, projects[t.ProjectID], team, t.Number, t.Title, t.Description, status, t.Priority, t.DueDate, t.Position, t.LexoRank, t.UserStory, t.AcceptanceCriteria, t.TechnicalDetails, t.TestingDetails, t.IsDraft, t.RepeatRule, t.RepeatAnchor, t.CreatedAt, now)
		if err != nil {
			return nil, err
		}
		for _, l := range t.Labels {
			if _, err = tx.Exec("INSERT INTO ticket_labels(ticket_id,label_id) VALUES(?,?)", id, labels[l.ID]); err != nil {
				return nil, err
			}
		}
		for _, sub := range t.Subtasks {
			if _, err = tx.Exec("INSERT INTO subtasks(id,ticket_id,title,completed,position) VALUES(?,?,?,?,?)", newID(), id, sub.Title, sub.Completed, sub.Position); err != nil {
				return nil, err
			}
		}
	}
	for _, t := range snapshot.Tickets {
		for _, blocker := range t.BlockedBy {
			if target := tickets[blocker]; target != "" {
				if _, err = tx.Exec("INSERT INTO ticket_dependencies(ticket_id,blocked_by_id) VALUES(?,?)", tickets[t.ID], target); err != nil {
					return nil, err
				}
			}
		}
		if t.RepeatSourceID != nil {
			if source := tickets[*t.RepeatSourceID]; source != "" {
				if _, err = tx.Exec("UPDATE tickets SET repeat_source_id=? WHERE id=?", source, tickets[t.ID]); err != nil {
					return nil, err
				}
			}
		}
	}
	payload, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO restore_receipts(request_id,project_ids,created_at) VALUES(?,?,?)", requestID, string(payload), now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}
