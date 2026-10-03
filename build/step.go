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

package build

import (
	"sync/atomic"

	"github.com/google/ninja-go/graph"
	"github.com/google/ninja-go/ninjafiles/ninjalog"
)

// Step represents the build information for a given node.
type Step struct {
	ID int
	// state is the current state of the step in the build state machine.
	state BuildState
	// inDegree is the number of input edges that have not yet been built.
	inDegree atomic.Int32
	// Node is the graph node that this step is associated with.
	Node *graph.Node
	// poolName is the name of the pool that this step is assigned to.
	poolName string
	// Pool is the scheduler that owns this step once it has been queued locally.
	Pool *Scheduler

	remotable bool

	// The following fields are only populated while the step is about to run
	// or running, and are cleared after execution to free memory.

	// command is the command to run.
	command string
	// rspPath is the path of the response file, if any.
	rspPath string
	// rspContent is the contents of the response file, if any.
	rspContent string
	// cmdHash is the hash of the command (and rsp content), as recorded in ninja_log.
	cmdHash uint64
}

// NewStep returns a Step in StateInit for the given graph node.
func NewStep(node *graph.Node) *Step {
	return &Step{
		state:    StateInit,
		Node:     node,
		poolName: node.Pool(),
	}
}

// String returns a human-readable identifier for s.
func (s *Step) String() string {
	return s.Node.String()
}

// IsPhony reports whether s wraps a phony build edge.
func (s *Step) IsPhony() bool {
	return s.Node.Build.Rule.IsPhony()
}

// IsPhonyOutput reports whether s declares phony_output.
func (s *Step) IsPhonyOutput() bool {
	return s.Node.Build.Scope.ExpandField("phony_output") != ""
}

// IsConsole reports whether s is in the console pool.
func (s *Step) IsConsole() bool {
	return s.poolName == "console"
}

// IsGenerator reports whether s sets the generator attribute.
func (s *Step) IsGenerator() bool {
	return s.Node.Build.Scope.ExpandField("generator") != ""
}

// Restat reports whether s sets the restat attribute.
func (s *Step) Restat() bool {
	return s.Node.Build.Scope.ExpandField("restat") != ""
}

// LoadCommand expands the command, response file, and command hash and stores
// them on s in preparation for execution.
func (s *Step) LoadCommand() {
	s.command = s.Node.Command()
	s.rspPath = s.Node.RSPFile()
	if s.rspPath != "" {
		s.rspContent = s.Node.RSPContent()
	}
	s.cmdHash = ninjalog.HashCommand(s.command, s.rspContent)
}

// ClearCommand discards the command and response-file fields so they can be
// garbage collected after execution.
func (s *Step) ClearCommand() {
	s.command = ""
	s.rspPath = ""
	s.rspContent = ""
	s.cmdHash = 0
}

// IsCommandLoaded reports whether LoadCommand has been called and ClearCommand
// has not.
func (s *Step) IsCommandLoaded() bool {
	return !(s.command == "" && s.cmdHash == 0)
}

// Command returns the expanded command. It panics if no command is loaded.
func (s *Step) Command() string {
	if !s.IsCommandLoaded() {
		panic("command is not loaded")
	}
	return s.command
}

// RSPFile returns the path of the response file, if any. It panics if no
// command is loaded.
func (s *Step) RSPFile() string {
	if !s.IsCommandLoaded() {
		panic("command is not loaded")
	}
	return s.rspPath
}

// RSPContent returns the response-file contents, if any. It panics if no
// command is loaded.
func (s *Step) RSPContent() string {
	if !s.IsCommandLoaded() {
		panic("command is not loaded")
	}
	return s.rspContent
}

// CmdHash returns the hash of the command and response-file contents. It
// panics if no command is loaded.
func (s *Step) CmdHash() uint64 {
	if !s.IsCommandLoaded() {
		panic("command is not loaded")
	}
	return s.cmdHash
}
