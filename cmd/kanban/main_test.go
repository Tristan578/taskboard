package main

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestEmbeddedWebAssets verifies that the embedded web/dist filesystem
// contains the expected files (index.html at minimum).
func TestEmbeddedWebAssets(t *testing.T) {
	var found []string
	err := fs.WalkDir(webEmbed, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking embedded FS: %v", err)
	}

	if len(found) == 0 {
		t.Fatal("embedded FS contains no files")
	}

	// Verify index.html is present
	hasIndex := false
	for _, f := range found {
		t.Logf("embedded file: %s", f)
		if filepath.Base(f) == "index.html" {
			hasIndex = true
		}
	}
	if !hasIndex {
		t.Error("index.html not found in embedded FS")
	}
}

// TestWebFS verifies that webFS() returns a valid sub-filesystem
// and that index.html is accessible from it.
func TestWebFS(t *testing.T) {
	wfs := webFS()
	if wfs == nil {
		t.Fatal("webFS() returned nil")
	}

	// Should be able to open index.html from the sub FS
	f, err := wfs.Open("index.html")
	if err != nil {
		t.Fatalf("opening index.html from sub FS: %v", err)
	}
	f.Close()
}

// freePort finds a free TCP port by binding to :0 and returning the assigned port.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// binaryName returns the expected binary name for the current OS.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "player2-kanban.exe"
	}
	return "player2-kanban"
}

// buildBinary compiles the cmd/kanban binary into a temp directory.
func buildBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, binaryName())

	// Build from the cmd/kanban package
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Dir(mustAbs(t, "main.go"))
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building binary: %v\n%s", err, out)
	}
	return binPath
}

func mustAbs(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

// TestBinaryBuilds just verifies the binary compiles without error.
func TestBinaryBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	bin := buildBinary(t)
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("binary is empty")
	}
	t.Logf("built binary: %s (%d bytes)", bin, info.Size())
}

// TestIntegrationSmokeTest builds the binary, starts the server on a random port,
// makes an HTTP GET to /api/projects, verifies 200, then shuts down.
func TestIntegrationSmokeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	bin := buildBinary(t)
	port := freePort(t)
	dbFile := filepath.Join(t.TempDir(), "test.db")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "--db", dbFile, "start", "--foreground", "--port", fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting binary: %v", err)
	}
	// Always kill and wait to prevent process leaks
	defer cmd.Process.Kill()
	defer cmd.Wait()

	// Wait for the server to be ready with retries
	addr := fmt.Sprintf("http://127.0.0.1:%d/api/projects", port)
	client := &http.Client{Timeout: 2 * time.Second}

	var lastErr error
	ready := false
	for i := 0; i < 30; i++ {
		resp, err := client.Get(addr)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ready = true
				t.Logf("server ready after %d attempts", i+1)
				break
			}
			lastErr = fmt.Errorf("unexpected status: %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("server never became ready: %v", lastErr)
	}

	// Verify /api/projects returns 200 with valid JSON
	resp, err := client.Get(addr)
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// TestGracefulShutdown verifies the server process terminates when killed.
func TestGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	bin := buildBinary(t)
	port := freePort(t)
	dbFile := filepath.Join(t.TempDir(), "test.db")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "--db", dbFile, "start", "--foreground", "--port", fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting binary: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for server to be ready
	client := &http.Client{Timeout: 2 * time.Second}
	addr := fmt.Sprintf("http://127.0.0.1:%d/api/projects", port)
	for i := 0; i < 30; i++ {
		resp, err := client.Get(addr)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if i == 29 {
			t.Fatal("server never became ready")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Kill the process and verify it exits
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("killing process: %v", err)
	}

	// Wait for process to exit with a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		t.Log("process terminated successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("process did not terminate within 5 seconds")
	}

	// Verify the port is no longer in use
	time.Sleep(200 * time.Millisecond)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
	if err == nil {
		conn.Close()
		t.Error("port still in use after process termination")
	}
}
