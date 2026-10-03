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

// maxPathsWithoutForce caps how many dependency paths are printed when -f is
// not given.
const maxPathsWithoutForce = 10

// pathCommandSpec describes the "path" subcommand.
var pathCommandSpec = &CommandSpec{
	name:  "path",
	desc:  "Find dependency path between two targets",
	usage: "[-a] [-f] [-c] [-e <path>]... <target-1> <target-2>",
	newCmd: func() Command {
		return new(PathCommand)
	},
}

// PathCommand executes the "path" subcommand.
type PathCommand struct {
	BaseCommand
	allPaths     bool
	force        bool
	count        bool
	excludes     []string
	excludesMap  map[string]struct{}
	excludeNodes map[*graph.Node]struct{}
}

// RegisterFlags implements Command.
func (c *PathCommand) RegisterFlags(fl *pflag.FlagSet) {
	fl.BoolVarP(&c.allPaths, "all", "a", false, "Find all dependency paths.")
	fl.StringSliceVarP(&c.excludes, "exclude", "e", nil, "Ignore dependency paths that include the given path (can be added multiple times).")
	fl.BoolVarP(&c.force, "force", "f", false, "Show all paths even if there are too many paths.")
	fl.BoolVarP(&c.count, "count", "c", false, "Count all dependency paths.")
}

// Run implements Command.
func (c *PathCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	target1, target2 := EnsureTwoArgs(fl)

	if !c.allPaths && c.force {
		return fmt.Errorf("-f option requires -a option")
	}
	if !c.allPaths && c.count {
		return fmt.Errorf("-c option requires -a option")
	}

	g, err := c.GenGraph(ctx, []string{target1}, false)
	if err != nil {
		return err
	}

	c.excludesMap = make(map[string]struct{}, len(c.excludes))
	c.excludeNodes = make(map[*graph.Node]struct{}, len(c.excludes))
	for _, p := range c.excludes {
		c.excludesMap[p] = struct{}{}
		if n := g.Targets.Get(p); n != nil {
			c.excludeNodes[n] = struct{}{}
		}
	}

	if !c.allPaths {
		return c.runSinglePath(g, target1, target2)
	}
	return c.runAllPaths(g, target1, target2)
}

// runAllPaths counts and optionally prints every dependency path from target
// to goal.
func (c *PathCommand) runAllPaths(g *graph.Graph, target, goal string) error {
	goalNode := g.Targets.Get(goal)

	pathCount := make(map[*graph.Node]int, len(g.Nodes))

	var mark func(n *graph.Node) int
	mark = func(n *graph.Node) int {
		if c, ok := pathCount[n]; ok {
			return c
		}
		if _, ok := c.excludeNodes[n]; ok {
			pathCount[n] = 0
			return 0
		}
		if n == goalNode {
			pathCount[n] = 1
			return 1
		}

		count := 0
		for _, ins := range n.Build.AllInputs() {
			for _, i := range ins {
				if _, ok := c.excludesMap[i]; ok {
					continue
				}

				if m := g.Targets.Get(i); m != nil {
					count += mark(m)
				} else if goal == i {
					count++
				}
			}
		}
		pathCount[n] = count
		return count
	}
	cc := mark(g.Targets.Get(target))

	if c.count {
		c.Log.Printf("%d", cc)
		return nil
	}
	if cc == 0 {
		return fmt.Errorf("%s does not depend on %s", target, goal)
	}
	if !c.force && cc > maxPathsWithoutForce {
		return fmt.Errorf("there are so many dependency paths (%d), to print them all use `-f` flag", cc)
	}

	stack := []string{target}
	printStack := func() {
		for _, n := range stack {
			c.Log.Printf("%s\n", n)
		}
		c.Log.Printf("\n")
	}

	var dfs func(n *graph.Node)
	dfs = func(n *graph.Node) {
		if n == goalNode {
			printStack()
			return
		}
		if pathCount[n] == 0 {
			return
		}

		for _, ins := range n.Build.AllInputs() {
			for _, i := range ins {
				stack = append(stack, i)
				if m := g.Targets.Get(i); m != nil {
					dfs(m)
				} else if goal == i {
					printStack()
				}
				stack = stack[:len(stack)-1]
			}
		}
	}

	dfs(g.Targets.Get(target))

	return nil
}

// runSinglePath prints one dependency path from target to goal, or reports
// that no such path exists.
func (c *PathCommand) runSinglePath(g *graph.Graph, target, goal string) error {
	queue := []string{target}
	mark := make(map[string]string)
	for len(queue) > 0 && mark[goal] == "" {
		v := queue[0]
		queue = queue[1:]

		n := g.Targets.Get(v)
		if n == nil {
			continue
		}
		if _, ok := c.excludeNodes[n]; ok {
			continue
		}

		for _, ins := range n.Build.InputsWithValidations() {
			for _, w := range ins {
				if _, ok := c.excludesMap[w]; ok {
					continue
				}

				if mark[w] == "" {
					mark[w] = v
					queue = append(queue, w)
				}
				if w == goal {
					break
				}
			}
		}
	}

	if mark[goal] == "" {
		return fmt.Errorf("%s does not depend on %s", target, goal)
	}

	stack := make([]string, 0)
	v := goal
	for v != "" {
		stack = append(stack, v)
		v = mark[v]
	}

	for i := range stack {
		c.Log.Printf("%s\n", stack[len(stack)-i-1])
	}
	return nil
}
