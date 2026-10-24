package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Desktop launchers have no console. Keep details local and show a short recovery message.
func showLaunchError(err error) {
	if p, e := pidFilePath(); e == nil {
		_ = os.MkdirAll(filepath.Dir(p), 0700)
		_ = os.WriteFile(filepath.Join(filepath.Dir(p), "launch-error.log"), []byte(err.Error()), 0600)
	}
	message := "Player2 could not open. It may already be starting or port 3010 may be busy. Try again shortly. Details are saved in launch-error.log in the Player2 configuration folder."
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("osascript", "-e", `display alert "Player2 could not open" message "Try again shortly. Details are saved in launch-error.log in your Player2 configuration folder."`)
	case "windows":
		command = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('Player2 could not open. Try again shortly. Details are saved in launch-error.log in the Player2 configuration folder.','Player2')`)
	default:
		command = exec.Command("zenity", "--error", "--title=Player2", "--text", message)
	}
	if e := command.Run(); e != nil {
		fmt.Fprintln(os.Stderr, message)
	}
}
