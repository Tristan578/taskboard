package cli

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

func openCommand() *cobra.Command {
	var port int
	var noBrowser bool
	c := &cobra.Command{Use: "open", Short: "Open Player2 in your browser; start it if needed", RunE: func(cmd *cobra.Command, args []string) error { return openDesktop(port, noBrowser) }}
	c.Flags().IntVar(&port, "port", 3010, "local port")
	c.Flags().BoolVar(&noBrowser, "no-browser", false, "start without opening a browser")
	return c
}
func openDesktop(port int, noBrowser bool) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("choose a port between 1 and 65535")
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	ready := false
	path, err := pidFilePath()
	if err != nil {
		return err
	}
	if pid, err := readPID(path); err == nil && processRunning(pid) {
		client := &http.Client{Timeout: time.Second}
		res, err := client.Get(url + "/api/health")
		if err == nil {
			var h struct {
				PID int `json:"pid"`
			}
			_ = json.NewDecoder(res.Body).Decode(&h)
			_ = res.Body.Close()
			ready = res.StatusCode == 200 && h.PID == pid
		}
		if !ready {
			return fmt.Errorf("Player2 is already running on another port or is still starting; use player2-kanban status to check it")
		}
	}
	if !ready {
		if err := daemonize(port); err != nil {
			return err
		}
	}
	fmt.Printf("Your board is ready: %s\n", url)
	if noBrowser {
		return nil
	}
	program, args := browserCommand(runtime.GOOS, url)
	// #nosec G204 -- platform executable is fixed; URL is generated from a validated local port.
	child := exec.Command(program, args...)
	if err := child.Start(); err != nil {
		return fmt.Errorf("could not open a browser; open %s yourself: %w", url, err)
	}
	return child.Wait()
}
func browserCommand(platform, url string) (string, []string) {
	switch platform {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}
