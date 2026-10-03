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

package ninjago

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractCmdArgsWithCmd(t *testing.T) {
	args := []string{"-v", "arg1", "-f", "build.ninja", "arg2", "-t", "cmd", "arg3"}
	newArgs, cmdArgs := extractCmdArgs(args)
	require.Equal(t, []string{"-v", "arg1", "-f", "build.ninja", "arg2", "-t", "cmd"}, newArgs)
	require.Equal(t, []string{"arg3"}, cmdArgs)
}

func TestExtractCmdArgsBadCmd(t *testing.T) {
	args := []string{"-t"}
	newArgs, cmdArgs := extractCmdArgs(args)
	require.Equal(t, []string{"-t"}, newArgs)
	require.Equal(t, []string{}, cmdArgs)
}

func TestExtractCmdArgsNoCmd(t *testing.T) {
	args := []string{"-v", "arg1", "-f", "build.ninja", "arg2", "-n", "arg3"}
	newArgs, cmdArgs := extractCmdArgs(args)
	require.Equal(t, args, newArgs)
	require.Nil(t, cmdArgs)
}

func TestExtractCmdArgsConcatenated(t *testing.T) {
	args := []string{"-j4", "-tclean", "target1"}
	newArgs, cmdArgs := extractCmdArgs(args)
	require.Equal(t, []string{"-j4", "-t", "clean"}, newArgs)
	require.Equal(t, []string{"target1"}, cmdArgs)
}

func TestExtractCmdArgsConcatenatedNoToolArgs(t *testing.T) {
	args := []string{"-tgraph"}
	newArgs, cmdArgs := extractCmdArgs(args)
	require.Equal(t, []string{"-t", "graph"}, newArgs)
	require.Empty(t, cmdArgs)
}

func TestExtractCmdArgsNoArgs(t *testing.T) {
	newArgs, cmdArgs := extractCmdArgs(nil)
	require.Nil(t, newArgs)
	require.Nil(t, cmdArgs)
}
