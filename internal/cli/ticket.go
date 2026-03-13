package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/Tristan578/taskboard/internal/models"
)

func ticketCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Manage tickets",
	}

	var projectID, status, priority string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tickets",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			tickets, err := store.ListTickets(models.TicketFilter{
				ProjectID: projectID,
				Status:    status,
				Priority:  priority,
			})
			if err != nil {
				return err
			}
			if len(tickets) == 0 {
				cmd.Println("No tickets found.")
				return nil
			}
			for _, t := range tickets {
				key := t.DisplayKey()
				cmd.Printf("[%s] %s - %s (%s, %s)\n", key, t.Title, t.Status, t.Priority, t.ID)
			}
			return nil
		},
	}
	listCmd.Flags().StringVar(&projectID, "project", "", "filter by project ID")
	listCmd.Flags().StringVar(&status, "status", "", "filter by status (todo|in_progress|done)")
	listCmd.Flags().StringVar(&priority, "priority", "", "filter by priority (urgent|high|medium|low)")

	var createProject, createPriority, createDue, createTeam string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new ticket",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			title, _ := cmd.Flags().GetString("title")
			req := models.CreateTicketRequest{
				ProjectID: createProject,
				Title:     title,
				Priority:  createPriority,
			}
			if createDue != "" {
				req.DueDate = &createDue
			}
			if createTeam != "" {
				req.TeamID = &createTeam
			}
			t, err := store.CreateTicket(req)
			if err != nil {
				return err
			}
			cmd.Printf("Created ticket %s: %s (%s)\n", t.DisplayKey(), t.Title, t.ID)
			return nil
		},
	}
	createCmd.Flags().StringVar(&createProject, "project", "", "project ID (required)")
	createCmd.MarkFlagRequired("project")
	createCmd.Flags().String("title", "", "ticket title (required)")
	createCmd.MarkFlagRequired("title")
	createCmd.Flags().StringVar(&createPriority, "priority", "medium", "priority (urgent|high|medium|low)")
	createCmd.Flags().StringVar(&createDue, "due", "", "due date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createTeam, "team", "", "team ID")

	var moveStatus string
	moveCmd := &cobra.Command{
		Use:   "move [id]",
		Short: "Move ticket to different status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			t, err := store.MoveTicket(args[0], models.MoveTicketRequest{Status: moveStatus})
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("ticket not found")
			}
			cmd.Printf("Moved %s to %s\n", t.DisplayKey(), t.Status)
			return nil
		},
	}
	moveCmd.Flags().StringVar(&moveStatus, "status", "", "target status (required)")
	moveCmd.MarkFlagRequired("status")

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			// Check existence for clean output
			t, _ := store.GetTicket(args[0])
			if t == nil {
				return fmt.Errorf("ticket not found")
			}
			if err := store.DeleteTicket(args[0]); err != nil {
				return err
			}
			cmd.Println("Ticket deleted.")
			return nil
		},
	}

	subtaskCmd := &cobra.Command{
		Use:   "subtask",
		Short: "Manage ticket subtasks",
	}

	subtaskAddCmd := &cobra.Command{
		Use:   "add [ticket_id] [title]",
		Short: "Add a subtask to a ticket",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			st, err := store.AddSubtask(args[0], models.CreateSubtaskRequest{Title: args[1]})
			if err != nil {
				return err
			}
			cmd.Printf("Created subtask %s (%s)\n", st.Title, st.ID)
			return nil
		},
	}

	subtaskToggleCmd := &cobra.Command{
		Use:   "toggle [id]",
		Short: "Toggle subtask completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			st, err := store.ToggleSubtask(args[0])
			if err != nil {
				return err
			}
			if st == nil {
				return fmt.Errorf("subtask not found")
			}
			cmd.Printf("Subtask %s is now completed: %v\n", st.Title, st.Completed)
			return nil
		},
	}

	subtaskDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a subtask",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			st, _ := store.GetSubtask(args[0])
			if st == nil {
				return fmt.Errorf("subtask not found")
			}
			if err := store.DeleteSubtask(args[0]); err != nil {
				return err
			}
			cmd.Println("Subtask deleted.")
			return nil
		},
	}

	subtaskCmd.AddCommand(subtaskAddCmd, subtaskToggleCmd, subtaskDeleteCmd)
	cmd.AddCommand(listCmd, createCmd, moveCmd, deleteCmd, subtaskCmd)
	return cmd
}
