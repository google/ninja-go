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

package subcommands

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/graph"
)

// inputsCommandSpec describes the "inputs" subcommand.
var inputsCommandSpec = &CommandSpec{
	name:  "inputs",
	desc:  "Show all (recursive) inputs for a target",
	usage: "[-i] [-d] [-p <path>] <target>...",
	newCmd: func() Command {
		return new(InputsCommand)
	},
}

// InputsCommand executes the "inputs" subcommand.
type InputsCommand struct {
	BaseCommand
	intermidiate bool
	deps         bool
	ignorePath   []string
}

// RegisterFlags implements Command.
func (c *InputsCommand) RegisterFlags(fl *pflag.FlagSet) {
	fl.BoolVarP(&c.intermidiate, "i", "i", false, "Include intermediate inputs.")
	fl.BoolVarP(&c.deps, "d", "d", false, "Include deps from the deps log file (.ninja_deps).")
	fl.StringSliceVarP(&c.ignorePath, "e", "e", nil, "Ignore dependency paths that include the given path (can be added multiple times).")
}

// Run implements Command.
func (c *InputsCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	args := EnsureArgs(fl)
	return c.run(ctx, args)
}

// run prints the recursive inputs of the given targets.
func (c *InputsCommand) run(ctx context.Context, targets []string) error {
	g, err := c.GenGraph(ctx, targets, c.deps)
	if err != nil {
		return err
	}

	result := make(chan string)

	go c.pump(g, targets, result)

	for t := range result {
		fmt.Println(t)
	}

	return nil
}

// pump walks the graph from the given targets and emits each input exactly
// once on result.
func (c *InputsCommand) pump(g *graph.Graph, targets []string, result chan<- string) {
	defer close(result)

	excludes := make(map[string]struct{}, len(c.ignorePath))
	for _, p := range c.ignorePath {
		excludes[p] = struct{}{}
	}

	seen := make(map[string]struct{}, len(g.Files)+len(g.Nodes))
	send := func(t string) {
		if _, ok := seen[t]; !ok {
			result <- t
			seen[t] = struct{}{}
		}
	}

	seenNodes := make(map[*graph.Node]struct{}, len(g.Nodes))

	var dfs func(n *graph.Node)
	dfs = func(n *graph.Node) {
		if _, ok := seenNodes[n]; ok {
			return
		}
		seenNodes[n] = struct{}{}

		for _, ins := range n.Build.AllInputs() {
			for _, i := range ins {
				if !c.intermidiate {
					if g.Targets.Get(i) != nil {
						continue
					}
				}
				send(i)
			}
		}
		if c.deps {
			if _, ok := excludes[n.DepFile()]; !ok {
				for _, d := range n.Deps {
					send(d)
				}
			}
		}

		for m := range n.Inputs {
			dfs(m)
		}
	}

	for _, t := range targets {
		n := g.Targets.Get(t)
		dfs(n)
	}
}
