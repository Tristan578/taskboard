package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
				content := fmt.Sprintf("#!/bin/sh\n# player2-kanban auto-sync hook\nplayer2-kanban project sync %s --async\n", projectID)
				
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

	cmd.AddCommand(installCmd)
	return cmd
}
