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
	"errors"
	"fmt"
	"os"
	gruntime "runtime"
	"runtime/debug"
	"time"

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/graph"
	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/parse"
	"github.com/google/ninja-go/runtime"
	"github.com/google/ninja-go/third_party/siso/frontend"
)

// Command is the interface implemented by every subcommand. It is similar to
// "go.coder.com/cli", which we cannot use unmodified because it calls os.Exit
// on errors — undesirable behaviour in interactive and server modes.
type Command interface {
	Base() *BaseCommand
	RegisterFlags(fl *pflag.FlagSet)
	Run(ctx context.Context, fl *pflag.FlagSet) error
}

// CommandSpec describes a subcommand: its identity, how to display it, the
// modes it is available in, and how to construct a fresh instance.
type CommandSpec struct {
	name               string
	desc               string
	usage              string
	hidden             bool
	IgnoreGraph        bool
	DisableInteractive bool
	DisableHTTP        bool
	newCmd             func() Command
}

// BaseCommand provides the fields and methods shared by all subcommands.
type BaseCommand struct {
	Log    runtime.Logger
	params *runtime.RunParameters
}

// Base returns c so that BaseCommand satisfies the Command interface.
func (c *BaseCommand) Base() *BaseCommand {
	return c
}

// Params returns the runtime parameters associated with c.
func (c *BaseCommand) Params() *runtime.RunParameters {
	return c.params
}

// RegisterFlags is the default no-op flag registration. Subcommands override it
// when they need their own flags.
func (c *BaseCommand) RegisterFlags(fl *pflag.FlagSet) {
}

// Run is a placeholder used when no subcommand has been resolved.
func (c *BaseCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	return errors.New("this command should not be run")
}

// GeneralBackend is implemented by backends that need to initialize themselves
// before a subcommand runs.
type GeneralBackend interface {
	Init(params *runtime.RunParameters, cmd *string, args *[]string) error
}

// newLogger returns a Logger appropriate for the configured frontend output.
func newLogger(params *runtime.RunParameters) (runtime.Logger, error) {
	if params.CliFlags.FrontendFile != "" {
		return newFrontendLogger(params)
	}
	return newConsoleLogger(params)
}

// newConsoleLogger returns a Logger that writes to the terminal.
func newConsoleLogger(params *runtime.RunParameters) (runtime.Logger, error) {
	return runtime.NewConsole(params), nil
}

// newFrontendLogger returns a Logger that streams serialized build events to
// the file configured via --frontend_file.
func newFrontendLogger(params *runtime.RunParameters) (runtime.Logger, error) {
	f, err := os.OpenFile(params.CliFlags.FrontendFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open frontend file %q: %w", params.CliFlags.FrontendFile, err)
	}
	return frontend.NewFrontend(f, params.CliFlags.Verbosity, params.CliFlags.Quiet), nil
}

// RunCommand resolves cmdName, parses args, and runs the corresponding
// subcommand using the backend and parameters stored on ctx.
func RunCommand(ctx context.Context, cmdName string, args []string) error {
	backend := runtime.Get[GeneralBackend](ctx, runtime.BackendKey)
	params := runtime.Get[*runtime.RunParameters](ctx, runtime.RunParamsKey)

	if err := backend.Init(params, &cmdName, &args); err != nil {
		return err
	}

	spec := findCommandSpec(cmdName)
	if spec == nil {
		return fmt.Errorf("unknown command %s", cmdName)
	}

	name := params.Prefix + spec.name
	fl := pflag.NewFlagSet(name, pflag.ContinueOnError)
	cmd := spec.newCmd()

	log, err := newLogger(params)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer func() {
		err = log.Close()
		if err != nil {
			log.Errorf("failed to close frontend file: %v", err)
		}
	}()

	params.Log = log
	cmd.Base().Log = log
	cmd.Base().params = params
	fl.Usage = func() {
		log.Printf("Usage: %s %s", name, spec.usage)
		if spec.desc != "" {
			log.Printf("Description: %s", spec.desc)
		}
		if usage := fl.FlagUsages(); usage != "" {
			log.Printf("\nFlags:\n%s", usage)
		}
	}
	cmd.RegisterFlags(fl)

	if err := fl.Parse(args); err != nil {
		return err
	}

	return cmd.Run(ctx, fl)
}

// Parse loads the ninja files referenced by the command's parameters.
func (c *BaseCommand) Parse(ctx context.Context) (*ninja.ParsedFiles, error) {
	parseStart := time.Now()
	ninjas, err := parse.Parse(ctx, c.params.NinjaFile)
	if err != nil {
		return nil, err
	}
	parseEnd := time.Now()
	c.Log.Debugf("%% parse files=%d builds=%d in %s", len(ninjas.Files), ninjas.BuildsCount(), parseEnd.Sub(parseStart))
	return ninjas, nil
}

// GenGraph parses the ninja files and constructs the build graph for the given
// targets. If targets is empty, the defaults from the ninja files are used; if
// no defaults are defined, every build target is included.
func (c *BaseCommand) GenGraph(ctx context.Context, targets []string, withDeps bool) (*graph.Graph, error) {
	ninjas, err := c.Parse(ctx)
	if err != nil {
		return nil, err
	}

	graphStart := time.Now()
	g := graph.NewGraph()
	if len(targets) == 0 {
		// If no goal targets are specified, use the defaults from the ninja-build files.
		targets = ninjas.Defaults()
	}
	if len(targets) == 0 {
		// If no defaults are defined in the ninja-build files, build all targets.
		_ = ninjas.Call(ctx, func(ctx context.Context, file *ninja.NinjaFile) error {
			for _, b := range file.Builds {
				targets = append(targets, b.Outputs[0])
			}
			return nil
		})
	}
	if err := g.Load(ctx, ninjas, targets, withDeps); err != nil {
		return nil, err
	}
	graphEnd := time.Now()
	c.Log.Debugf("%% graph nodes=%d targets=%d files=%d in %s", g.Nodes.Len(), g.Targets.Len(), g.Files.Len(), graphEnd.Sub(graphStart))

	// Prune raw AST and unused build definitions in a background goroutine to free
	// gigabytes of unreferenced memory without blocking the start of build actions.
	go func(pf *ninja.ParsedFiles) {
		defer func() {
			if r := recover(); r != nil {
				c.Log.Errorf("panic during async AST pruning: %v\n%s", r, debug.Stack())
			}
		}()
		pf.PruneAST()
		pf = nil
		gruntime.GC()
	}(ninjas)
	ninjas = nil

	return g, nil
}
