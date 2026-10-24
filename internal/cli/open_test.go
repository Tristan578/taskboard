package cli

import (
	"strings"
	"testing"
)

func TestDesktopCommands(t *testing.T) {
	for _, platform := range []string{"windows", "darwin", "linux"} {
		program, args := browserCommand(platform, "http://127.0.0.1:3010")
		if program == "" || !strings.Contains(strings.Join(args, " "), "http://127.0.0.1:3010") {
			t.Fatalf("invalid opener for %s", platform)
		}
	}
	if err := openDesktop(0, true); err == nil {
		t.Fatal("invalid port accepted")
	}
	if err := openDesktop(65536, true); err == nil {
		t.Fatal("invalid port accepted")
	}
	c := NewRootCmd(nil)
	c.SetArgs([]string{"open", "--no-browser"})
	if err := c.Execute(); err == nil {
		t.Fatal("test executable must not spawn a daemon")
	}
}
