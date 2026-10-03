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

package ninja

import (
	"testing"
)

func TestPruneAST_Basic(t *testing.T) {
	parsed := NewFiles("build.ninja")
	nf := &NinjaFile{
		FileName: "build.ninja",
		Builds: []*Build{
			{Outputs: []string{"out1"}, RuleName: "cc"},
			{Outputs: []string{"out2"}, RuleName: "link"},
		},
		KeyValues: []*KeyValue{
			{Key: "cflags", Value: "-O2"},
		},
		Default:   []string{"out1"},
		Subninjas: []string{"sub.ninja"},
		Pools: []*Pool{
			{Name: "local", Depth: 4},
		},
		Rules: []*Rule{
			{Name: "cc", Command: "gcc -c $in -o $out"},
		},
	}
	parsed.AddFile("build.ninja", nf)

	if got, want := len(parsed.Files), 1; got != want {
		t.Fatalf("parsed.Files count: got %d, want %d", got, want)
	}
	if got, want := parsed.BuildsCount(), 2; got != want {
		t.Fatalf("parsed.BuildsCount(): got %d, want %d", got, want)
	}

	parsed.PruneAST()

	if got := parsed.Files; got != nil {
		t.Errorf("parsed.Files after PruneAST: got %v, want nil", got)
	}
	if got := nf.Builds; got != nil {
		t.Errorf("nf.Builds: got %v, want nil", got)
	}
	if got := nf.KeyValues; got != nil {
		t.Errorf("nf.KeyValues: got %v, want nil", got)
	}
	if got := nf.Default; got != nil {
		t.Errorf("nf.Default: got %v, want nil", got)
	}
	if got := nf.Subninjas; got != nil {
		t.Errorf("nf.Subninjas: got %v, want nil", got)
	}
	if got := nf.Pools; got != nil {
		t.Errorf("nf.Pools: got %v, want nil", got)
	}
	if got := nf.Rules; got != nil {
		t.Errorf("nf.Rules: got %v, want nil", got)
	}
	if got := nf.Scope; got != nil {
		t.Errorf("nf.Scope: got %v, want nil", got)
	}
	if got := nf.Parent; got != nil {
		t.Errorf("nf.Parent: got %v, want nil", got)
	}
}

func TestPruneAST_NilReceiver(t *testing.T) {
	var pf *ParsedFiles
	// PruneAST must be safe on nil receiver without panicking.
	pf.PruneAST()
}

func TestPruneAST_NilFileEntry(t *testing.T) {
	parsed := NewFiles("build.ninja")
	parsed.Files["nil_entry.ninja"] = nil
	// PruneAST must handle nil elements in Files map safely.
	parsed.PruneAST()

	if got := parsed.Files; got != nil {
		t.Errorf("parsed.Files after PruneAST: got %v, want nil", got)
	}
}

func TestPruneAST_MultipleFiles(t *testing.T) {
	parsed := NewFiles("build.ninja")
	nfRoot := &NinjaFile{
		FileName: "build.ninja",
		Builds: []*Build{
			{Outputs: []string{"main.o"}, RuleName: "cc"},
		},
		Subninjas: []string{"sub1.ninja", "sub2.ninja"},
	}
	nfSub1 := &NinjaFile{
		FileName: "sub1.ninja",
		Builds: []*Build{
			{Outputs: []string{"lib1.a"}, RuleName: "ar"},
		},
	}
	nfSub2 := &NinjaFile{
		FileName: "sub2.ninja",
		Builds: []*Build{
			{Outputs: []string{"lib2.a"}, RuleName: "ar"},
		},
	}
	parsed.AddFile("build.ninja", nfRoot)
	parsed.AddFile("sub1.ninja", nfSub1)
	parsed.AddFile("sub2.ninja", nfSub2)

	if got, want := parsed.BuildsCount(), 3; got != want {
		t.Fatalf("parsed.BuildsCount() before PruneAST: got %d, want %d", got, want)
	}

	parsed.PruneAST()

	if got := parsed.Files; got != nil {
		t.Errorf("parsed.Files after PruneAST: got %v, want nil", got)
	}
	for _, nf := range []*NinjaFile{nfRoot, nfSub1, nfSub2} {
		if got := nf.Builds; got != nil {
			t.Errorf("nf(%s).Builds: got %v, want nil", nf.FileName, got)
		}
		if got := nf.Subninjas; got != nil {
			t.Errorf("nf(%s).Subninjas: got %v, want nil", nf.FileName, got)
		}
	}
}

func TestPruneAST_BuildScopePreserved(t *testing.T) {
	fileScope := &NinjaScope{
		vars: map[string]string{
			"cc":     "clang",
			"cflags": "-O3",
		},
	}
	rule := &Rule{
		Name:    "compile",
		Command: "$cc $cflags -c $in -o $out $extra_flags",
	}

	build := &Build{
		Outputs:  []string{"output.o"},
		Inputs:   []string{"input.c"},
		RuleName: "compile",
		Rule:     rule,
		KeyValues: []*KeyValue{
			{Key: "extra_flags", Value: "-Wall"},
		},
	}
	build.Scope = fileScope.NewBuildScope(build)

	nf := &NinjaFile{
		FileName: "build.ninja",
		Builds:   []*Build{build},
		Rules:    []*Rule{rule},
		Scope:    fileScope,
	}
	parsed := NewFiles("build.ninja")
	parsed.AddFile("build.ninja", nf)

	// Prune AST
	parsed.PruneAST()

	// Verify that the Build object retained by the build graph can still
	// expand its command line using its internal BuildScope and Rule.
	gotCmd := build.Scope.ExpandField("command")
	wantCmd := "clang -O3 -c input.c -o output.o -Wall"
	if gotCmd != wantCmd {
		t.Errorf("build.Scope.ExpandField(\"command\"): got %q, want %q", gotCmd, wantCmd)
	}
}
