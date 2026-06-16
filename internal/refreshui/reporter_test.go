package refreshui

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestLinearReporterStageLineIsAtomic(t *testing.T) {
	var buf bytes.Buffer
	r := NewLinearWriter(&buf)

	r.APIStart("mist", "mist", "")
	r.Stage("mist", "Fetching sites")
	r.Progress("mist", 1, 1) // no-op for linear; must not split the line
	r.StageResult("mist", "5 sites")
	r.APIDone("mist", 952*time.Millisecond)

	want := "  [mist] Refreshing mist API...\n" +
		"    Fetching sites... 5 sites\n" +
		"  [mist] Complete in 952ms\n"
	if got := buf.String(); got != want {
		t.Fatalf("linear output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestLinearReporterSiteAndEmptyResult(t *testing.T) {
	var buf bytes.Buffer
	r := NewLinearWriter(&buf)

	r.APIStart("meraki", "meraki", "US-LAB-01")
	r.Stage("meraki", "Skipping device configs (use 'refresh cache' to fetch)")
	r.StageResult("meraki", "")
	r.APIError("meraki", errors.New("login timeout"))

	want := "  [meraki] Refreshing meraki API (site US-LAB-01)...\n" +
		"    Skipping device configs (use 'refresh cache' to fetch)...\n" +
		"  [meraki] Failed: login timeout\n"
	if got := buf.String(); got != want {
		t.Fatalf("linear output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestResolveNilFallsBackToLinear(t *testing.T) {
	if Resolve(nil) == nil {
		t.Fatal("Resolve(nil) returned nil; want shared linear reporter")
	}
	r := NewLinearWriter(&bytes.Buffer{})
	if Resolve(r) != r {
		t.Fatal("Resolve(r) should return r unchanged")
	}
}
