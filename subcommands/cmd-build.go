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
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers.
	"os"
	"path"
	"time"

	"github.com/spf13/pflag"

	"github.com/google/ninja-go/build"
)

// buildCommandSpec describes the "build" subcommand.
var buildCommandSpec = &CommandSpec{
	name:               "build",
	desc:               "Builds the ninja file for the given targets (or default target)",
	usage:              "[<targets>...]",
	DisableHTTP:        true,
	DisableInteractive: true,
	newCmd: func() Command {
		return new(BuildCommand)
	},
}

// BuildCommand executes the "build" subcommand.
type BuildCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *BuildCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	return c.run(ctx, fl.Args())
}

// run drives the build engine for the given targets and exposes pprof on the
// .ninja_http unix socket while it runs.
func (c *BuildCommand) run(ctx context.Context, targets []string) error {
	graph, err := c.GenGraph(ctx, targets, true)
	if err != nil {
		return err
	}

	server := http.Server{
		Handler: http.DefaultServeMux,
	}
	go func() {
		p := path.Join(graph.BuildDir, ".ninja_http")
		os.Remove(p)
		unixListener, err := net.Listen("unix", p)
		if err != nil {
			log.Printf("Cannot listen on %q: %v", p, err)
			return
		}
		if err := server.Serve(unixListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Cannot serve on %q: %v", p, err)
		}
	}()
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Cannot shutdown server: %v", err)
		}
	}()

	buildStart := time.Now()
	e := build.NewEngine(ctx, graph)
	if err := e.Build(ctx); err != nil {
		return err
	}
	buildEnd := time.Now()
	c.Log.Debugf("%% built in %s", buildEnd.Sub(buildStart))

	return nil
}
