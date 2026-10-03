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
)

// defaultsCommandSpec describes the "defaults" subcommand.
var defaultsCommandSpec = &CommandSpec{
	name: "defaults",
	desc: "Output list of default targets",
	newCmd: func() Command {
		return new(DefaultsCommand)
	},
}

// DefaultsCommand executes the "defaults" subcommand.
type DefaultsCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *DefaultsCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	EnsureNoArgs(fl)

	ninjas, err := c.Parse(ctx)
	if err != nil {
		return err
	}

	for _, t := range ninjas.Defaults() {
		fmt.Println(t)
	}
	return nil
}
