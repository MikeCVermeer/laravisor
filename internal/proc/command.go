package proc

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandOutputMsg is sent when a command produces output
type CommandOutputMsg struct {
	Line     string
	IsStderr bool
}

// CommandExitedMsg is sent when a command exits
type CommandExitedMsg struct {
	ExitCode int
	Error    error
}

// CommandRunner runs one-off commands (artisan, make, quality tools)
type CommandRunner struct {
	workingDir string
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	outputCh   chan OutputLine
	exitCode   int
}

// NewCommandRunner creates a new command runner
func NewCommandRunner(workingDir string) *CommandRunner {
	return &CommandRunner{
		workingDir: workingDir,
		outputCh:   make(chan OutputLine, 100),
	}
}

// Run executes a command and returns a tea.Cmd to receive output
func (r *CommandRunner) Run(command string, args []string) tea.Cmd {
	// Cancel any existing command
	r.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = r.workingDir

	// Set up environment for color output
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "FORCE_COLOR=1")
	cmd.Env = append(cmd.Env, "CLICOLOR_FORCE=1")
	cmd.Env = append(cmd.Env, "COLORTERM=truecolor")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return func() tea.Msg {
			return CommandExitedMsg{ExitCode: 1, Error: err}
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return func() tea.Msg {
			return CommandExitedMsg{ExitCode: 1, Error: err}
		}
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return func() tea.Msg {
			return CommandExitedMsg{ExitCode: 1, Error: err}
		}
	}

	r.cmd = cmd

	// Create output channel
	r.outputCh = make(chan OutputLine, 100)

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case r.outputCh <- OutputLine{Content: scanner.Text(), IsStderr: false, Time: time.Now()}:
			}
		}
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case r.outputCh <- OutputLine{Content: scanner.Text(), IsStderr: true, Time: time.Now()}:
			}
		}
	}()

	// Wait for command completion
	go func() {
		err := cmd.Wait()
		cancel()

		r.exitCode = 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				r.exitCode = exitErr.ExitCode()
			} else {
				r.exitCode = 1
			}
		}

		// Signal completion
		close(r.outputCh)
		r.outputCh = nil
	}()

	return r.listenForOutput()
}

// listenForOutput returns a command to receive output
func (r *CommandRunner) listenForOutput() tea.Cmd {
	return func() tea.Msg {
		if r.outputCh == nil {
			return CommandExitedMsg{ExitCode: r.exitCode}
		}

		line, ok := <-r.outputCh
		if !ok {
			// Channel closed - command finished
			return CommandExitedMsg{ExitCode: r.exitCode}
		}

		return CommandOutputMsg{Line: line.Content, IsStderr: line.IsStderr}
	}
}

// ListenForMore returns a command to continue listening for output
func (r *CommandRunner) ListenForMore() tea.Cmd {
	return r.listenForOutput()
}

// Stop stops the running command
func (r *CommandRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.cmd != nil && r.cmd.Process != nil {
		r.cmd.Process.Kill()
		r.cmd = nil
	}
}

// IsRunning returns whether a command is currently running
func (r *CommandRunner) IsRunning() bool {
	return r.cmd != nil && r.cmd.ProcessState == nil
}
