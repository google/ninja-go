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

	"github.com/google/ninja-go/ninja"
)

// dumpCommandSpec describes the "dump" subcommand.
var dumpCommandSpec = &CommandSpec{
	name:  "dump",
	desc:  "Show all information about the specified builds (ninjago command)",
	usage: "<targets>...",
	newCmd: func() Command {
		return new(DumpCommand)
	},
}

// DumpCommand executes the "dump" subcommand.
type DumpCommand struct {
	BaseCommand
}

// RegisterFlags implements Command.
func (c *DumpCommand) RegisterFlags(fl *pflag.FlagSet) {
}

// Run implements Command.
func (c *DumpCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	targets := EnsureArgs(fl)

	g, err := c.GenGraph(ctx, targets, false)
	if err != nil {
		return err
	}

	for _, t := range targets {
		n := g.Targets.Get(t)
		if n != nil {
			c.dump(n.Build)
		}
	}

	return nil
}

// ruleFields is the list of rule-level fields printed for each build.
var ruleFields = []string{
	"command",
	"description",
	"rspfile",
	"rspfile_content",
	"pool",
	"dyndep",
	"generator",
	"deps",
	"depfile",
	"msvc_deps_prefix",
	"restat",
	"phony_output",
	"sandbox_disabled",
}

// dump pretty-prints b in a form similar to its source ninja statement.
func (c *DumpCommand) dump(b *ninja.Build) {
	sb := new(strings.Builder)
	sb.WriteString("build ")
	for i, t := range b.Outputs {
		if i > 0 {
			sb.WriteString(" $\n    ")
		}
		sb.WriteString(t)
	}
	if len(b.ImplicitOutputs) > 0 {
		sb.WriteString(" | ")
		for i, t := range b.ImplicitOutputs {
			if i > 0 {
				sb.WriteString(" $\n    ")
			}
			sb.WriteString(t)
		}
	}

	fmt.Fprintf(sb, " $\n    : %s ", b.RuleName)

	for i, t := range b.Inputs {
		if i > 0 {
			sb.WriteString(" $\n    ")
		}
		sb.WriteString(t)
	}
	if len(b.ImplicitInputs) > 0 {
		sb.WriteString(" | ")
		for i, t := range b.ImplicitInputs {
			if i > 0 {
				sb.WriteString(" $\n    ")
			}
			sb.WriteString(t)
		}
	}
	if len(b.OrderOnlyInputs) > 0 {
		sb.WriteString(" || ")
		for i, t := range b.OrderOnlyInputs {
			if i > 0 {
				sb.WriteString(" $\n    ")
			}
			sb.WriteString(t)
		}
	}
	if len(b.Validations) > 0 {
		sb.WriteString(" |@ ")
		for i, t := range b.Validations {
			if i > 0 {
				sb.WriteString(" $\n    ")
			}
			sb.WriteString(t)
		}
	}

	sb.WriteString("\n")

	for _, kv := range b.KeyValues {
		fmt.Fprintf(sb, "      %s=%s\n", kv.Key, b.Scope.ExpandField(kv.Key))
	}

	for _, k := range ruleFields {
		v := b.Scope.ExpandField(k)
		if v == "" {
			continue
		}
		fmt.Fprintf(sb, "      %s=%s\n", k, v)
	}
	c.Log.Printf("%s", sb.String())
}
