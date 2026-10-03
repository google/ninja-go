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

import "fmt"

// BuildState identifies a step's position in the build state machine.
type BuildState int

// BuildState values, in roughly the order steps move through them.
const (
	StateInit BuildState = iota
	StateBlocked
	StateCleanCheck
	StateClean
	StateCacheCheck
	StateCached
	StateRemoteQueued
	StateRemoteRunning
	StateLocalQueued
	StateLocalRunning
	StateFailed
	StateDone
	MaxState
)

// String returns a human-readable name for s.
func (s BuildState) String() string {
	switch s {
	case StateInit:
		return "Init"
	case StateBlocked:
		return "Blocked"
	case StateCleanCheck:
		return "CleanCheck"
	case StateClean:
		return "Clean"
	case StateCacheCheck:
		return "CacheCheck"
	case StateCached:
		return "Cached"
	case StateRemoteQueued:
		return "RemoteQueued"
	case StateRemoteRunning:
		return "RemoteRunning"
	case StateLocalQueued:
		return "LocalQueued"
	case StateLocalRunning:
		return "LocalRunning"
	case StateFailed:
		return "Failed"
	case StateDone:
		return "Done"
	default:
		return fmt.Sprintf("Unknown state (%d)", s)
	}
}

// IsFinal reports whether s is a terminal state.
func (s BuildState) IsFinal() bool {
	switch s {
	case StateFailed, StateDone, StateClean, StateCached:
		return true
	default:
		return false
	}
}

// IsSuccess reports whether s is a terminal state that indicates success.
func (s BuildState) IsSuccess() bool {
	switch s {
	case StateDone, StateClean, StateCached:
		return true
	default:
		return false
	}
}
