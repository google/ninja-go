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
	"regexp"

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/graph"
)

// ruleQueryCommandSpec describes the "rule-query" subcommand.
var ruleQueryCommandSpec = &CommandSpec{
	name:  "rule-query",
	desc:  "Show all builds associated with the given rule",
	usage: "[-t <target>,...] [-r] <rule>",
	newCmd: func() Command {
		return new(ruleQueryCommand)
	},
}

// ruleQueryCommand executes the "rule-query" subcommand.
type ruleQueryCommand struct {
	BaseCommand
	targets []string
	regex   bool
}

// RegisterFlags implements Command.
func (c *ruleQueryCommand) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringSliceVarP(&c.targets, "target", "t", nil, "Goal targets for accessing build targets.")
	fl.BoolVarP(&c.regex, "regex", "r", false, "Consider the rule a regular expression and search rules based on that.")
}

// Run implements Command.
func (c *ruleQueryCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	rule := EnsureSingleArg(fl)

	g, err := c.GenGraph(ctx, c.targets, false)
	if err != nil {
		return err
	}

	if c.regex {
		return c.findRules(g, rule)
	}

	g.Nodes.GoRange(func(n *graph.Node) bool {
		if n.Build.RuleName == rule {
			c.Log.Printf("%s", n)
		}
		return true
	})

	return nil
}

// findRules prints distinct rule names that match the regular expression rule.
func (c *ruleQueryCommand) findRules(g *graph.Graph, rule string) error {
	regex, err := regexp.Compile(rule)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	g.Nodes.GoRange(func(n *graph.Node) bool {
		if regex.MatchString(n.Build.RuleName) {
			if _, ok := seen[n.Build.RuleName]; !ok {
				c.Log.Printf("%s", n.Build.RuleName)
				seen[n.Build.RuleName] = struct{}{}
			}
		}
		return true
	})
	return nil
}
