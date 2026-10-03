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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookupWithPanic(key string) string {
	panic("this should not been called")
}

type mapLookup map[string]string

func (m mapLookup) lookup(key string) string {
	return m[key]
}

func TestDerefSimple(t *testing.T) {
	s := "some text with no dollar sign"
	r := ResolveVariables(s, lookupWithPanic)
	assert.Equal(t, s, r)
}

func TestEndDollar(t *testing.T) {
	r := ResolveVariables("$ end in $", lookupWithPanic)
	assert.Equal(t, " end in ", r)
}

func TestDerefSingles(t *testing.T) {
	r := ResolveVariables("$ text with$:$\tsingle $$", lookupWithPanic)
	assert.Equal(t, " text with:\tsingle $", r)
}

func TestDerefSinglesNotBoundies(t *testing.T) {
	r := ResolveVariables("text$ with$: single $$s", lookupWithPanic)
	assert.Equal(t, "text with: single $s", r)
}

func TestDerefVarNil(t *testing.T) {
	r := ResolveVariables("$text", mapLookup{}.lookup)
	assert.Empty(t, r)
}

func TestDerefVarSimple(t *testing.T) {
	r := ResolveVariables("$text", mapLookup{"text": "value"}.lookup)
	assert.Equal(t, "value", r)
}

func TestDerefVarContext(t *testing.T) {
	r := ResolveVariables("this is a $var inlined", mapLookup{"var": "value"}.lookup)
	assert.Equal(t, "this is a value inlined", r)
}

func TestDerefVarNil2(t *testing.T) {
	r := ResolveVariables("${text}", mapLookup{}.lookup)
	assert.Empty(t, r)
}

func TestDerefVarSimple2(t *testing.T) {
	r := ResolveVariables("${text}", mapLookup{"text": "value"}.lookup)
	assert.Equal(t, "value", r)
}

func TestDerefVarContext2(t *testing.T) {
	r := ResolveVariables("this is a ${var} inlined", mapLookup{"var": "value"}.lookup)
	assert.Equal(t, "this is a value inlined", r)
}

func TestDerefVarContextError(t *testing.T) {
	require.Panics(t, func() {
		ResolveVariables("this is a ${var inlined", mapLookup{"text": "value"}.lookup)
	})
}

// func TestCostumeVarContext(t *testing.T) {
// 	var resolver CustomResolverFunc = func(s string) string {
// 		if s == "out" {
// 			return "value1"
// 		}
// 		return ""
// 	}
// 	vars := []map[string]string{{"var": "var-value", "out": "out-value", "in_newline": "newline-value"}}
// 	r, err := ResolveVariables("var=$var out=$out in=$in in_newline=$in_newline end", vars, resolver)
// 	require.NoError(t, err)
// 	assert.Equal(t, "var=var-value out=value1 in= in_newline= end", r)
// }

// func TestDerefVars(t *testing.T) {
// 	r, err := ExpandToken("this is a ${var1} and $var2", []map[string]string{
// 		{"var1": "value"},
// 		{
// 			"var1": "wrong-value",
// 			"var2": "another value",
// 		},
// 	}, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, "this is a value and another value", r)
// }
