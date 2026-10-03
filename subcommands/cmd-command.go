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

// commandsCommandSpec describes the "commands" subcommand.
var commandsCommandSpec = &CommandSpec{
	name:  "commands",
	desc:  "List all commands required to rebuild given targets",
	usage: "[-s] <targets>...",
	newCmd: func() Command {
		return new(CommandsCommand)
	},
}

// CommandsCommand executes the "commands" subcommand.
type CommandsCommand struct {
	BaseCommand
	single bool
}

// RegisterFlags implements Command.
func (c *CommandsCommand) RegisterFlags(fl *pflag.FlagSet) {
	fl.BoolVarP(&c.single, "single", "s", false, "Only print the final command to build [target], not the whole chain")
}

// Run implements Command.
func (c *CommandsCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	targets := fl.Args() // May be empty.

	g, err := c.GenGraph(ctx, targets, false)
	if err != nil {
		return err
	}

	if c.single {
		for _, t := range targets {
			n := g.Targets.Get(t)
			if !n.Build.Rule.IsPhony() {
				fmt.Println(n.Command())
			}
		}
		return nil
	}

	seenNodes := make(map[*graph.Node]struct{}, len(g.Nodes))

	var dfs func(n *graph.Node)
	dfs = func(n *graph.Node) {
		if _, ok := seenNodes[n]; ok {
			return
		}
		seenNodes[n] = struct{}{}
		if !n.Build.Rule.IsPhony() {
			fmt.Println(n.Command())
		}

		for m := range n.Inputs {
			dfs(m)
		}
	}

	if len(targets) == 0 {
		targets = g.Defaults
	}
	for _, t := range targets {
		n := g.Targets.Get(t)
		dfs(n)
	}

	return nil
}
