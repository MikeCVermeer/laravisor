package proc

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ProcessOutputMsg is sent when a process produces output
type ProcessOutputMsg struct {
	ID       ProcessID
	Line     string
	IsStderr bool
}

// ProcessExitedMsg is sent when a process exits
type ProcessExitedMsg struct {
	ID       ProcessID
	ExitCode *int
}

// ProcessStartedMsg is sent when a process starts
type ProcessStartedMsg struct {
	ID  ProcessID
	PID int
}

// Manager handles spawning and managing processes
type Manager struct {
	processes map[ProcessID]*ManagedProcess
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// ManagedProcess wraps a Process with its exec.Cmd
type ManagedProcess struct {
	Process *Process
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	done    chan struct{} // Closed when process exits
}

// NewManager creates a new process manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		processes: make(map[ProcessID]*ManagedProcess),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Register registers a process with the manager
func (m *Manager) Register(proc *Process) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processes[proc.Config.ID] = &ManagedProcess{Process: proc}
}

// Spawn starts a process and returns a command to listen for output
// The outputCh parameter is the channel that will receive all process output
func (m *Manager) Spawn(id ProcessID, outputCh chan ProcessOutputMsg) tea.Cmd {
	m.mu.Lock()
	mp, exists := m.processes[id]
	if !exists {
		m.mu.Unlock()
		return nil
	}
	proc := mp.Process
	config := proc.Config
	m.mu.Unlock()

	// Kill existing process if running
	m.Kill(id)

	// Drain the output channel before starting
	for {
		select {
		case <-outputCh:
		default:
			goto drained
		}
	}
drained:

	// Create context for this process
	ctx, cancel := context.WithCancel(m.ctx)

	// Build command
	cmd := exec.CommandContext(ctx, config.Command, config.Args...)
	cmd.Dir = config.WorkingDir

	// Set up environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "FORCE_COLOR=1")
	cmd.Env = append(cmd.Env, "CLICOLOR_FORCE=1")
	cmd.Env = append(cmd.Env, "COLORTERM=truecolor")
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return func() tea.Msg {
			return ProcessOutputMsg{ID: id, Line: "Failed to create stdout pipe: " + err.Error(), IsStderr: true}
		}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return func() tea.Msg {
			return ProcessOutputMsg{ID: id, Line: "Failed to create stderr pipe: " + err.Error(), IsStderr: true}
		}
	}

	// Start process
	if err := cmd.Start(); err != nil {
		cancel()
		return func() tea.Msg {
			return ProcessOutputMsg{ID: id, Line: "Failed to start: " + err.Error(), IsStderr: true}
		}
	}

	// Create done channel for this process
	done := make(chan struct{})

	// Update state
	m.mu.Lock()
	mp.cmd = cmd
	mp.cancel = cancel
	mp.done = done
	proc.Status = ProcessStatusRunning
	if cmd.Process != nil {
		pid := cmd.Process.Pid
		proc.PID = &pid
	}
	proc.RestartCount = 0
	proc.BackoffSeconds = 1
	m.mu.Unlock()

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case outputCh <- ProcessOutputMsg{ID: id, Line: scanner.Text(), IsStderr: false}:
			}
		}
	}()

	// Read stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case outputCh <- ProcessOutputMsg{ID: id, Line: scanner.Text(), IsStderr: true}:
			}
		}
	}()

	// Wait for process in goroutine
	go func() {
		err := cmd.Wait()
		cancel()

		// Signal that process has exited
		close(done)

		m.mu.Lock()
		mp, exists := m.processes[id]
		if exists {
			mp.Process.Status = ProcessStatusStopped
			mp.Process.PID = nil

			// Check exit code
			var exitCode *int
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					code := exitErr.ExitCode()
					exitCode = &code
					if code != 0 {
						mp.Process.Status = ProcessStatusFailed
					}
				}
			} else {
				zero := 0
				exitCode = &zero
			}

			// Send exit message through channel (non-blocking)
			select {
			case outputCh <- ProcessOutputMsg{ID: id, Line: "Process exited", IsStderr: false}:
			default:
			}

			// Store exit code for restart decision
			mp.Process.lastExitCode = exitCode
		}
		m.mu.Unlock()
	}()

	// Return command to receive initial output
	return waitForOutput(id, outputCh)
}

// waitForOutput returns a tea.Cmd that waits for process output
func waitForOutput(id ProcessID, ch <-chan ProcessOutputMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			// Channel closed - process exited
			return ProcessExitedMsg{ID: id}
		}
		return msg
	}
}

// ListenForOutput returns a command to listen for more output
func ListenForOutput(id ProcessID, ch <-chan ProcessOutputMsg) tea.Cmd {
	return waitForOutput(id, ch)
}

// Kill stops a process gracefully (SIGTERM, wait, then SIGKILL)
func (m *Manager) Kill(id ProcessID) {
	m.mu.Lock()
	mp, exists := m.processes[id]
	if !exists || mp.cmd == nil || mp.cmd.Process == nil || mp.done == nil {
		m.mu.Unlock()
		return
	}
	cmd := mp.cmd
	cancel := mp.cancel
	done := mp.done
	m.mu.Unlock()

	// Send SIGTERM first
	cmd.Process.Signal(syscall.SIGTERM)

	// Wait for process to exit using the done channel (set by Spawn goroutine)
	// or timeout after 5 seconds
	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(5 * time.Second):
		// Force kill - process didn't respond to SIGTERM
		cmd.Process.Kill()
		// Wait for the done channel to close after kill
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			// Give up waiting, process may be zombie
		}
	}

	// Cancel context to stop output readers
	if cancel != nil {
		cancel()
	}

	// Update state
	m.mu.Lock()
	if mp, exists := m.processes[id]; exists {
		mp.Process.Status = ProcessStatusStopped
		mp.Process.PID = nil
		mp.cmd = nil
		mp.cancel = nil
		mp.done = nil
	}
	m.mu.Unlock()
}

// KillAll stops all processes
func (m *Manager) KillAll() {
	m.mu.RLock()
	ids := make([]ProcessID, 0, len(m.processes))
	for id := range m.processes {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	// Kill in parallel using goroutines
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(pid ProcessID) {
			defer wg.Done()
			m.Kill(pid)
		}(id)
	}
	wg.Wait()
}

// Restart stops and starts a process
func (m *Manager) Restart(id ProcessID, outputCh chan ProcessOutputMsg) tea.Cmd {
	m.Kill(id)
	time.Sleep(500 * time.Millisecond)
	return m.Spawn(id, outputCh)
}

// IsRunning checks if a process is running
func (m *Manager) IsRunning(id ProcessID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mp, exists := m.processes[id]
	if !exists {
		return false
	}
	return mp.Process.Status == ProcessStatusRunning
}

// GetPID returns the PID of a running process
func (m *Manager) GetPID(id ProcessID) *int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mp, exists := m.processes[id]
	if !exists {
		return nil
	}
	return mp.Process.PID
}

// ShouldRestart checks if a process should auto-restart
func (m *Manager) ShouldRestart(id ProcessID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mp, exists := m.processes[id]
	if !exists {
		return false
	}

	proc := mp.Process
	switch proc.Config.RestartPolicy {
	case RestartPolicyNever:
		return false
	case RestartPolicyOnFailure:
		// Restart only if exit code is non-zero
		if proc.lastExitCode != nil && *proc.lastExitCode == 0 {
			return false
		}
		return true
	case RestartPolicyAlways:
		return true
	}
	return false
}

// GetBackoffDelay returns the delay before restarting
func (m *Manager) GetBackoffDelay(id ProcessID) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mp, exists := m.processes[id]
	if !exists {
		return time.Second
	}

	delay := time.Duration(mp.Process.BackoffSeconds) * time.Second
	// Cap at 60 seconds
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}
	return delay
}

// RecordFailure records a failure for backoff calculation
func (m *Manager) RecordFailure(id ProcessID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mp, exists := m.processes[id]
	if !exists {
		return
	}
	mp.Process.RestartCount++
	// Exponential backoff: 2^failures, max 60
	mp.Process.BackoffSeconds = 1 << mp.Process.RestartCount
	if mp.Process.BackoffSeconds > 60 {
		mp.Process.BackoffSeconds = 60
	}
	mp.Process.LastRestartTime = time.Now()
}

// ResetBackoff resets the backoff state for a process
func (m *Manager) ResetBackoff(id ProcessID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mp, exists := m.processes[id]
	if !exists {
		return
	}
	mp.Process.RestartCount = 0
	mp.Process.BackoffSeconds = 1
}

// Shutdown cancels all processes and cleans up
func (m *Manager) Shutdown() {
	m.cancel()
	m.KillAll()
}
