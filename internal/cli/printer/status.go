package printer

import (
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/util/duration"
	"time"

	"capact.io/capact/internal/cli"

	"github.com/fatih/color"
)

// Spinner defines interface for terminal spinner.
type Spinner interface {
	Start(stage string)
	Active() bool
	Stop(msg string)
}

// Status provides functionality to display steps progress in terminal.
type Status struct {
	stage       string
	spinner     Spinner
	timeStarted time.Time
}

// NewStatus returns a new Status instance.
func NewStatus(w io.Writer, header string) *Status {
	if header != "" {
		fmt.Fprintln(w, header)
	}

	st := &Status{}
	if cli.IsSmartTerminal(w) {
		st.spinner = NewDynamicSpinner(w)
	} else {
		st.spinner = NewStaticSpinner(w)
	}

	return st
}

// Step starts spinner for a given step.
func (s *Status) Step(stageFmt string, args ...interface{}) {
	// Finish previously started step
	s.End(true)
	s.timeStarted = time.Now()

	s.stage = fmt.Sprintf(stageFmt, args...)
	s.spinner.Start(s.stage)
}

// End marks started step as completed.
func (s *Status) End(success bool) {
	if !s.spinner.Active() {
		return
	}

	var icon string
	if success {
		icon = color.GreenString("✓")
	} else {
		icon = color.RedString("✗")
	}

	durStyle := color.New(color.Faint, color.Italic)
	dur := durStyle.Sprintf("[took %s]", duration.HumanDuration(time.Since(s.timeStarted)))
	msg := fmt.Sprintf(" %s %s %s\n",
		icon, s.stage, dur)
	s.spinner.Stop(msg)
}
