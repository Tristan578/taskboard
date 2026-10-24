package cli

import (
	"encoding/json"
	"fmt"
	"github.com/Tristan578/taskboard/internal/models"
	"github.com/Tristan578/taskboard/internal/workflow"
	"github.com/spf13/cobra"
	"net/http"
	"time"
)

func activityCommands() *cobra.Command {
	var a models.ActivityRequest
	c := &cobra.Command{Use: "activity [ticket-id]", Short: "Claim or heartbeat work with a progress summary", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}

		defer func() { _ = s.Close() }()
		t, err := s.ReportActivity(args[0], a)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(t)
	}}
	c.Flags().StringVar(&a.Agent, "agent", "", "agent name (Claude, Codex, Devin, etc.)")
	c.Flags().StringVar(&a.SessionID, "session", "", "stable session identifier")
	c.Flags().StringVar(&a.Summary, "summary", "", "current progress")
	return c
}
func completeCommand() *cobra.Command {
	var evidence string
	c := &cobra.Command{Use: "complete [ticket-id]", Short: "Verify merged PR or successful deployment before marking Done", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}

		defer func() { _ = s.Close() }()
		t, err := workflow.Complete(cmd.Context(), s, args[0], evidence)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(t)
	}}
	c.Flags().StringVar(&evidence, "evidence", "", "GitHub PR URL or deployment API URL")
	return c
}
func statusCommand() *cobra.Command {
	var statusPort int
	c := &cobra.Command{Use: "status", Short: "Discover local board, database and synchronization health", RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}

		defer func() { _ = s.Close() }()
		sync, err := s.GetSyncStatus()
		if err != nil {
			return err
		}
		url := fmt.Sprintf("http://127.0.0.1:%d", statusPort)
		client := &http.Client{Timeout: 2 * time.Second}
		res, err := client.Get(url + "/api/health")
		running := err == nil && res.StatusCode == 200
		if res != nil {
			_ = res.Body.Close()
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"boardUrl": url, "serverRunning": running, "sync": sync, "staleAfterMinutes": 15, "instruction": "Use MCP or CLI with the same --db; run player2-kanban start if the browser server is offline"})
	}}
	c.Flags().IntVar(&statusPort, "port", 3010, "board port")
	return c
}
