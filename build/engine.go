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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/ninja-go/graph"
	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
)

// infiniteDegree is added to a step's in-degree while its dependencies are
// being enumerated, to prevent the step from becoming ready prematurely.
const infiniteDegree = 1e9

// Engine drives the build: it owns the build state machine, dispatches work to
// schedulers, and invokes the backend for clean checks, cache lookups, and
// command execution.
type Engine struct {
	cpus        int
	lastStepID  int
	backend     BuildBackend
	params      *runtime.RunParameters
	log         runtime.Logger
	graph       *graph.Graph
	ctx         context.Context
	cancel      context.CancelFunc
	remoteSched *Scheduler
	localScheds map[string]*Scheduler
	startTime   time.Time
	noRemote    bool

	// lock must be held to perform any state transition.
	// Public methods assume the lock is not held; private methods assume the
	// lock is held unless documented otherwise.
	lock            sync.Mutex
	cleanCheckQueue chan *Step
	cacheCheckQueue chan *Step
	nodesCount      int64
	stateCounts     []int64
	pending         sync.WaitGroup
}

// NewEngine constructs an Engine for graph using the backend and parameters
// stored on ctx.
func NewEngine(ctx context.Context, graph *graph.Graph) *Engine {
	backend := runtime.Get[BuildBackend](ctx, runtime.BackendKey)
	params := runtime.Get[*runtime.RunParameters](ctx, runtime.RunParamsKey)

	return &Engine{
		graph:       graph,
		params:      params,
		cpus:        params.CliFlags.ConcurrencyLevel,
		backend:     backend,
		log:         params.Log,
		localScheds: make(map[string]*Scheduler),

		startTime:   time.Now(),
		stateCounts: make([]int64, MaxState),
	}
}

// StateCounts returns a snapshot of the number of steps in each BuildState.
func (e *Engine) StateCounts() map[BuildState]int64 {
	e.lock.Lock()
	defer e.lock.Unlock()

	counts := make(map[BuildState]int64, len(e.stateCounts))
	for i, c := range e.stateCounts {
		counts[BuildState(i)] = c
	}
	return counts
}

// validateTransition reports whether step is allowed to move into newState.
func (e *Engine) validateTransition(step *Step, newState BuildState) error {
	oldState := step.state

	switch oldState {
	case StateInit:
		if newState != StateBlocked {
			return fmt.Errorf("invalid transition")
		}

	case StateBlocked:
		switch newState {
		case StateCleanCheck:
			if step.IsPhony() {
				return fmt.Errorf("cannot transit for phony targets")
			}
		case StateDone:
			if !step.IsPhony() {
				return fmt.Errorf("cannot transit for non-phony targets")
			}
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateCleanCheck:
		switch newState {
		case StateCacheCheck:
		case StateClean:
		case StateFailed:
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateClean:
		return fmt.Errorf("final state")

	case StateCacheCheck:
		switch newState {
		case StateRemoteQueued:
			if !step.remotable {
				return fmt.Errorf("not-remotable target")
			}
		case StateLocalQueued:
			if step.remotable {
				return fmt.Errorf("remotable target")
			}
		case StateCached:
		case StateFailed:
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateCached:
		return fmt.Errorf("final state")

	case StateRemoteQueued:
		switch newState {
		case StateRemoteRunning:
		case StateLocalQueued:
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateRemoteRunning:
		switch newState {
		case StateLocalQueued:
		case StateFailed:
		case StateDone:
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateLocalQueued:
		switch newState {
		case StateLocalRunning:
		case StateFailed:
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateLocalRunning:
		switch newState {
		case StateFailed:
		case StateDone:
		default:
			return fmt.Errorf("invalid transition")
		}

	case StateDone:
		return fmt.Errorf("final state")

	case StateFailed:
		return fmt.Errorf("final state")

	}

	return nil
}

// transit moves step into newState, validating the transition first and then
// running the post-transition bookkeeping.
func (e *Engine) transit(step *Step, newState BuildState) {
	oldState := step.state

	if oldState == newState {
		e.log.Warnf("%s already in %q state", step, newState)
		return
	}

	if err := e.validateTransition(step, newState); err != nil {
		err := fmt.Sprintf("failed validation for transit from %q to %q for %s: %s", oldState, newState, step, err)
		panic(err)
	}

	step.state = newState

	e.postTransit(step, oldState, newState)
}

// postTransit performs bookkeeping after step has moved from oldState into
// newState: it updates counters, adjusts the pending wait-group, enqueues the
// step in any matching queue, and finalizes terminal states.
func (e *Engine) postTransit(step *Step, oldState, newState BuildState) {
	// Ninja does not count phony steps in its stats.
	if !step.IsPhony() {
		e.stateCounts[newState]++
		e.stateCounts[oldState]--
	}

	// Cancel the build when the failure limit is reached.
	if newState == StateFailed {
		limit := e.params.CliFlags.FailLimit
		if limit > 0 && e.stateCounts[StateFailed] >= int64(limit) {
			e.cancel()
		}
	}

	// Update the pending wait-group.
	pendingDiff := 0
	if oldState == StateBlocked {
		pendingDiff++
	}
	if newState == StateBlocked || newState.IsFinal() {
		pendingDiff--
	}
	if pendingDiff != 0 {
		e.pending.Add(pendingDiff)
	}

	// Add the step to the appropriate queue if needed.
	switch newState {
	case StateCleanCheck:
		e.cleanCheckQueue <- step
	case StateCacheCheck:
		e.cacheCheckQueue <- step
	case StateRemoteQueued:
		e.remoteSched.readies <- step
	case StateLocalQueued:
		step.Pool = e.localScheds[step.poolName]
		e.localScheds[step.poolName].readies <- step
	}

	// If it is a final state, clear the command fields to reduce memory.
	if newState.IsFinal() {
		step.ClearCommand()
		e.backend.FinalizeStep(step)
	}
}

// addStep introduces step into the build state machine and recurses into its
// dependencies.
func (e *Engine) addStep(step *Step) {
	if step.state != StateInit { // Already added.
		return
	}

	if !step.IsPhony() {
		e.nodesCount++
	}
	e.pending.Add(2) // One to keep pending for the duration of addStep, one for the step itself.
	defer e.pending.Done()

	step.inDegree.Add(infiniteDegree)
	e.transit(step, StateBlocked)
	for n, cc := range step.Node.Inputs {
		t := n.Data.(*Step)

		step.inDegree.Add(int32(cc))
		if t.state == StateInit {
			e.addStep(t)
		}
	}
	step.inDegree.Add(-infiniteDegree)
	if step.inDegree.Load() == 0 {
		e.handleCleanReady(step)
	}
}

// aboutToExec is called before a step is executed. It acquires the engine
// lock for the duration of the call.
func (e *Engine) aboutToExec(step *Step, newState BuildState) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.logCommandStart(step)

	e.transit(step, newState)

	return nil
}

// logCommandStart assigns a step ID if needed and notifies the logger that
// the step is starting.
func (e *Engine) logCommandStart(step *Step) {
	if step.ID == 0 {
		e.lastStepID++
		step.ID = e.lastStepID
	}

	desc := step.Node.Description()
	cached := e.stateCounts[StateCached]
	all := e.nodesCount - e.stateCounts[StateClean] - cached

	if e.params.CliFlags.OptShowCachedAsDone {
		all += cached
	}

	e.log.PlanHasTotalSteps(int(all))
	e.log.BuildStepStarted(step.ID, step.Command(), desc, step.Node.Targets)
}

// handleExecRemote runs the step's command on the remote backend. It is
// called by the remote scheduler.
func (e *Engine) handleExecRemote(step *Step) {
	exec := func() ExecResult {
		if err := e.aboutToExec(step, StateRemoteRunning); err != nil {
			return ExecResult{Err: err}
		}

		return e.backend.ExecRemote(e.ctx, step)
	}

	result := exec()
	if result.Err != nil {
		result.ExitCode = 1
		result.Stdout = result.Err.Error()
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	if result.ExitCode == 0 {
		e.log.BuildStepFinished(step.ID, 0, result.Stdout+result.Stderr)
		e.handleSuccess(step, StateDone)
	} else if e.params.CliFlags.OptNoLocalFallback {
		e.log.BuildStepFinished(step.ID, result.ExitCode, result.Stdout+result.Stderr)
		e.transit(step, StateFailed)
	} else {
		e.log.BuildStepCanceled(step.ID)
		e.transit(step, StateLocalQueued)
	}
}

// handleExecLocal runs the step's command on the local backend. It is called
// by a local scheduler.
func (e *Engine) handleExecLocal(step *Step) {
	if e.ctx.Err() != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
		e.transit(step, StateFailed)
		return
	}

	exec := func() ExecResult {
		if err := e.aboutToExec(step, StateLocalRunning); err != nil {
			return ExecResult{Err: err}
		}

		return e.backend.ExecLocal(e.ctx, step)
	}

	result := exec()
	if result.Err != nil {
		result.ExitCode = 1
		result.Stdout = result.Err.Error()
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	e.log.BuildStepFinished(step.ID, result.ExitCode, result.Stdout+result.Stderr)
	if result.ExitCode == 0 {
		e.handleSuccess(step, StateDone)
	} else {
		e.transit(step, StateFailed)
	}
}

// handleSuccess processes a step's transition into a successful state,
// decrementing dependent in-degrees and scheduling any newly-ready steps.
func (e *Engine) handleSuccess(step *Step, newState BuildState) {
	for next, cc := range step.Node.Outputs {
		t := next.Data.(*Step)
		newDeg := t.inDegree.Add(int32(-cc))
		if newDeg == 0 && t.state == StateBlocked {
			// Even though this step has been built, its output may still be
			// considered clean if restat is specified or in non-time-based
			// backends.
			e.handleCleanReady(t)
		}
	}

	if newState == StateDone || newState == StateCached {
		for t := range step.Node.Validations {
			e.addStep(t.Data.(*Step))
		}
	}

	e.transit(step, newState)
}

// handleCleanReady transitions step to clean-checking, or completes it
// immediately if it is phony.
func (e *Engine) handleCleanReady(step *Step) {
	if step.IsPhony() {
		e.handleSuccess(step, StateDone)
	} else {
		e.transit(step, StateCleanCheck)
	}
}

// Pools returns all build pools, including the implicit "default" pool sized
// according to the configured concurrency level.
func (e *Engine) Pools() map[string]*ninja.Pool {
	parsedPools := e.graph.Pools
	pools := make(map[string]*ninja.Pool, len(parsedPools)+1)
	for n, p := range parsedPools {
		pools[n] = p
	}
	pools["default"] = &ninja.Pool{
		Name:  "default",
		Depth: e.cpus,
	}
	pools["console"] = &ninja.Pool{
		Name:  "console",
		Depth: 1,
	}
	return pools
}

// createSchedulers starts a local scheduler for each known pool.
func (e *Engine) createSchedulers() error {
	monitor := newSchedMonitor(e.params.CliFlags.PSICPU, e.params.CliFlags.PSIIO, e.params.CliFlags.PSIMem)

	pools := e.Pools()
	poolsCount := make(map[string]*atomic.Int32, len(pools))
	for p := range pools {
		poolsCount[p] = new(atomic.Int32)
	}

	var err error
	e.graph.Nodes.Range(func(n *graph.Node) bool {
		cc := poolsCount[n.Pool()]
		if cc == nil {
			err = fmt.Errorf("pool %q not found for target %s", n.Pool(), n)
			return false
		}
		cc.Add(1)
		return true
	})
	if err != nil {
		return err
	}

	for n, p := range pools {
		s := NewScheduler(p, int(poolsCount[n].Load()), e.handleExecLocal, monitor)
		e.localScheds[p.Name] = s
		s.Start()
	}

	return nil
}

// startCleanInspectors spawns the goroutines that consume cleanCheckQueue and
// classify each step as clean, dirty, or failed.
func (e *Engine) startCleanInspectors() {
	for i := 0; i < e.params.CleanInspectorsCount; i++ {
		go func() {
			for s := range e.cleanCheckQueue {
				s.LoadCommand()

				clean, err := e.backend.IsClean(s)
				e.lock.Lock()
				if err != nil {
					e.log.Errorf("failed to check cleanness of %s: %s", s, err)
					e.transit(s, StateFailed)
				} else if clean {
					e.handleSuccess(s, StateClean)
				} else {
					e.transit(s, StateCacheCheck)
				}
				e.lock.Unlock()
			}
		}()
	}
}

// applyIfCached prepares the directories and response file for step and asks
// the backend to apply a cached result if one is available.
func (e *Engine) applyIfCached(step *Step) (bool, error) {
	if err := e.backend.CreateDirs(step); err != nil {
		return false, err
	}

	if err := e.backend.CreateRSP(step); err != nil {
		return false, err
	}

	return e.backend.ApplyIfCached(step)
}

// startCacheCheckers spawns the goroutines that consume cacheCheckQueue and
// either apply a cached result, queue the step for execution, or fail it.
func (e *Engine) startCacheCheckers() {
	for i := 0; i < e.params.CacheCheckersCount; i++ {
		go func() {
			for s := range e.cacheCheckQueue {
				cached, err := e.applyIfCached(s)
				e.lock.Lock()
				if err != nil {
					e.log.Errorf("failed to check the cache for %s: %s", s, err)
					// TODO: should we skip to Ready if the cache check fails?
					e.transit(s, StateFailed)
				} else if cached {
					if e.params.CliFlags.OptShowCachedAsDone {
						e.logCommandStart(s)
					}
					e.handleSuccess(s, StateCached)
				} else {
					s.remotable = !e.noRemote && !s.IsConsole() && e.backend.IsRemotable(s)
					if s.remotable {
						e.transit(s, StateRemoteQueued)
					} else {
						e.transit(s, StateLocalQueued)
					}
				}
				e.lock.Unlock()
				if cached && e.params.CliFlags.OptShowCachedAsDone {
					e.log.BuildStepFinished(s.ID, 0, "")
				}
			}
		}()
	}
}

// Build runs the configured backend and engine to completion, returning a
// non-nil error if any step failed.
func (e *Engine) Build(ctx context.Context) error {
	tt := time.Now()
	e.ctx, e.cancel = context.WithCancel(ctx)

	e.log.BuildStarted()
	defer e.log.BuildFinished()

	if err := e.backend.BuildInit(e); err != nil {
		return err
	}
	e.log.Debugf("%% backend build init took %s", time.Since(tt))

	n := e.graph.Targets.Len()
	e.cleanCheckQueue = make(chan *Step, n)
	e.cacheCheckQueue = make(chan *Step, n)

	e.startCleanInspectors()
	e.startCacheCheckers()
	if !e.noRemote {
		remotePool := &ninja.Pool{
			Name:  "remote",
			Depth: e.params.RemoteExecutorsCount,
		}

		e.remoteSched = NewScheduler(remotePool, n, e.handleExecRemote, nil)
		e.remoteSched.Start()
	}
	if err := e.createSchedulers(); err != nil {
		return err
	}

	tt = time.Now()
	stateCounts := make([]atomic.Int64, int(MaxState))
	e.graph.Nodes.GoRange(func(n *graph.Node) bool {
		n.Data = NewStep(n)
		stateCounts[StateInit].Add(1)
		return true
	})
	for i := 0; i < len(e.stateCounts); i++ {
		e.stateCounts[i] = stateCounts[i].Load()
	}
	e.log.Debugf("%% allocating steps took %s", time.Since(tt))

	tt = time.Now()
	e.lock.Lock()
	e.graph.Goals.Range(func(n *graph.Node) bool {
		e.addStep(n.Data.(*Step))
		return true
	})
	e.lock.Unlock()
	e.log.Debugf("%% adding goal steps took %s", time.Since(tt))

	tt = time.Now()
	e.pending.Wait()
	e.log.Debugf("%% waiting for builds took %s", time.Since(tt))

	close(e.cleanCheckQueue)
	close(e.cacheCheckQueue)
	if !e.noRemote {
		e.remoteSched.Stop()
	}
	for _, sched := range e.localScheds {
		sched.Stop()
	}

	for i := 0; i < len(e.stateCounts); i++ {
		e.log.Debugf("@ %s: %d", BuildState(i), e.stateCounts[i])
	}

	if e.stateCounts[StateFailed] == 0 && e.stateCounts[StateDone]+e.stateCounts[StateCached] == 0 {
		e.log.Printf("ninja: no work to do.")
	}

	var err error
	if e.stateCounts[StateFailed] > 0 {
		err = fmt.Errorf("build stopped: cannot make progress due to previous errors")
	}

	return e.backend.FinalizeBuild(err)
}
