package apply

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// DisplayConfigWarnings formats and displays configuration warnings/errors nicely.
// Returns true if there are critical errors that should abort the operation.
func DisplayConfigWarnings(warnings []error, deviceMAC string) bool {
	if len(warnings) == 0 {
		return false
	}

	hasCriticalErrors := false

	// Separate errors by type
	var fieldMappingErrors []*vendors.FieldMappingError
	var unexpectedFields []*vendors.UnexpectedFieldWarning
	var missingFields []*vendors.MissingFieldWarning
	var otherErrors []error

	for _, w := range warnings {
		switch err := w.(type) {
		case *vendors.FieldMappingError:
			fieldMappingErrors = append(fieldMappingErrors, err)
			hasCriticalErrors = true // Field type mismatches are critical
		case *vendors.UnexpectedFieldWarning:
			unexpectedFields = append(unexpectedFields, err)
		case *vendors.MissingFieldWarning:
			missingFields = append(missingFields, err)
		default:
			otherErrors = append(otherErrors, err)
			hasCriticalErrors = true // Unknown errors are treated as critical
		}
	}

	// Display critical errors first
	if len(fieldMappingErrors) > 0 {
		fmt.Printf("\nCritical Configuration Errors for %s:\n", deviceMAC)
		fmt.Println("==================================================")
		for _, err := range fieldMappingErrors {
			fmt.Println(err.UserMessage())
			fmt.Println()
		}
	}

	// Display other errors
	if len(otherErrors) > 0 {
		fmt.Printf("\nConfiguration Errors for %s:\n", deviceMAC)
		for _, err := range otherErrors {
			fmt.Printf("  - %s\n", err.Error())
		}
		fmt.Println()
	}

	// Display warnings (non-critical)
	if len(unexpectedFields) > 0 || len(missingFields) > 0 {
		fmt.Printf("\nConfiguration Warnings for %s:\n", deviceMAC)
		fmt.Println("-----------------------------------------------")

		// Check if we're in debug mode by checking the logger's level
		logger := logging.GetLogger()
		isDebug := logger.GetLevel() <= logrus.DebugLevel

		for _, warn := range unexpectedFields {
			logging.Debugf("Unexpected field warning: %s", warn.Error())
			if isDebug {
				fmt.Println(warn.UserMessage())
				fmt.Println()
			} else {
				// Concise version for non-debug mode
				fmt.Printf("  - Unexpected field: %s (value: %v)\n", warn.Field, warn.Value)
			}
		}

		for _, warn := range missingFields {
			logging.Debugf("Missing field warning: %s", warn.Error())
			if isDebug {
				fmt.Println(warn.UserMessage())
				fmt.Println()
			} else {
				// Concise version for non-debug mode
				fmt.Printf("  - Missing expected field: %s\n", warn.Field)
			}
		}

		if !isDebug {
			fmt.Println("  (Run with --debug for more details)")
		}
		fmt.Println()
	}

	return hasCriticalErrors
}

// DisplayConfigValidationErrors displays vendor validation errors from ValidateForVendor.
// Returns true if there are errors that should abort the operation.
func DisplayConfigValidationErrors(errors []error, deviceMAC string, vendorName string) bool {
	if len(errors) == 0 {
		return false
	}

	fmt.Printf("\nConfiguration Validation Errors for %s (%s vendor):\n", deviceMAC, vendorName)
	fmt.Println("==================================================")

	for _, err := range errors {
		if configErr, ok := err.(*vendors.ConfigValidationError); ok {
			fmt.Printf("  Field: %s\n", configErr.Field)
			fmt.Printf("  Issue: %s\n\n", configErr.Message)
		} else {
			fmt.Printf("  - %s\n\n", err.Error())
		}
	}

	fmt.Println("Suggested Actions:")
	fmt.Println("  1. Review your device configuration file")
	fmt.Println("  2. Remove vendor-specific blocks that don't match the target vendor")
	fmt.Println("  3. Check field compatibility with the vendor API documentation")
	fmt.Println()

	return len(errors) > 0
}
