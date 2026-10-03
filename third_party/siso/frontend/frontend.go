// Copyright 2025 The Chromium Authors, Google LLC
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package frontend serializes build status updates over the Android ninja
// frontend protocol so that Soong (or another consumer reading the frontend
// FIFO) can render its own UI.
package frontend

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	pb "github.com/google/ninja-go/third_party/siso/frontend/proto"
)

// Frontend provides the same functionality as Android ninja's StatusSerializer.
// Soong reads status updates via the frontend FIFO file and updates its UI accordingly.
type Frontend struct {
	w         io.Writer
	verbosity int
	quiet     bool

	startTime time.Time
	ch        chan *pb.Status
	quit      chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	building  bool
	total     int
}

// NewFrontend creates new frontend to write to w.
func NewFrontend(w io.Writer, verbocity int, quiet bool) *Frontend {
	f := &Frontend{
		w:         w,
		verbosity: verbocity,
		quiet:     quiet,
		ch:        make(chan *pb.Status, 10000),
		quit:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	go f.run()
	return f
}

// run is the goroutine that serializes Status messages to the writer until
// Close is called.
func (f *Frontend) run() {
	defer close(f.done)
	var buf []byte
	for {
		select {
		case <-f.quit:
			return
		case m := <-f.ch:
			// Send the proto as a length-delimited message: a varint size
			// (without a tag) followed by the marshaled bytes.
			b, err := proto.Marshal(m)
			if err != nil {
				log.Printf("failed to marshal: %v", err)
				continue
			}
			n := uint64(len(b))
			buf = buf[:0]
			buf = protowire.AppendVarint(buf, n)
			buf = append(buf, b...)
			_, err = f.w.Write(buf)
			if err != nil {
				log.Printf("failed to send status: %v", err)
				return
			}
		}
	}
}

// Close closes the frontend and finishes writing.
func (f *Frontend) Close() error {
	close(f.quit)
	<-f.done
	return nil
}

// PlanHasTotalSteps is called when the total number of build steps changes.
func (f *Frontend) PlanHasTotalSteps(total int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if total == f.total {
		return
	}
	f.total = total
	m := &pb.Status{
		TotalEdges: &pb.Status_TotalEdges{
			TotalEdges: new(uint32(f.total)),
		},
	}
	f.ch <- m
}

// BuildStepStarted is called when build action started.
func (f *Frontend) BuildStepStarted(id int, cmd, desc string, outputs []string) {
	m := &pb.Status{
		EdgeStarted: &pb.Status_EdgeStarted{
			Id:        new(uint32(id)),
			StartTime: new(uint32(time.Since(f.startTime).Milliseconds())),
			Desc:      new(desc),
			Command:   new(cmd),
			Outputs:   outputs,
			// TODO: pass more info?
		},
	}
	f.ch <- m
}

// BuildStepFinished is called when build action finished.
func (f *Frontend) BuildStepFinished(id int, exitCode int, output string) {
	m := &pb.Status{
		EdgeFinished: &pb.Status_EdgeFinished{
			Id:      new(uint32(id)),
			EndTime: new(uint32(time.Since(f.startTime).Milliseconds())),
			Status:  new(int32(exitCode)),
			Output:  new(output),
			// TODO: pass more info?
		},
	}
	f.ch <- m
}

// BuildStepCanceled is called when build action is canceled.
func (f *Frontend) BuildStepCanceled(id int) {
	m := &pb.Status{
		EdgeFinished: &pb.Status_EdgeFinished{
			Id:       new(uint32(id)),
			EndTime:  new(uint32(time.Since(f.startTime).Milliseconds())),
			Canceled: new(true),
		},
	}
	f.ch <- m
}

// BuildStarted is called when build started.
// Once started, PrintLines will be suppressed to ignore build progress
// since these are reported bia BuildStepStarted / BuildStepFinished.
func (f *Frontend) BuildStarted() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.startTime = time.Now()
	f.building = true
	// TODO: pass more info?
	m := &pb.Status{
		BuildStarted: &pb.Status_BuildStarted{},
	}
	f.ch <- m
}

// BuildFinished is called when build finished.
func (f *Frontend) BuildFinished() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.building = false
	m := &pb.Status{
		BuildFinished: &pb.Status_BuildFinished{},
	}
	f.ch <- m
}

// message enqueues a Status message at the given log level.
func (f *Frontend) message(level pb.Status_Message_Level, msg string) {
	m := &pb.Status{
		Message: &pb.Status_Message{
			Level:   level.Enum(),
			Message: new(msg),
		},
	}
	f.ch <- m
}

// Printf emits the formatted message at info level.
func (f *Frontend) Printf(format string, args ...any) {
	f.message(pb.Status_Message_INFO, fmt.Sprintf(format, args...))
}

// Tracef emits the formatted message at debug level when verbosity > 1.
func (f *Frontend) Tracef(format string, args ...any) {
	if f.verbosity > 1 {
		f.message(pb.Status_Message_DEBUG, fmt.Sprintf(format, args...))
	}
}

// Debugf emits the formatted message at debug level when verbosity > 0.
func (f *Frontend) Debugf(format string, args ...any) {
	if f.verbosity > 0 {
		f.message(pb.Status_Message_DEBUG, fmt.Sprintf(format, args...))
	}
}

// Infof emits the formatted message at info level unless --quiet is set.
func (f *Frontend) Infof(format string, args ...any) {
	if !f.quiet {
		f.message(pb.Status_Message_INFO, fmt.Sprintf(format, args...))
	}
}

// Warnf emits the formatted message at warning level unless --quiet is set.
func (f *Frontend) Warnf(format string, args ...any) {
	if !f.quiet {
		f.message(pb.Status_Message_WARNING, fmt.Sprintf(format, args...))
	}
}

// Errorf emits the formatted message at error level.
func (f *Frontend) Errorf(format string, args ...any) {
	f.message(pb.Status_Message_ERROR, fmt.Sprintf(format, args...))
}
