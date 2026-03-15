//go:build ignore

// kill-stale-tests finds and kills orphaned Go test binaries (cli.test,
// player2-kanban.test, etc.) that were leaked by previous test runs.
//
// Cross-platform: Windows, macOS, Linux.
//
// Usage:
//
//	go run scripts/kill-stale-tests.go          # kill all stale .test processes
//	go run scripts/kill-stale-tests.go --dry-run # list without killing
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// knownTestBinaries lists Go test binary names that may leak.
// Add new entries here if new packages grow long-running test processes.
var knownTestBinaries = []string{
	"cli.test",
	"server.test",
	"db.test",
	"github.test",
	"mcp.test",
	"models.test",
	"kanban.test",
}

func main() {
	dryRun := flag.Bool("dry-run", false, "list stale processes without killing them")
	flag.Parse()

	self := os.Getpid()
	killed := 0

	switch runtime.GOOS {
	case "windows":
		killed = cleanupWindows(self, *dryRun)
	default:
		killed = cleanupUnix(self, *dryRun)
	}

	if killed == 0 {
		fmt.Println("No stale test processes found.")
	} else if *dryRun {
		fmt.Printf("Found %d stale test process(es). Run without --dry-run to kill them.\n", killed)
	} else {
		fmt.Printf("Killed %d stale test process(es).\n", killed)
	}
}

func cleanupWindows(self int, dryRun bool) int {
	killed := 0
	for _, name := range knownTestBinaries {
		exeName := name + ".exe"
		// Use filtered tasklist — fast, returns only matching processes.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		out, err := exec.CommandContext(ctx, "tasklist",
			"/FI", fmt.Sprintf("IMAGENAME eq %s", exeName),
			"/FO", "CSV", "/NH",
		).Output()
		cancel()
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "No tasks") || strings.HasPrefix(line, "INFO:") {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) < 2 {
				continue
			}
			pidStr := strings.Trim(parts[1], "\" ")
			pid, err := strconv.Atoi(pidStr)
			if err != nil || pid == self {
				continue
			}

			if dryRun {
				fmt.Printf("[dry-run] would kill PID %d (%s)\n", pid, exeName)
				killed++
				continue
			}

			killCtx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := exec.CommandContext(killCtx, "taskkill", "/F", "/PID", strconv.Itoa(pid)).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to kill PID %d (%s): %v\n", pid, exeName, err)
			} else {
				fmt.Printf("killed PID %d (%s)\n", pid, exeName)
				killed++
			}
			killCancel()
		}
	}
	return killed
}

func cleanupUnix(self int, dryRun bool) int {
	killed := 0
	for _, name := range knownTestBinaries {
		// pgrep -f matches against full command line
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := exec.CommandContext(ctx, "pgrep", "-x", name).Output()
		cancel()
		if err != nil {
			continue
		}

		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			pid, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || pid == self {
				continue
			}

			if dryRun {
				fmt.Printf("[dry-run] would kill PID %d (%s)\n", pid, name)
				killed++
				continue
			}

			if p, err := os.FindProcess(pid); err == nil {
				if err := p.Kill(); err == nil {
					fmt.Printf("killed PID %d (%s)\n", pid, name)
					killed++
				}
			}
		}
	}
	return killed
}
