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

// Logger receives build progress events and log messages.
type Logger interface {
	BuildStarted()
	BuildFinished()
	BuildStepStarted(id int, cmd, desc string, outputs []string)
	BuildStepFinished(id int, exitCode int, output string)
	BuildStepCanceled(id int)
	PlanHasTotalSteps(total int)
	Printf(format string, a ...any)
	Tracef(format string, a ...any)
	Debugf(format string, a ...any)
	Infof(format string, a ...any)
	Warnf(format string, a ...any)
	Errorf(format string, a ...any)
	Close() error
}
