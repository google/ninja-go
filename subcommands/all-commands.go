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

// Package subcommands implements the ninjago subcommands invoked via -t
// (build, graph, list, path, query, and friends), along with the shared
// argument plumbing and CommandSpec registry that dispatches to them.
package subcommands

// CommandSpecs lists every subcommand exposed by ninjago.
var CommandSpecs = []*CommandSpec{
	buildCommandSpec,
	commandsCommandSpec,
	defaultsCommandSpec,
	graphCommandSpec,
	inputsCommandSpec,
	listCommandSpec,
	ninjaFilesCommandSpec,
	pathCommandSpec,
	queryCommandSpec,
	urtleCommandSpec,
	varCommandSpec,
	dumpCommandSpec,
	ruleQueryCommandSpec,
}

// findCommandSpec returns the CommandSpec with the given name, or nil if no
// such command exists. The "help" alias is mapped to "list".
func findCommandSpec(name string) *CommandSpec {
	if name == "help" {
		name = "list"
	}
	for _, cmd := range CommandSpecs {
		if name == cmd.name {
			return cmd
		}
	}
	return nil
}
