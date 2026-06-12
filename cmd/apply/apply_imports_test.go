package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
)

// importEnvelope mirrors what `wifimgr import ... save` writes: a Config.Sites
// section plus import-only source/templates keys that apply must ignore.
const importEnvelope = `{
  "version": 1,
  "source": {"api": "mist", "site": "ZZ-TMP-SITE"},
  "config": {"sites": {"ZZ-TMP-SITE": {"site_config": {"name": "ZZ-TMP-SITE"}, "devices": {"ap": {}}}}},
  "templates": {"wlan": {"zz-tmp--ssid": {"ssid": "ZZ"}}}
}`

func TestGetSiteConfigsFromFilesParsesImportEnvelope(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "imp.json"), []byte(importEnvelope), 0600); err != nil {
		t.Fatalf("write envelope: %v", err)
	}

	got, err := getSiteConfigsFromFiles([]string{"imp.json"})
	if err != nil {
		t.Fatalf("getSiteConfigsFromFiles: %v", err)
	}
	if _, ok := got["ZZ-TMP-SITE"]; !ok {
		t.Fatalf("import envelope site not extracted; got %v", keysOf(got))
	}
}

func TestSiteConfigFilesIncludesImports(t *testing.T) {
	cfg := &config.Config{}
	cfg.Files.SiteConfigs = []string{"a.json"}
	cfg.Files.Imports = []string{"test/b.json"}

	files := siteConfigFiles(cfg)
	if len(files) != 2 || files[0] != "a.json" || files[1] != "test/b.json" {
		t.Errorf("siteConfigFiles = %v, want [a.json test/b.json]", files)
	}
}

func keysOf(m map[string]SiteConfig) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
