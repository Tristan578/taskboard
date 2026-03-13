package cli

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func find(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func contains(s, substr string) bool {
	return find(s, substr) >= 0
}

func TestCLI_Root(t *testing.T) {
	root := NewRootCmd(nil)
	b := bytes.NewBufferString("")
	root.SetOut(b)
	root.SetArgs([]string{"--help"})
	err := root.Execute()
	if err != nil { t.Fatalf("Root command failed: %v", err) }
	if !contains(b.String(), "player2-kanban") { t.Errorf("Unexpected help output") }
}

func TestCLI_ProjectCRUD(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-project")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := NewRootCmd(nil)
	b := bytes.NewBufferString("")
	root.SetOut(b)

	// 1. Create
	root.SetArgs([]string{"project", "create", "P1", "--prefix", "P1"})
	_ = root.Execute()
	
	// 2. List
	b.Reset()
	root.SetArgs([]string{"project", "list"})
	_ = root.Execute()
	if !contains(b.String(), "P1") { t.Errorf("Project list missing P1") }

	// 3. Link
	b.Reset()
	root.SetArgs([]string{"project", "link", "p1", "owner/repo"})
	_ = root.Execute()

	// 4. Sync (async)
	b.Reset()
	root.SetArgs([]string{"project", "sync", "p1", "--async"})
	_ = root.Execute()
	
	// 5. Delete
	b.Reset()
	root.SetArgs([]string{"project", "delete", "p1"})
	_ = root.Execute()
}

func TestCLI_TeamCRUD(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-team")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := NewRootCmd(nil)
	b := bytes.NewBufferString("")
	root.SetOut(b)

	root.SetArgs([]string{"team", "create", "Devs"})
	_ = root.Execute()

	b.Reset()
	root.SetArgs([]string{"team", "list"})
	_ = root.Execute()
	if !contains(b.String(), "Devs") { t.Errorf("Team list missing Devs") }

	b.Reset()
	root.SetArgs([]string{"team", "delete", "t1"})
	_ = root.Execute()
}

func TestCLI_TicketCRUD(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-ticket")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := NewRootCmd(nil)
	b := bytes.NewBufferString("")
	root.SetOut(b)

	root.SetArgs([]string{"project", "create", "P1", "--prefix", "P1"})
	_ = root.Execute()
	
	b.Reset()
	root.SetArgs([]string{"ticket", "list"})
	_ = root.Execute()

	b.Reset()
	root.SetArgs([]string{"ticket", "move", "t1", "--status", "done"})
	_ = root.Execute()

	b.Reset()
	root.SetArgs([]string{"ticket", "delete", "t1"})
	_ = root.Execute()
}

func TestCLI_ClearData(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-clear")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := NewRootCmd(nil)
	b := bytes.NewBufferString("")
	root.SetOut(b)

	root.SetArgs([]string{"project", "create", "P1", "--prefix", "P1"})
	_ = root.Execute()

	b.Reset()
	root.SetArgs([]string{"clear", "--force"})
	_ = root.Execute()
	if !contains(b.String(), "All data cleared") { t.Errorf("Expected 'All data cleared', got: %s", b.String()) }
}

func TestCLI_AgentConfig_All(t *testing.T) {
	agents := []string{"cursor", "claude", "gemini", "windsurf", "antigravity", "copilot", "codex"}
	root := NewRootCmd(nil)
	
	tempDir, _ := ioutil.TempDir("", "cli-test-agents")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	for _, a := range agents {
		root.SetArgs([]string{"agent-config", "install", a})
		_ = root.Execute()
	}
}

func TestCLI_ErrorPaths(t *testing.T) {
	root := NewRootCmd(nil)
	b := bytes.NewBufferString("")
	root.SetOut(b)

	// 1. Stop when not running
	root.SetArgs([]string{"stop"})
	_ = root.Execute()

	// 2. Sync without token
	os.Setenv("GITHUB_TOKEN", "")
	root.SetArgs([]string{"project", "sync", "p1"})
	_ = root.Execute()
}

func TestCLI_HookInstall(t *testing.T) {
	root := NewRootCmd(nil)
	
	tempDir, _ := ioutil.TempDir("", "cli-test-hook")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	os.Mkdir(".git", 0755)
	root.SetArgs([]string{"hook", "install", "p1"})
	_ = root.Execute()
}

func TestCLI_PIDMethods(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-pid")
	defer os.RemoveAll(tempDir)
	
	path := filepath.Join(tempDir, "test.pid")
	err := writePID(path, 1234)
	if err != nil { t.Fatal(err) }
	
	pid, err := readPID(path)
	if err != nil || pid != 1234 { t.Errorf("PID read/write failed") }
	
	_, err = readPID("nonexistent")
	if err == nil { t.Errorf("Expected error for nonexistent PID file") }
}

func TestCLI_SubtaskCommands(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-subtask")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := NewRootCmd(nil)
	
	root.SetArgs([]string{"ticket", "subtask", "add", "t1", "S1"})
	_ = root.Execute()

	root.SetArgs([]string{"ticket", "subtask", "toggle", "s1"})
	_ = root.Execute()
	
	root.SetArgs([]string{"ticket", "subtask", "delete", "s1"})
	_ = root.Execute()
}
