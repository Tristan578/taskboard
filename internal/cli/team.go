package cli

import (
	"github.com/spf13/cobra"
	"github.com/Tristan578/taskboard/internal/models"
)

func teamCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage teams",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			teams, _, err := store.ListTeams()
			if err != nil {
				return err
			}
			if len(teams) == 0 {
				cmd.Println("No teams found.")
				return nil
			}
			for _, t := range teams {
				cmd.Printf("%s (%s)\n", t.Name, t.ID)
			}
			return nil
		},
	}

	var color string
	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			t, err := store.CreateTeam(models.CreateTeamRequest{
				Name:  args[0],
				Color: color,
			})
			if err != nil {
				return err
			}
			cmd.Printf("Created team %s (%s)\n", t.Name, t.ID)
			return nil
		},
	}
	createCmd.Flags().StringVar(&color, "color", "#6366F1", "hex color")

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.DeleteTeam(args[0]); err != nil {
				return err
			}
			cmd.Println("Team deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, createCmd, deleteCmd)
	return cmd
}
