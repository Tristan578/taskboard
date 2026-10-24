package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

type Metadata struct {
	TicketID           string `json:"id,omitempty"`
	Status             string `json:"status,omitempty"`
	Priority           string `json:"priority,omitempty"`
	UserStory          string `json:"us,omitempty"`
	AcceptanceCriteria string `json:"ac,omitempty"`
	TechnicalDetails   string `json:"td,omitempty"`
	TestingDetails     string `json:"ts,omitempty"`
	LexoRank           string `json:"lr,omitempty"`
}

const metadataPrefix = "<!-- player2-metadata:"
const metadataSuffix = " -->"

func FormatIssueBody(description string, ticket *models.Ticket) string {
	meta := Metadata{
		TicketID: ticket.ID, Status: ticket.Status, Priority: ticket.Priority,
		UserStory:          ticket.UserStory,
		AcceptanceCriteria: ticket.AcceptanceCriteria,
		TechnicalDetails:   ticket.TechnicalDetails,
		TestingDetails:     ticket.TestingDetails,
		LexoRank:           ticket.LexoRank,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return description
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	comment := fmt.Sprintf("\n\n%s%s%s", metadataPrefix, encoded, metadataSuffix)

	// Clean existing metadata from description before appending new
	cleanDesc := stripMetadata(description)
	return cleanDesc + comment
}

func ParseIssueBody(body string) (description string, meta Metadata) {
	// 1. Try to find hidden HTML comment (New Format)
	if strings.Contains(body, metadataPrefix) {
		start := strings.Index(body, metadataPrefix)
		end := strings.Index(body[start:], metadataSuffix)
		if end != -1 {
			encoded := body[start+len(metadataPrefix) : start+end]
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err == nil {
				var m Metadata
				if err := json.Unmarshal(data, &m); err == nil {
					return strings.TrimSpace(body[:start] + body[start+end+len(metadataSuffix):]), m
				}
			}
		}
	}

	return body, Metadata{}
}

func stripMetadata(body string) string {
	start := strings.Index(body, metadataPrefix)
	if start < 0 {
		return body
	}
	end := strings.Index(body[start:], metadataSuffix)
	if end < 0 {
		return body
	}
	return strings.TrimSpace(body[:start] + body[start+end+len(metadataSuffix):])
}

// SyncProject reconciles content against the last acknowledged pair of hashes.
// A missing baseline never authorizes overwriting divergent content.
func SyncProject(ctx context.Context, client *Client, store SyncStore, projectID string) error {
	if client == nil || client.rest == nil || client.gql == nil {
		return fmt.Errorf("github client is not properly initialized")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	lockID := fmt.Sprintf("%d", time.Now().UnixNano())
	locked, err := store.AcquireSyncLock(projectID, lockID)
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("project sync already running")
	}
	defer store.ReleaseSyncLock(projectID, lockID)
	p, err := store.GetProject(projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("project not found")
	}
	if p.GitHubRepo == "" {
		return nil
	}
	owner, repo, err := ParseRepo(p.GitHubRepo)
	if err != nil {
		return err
	}
	issues, err := client.GetIssues(ctx, owner, repo)
	if err != nil {
		return err
	}
	tickets, _, err := store.ListTickets(models.TicketFilter{ProjectID: projectID})
	if err != nil {
		return err
	}
	deleted, err := store.ListDeletedTickets(projectID)
	if err != nil {
		return err
	}
	tombstones := map[int]bool{}
	for _, t := range deleted {
		if t.GitHubIssueNumber != nil {
			tombstones[*t.GitHubIssueNumber] = true
		}
	}
	local := map[int]models.Ticket{}
	for _, t := range tickets {
		if t.GitHubIssueNumber != nil {
			local[*t.GitHubIssueNumber] = t
		}
	}
	remote := map[int]Issue{}
	var failures []error
	record := func(err error) {
		if err != nil {
			failures = append(failures, err)
		}
	}
	for _, issue := range issues {
		remote[issue.Number] = issue
		if tombstones[issue.Number] {
			continue
		}
		_, identity := ParseIssueBody(issue.Body)
		if identity.TicketID != "" {
			for _, pending := range tickets {
				if pending.ID == identity.TicketID && pending.GitHubIssueNumber == nil {
					local[issue.Number] = pending
				}
			}
		}
		t, ok := local[issue.Number]
		if !ok {
			desc, meta := ParseIssueBody(issue.Body)
			if meta.Priority == "" {
				meta.Priority = "medium"
			}
			status := remoteStatus(issue, meta)
			// Imported closed issues are Review until evidence is verified here.
			draft := p.Strict && (meta.UserStory == "" || meta.AcceptanceCriteria == "")
			if draft {
				status = "todo"
			}
			created, e := store.CreateTicket(models.CreateTicketRequest{ProjectID: projectID, Title: issue.Title, Description: desc, Status: status, Priority: meta.Priority, UserStory: meta.UserStory, AcceptanceCriteria: meta.AcceptanceCriteria, TechnicalDetails: meta.TechnicalDetails, TestingDetails: meta.TestingDetails, IsDraft: draft})
			if e != nil {
				record(e)
				continue
			}
			created, e = store.UpdateTicket(created.ID, models.UpdateTicketRequest{GitHubIssueNumber: &issue.Number})
			if e != nil {
				record(e)
				continue
			}
			if created == nil {
				record(fmt.Errorf("import returned no ticket"))
				continue
			}
			record(saveBaseline(store, created, issue))
			continue
		}
		if t.IsDraft || t.GitHubIssueNumber == nil {
			continue
		}
		lh, rh := ticketHash(t), issueHash(issue)
		var base syncBaseline
		_ = json.Unmarshal([]byte(t.GitHubLastSyncedSHA), &base)
		if lh == rh {
			record(saveBaseline(store, &t, issue))
			continue
		}
		if base.Local == "" || (lh != base.Local && rh != base.Remote) {
			detail, _ := json.Marshal(issue)
			record(store.SetConflict(t.ID, string(detail)))
			record(fmt.Errorf("conflict on %s: both versions preserved; resolve explicitly", t.ID))
			continue
		}
		if rh != base.Remote {
			desc, meta := ParseIssueBody(issue.Body)
			if meta.Priority == "" {
				meta.Priority = "medium"
			}
			status := remoteStatus(issue, meta)
			if t.Status == "done" && strings.EqualFold(issue.State, "closed") {
				status = "done"
			}
			updated, e := store.UpdateTicket(t.ID, models.UpdateTicketRequest{ExpectedRevision: &t.Revision, Title: &issue.Title, Description: &desc, Status: &status, UserStory: &meta.UserStory, AcceptanceCriteria: &meta.AcceptanceCriteria, TechnicalDetails: &meta.TechnicalDetails, TestingDetails: &meta.TestingDetails, Priority: &meta.Priority})
			if e != nil {
				record(e)
				continue
			}
			if updated != nil {
				record(saveBaseline(store, updated, issue))
			}
			continue
		}
		if lh != base.Local {
			current, e := store.GetTicket(t.ID)
			if e != nil {
				record(e)
				continue
			}
			if current == nil || ticketHash(*current) != lh {
				record(fmt.Errorf("ticket changed during sync; retry"))
				continue
			}

			if e := client.UpdateIssue(ctx, owner, repo, issue.Number, t.Title, FormatIssueBody(t.Description, &t), mapStatusToGHState(t.Status)); e != nil {
				record(e)
				continue
			}
			pushed := Issue{Number: issue.Number, Title: t.Title, Body: FormatIssueBody(t.Description, &t), State: mapStatusToGHState(t.Status)}
			record(saveBaseline(store, &t, pushed))
		}
	}
	for _, t := range tickets {
		if t.IsDraft {
			continue
		}
		if t.GitHubIssueNumber != nil {
			if _, ok := remote[*t.GitHubIssueNumber]; !ok {
				record(fmt.Errorf("linked issue #%d missing; refusing to recreate or discard %s", *t.GitHubIssueNumber, t.ID))
			}
			continue
		}
		// Persist a stable ticket identity remotely so a retry after interruption adopts it.
		body := FormatIssueBody(t.Description, &t)
		var number int
		for _, issue := range issues {
			_, m := ParseIssueBody(issue.Body)
			if m.TicketID == t.ID {
				number = issue.Number
				break
			}
		}
		if number == 0 {
			number, err = client.CreateIssue(ctx, owner, repo, t.Title, body)
			if err != nil {
				record(err)
				continue
			}
		}
		updated, e := store.UpdateTicket(t.ID, models.UpdateTicketRequest{GitHubIssueNumber: &number})
		if e != nil {
			record(e)
			continue
		}
		if e = client.UpdateIssue(ctx, owner, repo, number, t.Title, body, mapStatusToGHState(t.Status)); e != nil {
			record(e)
			continue
		}
		if updated != nil {
			record(saveBaseline(store, updated, Issue{Number: number, Title: t.Title, Body: body, State: mapStatusToGHState(t.Status)}))
		}
	}
	for _, t := range deleted {
		if t.GitHubLastSyncedSHA == "deleted-acknowledged" {
			continue
		}
		if t.GitHubIssueNumber != nil {
			if e := client.UpdateIssue(ctx, owner, repo, *t.GitHubIssueNumber, t.Title, FormatIssueBody(t.Description, &t), "closed"); e != nil {
				record(e)
				continue
			}
		}
		record(store.AcknowledgeDeletedTicket(t.ID))
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	return store.MarkProjectSynced(projectID)
}

type SyncStore interface {
	GetProject(string) (*models.Project, error)
	ListTickets(models.TicketFilter) ([]models.Ticket, int, error)
	GetTicket(string) (*models.Ticket, error)
	CreateTicket(models.CreateTicketRequest) (*models.Ticket, error)
	UpdateTicket(string, models.UpdateTicketRequest) (*models.Ticket, error)
	ListDeletedTickets(string) ([]models.Ticket, error)
	AcknowledgeDeletedTicket(string) error
	SaveSyncBaseline(string, string) error
	SetConflict(string, string) error
	MarkProjectSynced(string) error
	AcquireSyncLock(string, string) (bool, error)
	ReleaseSyncLock(string, string) error
}
type syncBaseline struct {
	Local  string
	Remote string
}

func saveBaseline(store SyncStore, t *models.Ticket, i Issue) error {
	data, _ := json.Marshal(syncBaseline{ticketHash(*t), issueHash(i)})
	if err := store.SaveSyncBaseline(t.ID, string(data)); err != nil {
		return err
	}
	return store.SetConflict(t.ID, "")
}
func contentHash(title, desc, status, us, ac, td, ts, priority string) string {
	data, _ := json.Marshal([]string{title, strings.TrimSpace(desc), status, us, ac, td, ts, priority})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func ticketHash(t models.Ticket) string {
	return contentHash(t.Title, t.Description, t.Status, t.UserStory, t.AcceptanceCriteria, t.TechnicalDetails, t.TestingDetails, t.Priority)
}
func issueHash(i Issue) string {
	d, m := ParseIssueBody(i.Body)
	priority := m.Priority
	if priority == "" {
		priority = "medium"
	}
	status := remoteStatus(i, m)
	if strings.EqualFold(i.State, "closed") && m.Status == "done" {
		status = "done"
	}
	return contentHash(i.Title, d, status, m.UserStory, m.AcceptanceCriteria, m.TechnicalDetails, m.TestingDetails, priority)
}
func remoteStatus(i Issue, m Metadata) string {
	if strings.EqualFold(i.State, "closed") {
		return "review"
	}
	switch m.Status {
	case "todo", "in_progress", "review", "blocked":
		return m.Status
	}
	return mapGHStateToStatus(i.State, i.Labels)
}
func mapGHStateToStatus(state string, labels []string) string {
	if strings.EqualFold(state, "closed") {
		return "review"
	}
	for _, l := range labels {
		if l == "in-progress" || l == "doing" {
			return "in_progress"
		}
	}
	return "todo"
}
func mapStatusToGHState(status string) string {
	if status == "done" {
		return "closed"
	}
	return "open"
}

// ResolveConflict acknowledges only the saved remote version. A subsequent remote
// edit creates another conflict instead of being overwritten by the user's choice.
func ResolveConflict(ctx context.Context, client *Client, store SyncStore, id, choice string) error {
	t, err := store.GetTicket(id)
	if err != nil {
		return err
	}
	if t == nil || t.Work == nil || t.Work.Conflict == "" {
		return fmt.Errorf("no conflict for ticket")
	}
	var issue Issue
	if err = json.Unmarshal([]byte(t.Work.Conflict), &issue); err != nil {
		return err
	}
	if choice == "local" {
		data, _ := json.Marshal(syncBaseline{Local: "resolved-local", Remote: issueHash(issue)})
		if err = store.SaveSyncBaseline(id, string(data)); err != nil {
			return err
		}
	} else if choice == "remote" {
		// Mark local as acknowledged; the regular pull path still performs validation.
		data, _ := json.Marshal(syncBaseline{Local: ticketHash(*t), Remote: "resolved-remote"})
		if err = store.SaveSyncBaseline(id, string(data)); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("choice must be local or remote")
	}
	return SyncProject(ctx, client, store, t.ProjectID)
}
