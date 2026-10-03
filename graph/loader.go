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

package graph

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
	"github.com/google/ninja-go/xxmap"
	"golang.org/x/sync/errgroup"
)

// graphLoader walks the build graph from a set of goals, populating the maps
// used by Graph.Load.
type graphLoader struct {
	ninjas     *ninja.ParsedFiles
	backend    GraphBackend
	allRules   map[string]map[string]*ninja.Rule
	allTargets xxmap.StringMap[*ninja.Build]
	todo       chan *Node
	wg         sync.WaitGroup

	WithDeps bool
	Targets  xxmap.StringMap[*Node]
	Nodes    xxmap.ObjectsSet[Node]
	Files    xxmap.StringsSet
	Goals    xxmap.ObjectsSet[Node]
}

// dfs returns the Node that produces target t, creating one on first visit.
// It returns (nil, nil) if t is a source file rather than a build output.
func (gl *graphLoader) dfs(t string) (*Node, error) {
	build := gl.allTargets.Get(t)
	if build == nil {
		gl.Files.Put(t)
		return nil, nil
	}
	if n := gl.Targets.Get(t); n != nil {
		return n, nil
	}

	n := newNode(build)
	if !gl.Targets.PutIfMissing(build.Outputs[0], n) {
		return gl.Targets.Get(build.Outputs[0]), nil
	}
	for _, outs := range build.AllOutputs() {
		for _, o := range outs {
			n.AddTarget(o)
			gl.Targets.Put(o, n)
		}
	}
	gl.Nodes.Put(n)
	if build.RuleName != "" {
		// Search the parent ninja files for the rule if it is missing.
		m := build.NinjaFile
		for {
			build.Rule = gl.allRules[m][build.RuleName]
			if build.Rule != nil {
				break
			}
			f := gl.ninjas.Files[m].Parent
			if f == nil {
				break
			}
			m = f.FileName
		}
		if build.Rule == nil {
			return nil, fmt.Errorf("cannot find rule %q for build %s", build.RuleName, build.Outputs[0])
		}
	}
	gl.wg.Add(1)
	gl.todo <- n
	return n, nil
}

// handle resolves the inputs, validations, and (optionally) recorded deps of
// n by walking outward from it.
func (gl *graphLoader) handle(n *Node) error {
	defer gl.wg.Done()

	n.lock.Lock()
	defer n.lock.Unlock()

	build := n.Build
	for _, ins := range build.AllInputs() {
		for _, i := range ins {
			m, err := gl.dfs(i)
			if err != nil {
				return err
			}
			if m == nil { // Source file, not a build output.
				continue
			}
			n.Inputs[m]++
			m.lock.Lock()
			m.Outputs[n]++
			m.lock.Unlock()
		}
	}
	if len(build.Validations) > 0 {
		n.Validations = make(map[*Node]struct{})
	}
	for _, v := range build.Validations {
		m, err := gl.dfs(v)
		if err != nil {
			return err
		}
		if m == nil {
			return fmt.Errorf("no build for verification %q of %q", v, build.Outputs[0])
		}
		n.Validations[m] = struct{}{}
	}
	if gl.WithDeps && build.Rule != nil {
		deps, err := gl.backend.GetDeps(n)
		if err != nil {
			return err
		}
		p := 0
		for i := 0; i < len(deps); i++ {
			d := deps[i]
			if dn := gl.allTargets.Get(d); dn != nil {
				m, err := gl.dfs(d)
				if err != nil {
					return err
				}
				if m == nil { // File
					log.Printf("%s's depfile contains %s which is a target but not a dependency of depfile's target", build.Outputs[0], d)
					continue
				}
				n.Inputs[m]++
				m.lock.Lock()
				m.Outputs[n]++
				m.lock.Unlock()
				continue
			}
			gl.Files.Put(d)
			deps[p] = d
			p++
		}
		n.Deps = deps[0:p]
	}
	return nil
}

// load drives the parallel dfs/handle workers from the given goals.
func (gl *graphLoader) load(ctx context.Context, goals []string) error {
	gl.backend = runtime.Get[GraphBackend](ctx, runtime.BackendKey)
	params := runtime.Get[*runtime.RunParameters](ctx, runtime.RunParamsKey)

	cpus := params.CliFlags.ConcurrencyLevel

	grp := new(errgroup.Group)
	for i := 0; i < cpus; i++ {
		grp.Go(func() error {
			var errs error
			for v := range gl.todo {
				if err := gl.handle(v); err != nil {
					errs = errors.Join(errs, err)
				}
			}
			return errs
		})
	}

	for _, t := range goals {
		if strings.HasSuffix(t, "^") {
			base := strings.TrimSuffix(t, "^")
			resolved := gl.resolveCaretTarget(base)
			if resolved == "" {
				return fmt.Errorf("cannot find target %q", t)
			}
			t = resolved
		}
		n, err := gl.dfs(t)
		if err != nil {
			return err
		}
		if n == nil {
			return fmt.Errorf("cannot find target %q", t)
		}
		gl.Goals.Put(n)
	}

	gl.wg.Wait()
	close(gl.todo)
	return grp.Wait()
}

// resolveCaretTarget finds the first output of the build edge that consumes
// base as an input in manifest order, implementing the "target^" syntax.
func (gl *graphLoader) resolveCaretTarget(base string) string {
	return gl.resolveCaretIn(gl.ninjas.Files[gl.ninjas.EntryFile], base)
}

func (gl *graphLoader) resolveCaretIn(f *ninja.NinjaFile, base string) string {
	if f == nil {
		return ""
	}
	for _, b := range f.Builds {
		for _, in := range b.AllInputs() {
			for _, name := range in {
				if name == base {
					return b.Outputs[0]
				}
			}
		}
	}
	for _, sub := range f.Subninjas {
		if result := gl.resolveCaretIn(gl.ninjas.Files[sub], base); result != "" {
			return result
		}
	}
	return ""
}
