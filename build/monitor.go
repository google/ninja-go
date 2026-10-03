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
	"fmt"
	"log"
	"os"
	"time"

	"go.uber.org/atomic"
)

// monitorCheckInterval is the period at which PSI files are sampled.
const monitorCheckInterval = 1 * time.Second

// SchedMonitor watches Linux pressure-stall information (PSI) and pauses
// scheduling when CPU, IO, or memory pressure exceeds the configured limits.
type SchedMonitor struct {
	cpuPSI     float64
	ioPSI      float64
	memPSI     float64
	currentCPU atomic.Float64
	currentIO  atomic.Float64
	currentMem atomic.Float64
}

// newSchedMonitor returns a SchedMonitor that samples each non-zero PSI
// resource in its own goroutine.
func newSchedMonitor(cpuPSI, ioPSI, memPSI float64) *SchedMonitor {
	m := &SchedMonitor{
		cpuPSI: cpuPSI,
		ioPSI:  ioPSI,
		memPSI: memPSI,
	}
	if cpuPSI > 0 {
		go m.monitor("cpu", &m.currentCPU, cpuPSI)
	}
	if ioPSI > 0 {
		go m.monitor("io", &m.currentIO, ioPSI)
	}
	if memPSI > 0 {
		go m.monitor("memory", &m.currentMem, memPSI)
	}
	return m
}

// PressureItem holds the four values reported on each line of a PSI file.
type PressureItem struct {
	Avg10  float64
	Avg60  float64
	Avg300 float64
	Total  int64
}

// Pressure holds the "some" and "full" lines reported by /proc/pressure/*.
type Pressure struct {
	Some PressureItem
	Full PressureItem
}

// readPressureInfo reads and parses /proc/pressure/<resource>.
func readPressureInfo(resource string) (Pressure, error) {
	var p Pressure
	path := fmt.Sprintf("/proc/pressure/%s", resource)
	content, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("failed to read %s pressure info: %w", resource, err)
	}
	n, err := fmt.Sscanf(string(content), "some avg10=%f avg60=%f avg300=%f total=%d\nfull avg10=%f avg60=%f avg300=%f total=%d\n",
		&p.Some.Avg10, &p.Some.Avg60, &p.Some.Avg300, &p.Some.Total,
		&p.Full.Avg10, &p.Full.Avg60, &p.Full.Avg300, &p.Full.Total,
	)
	if err != nil {
		return p, fmt.Errorf("failed to parse %s pressure info: %w", resource, err)
	}
	if n != 8 {
		return p, fmt.Errorf("failed to parse %s pressure info: could only parse %d fields", resource, n)
	}
	return p, nil
}

// monitor periodically samples the PSI for res and updates value, logging
// transitions across limit.
func (m *SchedMonitor) monitor(res string, value *atomic.Float64, limit float64) {
	for range time.NewTicker(monitorCheckInterval).C {
		p, err := readPressureInfo(res)
		if err != nil {
			log.Printf("failed to read %s psi: %v", res, err)
			continue
		}
		old := (*value).Load()
		cur := p.Some.Avg10
		if old < limit && cur >= limit {
			log.Printf("system under %s pressure (%f >= %f), pausing scheduling new jobs", res, cur, limit)
		}
		if old >= limit && cur < limit {
			log.Printf("system no longer under %s pressure (%f < %f), resuming scheduling new jobs", res, cur, limit)
		}
		(*value).Store(cur)
	}
}

// MayI blocks while any monitored resource is above its configured threshold,
// returning true once it is safe to schedule a new job.
func (m *SchedMonitor) MayI() bool {
	for m.underPressure() {
		time.Sleep(time.Second)
	}
	return true
}

// underPressure reports whether any monitored resource currently exceeds its
// configured threshold.
func (m *SchedMonitor) underPressure() bool {
	if m.cpuPSI > 0 && m.cpuPSI <= m.currentCPU.Load() {
		return true
	}
	if m.ioPSI > 0 && m.ioPSI <= m.currentIO.Load() {
		return true
	}
	if m.memPSI > 0 && m.memPSI <= m.currentMem.Load() {
		return true
	}
	return false
}
