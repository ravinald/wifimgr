package config

import "testing"

func TestValidAPILabel(t *testing.T) {
	valid := []string{"mist", "meraki", "lab-01", "prod_us", "v2.api", "Mist"}
	for _, label := range valid {
		if !validAPILabel(label) {
			t.Errorf("validAPILabel(%q) = false, want true", label)
		}
	}

	// Each of these would escape apis/<label>.json if it reached filepath.Join.
	invalid := []string{"", ".", "..", "../evil", "../../etc/cron.d/x", "a/b", `a\b`, "with space", "name;rm"}
	for _, label := range invalid {
		if validAPILabel(label) {
			t.Errorf("validAPILabel(%q) = true, want false", label)
		}
	}
}
