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

// Package classic implements a ninjago backend that mirrors the on-disk
// semantics of the upstream ninja binary, including its .ninja_log and
// .ninja_deps files and local-only command execution.
package classic

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/ninja-go/backends"
	"github.com/google/ninja-go/build"
	"github.com/google/ninja-go/graph"
	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/ninjafiles/ninjadeps"
	"github.com/google/ninja-go/ninjafiles/ninjalog"
	"github.com/google/ninja-go/parse"
	"github.com/google/ninja-go/runtime"
	"github.com/google/ninja-go/xxmap"
)

// FileMeta is the cached file-system metadata about a single file.
type FileMeta struct {
	Mode    os.FileMode
	ModTime time.Time
}

// IsMissing reports whether the file the FileMeta describes does not exist.
func (f *FileMeta) IsMissing() bool {
	return f.Mode == 0
}

// BuildInfo records the result of a previous build of a target as read from
// the ninja log.
type BuildInfo struct {
	CmdHash uint64
	Time    time.Time
}

// ClassicBackend implements backends.Backend with the same on-disk semantics
// as the upstream ninja binary.
type ClassicBackend struct {
	startTime   time.Time
	params      *runtime.RunParameters
	dryRun      bool
	ninjas      *ninja.ParsedFiles
	buildDir    string
	graph       *graph.Graph
	logFile     *ninjalog.NinjaLog
	depsFile    *ninjadeps.NinjaDeps
	pendingDeps sync.WaitGroup
	// files maps file paths to their metadata.
	files xxmap.StringMap[*FileMeta]
	// builds is a map from first taget of a step to the BuildInfo.
	builds xxmap.StringMap[*BuildInfo]
}

// NewClassicBackend returns a ClassicBackend with empty caches.
func NewClassicBackend() *ClassicBackend {
	return &ClassicBackend{
		files:  xxmap.NewStringMap[*FileMeta](),
		builds: xxmap.NewStringMap[*BuildInfo](),
	}
}

var _ backends.Backend = (*ClassicBackend)(nil)

// Init implements subcommands.GeneralBackend.
func (c *ClassicBackend) Init(params *runtime.RunParameters, cmd *string, args *[]string) error {
	c.startTime = time.Now()
	c.params = params
	c.dryRun = params.CliFlags.DryRun

	return nil
}

// GetDeps implements graph.GraphBackend.
func (c *ClassicBackend) GetDeps(n *graph.Node) ([]string, error) {
	depFile := n.DepFile()
	if depFile == "" {
		return nil, nil
	}

	stat, err := os.Stat(depFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		// Depfile absent — use cached deps from .ninja_deps if the
		// cache entry is from the most recent build of this target.
		build := c.builds.Get(n.Targets[0])
		if build != nil {
			if t, deps := c.depsFile.GetDeps(n.Targets[0]); !t.IsZero() && !t.Before(build.Time) {
				return deps, nil
			}
		}
		return nil, nil
	}

	build := c.builds.Get(n.Targets[0])
	if build != nil {
		if t, deps := c.depsFile.GetDeps(n.Targets[0]); !t.IsZero() && !stat.ModTime().After(build.Time.Add(time.Second)) {
			return deps, nil
		}
	}

	deps, err := parse.ReadDepFile(n.Targets, depFile)
	if err != nil {
		return nil, IgnoreNotExist(err)
	}
	return deps, nil
}

// Parsed implements parse.ParseBackend.
func (c *ClassicBackend) Parsed(ninjas *ninja.ParsedFiles) error {
	c.ninjas = ninjas
	return nil
}

// GraphLoading implements graph.GraphBackend.
func (c *ClassicBackend) GraphLoading() error {
	c.buildDir = c.ninjas.BuildDir()
	c.ninjas = nil // Drop the reference so the parsed files can be garbage-collected.

	c.logFile = ninjalog.New(path.Join(c.buildDir, ninjalog.DefaultLogFileName))
	if err := c.loadNinjaLog(); err != nil {
		return err
	}

	c.depsFile = ninjadeps.New(path.Join(c.buildDir, ninjadeps.DefaultLogFileName))
	if err := c.depsFile.Load(); err != nil {
		return err
	}
	return nil
}

// GraphLoaded implements graph.GraphBackend.
func (c *ClassicBackend) GraphLoaded(graph *graph.Graph) error {
	c.graph = graph
	return nil
}

// Files returns the cached file-system metadata gathered by collectFilesMeta.
func (c *ClassicBackend) Files() xxmap.StringMap[*FileMeta] {
	return c.files
}

// loadNinjaLog reads the ninja log and indexes the most recent entry per
// target into c.builds.
func (c *ClassicBackend) loadNinjaLog() error {
	entries, err := c.logFile.Parse()
	if err != nil {
		return err
	}

	var build = new(BuildInfo)
	for _, ent := range entries {
		hash, err := strconv.ParseUint(ent.Checksum, 16, 64)
		if err != nil {
			return fmt.Errorf("cannot parse hash of %s in ninjalog: %w", ent.Target, err)
		}
		if build.CmdHash != hash || !build.Time.Equal(ent.Timestamp) {
			build = &BuildInfo{
				CmdHash: hash,
				Time:    ent.Timestamp,
			}
		}
		c.builds.Put(ent.Target, build)
	}
	return nil
}

// BuildInit implements build.BuildBackend.
func (c *ClassicBackend) BuildInit(engine *build.Engine) error {
	if err := c.collectFilesMeta(); err != nil {
		return err
	}

	if !c.dryRun {
		if err := c.logFile.OpenForWrite(); err != nil {
			return err
		}

		if err := c.depsFile.OpenForWrite(); err != nil {
			return err
		}
	}

	return nil
}

// statFileMeta returns FileMeta for f, or a zero-valued meta if f is missing.
func statFileMeta(f string) (*FileMeta, error) {
	stat, err := os.Lstat(f)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &FileMeta{}, nil
		} else {
			return nil, err
		}
	}
	return &FileMeta{ModTime: stat.ModTime(), Mode: stat.Mode()}, nil
}

// collectFilesMeta concurrently stats every input, output, and response file
// referenced by the graph and stores the result in c.files.
func (c *ClassicBackend) collectFilesMeta() error {
	todo := make(chan string)

	var grp errgroup.Group
	for i := 0; i < c.params.CliFlags.ConcurrencyLevel; i++ {
		grp.Go(func() error {
			for fn := range todo {
				m, err := statFileMeta(fn)
				if err != nil {
					return err
				}
				c.files.Put(fn, m)
			}
			return nil
		})
	}

	c.graph.Files.GoRange(func(key string) bool {
		todo <- key
		return true
	})
	c.graph.Nodes.GoRange(func(n *graph.Node) bool {
		if n.Build.Rule.IsPhony() || n.Build.Rule.IsPhonyOutput() {
			return true
		}
		for _, t := range n.Targets {
			todo <- t
		}
		if rsp := n.RSPFile(); rsp != "" {
			todo <- rsp
		}
		return true
	})
	close(todo)

	if err := grp.Wait(); err != nil {
		return err
	}
	return nil
}

// IgnoreNotExist returns nil if err is fs.ErrNotExist, otherwise it returns
// err unchanged.
func IgnoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// IsClean implements build.BuildBackend.
func (c *ClassicBackend) IsClean(step *build.Step) (bool, error) {
	if step.IsPhonyOutput() { // phony_output is always dirty.
		return false, nil
	}

	build := c.builds.Get(step.Node.Build.Outputs[0])
	if build == nil {
		return false, nil
	}

	if !step.IsGenerator() && build.CmdHash != step.CmdHash() {
		return false, nil
	}

	for _, outs := range step.Node.Build.AllOutputs() {
		for _, out := range outs {
			m, err := c.getMeta(out)
			if err != nil {
				return false, err
			}
			if m.IsMissing() || m.ModTime.After(build.Time) {
				return false, nil
			}
		}
	}

	if step.Node.Build.Scope.ExpandField("deps") != "" {
		t, _ := c.depsFile.GetDeps(step.Node.Build.Outputs[0])
		if t.IsZero() {
			return false, nil
		}
	}

	depTime, err := c.inputModTime(step)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	rspFile := step.Node.RSPFile()
	if rspFile != "" {
		m, err := c.meta(rspFile)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return false, err
			}
		} else {
			depTime = maxTime(depTime, m.ModTime)
		}
	}

	if build.Time.Before(depTime) {
		return false, nil
	}

	return true, nil
}

// ApplyIfCached implements build.BuildBackend. The classic backend does not
// have a cache, so this always returns false.
func (c *ClassicBackend) ApplyIfCached(step *build.Step) (bool, error) {
	return false, nil
}

// IsRemotable implements build.BuildBackend. The classic backend always runs
// commands locally.
func (c *ClassicBackend) IsRemotable(step *build.Step) bool {
	return false
}

// ExecRemote implements build.BuildBackend. The classic backend does not
// support remote execution.
func (c *ClassicBackend) ExecRemote(ctx context.Context, step *build.Step) build.ExecResult {
	return build.ExecResult{Err: errors.New("remote execution is not supported in classic backend")}
}

// inputModTime returns the most recent modification time among the inputs of
// step, considering both produced builds and on-disk files.
func (c *ClassicBackend) inputModTime(step *build.Step) (time.Time, error) {
	result, err := c.maxBuildTime(step.Node.Inputs)
	if err != nil {
		return time.Time{}, err
	}

	t, err := c.maxModTime(step.Node.Build.Inputs)
	if err != nil {
		return time.Time{}, err
	}
	result = maxTime(result, t)

	t, err = c.maxModTime(step.Node.Build.ImplicitInputs)
	if err != nil {
		return time.Time{}, err
	}
	result = maxTime(result, t)

	t, err = c.maxModTime(step.Node.Deps)
	if err != nil {
		return time.Time{}, err
	}
	result = maxTime(result, t)

	return result, nil
}

// maxTime returns the later of a and b.
func maxTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return b
	}
	return a
}

// maxBuildTime returns the most recent build time recorded in the ninja log
// for any of the given input nodes, skipping phony and restat rules.
func (c *ClassicBackend) maxBuildTime(nodes map[*graph.Node]int) (time.Time, error) {
	var t time.Time
	for n := range nodes {
		if n.Build.Rule.IsPhony() {
			continue
		}
		if n.Build.Rule.Restat != "" {
			continue
		}
		build := c.builds.Get(n.Targets[0])
		if build == nil {
			return time.Time{}, fmt.Errorf("build info for input %s is missing", n.Targets[0])
		}
		t = maxTime(t, build.Time)
	}
	return t, nil
}

// maxModTime returns the most recent on-disk modification time among the
// given files, skipping directories and phony targets.
func (c *ClassicBackend) maxModTime(files []string) (time.Time, error) {
	var errs []error
	var t time.Time
	for _, f := range files {
		if n := c.graph.Targets.Get(f); n != nil {
			if n.Build.Rule.IsPhony() || n.Build.Rule.IsPhonyOutput() {
				continue
			}
		}
		m, err := c.meta(f)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if m.Mode.IsDir() {
			continue
		}
		t = maxTime(t, m.ModTime)
	}
	return t, errors.Join(errs...)
}

// getMeta returns the cached FileMeta for f, falling back to a fresh stat if
// the cache does not contain it.
func (c *ClassicBackend) getMeta(f string) (*FileMeta, error) {
	m := c.files.Get(f)
	if m != nil {
		return m, nil
	}
	c.params.Log.Warnf("metadata for %q is missing from the backend cache", f)
	return statFileMeta(f)
}

// meta is like getMeta but turns a missing file into fs.ErrNotExist.
func (c *ClassicBackend) meta(f string) (*FileMeta, error) {
	m, err := c.getMeta(f)
	if err != nil {
		return nil, fmt.Errorf("cannot stat %s: %w", f, err)
	}

	if m.IsMissing() {
		return nil, fmt.Errorf("cannot stat %s: %w", f, fs.ErrNotExist)
	}
	return m, nil
}

// ExecLocal implements build.BuildBackend.
func (c *ClassicBackend) ExecLocal(ctx context.Context, step *build.Step) build.ExecResult {
	return c.ExecuteCmd(ctx, step, execCmd)
}

// ExecuteCmd runs exec for step (unless --dry-run is set) and updates the
// cached file metadata, ninja log, and ninja deps file based on the result.
func (c *ClassicBackend) ExecuteCmd(ctx context.Context, step *build.Step, exec func(context.Context, *build.Step) build.ExecResult) build.ExecResult {
	start := time.Now()
	var r build.ExecResult
	if !c.dryRun {
		r = exec(ctx, step)
	}
	if step.IsPhonyOutput() {
		return r
	}
	end := time.Now()

	var build = BuildInfo{
		CmdHash: step.CmdHash(),
		Time:    end,
	}

	// Repopulate the cached files timestamps.
	for _, f := range step.Node.Targets {
		m, err := statFileMeta(f)
		if err != nil {
			c.params.Log.Errorf("backend failed to get timestamp of %s: %v", f, err)
			continue
		}
		c.files.Put(f, m)
		if m.IsMissing() && r.Err == nil {
			c.params.Log.Warnf("backend target file is missing %s", f)
		}

		c.builds.Put(f, &build)
	}

	if !c.dryRun && r.Err == nil {
		c.AddNewEntriesToLog(step, start, end)

		// Remove the response file after successful execution.
		if rspFile := step.RSPFile(); rspFile != "" && !c.params.CliFlags.DebugKeepRSP {
			os.Remove(rspFile)
		}

		// Update .ninja_deps.
		depFile := step.Node.DepFile()
		if depFile != "" {
			deps, err := parse.ReadDepFile(step.Node.Targets, depFile)
			if err != nil {
				c.params.Log.Errorf("backend could not read dep file (%q) for %s: %v", depFile, step.Node.Targets, err)
			} else {
				c.WriteDeps(step, end, deps)
				if !c.params.CliFlags.DebugKeepDepFile {
					os.Remove(depFile)
				}
			}
		}
	}

	return r
}

// AddNewEntriesToLog appends an entry per output target of step to the ninja
// log.
func (c *ClassicBackend) AddNewEntriesToLog(step *build.Step, start, end time.Time) {
	entry := &ninjalog.LogEntry{
		Start:     start.Sub(c.startTime),
		End:       end.Sub(c.startTime),
		Timestamp: end,
		Checksum:  fmt.Sprintf("%016x", step.CmdHash()),
	}
	for _, f := range step.Node.Targets {
		entry.Target = f
		if err := c.logFile.WriteEntry(entry); err != nil {
			c.params.Log.Errorf("backend could not write log entry for %s: %v", f, err)
		}
	}
}

// WriteDeps asynchronously appends the given deps for step to the .ninja_deps
// log.
func (c *ClassicBackend) WriteDeps(step *build.Step, end time.Time, deps []string) {
	c.pendingDeps.Add(1)
	go func() {
		defer c.pendingDeps.Done()
		if err := c.depsFile.NewDeps(step.Node.Targets[0], end, deps); err != nil {
			c.params.Log.Errorf("backend failed to record %d deps (%q) for %s: %v", len(deps), step.Node.DepFile(), step.Node.Targets, err)
		}
	}()
}

// CreateDirs implements build.BuildBackend.
func (c *ClassicBackend) CreateDirs(step *build.Step) error {
	if c.dryRun {
		return nil
	}

	for _, t := range step.Node.Targets {
		if err := os.MkdirAll(path.Dir(t), 0755); err != nil {
			return err
		}
	}

	depFile := step.Node.DepFile()
	if depFile != "" {
		if err := os.MkdirAll(path.Dir(depFile), 0755); err != nil {
			return err
		}
	}
	return nil
}

// CreateRSP implements build.BuildBackend.
func (c *ClassicBackend) CreateRSP(step *build.Step) error {
	if c.dryRun {
		return nil
	}

	rspFile := step.RSPFile()
	if rspFile == "" {
		return nil
	}

	if err := os.MkdirAll(path.Dir(rspFile), 0755); err != nil {
		return err
	}

	err := os.WriteFile(rspFile, []byte(step.RSPContent()), 0644)
	if err != nil {
		return err
	}

	c.files.Put(rspFile, &FileMeta{
		Mode:    0644,
		ModTime: time.Now(),
	})

	return err
}

// ReadNinjaFile implements parse.ParseBackend.
func (c *ClassicBackend) ReadNinjaFile(fileName string) ([]byte, parse.ReleaseFileFunc, error) {
	return runtime.ReadMMapFile(fileName)
}

// FinalizeStep implements build.BuildBackend.
func (c *ClassicBackend) FinalizeStep(step *build.Step) {
}

// FinalizeBuild implements build.BuildBackend.
func (c *ClassicBackend) FinalizeBuild(err error) error {
	c.pendingDeps.Wait()

	if !c.dryRun {
		if err2 := c.logFile.Close(); err2 != nil && err == nil {
			err = err2
		}

		if err2 := c.depsFile.Close(); err2 != nil && err == nil {
			err = err2
		}
	}

	return err
}
