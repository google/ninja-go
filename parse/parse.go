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

package parse

import (
	"context"
	"log"
	"sync"

	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
	"golang.org/x/sync/errgroup"
)

// Parse loads the ninja file at fn and every subninja it includes, returning
// the parsed and expanded result.
func Parse(ctx context.Context, fn string) (*ninja.ParsedFiles, error) {
	backend := runtime.Get[ParseBackend](ctx, runtime.BackendKey)

	var lock sync.Mutex
	ninjas := ninja.NewFiles(fn)

	var eg errgroup.Group
	var loadSingleFile func(string, *ninja.NinjaFile) error
	loadSingleFile = func(fn string, parent *ninja.NinjaFile) error {
		s, release, err := backend.ReadNinjaFile(fn)
		if err != nil {
			return err
		}
		defer func() {
			if err := release(); err != nil {
				log.Println("failed to release ninja file", fn)
			}
		}()

		n, err := ParseNinjaFile(ctx, fn, s, parent)
		if err != nil {
			return err
		}

		if err := ninja.ExpandNinja(ctx, n); err != nil {
			return err
		}

		lock.Lock()
		ninjas.AddFile(fn, n)
		lock.Unlock()

		for _, s := range n.Subninjas {
			eg.Go(func() error {
				return loadSingleFile(s, n)
			})
		}
		return nil
	}
	eg.Go(func() error {
		return loadSingleFile(fn, nil)
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if err := backend.Parsed(ninjas); err != nil {
		return nil, err
	}

	return ninjas, nil
}
