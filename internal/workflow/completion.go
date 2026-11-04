package workflow

import (
	"context"
	"fmt"
	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/github"
	"github.com/Tristan578/taskboard/internal/models"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Complete verifies a GitHub merge or successful deployment in the linked repo.
// The caller explicitly associates that evidence with this ticket.
func Complete(ctx context.Context, store *db.Store, id, evidence string, expected ...int) (*models.Ticket, error) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required to verify completion")
	}
	t, err := store.GetTicket(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("ticket not found")
	}
	if len(expected) > 0 && t.Revision != expected[0] {
		return nil, &db.ConflictError{}
	}
	p, err := store.GetProject(t.ProjectID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.GitHubRepo == "" {
		return nil, fmt.Errorf("link a GitHub repository before verifying completion")
	}
	owner, repo, err := github.ParseRepo(p.GitHubRepo)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	c := github.NewClient(ctx, os.Getenv("GITHUB_TOKEN"))
	if err := VerifyEvidence(ctx, c, owner, repo, evidence); err != nil {
		return nil, err
	}
	return store.CompleteVerified(id, evidence, t.Revision)
}
func VerifyEvidence(ctx context.Context, c *github.Client, owner, repo, evidence string) error {
	u, err := url.Parse(evidence)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("provide a canonical HTTPS GitHub pull request or deployment API URL")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if u.Host == "github.com" && len(parts) == 4 && parts[0] == owner && parts[1] == repo && parts[2] == "pull" {
		n, err := strconv.Atoi(parts[3])
		if err != nil || n < 1 {
			return fmt.Errorf("invalid pull request number")
		}
		pr, _, err := c.REST().PullRequests.Get(ctx, owner, repo, n)
		if err != nil {
			return fmt.Errorf("verify pull request: %w", err)
		}
		if !pr.GetMerged() || pr.MergedAt == nil {
			return fmt.Errorf("pull request is not merged")
		}
		return nil
	}
	if u.Host == "api.github.com" && len(parts) == 5 && parts[0] == "repos" && parts[1] == owner && parts[2] == repo && parts[3] == "deployments" {
		n, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil || n < 1 {
			return fmt.Errorf("invalid deployment ID")
		}
		statuses, _, err := c.REST().Repositories.ListDeploymentStatuses(ctx, owner, repo, n, nil)
		if err != nil {
			return fmt.Errorf("verify deployment: %w", err)
		}
		if len(statuses) == 0 || statuses[0].GetState() != "success" {
			return fmt.Errorf("latest deployment status is not success")
		}
		return nil
	}
	return fmt.Errorf("evidence must belong to linked repository %s/%s", owner, repo)
}
