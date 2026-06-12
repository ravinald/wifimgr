// Command gen writes the embedded schemas out to docs/schemas so the published
// mirror stays byte-identical to the in-binary source. Run via `go generate
// ./internal/schemadefs` (or `go generate ./...`); TestEmbeddedMatchesPublished
// fails if the mirror drifts.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ravinald/wifimgr/internal/schemadefs"
)

// outDir is relative to internal/schemadefs, where go generate runs this.
const outDir = "../../docs/schemas"

func main() {
	names := schemadefs.Names()
	if len(names) == 0 {
		log.Fatal("gen: no embedded schemas found")
	}
	for _, name := range names {
		data, err := schemadefs.Read(name)
		if err != nil {
			log.Fatalf("gen: read %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(outDir, name), data, 0600); err != nil {
			log.Fatalf("gen: write %s: %v", name, err)
		}
	}
}
