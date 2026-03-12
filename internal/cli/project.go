package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Tristan578/taskboard/internal/github"
	"github.com/Tristan578/taskboard/internal/models"
)

func projectCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			projects, err := store.ListProjects("")
			if err != nil {
				return err
			}
			if len(projects) == 0 {
				cmd.Println("No projects found.")
				return nil
			}
			for _, p := range projects {
				icon := p.Icon
				if icon == "" {
					icon = " "
				}
				cmd.Printf("%s %s [%s] (%s) - %s\n", icon, p.Name, p.Prefix, p.Status, p.ID)
			}
			return nil
		},
	}

	var prefix, icon, color string
	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			p, err := store.CreateProject(models.CreateProjectRequest{
				Name:   args[0],
				Prefix: prefix,
				Icon:   icon,
				Color:  color,
			})
			if err != nil {
				return err
			}
			cmd.Printf("Created project %s [%s] (%s)\n", p.Name, p.Prefix, p.ID)
			return nil
		},
	}
	createCmd.Flags().StringVar(&prefix, "prefix", "", "project prefix (required)")
	createCmd.MarkFlagRequired("prefix")
	createCmd.Flags().StringVar(&icon, "icon", "", "emoji icon")
	createCmd.Flags().StringVar(&color, "color", "#3B82F6", "hex color")

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.DeleteProject(args[0]); err != nil {
				return err
			}
			fmt.Println("Project deleted.")
			return nil
		},
	}

	linkCmd := &cobra.Command{
		Use:   "link [id] [repo_url]",
		Short: "Link a project to a GitHub repository",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			repo := args[1]
			_, err = store.UpdateProject(args[0], models.UpdateProjectRequest{
				GitHubRepo: &repo,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Linked project to %s\n", repo)
			return nil
		},
	}

	var syncAsync bool
	syncCmd := &cobra.Command{
		Use:   "sync [id]",
		Short: "Sync project with GitHub issues",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}

			if syncAsync {
				if err := store.QueueSyncJob(args[0], "", "full_sync", nil); err != nil {
					return err
				}
				fmt.Println("Sync job queued.")
				return nil
			}

			token := os.Getenv("GITHUB_TOKEN")
			if token == "" {
				return fmt.Errorf("GITHUB_TOKEN environment variable not set")
			}
			client := github.NewClient(cmd.Context(), token)
			fmt.Printf("Syncing project %s with GitHub...\n", args[0])
			if err := github.SyncProject(cmd.Context(), client, store, args[0]); err != nil {
				return err
			}
			fmt.Println("Sync complete.")
			return nil
		},
	}
	syncCmd.Flags().BoolVar(&syncAsync, "async", false, "queue the sync job and exit immediately")

	cmd.AddCommand(listCmd, createCmd, deleteCmd, linkCmd, syncCmd)
	return cmd
}
