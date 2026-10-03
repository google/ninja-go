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

// Split returns the i-th of j approximately equal contiguous chunks of list.
// It is used to partition work across worker goroutines.
func Split[T any](list []T, j, i int) []T {
	n := len(list)
	s := (n + j - 1) / j
	if s < 2 {
		if i == 0 {
			return list
		}
		return nil
	}
	a1 := s * i
	a2 := s * (i + 1)
	if a1 > n {
		a1 = n
	}
	if a2 > n {
		a2 = n
	}
	return list[a1:a2]
}
