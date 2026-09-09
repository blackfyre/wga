// Package generatedpublication atomically selects one complete generated
// catalogue publication shared by sitemap and agent-content readers.
package generatedpublication

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/sys/unix"
)

const (
	directoryName = "generated-publications"
	versionsName  = "versions"
	currentName   = "current"
	lockName      = "publication.lock"
)

// ErrNoCurrentPublication reports that no shared publication has been selected
// yet. Readers may use a compatible legacy publication during migration only
// for this state, not for a transient race involving an existing marker.
var ErrNoCurrentPublication = errors.New("no current generated publication")

// Lock serializes generation against every process sharing an application data
// directory. Closing it releases the advisory filesystem lock.
type Lock struct {
	file *os.File
}

// AcquireLock obtains the cross-process lock that must cover staging,
// publication selection, and pruning as one lifecycle.
func AcquireLock(app core.App) (*Lock, error) {
	root := Directory(app)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create generated-publication directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(root, lockName), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open generated-publication lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock generated publication: %w", err)
	}
	return &Lock{file: file}, nil
}

// Close releases the cross-process publication lock.
func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	return errors.Join(unlockErr, closeErr)
}

func Directory(app core.App) string {
	return filepath.Join(app.DataDir(), directoryName)
}

func NewStaging(app core.App) (string, error) {
	versions := filepath.Join(Directory(app), versionsName)
	if err := os.MkdirAll(versions, 0o755); err != nil {
		return "", fmt.Errorf("create generated-publication directory: %w", err)
	}
	staging, err := os.MkdirTemp(versions, ".staging-")
	if err != nil {
		return "", fmt.Errorf("create generated-publication staging directory: %w", err)
	}
	return staging, nil
}

func CurrentDirectory(app core.App) (string, error) {
	root := Directory(app)
	data, err := os.ReadFile(filepath.Join(root, currentName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %w", ErrNoCurrentPublication, err)
		}
		return "", fmt.Errorf("read current generated publication: %w", err)
	}
	version := strings.TrimSpace(string(data))
	if !validVersion(version) {
		return "", fmt.Errorf("invalid current generated publication")
	}
	current := filepath.Join(root, versionsName, version)
	info, err := os.Stat(current)
	if err != nil {
		return "", fmt.Errorf("stat current generated publication: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("current generated publication is not a directory")
	}
	return current, nil
}

// Publish atomically makes one fully staged directory visible to every
// generated-resource reader through a single current-version marker.
func Publish(app core.App, staging string, version string) (string, error) {
	if !validVersion(version) {
		return "", fmt.Errorf("invalid generated-publication version")
	}
	root := Directory(app)
	published := filepath.Join(root, versionsName, version)
	if err := os.Rename(staging, published); err != nil {
		return "", fmt.Errorf("publish generated resources: %w", err)
	}
	if err := selectVersion(root, version); err != nil {
		_ = os.RemoveAll(published)
		return "", err
	}
	return published, nil
}

func Prune(app core.App, current string) error {
	versions := filepath.Join(Directory(app), versionsName)
	entries, err := os.ReadDir(versions)
	if err != nil {
		return fmt.Errorf("list generated publications: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == current || !entry.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(versions, entry.Name())); err != nil {
			return fmt.Errorf("remove stale generated publication: %w", err)
		}
	}
	return nil
}

func selectVersion(root string, version string) error {
	marker, err := os.CreateTemp(root, ".current-")
	if err != nil {
		return fmt.Errorf("create generated-publication marker: %w", err)
	}
	markerPath := marker.Name()
	defer os.Remove(markerPath)
	if _, err := marker.WriteString(version + "\n"); err != nil {
		_ = marker.Close()
		return fmt.Errorf("write generated-publication marker: %w", err)
	}
	if err := marker.Sync(); err != nil {
		_ = marker.Close()
		return fmt.Errorf("sync generated-publication marker: %w", err)
	}
	if err := marker.Close(); err != nil {
		return fmt.Errorf("close generated-publication marker: %w", err)
	}
	if err := os.Rename(markerPath, filepath.Join(root, currentName)); err != nil {
		return fmt.Errorf("select generated publication: %w", err)
	}
	return nil
}

func validVersion(version string) bool {
	return version != "" && filepath.Base(version) == version && version != "." && version != ".."
}
