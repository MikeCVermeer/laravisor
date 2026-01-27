package log

import (
	"strings"
)

// LogLevel represents a Laravel log level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelNotice
	LogLevelWarning
	LogLevelError
	LogLevelCritical
	LogLevelAlert
	LogLevelEmergency
	LogLevelUnknown
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelNotice:
		return "notice"
	case LogLevelWarning:
		return "warning"
	case LogLevelError:
		return "error"
	case LogLevelCritical:
		return "critical"
	case LogLevelAlert:
		return "alert"
	case LogLevelEmergency:
		return "emergency"
	default:
		return "unknown"
	}
}

// ParseLogLevel parses a log level string
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "notice":
		return LogLevelNotice
	case "warning":
		return LogLevelWarning
	case "error":
		return LogLevelError
	case "critical":
		return LogLevelCritical
	case "alert":
		return LogLevelAlert
	case "emergency":
		return LogLevelEmergency
	default:
		return LogLevelUnknown
	}
}

// LogEntry represents a parsed log entry
type LogEntry struct {
	Content string
	Level   LogLevel
	File    string
}

// DetectLogLevel detects the log level from a Laravel log line
// Laravel format: [YYYY-MM-DD HH:MM:SS] environment.LEVEL: message
func DetectLogLevel(line string) LogLevel {
	// Find the closing bracket
	bracketEnd := strings.Index(line, "] ")
	if bracketEnd == -1 {
		return LogLevelUnknown
	}

	// Get the part after the bracket
	afterBracket := line[bracketEnd+2:]

	// Find the colon
	colonPos := strings.Index(afterBracket, ":")
	if colonPos == -1 {
		return LogLevelUnknown
	}

	// Get the environment.LEVEL part
	envLevel := afterBracket[:colonPos]

	// Find the last dot to extract the level
	dotPos := strings.LastIndex(envLevel, ".")
	if dotPos == -1 {
		return LogLevelUnknown
	}

	level := envLevel[dotPos+1:]
	return ParseLogLevel(level)
}

// IsStackTraceLine checks if a line is part of a stack trace
func IsStackTraceLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// PHP stack trace patterns
	if strings.HasPrefix(trimmed, "#") && len(trimmed) > 1 {
		// Check if second char is a digit
		if trimmed[1] >= '0' && trimmed[1] <= '9' {
			return true
		}
	}

	if strings.HasPrefix(trimmed, "Stack trace:") {
		return true
	}

	if strings.Contains(trimmed, " at ") && (strings.Contains(trimmed, ".php:") || strings.Contains(trimmed, "vendor/")) {
		return true
	}

	if strings.HasPrefix(trimmed, "in ") && strings.Contains(trimmed, ".php") {
		return true
	}

	if strings.HasPrefix(trimmed, "at ") && strings.Contains(trimmed, "::") {
		return true
	}

	return false
}

// IsErrorLine checks if a log line indicates an error
func IsErrorLine(line string) bool {
	level := DetectLogLevel(line)
	switch level {
	case LogLevelError, LogLevelCritical, LogLevelAlert, LogLevelEmergency:
		return true
	}

	// Fallback to content-based detection
	lower := strings.ToLower(line)
	return strings.Contains(lower, "exception") ||
		strings.Contains(lower, "error:") ||
		strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "[error]")
}
