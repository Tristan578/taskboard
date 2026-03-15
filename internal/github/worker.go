package github

import (
	"context"
	"log"
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
	ListDeletedTickets(projectID string) ([]models.Ticket, error)
	PurgeDeletedTickets(projectID string) error
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
		log.Printf("Worker error fetching jobs: %v", err)
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
	
	status := "completed"
	lastError := ""
	attempts := job.Attempts + 1

	if err != nil {
		status = "failed"
		lastError = err.Error()
		log.Printf("Job %s failed: %v", job.ID, err)
	}

	if err := w.store.UpdateSyncJobStatus(job.ID, status, attempts, lastError); err != nil {
		log.Printf("Worker failed to update status for job %s: %v", job.ID, err)
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
