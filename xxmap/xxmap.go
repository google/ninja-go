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

// Package xxmap provides a concurrent string-keyed map sharded by xxhash,
// trading a fixed amount of memory for reduced contention compared to a
// single sync.Map under highly parallel access.
package xxmap

import (
	"sync"
	"unsafe"

	"github.com/cespare/xxhash/v2"
)

// shards is the number of sync.Map instances used to reduce contention.
const shards = 128

// StringMap is a concurrent map from string keys to values of type T,
// internally sharded across shards sync.Map instances.
type StringMap[T any] []sync.Map

// NewStringMap returns an empty StringMap.
func NewStringMap[T any]() StringMap[T] {
	return make(StringMap[T], shards)
}

// PutIfMissing stores value under key only if no value is present, and reports
// whether the value was inserted.
func (t StringMap[T]) PutIfMissing(key string, value T) bool {
	hash := xxhash.Sum64String(key)
	_, exists := t[int(hash%shards)].LoadOrStore(key, value)
	return !exists
}

// Put stores value under key, replacing any previous value.
func (t StringMap[T]) Put(key string, value T) {
	hash := xxhash.Sum64String(key)
	t[int(hash%shards)].Store(key, value)
}

// Get returns the value stored under key, or the zero value of T if absent.
func (t StringMap[T]) Get(key string) T {
	var zero T
	hash := xxhash.Sum64String(key)
	b, exists := t[int(hash%shards)].Load(key)
	if !exists {
		return zero
	}
	return b.(T)
}

// Remove deletes any value stored under key.
func (t StringMap[T]) Remove(key string) {
	hash := xxhash.Sum64String(key)
	t[int(hash%shards)].Delete(key)
}

// Range calls f sequentially for every (key, value) pair. Iteration stops
// when f returns false.
func (t StringMap[T]) Range(f func(key string, value T) bool) {
	done := false
	f2 := func(key, value any) bool {
		if f(key.(string), value.(T)) {
			return true
		}
		done = true
		return false
	}

	for i := 0; i < shards && !done; i++ {
		t[i].Range(f2)
	}
}

// GoRange is like Range but iterates the shards concurrently.
func (t StringMap[T]) GoRange(f func(key string, value T) bool) {
	done := false
	f2 := func(key, value any) bool {
		if done {
			return false
		}
		if f(key.(string), value.(T)) {
			return true
		}
		done = true
		return false
	}

	var wg sync.WaitGroup
	wg.Add(shards)
	for i := 0; i < shards; i++ {
		go func() {
			defer wg.Done()
			t[i].Range(f2)
		}()
	}
	wg.Wait()
}

// Len returns the number of entries in t.
func (t StringMap[T]) Len() int {
	c := 0
	t.Range(func(key string, value T) bool {
		c++
		return true
	})
	return c
}

// StringsSet is a concurrent set of strings.
type StringsSet []sync.Map

// NewStringsSet returns an empty StringsSet.
func NewStringsSet() StringsSet {
	return make(StringsSet, shards)
}

// Put inserts key into the set and reports whether it was newly added.
func (t StringsSet) Put(key string) bool {
	hash := xxhash.Sum64String(key)
	_, exists := t[int(hash%shards)].LoadOrStore(key, struct{}{})
	return !exists
}

// Has reports whether key is in the set.
func (t StringsSet) Has(key string) bool {
	hash := xxhash.Sum64String(key)
	_, exists := t[int(hash%shards)].Load(key)
	return exists
}

// Range calls f sequentially for every key in the set. Iteration stops when
// f returns false.
func (t StringsSet) Range(f func(key string) bool) {
	done := false
	f2 := func(key, value any) bool {
		if f(key.(string)) {
			return true
		}
		done = true
		return false
	}

	for i := 0; i < shards && !done; i++ {
		t[i].Range(f2)
	}
}

// GoRange is like Range but iterates the shards concurrently.
func (t StringsSet) GoRange(f func(key string) bool) {
	done := false
	f2 := func(key, value any) bool {
		if done {
			return false
		}
		if f(key.(string)) {
			return true
		}
		done = true
		return false
	}

	var wg sync.WaitGroup
	wg.Add(shards)
	for i := 0; i < shards; i++ {
		go func() {
			defer wg.Done()
			t[i].Range(f2)
		}()
	}
	wg.Wait()
}

// Len returns the number of strings in the set.
func (t StringsSet) Len() int {
	c := 0
	t.Range(func(key string) bool {
		c++
		return true
	})
	return c
}

// ObjectsSet is a concurrent set of *T values, sharded by pointer address.
// The element type is given as T (rather than *T) to work around Go's
// limitations on using pointer types in generic constraints.
type ObjectsSet[T any] []sync.Map

// NewObjectsSet returns an empty ObjectsSet.
func NewObjectsSet[T any]() ObjectsSet[T] {
	return make(ObjectsSet[T], shards)
}

// Put inserts obj into the set and reports whether it was newly added.
func (t ObjectsSet[T]) Put(obj *T) bool {
	ptr := uintptr(unsafe.Pointer(obj))
	_, exists := t[(ptr/16)%shards].LoadOrStore(obj, struct{}{})
	return !exists
}

// Has reports whether obj is in the set.
func (t ObjectsSet[T]) Has(obj *T) bool {
	ptr := uintptr(unsafe.Pointer(obj))
	_, exists := t[(ptr/16)%shards].Load(obj)
	return exists
}

// Range calls f sequentially for every object in the set. Iteration stops
// when f returns false.
func (t ObjectsSet[T]) Range(f func(obj *T) bool) {
	done := false
	f2 := func(key, value any) bool {
		if f(key.(*T)) {
			return true
		}
		done = true
		return false
	}

	for i := 0; i < shards && !done; i++ {
		t[i].Range(f2)
	}
}

// GoRange is like Range but iterates the shards concurrently.
func (t ObjectsSet[T]) GoRange(f func(obj *T) bool) {
	done := false
	f2 := func(key, value any) bool {
		if done {
			return false
		}
		if f(key.(*T)) {
			return true
		}
		done = true
		return false
	}

	var wg sync.WaitGroup
	wg.Add(shards)
	for i := 0; i < shards; i++ {
		go func() {
			defer wg.Done()
			t[i].Range(f2)
		}()
	}
	wg.Wait()
}

// Len returns the number of objects in the set.
func (t ObjectsSet[T]) Len() int {
	c := 0
	t.Range(func(obj *T) bool {
		c++
		return true
	})
	return c
}

// ObjectsMap is a concurrent map keyed by *K values, sharded by pointer
// address.
type ObjectsMap[K, V any] []sync.Map

// NewObjectsMap returns an empty ObjectsMap.
func NewObjectsMap[K, V any]() ObjectsMap[K, V] {
	return make(ObjectsMap[K, V], shards)
}

// PutIfMissing stores value under key only if no value is present, and reports
// whether the value was inserted.
func (t ObjectsMap[K, V]) PutIfMissing(key *K, value V) bool {
	ptr := uintptr(unsafe.Pointer(key))
	_, exists := t[(ptr/16)%shards].LoadOrStore(key, value)
	return !exists
}

// Put stores value under key, replacing any previous value.
func (t ObjectsMap[K, V]) Put(key *K, value V) {
	ptr := uintptr(unsafe.Pointer(key))
	t[(ptr/16)%shards].Store(key, value)
}

// Get returns the value stored under key and whether such an entry exists.
func (t ObjectsMap[K, V]) Get(key *K) (V, bool) {
	var zero V
	ptr := uintptr(unsafe.Pointer(key))
	value, exists := t[(ptr/16)%shards].Load(key)
	if !exists {
		return zero, false
	}
	return value.(V), true
}

// Remove deletes any value stored under key.
func (t ObjectsMap[K, V]) Remove(key *K) {
	ptr := uintptr(unsafe.Pointer(key))
	t[(ptr/16)%shards].Delete(key)
}

// Range calls f sequentially for every (key, value) pair. Iteration stops
// when f returns false.
func (t ObjectsMap[K, V]) Range(f func(key *K, value V) bool) {
	done := false
	f2 := func(key, value any) bool {
		if f(key.(*K), value.(V)) {
			return true
		}
		done = true
		return false
	}

	for i := 0; i < shards && !done; i++ {
		t[i].Range(f2)
	}
}

// GoRange is like Range but iterates the shards concurrently.
func (t ObjectsMap[K, V]) GoRange(f func(key *K, value V) bool) {
	done := false
	f2 := func(key, value any) bool {
		if done {
			return false
		}
		if f(key.(*K), value.(V)) {
			return true
		}
		done = true
		return false
	}

	var wg sync.WaitGroup
	wg.Add(shards)
	for i := 0; i < shards; i++ {
		go func() {
			defer wg.Done()
			t[i].Range(f2)
		}()
	}
	wg.Wait()
}

// Len returns the number of entries in t.
func (t ObjectsMap[K, V]) Len() int {
	c := 0
	t.Range(func(key *K, value V) bool {
		c++
		return true
	})
	return c
}
