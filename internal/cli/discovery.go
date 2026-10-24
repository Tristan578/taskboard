package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
)

type boardProbe struct {
	Running         bool
	Detected        bool
	DatabaseMatches bool
	DatabasePath    string
	PID             int
	Instruction     string
}

func configuredDatabasePath() (string, error) {
	if dbPath != "" {
		return dbPath, nil
	}
	return db.DefaultDBPath()
}

// Probe the requested local instance, without following redirects or trusting an
// HTTP 200 alone. File identity also handles symlinks and Windows path casing.
func probeBoard(ctx context.Context, address, database string, timeout time.Duration) boardProbe {
	result := boardProbe{Instruction: "Player2 is not reachable on this port. Open it using the same --db and --port."}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address+"/api/health", nil)
	if err != nil {
		return result
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return result
	}
	defer response.Body.Close()
	var health struct {
		Name         string `json:"name"`
		Status       string `json:"status"`
		PID          int    `json:"pid"`
		DatabasePath string `json:"databasePath"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<10)).Decode(&health); err != nil || health.Name != "player2-kanban" || health.PID <= 0 {
		result.Instruction = "This port did not identify a Player2 instance. Check the port and restart Player2."
		return result
	}
	result.Detected, result.PID, result.DatabasePath = true, health.PID, health.DatabasePath
	if response.StatusCode != http.StatusOK || health.Status != "ok" {
		result.Instruction = "Player2 responded but is not healthy. Check its data folder and server log before continuing."
		return result
	}
	if health.DatabasePath == "" {
		result.Instruction = "This server cannot identify its database. Restart it with the updated Player2 before connecting an agent."
		return result
	}
	expected, expectedErr := os.Stat(database)
	actual, actualErr := os.Stat(health.DatabasePath)
	result.DatabaseMatches = expectedErr == nil && actualErr == nil && os.SameFile(expected, actual)
	if !result.DatabaseMatches {
		result.Instruction = "The browser is using a different or inaccessible database. Match your agent's --db to the browser connection settings under Help."
		return result
	}
	result.Running = true
	result.Instruction = "Player2 is running with this database. Use these same --db settings for your agent."
	return result
}
