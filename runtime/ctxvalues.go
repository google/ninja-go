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
	"context"
	"fmt"
)

// ContextKey is the type used for keys placed on the context by ninjago.
type ContextKey string

// Well-known keys used to attach ninjago values to a context.Context.
const (
	BackendKey   ContextKey = "backend"
	RunParamsKey ContextKey = "run-params"
)

// Get retrieves the value stored under key from ctx and asserts it as T.
// It panics if the value is missing or has a different type.
func Get[T any](ctx context.Context, key ContextKey) T {
	var t T
	v, ok := ctx.Value(key).(T)
	if !ok {
		if ctx.Value(key) != nil {
			panic(fmt.Sprintf("context value of %q of %T is not set", key, t))
		}
		panic(fmt.Sprintf("context value of %q is not a type of %T and is %T", key, t, v))
	}
	return v
}
