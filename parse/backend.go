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

// Package parse parses ninja build files and depfiles into the data
// structures consumed by the rest of ninjago, and defines the ParseBackend
// hook used to source file contents and observe the parsed result.
package parse

import "github.com/google/ninja-go/ninja"

// ReleaseFileFunc releases the resources associated with a previously read
// ninja file (e.g. unmaps a memory-mapped buffer).
type ReleaseFileFunc func() error

// ParseBackend is implemented by backends that supply ninja file contents to
// the parser and observe the parsed result.
type ParseBackend interface {
	ReadNinjaFile(file string) ([]byte, ReleaseFileFunc, error)
	Parsed(ninjas *ninja.ParsedFiles) error
}
