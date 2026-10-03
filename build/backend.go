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

// Package build schedules and executes the build steps in a Ninja graph,
// tracking per-step state and delegating command execution and on-disk effects
// to a pluggable BuildBackend.
package build

import "context"

// ExecResult holds the outcome of running a build step.
type ExecResult struct {
	Err      error
	ExitCode int
	Stdout   string
	Stderr   string
}

// BuildBackend is implemented by execution backends used by the build engine.
type BuildBackend interface {
	BuildInit(engine *Engine) error
	IsClean(step *Step) (bool, error)
	ApplyIfCached(step *Step) (bool, error)
	IsRemotable(step *Step) bool
	ExecRemote(ctx context.Context, step *Step) ExecResult
	CreateDirs(step *Step) error
	CreateRSP(step *Step) error
	ExecLocal(ctx context.Context, step *Step) ExecResult
	FinalizeStep(step *Step)
	FinalizeBuild(err error) error
}
