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

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/graph"
)

// queryCommandSpec describes the "query" subcommand.
var queryCommandSpec = &CommandSpec{
	name:  "query",
	desc:  "Show inputs/outputs for a path",
	usage: "<target>...",
	newCmd: func() Command {
		return new(QueryCommand)
	},
}

// QueryCommand executes the "query" subcommand.
type QueryCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *QueryCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	targets := EnsureArgs(fl)

	g, err := c.GenGraph(ctx, nil, false)
	if err != nil {
		return err
	}

	nodes := make([]*graph.Node, len(targets))
	for i, t := range targets {
		n := g.Targets.Get(t)
		if n == nil {
			return fmt.Errorf("target %s not found", t)
		}
		nodes[i] = n
	}

	for _, n := range nodes {
		c.Log.Printf("%s:", strings.Join(n.Targets, ",\n"))
		c.Log.Printf("  input: %s", n.Build.RuleName)
		for _, i := range n.Build.Inputs {
			c.Log.Printf("    %s", i)
		}
		for _, i := range n.Build.ImplicitInputs {
			c.Log.Printf("    | %s", i)
		}
		for _, i := range n.Build.OrderOnlyInputs {
			c.Log.Printf("    || %s", i)
		}
		if n.Build.Validations != nil {
			c.Log.Printf("  validations:")
			for _, v := range n.Build.Validations {
				c.Log.Printf("    %s", v)
			}
		}
		c.Log.Printf("  outputs:")
		for i := range n.Outputs {
			c.Log.Printf("    %s", i)
		}
	}
	return nil
}
