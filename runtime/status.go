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

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Status describes a snapshot of build progress used to render the NINJA_STATUS
// indicator. See https://ninja-build.org/manual.html#_environment_variables for
// the format directives accepted by Format.
//
//	%s The number of started edges.
//	%t The total number of edges that must be run to complete the build.
//	%p The percentage of finished edges.
//	%r The number of currently running edges.
//	%u The number of remaining edges to start.
//	%f The number of finished edges.
//	%o Overall rate of finished edges per second.
//	%c Current rate of finished edges per second (average over builds specified by -j or its default).
//	%e Elapsed time in seconds. (Available since Ninja 1.2.)
//	%E Remaining time (ETA) in seconds. (Available since Ninja 1.12.)
//	%w Elapsed time in [h:]mm:ss format. (Available since Ninja 1.12.)
//	%W Remaining time (ETA) in [h:]mm:ss format. (Available since Ninja 1.12.)
//	%P The percentage (in ppp%% format) of time elapsed out of predicted total runtime. (Available since Ninja 1.12.)
//	%% A plain %% character.
type Status struct {
	Started     int
	Total       int
	Running     int
	Finished    int
	Elapsed     time.Duration
	ETA         time.Duration
	OverallRate float64
	CurrentRate float64
}

// formatTime renders d as [h:]mm:ss.
func formatTime(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// Format expands the NINJA_STATUS-style directives in format using the values
// from s.
func (s *Status) Format(format string) string {
	var sb strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 's':
				fmt.Fprintf(&sb, "%d", s.Started)
			case 't':
				fmt.Fprintf(&sb, "%d", s.Total)
			case 'p':
				percent := 0.0
				if s.Total != 0 {
					percent = math.Round(float64(s.Finished) / float64(s.Total) * 100)
				}
				fmt.Fprintf(&sb, "%3.0f%%", percent)
			case 'r':
				fmt.Fprintf(&sb, "%d", s.Running)
			case 'u':
				remaining := s.Total - s.Started
				fmt.Fprintf(&sb, "%d", remaining)
			case 'f':
				fmt.Fprintf(&sb, "%d", s.Finished)
			case 'o':
				fmt.Fprintf(&sb, "%.1f", s.OverallRate)
			case 'c':
				fmt.Fprintf(&sb, "%.1f", s.CurrentRate)
			case 'e':
				fmt.Fprintf(&sb, "%.2f", s.Elapsed.Seconds())
			case 'E':
				fmt.Fprintf(&sb, "%.2f", s.ETA.Seconds())
			case 'w':
				sb.WriteString(formatTime(s.Elapsed))
			case 'W':
				sb.WriteString(formatTime(s.ETA))
			case 'P':
				percent := 0.0

				if s.Elapsed > 0 && s.ETA > 0 {
					predictedTotal := s.Elapsed + s.ETA
					percent = float64(s.Elapsed) / float64(predictedTotal) * 100
				}
				fmt.Fprintf(&sb, "%3.0f%%", percent)
			case '%':
				sb.WriteString("%")
			default:
				sb.WriteRune(rune(format[i]))
				sb.WriteRune(rune(format[i+1]))
			}
			i++
		} else {
			sb.WriteRune(rune(format[i]))
		}
	}
	return sb.String()
}
