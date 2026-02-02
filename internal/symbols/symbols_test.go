package symbols

import (
	"strings"
	"testing"
)

func TestStatusPrefixes(t *testing.T) {
	success := SuccessPrefix()
	failure := FailurePrefix()

	// Check that prefixes are not empty
	if success == "" {
		t.Errorf("SuccessPrefix() should not be empty")
	}
	if failure == "" {
		t.Errorf("FailurePrefix() should not be empty")
	}

	// Should contain expected text (possibly with color formatting)
	if !strings.Contains(success, "OK") {
		t.Errorf("SuccessPrefix() should contain 'OK', got: %q", success)
	}
	if !strings.Contains(failure, "FAIL") {
		t.Errorf("FailurePrefix() should contain 'FAIL', got: %q", failure)
	}
}

func TestTextFormatting(t *testing.T) {
	// Test that text formatting functions work
	greenText := GreenText("C")
	redText := RedText("D")
	blueText := BlueText("?")

	// Check that text functions are not empty
	if greenText == "" {
		t.Errorf("GreenText() should not be empty")
	}
	if redText == "" {
		t.Errorf("RedText() should not be empty")
	}
	if blueText == "" {
		t.Errorf("BlueText() should not be empty")
	}

	// Should contain the expected text (possibly with color formatting)
	if !strings.Contains(greenText, "C") {
		t.Errorf("GreenText('C') should contain 'C', got: %q", greenText)
	}
	if !strings.Contains(redText, "D") {
		t.Errorf("RedText('D') should contain 'D', got: %q", redText)
	}
	if !strings.Contains(blueText, "?") {
		t.Errorf("BlueText('?') should contain '?', got: %q", blueText)
	}

	// Functions should be callable without panic
	t.Logf("Green: %q", greenText)
	t.Logf("Red: %q", redText)
	t.Logf("Blue: %q", blueText)
}

func TestFormatBooleanValue(t *testing.T) {
	// Test connection field formatting
	connectedTrue := FormatBooleanValue(true, true)
	connectedFalse := FormatBooleanValue(false, true)

	// Test regular boolean field formatting
	regularTrue := FormatBooleanValue(true, false)
	regularFalse := FormatBooleanValue(false, false)

	// Connection fields should use C/D (possibly colored)
	validConnTrue := connectedTrue == "C" || strings.Contains(connectedTrue, "C")
	validConnFalse := connectedFalse == "D" || strings.Contains(connectedFalse, "D")

	// Regular fields should use Yes/No
	if regularTrue != "Yes" {
		t.Errorf("Regular boolean true should return 'Yes', got: %q", regularTrue)
	}
	if regularFalse != "No" {
		t.Errorf("Regular boolean false should return 'No', got: %q", regularFalse)
	}

	if !validConnTrue {
		t.Errorf("Connection field true should return 'C' or contain C, got: %q", connectedTrue)
	}
	if !validConnFalse {
		t.Errorf("Connection field false should return 'D' or contain D, got: %q", connectedFalse)
	}
}
