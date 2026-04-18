package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path via a temp file in the same directory
// followed by an atomic rename. The rename is atomic on POSIX filesystems; a
// crash mid-write cannot leave path in a partially-written state. The temp
// file is removed on error so the directory does not accumulate orphans.
//
// The temp file is created with 0600 (os.CreateTemp default). perm is applied
// via os.Chmod before the rename.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write: create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = os.Remove(tmpPath) // #nosec G703 -- best-effort orphan cleanup
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("atomic write: write temp file %s: %w", tmpPath, err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("atomic write: sync temp file %s: %w", tmpPath, err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("atomic write: close temp file %s: %w", tmpPath, err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		cleanup()
		return fmt.Errorf("atomic write: chmod temp file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("atomic write: rename %s to %s: %w", tmpPath, path, err)
	}

	return nil
}
