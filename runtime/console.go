// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Console is a Logger implementation that writes status updates and log
// messages to a terminal.
type Console struct {
	statusFormat string
	out          io.Writer
	verbosity    int
	quiet        bool

	lock      sync.Mutex
	startTime time.Time
	started   int
	finished  int
	total     int
}

// defaultStatusFormat is used when NINJA_STATUS is not set in the environment.
const defaultStatusFormat = "[%p %s/%t] "

// NewConsole constructs a Console using settings from params and the
// NINJA_STATUS environment variable.
func NewConsole(params *RunParameters) *Console {
	format := os.Getenv("NINJA_STATUS")
	if format == "" {
		format = defaultStatusFormat
	}

	return &Console{
		statusFormat: format,
		out:          params.Out,
		verbosity:    params.CliFlags.Verbosity,
		quiet:        params.CliFlags.Quiet,
	}
}

var _ Logger = (*Console)(nil)

// BuildStarted implements Logger.
func (c *Console) BuildStarted() {
}

// status returns a Status snapshot reflecting the current build progress.
func (c *Console) status() *Status {
	return &Status{
		Started:     c.started,
		Total:       c.total,
		Running:     c.finished - c.started,
		Finished:    c.finished,
		Elapsed:     time.Since(c.startTime),
		ETA:         time.Duration(float64(time.Since(c.startTime)) * float64(c.total-c.finished) / float64(c.finished)),
		OverallRate: float64(c.finished) / time.Since(c.startTime).Seconds(),
		CurrentRate: float64(c.finished) / time.Since(c.startTime).Seconds(), // FIXME: create a window for the current rate.
	}
}

// BuildFinished implements Logger.
func (c *Console) BuildFinished() {
}

// BuildStepStarted implements Logger.
func (c *Console) BuildStepStarted(id int, cmd string, desc string, outputs []string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.started++

	if c.quiet {
		return
	}

	display := desc
	if c.verbosity > 0 || desc == "" {
		display = cmd
	}

	fmt.Fprintf(c.out, "%s%s\n", c.status().Format(c.statusFormat), display)
}

// BuildStepFinished implements Logger.
func (c *Console) BuildStepFinished(id int, exitCode int, output string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.finished++
}

// BuildStepFinished implements Logger.
func (c *Console) BuildStepCanceled(id int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.started--
}

// PlanHasTotalSteps implements Logger.
func (c *Console) PlanHasTotalSteps(total int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.total = total
}

// Printf implements Logger.
func (c *Console) Printf(format string, a ...any) {
	c.lock.Lock()
	defer c.lock.Unlock()

	fmt.Fprintf(c.out, format, a...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(c.out)
	}
}

// Tracef implements Logger.
func (c *Console) Tracef(format string, a ...any) {
	if c.verbosity > 1 {
		c.Printf("TRACE "+format, a...)
	}
}

// Debugf implements Logger.
func (c *Console) Debugf(format string, a ...any) {
	if c.verbosity > 0 {
		c.Printf("DEBUG "+format, a...)
	}
}

// Infof implements Logger.
func (c *Console) Infof(format string, a ...any) {
	if !c.quiet {
		c.Printf("INFO  "+format, a...)
	}
}

// Warnf implements Logger.
func (c *Console) Warnf(format string, a ...any) {
	if !c.quiet {
		c.Printf("WARN  "+format, a...)
	}
}

// Errorf implements Logger.
func (c *Console) Errorf(format string, a ...any) {
	c.Printf("ERROR "+format, a...)
}

// Close implements Logger.
func (c *Console) Close() error {
	return nil
}
