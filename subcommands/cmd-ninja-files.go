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
	"sort"

	"github.com/spf13/pflag"
)

// ninjaFilesCommandSpec describes the "ninja_files" subcommand.
var ninjaFilesCommandSpec = &CommandSpec{
	name: "ninja_files",
	desc: "List all ninja files recursively loaded from the root one",
	newCmd: func() Command {
		return new(NinjaFilesCommand)
	},
}

// NinjaFilesCommand executes the "ninja_files" subcommand.
type NinjaFilesCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *NinjaFilesCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	EnsureNoArgs(fl)

	ninjas, err := c.Parse(ctx)
	if err != nil {
		return err
	}

	files := make([]string, 0, len(ninjas.Files))
	for f := range ninjas.Files {
		files = append(files, f)
	}

	sort.Strings(files)

	for _, f := range files {
		fmt.Println(f)
	}
	return nil
}
