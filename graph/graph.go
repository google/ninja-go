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
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
	"github.com/google/ninja-go/xxmap"
)

// Graph holds the resolved build graph and the metadata needed to drive a
// build.
type Graph struct {
	BuildStartTime time.Time
	GoalTargets    []string
	Defaults       []string
	BuildDir       string
	Pools          map[string]*ninja.Pool
	Goals          xxmap.ObjectsSet[Node]
	Targets        xxmap.StringMap[*Node]
	Nodes          xxmap.ObjectsSet[Node]
	Files          xxmap.StringsSet
}

// NewGraph returns an empty Graph ready to be populated by Load.
func NewGraph() *Graph {
	return &Graph{}
}

// Load resolves the supplied goals against ninjas and populates g with the
// transitive build graph. If withDeps is true, the graph also includes
// dependencies recorded in .ninja_deps.
func (g *Graph) Load(ctx context.Context, ninjas *ninja.ParsedFiles, goals []string, withDeps bool) error {
	g.BuildStartTime = time.Now()
	g.BuildDir = ninjas.BuildDir()
	g.Defaults = ninjas.Defaults()
	g.Pools = ninjas.Pools()

	backend := runtime.Get[GraphBackend](ctx, runtime.BackendKey)
	if err := backend.GraphLoading(); err != nil {
		return err
	}

	rules := g.allRules(ctx, ninjas)
	targets, err := g.allTargets(ctx, ninjas)
	if err != nil {
		return err
	}

	gl := &graphLoader{
		ninjas:     ninjas,
		allRules:   rules,
		allTargets: targets,
		todo:       make(chan *Node, ninjas.BuildsCount()),
		WithDeps:   withDeps,
		Targets:    xxmap.NewStringMap[*Node](),
		Nodes:      xxmap.NewObjectsSet[Node](),
		Files:      xxmap.NewStringsSet(),
		Goals:      xxmap.NewObjectsSet[Node](),
	}

	if err := gl.load(ctx, goals); err != nil {
		return err
	}

	g.Targets = gl.Targets
	g.Nodes = gl.Nodes
	g.Files = gl.Files
	g.GoalTargets = goals
	g.Goals = gl.Goals

	if err := backend.GraphLoaded(g); err != nil {
		return err
	}

	return nil
}

// allTargets returns a map from every output path declared in ninjas to the
// build that produces it.
func (g *Graph) allTargets(ctx context.Context, ninjas *ninja.ParsedFiles) (xxmap.StringMap[*ninja.Build], error) {
	targets := xxmap.NewStringMap[*ninja.Build]()

	for _, f := range ninjas.Files {
		if err := g.addTargets(ctx, f, targets); err != nil {
			return nil, err
		}
	}
	return targets, nil
}

// addTargets indexes the outputs of every build in nf into targets, returning
// an error if the same output is produced by more than one build.
func (g *Graph) addTargets(ctx context.Context, nf *ninja.NinjaFile, targets xxmap.StringMap[*ninja.Build]) error {
	params := runtime.Get[*runtime.RunParameters](ctx, runtime.RunParamsKey)
	cpus := params.CliFlags.ConcurrencyLevel

	grp := new(errgroup.Group)
	for i := 0; i < cpus; i++ {
		grp.Go(func() error {
			for _, b := range ninja.Split(nf.Builds, cpus, i) {
				for _, outs := range b.AllOutputs() {
					for _, o := range outs {
						if !targets.PutIfMissing(o, b) {
							return fmt.Errorf("target %q appears in multiple builds", o)
						}
					}
				}
			}
			return nil
		})
	}
	return grp.Wait()
}

// allRules returns a map from each ninja file to the rules visible from it,
// including the implicit "phony" rule.
func (g *Graph) allRules(ctx context.Context, ninjas *ninja.ParsedFiles) map[string]map[string]*ninja.Rule {
	var lock sync.Mutex
	allRules := make(map[string]map[string]*ninja.Rule, len(ninjas.Files))

	_ = ninjas.Call(ctx, func(ctx context.Context, file *ninja.NinjaFile) error {
		rules := make(map[string]*ninja.Rule, len(file.Rules))

		for _, r := range file.Rules {
			rules[r.Name] = r
		}
		phony := &ninja.Rule{
			Name: "phony",
		}
		rules[phony.Name] = phony

		lock.Lock()
		defer lock.Unlock()
		allRules[file.FileName] = rules
		return nil
	})
	return allRules
}
