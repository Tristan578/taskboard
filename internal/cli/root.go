package cli

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/github"
	"github.com/Tristan578/taskboard/internal/mcp"
	"github.com/Tristan578/taskboard/internal/server"
	)
var (
	port       int
	foreground bool
	dbPath     string
)

func NewRootCmd(webFS fs.FS) *cobra.Command {
	root := &cobra.Command{
		Use:   "player2-kanban",
		Short: "Agent-native local project management with GitHub sync and MCP",
	}
	root.PersistentFlags().StringVar(&dbPath, "db", "", "path to SQLite database file (default: OS config dir)")

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the web UI server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !foreground {
				return daemonize(port)
			}
			database, err := openDB()
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			store := db.NewStore(database)
			
			// Start background worker if GITHUB_TOKEN is set
			token := os.Getenv("GITHUB_TOKEN")
			if token != "" {
				client := github.NewClient(cmd.Context(), token)
				worker := github.NewWorker(store, client)
				go worker.Start(cmd.Context())
				fmt.Println("GitHub Sync Worker started.")
			}

			srv := server.New(store, webFS)
			return srv.ListenAndServe(port)
		},
	}
	startCmd.Flags().IntVarP(&port, "port", "p", 3010, "port to listen on")
	startCmd.Flags().BoolVar(&foreground, "foreground", false, "run in foreground instead of as a daemon")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the running server",
		RunE: func(cmd *cobra.Command, args []string) error {
			pidPath, err := pidFilePath()
			if err != nil {
				return err
			}

			pid, err := readPID(pidPath)
			if err != nil {
				return fmt.Errorf("player2-kanban is not running")
			}

			process, err := os.FindProcess(pid)
			if err != nil {
				_ = os.Remove(pidPath)
				return fmt.Errorf("player2-kanban is not running")
			}

			if err := process.Signal(syscall.Signal(0)); err != nil {
				_ = os.Remove(pidPath)
				return fmt.Errorf("player2-kanban is not running (stale pid file removed)")
			}

			if err := process.Signal(syscall.SIGTERM); err != nil {
				_ = os.Remove(pidPath)
				return fmt.Errorf("failed to stop player2-kanban: %w", err)
			}

			_ = os.Remove(pidPath)
			fmt.Printf("Player2 Kanban stopped (pid %d)\n", pid)
			return nil
		},
	}

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openDB()
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			store := db.NewStore(database)
			server := mcp.NewServer(store)
			return server.Run(os.Stdin, os.Stdout)
		},
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete all data from the database (keeps schema intact)",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				cmd.Print("This will delete all projects, tickets, teams, and labels. Continue? [y/N] ")
				var answer string
				_, _ = fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					cmd.Println("Aborted.")
					return nil
				}
			}

			store, err := openStore()
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			if err := store.ClearData(); err != nil {
				return fmt.Errorf("clearing data: %w", err)
			}
			cmd.Println("All data cleared.")
			return nil
		},
	}
	clearCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	root.AddCommand(startCmd, stopCmd, mcpCmd, clearCmd)
	root.AddCommand(projectCommands())
	root.AddCommand(teamCommands())
	root.AddCommand(ticketCommands())
	root.AddCommand(hookCommands())
	root.AddCommand(agentConfigCommands())

	return root
}

func Execute(webFS fs.FS) {
	if err := NewRootCmd(webFS).Execute(); err != nil {
		os.Exit(1)
	}
}

func openDB() (*sql.DB, error) {
	if dbPath != "" {
		return db.OpenAt(dbPath)
	}
	return db.Open()
}

func openStore() (*db.Store, error) {
	database, err := openDB()
	if err != nil {
		return nil, err
	}
	return db.NewStore(database), nil
}

func isTestBinary() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	base := strings.ToLower(filepath.Base(exe))
	return strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".test.exe")
}

func daemonize(port int) error {
	// Guard: never spawn daemon processes from Go test binaries.
	// Go test binaries always end in ".test" / ".test.exe" — this is
	// automatic, requires no env var, and cannot be accidentally bypassed.
	// Without this, daemonize() spawns immortal cli.test.exe children
	// that accumulate across test runs and exhaust system resources.
	if isTestBinary() {
		return fmt.Errorf("daemonize disabled in test binary")
	}

	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}

	if pid, err := readPID(pidPath); err == nil {
		if p, err := os.FindProcess(pid); err == nil {
			if err := p.Signal(syscall.Signal(0)); err == nil {
				return fmt.Errorf("player2-kanban is already running (pid %d)", pid)
			}
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	daemonArgs := []string{"start", "--port", strconv.Itoa(port), "--foreground"}
	if dbPath != "" {
		daemonArgs = append([]string{"--db", dbPath}, daemonArgs...)
	}
	// #nosec G204
	cmd := exec.Command(exe, daemonArgs...)
	// cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	if err := writePID(pidPath, cmd.Process.Pid); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}

	fmt.Printf("Player2 Kanban running at http://localhost:%d (pid %d)\n", port, cmd.Process.Pid)
	return nil
}

func pidFilePath() (string, error) {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		home, err2 := os.UserHomeDir()
		if err2 != nil {
			return "", fmt.Errorf("finding home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".config")
	}
	return filepath.Join(dataDir, "player2-kanban", "player2-kanban.pid"), nil
}

func writePID(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0600)
}

func readPID(path string) (int, error) {
	// #nosec G304
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
