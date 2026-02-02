package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Logger configuration constants
const (
	// Log format types
	FormatText = "text"
	FormatJSON = "json"

	// Environment variable names
	EnvLogLevel  = "MIST_CLI_LOG_LEVEL"
	EnvLogFormat = "MIST_CLI_LOG_FORMAT"
)

// LogConfig represents logging configuration settings.
type LogConfig struct {
	Enable   bool   // logging enabled
	Level    string // debug, info, warn, or error
	Format   string // text or json
	ToStdout bool   // also log to stdout
	LogFile  string // path to log file (empty = no file logging)
	Silent   bool   // suppress logging destination message
}

var (
	// The main logger instance
	defaultLogger = logrus.New()

	// For site and org name formatting
	siteNameLookup SiteNameLookupFunc
	orgNameLookup  OrgNameLookupFunc

	// Current log file
	logFile *os.File

	// Original stdout/stderr for restoration when needed
	originalStdout = os.Stdout
)

// SiteNameLookupFunc is a function type for looking up site names from site IDs
type SiteNameLookupFunc func(siteID string) (string, bool)

// OrgNameLookupFunc is a function type for looking up organization names from org IDs
type OrgNameLookupFunc func(orgID string) (string, bool)

// init initializes the default logger with basic settings
func init() {
	// Set default configuration
	defaultLogger.SetOutput(os.Stdout)
	defaultLogger.SetLevel(logrus.InfoLevel)
	defaultLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

}

// SetSiteNameLookupFunc sets the function to be used for site name lookups
func SetSiteNameLookupFunc(fn SiteNameLookupFunc) {
	siteNameLookup = fn
}

// SetOrgNameLookupFunc sets the function to be used for organization name lookups
func SetOrgNameLookupFunc(fn OrgNameLookupFunc) {
	orgNameLookup = fn
}

// FormatSiteID formats a site ID with its name if available
func FormatSiteID(siteID string) string {
	if siteNameLookup != nil {
		if siteName, found := siteNameLookup(siteID); found {
			return fmt.Sprintf("%s (%s)", siteName, siteID)
		}
	}
	return siteID
}

// FormatOrgID formats an organization ID with its name if available
func FormatOrgID(orgID string) string {
	if orgNameLookup != nil {
		if orgName, found := orgNameLookup(orgID); found {
			return fmt.Sprintf("%s (%s)", orgName, orgID)
		}
	}
	return orgID
}

// NOTE: Original stdout/stderr handles are declared globally at the top of the file

// ConfigureLogging configures the logging system according to the given configuration
// This is the main entry point for setting up logging in the application
func ConfigureLogging(config LogConfig) error {
	// Check environment variables first (they override explicit parameters)
	if envLevel := os.Getenv(EnvLogLevel); envLevel != "" {
		config.Level = envLevel
	}

	if envFormat := os.Getenv(EnvLogFormat); envFormat != "" {
		config.Format = envFormat
	}

	// Ensure we have valid defaults
	if config.Level == "" {
		config.Level = "info"
	}

	if config.Format == "" {
		config.Format = FormatText
	}

	// Convert log level string to logrus level
	level := GetLogLevelFromString(config.Level)

	// Set the log level
	defaultLogger.SetLevel(level)

	// Set the log formatter
	if config.Format == FormatJSON {
		defaultLogger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		defaultLogger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Start with a clean slate by closing any existing log file
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	// We should NOT modify os.Stdout or os.Stderr
	// Doing so will break applications that need direct terminal control,
	// like TUI applications

	// If we have a logfile requested, set up the log destination
	if config.LogFile != "" {
		// Ensure the log directory exists
		dir := filepath.Dir(config.LogFile)
		if dir != "" && dir != "." {
			// Create directory if it doesn't exist (XDG state dir may not exist yet)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create log directory %s: %w", dir, err)
			}
		}

		// Open log file
		var err error
		logFile, err = os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		if config.ToStdout {
			// Log to both file and stdout
			defaultLogger.SetOutput(io.MultiWriter(originalStdout, logFile))
			// Keep stdout/stderr as they are - pointing to original outputs
		} else {
			// Log ONLY to file - don't redirect stdout/stderr
			// IMPORTANT: We configure the logger only, not OS-level stdout/stderr
			// This allows normal program output (like UI rendering) to still work
			defaultLogger.SetOutput(logFile)
		}
	} else {
		// No log file specified, use stdout only
		defaultLogger.SetOutput(originalStdout)
	}

	// Log destination message (will go to the configured destination) unless silenced
	if !config.Silent {
		if config.LogFile != "" && config.ToStdout {
			Infof("Logging to both file %s and stdout", config.LogFile)
		} else if config.LogFile != "" && !config.ToStdout {
			Infof("Logging to file only: %s", config.LogFile)
		} else if config.ToStdout && config.LogFile == "" {
			Infof("Logging to stdout only")
		} else {
			Infof("Logging disabled")
		}
	}

	return nil
}

// GetLogLevelFromString converts a string level to a logrus.Level
func GetLogLevelFromString(level string) logrus.Level {
	switch level {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

// GetLogger returns the default logger
func GetLogger() *logrus.Logger {
	return defaultLogger
}

// Cleanup closes log files and performs any necessary cleanup
func Cleanup() {
	// Note: We no longer need to restore stdout/stderr since we don't change them

	// Close any open log file
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}

// Debug logs a message at the debug level
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

// Debugf logs a formatted message at the debug level
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info logs a message at the info level
func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

// Infof logs a formatted message at the info level
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn logs a message at the warn level
func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

// Warnf logs a formatted message at the warn level
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error logs a message at the error level
func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

// Errorf logs a formatted message at the error level
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatalf logs a formatted message at the fatal level and then calls os.Exit(1)
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// ConfigureLogger configures the logger with a simplified interface
// This is an alias for ConfigureLogging to maintain backward compatibility with existing code
func ConfigureLogger(level string, toStdout bool) error {
	config := LogConfig{
		Enable:   true,
		Level:    level,
		Format:   "text", // Default to text format
		ToStdout: toStdout,
		LogFile:  "", // No log file by default
	}

	return ConfigureLogging(config)
}

// SetLogger sets the default logger to a provided logger instance
// This is primarily used for testing
func SetLogger(logger *logrus.Logger) *logrus.Logger {
	old := defaultLogger
	defaultLogger = logger
	return old
}
