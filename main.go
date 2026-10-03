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

// Package ninjago is the top-level library entry point for the ninjago CLI:
// it parses flags, sets up signals and the runtime context, and dispatches
// the requested subcommand against the supplied backend.
package ninjago

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	gruntime "runtime"
	"syscall"

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/backends"
	"github.com/google/ninja-go/runtime"
	"github.com/google/ninja-go/subcommands"
)

// ninjagoVersion is the version string reported by --version.
const ninjagoVersion = "0.0.1"

// Main is the entry point of the ninjago CLI. It parses command-line flags,
// installs signal handlers, and dispatches to the requested subcommand using
// the supplied backend.
func Main(backend backends.Backend) int {
	version := false
	cwd := ""
	cliFlags := new(runtime.CliFlags)

	o := pflag.StringArrayP("o", "o", nil, "Specify the ninja options")
	w := pflag.StringArrayP("w", "w", nil, "Specify the ninja warning flags")
	d := pflag.StringArrayP("d", "d", nil, "Specify the debug modes")
	f := pflag.StringP("f", "f", "build.ninja", "Specify the ninja file")
	t := pflag.StringP("t", "t", "build", "Command to run")

	pflag.IntVarP(&cliFlags.ConcurrencyLevel, "j", "j", gruntime.NumCPU(), "The level of concurrency")
	pflag.IntVarP(&cliFlags.FailLimit, "k", "k", 1, "Keep going until N jobs fail (0 means infinity) [default=1]")
	pflag.BoolVarP(&cliFlags.DryRun, "n", "n", false, "Dry run (don't run commands but act like they succeeded)")
	pflag.BoolVar(&version, "version", false, fmt.Sprintf("Print ninja version (\"%s\")", ninjagoVersion))
	pflag.CountVarP(&cliFlags.Verbosity, "verbose", "v", "Show more details about builds (-v, -vv or -vvv)")
	pflag.BoolVarP(&cliFlags.Quiet, "quiet", "q", false, "Don't show progress status")
	pflag.StringVarP(&cwd, "C", "C", "", "Change to DIR before doing anything")
	pflag.StringVar(&cliFlags.FrontendCmd, "frontend", "", "Execute COMMAND and pass serialized build output to it")
	pflag.StringVar(&cliFlags.FrontendFile, "frontend_file", "", "Write serialized build output to FILE")

	pflag.Float64Var(&cliFlags.PSIMem, "psi-mem", 0, "Memory pressure threshold to pause scheduling")
	pflag.Float64Var(&cliFlags.PSICPU, "psi-cpu", 0, "CPU pressure threshold to pause scheduling")
	pflag.Float64Var(&cliFlags.PSIIO, "psi-io", 0, "IO pressure threshold to pause scheduling")

	newArgs, cmdArgs := extractCmdArgs(os.Args[1:])
	if err := pflag.CommandLine.Parse(newArgs); err != nil {
		log.Fatal(err)
	}

	if version {
		fmt.Println(ninjagoVersion)
		return 0
	}

	cmdArgs = append(cmdArgs, pflag.Args()...)

	if cliFlags.Verbosity > 2 {
		fmt.Println("ninja: Maximum verbosity exceeded, setting to 2.")
		cliFlags.Verbosity = 2
	}

	if err := cliFlags.LoadDebugs(*d); err != nil {
		log.Fatal(err)
	}
	if err := cliFlags.LoadWarns(*w); err != nil {
		log.Fatal(err)
	}
	if err := cliFlags.LoadOptions(*o); err != nil {
		log.Fatal(err)
	}

	if cwd != "" {
		if *t == "build" {
			fmt.Printf("ninja: Entering directory `%s'\n", cwd)
		}
		if err := os.Chdir(cwd); err != nil {
			log.Fatalf("failed to change directory to %s: %v", cwd, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and other termination signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("ninja: interrupted")
		cancel()
	}()

	ctx = context.WithValue(ctx, runtime.BackendKey, backend)
	err := run(ctx, *f, *t, cmdArgs, cliFlags, os.Stdout)
	if err != nil {
		// We can't print FAILED: unconditionally here because regular ninja doesn't do that in all error cases.
		fmt.Printf("ninja: %s.\n", err)
		return 1
	}

	return 0
}

// extractCmdArgs finds the -t flag and splits the argument list: everything
// up to and including "-t <command>" goes to flagArgs for pflag, while
// everything after becomes cmdArgs (tool-specific arguments). When no -t
// is present, all arguments go to pflag and positional args (build targets)
// are obtained via pflag.Args() by the caller.
func extractCmdArgs(args []string) (flagArgs []string, cmdArgs []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "-t" {
			i++
			if i < len(args) {
				i++
			}
			return args[:i], args[i:]
		}
		// Handle concatenated form: -tcommand
		if len(args[i]) > 2 && args[i][0] == '-' && args[i][1] == 't' {
			result := make([]string, i, i+2)
			copy(result, args[:i])
			result = append(result, "-t", args[i][2:])
			return result, args[i+1:]
		}
	}
	return args, nil
}

// run wires up runtime parameters and invokes the named subcommand.
func run(ctx context.Context, fn string, command string, args []string, cliFlags *runtime.CliFlags, w io.Writer) error {
	params := runtime.NewRunParams()
	params.CliFlags = cliFlags
	params.Out = w
	params.NinjaFile = fn

	ctx = context.WithValue(ctx, runtime.RunParamsKey, params)

	return subcommands.RunCommand(ctx, command, args)
}
