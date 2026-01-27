package proc

import "time"

// MaxRestartBackoff is the maximum backoff delay (60 seconds)
const MaxRestartBackoff = 60 * time.Second

// RestartState tracks restart state for a process
type RestartState struct {
	// ConsecutiveFailures counts failures since last successful start
	ConsecutiveFailures int
	// LastRestartTime is when the last restart was attempted
	LastRestartTime time.Time
}

// BackoffDelay calculates the backoff delay based on consecutive failures
// Uses exponential backoff: 2^failures seconds, capped at 60 seconds
func (s *RestartState) BackoffDelay() time.Duration {
	if s.ConsecutiveFailures <= 0 {
		return time.Second
	}

	// Calculate 2^failures
	delay := time.Second
	for i := 0; i < s.ConsecutiveFailures; i++ {
		delay *= 2
		if delay > MaxRestartBackoff {
			return MaxRestartBackoff
		}
	}
	return delay
}

// RecordFailure records a failure for backoff calculation
func (s *RestartState) RecordFailure() {
	s.ConsecutiveFailures++
	s.LastRestartTime = time.Now()
}

// Reset resets the restart state after a successful start
func (s *RestartState) Reset() {
	s.ConsecutiveFailures = 0
	s.LastRestartTime = time.Now()
}

// ShouldRestart determines if a process should be restarted based on its
// exit code and restart policy
func ShouldRestart(policy RestartPolicy, exitCode *int) bool {
	switch policy {
	case RestartPolicyNever:
		return false

	case RestartPolicyOnFailure:
		// Restart only if exit code is non-zero or unknown
		if exitCode == nil {
			return true // Unknown exit code, assume failure
		}
		return *exitCode != 0

	case RestartPolicyAlways:
		return true

	default:
		return false
	}
}
