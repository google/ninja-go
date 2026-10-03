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

// Package ninjadeps reads and writes the .ninja_deps log, the binary
// per-target dependency database that ninja uses to remember discovered
// header dependencies between builds.
package ninjadeps

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/google/ninja-go/runtime"
)

// On-disk constants for the .ninja_deps file format.
const (
	// DefaultLogFileName is the default location of the .ninja_deps file
	// relative to the build directory.
	DefaultLogFileName = ".ninja_deps"
	ninjaDepsPrefix    = "# ninjadeps\n"
	maxRecordSize      = (1 << 19) - 1 // 512KB - 1, same as Ninja's deps log.
	fileVersion        = 4
)

// Kind identifies the kind of record decoded from the deps log.
type Kind int

// Kind values.
const (
	InvalidKind Kind = iota
	PathRecordKind
	DepsRecordKind
)

// InputRecord is the interface implemented by records read from the deps log.
type InputRecord interface {
	Base() *BaseRecord
}

// BaseRecord holds the fields common to every record kind.
type BaseRecord struct {
	size uint32
	kind Kind
}

// Base returns b, allowing BaseRecord to satisfy InputRecord.
func (b *BaseRecord) Base() *BaseRecord {
	return b
}

// PathRecord is a path-table entry from the deps log.
type PathRecord struct {
	*BaseRecord
	path     string
	checksum uint32
}

// DepsRecord is a dependency-list entry from the deps log.
type DepsRecord struct {
	*BaseRecord
	id    uint32
	mtime time.Time
	deps  []uint32
}

// Path is the in-memory representation of a single output and its recorded
// dependencies.
type Path struct {
	id    uint32
	path  string
	mtime time.Time
	deps  []uint32
}

// NinjaDeps is a read/write handle for a .ninja_deps file.
type NinjaDeps struct {
	fileName    string
	pathLock    sync.RWMutex
	paths       []*Path
	pathsByName map[string]*Path
	writeLock   sync.Mutex
	writeBuf    []byte
	zerosBuf    []byte
	file        *os.File
}

// New returns a NinjaDeps backed by the given file path. The file is not
// opened until Load or OpenForWrite is called.
func New(fileName string) *NinjaDeps {
	return &NinjaDeps{
		fileName:    fileName,
		pathsByName: make(map[string]*Path),
		writeBuf:    make([]byte, 8),
		zerosBuf:    make([]byte, 4),
	}
}

// OpenForWrite opens the deps log for appending, writing the header if the
// file is empty.
func (d *NinjaDeps) OpenForWrite() error {
	f, err := os.OpenFile(d.fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		_, err = f.Write([]byte(ninjaDepsPrefix))
		if err != nil {
			return err
		}
		if err := d.writeUint32(f, fileVersion); err != nil {
			return err
		}
	}
	d.file = f
	return f.Sync()
}

// Close closes the underlying deps file.
func (d *NinjaDeps) Close() error {
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	return d.file.Close()
}

// GetDeps returns the recorded mtime and dependency list for path, or zero
// values if no record exists.
func (d *NinjaDeps) GetDeps(path string) (time.Time, []string) {
	d.pathLock.RLock()
	defer d.pathLock.RUnlock()

	p, ok := d.pathsByName[path]
	if !ok {
		return time.Time{}, nil
	}
	deps := make([]string, len(p.deps))
	for i, id := range p.deps {
		deps[i] = d.paths[int(id)].path
	}
	return p.mtime, deps
}

// path returns the Path for the given file, allocating and persisting a new
// one on first use.
func (d *NinjaDeps) path(path string) (*Path, error) {
	d.pathLock.RLock()
	p, ok := d.pathsByName[path]
	d.pathLock.RUnlock()
	if ok {
		return p, nil
	}

	d.pathLock.Lock()
	defer d.pathLock.Unlock()
	p, ok = d.pathsByName[path]
	if ok {
		return p, nil
	}

	p = &Path{
		id:   uint32(len(d.paths)),
		path: path,
	}
	if err := d.writePath(p); err != nil {
		return nil, err
	}
	d.paths = append(d.paths, p)
	d.pathsByName[path] = p
	return p, nil
}

// NewDeps records that path was last built at mtime and depended on deps.
func (d *NinjaDeps) NewDeps(path string, mtime time.Time, deps []string) error {
	p, err := d.path(path)
	if err != nil {
		return err
	}
	depIDs := make([]uint32, len(deps))
	for i, dep := range deps {
		dp, err := d.path(dep)
		if err != nil {
			return err
		}
		depIDs[i] = dp.id
	}
	p.mtime = mtime
	p.deps = depIDs
	if err := d.writeDeps(p); err != nil {
		return err
	}
	return nil
}

// Load reads the deps file from disk into memory. A non-existent file is
// treated as empty.
func (d *NinjaDeps) Load() error {
	records, err := d.readRecords()
	if err != nil {
		return err
	}
	for _, r := range records {
		if n, ok := r.(*PathRecord); ok {
			p := &Path{
				id:   0xffffffff ^ n.checksum,
				path: n.path,
			}
			if int(p.id) != len(d.paths) {
				return fmt.Errorf("expected %d as node id but got %d", p.id, len(d.paths))
			}
			d.paths = append(d.paths, p)
			d.pathsByName[p.path] = p
		}
	}
	for i := len(records) - 1; i >= 0; i-- {
		if n, ok := records[i].(*DepsRecord); ok {
			if int(n.id) >= len(d.paths) {
				return fmt.Errorf("cannot find node %d", n.id)
			}
			p := d.paths[int(n.id)]
			if !p.mtime.IsZero() {
				// Stale record; a newer entry already updated this path.
				continue
			}
			p.mtime = n.mtime
			p.deps = n.deps
			for _, id := range n.deps {
				if int(id) >= len(d.paths) {
					return fmt.Errorf("cannot find dep node %d for %s", id, p.path)
				}
			}
		}
	}
	return nil
}

// DecodeInt32 decodes a little-endian 32-bit integer.
var DecodeInt32 = binary.LittleEndian.Uint32

// DecodeInt64 decodes a little-endian 64-bit integer.
var DecodeInt64 = binary.LittleEndian.Uint64

// writeUint32 writes a little-endian 32-bit integer using the shared buffer.
func (d *NinjaDeps) writeUint32(w io.Writer, i uint32) error {
	binary.LittleEndian.PutUint32(d.writeBuf, i)
	_, err := w.Write(d.writeBuf[:4])
	return err
}

// writeUint64 writes a little-endian 64-bit integer using the shared buffer.
func (d *NinjaDeps) writeUint64(w io.Writer, i uint64) error {
	binary.LittleEndian.PutUint64(d.writeBuf, i)
	_, err := w.Write(d.writeBuf[:8])
	return err
}

// readRecords reads and validates the deps file header, returning its parsed
// records.
func (d *NinjaDeps) readRecords() ([]InputRecord, error) {
	depsContent, _, err := runtime.ReadMMapFile(d.fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if bytes.HasPrefix(depsContent, []byte(ninjaDepsPrefix)) {
		depsContent = depsContent[len(ninjaDepsPrefix):]
	} else {
		return nil, fmt.Errorf("file %q is not a ninja deps file", d.fileName)
	}

	if len(depsContent) < 4 {
		return nil, fmt.Errorf("depfile %q is too short to read its version", d.fileName)
	}

	version := DecodeInt32(depsContent[0:4])
	if version != fileVersion {
		return nil, fmt.Errorf("depfile %q version %d is not supported", d.fileName, version)
	}
	depsContent = depsContent[4:]

	return parseRecords(depsContent)
}

// parseRecords decodes a sequence of records from the deps log body.
func parseRecords(content []byte) ([]InputRecord, error) {
	// TODO: parse records concurrently in the future to improve performance.
	records := make([]InputRecord, 0)
	for len(content) > 0 {
		r := parseRecord(content)
		if r.Base().kind == InvalidKind {
			fmt.Printf("bad record: %+v %q\n", r, content[0:12])
		}
		records = append(records, r)
		content = content[r.Base().size+4:]
	}
	return records, nil
}

// parseRecord decodes a single record from the start of buf.
func parseRecord(buf []byte) InputRecord {
	h := DecodeInt32(buf[0:4])
	size := h & 0x7fffffff
	base := &BaseRecord{
		kind: InvalidKind,
		size: size,
	}
	if size%4 != 0 || size > maxRecordSize {
		return base
	}
	buf = buf[4:]

	if len(buf) < int(size) {
		return base
	}

	if h&0x80000000 == 0 {
		// Path record (header, content, checksum).
		if size < 8 {
			return base
		}

		base.kind = PathRecordKind
		result := &PathRecord{
			BaseRecord: base,
			path:       string(bytes.TrimRight(buf[:size-4], "\x00")),
			checksum:   DecodeInt32(buf[size-4 : size]),
		}
		return result
	}

	// Deps record (header, output_id, mtime_lo, mtime_hi).
	if size < 12 {
		return base
	}
	base.kind = DepsRecordKind
	result := &DepsRecord{
		BaseRecord: base,
		id:         DecodeInt32(buf[:4]),
		mtime:      time.Unix(0, int64(DecodeInt64(buf[4:12]))),
	}
	for i := 12; i < int(size); i += 4 {
		result.deps = append(result.deps, DecodeInt32(buf[i:i+4]))
	}
	return result
}

// writePath appends a path record for p to the deps file.
func (d *NinjaDeps) writePath(p *Path) error {
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	path := []byte(p.path)
	pad := (4 - len(path)%4) % 4

	size := len(path) + pad + 4
	if size > maxRecordSize {
		return fmt.Errorf("path record for %s is too big (%d)", p.path, size)
	}

	if err := d.writeUint32(d.file, uint32(size)); err != nil {
		return err
	}

	if _, err := d.file.Write(path); err != nil {
		return err
	}

	if _, err := d.file.Write(d.zerosBuf[:pad]); err != nil {
		return err
	}

	if err := d.writeUint32(d.file, 0xffffffff^p.id); err != nil {
		return err
	}

	return nil
}

// writeDeps appends a deps record for p to the deps file.
func (d *NinjaDeps) writeDeps(p *Path) error {
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	size := 4 * (1 + 2 + len(p.deps))

	if size > maxRecordSize {
		return fmt.Errorf("deps record for %s is too big (%d)", p.path, size)
	}

	size |= 0x80000000 // Deps record: set high bit.

	if err := d.writeUint32(d.file, uint32(size)); err != nil {
		return err
	}

	if err := d.writeUint32(d.file, p.id); err != nil {
		return err
	}

	if err := d.writeUint64(d.file, uint64(p.mtime.UnixNano())); err != nil {
		return err
	}

	for _, id := range p.deps {
		if err := d.writeUint32(d.file, id); err != nil {
			return err
		}
	}

	return nil
}
