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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDepsParseSimple(t *testing.T) {
	deps, err := ParseDeps([]byte("target: dep1 dep2\n"))
	require.NoError(t, err)
	require.Equal(t, []*DepsRecord{{
		seenColon: true,
		Target:    "target",
		Deps:      []string{"dep1", "dep2"},
	}}, deps)
}

func TestDepsParseMulti(t *testing.T) {
	deps, err := ParseDeps([]byte("target1: dep1 dep2\ntarget2:dep3"))
	require.NoError(t, err)
	require.Equal(t, []*DepsRecord{{
		seenColon: true,
		Target:    "target1",
		Deps:      []string{"dep1", "dep2"},
	}, {
		seenColon: true,
		Target:    "target2",
		Deps:      []string{"dep3"},
	}}, deps)
}

func TestDepsParseEmpty(t *testing.T) {
	deps, err := ParseDeps([]byte(""))
	require.NoError(t, err)
	require.Empty(t, deps)
}

func TestDepsParseJustSpace(t *testing.T) {
	deps, err := ParseDeps([]byte(" \t\n\n    "))
	require.NoError(t, err)
	require.Empty(t, deps)
}

func TestDepsParseMissingColon(t *testing.T) {
	_, err := ParseDeps([]byte("target dep1 dep2\n"))
	require.Error(t, err)
}

func TestDepsParseMultiColon(t *testing.T) {
	_, err := ParseDeps([]byte("target: dep1: dep2\n"))
	require.Error(t, err)
}

func TestDepsParseMultiColon2(t *testing.T) {
	_, err := ParseDeps([]byte("target1: dep1 \\\ntarget2: dep2\n"))
	require.Error(t, err)
}

func TestDepsParseComment(t *testing.T) {
	deps, err := ParseDeps([]byte("# a comment\ntarget: dep1 dep2 # this is another comment\n"))
	require.NoError(t, err)
	require.Equal(t, []*DepsRecord{{
		seenColon: true,
		Target:    "target",
		Deps:      []string{"dep1", "dep2"},
	}}, deps)
}

func TestDepsParseWithLineBreaks(t *testing.T) {
	deps, err := ParseDeps([]byte("target: \\\n dep1 \\\n dep2"))
	require.NoError(t, err)
	require.Equal(t, []*DepsRecord{{
		seenColon: true,
		Target:    "target",
		Deps:      []string{"dep1", "dep2"},
	}}, deps)
}

func TestDepsParseEscapedSpace(t *testing.T) {
	deps, err := ParseDeps([]byte("target: \\\n dep1\\ dep2"))
	require.NoError(t, err)
	require.Equal(t, []*DepsRecord{{
		seenColon: true,
		Target:    "target",
		Deps:      []string{"dep1 dep2"},
	}}, deps)
}
