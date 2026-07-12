package proc

import (
	"testing"
	"time"
)

func TestRestartStateBackoffDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		failures int
		want     time.Duration
	}{
		{name: "negative failure count uses minimum", failures: -1, want: time.Second},
		{name: "no failures uses minimum", failures: 0, want: time.Second},
		{name: "first failure", failures: 1, want: 2 * time.Second},
		{name: "second failure", failures: 2, want: 4 * time.Second},
		{name: "fifth failure", failures: 5, want: 32 * time.Second},
		{name: "sixth failure reaches cap", failures: 6, want: MaxRestartBackoff},
		{name: "large failure count remains capped", failures: 1_000, want: MaxRestartBackoff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := RestartState{ConsecutiveFailures: tt.failures}
			if got := state.BackoffDelay(); got != tt.want {
				t.Fatalf("BackoffDelay() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestShouldRestart(t *testing.T) {
	t.Parallel()

	success := 0
	failure := 1
	negativeExit := -1

	tests := []struct {
		name     string
		policy   RestartPolicy
		exitCode *int
		want     bool
	}{
		{name: "never after success", policy: RestartPolicyNever, exitCode: &success, want: false},
		{name: "never after failure", policy: RestartPolicyNever, exitCode: &failure, want: false},
		{name: "never with unknown exit", policy: RestartPolicyNever, exitCode: nil, want: false},
		{name: "on failure after success", policy: RestartPolicyOnFailure, exitCode: &success, want: false},
		{name: "on failure after non-zero exit", policy: RestartPolicyOnFailure, exitCode: &failure, want: true},
		{name: "on failure after negative exit", policy: RestartPolicyOnFailure, exitCode: &negativeExit, want: true},
		{name: "on failure with unknown exit", policy: RestartPolicyOnFailure, exitCode: nil, want: true},
		{name: "always after success", policy: RestartPolicyAlways, exitCode: &success, want: true},
		{name: "always after failure", policy: RestartPolicyAlways, exitCode: &failure, want: true},
		{name: "always with unknown exit", policy: RestartPolicyAlways, exitCode: nil, want: true},
		{name: "unknown policy fails closed", policy: RestartPolicy(99), exitCode: &failure, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ShouldRestart(tt.policy, tt.exitCode); got != tt.want {
				t.Fatalf("ShouldRestart(%v, %v) = %t, want %t", tt.policy, tt.exitCode, got, tt.want)
			}
		})
	}
}
