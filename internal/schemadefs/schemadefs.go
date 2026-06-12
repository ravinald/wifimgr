// Package schemadefs embeds wifimgr's JSON Schemas into the binary so schema
// validation never depends on an out-of-band install. The embedded copies are
// the canonical source; docs/schemas mirrors them for publication, and a test
// keeps the two byte-identical.
package schemadefs

import (
	"embed"
	"io/fs"
)

//go:generate go run ./gen

//go:embed *.json
var fsys embed.FS

// Read returns the embedded schema by file name (e.g. "site-config-schema.json").
func Read(name string) ([]byte, error) {
	return fsys.ReadFile(name)
}

// Has reports whether a schema with the given file name is embedded.
func Has(name string) bool {
	_, err := fsys.Open(name)
	return err == nil
}

// Names returns the embedded schema file names.
func Names() []string {
	entries, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil
	}
	return entries
}
