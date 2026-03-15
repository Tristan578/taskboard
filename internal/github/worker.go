package github

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/Tristan578/taskboard/internal/models"
)

type Store interface {
	GetProject(id string) (*models.Project, error)
	ListTickets(filter models.TicketFilter) ([]models.Ticket, int, error)
	GetTicket(id string) (*models.Ticket, error)
	UpdateTicket(id string, req models.UpdateTicketRequest) (*models.Ticket, error)
	CreateTicket(req models.CreateTicketRequest) (*models.Ticket, error)
	UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error)
	GetPendingSyncJobs() ([]models.SyncJob, error)
	UpdateSyncJobStatus(id, status string, attempts int, lastError string) error
	UpdateSyncJobRetry(id string, attempts int, lastError string, nextRetryAt time.Time) error
	ListDeletedTickets(projectID string) ([]models.Ticket, error)
	PurgeDeletedTickets(projectID string) error
}

// CalculateNextRetry computes the next retry time with exponential backoff.
// Base 30s, factor 2^attempts, jitter +/-25%, cap 1 hour.
func CalculateNextRetry(attempts int) time.Time {
	base := 30.0 * math.Pow(2, float64(attempts))
	if base > 3600 {
		base = 3600
	}
	// #nosec G404 -- jitter for backoff timing, not security-sensitive
	jitter := base * 0.25 * (2*rand.Float64() - 1) // +/-25%
	delay := time.Duration(base+jitter) * time.Second
	return time.Now().Add(delay)
}


type Worker struct {
	store  Store
	client *Client
}

func NewWorker(store Store, client *Client) *Worker {
	return &Worker{
		store:  store,
		client: client,
	}
}


func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processJobs(ctx)
		}
	}
}

func (w *Worker) processJobs(ctx context.Context) {
	jobs, err := w.store.GetPendingSyncJobs()
	if err != nil {
		slog.Error("failed to fetch sync jobs", "error", err)
		return
	}

	for _, job := range jobs {
		w.processJob(ctx, job)
		// Small delay between jobs to respect rate limits
		time.Sleep(500 * time.Millisecond)
	}
}

func (w *Worker) processJob(ctx context.Context, job models.SyncJob) {
	err := w.executeAction(ctx, job)

	attempts := job.Attempts + 1

	if err != nil {
		slog.Error("sync job failed", "jobId", job.ID, "action", job.Action, "projectId", job.ProjectID, "attempts", attempts, "error", err)

		// Check for rate limit errors — use the reset time directly
		var rlErr *RateLimitError
		if errors.As(err, &rlErr) {
			if updateErr := w.store.UpdateSyncJobRetry(job.ID, attempts, err.Error(), rlErr.ResetAt); updateErr != nil {
				slog.Error("failed to update job retry status", "jobId", job.ID, "error", updateErr)
			}
			return
		}

		// Standard exponential backoff
		nextRetry := CalculateNextRetry(attempts)
		if updateErr := w.store.UpdateSyncJobRetry(job.ID, attempts, err.Error(), nextRetry); updateErr != nil {
			slog.Error("failed to update job retry status", "jobId", job.ID, "error", updateErr)
		}
		return
	}

	slog.Info("sync job completed", "jobId", job.ID, "action", job.Action, "projectId", job.ProjectID, "attempts", attempts)
	if err := w.store.UpdateSyncJobStatus(job.ID, "completed", attempts, ""); err != nil {
		slog.Error("failed to update job completion status", "jobId", job.ID, "error", err)
	}
}

func (w *Worker) executeAction(ctx context.Context, job models.SyncJob) error {
	switch job.Action {
	case "full_sync":
		return SyncProject(ctx, w.client, w.store, job.ProjectID)
	// We can add more granular actions like 'update_issue' later
	default:
		return SyncProject(ctx, w.client, w.store, job.ProjectID)
	}
}
