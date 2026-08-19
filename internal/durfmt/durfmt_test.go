package durfmt

import (
	"strings"
	"testing"
	"time"
)

func TestElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{-5 * time.Second, "0ms"},
		{0, "0ms"},
		{748 * time.Millisecond, "748ms"},
		{999 * time.Millisecond, "999ms"},
		{time.Second, "1s"},
		{2486 * time.Millisecond, "2.5s"},
		{59500 * time.Millisecond, "59.5s"},
		{time.Minute, "1m0s"},
		{269559 * time.Millisecond, "4m30s"},
		{2 * time.Hour, "2h0m0s"},
	}
	for _, c := range cases {
		if got := Elapsed(c.d); got != c.want {
			t.Errorf("Elapsed(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{-time.Hour, "0s"},
		{0, "0s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m"},
		{15*time.Minute + 20*time.Second, "15m20s"},
		{time.Hour, "1h"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
		{2*time.Hour + 30*time.Minute + 59*time.Second, "2h30m"},
		{24 * time.Hour, "1d"},
		{3*24*time.Hour + 4*time.Hour, "3d4h"},
		{30 * 24 * time.Hour, "30d"},
	}
	for _, c := range cases {
		if got := Age(c.d); got != c.want {
			t.Errorf("Age(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestAgeSince(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		in     time.Time
		want   string
		prefix bool // clock advances mid-test, so only the leading unit is stable
	}{
		{name: "zero time renders dash", in: time.Time{}, want: "—"},
		{name: "future time renders dash", in: now.Add(5 * time.Minute), want: "—"},
		{name: "minutes", in: now.Add(-15 * time.Minute), want: "15m", prefix: true},
		{name: "hours and minutes", in: now.Add(-(2*time.Hour + 30*time.Minute)), want: "2h30m", prefix: true},
		{name: "days and hours", in: now.Add(-(3*24*time.Hour + 4*time.Hour)), want: "3d4h", prefix: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AgeSince(c.in)
			if c.prefix {
				if !strings.HasPrefix(got, c.want) {
					t.Errorf("AgeSince = %q, want prefix %q", got, c.want)
				}
				return
			}
			if got != c.want {
				t.Errorf("AgeSince = %q, want %q", got, c.want)
			}
		})
	}
}
