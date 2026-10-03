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
	"errors"
	"log"
	"path/filepath"
	"strings"

	"github.com/google/ninja-go/runtime"
)

// ParseDepFile mmap-reads the depfile at fileName and parses its records.
func ParseDepFile(fileName string) ([]*DepsRecord, error) {
	buf, closeFile, err := runtime.ReadMMapFile(fileName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := closeFile(); err != nil {
			log.Printf("could not close mmap for %q: %v", fileName, err)
		}
	}()
	return ParseDeps(buf)
}

// DepsRecord is a single Makefile-style dependency record: one target and the
// list of files it depends on.
type DepsRecord struct {
	seenColon bool
	Target    string
	Deps      []string
}

// ParseDeps parses Makefile-style dependency records from buf.
func ParseDeps(buf []byte) ([]*DepsRecord, error) {
	var records []*DepsRecord

	var active *DepsRecord
	var token string
	i := 0
	for {
		token, i = depsNextToken(buf, i)
		switch token {
		case "":
			if active != nil {
				if !active.seenColon {
					return nil, errors.New("missing colon in deps record")
				}
				records = append(records, active)
			}
			return records, nil

		case "\n":
			if active != nil {
				if !active.seenColon {
					return nil, errors.New("missing colon in deps record")
				}
				records = append(records, active)
				active = nil
			}
		case ":":
			if active == nil {
				active = &DepsRecord{Target: ""}
			}
			if active.seenColon {
				return nil, errors.New("multiple colons in deps record")
			}
			active.seenColon = true
		default:
			if active != nil {
				active.Deps = append(active.Deps, token)
			} else {
				active = &DepsRecord{Target: token}
			}
		}
	}
}

// depsNextToken returns the next token in s starting at i, along with the
// position immediately after it.
func depsNextToken(s []byte, i int) (string, int) {
	for ; i < len(s); i++ {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i == len(s) {
			break
		}

		switch s[i] {
		case '\n':
			return "\n", i + 1
		case ':':
			return ":", i + 1
		case '#':
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		case '\\':
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
				continue
			}
			if i+2 < len(s) && s[i+1] == '\r' && s[i+2] == '\n' {
				i += 2
				continue
			}
			fallthrough
		default:
			return depsNextName(s, i)
		}
	}
	return "", i
}

// depsNextName reads a single (possibly backslash-escaped) name token from s.
func depsNextName(s []byte, i int) (string, int) {
	var sb strings.Builder
	for ; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', ':':
			return sb.String(), i
		case '\\':
			if i+1 < len(s) {
				i++
				sb.WriteByte(s[i])
				continue
			}
			fallthrough
		default:
			sb.WriteByte(s[i])
		}
	}
	return sb.String(), i
}

// ReadDepFile reads a .d file and returns the dependencies of the first
// target in "targets" that appears in it.
func ReadDepFile(targets []string, depFile string) ([]string, error) {
	depsList, err := ParseDepFile(depFile)
	if err != nil {
		return nil, err
	}
	return ResolveDeps(targets, depsList)
}

// ResolveDeps returns the dependencies of the first record in depsList whose
// target matches one of targets, with each dependency resolved through any
// symlinks.
func ResolveDeps(targets []string, depsList []*DepsRecord) ([]string, error) {
	if len(depsList) == 0 {
		return nil, nil
	}

	targetsSet := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		targetsSet[t] = struct{}{}
	}
	for _, deps := range depsList {
		if _, ok := targetsSet[deps.Target]; !ok {
			continue
		}
		for i, d := range deps.Deps {
			d, err := filepath.EvalSymlinks(d)
			if err != nil {
				return nil, err
			}
			deps.Deps[i] = d
		}
		return deps.Deps, nil
	}
	return nil, nil
}
