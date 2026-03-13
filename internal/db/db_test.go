package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDB_OpenAt(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "db-test")
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := OpenAt(dbPath)
	if err != nil {
		t.Fatalf("OpenAt failed: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("DB file not created")
	}
}

func TestDB_DefaultDBPath(t *testing.T) {
	path, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath failed: %v", err)
	}
	if path == "" {
		t.Errorf("DefaultDBPath returned empty string")
	}
}

func TestDB_Open(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "db-open-test")
	defer os.RemoveAll(tempDir)

	// Override APPDATA/XDG_CONFIG_HOME
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tempDir)
	defer os.Setenv("APPDATA", oldAppData)

	db, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	db.Close()
}
