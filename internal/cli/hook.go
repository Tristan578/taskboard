package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func hookCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage Git hooks for auto-sync",
	}

	installCmd := &cobra.Command{
		Use:   "install [project_id]",
		Short: "Install pre-push and post-merge hooks for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			
			// Find .git directory
			gitDir := ".git"
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				return fmt.Errorf("current directory is not a git repository")
			}

			hooksDir := filepath.Join(gitDir, "hooks")
			if err := os.MkdirAll(hooksDir, 0700); err != nil {
				return fmt.Errorf("creating hooks directory: %w", err)
			}

			hooks := []string{"pre-push", "post-merge"}
			for _, hook := range hooks {
				hookPath := filepath.Join(hooksDir, hook)
				content := fmt.Sprintf("#!/bin/sh\n# player2-kanban auto-sync hook\nif command -v player2-kanban >/dev/null 2>&1; then\n  player2-kanban project sync %s --async\nfi\n", projectID)
				
				// #nosec G306
				if err := os.WriteFile(hookPath, []byte(content), 0700); err != nil {
					return fmt.Errorf("writing %s hook: %w", hook, err)
				}
				fmt.Printf("Installed %s hook.\n", hook)
			}

			fmt.Println("Hooks installed successfully. Make sure player2-kanban is in your PATH and GITHUB_TOKEN is set.")
			return nil
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove player2-kanban Git hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			gitDir := ".git"
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				return fmt.Errorf("current directory is not a git repository")
			}

			hooks := []string{"pre-push", "post-merge"}
			removed := 0
			for _, hook := range hooks {
				hookPath := filepath.Join(gitDir, "hooks", hook)
				// #nosec G304 -- path is constructed from hardcoded .git/hooks/ + hardcoded hook names
				data, err := os.ReadFile(hookPath)
				if err != nil {
					continue
				}
				// Only remove hooks we installed
				if !strings.Contains(string(data), "player2-kanban") {
					continue
				}
				if err := os.Remove(hookPath); err != nil {
					return fmt.Errorf("removing %s hook: %w", hook, err)
				}
				fmt.Printf("Removed %s hook.\n", hook)
				removed++
			}
			if removed == 0 {
				fmt.Println("No player2-kanban hooks found.")
			} else {
				fmt.Println("Hooks removed successfully.")
			}
			return nil
		},
	}

	cmd.AddCommand(installCmd, uninstallCmd)
	return cmd
}

