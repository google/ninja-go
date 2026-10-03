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

package graph

// func (g *Graph) StringsToNodes(nodes []string) ([]*Node, error) {
// 	result := make([]*Node, len(nodes))
// 	for i, n := range nodes {
// 		result[i] = g.Targets[n]
// 		if result[i] == nil {
// 			return nil, fmt.Errorf("unknown target '%s'", n)
// 		}
// 	}
// 	return result, nil
// }

// func (g *Graph) StringsToNodesMap(nodes []string) (map[*Node]struct{}, error) {
// 	result := make(map[*Node]struct{}, len(nodes))
// 	for _, n := range nodes {
// 		m := g.Targets[n]
// 		if m == nil {
// 			return nil, fmt.Errorf("unknown target '%s'", n)
// 		}
// 		result[m] = struct{}{}
// 	}
// 	return result, nil
// }

// func NodesMapToStrings(nodes map[*Node]struct{}) []string {
// 	result := make([]string, 0, len(nodes))
// 	for n := range nodes {
// 		result = append(result, n.Targets...)
// 	}
// 	return result
// }

// func NodesToStrings(nodes []*Node) []string {
// 	result := make([]string, 0, len(nodes))
// 	for _, n := range nodes {
// 		result = append(result, n.Targets...)
// 	}
// 	return result
// }
