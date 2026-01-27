package proc

import (
	"os/exec"
	"sync"
	"time"
)

// ProcessID is a unique identifier for a process
type ProcessID string

// ProcessStatus represents the current state of a process
type ProcessStatus int

const (
	ProcessStatusStopped ProcessStatus = iota
	ProcessStatusRunning
	ProcessStatusRestarting
	ProcessStatusFailed
)

// String returns the display name for the status
func (s ProcessStatus) String() string {
	switch s {
	case ProcessStatusStopped:
		return "stopped"
	case ProcessStatusRunning:
		return "running"
	case ProcessStatusRestarting:
		return "restarting"
	case ProcessStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// RestartPolicy defines how a process should be restarted
type RestartPolicy int

const (
	RestartPolicyNever RestartPolicy = iota
	RestartPolicyOnFailure
	RestartPolicyAlways
)

// String returns the display name for the policy
func (p RestartPolicy) String() string {
	switch p {
	case RestartPolicyNever:
		return "never"
	case RestartPolicyOnFailure:
		return "on_failure"
	case RestartPolicyAlways:
		return "always"
	default:
		return "never"
	}
}

// ProcessKind identifies the type of managed process
type ProcessKind string

const (
	ProcessKindServe   ProcessKind = "serve"
	ProcessKindVite    ProcessKind = "vite"
	ProcessKindQueue   ProcessKind = "queue"
	ProcessKindHorizon ProcessKind = "horizon"
	ProcessKindReverb  ProcessKind = "reverb"
	ProcessKindCustom  ProcessKind = "custom"
)

// OutputLine represents a line of process output
type OutputLine struct {
	Content  string
	IsStderr bool
	Time     time.Time
}

// Stdout creates a stdout output line
func Stdout(content string) OutputLine {
	return OutputLine{Content: content, IsStderr: false, Time: time.Now()}
}

// Stderr creates a stderr output line
func Stderr(content string) OutputLine {
	return OutputLine{Content: content, IsStderr: true, Time: time.Now()}
}

// ProcessConfig defines how to start a process
type ProcessConfig struct {
	ID            ProcessID
	Kind          ProcessKind
	DisplayName   string
	Command       string
	Args          []string
	WorkingDir    string
	Env           map[string]string
	RestartPolicy RestartPolicy
	Hotkey        *rune
}

// Process represents a managed process
type Process struct {
	Config ProcessConfig
	Status ProcessStatus
	PID    *int

	// Output buffer (ring buffer)
	Output       []OutputLine
	MaxOutput    int
	ScrollOffset int

	// Restart tracking
	RestartCount    int
	LastRestartTime time.Time
	BackoffSeconds  int

	// Internal state
	cmd          *exec.Cmd
	mu           sync.RWMutex
	stopCh       chan struct{}
	lastExitCode *int
}

// NewProcess creates a new process with the given config
func NewProcess(config ProcessConfig) *Process {
	return &Process{
		Config:    config,
		Status:    ProcessStatusStopped,
		Output:    make([]OutputLine, 0),
		MaxOutput: 1000,
		stopCh:    make(chan struct{}),
	}
}

// AddOutput adds an output line to the process buffer
func (p *Process) AddOutput(line OutputLine) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.Output) >= p.MaxOutput {
		// Remove oldest entry
		p.Output = p.Output[1:]
	}
	p.Output = append(p.Output, line)
}

// GetOutput returns a copy of the output buffer
func (p *Process) GetOutput() []OutputLine {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]OutputLine, len(p.Output))
	copy(result, p.Output)
	return result
}

// ClearOutput clears the output buffer
func (p *Process) ClearOutput() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Output = make([]OutputLine, 0)
	p.ScrollOffset = 0
}
