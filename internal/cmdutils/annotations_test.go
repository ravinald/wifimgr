package cmdutils

import "testing"

func TestGetCommandTier_NilAnnotations(t *testing.T) {
	tier := GetCommandTier(nil)
	if tier != TierFullAPI {
		t.Errorf("expected TierFullAPI for nil annotations, got %d", tier)
	}
}

func TestGetCommandTier_EmptyAnnotations(t *testing.T) {
	tier := GetCommandTier(map[string]string{})
	if tier != TierFullAPI {
		t.Errorf("expected TierFullAPI for empty annotations, got %d", tier)
	}
}

func TestGetCommandTier_NoInit(t *testing.T) {
	annotations := map[string]string{
		AnnotationNoInit: "true",
	}
	tier := GetCommandTier(annotations)
	if tier != TierNoInit {
		t.Errorf("expected TierNoInit, got %d", tier)
	}
}

func TestGetCommandTier_NeedsConfig(t *testing.T) {
	annotations := map[string]string{
		AnnotationNeedsConfig: "true",
	}
	tier := GetCommandTier(annotations)
	if tier != TierConfigOnly {
		t.Errorf("expected TierConfigOnly, got %d", tier)
	}
}

func TestGetCommandTier_NeedsAPI(t *testing.T) {
	annotations := map[string]string{
		AnnotationNeedsAPI: "true",
	}
	tier := GetCommandTier(annotations)
	if tier != TierFullAPI {
		t.Errorf("expected TierFullAPI, got %d", tier)
	}
}

func TestGetCommandTier_NoInitTakesPrecedence(t *testing.T) {
	// If both no-init and needs-config are set, no-init should win
	annotations := map[string]string{
		AnnotationNoInit:      "true",
		AnnotationNeedsConfig: "true",
	}
	tier := GetCommandTier(annotations)
	if tier != TierNoInit {
		t.Errorf("expected TierNoInit to take precedence, got %d", tier)
	}
}

func TestTierConstants(t *testing.T) {
	// Verify tier ordering
	if TierNoInit >= TierConfigOnly {
		t.Error("TierNoInit should be less than TierConfigOnly")
	}
	if TierConfigOnly >= TierFullAPI {
		t.Error("TierConfigOnly should be less than TierFullAPI")
	}
}
