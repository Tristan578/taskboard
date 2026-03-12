package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/Tristan578/taskboard/internal/models"
)

type Store struct {
	db *sql.DB
}

func NewStore(database *sql.DB) *Store {
	return &Store{db: database}
}

func (s *Store) ClearData() error {
	tables := []string{
		"ticket_dependencies",
		"ticket_labels",
		"subtasks",
		"tickets",
		"labels",
		"teams",
		"projects",
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	for _, table := range tables {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			tx.Rollback()
			return fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	return tx.Commit()
}

func newID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

func (s *Store) ListProjects(status string) ([]models.Project, error) {
	query := "SELECT id, name, prefix, description, icon, color, status, github_repo, github_last_synced, strict, created_at, updated_at FROM projects"
	args := []any{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Prefix, &p.Description, &p.Icon, &p.Color, &p.Status, &p.GitHubRepo, &p.GitHubLastSynced, &p.Strict, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(id string) (*models.Project, error) {
	var p models.Project
	err := s.db.QueryRow(
		"SELECT id, name, prefix, description, icon, color, status, github_repo, github_last_synced, strict, created_at, updated_at FROM projects WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Prefix, &p.Description, &p.Icon, &p.Color, &p.Status, &p.GitHubRepo, &p.GitHubLastSynced, &p.Strict, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (s *Store) CreateProject(req models.CreateProjectRequest) (*models.Project, error) {
	p := models.Project{
		ID:          newID(),
		Name:        req.Name,
		Prefix:      req.Prefix,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		Status:      "active",
		GitHubRepo:  req.GitHubRepo,
		Strict:      req.Strict,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if p.Color == "" {
		p.Color = "#3B82F6"
	}

	_, err := s.db.Exec(
		"INSERT INTO projects (id, name, prefix, description, icon, color, status, github_repo, strict, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Prefix, p.Description, p.Icon, p.Color, p.Status, p.GitHubRepo, p.Strict, p.CreatedAt, p.UpdatedAt,
	)
	return &p, err
}

func (s *Store) UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error) {
	p, err := s.GetProject(id)
	if err != nil || p == nil {
		return nil, err
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Prefix != nil {
		p.Prefix = *req.Prefix
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Icon != nil {
		p.Icon = *req.Icon
	}
	if req.Color != nil {
		p.Color = *req.Color
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.GitHubRepo != nil {
		p.GitHubRepo = *req.GitHubRepo
	}
	if req.Strict != nil {
		p.Strict = *req.Strict
	}
	p.UpdatedAt = time.Now()

	_, err = s.db.Exec(
		"UPDATE projects SET name=?, prefix=?, description=?, icon=?, color=?, status=?, github_repo=?, strict=?, updated_at=? WHERE id=?",
		p.Name, p.Prefix, p.Description, p.Icon, p.Color, p.Status, p.GitHubRepo, p.Strict, p.UpdatedAt, p.ID,
	)
	return p, err
}

func (s *Store) DeleteProject(id string) error {
	_, err := s.db.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (s *Store) ListTeams() ([]models.Team, error) {
	rows, err := s.db.Query("SELECT id, name, color, created_at FROM teams ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (s *Store) GetTeam(id string) (*models.Team, error) {
	var t models.Team
	err := s.db.QueryRow("SELECT id, name, color, created_at FROM teams WHERE id = ?", id).
		Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (s *Store) CreateTeam(req models.CreateTeamRequest) (*models.Team, error) {
	t := models.Team{
		ID:        newID(),
		Name:      req.Name,
		Color:     req.Color,
		CreatedAt: time.Now(),
	}
	if t.Color == "" {
		t.Color = "#6366F1"
	}

	_, err := s.db.Exec("INSERT INTO teams (id, name, color, created_at) VALUES (?, ?, ?, ?)",
		t.ID, t.Name, t.Color, t.CreatedAt)
	return &t, err
}

func (s *Store) UpdateTeam(id string, req models.UpdateTeamRequest) (*models.Team, error) {
	t, err := s.GetTeam(id)
	if err != nil || t == nil {
		return nil, err
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Color != nil {
		t.Color = *req.Color
	}

	_, err = s.db.Exec("UPDATE teams SET name=?, color=? WHERE id=?", t.Name, t.Color, t.ID)
	return t, err
}

func (s *Store) DeleteTeam(id string) error {
	_, err := s.db.Exec("DELETE FROM teams WHERE id = ?", id)
	return err
}

func (s *Store) nextTicketNumber(projectID string) (int, error) {
	var num int
	err := s.db.QueryRow("SELECT COALESCE(MAX(number), 0) + 1 FROM tickets WHERE project_id = ?", projectID).Scan(&num)
	return num, err
}

func (s *Store) ListTickets(filter models.TicketFilter) ([]models.Ticket, error) {
	query := `SELECT t.id, t.project_id, t.team_id, t.number, t.title, t.description,
		t.status, t.priority, t.due_date, t.position, t.lexo_rank, t.github_issue_number, t.github_last_synced_at,
		t.github_last_synced_sha, t.user_story, t.acceptance_criteria, t.technical_details, t.testing_details,
		t.is_draft, t.created_at, t.updated_at, COALESCE(p.prefix, '') as project_prefix
		FROM tickets t LEFT JOIN projects p ON t.project_id = p.id WHERE t.deleted_at IS NULL`
	args := []any{}

	if filter.ProjectID != "" {
		query += " AND t.project_id = ?"
		args = append(args, filter.ProjectID)
	}
	if filter.TeamID != "" {
		query += " AND t.team_id = ?"
		args = append(args, filter.TeamID)
	}
	if filter.Status != "" {
		query += " AND t.status = ?"
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		query += " AND t.priority = ?"
		args = append(args, filter.Priority)
	}
	query += " ORDER BY t.lexo_rank ASC, t.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.TeamID, &t.Number, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.DueDate, &t.Position, &t.LexoRank, &t.GitHubIssueNumber, &t.GitHubLastSyncedAt,
			&t.GitHubLastSyncedSHA, &t.UserStory, &t.AcceptanceCriteria, &t.TechnicalDetails, &t.TestingDetails,
			&t.IsDraft, &t.CreatedAt, &t.UpdatedAt, &t.ProjectPrefix); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range tickets {
		tickets[i].Labels, _ = s.getTicketLabels(tickets[i].ID)
		tickets[i].Subtasks, _ = s.getTicketSubtasks(tickets[i].ID)
		tickets[i].BlockedBy, _ = s.getTicketBlockedBy(tickets[i].ID)
	}

	return tickets, nil
}

func (s *Store) GetTicket(id string) (*models.Ticket, error) {
	var t models.Ticket
	err := s.db.QueryRow(
		`SELECT t.id, t.project_id, t.team_id, t.number, t.title, t.description,
		t.status, t.priority, t.due_date, t.position, t.lexo_rank, t.github_issue_number, t.github_last_synced_at,
		t.github_last_synced_sha, t.user_story, t.acceptance_criteria, t.technical_details, t.testing_details,
		t.is_draft, t.created_at, t.updated_at, COALESCE(p.prefix, '') as project_prefix
		FROM tickets t LEFT JOIN projects p ON t.project_id = p.id WHERE t.id = ?`, id,
	).Scan(&t.ID, &t.ProjectID, &t.TeamID, &t.Number, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.DueDate, &t.Position, &t.LexoRank, &t.GitHubIssueNumber, &t.GitHubLastSyncedAt,
		&t.GitHubLastSyncedSHA, &t.UserStory, &t.AcceptanceCriteria, &t.TechnicalDetails, &t.TestingDetails,
		&t.IsDraft, &t.CreatedAt, &t.UpdatedAt, &t.ProjectPrefix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t.Labels, _ = s.getTicketLabels(t.ID)
	t.Subtasks, _ = s.getTicketSubtasks(t.ID)
	t.BlockedBy, _ = s.getTicketBlockedBy(t.ID)

	return &t, nil
}

func (s *Store) CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error) {
	num, err := s.nextTicketNumber(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("getting next ticket number: %w", err)
	}

	status := req.Status
	if status == "" {
		status = "todo"
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	t := models.Ticket{
		ID:                 newID(),
		ProjectID:          req.ProjectID,
		TeamID:             req.TeamID,
		Number:             num,
		Title:              req.Title,
		Description:        req.Description,
		Status:             status,
		Priority:           priority,
		Position:           float64(num) * 1000,
		LexoRank:           fmt.Sprintf("%010d", num*1000),
		UserStory:          req.UserStory,
		AcceptanceCriteria: req.AcceptanceCriteria,
		TechnicalDetails:   req.TechnicalDetails,
		TestingDetails:     req.TestingDetails,
		IsDraft:            req.IsDraft,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if req.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err == nil {
			t.DueDate = &parsed
		}
	}

	_, err = s.db.Exec(
		`INSERT INTO tickets (id, project_id, team_id, number, title, description, status, priority, due_date, position, lexo_rank, user_story, acceptance_criteria, technical_details, testing_details, is_draft, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.TeamID, t.Number, t.Title, t.Description, t.Status, t.Priority, t.DueDate, t.Position, t.LexoRank, t.UserStory, t.AcceptanceCriteria, t.TechnicalDetails, t.TestingDetails, t.IsDraft, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(req.Labels) > 0 {
		for _, labelID := range req.Labels {
			s.db.Exec("INSERT OR IGNORE INTO ticket_labels (ticket_id, label_id) VALUES (?, ?)", t.ID, labelID)
		}
	}

	if len(req.BlockedBy) > 0 {
		for _, blockerID := range req.BlockedBy {
			s.db.Exec("INSERT OR IGNORE INTO ticket_dependencies (ticket_id, blocked_by_id) VALUES (?, ?)", t.ID, blockerID)
		}
	}

	return s.GetTicket(t.ID)
}

func (s *Store) UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error) {
	t, err := s.GetTicket(id)
	if err != nil || t == nil {
		return nil, err
	}

	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.Position != nil {
		t.Position = *req.Position
	}
	if req.LexoRank != nil {
		t.LexoRank = *req.LexoRank
	}
	if req.TeamID != nil {
		t.TeamID = req.TeamID
	}
	if req.GitHubIssueNumber != nil {
		t.GitHubIssueNumber = req.GitHubIssueNumber
	}
	if req.GitHubLastSyncedSHA != nil {
		t.GitHubLastSyncedSHA = *req.GitHubLastSyncedSHA
	}
	if req.UserStory != nil {
		t.UserStory = *req.UserStory
	}
	if req.AcceptanceCriteria != nil {
		t.AcceptanceCriteria = *req.AcceptanceCriteria
	}
	if req.TechnicalDetails != nil {
		t.TechnicalDetails = *req.TechnicalDetails
	}
	if req.TestingDetails != nil {
		t.TestingDetails = *req.TestingDetails
	}
	if req.IsDraft != nil {
		t.IsDraft = *req.IsDraft
	}
	if req.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err == nil {
			t.DueDate = &parsed
		}
	}
	t.UpdatedAt = time.Now()

	_, err = s.db.Exec(
		`UPDATE tickets SET team_id=?, title=?, description=?, status=?, priority=?, due_date=?, position=?, lexo_rank=?, github_issue_number=?, github_last_synced_sha=?, user_story=?, acceptance_criteria=?, technical_details=?, testing_details=?, is_draft=?, updated_at=? WHERE id=?`,
		t.TeamID, t.Title, t.Description, t.Status, t.Priority, t.DueDate, t.Position, t.LexoRank, t.GitHubIssueNumber, t.GitHubLastSyncedSHA, t.UserStory, t.AcceptanceCriteria, t.TechnicalDetails, t.TestingDetails, t.IsDraft, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return nil, err
	}

	if req.Labels != nil {
		s.db.Exec("DELETE FROM ticket_labels WHERE ticket_id = ?", id)
		for _, labelID := range req.Labels {
			s.db.Exec("INSERT OR IGNORE INTO ticket_labels (ticket_id, label_id) VALUES (?, ?)", id, labelID)
		}
	}

	if req.BlockedBy != nil {
		s.db.Exec("DELETE FROM ticket_dependencies WHERE ticket_id = ?", id)
		for _, blockerID := range req.BlockedBy {
			s.db.Exec("INSERT OR IGNORE INTO ticket_dependencies (ticket_id, blocked_by_id) VALUES (?, ?)", id, blockerID)
		}
	}

	return s.GetTicket(id)
}

func (s *Store) MoveTicket(id string, req models.MoveTicketRequest) (*models.Ticket, error) {
	now := time.Now()
	position := float64(0)
	lexoRank := ""

	if req.Position != nil {
		position = *req.Position
		// Very basic LexoRank: just convert position to zero-padded string
		// In a real brick building, we'd use a LexoRank library to find the midpoint
		lexoRank = fmt.Sprintf("%010d", int(position))
	} else {
		var maxPos float64
		s.db.QueryRow("SELECT COALESCE(MAX(position), 0) + 1000 FROM tickets WHERE status = ?", req.Status).Scan(&maxPos)
		position = maxPos
		lexoRank = fmt.Sprintf("%010d", int(position))
	}

	_, err := s.db.Exec("UPDATE tickets SET status=?, position=?, lexo_rank=?, updated_at=? WHERE id=?",
		req.Status, position, lexoRank, now, id)
	if err != nil {
		return nil, err
	}
	return s.GetTicket(id)
}

func (s *Store) DeleteTicket(id string) error {
	_, err := s.db.Exec("UPDATE tickets SET deleted_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (s *Store) PurgeDeletedTickets(projectID string) error {
	_, err := s.db.Exec("DELETE FROM tickets WHERE project_id = ? AND deleted_at IS NOT NULL", projectID)
	return err
}

func (s *Store) ListDeletedTickets(projectID string) ([]models.Ticket, error) {
	query := `SELECT t.id, t.project_id, t.team_id, t.number, t.title, t.description,
		t.status, t.priority, t.due_date, t.position, t.lexo_rank, t.github_issue_number, t.github_last_synced_at,
		t.github_last_synced_sha, t.user_story, t.acceptance_criteria, t.technical_details, t.testing_details,
		t.is_draft, t.created_at, t.updated_at, COALESCE(p.prefix, '') as project_prefix
		FROM tickets t LEFT JOIN projects p ON t.project_id = p.id 
		WHERE t.project_id = ? AND t.deleted_at IS NOT NULL`
	
	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.TeamID, &t.Number, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.DueDate, &t.Position, &t.LexoRank, &t.GitHubIssueNumber, &t.GitHubLastSyncedAt,
			&t.GitHubLastSyncedSHA, &t.UserStory, &t.AcceptanceCriteria, &t.TechnicalDetails, &t.TestingDetails,
			&t.IsDraft, &t.CreatedAt, &t.UpdatedAt, &t.ProjectPrefix); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (s *Store) GetBoard(projectID string) (*models.Board, error) {
	statuses := []string{"todo", "in_progress", "done"}
	board := &models.Board{
		ProjectID: projectID,
		Columns:   make([]models.Column, len(statuses)),
	}

	for i, status := range statuses {
		filter := models.TicketFilter{Status: status}
		if projectID != "" {
			filter.ProjectID = projectID
		}
		tickets, err := s.ListTickets(filter)
		if err != nil {
			return nil, err
		}
		if tickets == nil {
			tickets = []models.Ticket{}
		}
		board.Columns[i] = models.Column{
			Status:  status,
			Tickets: tickets,
		}
	}

	return board, nil
}

func (s *Store) ListLabels() ([]models.Label, error) {
	rows, err := s.db.Query("SELECT id, name, color FROM labels ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []models.Label
	for rows.Next() {
		var l models.Label
		if err := rows.Scan(&l.ID, &l.Name, &l.Color); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (s *Store) CreateLabel(req models.CreateLabelRequest) (*models.Label, error) {
	l := models.Label{ID: newID(), Name: req.Name, Color: req.Color}
	_, err := s.db.Exec("INSERT INTO labels (id, name, color) VALUES (?, ?, ?)", l.ID, l.Name, l.Color)
	return &l, err
}

func (s *Store) UpdateLabel(id string, req models.UpdateLabelRequest) (*models.Label, error) {
	var l models.Label
	err := s.db.QueryRow("SELECT id, name, color FROM labels WHERE id = ?", id).Scan(&l.ID, &l.Name, &l.Color)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		l.Name = *req.Name
	}
	if req.Color != nil {
		l.Color = *req.Color
	}
	_, err = s.db.Exec("UPDATE labels SET name=?, color=? WHERE id=?", l.Name, l.Color, l.ID)
	return &l, err
}

func (s *Store) DeleteLabel(id string) error {
	_, err := s.db.Exec("DELETE FROM labels WHERE id = ?", id)
	return err
}

func (s *Store) AddSubtask(ticketID string, req models.CreateSubtaskRequest) (*models.Subtask, error) {
	var maxPos int
	s.db.QueryRow("SELECT COALESCE(MAX(position), -1) + 1 FROM subtasks WHERE ticket_id = ?", ticketID).Scan(&maxPos)

	st := models.Subtask{
		ID:       newID(),
		TicketID: ticketID,
		Title:    req.Title,
		Position: maxPos,
	}
	_, err := s.db.Exec("INSERT INTO subtasks (id, ticket_id, title, completed, position) VALUES (?, ?, ?, ?, ?)",
		st.ID, st.TicketID, st.Title, st.Completed, st.Position)
	return &st, err
}

func (s *Store) ToggleSubtask(id string) (*models.Subtask, error) {
	_, err := s.db.Exec("UPDATE subtasks SET completed = NOT completed WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	var st models.Subtask
	err = s.db.QueryRow("SELECT id, ticket_id, title, completed, position FROM subtasks WHERE id = ?", id).
		Scan(&st.ID, &st.TicketID, &st.Title, &st.Completed, &st.Position)
	return &st, err
}

func (s *Store) DeleteSubtask(id string) error {
	_, err := s.db.Exec("DELETE FROM subtasks WHERE id = ?", id)
	return err
}

func (s *Store) getTicketLabels(ticketID string) ([]models.Label, error) {
	rows, err := s.db.Query(
		"SELECT l.id, l.name, l.color FROM labels l JOIN ticket_labels tl ON l.id = tl.label_id WHERE tl.ticket_id = ?",
		ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []models.Label
	for rows.Next() {
		var l models.Label
		if err := rows.Scan(&l.ID, &l.Name, &l.Color); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (s *Store) getTicketSubtasks(ticketID string) ([]models.Subtask, error) {
	rows, err := s.db.Query(
		"SELECT id, ticket_id, title, completed, position FROM subtasks WHERE ticket_id = ? ORDER BY position",
		ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subtasks []models.Subtask
	for rows.Next() {
		var st models.Subtask
		if err := rows.Scan(&st.ID, &st.TicketID, &st.Title, &st.Completed, &st.Position); err != nil {
			return nil, err
		}
		subtasks = append(subtasks, st)
	}
	return subtasks, rows.Err()
}

func (s *Store) getTicketBlockedBy(ticketID string) ([]string, error) {
	rows, err := s.db.Query("SELECT blocked_by_id FROM ticket_dependencies WHERE ticket_id = ?", ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) QueueSyncJob(projectID, ticketID, action string, payload any) error {
	payloadJSON, _ := json.Marshal(payload)
	_, err := s.db.Exec(
		"INSERT INTO sync_jobs (id, project_id, ticket_id, action, payload, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?)",
		newID(), projectID, ticketID, action, string(payloadJSON), time.Now(), time.Now(),
	)
	return err
}

func (s *Store) GetPendingSyncJobs() ([]models.SyncJob, error) {
	rows, err := s.db.Query("SELECT id, project_id, ticket_id, action, payload, status, attempts, last_error, created_at, updated_at FROM sync_jobs WHERE status IN ('pending', 'failed') AND attempts < 5 ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.SyncJob
	for rows.Next() {
		var j models.SyncJob
		if err := rows.Scan(&j.ID, &j.ProjectID, &j.TicketID, &j.Action, &j.Payload, &j.Status, &j.Attempts, &j.LastError, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (s *Store) UpdateSyncJobStatus(id, status string, attempts int, lastError string) error {
	_, err := s.db.Exec(
		"UPDATE sync_jobs SET status = ?, attempts = ?, last_error = ?, updated_at = ? WHERE id = ?",
		status, attempts, lastError, time.Now(), id,
	)
	return err
}
