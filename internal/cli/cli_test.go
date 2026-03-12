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
	root := &cobra.Command{Use: "test"}
	root.AddCommand(projectCommands())
	b := bytes.NewBufferString("")
	root.SetOut(b)

	// Create
	root.SetArgs([]string{"project", "create", "P1", "--prefix", "P1"})
	_ = root.Execute()
	if !contains(b.String(), "Created project") { t.Errorf("Project create output missing. Got: %s", b.String()) }

	// List
	b.Reset()
	root.SetArgs([]string{"project", "list"})
	_ = root.Execute()
	if !contains(b.String(), "P1") { t.Errorf("Project list missing P1. Got: %s", b.String()) }
}

func TestCLI_TeamCRUD(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-team")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := &cobra.Command{Use: "test"}
	root.AddCommand(teamCommands())
	b := bytes.NewBufferString("")
	root.SetOut(b)

	// Create
	root.SetArgs([]string{"team", "create", "Devs"})
	_ = root.Execute()
	if !contains(b.String(), "Created team") { t.Errorf("Team create output missing. Got: %s", b.String()) }

	// List
	b.Reset()
	root.SetArgs([]string{"team", "list"})
	_ = root.Execute()
	if !contains(b.String(), "Devs") { t.Errorf("Team list missing Devs. Got: %s", b.String()) }
}

func TestCLI_TicketCRUD(t *testing.T) {
	tempDir, _ := ioutil.TempDir("", "cli-test-ticket")
	defer os.RemoveAll(tempDir)
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	dbPath = "test.db"
	root := &cobra.Command{Use: "test"}
	root.AddCommand(projectCommands())
	root.AddCommand(ticketCommands())
	b := bytes.NewBufferString("")
	root.SetOut(b)

	// Create project
	root.SetArgs([]string{"project", "create", "P1", "--prefix", "P1"})
	_ = root.Execute()
	
	// List tickets (empty)
	b.Reset()
	root.SetArgs([]string{"ticket", "list"})
	_ = root.Execute()
	if !contains(b.String(), "No tickets found") { t.Errorf("Expected 'No tickets found', got: %s", b.String()) }
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
