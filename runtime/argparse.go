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

// Package runtime provides the CLI-facing runtime support shared across
// ninjago commands: flag and argument parsing, logging, console and status
// rendering, memory-mapped file I/O, and the RunParameters bag that ties them
// together.
package runtime

import (
	"fmt"
	"log"
	"strings"
)

// LoadDebugs parses the values supplied via -d and sets the matching debug
// flags on f.
func (f *CliFlags) LoadDebugs(d []string) error {
	for _, s := range d {
		switch s {
		case "stats":
			f.DebugStats = true
		case "explain":
			f.DebugExplain = true
		case "keepdepfile":
			f.DebugKeepDepFile = true
		case "keeprsp":
			f.DebugKeepRSP = true
		case "nothreads":
			f.DebugNoThreads = true
		default:
			return fmt.Errorf("unknown debug option: %s", s)
		}
	}
	return nil
}

// LoadWarns parses the values supplied via -w and sets the matching warning
// levels on f.
func (f *CliFlags) LoadWarns(w []string) error {
	for _, s := range w {
		switch s {
		case "dupbuild=warn":
			f.WarnDupBuild = WarnWarn
		case "dupbuild=err":
			f.WarnDupBuild = WarnError
		case "phonycycle=warn":
			f.WarnPhonyCycle = WarnWarn
		case "depfilemulti=err":
			f.WarnDepFileMulti = WarnError
		case "depfilemulti=warn":
			f.WarnDepFileMulti = WarnWarn
		case "missingdepfile=err":
			f.WarnMissingDepFile = WarnError
		case "missingdepfile=warn":
			f.WarnMissingDepFile = WarnWarn
		case "outputdir=err":
			f.WarnOutputDir = WarnError
		case "outputdir=warn":
			f.WarnOutputDir = WarnWarn
		case "missingoutfile=err":
			f.WarnMissingOutFile = WarnError
		case "missingoutfile=warn":
			f.WarnMissingOutFile = WarnWarn
		case "oldoutput=err":
			f.WarnOldOutput = WarnError
		case "oldoutput=warn":
			f.WarnOldOutput = WarnWarn
		case "undeclaredsymlinkoutputs=err":
			f.WarnUndeclaredSymlinkOutputs = WarnError
		case "undeclaredsymlinkoutputs=warn":
			f.WarnUndeclaredSymlinkOutputs = WarnWarn
		default:
			return fmt.Errorf("unknown warn option: %s", s)
		}
	}
	return nil
}

// LoadOptions parses the values supplied via -o and sets the matching option
// fields on f.
func (f *CliFlags) LoadOptions(o []string) error {

	for _, s := range o {
		split := strings.SplitN(s, "=", 2)
		var k, value string
		k = split[0]
		if len(split) > 1 {
			value = split[1]
		}
		switch k {
		case "usesphonyoutputs":
			f.OptUsesPhonyOutputs = (value == "yes")
		case "usessymlinkoutputs":
			f.OptUsesSymlinkOutputs = (value == "yes")
		case "preremoveoutputs":
			f.OptPreremoveOutputs = (value == "yes")
		case "acceptoutputexists":
			f.OptAcceptOutputExists = (value == "yes")
		case "showcachedasdone":
			f.OptShowCachedAsDone = (value == "yes")
		case "noremote":
			f.OptNoRemote = (value == "yes")
		case "nolocalfallback":
			f.OptNoLocalFallback = (value == "yes")
		case "usesninjalogasweightlist", "usesweightlist":
			log.Printf("Ninja: Option %q is not currently supported.", s)
		default:
			return fmt.Errorf("unknown option flag: %s", s)
		}
	}
	return nil
}
