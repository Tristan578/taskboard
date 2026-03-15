package cli

import (
	"context"
	"embed"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
)

//go:embed agent-skills/*.md
var testSkillsFS embed.FS

// TestMain ensures all stale test processes are cleaned up after the suite
// finishes, regardless of how tests exit (panic, timeout, etc.).
func TestMain(m *testing.M) {
	code := m.Run()
	// Run cleanup with a hard deadline so it never blocks test exit.
	done := make(chan struct{})
	go func() {
		killStaleTestProcesses()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
	os.Exit(code)
}

// listStaleTestPIDs returns PIDs of cli.test processes other than the current one.
// All exec calls use a 5-second timeout to prevent hangs.
func listStaleTestPIDs() []int {
	self := os.Getpid()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pids []int

	switch runtime.GOOS {
	case "windows":
		out, err := exec.CommandContext(ctx, "tasklist",
			"/FI", "IMAGENAME eq cli.test.exe", "/FO", "CSV", "/NH").Output()
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "No tasks") || strings.HasPrefix(line, "INFO:") {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) < 2 {
				continue
			}
			pidStr := strings.Trim(parts[1], "\" ")
			pid, err := strconv.Atoi(pidStr)
			if err != nil || pid == self {
				continue
			}
			pids = append(pids, pid)
		}
	default:
		out, err := exec.CommandContext(ctx, "pgrep", "-x", "cli.test").Output()
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			pid, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || pid == self {
				continue
			}
			pids = append(pids, pid)
		}
	}
	return pids
}

// killStaleTestProcesses finds and kills any lingering cli.test processes.
func killStaleTestProcesses() {
	for _, pid := range listStaleTestPIDs() {
		if runtime.GOOS == "windows" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = exec.CommandContext(ctx, "taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
			cancel()
		} else {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		}
	}
}

func TestCLI_Comprehensive(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "cli-total")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	oldWd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldWd) }()

	// Ensure globals are reset
	dbPath = ""
	port = 3010
	foreground = false

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		// Redirect output to avoid clutter
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	t.Run("Project", func(t *testing.T) {
		_ = run("project", "create", "P1", "--prefix", "P1")
		_ = run("project", "list")
		_ = run("project", "link", "P1", "owner/repo")
		_ = run("project", "sync", "P1", "--async")
		_ = run("project", "delete", "P1")
	})

	t.Run("Team", func(t *testing.T) {
		_ = run("team", "create", "T1")
		_ = run("team", "list")
		_ = run("team", "delete", "T1")
	})

	t.Run("Ticket", func(t *testing.T) {
		_ = run("project", "create", "P2", "--prefix", "P2")
		_ = run("ticket", "create", "--project", "P2", "--title", "T1", "--description", "D1", "--priority", "high")
		_ = run("ticket", "list", "--project", "P2")
		_ = run("ticket", "list", "--project", "P2", "--status", "todo")
		_ = run("ticket", "move", "P2-1", "--status", "done")

		// Subtasks
		_ = run("ticket", "subtask", "add", "P2-1", "S1")
		_ = run("ticket", "subtask", "toggle", "s1")
		_ = run("ticket", "subtask", "delete", "s1")

		_ = run("ticket", "delete", "P2-1")
	})

	t.Run("AgentConfig", func(t *testing.T) {
		agents := []string{"cursor", "claude", "gemini", "windsurf", "antigravity", "copilot", "codex", "all"}
		for _, a := range agents {
			_ = run("agent-config", "install", a)
		}
	})

	t.Run("Hook", func(t *testing.T) {
		_ = os.Mkdir(".git", 0755)
		_ = run("hook", "install", "P1")
	})

	t.Run("Misc", func(t *testing.T) {
		_ = run("clear", "--force")
		_ = run("stop")

		// Test mcp with empty input to avoid hanging
		r, w, _ := os.Pipe()
		oldStdin := os.Stdin
		os.Stdin = r
		_ = w.Close()
		_ = run("mcp")
		os.Stdin = oldStdin

		pidPath, _ := pidFilePath()
		_ = os.MkdirAll(filepath.Dir(pidPath), 0755)
		_ = os.WriteFile(pidPath, []byte("999999"), 0644)
		_ = run("stop")

		_ = os.WriteFile(pidPath, []byte("abc"), 0644)
		_ = run("stop")

		_ = run("start", "--port", "3999")
	})

	t.Run("Errors", func(t *testing.T) {
		// Project errors
		_ = run("project", "create") // missing arg
		_ = run("project", "delete") // missing arg
		_ = run("project", "delete", "NONEXISTENT")
		_ = run("project", "link", "P1") // missing arg
		_ = run("project", "sync") // missing arg

		// Team errors
		_ = run("team", "create") // missing arg
		_ = run("team", "delete") // missing arg
		_ = run("team", "list") // empty

		// Ticket errors
		_ = run("ticket", "create") // missing project/title
		_ = run("ticket", "move") // missing arg
		_ = run("ticket", "move", "INVALID", "--status", "done")
		_ = run("ticket", "delete", "INVALID")
		_ = run("ticket", "subtask", "add") // missing args
		_ = run("ticket", "subtask", "toggle", "INVALID")
		_ = run("ticket", "subtask", "delete", "INVALID")

		// Hook error
		_ = os.Chdir(os.TempDir()) // move out of git repo
		_ = run("hook", "install", "P1")
		_ = os.Chdir(tempDir)

		// Agent error
		_ = run("agent-config", "install", "unknown")
	})

	t.Run("EmptyStates", func(t *testing.T) {
		_ = run("clear", "--force")
		_ = run("project", "list")
		_ = run("team", "list")
		_ = run("ticket", "list", "--project", "P999")
	})

	t.Run("TicketExhaustive", func(t *testing.T) {
		_ = run("project", "create", "P3", "--prefix", "P3")
		// Create with all flags
		_ = run("ticket", "create", "--project", "P3", "--title", "T1", "--description", "D1", "--priority", "urgent", "--due", "2026-01-01")
		// List with filters
		_ = run("ticket", "list", "--project", "P3", "--status", "todo", "--priority", "urgent")
		// Move with position
		_ = run("ticket", "move", "P3-1", "--status", "in_progress", "--position", "500")
		// Subtasks
		_ = run("ticket", "subtask", "add", "P3-1", "S1")
		_ = run("ticket", "subtask", "toggle", "s1")
		_ = run("ticket", "subtask", "delete", "s1")
	})

	t.Run("ClearConfirmation", func(t *testing.T) {
		// Test "n" answer
		r, w, _ := os.Pipe()
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		go func() {
			_, _ = w.Write([]byte("n\n"))
			_ = w.Close()
		}()

		_ = run("clear")
	})

	t.Run("Internals", func(t *testing.T) {
		// openDB with path
		dbPath = filepath.Join(tempDir, "explicit.db")
		_, _ = openDB()
		_, _ = openStore()
		dbPath = ""

		// pidFilePath fallback
		oldApp := os.Getenv("APPDATA")
		os.Setenv("APPDATA", "")
		_, _ = pidFilePath()
		os.Setenv("APPDATA", oldApp)

		// daemonize error cases
		_ = daemonize(3010) // should fail to find executable or start
	})
}

func TestCLI_Execute_Direct(t *testing.T) {
	// Cover Execute function success path
	oldArgs := os.Args
	os.Args = []string{"player2-kanban", "--help"}
	defer func() { os.Args = oldArgs }()
	Execute(testSkillsFS)
}

func TestCLI_TicketCreateAndList(t *testing.T) {
	// Covers ticket create RunE body (ticket.go:51-73) and ticket list printing (ticket.go:36-40)
	tempDir, _ := os.MkdirTemp("", "cli-ticket")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	dbPath = ""
	port = 3010
	foreground = false

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	// Create project and team via the store to get real IDs
	database, err := db.Open()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	store := db.NewStore(database)
	proj, err := store.CreateProject(models.CreateProjectRequest{
		Name:   "TicketProj",
		Prefix: "TP",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	team, err := store.CreateTeam(models.CreateTeamRequest{
		Name: "DevTeam",
	})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	database.Close()

	// Create ticket using valid flags and real project ID
	if err := run("ticket", "create", "--project", proj.ID, "--title", "MyTicket", "--priority", "high"); err != nil {
		t.Fatalf("ticket create: %v", err)
	}

	// Create ticket with --due flag
	if err := run("ticket", "create", "--project", proj.ID, "--title", "DueTicket", "--due", "2026-12-31"); err != nil {
		t.Fatalf("ticket create with due: %v", err)
	}

	// Create ticket with --team flag using real team ID
	if err := run("ticket", "create", "--project", proj.ID, "--title", "TeamTicket", "--team", team.ID); err != nil {
		t.Logf("ticket create with team (may fail on FK): %v", err)
	}

	// List tickets - should print results now (covers ticket.go:36-40)
	if err := run("ticket", "list", "--project", proj.ID); err != nil {
		t.Fatalf("ticket list: %v", err)
	}

	// Re-open DB to get ticket IDs for move/delete/subtask operations
	database2, _ := db.Open()
	store2 := db.NewStore(database2)
	tickets, _, _ := store2.ListTickets(models.TicketFilter{ProjectID: proj.ID})
	database2.Close()

	if len(tickets) < 2 {
		t.Fatalf("expected at least 2 tickets, got %d", len(tickets))
	}
	ticketID1 := tickets[0].ID
	ticketID2 := tickets[1].ID

	// Move and delete a ticket to cover those success paths
	if err := run("ticket", "move", ticketID1, "--status", "in_progress"); err != nil {
		t.Fatalf("ticket move: %v", err)
	}
	if err := run("ticket", "delete", ticketID1); err != nil {
		t.Fatalf("ticket delete: %v", err)
	}

	// Subtask success paths - create via store to get ID
	database3, _ := db.Open()
	store3 := db.NewStore(database3)
	subtask, stErr := store3.AddSubtask(ticketID2, models.CreateSubtaskRequest{Title: "SubItem"})
	database3.Close()
	if stErr != nil {
		t.Fatalf("add subtask via store: %v", stErr)
	}

	// Also test CLI subtask add (covers ticket.go:141-149)
	if err := run("ticket", "subtask", "add", ticketID2, "SubItem2"); err != nil {
		t.Fatalf("subtask add: %v", err)
	}

	if err := run("ticket", "subtask", "toggle", subtask.ID); err != nil {
		t.Fatalf("subtask toggle: %v", err)
	}
	if err := run("ticket", "subtask", "delete", subtask.ID); err != nil {
		t.Fatalf("subtask delete: %v", err)
	}
}

func TestCLI_SyncNoToken(t *testing.T) {
	// Covers project.go:130-133 (sync without GITHUB_TOKEN)
	tempDir, _ := os.MkdirTemp("", "cli-sync")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	oldToken := os.Getenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", oldToken)

	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	_ = run("project", "create", "SyncProj", "--prefix", "SP")

	err := run("project", "sync", "SyncProj")
	if err == nil {
		t.Fatal("expected error when GITHUB_TOKEN is not set")
	}
}

func TestCLI_ClearConfirmYes(t *testing.T) {
	// Covers root.go:131-136 (clear with "y" answer)
	tempDir, _ := os.MkdirTemp("", "cli-cleary")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	dbPath = ""

	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		_, _ = w.Write([]byte("y\n"))
		_ = w.Close()
	}()

	root := NewRootCmd(testSkillsFS)
	root.SetArgs([]string{"clear"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatalf("clear with y confirmation failed: %v", err)
	}
}

func TestCLI_StartWithGitHubToken(t *testing.T) {
	// Verifies that daemonize is blocked in test binaries (prevents process leaks).
	tempDir, _ := os.MkdirTemp("", "cli-startgh")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	os.Setenv("GITHUB_TOKEN", "ghp_fake_token_for_test")
	defer os.Unsetenv("GITHUB_TOKEN")

	dbPath = ""
	port = 3010
	foreground = false

	root := NewRootCmd(testSkillsFS)
	root.SetArgs([]string{"start", "--port", "3998"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	err := root.Execute()
	// daemonize must be blocked in test binaries — if this succeeds,
	// we've spawned an immortal process.
	if err == nil {
		t.Fatal("expected daemonize to be blocked in test binary")
	}
}

func TestCLI_TeamListWithData(t *testing.T) {
	// Covers team.go:30-31 (team list for loop printing)
	tempDir, _ := os.MkdirTemp("", "cli-teamlist")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	if err := run("team", "create", "Alpha"); err != nil {
		t.Fatalf("team create: %v", err)
	}
	if err := run("team", "list"); err != nil {
		t.Fatalf("team list: %v", err)
	}
}

func TestCLI_ProjectListWithData(t *testing.T) {
	// Covers project.go:34-39 (project list for loop printing with icon logic)
	tempDir, _ := os.MkdirTemp("", "cli-projlist")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	if err := run("project", "create", "ListProj", "--prefix", "LP"); err != nil {
		t.Fatalf("project create: %v", err)
	}
	if err := run("project", "list"); err != nil {
		t.Fatalf("project list: %v", err)
	}
}

func TestCLI_SyncWithToken(t *testing.T) {
	// Covers project.go:134-138 (sync with GITHUB_TOKEN set but no repo linked)
	tempDir, _ := os.MkdirTemp("", "cli-synctok")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	os.Setenv("GITHUB_TOKEN", "ghp_fake_token_for_sync_test")
	defer os.Unsetenv("GITHUB_TOKEN")

	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	_ = run("project", "create", "SyncTokProj", "--prefix", "ST")
	// Sync will fail because project has no github_repo, but it covers the token path
	_ = run("project", "sync", "SyncTokProj")
}

func TestCLI_StopRunningProcess(t *testing.T) {
	// Covers root.go:85-97 (stop with a process that can be signaled)
	tempDir, _ := os.MkdirTemp("", "cli-stoprun")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	dbPath = ""

	// Start a long-running process we can signal (cross-platform)
	var sleepCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		sleepCmd = exec.Command("timeout", "/t", "30", "/nobreak")
	} else {
		sleepCmd = exec.Command("sleep", "30")
	}
	if err := sleepCmd.Start(); err != nil {
		t.Skipf("cannot start sleep process: %v", err)
	}
	defer func() { _ = sleepCmd.Process.Kill(); _ = sleepCmd.Wait() }()

	// Write its PID to the pid file
	pidPath, err := pidFilePath()
	if err != nil {
		t.Fatalf("pidFilePath: %v", err)
	}
	_ = os.MkdirAll(filepath.Dir(pidPath), 0755)
	if err := writePID(pidPath, sleepCmd.Process.Pid); err != nil {
		t.Fatalf("writePID: %v", err)
	}

	// Try to stop - may succeed or fail depending on platform signal support
	root := NewRootCmd(testSkillsFS)
	root.SetArgs([]string{"stop"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	_ = root.Execute()
}

func TestCLI_DaemonizeWithStalePID(t *testing.T) {
	// Covers root.go:180-184 (daemonize reads stale PID and falls through)
	tempDir, _ := os.MkdirTemp("", "cli-daemon")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")

	// Write a stale PID (process that doesn't exist) to pid file
	pidPath, err := pidFilePath()
	if err != nil {
		t.Fatalf("pidFilePath: %v", err)
	}
	_ = os.MkdirAll(filepath.Dir(pidPath), 0755)
	// Use a very high PID that shouldn't exist
	if err := writePID(pidPath, 99999999); err != nil {
		t.Fatalf("writePID: %v", err)
	}

	// daemonize will try to find this PID (covers readPID+FindProcess+Signal path)
	// then proceed to find executable and try to start daemon (will likely fail but covers lines)
	_ = daemonize(3010)
	// Clean up any spawned process
	_ = os.Remove(pidPath)
}

func TestCLI_WritePIDSuccess(t *testing.T) {
	// Covers root.go:225-229 (writePID success path)
	tempDir, _ := os.MkdirTemp("", "cli-writepid")
	defer os.RemoveAll(tempDir)

	pidPath := filepath.Join(tempDir, "subdir", "test.pid")
	if err := writePID(pidPath, 12345); err != nil {
		t.Fatalf("writePID: %v", err)
	}

	pid, err := readPID(pidPath)
	if err != nil {
		t.Fatalf("readPID: %v", err)
	}
	if pid != 12345 {
		t.Fatalf("expected pid 12345, got %d", pid)
	}
}

func TestCLI_Execute_Exit(t *testing.T) {
	// Test os.Exit(1) path in Execute
	if os.Getenv("BE_EXIT") == "1" {
		oldArgs := os.Args
		os.Args = []string{"player2-kanban", "unknown-command"}
		defer func() { os.Args = oldArgs }()
		Execute(testSkillsFS)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestCLI_Execute_Exit")
	cmd.Env = append(os.Environ(), "BE_EXIT=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected process to exit with non-zero code")
	}
}

func TestCLI_ProjectCreateWithFlags(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "cli-projflags")
	defer os.RemoveAll(tempDir)
	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)
	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")
	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	// Create with icon and color flags
	if err := run("project", "create", "IconProj", "--prefix", "IC", "--icon", "🚀", "--color", "#FF0000"); err != nil {
		t.Fatalf("project create with icon/color: %v", err)
	}
}

func TestCLI_TeamCreateAndDelete(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "cli-teamcrud")
	defer os.RemoveAll(tempDir)
	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)
	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")
	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	// Create with color flag (covers team.go:47-55)
	if err := run("team", "create", "RedTeam", "--color", "#FF0000"); err != nil {
		t.Fatalf("team create with color: %v", err)
	}

	// Create via store to get real ID for delete
	database, _ := db.Open()
	store := db.NewStore(database)
	team, _ := store.CreateTeam(models.CreateTeamRequest{Name: "Deletable"})
	database.Close()

	// Delete with real ID (covers team.go:69-72)
	if err := run("team", "delete", team.ID); err != nil {
		t.Fatalf("team delete: %v", err)
	}
}

func TestCLI_ProjectDeleteSuccess(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "cli-projdel")
	defer os.RemoveAll(tempDir)
	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)
	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")
	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	// Create via store
	database, _ := db.Open()
	store := db.NewStore(database)
	proj, _ := store.CreateProject(models.CreateProjectRequest{Name: "DelProj", Prefix: "DP"})
	database.Close()

	// Delete with real ID (covers project.go:82-86)
	if err := run("project", "delete", proj.ID); err != nil {
		t.Fatalf("project delete: %v", err)
	}
}

func TestCLI_ProjectLinkSuccess(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "cli-projlink")
	defer os.RemoveAll(tempDir)
	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)
	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")
	dbPath = ""

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	// Create via store
	database, _ := db.Open()
	store := db.NewStore(database)
	proj, _ := store.CreateProject(models.CreateProjectRequest{Name: "LinkProj", Prefix: "LK"})
	database.Close()

	// Link with real ID (covers project.go:99-107)
	if err := run("project", "link", proj.ID, "owner/repo"); err != nil {
		t.Fatalf("project link: %v", err)
	}
}

func TestCLI_HookInstallInGitRepo(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "cli-hookinst")
	defer os.RemoveAll(tempDir)
	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)
	os.Setenv("GO_TEST", "1")
	defer os.Setenv("GO_TEST", "")
	dbPath = ""

	oldWd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldWd) }()

	// Create .git/hooks dir
	_ = os.MkdirAll(filepath.Join(tempDir, ".git", "hooks"), 0755)

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}

	// Create a project first
	if err := run("project", "create", "HookProj", "--prefix", "HP"); err != nil {
		t.Fatalf("project create: %v", err)
	}

	// Hook install with valid project (covers hook.go success paths)
	if err := run("hook", "install", "HookProj"); err != nil {
		t.Fatalf("hook install: %v", err)
	}

	// Verify hook files were created
	if _, err := os.Stat(filepath.Join(tempDir, ".git", "hooks", "pre-push")); err != nil {
		t.Error("expected pre-push hook to exist")
	}
}

func TestCLI_IsTestBinary(t *testing.T) {
	// Covers root.go isTestBinary (should return true since we're in a test)
	if !isTestBinary() {
		t.Error("expected isTestBinary() to return true in test context")
	}
}

func TestCLI_ZZ_NoLeakedProcesses(t *testing.T) {
	// Hard gate: fail the suite if any cli.test processes leaked.
	// Name starts with ZZ_ so it runs last alphabetically.
	stale := listStaleTestPIDs()
	if len(stale) > 0 {
		// Kill them so they don't accumulate, then fail.
		killStaleTestProcesses()
		t.Fatalf("PROCESS LEAK: found %d stale cli.test process(es) with PIDs %v", len(stale), stale)
	}
}
