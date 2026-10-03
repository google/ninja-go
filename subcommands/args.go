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
	"os"

	"github.com/spf13/pflag"
)

// EnsureNoArgs prints usage and exits if fl has any positional arguments.
func EnsureNoArgs(fl *pflag.FlagSet) {
	if fl.NArg() != 0 {
		fl.Usage()
		os.Exit(1)
	}
}

// EnsureArgs prints usage and exits if fl has no positional arguments;
// otherwise it returns them.
func EnsureArgs(fl *pflag.FlagSet) []string {
	if fl.NArg() == 0 {
		fl.Usage()
		os.Exit(1)
	}
	return fl.Args()
}

// EnsureSingleArg prints usage and exits unless fl has exactly one positional
// argument; otherwise it returns that argument.
func EnsureSingleArg(fl *pflag.FlagSet) string {
	if fl.NArg() != 1 {
		fl.Usage()
		os.Exit(1)
	}
	return fl.Arg(0)
}

// EnsureTwoArgs prints usage and exits unless fl has exactly two positional
// arguments; otherwise it returns them.
func EnsureTwoArgs(fl *pflag.FlagSet) (string, string) {
	if fl.NArg() != 2 {
		fl.Usage()
		os.Exit(1)
	}
	return fl.Arg(0), fl.Arg(1)
}
