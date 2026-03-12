package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
	"gopkg.in/yaml.v3"
)

func indent(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = strings.Repeat(" ", n) + line
		}
	}
	return strings.Join(lines, "\n")
}

type Metadata struct {
	UserStory          string `yaml:"user_story,omitempty"`
	AcceptanceCriteria string `yaml:"acceptance_criteria,omitempty"`
	TechnicalDetails   string `yaml:"technical_details,omitempty"`
	TestingDetails     string `yaml:"testing_details,omitempty"`
}
func FormatIssueBody(description string, ticket *models.Ticket) string {
	meta := Metadata{
		UserStory:          ticket.UserStory,
		AcceptanceCriteria: ticket.AcceptanceCriteria,
		TechnicalDetails:   ticket.TechnicalDetails,
		TestingDetails:     ticket.TestingDetails,
	}

	data, err := yaml.Marshal(meta)
	if err != nil {
		return description
	}

	// Wrap in player2: key and indent
	yamlStr := "player2:\n" + indent(string(data), 2)
	return fmt.Sprintf("---\n%s---\n\n%s", yamlStr, description)
}

func ParseIssueBody(body string) (description string, meta Metadata) {
	if !strings.HasPrefix(body, "---") {
		return body, Metadata{}
	}

	parts := strings.SplitN(body, "---", 3)
	if len(parts) < 3 {
		return body, Metadata{}
	}

	yamlContent := parts[1]
	description = strings.TrimSpace(parts[2])

	var root struct {
		Player2 Metadata `yaml:"player2"`
	}

	err := yaml.Unmarshal([]byte(yamlContent), &root)
	if err != nil {
		return body, Metadata{}
	}

	return description, root.Player2
}

func SyncProject(ctx context.Context, client *Client, store interface {
	GetProject(id string) (*models.Project, error)
	ListTickets(filter models.TicketFilter) ([]models.Ticket, error)
	GetTicket(id string) (*models.Ticket, error)
	UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error)
	CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error)
	UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error)
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
			}
			newTicket, err := store.CreateTicket(req)
			if err != nil {
				continue
			}

			// Update with GH number (CreateTicket doesn't set it)
			store.UpdateTicket(newTicket.ID, models.UpdateTicketRequest{
				GitHubIssueNumber: &ghIssue.Number,
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

	now := time.Now()
	store.UpdateProject(projectID, models.UpdateProjectRequest{
		// GitHubLastSynced: &now, // We'll need to update models to handle time.Time pointer in UpdateRequest if needed
	})
	_ = now

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
