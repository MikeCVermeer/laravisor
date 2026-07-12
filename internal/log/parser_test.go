package log

import "testing"

func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  LogLevel
	}{
		{input: "debug", want: LogLevelDebug},
		{input: "INFO", want: LogLevelInfo},
		{input: "Notice", want: LogLevelNotice},
		{input: "warning", want: LogLevelWarning},
		{input: "ERROR", want: LogLevelError},
		{input: "critical", want: LogLevelCritical},
		{input: "alert", want: LogLevelAlert},
		{input: "emergency", want: LogLevelEmergency},
		{input: "trace", want: LogLevelUnknown},
		{input: "", want: LogLevelUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			if got := ParseLogLevel(tt.input); got != tt.want {
				t.Fatalf("ParseLogLevel(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogLevelString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level LogLevel
		want  string
	}{
		{level: LogLevelDebug, want: "debug"},
		{level: LogLevelInfo, want: "info"},
		{level: LogLevelNotice, want: "notice"},
		{level: LogLevelWarning, want: "warning"},
		{level: LogLevelError, want: "error"},
		{level: LogLevelCritical, want: "critical"},
		{level: LogLevelAlert, want: "alert"},
		{level: LogLevelEmergency, want: "emergency"},
		{level: LogLevelUnknown, want: "unknown"},
		{level: LogLevel(99), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.level.String(); got != tt.want {
				t.Fatalf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestDetectLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want LogLevel
	}{
		{
			name: "standard Laravel line",
			line: "[2026-07-12 10:11:12] local.ERROR: Connection refused",
			want: LogLevelError,
		},
		{
			name: "uppercase level is accepted",
			line: "[2026-07-12 10:11:12] production.CRITICAL: Database unavailable",
			want: LogLevelCritical,
		},
		{
			name: "environment may contain dots",
			line: "[2026-07-12 10:11:12] app.worker.WARNING: Retrying job",
			want: LogLevelWarning,
		},
		{
			name: "unknown level",
			line: "[2026-07-12 10:11:12] local.TRACE: Request details",
			want: LogLevelUnknown,
		},
		{name: "missing timestamp delimiter", line: "local.ERROR: failure", want: LogLevelUnknown},
		{name: "missing colon", line: "[2026-07-12 10:11:12] local.ERROR failure", want: LogLevelUnknown},
		{name: "missing environment separator", line: "[2026-07-12 10:11:12] ERROR: failure", want: LogLevelUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DetectLogLevel(tt.line); got != tt.want {
				t.Fatalf("DetectLogLevel(%q) = %s, want %s", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsStackTraceLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "numbered frame", line: "#12 /app/vendor/laravel/framework/src/Container.php(123): call()", want: true},
		{name: "stack trace heading", line: "Stack trace:", want: true},
		{name: "PHP location", line: "Thrown at /app/Http/Controller.php:42", want: true},
		{name: "vendor location", line: "failure at vendor/laravel/framework", want: true},
		{name: "in PHP file", line: "in /app/Models/User.php on line 10", want: true},
		{name: "static method frame", line: "at App\\Jobs\\Import::handle", want: true},
		{name: "hash without frame number", line: "# frame", want: false},
		{name: "ordinary PHP message", line: "Updated Controller.php successfully", want: false},
		{name: "ordinary text", line: "Queue worker started", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsStackTraceLine(tt.line); got != tt.want {
				t.Fatalf("IsStackTraceLine(%q) = %t, want %t", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsErrorLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "Laravel error", line: "[2026-07-12 10:11:12] local.ERROR: failed", want: true},
		{name: "Laravel emergency", line: "[2026-07-12 10:11:12] production.EMERGENCY: down", want: true},
		{name: "warning is not an error", line: "[2026-07-12 10:11:12] local.WARNING: retrying", want: false},
		{name: "exception fallback", line: "RuntimeException while handling request", want: true},
		{name: "error fallback", line: "worker error: connection reset", want: true},
		{name: "fatal fallback", line: "PHP Fatal Error", want: true},
		{name: "bracketed error fallback", line: "[error] service unavailable", want: true},
		{name: "ordinary line", line: "Request completed successfully", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsErrorLine(tt.line); got != tt.want {
				t.Fatalf("IsErrorLine(%q) = %t, want %t", tt.line, got, tt.want)
			}
		})
	}
}
