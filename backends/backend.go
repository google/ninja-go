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

// Package backends defines the umbrella Backend interface that ties together
// the per-stage backend interfaces (subcommands, parse, graph, build) into the
// single contract a ninjago binary must satisfy.
package backends

import (
	"github.com/google/ninja-go/build"
	"github.com/google/ninja-go/graph"
	"github.com/google/ninja-go/parse"
	"github.com/google/ninja-go/subcommands"
)

// Backend bundles every backend interface that a ninjago binary must
// implement: command initialization, ninja-file parsing, graph loading, and
// build execution.
type Backend interface {
	subcommands.GeneralBackend
	parse.ParseBackend
	graph.GraphBackend
	build.BuildBackend
}
