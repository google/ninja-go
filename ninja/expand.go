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

	"golang.org/x/sync/errgroup"

	"github.com/google/ninja-go/runtime"
)

// ExpandNinja installs scopes on n and expands every variable reference in
// its top-level fields, pools, defaults, subninjas, and builds.
func ExpandNinja(ctx context.Context, n *NinjaFile) error {
	if n.Parent == nil {
		n.Scope = &NinjaScope{vars: make(map[string]string)}
	} else {
		n.Scope = n.Parent.Scope.NewNinjaScope()
	}

	for _, kv := range n.KeyValues {
		kv.Value = n.Scope.Expand(kv.Value)
		n.Scope.Add(kv.Key, kv.Value)
	}

	for _, p := range n.Pools {
		p.Name = n.Scope.Expand(p.Name)
	}

	n.BuildDir = n.Scope.Expand(n.BuildDir)

	for i, t := range n.Default {
		n.Default[i] = n.Scope.Expand(t)
	}

	for i, s := range n.Subninjas {
		n.Subninjas[i] = n.Scope.Expand(s)
	}

	if err := expandBuilds(ctx, n); err != nil {
		return err
	}

	return nil
}

// expandBuilds expands the inputs and outputs of every build in n in parallel.
func expandBuilds(ctx context.Context, n *NinjaFile) error {
	params := runtime.Get[*runtime.RunParameters](ctx, runtime.RunParamsKey)
	cpus := params.CliFlags.ConcurrencyLevel

	grp := new(errgroup.Group)
	for i := 0; i < cpus; i++ {
		grp.Go(func() (err error) {
			for _, b := range Split(n.Builds, cpus, i) {
				b.Scope = n.Scope.NewBuildScope(b)

				if err := b.Scope.ExpandPathList(b.Outputs); err != nil {
					return err
				}
				if err := b.Scope.ExpandPathList(b.ImplicitOutputs); err != nil {
					return err
				}
				if err := b.Scope.ExpandPathList(b.Inputs); err != nil {
					return err
				}
				if err := b.Scope.ExpandPathList(b.ImplicitInputs); err != nil {
					return err
				}
				if err := b.Scope.ExpandPathList(b.OrderOnlyInputs); err != nil {
					return err
				}
				if err := b.Scope.ExpandPathList(b.Validations); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return grp.Wait()
}
