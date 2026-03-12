package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tristan578/taskboard/internal/models"
)

type Metadata struct {
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
					return strings.TrimSpace(body[:start]), m
				}
			}
		}
	}

	// 2. Fallback to YAML Frontmatter (Legacy Format for migration)
	if strings.HasPrefix(body, "---") {
		parts := strings.SplitN(body, "---", 3)
		if len(parts) >= 3 {
			// We won't re-implement full yaml parsing here to avoid bloat, 
			// just return the desc. The next push will "repair" it to HTML format.
			return strings.TrimSpace(parts[2]), Metadata{}
		}
	}

	return body, Metadata{}
}

func stripMetadata(body string) string {
	if idx := strings.Index(body, metadataPrefix); idx != -1 {
		return strings.TrimSpace(body[:idx])
	}
	// Also strip legacy YAML if present
	if strings.HasPrefix(body, "---") {
		parts := strings.SplitN(body, "---", 3)
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[2])
		}
	}
	return body
}

func SyncProject(ctx context.Context, client *Client, store interface {
	GetProject(id string) (*models.Project, error)
	ListTickets(filter models.TicketFilter) ([]models.Ticket, error)
	GetTicket(id string) (*models.Ticket, error)
	UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error)
	CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error)
	UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error)
	ListDeletedTickets(projectID string) ([]models.Ticket, error)
	PurgeDeletedTickets(projectID string) error
}, projectID string) error {
	p, err := store.GetProject(projectID)
	if err != nil || p == nil {
		return fmt.Errorf("getting project: %w", err)
	}

	if p.GitHubRepo == "" {
		return nil // Not linked
	}

	owner, repo, err := ParseRepo(p.GitHubRepo)
	if err != nil {
		return err
	}

	// 1. Fetch GitHub Issues
	ghIssues, err := client.GetIssues(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("fetching github issues: %w", err)
	}

	// 2. Fetch Local Tickets
	localTickets, err := store.ListTickets(models.TicketFilter{ProjectID: projectID})
	if err != nil {
		return fmt.Errorf("listing local tickets: %w", err)
	}

	// Maps for matching
	ghMap := make(map[int]Issue)
	for _, issue := range ghIssues {
		ghMap[issue.Number] = issue
	}

	localMap := make(map[int]*models.Ticket)
	for i := range localTickets {
		if localTickets[i].GitHubIssueNumber != nil {
			localMap[*localTickets[i].GitHubIssueNumber] = &localTickets[i]
		}
	}

	// 3. GitHub -> Local
	for _, ghIssue := range ghIssues {
		localTicket, exists := localMap[ghIssue.Number]
		if !exists {
			// Create local ticket from GH issue
			desc, meta := ParseIssueBody(ghIssue.Body)
			status := mapGHStateToStatus(ghIssue.State, ghIssue.Labels)

			req := models.CreateTicketRequest{
				ProjectID:          projectID,
				Title:              ghIssue.Title,
				Description:        desc,
				Status:             status,
				UserStory:          meta.UserStory,
				AcceptanceCriteria: meta.AcceptanceCriteria,
				TechnicalDetails:   meta.TechnicalDetails,
				TestingDetails:     meta.TestingDetails,
				IsDraft:            false,
			}
			newTicket, err := store.CreateTicket(req)
			if err != nil {
				continue
			}

			// Update with GH number and Rank
			store.UpdateTicket(newTicket.ID, models.UpdateTicketRequest{
				GitHubIssueNumber: &ghIssue.Number,
				LexoRank:          &meta.LexoRank,
			})
		} else {
			// Update local ticket if GH is newer
			if ghIssue.UpdatedAt.After(localTicket.UpdatedAt) {
				desc, meta := ParseIssueBody(ghIssue.Body)
				status := mapGHStateToStatus(ghIssue.State, ghIssue.Labels)

				store.UpdateTicket(localTicket.ID, models.UpdateTicketRequest{
					Title:              &ghIssue.Title,
					Description:        &desc,
					Status:             &status,
					UserStory:          &meta.UserStory,
					AcceptanceCriteria: &meta.AcceptanceCriteria,
					TechnicalDetails:   &meta.TechnicalDetails,
					TestingDetails:     &meta.TestingDetails,
					LexoRank:           &meta.LexoRank,
				})
			}
		}
	}

	// 4. Local -> GitHub
	for _, localTicket := range localTickets {
		if localTicket.GitHubIssueNumber == nil {
			// Create GH issue from local ticket
			body := FormatIssueBody(localTicket.Description, &localTicket)
			num, err := client.CreateIssue(ctx, owner, repo, localTicket.Title, body)
			if err != nil {
				continue
			}
			store.UpdateTicket(localTicket.ID, models.UpdateTicketRequest{
				GitHubIssueNumber: &num,
			})
		} else {
			// Update GH issue if local is newer
			ghIssue, exists := ghMap[*localTicket.GitHubIssueNumber]
			if exists && localTicket.UpdatedAt.After(ghIssue.UpdatedAt) {
				body := FormatIssueBody(localTicket.Description, &localTicket)
				state := mapStatusToGHState(localTicket.Status)
				client.UpdateIssue(ctx, owner, repo, ghIssue.Number, localTicket.Title, body, state)
			}
		}
	}

	store.UpdateProject(projectID, models.UpdateProjectRequest{
		// GitHubLastSynced: &now, // We'll need to update models to handle time.Time pointer in UpdateRequest if needed
	})

	// 5. Sync Deletions (Local -> GitHub)
	deleted, err := store.ListDeletedTickets(projectID)
	if err == nil {
		for _, t := range deleted {
			if t.GitHubIssueNumber != nil {
				// Close the issue on GitHub
				client.UpdateIssue(ctx, owner, repo, *t.GitHubIssueNumber, t.Title, FormatIssueBody(t.Description, &t), "closed")
			}
		}
		store.PurgeDeletedTickets(projectID)
	}

	return nil
}

func mapGHStateToStatus(state string, labels []string) string {
	if state == "CLOSED" {
		return "done"
	}
	for _, l := range labels {
		l = strings.ToLower(l)
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
