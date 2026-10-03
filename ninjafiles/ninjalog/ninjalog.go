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

// Package ninjalog reads and writes the .ninja_log file that records the
// command hash and timing for each previously built target so ninja can
// detect dirty steps on the next build.
package ninjalog

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aviddiviner/go-murmur"
)

// Constants describing the ninja log file format.
const (
	// DefaultLogFileName is the default location of the .ninja_log file
	// relative to the build directory.
	DefaultLogFileName = ".ninja_log"
	ninjaLogHeader     = "# ninja log v5"
)

// NinjaLog is a read/write handle for a .ninja_log file.
type NinjaLog struct {
	fileName  string
	writeLock sync.Mutex
	writeFile *os.File
}

// LogEntry describes a single line in the ninja log.
type LogEntry struct {
	Start     time.Duration
	End       time.Duration
	Timestamp time.Time
	Target    string
	Checksum  string
}

// New returns a NinjaLog backed by the given file path. The file is not
// opened until OpenForWrite or Parse is called.
func New(fileName string) *NinjaLog {
	return &NinjaLog{fileName: fileName}
}

// OpenForWrite opens the log file for appending, writing the header if the
// file is empty.
func (n *NinjaLog) OpenForWrite() error {
	n.writeLock.Lock()
	defer n.writeLock.Unlock()

	if n.writeFile != nil {
		return fmt.Errorf("ninja log (%q) already open for write", n.fileName)
	}

	f, err := os.OpenFile(n.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	if fi.Size() == 0 {
		fmt.Fprintln(f, ninjaLogHeader)
	}
	n.writeFile = f
	return nil
}

// WriteEntry appends entry to the log.
func (n *NinjaLog) WriteEntry(entry *LogEntry) error {
	n.writeLock.Lock()
	defer n.writeLock.Unlock()

	if n.writeFile == nil {
		return fmt.Errorf("ninja log (%q) not open for write", n.fileName)
	}

	_, err := fmt.Fprintf(n.writeFile, "%d\t%d\t%d\t%s\t%s\n", entry.Start/time.Millisecond, entry.End/time.Millisecond, entry.Timestamp.UnixNano(), entry.Target, entry.Checksum)
	return err
}

// Close closes the underlying log file.
func (n *NinjaLog) Close() error {
	n.writeLock.Lock()
	defer n.writeLock.Unlock()

	return n.writeFile.Close()
}

// HashCommand returns the 64-bit MurmurHash of cmdLine and (optionally) the
// response file content, matching the hashing scheme used in upstream ninja's
// log file.
func HashCommand(cmdLine, rspContent string) uint64 {
	var toBeHashed []byte

	if rspContent != "" {
		buf := new(bytes.Buffer)
		buf.WriteString(cmdLine)
		buf.WriteString(";rspfile=")
		buf.WriteString(rspContent)
		toBeHashed = buf.Bytes()
	} else {
		toBeHashed = []byte(cmdLine)
	}

	return murmur.MurmurHash64A(toBeHashed, 0xDECAFBADDECAFBAD)
}
