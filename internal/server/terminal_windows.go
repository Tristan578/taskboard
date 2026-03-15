//go:build windows

package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/UserExistsError/conpty"
	"github.com/gorilla/websocket"
)

func getWindowsShell() string {
	if ps, err := exec.LookPath("powershell.exe"); err == nil {
		return ps
	}
	return "cmd.exe"
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	shell := os.Getenv("COMSPEC")
	if shell == "" {
		shell = getWindowsShell()
	}

	cpty, err := conpty.Start(shell, conpty.ConPtyDimensions(120, 30))
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"failed to start terminal: `+err.Error()+`"}`))
		return
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = cpty.Close()
		})
	}
	defer cleanup()

	// ConPTY output → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := cpty.Read(buf)
			if err != nil {
				_ = conn.Close()
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket → ConPTY input
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			cleanup()
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			_, _ = cpty.Write(msg)
		case websocket.TextMessage:
			var ctrl struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal(msg, &ctrl) == nil && ctrl.Type == "resize" {
				_ = cpty.Resize(int(ctrl.Cols), int(ctrl.Rows))
			}
		}
	}
}
