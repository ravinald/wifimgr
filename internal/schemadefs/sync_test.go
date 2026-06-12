package schemadefs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// publishedSchemaDir is the human-facing mirror, relative to this package.
const publishedSchemaDir = "../../docs/schemas"

// TestEmbeddedMatchesPublished keeps the single source of truth honest: the
// schemas embedded in the binary (this package) must be byte-identical to the
// published copies under docs/schemas. Edit the embedded copy, then refresh the
// mirror — this fails if they diverge, so the two can never drift silently the
// way config/schemas and docs/schemas once did.
func TestEmbeddedMatchesPublished(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("no embedded schemas found")
	}
	for _, name := range names {
		embedded, err := Read(name)
		if err != nil {
			t.Fatalf("read embedded %s: %v", name, err)
		}
		published, err := os.ReadFile(filepath.Join(publishedSchemaDir, name)) //nolint:gosec // repo-relative test path
		if err != nil {
			t.Errorf("published mirror missing %s: %v", name, err)
			continue
		}
		if !bytes.Equal(embedded, published) {
			t.Errorf("schema %s differs between embedded source and docs/schemas; refresh the mirror", name)
		}
	}
}
