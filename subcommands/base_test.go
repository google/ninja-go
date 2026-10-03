// Copyright 2026 Google LLC
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
	"os"
	"path/filepath"
	gruntime "runtime"
	"testing"

	"github.com/google/ninja-go/graph"
	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/parse"
	"github.com/google/ninja-go/runtime"
)

type mockBackend struct{}

func (m *mockBackend) ReadNinjaFile(file string) ([]byte, parse.ReleaseFileFunc, error) {
	b, err := os.ReadFile(file)
	return b, func() error { return nil }, err
}

func (m *mockBackend) Parsed(ninjas *ninja.ParsedFiles) error {
	return nil
}

func (m *mockBackend) GraphLoading() error {
	return nil
}

func (m *mockBackend) GraphLoaded(g *graph.Graph) error {
	return nil
}

func (m *mockBackend) GetDeps(n *graph.Node) ([]string, error) {
	return nil, nil
}

func TestGenGraph_PruneAndScopeIntegrity(t *testing.T) {
	tempDir := t.TempDir()
	ninjaContent := `
cflags = -O2

rule cc
  command = gcc -c $in -o $out $cflags

rule link
  command = gcc $in -o $out

build foo.o: cc foo.c
  cflags = -O3

build app: link foo.o

default app
`
	ninjaPath := filepath.Join(tempDir, "build.ninja")
	if err := os.WriteFile(ninjaPath, []byte(ninjaContent), 0644); err != nil {
		t.Fatalf("failed to write test ninja file: %v", err)
	}

	params := runtime.NewRunParams()
	params.NinjaFile = ninjaPath
	params.CliFlags = &runtime.CliFlags{
		ConcurrencyLevel: 4,
	}
	log := runtime.NewConsole(params)
	params.Log = log

	cmd := &BaseCommand{
		Log:    log,
		params: params,
	}

	backend := &mockBackend{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, runtime.BackendKey, backend)
	ctx = context.WithValue(ctx, runtime.RunParamsKey, params)

	g, err := cmd.GenGraph(ctx, nil, true)
	if err != nil {
		t.Fatalf("GenGraph failed: %v", err)
	}

	if g == nil {
		t.Fatal("expected non-nil graph, got nil")
	}

	// Verify target count
	if got, want := g.Targets.Len(), 2; got != want {
		t.Errorf("g.Targets.Len(): got %d, want %d", got, want)
	}

	// Verify app node and command expansion
	appNode := g.Targets.Get("app")
	if appNode == nil {
		t.Fatal("target 'app' not found in graph")
	}
	if appNode.Build == nil {
		t.Fatal("target 'app' has nil Build")
	}
	if got, want := appNode.Build.Scope.ExpandField("command"), "gcc foo.o -o app"; got != want {
		t.Errorf("app command: got %q, want %q", got, want)
	}

	// Verify foo.o node and scoped variable expansion
	fooNode := g.Targets.Get("foo.o")
	if fooNode == nil {
		t.Fatal("target 'foo.o' not found in graph")
	}
	if fooNode.Build == nil {
		t.Fatal("target 'foo.o' has nil Build")
	}
	if got, want := fooNode.Build.Scope.ExpandField("command"), "gcc -c foo.c -o foo.o -O3"; got != want {
		t.Errorf("foo.o command: got %q, want %q", got, want)
	}

	// Verify that after the background pruning goroutine has finished, the graph
	// and its scoped command expansion remain completely intact.
	gruntime.GC()
	if got, want := appNode.Build.Scope.ExpandField("command"), "gcc foo.o -o app"; got != want {
		t.Errorf("app command after GC: got %q, want %q", got, want)
	}
	if got, want := fooNode.Build.Scope.ExpandField("command"), "gcc -c foo.c -o foo.o -O3"; got != want {
		t.Errorf("foo.o command after GC: got %q, want %q", got, want)
	}
}
