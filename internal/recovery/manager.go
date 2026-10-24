package recovery

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/oklog/ulid/v2"
)

const MaxUpload = 32 << 20
const RetainedBackups = 14

var managedName = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}-(automatic|manual|before-restore)\.db$`)
var importID = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

type Backup struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Bytes     int64     `json:"bytes"`
	Reason    string    `json:"reason"`
}
type Status struct {
	Backups   []Backup `json:"backups"`
	LastError string   `json:"lastError,omitempty"`
	Directory string   `json:"directory"`
}
type Manager struct {
	store     *db.Store
	mu        sync.Mutex
	lastError string
}

func New(store *db.Store) *Manager { return &Manager{store: store} }
func (m *Manager) directory() (string, error) {
	p, e := m.store.DatabasePath()
	if e != nil {
		return "", e
	}
	if p == "" {
		return "", fmt.Errorf("a saved database is required for backups")
	}
	return p + ".backups", nil
}
func (m *Manager) list() (Status, error) {
	dir, err := m.directory()
	if err != nil {
		return Status{}, err
	}
	result := Status{Directory: dir, Backups: []Backup{}, LastError: m.lastError}
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	for _, f := range files {
		if !managedName.MatchString(f.Name()) || f.Type()&os.ModeSymlink != 0 || f.IsDir() {
			continue
		}
		info, e := f.Info()
		if e != nil {
			return result, e
		}
		if !info.Mode().IsRegular() {
			continue
		}
		result.Backups = append(result.Backups, Backup{ID: f.Name(), CreatedAt: info.ModTime(), Bytes: info.Size(), Reason: strings.TrimSuffix(f.Name()[27:], ".db")})
	}
	sort.Slice(result.Backups, func(i, j int) bool { return result.Backups[i].ID > result.Backups[j].ID })
	return result, nil
}
func (m *Manager) Status() (Status, error) { m.mu.Lock(); defer m.mu.Unlock(); return m.list() }
func (m *Manager) create(reason string) (Backup, error) {
	if reason != "automatic" && reason != "manual" && reason != "before-restore" {
		return Backup{}, fmt.Errorf("invalid backup type")
	}
	dir, err := m.directory()
	if err != nil {
		return Backup{}, err
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return Backup{}, err
	}
	id := ulid.Make().String() + "-" + reason + ".db"
	temp := filepath.Join(dir, id+".partial")
	destination := filepath.Join(dir, id)
	defer func() { _ = os.Remove(temp) }()
	if err = m.store.Backup(temp); err != nil {
		return Backup{}, err
	}
	if err = os.Chmod(temp, 0600); err != nil {
		return Backup{}, err
	}
	// #nosec G304 -- path is generated from a private database directory and a new ULID, never an upload filename.
	f, err := os.OpenFile(temp, os.O_RDWR, 0600)
	if err != nil {
		return Backup{}, err
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return Backup{}, err
	}
	if closeErr != nil {
		return Backup{}, closeErr
	}
	if err = os.Rename(temp, destination); err != nil {
		return Backup{}, err
	}
	status, err := m.list()
	if err != nil {
		return Backup{}, err
	}
	for _, old := range status.Backups[minimum(len(status.Backups), RetainedBackups):] {
		if err = os.Remove(filepath.Join(dir, old.ID)); err != nil {
			return Backup{}, err
		}
	}
	for _, b := range status.Backups {
		if b.ID == id {
			return b, nil
		}
	}
	return Backup{}, fmt.Errorf("backup could not be located")
}
func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func (m *Manager) Create() (Backup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, e := m.create("manual")
	m.record(e)
	return b, e
}
func (m *Manager) record(err error) {
	if err == nil {
		m.lastError = ""
	} else {
		m.lastError = "A backup could not be saved. Check available disk space and folder permissions, then try again."
		slog.Error("backup failed", "error", err)
	}
}
func (m *Manager) EnsureAutomatic(now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	status, err := m.list()
	if err == nil {
		for _, b := range status.Backups {
			if b.Reason == "automatic" && now.Sub(b.CreatedAt) >= 0 && now.Sub(b.CreatedAt) < 24*time.Hour {
				return nil
			}
		}
	}
	if err == nil {
		_, err = m.create("automatic")
	}
	m.record(err)
	return err
}
func (m *Manager) Run(ctx context.Context) {
	_ = m.EnsureAutomatic(time.Now())
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = m.EnsureAutomatic(now)
		}
	}
}
func (m *Manager) BackupPath(id string) (string, error) {
	if !managedName.MatchString(id) {
		return "", fmt.Errorf("backup not found")
	}
	dir, err := m.directory()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, id)
	info, err := os.Lstat(p)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("backup not found")
	}
	return p, nil
}
func (m *Manager) stage(reader io.Reader) (db.RecoveryPreview, error) {
	dir, err := m.directory()
	if err != nil {
		return db.RecoveryPreview{}, err
	}
	dir = filepath.Join(dir, "imports")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return db.RecoveryPreview{}, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return db.RecoveryPreview{}, err
	}
	active := 0
	for _, entry := range files {
		if !importID.MatchString(strings.TrimSuffix(entry.Name(), ".db")) || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			continue
		}
		if time.Since(info.ModTime()) > time.Hour {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		} else {
			active++
		}
	}
	if active >= 5 {
		return db.RecoveryPreview{}, fmt.Errorf("Five restore previews are already waiting. Restore one or try again in an hour.")
	}
	id := ulid.Make().String()
	filename := filepath.Join(dir, id+".db")
	// #nosec G304 -- fresh ULID in the private staging directory; exclusive create rejects existing files.
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return db.RecoveryPreview{}, err
	}
	n, err := io.Copy(file, io.LimitReader(reader, MaxUpload+1))
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if n > MaxUpload {
		err = fmt.Errorf("choose a backup smaller than 32 MB")
	}
	if err != nil {
		_ = os.Remove(filename)
		return db.RecoveryPreview{}, err
	}
	snapshot, err := db.ReadBackup(filename)
	if err != nil {
		_ = os.Remove(filename)
		return db.RecoveryPreview{}, err
	}
	return snapshot.Preview(id), nil
}
func (m *Manager) Preview(reader io.Reader) (db.RecoveryPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stage(reader)
}
func (m *Manager) PreviewSaved(id string) (db.RecoveryPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, e := m.BackupPath(id)
	if e != nil {
		return db.RecoveryPreview{}, e
	}
	// #nosec G304 -- BackupPath validates the managed filename and rejects symlinks and non-regular files.
	f, e := os.Open(p)
	if e != nil {
		return db.RecoveryPreview{}, e
	}
	defer f.Close()
	return m.stage(f)
}
func (m *Manager) Restore(id string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !importID.MatchString(id) {
		return nil, fmt.Errorf("choose a backup to preview first")
	}
	if ids, e := m.store.RecoveryReceipt(id); e != nil || ids != nil {
		return ids, e
	}
	dir, e := m.directory()
	if e != nil {
		return nil, e
	}
	filename := filepath.Join(dir, "imports", id+".db")
	info, e := os.Lstat(filename)
	if e != nil || !info.Mode().IsRegular() || time.Since(info.ModTime()) > time.Hour {
		return nil, fmt.Errorf("This preview expired. Choose your backup again.")
	}
	snapshot, e := db.ReadBackup(filename)
	if e != nil {
		return nil, e
	}
	if _, e = m.create("before-restore"); e != nil {
		m.record(e)
		return nil, fmt.Errorf("Could not save a safety backup. Your boards have not changed.")
	}
	ids, e := m.store.RestoreBoards(id, snapshot)
	if e == nil {
		_ = os.Remove(filename)
	}
	return ids, e
}

func (m *Manager) CancelPreview(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !importID.MatchString(id) {
		return fmt.Errorf("invalid preview")
	}
	dir, err := m.directory()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, "imports", id+".db"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
