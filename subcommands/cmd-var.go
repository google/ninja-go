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

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/ninja"
)

// varCommandSpec describes the "var" subcommand.
var varCommandSpec = &CommandSpec{
	name:  "var",
	desc:  "Show the value of the specified variable",
	usage: "<var>...",
	newCmd: func() Command {
		return new(VarCommand)
	},
}

// VarCommand executes the "var" subcommand.
type VarCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *VarCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	vars := EnsureArgs(fl)

	ninjas, err := c.Parse(ctx)
	if err != nil {
		return err
	}

	for _, key := range vars {
		c.PrintVar(key, ninjas)
	}
	return nil
}

// PrintVar prints the value of key in every ninja file that defines it.
func (c *VarCommand) PrintVar(key string, ninjas *ninja.ParsedFiles) {
	for nf, n := range ninjas.Files {
		value, found := n.Scope.LocalLookup(key)
		if found {
			c.Log.Printf("[%s]: %s=%s", nf, key, value)
		}
	}
}
