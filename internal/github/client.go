package github

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Client struct {
	rest *github.Client
	gql  *githubv4.Client
}

func NewClient(ctx context.Context, token string) *Client {
	return NewClientWithURLs(ctx, token, "", "")
}

func NewClientWithURLs(ctx context.Context, token string, restURL, gqlURL string) *Client {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	restClient := github.NewClient(tc)
	if restURL != "" {
		if !strings.HasSuffix(restURL, "/") {
			restURL += "/"
		}
		parsed, _ := url.Parse(restURL)
		restClient.BaseURL = parsed
	}

	gqlClient := githubv4.NewClient(tc)
	if gqlURL != "" {
		// Use a local URL for GraphQL if provided
		gqlClient = githubv4.NewEnterpriseClient(gqlURL, tc)
	}

	return &Client{
		rest: restClient,
		gql:  gqlClient,
	}
}

func (c *Client) REST() *github.Client {
	return c.rest
}

func (c *Client) GQL() *githubv4.Client {
	return c.gql
}

type Issue struct {
	Number    int
	Title     string
	Body      string
	State     string // OPEN or CLOSED
	Labels    []string
	UpdatedAt time.Time
}

func (c *Client) GetIssues(ctx context.Context, owner, repo string) ([]Issue, error) {
	var query struct {
		Repository struct {
			Issues struct {
				Nodes []struct {
					Number int
					Title  string
					Body   string
					State  string
					Labels struct {
						Nodes []struct {
							Name string
						}
					} `graphql:"labels(first: 10)"`
					UpdatedAt time.Time
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"issues(first: 100, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(repo),
		"cursor": (*githubv4.String)(nil),
	}

	var allIssues []Issue
	for {
		err := c.gql.Query(ctx, &query, variables)
		if err != nil {
			return nil, err
		}

		for _, node := range query.Repository.Issues.Nodes {
			issue := Issue{
				Number:    node.Number,
				Title:     node.Title,
				Body:      node.Body,
				State:     node.State,
				UpdatedAt: node.UpdatedAt,
			}
			for _, label := range node.Labels.Nodes {
				issue.Labels = append(issue.Labels, label.Name)
			}
			allIssues = append(allIssues, issue)
		}

		if !query.Repository.Issues.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Repository.Issues.PageInfo.EndCursor)
	}

	return allIssues, nil
}

func (c *Client) CreateIssue(ctx context.Context, owner, repo string, title, body string) (int, error) {
	issue, _, err := c.rest.Issues.Create(ctx, owner, repo, &github.IssueRequest{
		Title: &title,
		Body:  &body,
	})
	if err != nil {
		return 0, err
	}
	return issue.GetNumber(), nil
}

func (c *Client) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body, state string) error {
	req := &github.IssueRequest{
		Title: &title,
		Body:  &body,
		State: &state,
	}
	_, _, err := c.rest.Issues.Edit(ctx, owner, repo, number, req)
	return err
}

func ParseRepo(repoStr string) (owner, repo string, err error) {
	repoStr = strings.TrimPrefix(repoStr, "https://github.com/")
	repoStr = strings.TrimPrefix(repoStr, "git@github.com:")
	repoStr = strings.TrimSuffix(repoStr, ".git")
	parts := strings.Split(repoStr, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repo format: %s", repoStr)
	}
	return parts[0], parts[1], nil
}
