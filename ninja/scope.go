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
	"fmt"
	"path"
	"strings"
)

// Scope resolves variable lookups within a ninja file or build.
type Scope interface {
	Parent() Scope
	Resolve(key string) string
	Expand(input string) string
}

// NinjaScope is a Scope that holds variables defined at the file level.
type NinjaScope struct {
	parent *NinjaScope
	vars   map[string]string
}

var _ Scope = (*NinjaScope)(nil)

// Parent implements Scope.
func (s *NinjaScope) Parent() Scope {
	return s.parent
}

// Resolve implements Scope.
func (s *NinjaScope) Resolve(key string) string {
	for s != nil {
		if v, ok := s.vars[key]; ok {
			return v
		}
		s = s.parent
	}
	return ""
}

// LocalLookup returns the value of key from this scope only, without
// consulting parent scopes.
func (s *NinjaScope) LocalLookup(key string) (string, bool) {
	v, ok := s.vars[key]
	return v, ok
}

// Expand implements Scope.
func (s *NinjaScope) Expand(input string) string {
	return ResolveVariables(input, s.Resolve)
}

// Add stores the value of key in this scope, overriding any previous value.
func (s *NinjaScope) Add(key, value string) {
	s.vars[key] = value
}

// NewNinjaScope returns a fresh NinjaScope nested inside s.
func (s *NinjaScope) NewNinjaScope() *NinjaScope {
	return &NinjaScope{
		parent: s,
		vars:   make(map[string]string),
	}
}

// NewBuildScope returns a BuildScope rooted at s and tied to build.
func (s *NinjaScope) NewBuildScope(build *Build) *BuildScope {
	return &BuildScope{
		parent: s,
		build:  build,
		cached: make([]*string, len(build.KeyValues)),
	}
}

// BuildScope is the variable scope for a single build statement.
type BuildScope struct {
	parent Scope
	build  *Build
	cached []*string
}

var _ Scope = (*BuildScope)(nil)

// Parent implements Scope.
func (s *BuildScope) Parent() Scope {
	return s.parent
}

// Resolve implements Scope.
func (s *BuildScope) Resolve(key string) string {
	return s.varLookupNoEscape(key)
}

// Expand implements Scope.
func (s *BuildScope) Expand(input string) string {
	return ResolveVariables(input, s.Resolve)
}

// localLookup resolves key from the build's "in"/"out" pseudo-variables and
// its KeyValues defined before index, with optional shell escaping.
func (s *BuildScope) localLookup(key string, escape bool, index int) (string, bool) {
	switch key {
	case "out":
		return joinPaths(s.build.Outputs, " ", escape), true
	case "in":
		return joinPaths(s.build.Inputs, " ", escape), true
	case "in_newline":
		return joinPaths(s.build.Inputs, "\n", escape), true
	}

	for index > 0 {
		index--
		kv := s.build.KeyValues[index]
		if kv.Key == key {
			if s.cached[index] != nil {
				return *s.cached[index], true
			}

			result := ResolveVariables(kv.Value, func(key string) string {
				return s.varLookup(key, escape, index)
			})

			s.cached[index] = &result
			return result, true
		}
	}
	return "", false
}

// varLookup resolves key, falling through to rule bindings and then
// the parent scope on miss.
func (s *BuildScope) varLookup(key string, escape bool, index int) string {
	if value, found := s.localLookup(key, escape, index); found {
		return value
	}
	if s.build.Rule != nil {
		if value := ruleLookup(key, s.build.Rule); value != "" {
			if escape {
				return ResolveVariables(value, s.varLookupEscape)
			}
			return ResolveVariables(value, s.varLookupNoEscape)
		}
	}
	return s.parent.Resolve(key)
}

// varLookupEscape is varLookup with shell escaping enabled.
func (s *BuildScope) varLookupEscape(key string) string {
	return s.varLookup(key, true, len(s.build.KeyValues))
}

// varLookupNoEscape is varLookup without shell escaping.
func (s *BuildScope) varLookupNoEscape(key string) string {
	return s.varLookup(key, false, len(s.build.KeyValues))
}

// ExpandField returns the fully expanded value of the given build field,
// falling back to the rule's value and finally the enclosing ninja scope.
func (s *BuildScope) ExpandField(key string) string {
	escape := fieldNeedsEscape(key)
	if value, found := s.localLookup(key, escape, len(s.build.KeyValues)); found {
		return value
	}

	if s.build.Rule != nil {
		if value := ruleLookup(key, s.build.Rule); value != "" {
			if escape {
				return ResolveVariables(value, s.varLookupEscape)
			}
			return ResolveVariables(value, s.varLookupNoEscape)
		}
	}

	return s.parent.Resolve(key)
}

// ExpandPathList expands every token in tokens as a path and rewrites it back
// in place after path.Clean.
func (s *BuildScope) ExpandPathList(tokens []string) error {
	for i, t := range tokens {
		tokens[i] = path.Clean(s.Expand(t))
	}
	return nil
}

// fieldNeedsEscape reports whether expansion of field should shell-escape its
// substituted values.
func fieldNeedsEscape(field string) bool {
	switch field {
	case "command", "rspfile_content", "deps", "pool":
		return true
	default:
		return false
	}
}

// ruleLookup returns the value of field on rule, or "" if there is no such
// field.
func ruleLookup(field string, rule *Rule) string {
	switch field {
	case "name":
		return rule.Name
	case "command":
		return rule.Command
	case "description":
		return rule.Description
	case "rspfile":
		return rule.RSPFile
	case "rspfile_content":
		return rule.RSPFileContent
	case "pool":
		return rule.Pool
	case "dyndep":
		return rule.DynDep
	case "generator":
		return rule.Generator
	case "deps":
		return rule.Deps
	case "depfile":
		return rule.DepFile
	case "msvc_deps_prefix":
		return rule.MSVCDepsPrefix
	case "restat":
		return rule.Restat
	case "phony_output":
		return rule.PhonyOutput
	default:
		return ""
	}
}

// InlineVar describes a single $var or ${var} reference detected in a token.
type InlineVar struct {
	Name  string
	Start int
}

// detectVars returns the list of inline variable references in token.
func detectVars(token string) []InlineVar {
	vars := make([]InlineVar, 0)
	p := strings.IndexByte(token, '$')
	if p == -1 {
		return vars
	}
	for p+1 < len(token) {
		switch token[p+1] {
		case '\t', '\n', ' ', '$', ':':
			vars = append(vars, InlineVar{Name: token[p+1 : p+2], Start: p})
			p += 2
		case '{':
			c := p + 2
			for c < len(token) && token[c] != '}' {
				c++
			}
			if c == len(token) {
				// This shouldn't happen, since the parser should already have
				// rejected unbalanced brackets.
				panic(fmt.Errorf("cannot find a closing bracket: %s", token))
			}
			vars = append(vars, InlineVar{Name: token[p+1 : c+1], Start: p})
			p = c + 1
		default:
			c := p + 1
			for c < len(token) && (token[c] >= 'a' && token[c] <= 'z' ||
				token[c] >= 'A' && token[c] <= 'Z' ||
				token[c] >= '0' && token[c] <= '9' ||
				token[c] == '@' || token[c] == '_') {
				c++
			}
			vars = append(vars, InlineVar{Name: token[p+1 : c], Start: p})
			p = c
		}
		for p < len(token) && token[p] != '$' {
			p++
		}
	}
	if p < len(token) {
		vars = append(vars, InlineVar{Name: "", Start: p})
	}
	return vars
}

// resolveVars substitutes the variable references in vars within token using
// lookup.
func resolveVars(token string, vars []InlineVar, lookup func(key string) string) string {
	if len(vars) == 0 { // Shortcut to avoid unnecessary allocations
		return token
	}
	newLen := len(token)
	values := make([]string, len(vars))
	for i, v := range vars {
		switch v.Name {
		case "\t", " ", "$", ":":
			values[i] = v.Name
		case "", "\n":
			// Just eat newlines.
		default:
			name := v.Name
			if name[0] == '{' {
				name = name[1 : len(name)-1]
			}

			values[i] = lookup(name)
		}

		newLen += len(values[i]) - len(v.Name) - 1
	}

	var r strings.Builder
	r.Grow(newLen)
	p := 0
	for i, v := range vars {
		r.WriteString(token[p:v.Start])
		r.WriteString(values[i])
		p = v.Start + len(v.Name) + 1
	}
	r.WriteString(token[p:])
	return r.String()
}

// ResolveVariables expands every $var/${var} reference in token using lookup.
func ResolveVariables(token string, lookup func(key string) string) string {
	return resolveVars(token, detectVars(token), lookup)
}

// joinPaths joins elems with sep, optionally shell-escaping each element.
func joinPaths(elems []string, sep string, escape bool) string {
	if len(elems) == 0 {
		return ""
	}
	n := (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i]) + len(sep)
	}

	var b strings.Builder
	b.Grow(n)
	for i, s := range elems {
		if i > 0 {
			b.WriteString(sep)
		}
		if escape {
			b.WriteString(ShellEscape(s))
		} else {
			b.WriteString(s)
		}
	}
	return b.String()
}

// ShellEscape returns s wrapped in single quotes if it contains any character
// that is unsafe in a POSIX shell command line.
func ShellEscape(s string) string {
	if !needsShellEscaping(s) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 2)
	quote := '\''
	b.WriteRune(quote)
	for _, c := range s {
		if c == quote {
			b.WriteString(`'\''`)
		} else {
			b.WriteRune(c)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

// needsShellEscaping reports whether input contains any character that
// requires shell quoting.
func needsShellEscaping(input string) bool {
	for _, c := range input {
		if !isShellSafeChar(c) {
			return true
		}
	}
	return false
}

// isShellSafeChar reports whether c may appear unquoted in a shell argument.
func isShellSafeChar(c rune) bool {
	isAlphaNumeric := 'A' <= c && c <= 'Z' ||
		'a' <= c && c <= 'z' ||
		'0' <= c && c <= '9'
	if isAlphaNumeric {
		return true
	}

	switch c {
	case '_', '+', '-', '.', '/':
		return true
	default:
		return false
	}
}
