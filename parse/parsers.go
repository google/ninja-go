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
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
)

// Top-level statement prefixes recognised by parseSegment.
var (
	buildPrefix    = []byte("build ")
	rulePrefix     = []byte("rule ")
	poolPrefix     = []byte("pool ")
	defaultPrefix  = []byte("default ")
	subninjaPrefix = []byte("subninja ")
)

// minSegmentSize is the smallest size (in bytes) at which the parser will
// split a ninja file into parallel segments.
const minSegmentSize = 100

// ParseNinjaFile parses the contents of a ninja file. The input is split into
// segments and parsed in parallel using params.CliFlags.ConcurrencyLevel
// goroutines.
func ParseNinjaFile(ctx context.Context, name string, full []byte, parent *ninja.NinjaFile) (*ninja.NinjaFile, error) {
	params := runtime.Get[*runtime.RunParameters](ctx, runtime.RunParamsKey)
	cpus := params.CliFlags.ConcurrencyLevel

	segs := splitSegments(full, cpus)

	var wg sync.WaitGroup
	var lock sync.Mutex
	parsed := make([]*ninja.NinjaFile, len(segs))

	wg.Add(len(segs))
	cc := 0
	var err error
	for si, s := range segs {
		xcc := cc
		go func() {
			defer wg.Done()
			i, n, e := parseSegment(name, cc, s, parent)
			if err == nil && e != nil {
				err = fmt.Errorf("error in parsing %s:%d %w", name, xcc+i, e)
				return
			}
			lock.Lock()
			parsed[si] = n
			lock.Unlock()
		}()
		cc += len(s)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}

	n := mergeSegments(parsed)
	return n, nil
}

// splitSegments divides full into roughly j segments along statement
// boundaries so that each segment can be parsed independently.
func splitSegments(full []byte, j int) [][]byte {
	segSize := len(full) / j
	if segSize < minSegmentSize {
		return [][]byte{full}
	}
	result := make([][]byte, 0, j)

	p := 0
	n := segSize

	for ; n < len(full); n++ {
		if full[n] == '\n' && full[n-1] != '$' && n+1 < len(full) && full[n+1] != ' ' && full[n+1] != '\t' && full[n+1] != '$' && full[n+1] != '#' {
			result = append(result, full[p:n])
			p = n
			n += segSize
			if n >= len(full) {
				n = len(full) - 1
			}
		}
	}
	if n-p > 2 {
		result = append(result, full[p:n])
	}
	return result
}

// parseSegment parses a single segment of a ninja file. cc is the byte offset
// of s within the full file, used only for error messages.
func parseSegment(name string, cc int, s []byte, parent *ninja.NinjaFile) (int, *ninja.NinjaFile, error) {
	n := &ninja.NinjaFile{
		Parent:   parent,
		FileName: name,
	}
	ln := len(s)
	sp := false
	for i := 0; i < ln; i++ {
		switch s[i] {
		case ' ', '\t':
			sp = true
		case '#':
			for ; i < ln; i++ {
				if s[i] == '\n' {
					sp = false
					break
				}
			}
		case '\n':
			sp = false
		default:
			if sp {
				return i, nil, fmt.Errorf("unexpeted indentation at %s:%d", name, cc+i)
			}
			var err error
			switch {
			case bytes.HasPrefix(s[i:], poolPrefix):
				var p *ninja.Pool
				p, i, err = parsePool(s, i+len(poolPrefix))
				if err != nil {
					return i, nil, err
				}
				n.Pools = append(n.Pools, p)
			case bytes.HasPrefix(s[i:], rulePrefix):
				var r *ninja.Rule
				r, i, err = parseRule(s, i+len(rulePrefix))
				if err != nil {
					return i, nil, err
				}
				n.Rules = append(n.Rules, r)
			case bytes.HasPrefix(s[i:], buildPrefix):
				var b *ninja.Build
				b, i, err = parseBuild(s, i+len(buildPrefix))
				if err != nil {
					return i, nil, err
				}
				n.Builds = append(n.Builds, b)
			case bytes.HasPrefix(s[i:], subninjaPrefix):
				var r string
				r, i, err = expectSingleToken(s, i+len(subninjaPrefix))
				if err != nil {
					return i, nil, err
				}
				n.Subninjas = append(n.Subninjas, r)
			case bytes.HasPrefix(s[i:], defaultPrefix):
				var r []string
				r, i, err = parseTokens(s, i+len(defaultPrefix))
				if err != nil {
					return i, nil, err
				}
				n.Default = append(n.Default, r...)
			default:
				var kv *ninja.KeyValue
				kv, i, err = parseKeyValue(s, i)
				if err != nil {
					return i, nil, err
				}
				switch kv.Key {
				case "builddir":
					n.BuildDir = kv.Value
				case "ninja_required_version":
					n.NinjaRequiredVersion = kv.Value
				case "cflags":
					n.CFlags = kv.Value
				case "msvc_deps_prefix":
					n.MSVCDepsPrefix = kv.Value
				default:
					n.KeyValues = append(n.KeyValues, kv)
				}
			}
		}
	}
	return ln, n, nil
}

// mergeSegments combines the per-segment results back into a single NinjaFile.
func mergeSegments(parsed []*ninja.NinjaFile) *ninja.NinjaFile {
	n := parsed[0]
	for _, m := range parsed[1:] {
		if m.NinjaRequiredVersion != "" {
			n.NinjaRequiredVersion = m.NinjaRequiredVersion
		}
		if m.MSVCDepsPrefix != "" {
			n.MSVCDepsPrefix = m.MSVCDepsPrefix
		}
		if m.CFlags != "" {
			n.CFlags = m.CFlags
		}
		if m.BuildDir != "" {
			n.BuildDir = m.BuildDir
		}
		n.Subninjas = append(n.Subninjas, m.Subninjas...)
		n.Pools = append(n.Pools, m.Pools...)
		n.Rules = append(n.Rules, m.Rules...)
		n.Builds = append(n.Builds, m.Builds...)
		n.Default = append(n.Default, m.Default...)
		n.KeyValues = append(n.KeyValues, m.KeyValues...)
	}
	for _, b := range n.Builds {
		b.NinjaFile = n.FileName
	}
	return n
}

// expectSingleToken parses exactly one token from s starting at i.
func expectSingleToken(s []byte, i int) (string, int, error) {
	var r []string
	var err error
	r, i, err = parseTokens(s, i)
	if err != nil {
		return "", i, err
	}
	if len(r) != 1 {
		return "", i, fmt.Errorf("unexpected tokens %s", r)
	}
	return r[0], i, nil
}

// nextToken returns the next ninja token in s starting at i, handling escape
// sequences and the special separators ":", "|", "||", and "|@".
func nextToken(s []byte, i int) (string, int, error) {
	openBracket := false
	bracketClosed := func() error {
		if openBracket {
			return fmt.Errorf("unclosed '${'")
		}
		return nil
	}

	ln := len(s)
	st := -1
	for ; ; i++ {
		if i == ln || s[i] == '\n' {
			if st == -1 {
				return "", i, bracketClosed()
			}
			return string(s[st:i]), i, bracketClosed()
		}
		var next byte
		if i+1 < ln {
			next = s[i+1]
		}
		switch s[i] {
		case '$':
			switch next {
			case ' ', '\t':
				if st == -1 {
					st = i
				}
				i++
				continue
			case '\n':
				if st == -1 {
					i++
					continue
				}
				return string(s[st:i]), i, bracketClosed()
			case '{':
				openBracket = true
			default:
			}
		case ' ', '\t':
			if st == -1 {
				continue
			}
			return string(s[st:i]), i, bracketClosed()
		case ':':
			if st == -1 {
				return ":", i + 1, bracketClosed()
			}
			return string(s[st:i]), i, bracketClosed()
		case '|':
			if st != -1 {
				return string(s[st:i]), i, bracketClosed()
			}
			switch next {
			case '|':
				return "||", i + 2, bracketClosed()
			case '@':
				return "|@", i + 2, bracketClosed()
			default:
				return "|", i + 1, bracketClosed()
			}
		case '}':
			openBracket = false
		}
		if st == -1 {
			st = i
		}
	}
}

// parseTokens reads tokens from s until a newline or EOF.
func parseTokens(s []byte, i int) ([]string, int, error) {
	r := make([]string, 0)
	for {
		var t string
		var err error
		t, i, err = nextToken(s, i)
		if err != nil {
			return nil, i, err
		}
		if t == "" {
			break
		}
		r = append(r, t)
	}
	return r, i, nil
}

// parseKeyValue parses a single "key = value" line.
func parseKeyValue(s []byte, i int) (*ninja.KeyValue, int, error) {
	ln := len(s)
	eq := -1
	j := i
	openBracket := false
	for ; j < ln; j++ {
		if s[j] == '=' && eq == -1 {
			eq = j
		}

		if s[j] == '\n' && !(j-1 >= 0 && s[j-1] == '$') {
			break
		}
		if s[j] == '}' {
			openBracket = false
		}
		if s[j] == '{' && j-1 >= 0 && s[j-1] == '$' {
			openBracket = true
		}
	}
	if eq == -1 {
		return nil, j, fmt.Errorf("cannot find = sign in a key-value")
	}
	if openBracket {
		return nil, j, fmt.Errorf("unclosed '${'")
	}

	key := bytes.TrimSpace(s[i:eq])

	value := trimPrefixSpaces(s[eq+1 : j])
	if len(key) == 0 {
		return nil, j, fmt.Errorf("empty key is not valid")
	}
	return &ninja.KeyValue{Key: string(key), Value: string(value)}, j, nil
}

// trimPrefixSpaces returns s with any leading spaces removed.
func trimPrefixSpaces(s []byte) []byte {
	// Effectively bytes.TrimLeftFunc(s, unicode.IsSpace).
	for i, c := range s {
		if c != ' ' {
			return s[i:]
		}
	}
	return nil
}

// parseKeyValues parses an indented block of key-value lines.
func parseKeyValues(s []byte, i int) ([]*ninja.KeyValue, int, error) {
	r := make([]*ninja.KeyValue, 0)
	ln := len(s)
	for ; i < ln && (s[i] == ' ' || s[i] == '\t'); i++ {
		for i < ln && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i == ln {
			break
		}
		var err error
		var kv *ninja.KeyValue
		kv, i, err = parseKeyValue(s, i)
		if err != nil {
			return nil, i, err
		}
		r = append(r, kv)
	}
	return r, i - 1, nil
}

// parsePool parses a "pool" statement and its body.
func parsePool(s []byte, i int) (*ninja.Pool, int, error) {
	p := &ninja.Pool{}
	var err error
	p.Name, i, err = expectSingleToken(s, i)
	if err != nil {
		return nil, i, err
	}

	kvs, i, err := parseKeyValues(s, i+1)
	if err != nil {
		return nil, i, err
	}
	for _, kv := range kvs {
		switch kv.Key {
		case "depth":
			depth, err := strconv.Atoi(kv.Value)
			if err != nil {
				return nil, i, fmt.Errorf("cannot parse int in pool line: %s = %s", kv.Key, kv.Value)
			}
			p.Depth = depth
		default:
			return nil, i, fmt.Errorf("invalid line in pool %s", p.Name)
		}
	}
	return p, i, nil
}

// parseRule parses a "rule" statement and its body.
func parseRule(s []byte, i int) (*ninja.Rule, int, error) {
	r := &ninja.Rule{}
	var err error
	r.Name, i, err = expectSingleToken(s, i)
	if err != nil {
		return nil, i, err
	}

	kvs, i, err := parseKeyValues(s, i+1)
	if err != nil {
		return nil, i, err
	}
	for _, kv := range kvs {
		switch kv.Key {
		case "command":
			r.Command = kv.Value
		case "deps":
			r.Deps = kv.Value
		case "msvc_deps_prefix":
			r.MSVCDepsPrefix = kv.Value
		case "description":
			r.Description = kv.Value
		case "depfile":
			r.DepFile = kv.Value
		case "dyndep":
			r.DynDep = kv.Value
		case "generator":
			r.Generator = kv.Value
		case "pool":
			r.Pool = kv.Value
		case "restat":
			r.Restat = kv.Value
		case "rspfile":
			r.RSPFile = kv.Value
		case "rspfile_content":
			r.RSPFileContent = kv.Value
		case "phony_output":
			r.PhonyOutput = kv.Value
		case "symlink_outputs":
			r.SymlinkOutputs = strings.Fields(kv.Value)
		case "sandbox_disabled":
			r.SandboxDisabled = kv.Value
		case "remote_enabled", "remote_platform_ref", "remote_timeout", "restat_content":
			// Ignored for now.
		default:
			return nil, i, fmt.Errorf("invalid line in rule %s: %q", r.Name, kv.Key)
		}
	}
	return r, i, nil
}

// States used by parseBuild's small token-driven state machine.
const (
	buildStateOutputs = iota
	buildStateImplicitOutputs
	buildStateRule
	buildStateInputs
	buildStateImplicitInputs
	buildStateOrderOnlyInputs
	buildStateValidations
)

// parseBuild parses a "build" statement and its body.
func parseBuild(s []byte, i int) (*ninja.Build, int, error) {
	b := &ninja.Build{}
	var err error
	var t string

	state := buildStateOutputs
	for {
		t, i, err = nextToken(s, i)
		if err != nil {
			return b, i, err
		}
		if t == "" {
			break
		}
		switch t {
		case "|":
			switch state {
			case buildStateOutputs:
				state = buildStateImplicitOutputs
			case buildStateInputs:
				state = buildStateImplicitInputs
			default:
				return b, i, fmt.Errorf("unexpected '|' sign")
			}
		case ":":
			switch state {
			case buildStateOutputs, buildStateImplicitOutputs:
				state = buildStateRule
			default:
				return b, i, fmt.Errorf("unexpected ':' sign")
			}
		case "||":
			switch state {
			case buildStateInputs, buildStateImplicitInputs:
				state = buildStateOrderOnlyInputs
			default:
				return b, i, fmt.Errorf("unexpected '||' sign")
			}
		case "|@":
			switch state {
			case buildStateInputs, buildStateImplicitInputs, buildStateOrderOnlyInputs:
				state = buildStateValidations
			default:
				return b, i, fmt.Errorf("unexpected '|@' sign")
			}
		default:
			switch state {
			case buildStateOutputs:
				b.Outputs = append(b.Outputs, t)
			case buildStateImplicitOutputs:
				b.ImplicitOutputs = append(b.ImplicitOutputs, t)
			case buildStateRule:
				b.RuleName = t
				state = buildStateInputs
			case buildStateInputs:
				b.Inputs = append(b.Inputs, t)
			case buildStateImplicitInputs:
				b.ImplicitInputs = append(b.ImplicitInputs, t)
			case buildStateOrderOnlyInputs:
				b.OrderOnlyInputs = append(b.OrderOnlyInputs, t)
			case buildStateValidations:
				b.Validations = append(b.Validations, t)
			}
		}
	}

	kvs, i, err := parseKeyValues(s, i+1)
	if err != nil {
		return nil, i, err
	}
	b.KeyValues = append(b.KeyValues, kvs...)
	return b, i, nil
}
