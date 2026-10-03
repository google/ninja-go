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

package runtime

// WarnLevel describes how a particular warning should be reported.
type WarnLevel int

// WarnLevel values, in increasing order of severity.
const (
	WarnNone = iota
	WarnWarn
	WarnError
)

// CliFlags collects the values of all command-line flags accepted by the
// ninjago binary.
type CliFlags struct {
	Trace            bool
	Verbosity        int
	Quiet            bool
	ConcurrencyLevel int
	DryRun           bool
	FrontendCmd      string
	FrontendFile     string
	FailLimit        int

	PSICPU float64
	PSIMem float64
	PSIIO  float64

	// stats        print operation counts/timing info
	DebugStats bool
	// explain      explain what caused a command to execute
	DebugExplain bool
	// keepdepfile  don't delete depfiles after they're read by ninja
	DebugKeepDepFile bool
	// keeprsp      don't delete @response files on success
	DebugKeepRSP bool
	// nothreads    don't use threads to parallelize ninja
	DebugNoThreads bool

	// dupbuild={err,warn}  multiple build lines for one target
	WarnDupBuild WarnLevel
	// phonycycle={err,warn}  phony build statement references itself
	WarnPhonyCycle WarnLevel
	// depfilemulti={err,warn}  depfile has multiple output paths on separate lines
	WarnDepFileMulti WarnLevel
	// missingdepfile={err,warn}  how to treat missing depfiles
	WarnMissingDepFile WarnLevel

	// requires -o usesphonyoutputs=yes
	//    outputdir={err,warn}  how to treat outputs that are directories
	WarnOutputDir WarnLevel
	//    missingoutfile={err,warn}  how to treat missing output files
	WarnMissingOutFile WarnLevel
	//    oldoutput={err,warn}  how to treat output files older than their inputs
	WarnOldOutput WarnLevel

	// requires -o usessymlinkoutputs=yes
	//    undeclaredsymlinkoutputs={err,warn}
	WarnUndeclaredSymlinkOutputs WarnLevel

	// usesphonyoutputs={yes,no}: whether the generator uses 'phony_outputs' so
	// that these warnings work: outputdir, missingoutfile, oldoutput.
	OptUsesPhonyOutputs bool
	// usessymlinkoutputs={yes,no}: whether the generator uses 'symlink_outputs'
	// so that these warnings work: undeclaredsymlinkoutputs.
	OptUsesSymlinkOutputs bool
	// preremoveoutputs={yes,no}: whether to remove outputs before running a rule.
	OptPreremoveOutputs bool
	// usesninjalogasweightlist={yes,no}: whether to use the ninja log as the
	// source of the weight list.
	OptUsesNinjaLogAsWeightlist bool
	// usesweightlist={<file path>,no}: whether to prioritize some rules based
	// on a weight list read from the given file.
	OptUsesWeightList string
	// acceptoutputexists={yes,no}: if true, accept an existing output as valid
	// even if it is dirty.
	OptAcceptOutputExists bool
	// showcachedasdone={yes,no}: if true, show cached steps as done
	// (ninjago-specific).
	OptShowCachedAsDone bool
	// noremote={yes,no}: if true, disable remote execution (ninjago-specific).
	OptNoRemote bool
	// nolocalfallback={yes,no}: if true, do not fall back to local execution
	// when a remote execution fails (ninjago-specific).
	OptNoLocalFallback bool
}
