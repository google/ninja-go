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

package graph

import (
	"regexp"
	"sync"

	"github.com/google/ninja-go/ninja"
)

// Node is a vertex in the build graph: one build edge with its inputs,
// outputs, and optional validations.
type Node struct {
	lock        sync.Mutex
	Targets     []string
	Build       *ninja.Build
	Inputs      map[*Node]int
	Outputs     map[*Node]int
	Validations map[*Node]struct{}
	Deps        []string
	Data        any
}

// newNode returns a Node wrapping build with empty input/output maps.
func newNode(build *ninja.Build) *Node {
	return &Node{
		Build:   build,
		Inputs:  make(map[*Node]int),
		Outputs: make(map[*Node]int),
	}
}

// String returns the first declared target of n.
func (n *Node) String() string {
	// FIXME: would joining all targets read better than just the first one?
	return n.Targets[0]
}

// AddTarget records t as one of the outputs produced by n.
func (n *Node) AddTarget(t string) {
	n.Targets = append(n.Targets, t)
}

// InDegree returns the number of incoming edges, summed across all inputs.
func (n *Node) InDegree() int {
	cc := 0
	for _, i := range n.Inputs {
		cc += i
	}
	return cc
}

// OutDegree returns the number of outgoing edges, summed across all outputs.
func (n *Node) OutDegree() int {
	cc := 0
	for _, i := range n.Outputs {
		cc += i
	}
	return cc
}

// Description returns the rule's expanded "description" field, falling back to
// the command and finally the node's first target.
func (n *Node) Description() string {
	desc := n.Build.Scope.ExpandField("description")
	if desc != "" {
		return desc
	}
	cmd := n.Build.Scope.ExpandField("command")
	if cmd != "" {
		return cmd
	}
	return n.String()
}

// DepFile returns the expanded value of the build's depfile attribute.
func (n *Node) DepFile() string {
	return n.Build.Scope.ExpandField("depfile")
}

// Command returns the expanded command line for n.
func (n *Node) Command() string {
	return n.Build.Scope.ExpandField("command")
}

// RSPFile returns the expanded path of the response file, if any.
func (n *Node) RSPFile() string {
	return n.Build.Scope.ExpandField("rspfile")
}

// RSPContent returns the expanded contents of the response file, if any.
func (n *Node) RSPContent() string {
	return n.Build.Scope.ExpandField("rspfile_content")
}

// Pool returns the name of the pool that owns n, or "default" if none was
// specified.
func (n *Node) Pool() string {
	p := n.Build.Scope.ExpandField("pool")
	if p == "" {
		return "default"
	}
	return p
}

// TargetsMatch reports whether any of n's declared targets match re.
func (n *Node) TargetsMatch(re *regexp.Regexp) bool {
	for _, t := range n.Targets {
		if re.MatchString(t) {
			return true
		}
	}
	return false
}
