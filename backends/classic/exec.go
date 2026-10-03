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

package classic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/ninja-go/build"
)

// execCmd runs the step's command via /bin/sh -c and captures its stdout,
// stderr, and exit status.
func execCmd(ctx context.Context, step *build.Step) (r build.ExecResult) {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", step.Command())

	if step.IsConsole() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		defer func() {
			r.Stdout = stdout.String()
			r.Stderr = stderr.String()
		}()
	}

	var exitErr *exec.ExitError
	r.Err = cmd.Run()
	if r.Err != nil && !errors.As(r.Err, &exitErr) {
		r.Err = fmt.Errorf("failed to exec command: %w", r.Err)
		return
	}
	r.ExitCode = cmd.ProcessState.ExitCode()

	if r.ExitCode != 0 {
		r.Err = fmt.Errorf("failed with exit-code %d\n%s\n%s", r.ExitCode, r.Stdout, r.Stderr)
	}
	return
}
