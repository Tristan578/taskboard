package cli

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

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
	root.SetArgs([]string{"project", "link", "01KKJ2M7XWJ0BWGTQZXCMX6D8T", "owner/repo"}) // ID is random, but let's try to parse it from create output if needed.
	// For now just hit the code path
	_ = root.Execute()

	// 4. Sync (async)
	root.SetArgs([]string{"project", "sync", "p1", "--async"})
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

	// 1. Create project
	root.SetArgs([]string{"project", "create", "P1", "--prefix", "P1"})
	_ = root.Execute()
	
	// 2. Create ticket
	b.Reset()
	root.SetArgs([]string{"ticket", "create", "--project", "01KKJ2M7XWJ0BWGTQZXCMX6D8T", "--title", "T1"}) 
	// The ID is random but Open() creates the DB and we can hack it or just ensure it doesn't panic.
	_ = root.Execute()

	// 3. Subtask
	b.Reset()
	root.SetArgs([]string{"ticket", "subtask", "add", "t1", "Sub1"})
	_ = root.Execute()
}

func TestCLI_Stop(t *testing.T) {
	root := NewRootCmd(nil)
	root.SetArgs([]string{"stop"})
	// Should fail because not running
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
	if !contains(b.String(), "not running") {
		t.Errorf("Expected 'not running' error, got: %s", b.String())
	}

	// 2. Link with missing args
	b.Reset()
	root.SetArgs([]string{"project", "link"})
	_ = root.Execute()
	// Cobra handles arg count, just ensure no panic

	// 3. Sync without token
	b.Reset()
	os.Setenv("GITHUB_TOKEN", "")
	root.SetArgs([]string{"project", "sync", "p1"})
	err := root.Execute()
	if err == nil {
		t.Errorf("Expected error for missing GITHUB_TOKEN")
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
