// Package run abstracts subprocess execution so command logic stays testable.
package run

import (
	"os"
	"os/exec"

	"starnix.net/pac/internal/progress"
)

// Runner executes external commands.
type Runner interface {
	// Run executes name with args attached to the current process stdio,
	// so child tools (e.g. pacman) own the TTY and draw their own output.
	Run(name string, args ...string) error
	// Capture executes name with args and returns its stdout.
	Capture(name string, args ...string) (string, error)
	// RunBar runs name with args, capturing its stdout to render a progress
	// bar; stderr and stdin pass through to the terminal. label names the
	// target being acted on (e.g. the flatpak app id) and is shown beside the
	// bar; pass "" when the stream supplies its own per-item names.
	RunBar(label, name string, args ...string) error
}

// Real is the production Runner backed by os/exec.
type Real struct{}

func (Real) Run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

func (Real) RunBar(label, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	progress.Render(pipe, os.Stderr, 24, label) // blocks until the command closes stdout
	return cmd.Wait()
}

func (Real) Capture(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

// Call is a queued fake result.
type Call struct {
	Out string
	Err error
}

// Fake is a test Runner: it records calls and returns queued Results in order.
type Fake struct {
	Calls   [][]string
	Results []Call
	idx     int
}

func (f *Fake) record(name string, args []string) Call {
	f.Calls = append(f.Calls, append([]string{name}, args...))
	var c Call
	if f.idx < len(f.Results) {
		c = f.Results[f.idx]
	}
	f.idx++
	return c
}

func (f *Fake) Run(name string, args ...string) error {
	return f.record(name, args).Err
}

func (f *Fake) RunBar(label, name string, args ...string) error {
	// Record only the command (name+args); the label is display-only and not
	// part of what gets executed, so call assertions stay focused on the cmd.
	return f.record(name, args).Err
}

func (f *Fake) Capture(name string, args ...string) (string, error) {
	c := f.record(name, args)
	return c.Out, c.Err
}
