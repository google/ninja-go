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
)

// listCommandSpec describes the "list" subcommand.
var listCommandSpec = &CommandSpec{
	name: "list",
	desc: "List all available commands",
	newCmd: func() Command {
		return new(ListCommand)
	},
}

// ListCommand executes the "list" subcommand.
type ListCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *ListCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	EnsureNoArgs(fl)

	c.Log.Printf("%11s  %s", "Command", "Description")

	for _, spec := range CommandSpecs {
		if spec.hidden || c.params.IsHTTP && spec.DisableHTTP || c.params.IsInteractive && spec.DisableInteractive {
			continue
		}
		c.Log.Printf("%11s  %s", spec.name, spec.desc)
	}

	return nil
}
