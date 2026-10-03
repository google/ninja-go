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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
)

func parse(ctx context.Context, files map[string]string, entry string) (*ninja.ParsedFiles, error) {
	ninjas := ninja.NewFiles(entry)
	var dfs func(f string, parent *ninja.NinjaFile) error
	dfs = func(f string, parent *ninja.NinjaFile) error {
		n, err := ParseNinjaFile(ctx, f, []byte(files[f]), parent)
		if err != nil {
			return err
		}
		ninjas.AddFile(f, n)
		if err := ninja.ExpandNinja(ctx, n); err != nil {
			return err
		}
		for _, s := range n.Subninjas {
			if err := dfs(s, n); err != nil {
				return err
			}
		}
		return nil
	}
	return ninjas, dfs(entry, nil)
}

func TestScopes(t *testing.T) {
	ctx := context.WithValue(context.Background(), runtime.RunParamsKey, &runtime.RunParameters{
		CliFlags: &runtime.CliFlags{
			ConcurrencyLevel: 10,
		},
	})

	files := map[string]string{
		"base.ninja": `
var1 = value1
var2 = value2
var3 = $var1-${var2}
subninja sub.ninja-$var3
	`, "sub.ninja-value1-value2": `
var4 = new$var3
rule r1
	command = echo hello $in

build b1: r1 in1 in2 in$var1
	`}

	ninjas, err := parse(ctx, files, "base.ninja")
	require.NoError(t, err)
	require.Len(t, ninjas.Files, 2)

	r := ninjas.Files["sub.ninja-value1-value2"].Rules[0]
	b := ninjas.Files["sub.ninja-value1-value2"].Builds[0]
	b.Rule = r
	assert.Equal(t, "echo hello in1 in2 invalue1", b.Scope.ExpandField("command"))
}
