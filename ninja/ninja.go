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

// Package ninja models the ninja build language: parsed files with their
// pools, rules, builds, defaults, and key/value bindings, along with the
// scopes used to expand variables.
package ninja

import (
	"sort"
)

// KeyValue is a parsed key-value pair from a ninja file.
type KeyValue struct {
	Key   string
	Value string
}

// NinjaFile is the parsed representation of a single ninja file.
type NinjaFile struct {
	// Parent is the ninja file that included this one via a "subninja" statement.
	Parent               *NinjaFile  `json:"-"`
	FileName             string      `json:",omitempty"`
	NinjaRequiredVersion string      `json:",omitempty"`
	MSVCDepsPrefix       string      `json:",omitempty"`
	CFlags               string      `json:",omitempty"`
	BuildDir             string      `json:",omitempty"`
	Subninjas            []string    `json:",omitempty"`
	Pools                []*Pool     `json:",omitempty"`
	Rules                []*Rule     `json:",omitempty"`
	Builds               []*Build    `json:",omitempty"`
	Default              []string    `json:",omitempty"`
	KeyValues            []*KeyValue `json:",omitempty"`

	Scope *NinjaScope `json:"-"`
}

// Sort sorts every slice on n by the natural identifier (name or first
// output) so that two semantically equivalent files compare equal.
func (n *NinjaFile) Sort() {
	sort.Strings(n.Subninjas)
	sort.Strings(n.Default)
	sort.Slice(n.Pools, func(i, j int) bool {
		return n.Pools[i].Name < n.Pools[j].Name
	})
	sort.Slice(n.Rules, func(i, j int) bool {
		return n.Rules[i].Name < n.Rules[j].Name
	})
	sort.Slice(n.Builds, func(i, j int) bool {
		if len(n.Builds[i].Outputs) == 0 || len(n.Builds[j].Outputs) == 0 {
			return i < j
		}
		return n.Builds[i].Outputs[0] < n.Builds[j].Outputs[0]
	})
}

// Pool is a parsed "pool" statement.
type Pool struct {
	Name  string `json:",omitempty"`
	Depth int    `json:",omitempty"`
}

// Rule is a parsed "rule" statement.
type Rule struct {
	Name            string   `json:",omitempty"`
	Command         string   `json:",omitempty"`
	Description     string   `json:",omitempty"`
	RSPFile         string   `json:",omitempty"`
	RSPFileContent  string   `json:",omitempty"`
	Pool            string   `json:",omitempty"`
	DepFile         string   `json:",omitempty"`
	DynDep          string   `json:",omitempty"`
	Generator       string   `json:",omitempty"`
	Deps            string   `json:",omitempty"`
	MSVCDepsPrefix  string   `json:",omitempty"`
	Restat          string   `json:",omitempty"`
	PhonyOutput     string   `json:",omitempty"`
	SymlinkOutputs  []string `json:",omitempty"`
	SandboxDisabled string   `json:",omitempty"`
}

// IsPhony reports whether r is the built-in "phony" rule.
func (r *Rule) IsPhony() bool {
	return r.Name == "phony"
}

// IsPhonyOutput reports whether r declares phony_output.
func (r *Rule) IsPhonyOutput() bool {
	return r.PhonyOutput != ""
}

// Build is a parsed "build" statement.
type Build struct {
	Outputs         []string    `json:",omitempty"`
	ImplicitOutputs []string    `json:",omitempty"`
	RuleName        string      `json:",omitempty"`
	Inputs          []string    `json:",omitempty"`
	ImplicitInputs  []string    `json:",omitempty"`
	OrderOnlyInputs []string    `json:",omitempty"`
	Validations     []string    `json:",omitempty"`
	Deps            []string    `json:",omitempty"`
	KeyValues       []*KeyValue `json:",omitempty"`

	Scope     *BuildScope `json:"-"`
	NinjaFile string      `json:"-"`
	Rule      *Rule       `json:"-"`
}

// OutputsType identifies the slice indices returned by AllOutputs.
type OutputsType int

// OutputsType values.
const (
	OutputsTypeRegular OutputsType = iota
	OutputsTypeImplicit
)

// AllOutputs returns the regular and implicit output lists in OutputsType order.
func (b *Build) AllOutputs() [][]string {
	return [][]string{b.Outputs, b.ImplicitOutputs}
}

// InputsType identifies the slice indices returned by AllInputs.
type InputsType int

// InputsType values.
const (
	InputsTypeRegular InputsType = iota
	InputsTypeImplicit
	InputsTypeOrderOnly
)

// AllInputs returns the regular, implicit, and order-only input lists in
// InputsType order.
func (b *Build) AllInputs() [][]string {
	return [][]string{b.Inputs, b.ImplicitInputs, b.OrderOnlyInputs}
}

// InputsWithValidations returns AllInputs followed by the validations list.
func (b *Build) InputsWithValidations() [][]string {
	return [][]string{b.Inputs, b.ImplicitInputs, b.OrderOnlyInputs, b.Validations}
}
