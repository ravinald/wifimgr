package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestRefreshStampAndParens(t *testing.T) {
	now := time.Now()

	t.Run("healthy success only", func(t *testing.T) {
		success := now.Add(-9 * time.Hour)
		stamp, parens := refreshStampAndParens(success, time.Time{}, "")
		if stamp != success.Format("2006-01-02 15:04:05") {
			t.Errorf("stamp = %q", stamp)
		}
		if !strings.HasSuffix(parens, "ago") || strings.Contains(parens, ",") {
			t.Errorf("healthy parens should be plain '<dur> ago', got %q", parens)
		}
	})

	t.Run("failing after a prior success", func(t *testing.T) {
		success := now.Add(-2 * 24 * time.Hour)
		failure := now.Add(-3 * time.Minute)
		stamp, parens := refreshStampAndParens(success, failure, "connection failure")
		if stamp != success.Format("2006-01-02 15:04:05") {
			t.Errorf("stamp should be the last success, got %q", stamp)
		}
		if !strings.Contains(parens, "connection failure") {
			t.Errorf("parens should name the failure, got %q", parens)
		}
	})

	t.Run("never succeeded but failed", func(t *testing.T) {
		failure := now.Add(-1 * time.Minute)
		stamp, parens := refreshStampAndParens(time.Time{}, failure, "")
		if stamp != failure.Format("2006-01-02 15:04:05") {
			t.Errorf("stamp should fall back to failure time, got %q", stamp)
		}
		if !strings.Contains(parens, "never succeeded") || !strings.Contains(parens, "refresh failed") {
			t.Errorf("parens = %q", parens)
		}
	})
}
