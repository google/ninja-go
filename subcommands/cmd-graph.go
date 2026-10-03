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
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/pflag"

	"github.com/google/ninja-go/graph"
)

// graphCommandSpec describes the "graph" subcommand.
var graphCommandSpec = &CommandSpec{
	name:  "graph",
	desc:  "Output graphviz dot file for targets",
	usage: "<targets>...",
	newCmd: func() Command {
		return new(GraphCommand)
	},
}

// GraphCommand executes the "graph" subcommand.
type GraphCommand struct {
	BaseCommand
	seen map[*graph.Node]bool
}

// Run implements Command.
func (c *GraphCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	targets := EnsureArgs(fl)

	g, err := c.GenGraph(ctx, targets, false)
	if err != nil {
		return err
	}

	nodes := make(map[*graph.Node]struct{}, len(g.Nodes))
	for _, t := range targets {
		n := g.Targets.Get(t)
		if n == nil {
			return fmt.Errorf("target %s not found", t)
		}
		nodes[n] = struct{}{}
	}

	c.seen = make(map[*graph.Node]bool)
	c.start()
	for n := range nodes {
		c.addNode(n, g)
	}
	c.finish()

	return nil
}

// start writes the graphviz preamble.
func (c *GraphCommand) start() {
	c.Log.Printf("digraph ninja {")
	c.Log.Printf("rankdir=\"LR\"")
	c.Log.Printf("node [fontsize=10, shape=box, height=0.25]")
	c.Log.Printf("edge [fontsize=10]")
}

// addNode emits a graphviz node for n and recurses into its inputs.
func (c *GraphCommand) addNode(n *graph.Node, g *graph.Graph) {
	if c.seen[n] {
		return
	}
	c.seen[n] = true

	path := strings.ReplaceAll(n.String(), "\\", "/")
	c.Log.Printf("\"%p\" [label=\"%s\"]", n, path)

	for _, ins := range n.Build.AllInputs() {
		for _, i := range ins {
			if m := g.Targets.Get(i); m != nil {
				c.Log.Printf("\"%p\" -> \"%p\" [label=\"%s\"]", n, m, m.Build.RuleName)
				c.addNode(m, g)
			} else {
				h := xxhash.Sum64([]byte(i))
				c.Log.Printf("\"%p\" -> \"%x\"", n, h)
				c.Log.Printf("\"%x\" [label=\"%s\", shape=ellipse]", h, strings.ReplaceAll(i, "\\", "/"))
			}
		}
	}
}

// finish writes the closing graphviz brace.
func (c *GraphCommand) finish() {
	c.Log.Printf("}")
}
