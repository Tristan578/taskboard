package recovery

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/models"
)

func setup(t *testing.T) (*Manager, *db.Store) {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "work.db"))
	if err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(database)
	t.Cleanup(func() { _ = store.Close() })
	return New(store), store
}
func TestAutomaticBackupRetentionAndRecovery(t *testing.T) {
	m, store := setup(t)
	p, _ := store.CreateProject(models.CreateProjectRequest{Name: "Home", Prefix: "H", CompletionPolicy: "manual"})
	_, _ = store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Plan the week"})
	if err := m.EnsureAutomatic(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := m.EnsureAutomatic(time.Now()); err != nil {
		t.Fatal(err)
	}
	status, err := m.Status()
	if err != nil || len(status.Backups) != 1 {
		t.Fatal("daily duplicate", status, err)
	}
	first := status.Backups[0]
	preview, err := m.PreviewSaved(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Boards) != 1 || preview.Tasks != 1 {
		t.Fatal(preview)
	}
	ids, err := m.Restore(preview.ID)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := m.Restore(preview.ID)
	if err != nil || len(retry) != 1 || retry[0] != ids[0] {
		t.Fatal("restore not idempotent", err)
	}
	if _, err = m.BackupPath("../../work.db"); err == nil {
		t.Fatal("path traversal allowed")
	}
	for i := 0; i < RetainedBackups; i++ {
		if _, err = m.Create(); err != nil {
			t.Fatal(err)
		}
	}
	status, err = m.Status()
	if err != nil || len(status.Backups) != RetainedBackups {
		t.Fatal("unbounded retention", err, len(status.Backups))
	}
	if _, err = m.BackupPath(first.ID); err == nil {
		t.Fatal("old backup not rotated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.Run(ctx)
}
func TestRecoveryFailuresPreserveData(t *testing.T) {
	m, store := setup(t)
	_, _ = store.CreateProject(models.CreateProjectRequest{Name: "Keep", Prefix: "K"})
	if _, err := m.Preview(bytes.NewBufferString("not SQLite")); err == nil {
		t.Fatal("invalid file accepted")
	}
	if _, err := m.Restore("../bad"); err == nil {
		t.Fatal("invalid restore ID accepted")
	}
	b, err := m.Create()
	if err != nil {
		t.Fatal(err)
	}
	preview, err := m.PreviewSaved(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := m.directory()
	file := filepath.Join(dir, "imports", preview.ID+".db")
	old := time.Now().Add(-2 * time.Hour)
	if err = os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Restore(preview.ID); err == nil {
		t.Fatal("expired preview accepted")
	}
	for i := 0; i < 5; i++ {
		if _, err = m.PreviewSaved(b.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = m.PreviewSaved(b.ID); err == nil {
		t.Fatal("unbounded staging files")
	}
	_, count, _ := store.ListProjects("")
	if count != 1 {
		t.Fatal("bad recovery changed boards")
	}
}
func TestBackupFailureIsVisible(t *testing.T) {
	m, store := setup(t)
	dir, _ := m.directory()
	if err := os.WriteFile(dir, []byte("block directory creation"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.EnsureAutomatic(time.Now()); err == nil {
		t.Fatal("backup should fail")
	}
	if m.lastError == "" {
		t.Fatal("backup failure invisible")
	}
	_ = os.Remove(dir)
	if _, err := m.Create(); err != nil {
		t.Fatal(err)
	}
	status, _ := m.Status()
	if status.LastError != "" {
		t.Fatal("successful retry did not clear error")
	}
	_ = store.Close()
	if _, err := m.Create(); err == nil {
		t.Fatal("backup closed database")
	}
}
