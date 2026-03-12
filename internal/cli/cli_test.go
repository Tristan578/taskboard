package cli

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestCLI_Root(t *testing.T) {
	root := NewRootCmd(nil)
	
	b := bytes.NewBufferString("")
	root.SetOut(b)
	root.SetArgs([]string{"--help"})
	
	err := root.Execute()
	if err != nil {
		t.Fatalf("Root command failed: %v", err)
	}
	
	if !contains(b.String(), "player2-kanban") {
		t.Errorf("Unexpected help output")
	}
}

func TestCLI_HookInstall(t *testing.T) {
	root := &cobra.Command{Use: "test"}
	root.AddCommand(hookCommands())
	
	tempDir, _ := ioutil.TempDir("", "cli-test-hook")
	defer os.RemoveAll(tempDir)
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)
	
	// Fail if not a git repo
	root.SetArgs([]string{"hook", "install", "p1"})
	err := root.Execute()
	if err == nil {
		t.Errorf("Expected error when not in a git repo")
	}

	// Success in a git repo
	os.Mkdir(".git", 0755)
	err = root.Execute()
	if err != nil {
		t.Fatalf("hook command failed: %v", err)
	}
	
	if _, err := os.Stat(".git/hooks/pre-push"); os.IsNotExist(err) {
		t.Errorf("pre-push hook not created")
	}
}

func TestCLI_AgentConfig(t *testing.T) {
	root := &cobra.Command{Use: "test"}
	root.AddCommand(agentConfigCommands())
	
	tempDir, _ := ioutil.TempDir("", "cli-test-agent")
	defer os.RemoveAll(tempDir)
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)
	
	root.SetArgs([]string{"agent-config", "install", "cursor"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("agent-config command failed: %v", err)
	}
	
	if _, err := os.Stat(".cursorrules"); os.IsNotExist(err) {
		t.Errorf(".cursorrules not created")
	}
}

func TestCLI_ProjectSync(t *testing.T) {
	// This will try to open store, so we might need a way to mock openStore
	// For now, just test the command definition
	cmd := projectCommands()
	if cmd.Use != "project" {
		t.Errorf("Unexpected command use")
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
