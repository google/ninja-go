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
)

// RunParameters holds all of the general parameters that are required to run a ninjago command.
type RunParameters struct {
	// NinjaFile is the path to the ninja file passed to the CLI (-f flag).
	NinjaFile string
	// CliFlags holds the flags and options passed to the CLI.
	CliFlags *CliFlags
	// Prefix is the command prefix used when showing command usage.
	Prefix string
	// Out is the writer used for command output (distinct from application logs).
	Out io.Writer
	// Log is the logger used to log messages.
	Log Logger
	// IsInteractive reports whether the command is being used interactively.
	IsInteractive bool
	// IsHTTP reports whether the command is being used in HTTP mode.
	IsHTTP bool
	// CleanInspectorsCount is the number of concurrent clean inspections the build engine runs.
	CleanInspectorsCount int
	// CacheCheckersCount is the number of concurrent cache checkers the build engine runs.
	CacheCheckersCount int
	// RemoteExecutorsCount is the maximum number of concurrent remote executions.
	RemoteExecutorsCount int
}

// NewRunParams returns a RunParameters with default values.
func NewRunParams() *RunParameters {
	return &RunParameters{
		Out:                  os.Stdout,
		Prefix:               "ninjago -t ",
		CleanInspectorsCount: 2500, // It should be less than 10000 to avoid resource limit issues. 2500 seem to perform well.
		CacheCheckersCount:   2500,
		RemoteExecutorsCount: 2000,
	}
}

// Print writes a formatted message to the configured output, ensuring it ends
// with a newline.
func (c *RunParameters) Print(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(c.Out, format, a...)
}
