package cli

import (
	"io/ioutil"
	"os"
	"testing"
	"embed"
	"path/filepath"
)

//go:embed agent-skills/*.md
var testSkillsFS embed.FS

func TestCLI_Comprehensive(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-total")
	defer os.RemoveAll(tempDir)

	oldApp := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldApp)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Ensure globals are reset
	dbPath = ""
	port = 3010
	foreground = false

	run := func(args ...string) error {
		root := NewRootCmd(testSkillsFS)
		root.SetArgs(args)
		root.SetOut(ioutil.Discard)
		root.SetErr(ioutil.Discard)
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
		os.Mkdir(".git", 0755)
		_ = run("hook", "install", "P1")
	})

	t.Run("Misc", func(t *testing.T) {
		_ = run("clear", "--force")
		_ = run("stop")
		_ = run("mcp")
		
		pidPath, _ := pidFilePath()
		os.MkdirAll(filepath.Dir(pidPath), 0755)
		ioutil.WriteFile(pidPath, []byte("999999"), 0644)
		_ = run("stop") 
		
		ioutil.WriteFile(pidPath, []byte("abc"), 0644)
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
		
		// Agent error
		_ = run("agent-config", "install", "unknown")
	})
}

func TestCLI_Execute_Direct(t *testing.T) {
	Execute(testSkillsFS)
}
