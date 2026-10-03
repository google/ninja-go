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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatus1(t *testing.T) {
	s := Status{
		Started:     100,
		Total:       1234,
		Running:     10,
		Finished:    1200,
		Elapsed:     5 * time.Minute,
		ETA:         6*time.Minute + 13*time.Second + 20*time.Millisecond,
		OverallRate: 100,
		CurrentRate: 1020,
	}
	format := `started=%s total=%t percent=%p running=%r remaining=%u finished=%f overall-speed=%o current-speed=%c elapsed-in-sec=%e remaining-in-sec=%E elapsed=%w remaining=%W percent-time=%P`
	expect := `started=100 total=1234 percent= 97% running=10 remaining=1134 finished=1200 overall-speed=100.0 current-speed=1020.0 elapsed-in-sec=300.00 remaining-in-sec=373.02 elapsed=05:00 remaining=06:13 percent-time= 45%`
	require.Equal(t, expect, s.Format(format))
}
