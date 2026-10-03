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

package build

import (
	"sync"

	"github.com/google/ninja-go/ninja"
)

// ExecutionCallback is invoked by the scheduler to run a step.
type ExecutionCallback func(s *Step)

// Scheduler dispatches ready steps to a fixed pool of worker goroutines.
type Scheduler struct {
	readies  chan *Step
	pool     *ninja.Pool
	queued   chan *Step
	callback ExecutionCallback
	monitor  *SchedMonitor
	wg       sync.WaitGroup
}

// NewScheduler returns a Scheduler bound to pool, sized for at most maxCount
// queued steps and using callback to execute them.
func NewScheduler(pool *ninja.Pool, maxCount int, callback ExecutionCallback, monitor *SchedMonitor) *Scheduler {
	return &Scheduler{
		readies:  make(chan *Step, maxCount),
		queued:   make(chan *Step),
		pool:     pool,
		callback: callback,
		monitor:  monitor,
	}
}

// Start launches the dispatch and worker goroutines.
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.runPusher()
	s.startBuilders()
}

// Stop closes the input channel and blocks until all worker goroutines exit.
func (s *Scheduler) Stop() {
	close(s.readies)
	s.wg.Wait()
}

// Pool returns the ninja pool that backs this scheduler.
func (s *Scheduler) Pool() *ninja.Pool {
	return s.pool
}

// runPusher forwards ready steps onto the queued channel until readies closes.
func (s *Scheduler) runPusher() {
	defer s.wg.Done()
	for step := range s.readies {
		s.queued <- step
	}
	close(s.queued)
}

// mayI reports whether the monitor (if any) currently allows new work.
func (s *Scheduler) mayI() bool {
	return s.monitor == nil || s.monitor.MayI()
}

// startBuilders launches pool.Depth worker goroutines that consume queued
// steps and run them through the configured callback.
func (s *Scheduler) startBuilders() {
	for i := 0; i < s.pool.Depth; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for step := range s.queued {
				if !s.mayI() {
					break
				}
				s.callback(step)
			}
		}()
	}
}
