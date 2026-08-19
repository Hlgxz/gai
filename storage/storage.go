package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Disk is a filesystem abstraction (local, S3-compatible, etc.).
type Disk interface {
	Put(path string, r io.Reader) error
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
	Exists(path string) (bool, error)
	URL(path string) string
}

// Local stores files under Root and optionally prefixes public URLs with BaseURL.
type Local struct {
	Root    string
	BaseURL string
}

func (l *Local) abs(path string) (string, error) {
	clean := filepath.Clean("/" + path)
	full := filepath.Join(l.Root, strings.TrimPrefix(clean, string(filepath.Separator)))
	rel, err := filepath.Rel(l.Root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("gai/storage: path escapes root")
	}
	return full, nil
}

func (l *Local) Put(path string, r io.Reader) error {
	full, err := l.abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Get(path string) (io.ReadCloser, error) {
	full, err := l.abs(path)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

func (l *Local) Delete(path string) error {
	full, err := l.abs(path)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

func (l *Local) Exists(path string) (bool, error) {
	full, err := l.abs(path)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(full)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (l *Local) URL(path string) string {
	p := strings.TrimPrefix(path, "/")
	if l.BaseURL == "" {
		return "/" + p
	}
	return strings.TrimSuffix(l.BaseURL, "/") + "/" + p
}

// Manager holds named disks.
type Manager struct {
	disks       map[string]Disk
	defaultName string
}

// NewManager creates a disk manager. The first added disk becomes default.
func NewManager() *Manager {
	return &Manager{disks: make(map[string]Disk)}
}

func (m *Manager) Add(name string, d Disk) {
	m.disks[name] = d
	if m.defaultName == "" {
		m.defaultName = name
	}
}

func (m *Manager) Disk(name ...string) (Disk, error) {
	key := m.defaultName
	if len(name) > 0 && name[0] != "" {
		key = name[0]
	}
	d, ok := m.disks[key]
	if !ok {
		return nil, fmt.Errorf("gai/storage: disk %q not registered", key)
	}
	return d, nil
}
