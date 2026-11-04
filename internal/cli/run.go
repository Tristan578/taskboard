package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
	"github.com/spf13/cobra"
)

func runCommand() *cobra.Command {
	var ticketID, agent string
	c := &cobra.Command{Use: "run --ticket ID --agent NAME -- COMMAND [ARGS...]", Short: "Run an agent with automatic board tracking and heartbeats", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}

		defer func() { _ = store.Close() }()
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		return runTracked(ctx, store, ticketID, agent, args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), time.Minute)
	}}
	c.Flags().StringVar(&ticketID, "ticket", "", "existing ticket ID")
	c.Flags().StringVar(&agent, "agent", "", "agent/provider name")
	_ = c.MarkFlagRequired("ticket")
	_ = c.MarkFlagRequired("agent")
	return c
}

func runTracked(ctx context.Context, store *db.Store, id, agent string, args []string, in io.Reader, out, errOut io.Writer, interval time.Duration) error {
	if len(args) == 0 || interval <= 0 {
		return fmt.Errorf("command and positive heartbeat interval are required")
	}
	session := fmt.Sprintf("process-%d-%d", os.Getpid(), time.Now().UnixNano())
	activity := models.ActivityRequest{Agent: agent, SessionID: session, Summary: "Starting agent process"}
	if _, err := store.ReportActivity(id, activity); err != nil {
		return err
	}
	if _, err := store.MoveTicket(id, models.MoveTicketRequest{Status: "in_progress"}); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// #nosec G204 -- this command and arguments are explicitly supplied by the CLI user.
	child := exec.CommandContext(ctx, args[0], args[1:]...)
	child.Stdin = in
	child.Stdout = out
	child.Stderr = errOut
	finish := func(runErr error) error {
		ticket, err := store.GetTicket(id)
		if err != nil {
			return err
		}
		if ticket == nil {
			return fmt.Errorf("tracked ticket was removed")
		}
		// Respect explicit completion and status changes made while the process ran.
		if ticket.Status == "done" {
			return runErr
		}
		status := "review"
		activity.Summary = "Agent process exited successfully; review the work and tests before merge"
		if runErr != nil {
			status = "blocked"
			activity.Summary = "Agent process failed or was interrupted: " + runErr.Error()
		}
		if _, err = store.ReportActivity(id, activity); err != nil {
			return err
		}
		if ticket.Status == "in_progress" {
			if _, err = store.MoveTicket(id, models.MoveTicketRequest{Status: status}); err != nil {
				return err
			}
		}
		return runErr
	}
	if err := child.Start(); err != nil {
		return finish(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- child.Wait() }()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case err := <-finished:
			return finish(err)
		case <-ticker.C:
			activity.Summary = "Agent process is running; process heartbeat (not a claim of task progress)"
			if _, err := store.ReportActivity(id, activity); err != nil {
				cancel()
				<-finished
				return fmt.Errorf("tracking failed; agent stopped: %w", err)
			}
		}
	}
}
