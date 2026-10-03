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

package ninja

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// ParsedFiles is the collection of ninja files reachable from an entry file.
type ParsedFiles struct {
	EntryFile string
	Files     map[string]*NinjaFile
}

// NewFiles returns an empty ParsedFiles rooted at entryFile.
func NewFiles(entryFile string) *ParsedFiles {
	return &ParsedFiles{
		EntryFile: entryFile,
		Files:     make(map[string]*NinjaFile),
	}
}

// AddFile registers file in p under name.
func (p *ParsedFiles) AddFile(name string, file *NinjaFile) {
	p.Files[name] = file
}

// NinjaHandler is invoked by ParsedFiles.Call once per file.
type NinjaHandler func(ctx context.Context, file *NinjaFile) error

// Call invokes handler concurrently on every parsed file, starting from the
// entry file and traversing into subninjas.
func (p *ParsedFiles) Call(ctx context.Context, handler NinjaHandler) error {
	var step func(fn string) error
	step = func(fn string) error {
		f := p.Files[fn]
		if f == nil {
			return fmt.Errorf("cannot find %q in the collection of %d files", fn, len(p.Files))
		}
		if err := handler(ctx, f); err != nil {
			return err
		}
		grp := new(errgroup.Group)
		for _, fm := range f.Subninjas {
			grp.Go(func() error {
				return step(fm)
			})
		}
		return grp.Wait()
	}

	return step(p.EntryFile)
}

// BuildDir returns the first non-empty builddir defined in any parsed file.
func (p *ParsedFiles) BuildDir() string {
	for _, n := range p.Files {
		if n.BuildDir != "" {
			return n.BuildDir
		}
	}
	return ""
}

// BuildsCount returns the total number of build statements across all files.
func (p *ParsedFiles) BuildsCount() int {
	c := 0
	for _, n := range p.Files {
		c += len(n.Builds)
	}
	return c
}

// TargetsCount returns the total number of declared outputs across all builds.
func (p *ParsedFiles) TargetsCount() int {
	c := 0
	for _, n := range p.Files {
		for _, b := range n.Builds {
			c += len(b.Outputs)
			c += len(b.ImplicitOutputs)
		}
	}
	return c
}

// Defaults returns the concatenation of every "default" statement across all
// parsed files.
func (p *ParsedFiles) Defaults() []string {
	defaults := make([]string, 0)
	for _, n := range p.Files {
		defaults = append(defaults, n.Default...)
	}
	return defaults
}

// Pools returns the union of all pools defined across the parsed files,
// keyed by pool name.
func (p *ParsedFiles) Pools() map[string]*Pool {
	pools := make(map[string]*Pool)
	for _, n := range p.Files {
		for _, p := range n.Pools {
			pools[p.Name] = p
		}
	}
	return pools
}

// PruneAST clears the raw AST lists from all parsed files and disposes of the
// file map, allowing the Go garbage collector to reclaim unused build
// statements that are not part of the resolved build graph. After calling
// PruneAST, p must not be queried.
func (p *ParsedFiles) PruneAST() {
	if p == nil {
		return
	}
	for _, n := range p.Files {
		if n == nil {
			continue
		}
		n.Builds = nil
		n.KeyValues = nil
		n.Default = nil
		n.Subninjas = nil
		n.Pools = nil
		n.Rules = nil
		n.Scope = nil
		n.Parent = nil
	}
	p.Files = nil
}
