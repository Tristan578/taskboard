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
		_ = run("mcp")

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
		_ = run("project", "link", "P1") // missing arg
		_ = run("project", "sync") // missing arg

		// Ticket errors
		_ = run("ticket", "create") // missing project/title
		_ = run("ticket", "move") // missing arg
		_ = run("ticket", "subtask", "add") // missing args

		// Hook error
		_ = os.Chdir(os.TempDir()) // move out of git repo
		_ = run("hook", "install", "P1")

		// Agent error
		_ = run("agent-config", "install", "unknown")
	})
}

func TestCLI_Execute_Direct(t *testing.T) {
	Execute(testSkillsFS)
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
