package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestWriteFileAtomic_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	want := []byte(`{"hello":"world"}`)
	if err := WriteFileAtomic(path, want, 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path) // #nosec G304 -- test path under TempDir
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("content mismatch: got %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("perm = %o, want 0644", info.Mode().Perm())
	}
}

func TestWriteFileAtomic_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := WriteFileAtomic(path, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	got, err := os.ReadFile(path) // #nosec G304 -- test path under TempDir
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
}

func TestWriteFileAtomic_NoOrphanTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	if err := WriteFileAtomic(path, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("orphan temp file left behind: %s", e.Name())
		}
	}
}

func TestWriteFileAtomic_FailsOnMissingDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent-subdir", "out.json")

	err := WriteFileAtomic(path, []byte("x"), 0644)
	if err == nil {
		t.Fatal("expected error for missing parent directory, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) && !strings.Contains(err.Error(), "no such file") {
		// Accept either the wrapped sentinel or the OS message — behaviour
		// varies across platforms.
		t.Logf("error (informational): %v", err)
	}
}

func TestWriteFileAtomic_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	for i := range N {
		go func(i int) {
			defer wg.Done()
			payload := []byte(strings.Repeat("x", i+1))
			if err := WriteFileAtomic(path, payload, 0644); err != nil {
				t.Errorf("goroutine %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// One of the N writes must have landed; the file must be a valid run of
	// x's with length 1..N (no corruption from interleaved writes).
	got, err := os.ReadFile(path) // #nosec G304 -- test path under TempDir
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(got) < 1 || len(got) > N {
		t.Errorf("final length %d out of range [1,%d]", len(got), N)
	}
	if strings.Trim(string(got), "x") != "" {
		t.Errorf("corrupt content (non-x bytes present): %q", got)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("orphan temp file after concurrent writes: %s", e.Name())
		}
	}
}
