package utils

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// Basic test for FormatOutputWithWarning without mocking
func TestFormatOutputWithWarning_Basic(t *testing.T) {
	testText := "Test output"
	result := FormatOutputWithWarning(testText)
	// We can't control the cache integrity state in tests without a mock,
	// so we just check that the output is either the original text or
	// it contains the original text if a warning was added
	if !strings.Contains(result, testText) {
		t.Errorf("FormatOutputWithWarning(%s) = %s, which doesn't contain the original text", testText, result)
	}
}

// Basic test for PrintWithWarning without mocking
func TestPrintWithWarning_Basic(t *testing.T) {
	// Create testable environment
	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Restore everything after the test
	defer func() {
		os.Stdout = oldStdout
	}()

	// Test case
	testFormat := "Test %s %d"
	testArgs := []interface{}{"output", 123}
	expectedText := "Test output 123"

	// Create a new pipe to capture output
	r, w, _ := os.Pipe()
	oldStdout = os.Stdout
	os.Stdout = w

	// Call the function being tested
	PrintWithWarning(testFormat, testArgs...)

	// Close the pipe and restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Check the output contains the expected string
	// We can't control the warning state, so just check for the basic text
	if !strings.Contains(output, expectedText) {
		t.Errorf("PrintWithWarning(%s, %v) output = %q, which doesn't contain the expected text %q",
			testFormat, testArgs, output, expectedText)
	}
}

// Basic test for PrintDetailWithWarning without mocking
func TestPrintDetailWithWarning_Basic(t *testing.T) {
	// Create testable environment
	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Restore everything after the test
	defer func() {
		os.Stdout = oldStdout
	}()

	// Test case
	testFormat := "Test %s %d"
	testArgs := []interface{}{"output", 123}
	expectedText := "Test output 123"

	// Create a new pipe to capture output
	r, w, _ := os.Pipe()
	oldStdout = os.Stdout
	os.Stdout = w

	// Call the function being tested
	PrintDetailWithWarning(testFormat, testArgs...)

	// Close the pipe and restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Check the output contains the expected string
	// We can't control the warning state, so just check for the basic text
	if !strings.Contains(output, expectedText) {
		t.Errorf("PrintDetailWithWarning(%s, %v) output = %q, which doesn't contain the expected text %q",
			testFormat, testArgs, output, expectedText)
	}
}
