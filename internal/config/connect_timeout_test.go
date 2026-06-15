package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestResolveConnectTimeout(t *testing.T) {
	orig := viper.Get("api.connection_timeout")
	t.Cleanup(func() { viper.Set("api.connection_timeout", orig) })

	viper.Set("api.connection_timeout", 8) // global

	cases := []struct {
		name   string
		nested map[string]interface{}
		want   time.Duration
	}{
		{"per-API override beats global", map[string]interface{}{"connection_timeout": 2}, 2 * time.Second},
		{"falls back to global", map[string]interface{}{}, 8 * time.Second},
		{"non-positive per-API falls back to global", map[string]interface{}{"connection_timeout": 0}, 8 * time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveConnectTimeout(c.nested); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}

	t.Run("built-in 5s when global unset", func(t *testing.T) {
		viper.Set("api.connection_timeout", 0)
		if got := resolveConnectTimeout(map[string]interface{}{}); got != 5*time.Second {
			t.Errorf("got %v, want 5s", got)
		}
	})
}
