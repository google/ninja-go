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

package ninjalog

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// fieldsCount is the number of tab-separated fields per ninja log entry.
const fieldsCount = 5

// Parse reads and parses the ninja log file. A non-existent file returns a
// nil slice and no error.
func (n *NinjaLog) Parse() ([]*LogEntry, error) {
	f, err := os.Open(n.fileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	return parse(f)
}

// parse decodes ninja log entries from logReader.
func parse(logReader io.Reader) ([]*LogEntry, error) {

	s := bufio.NewScanner(logReader)

	if !s.Scan() {
		return nil, s.Err()
	}
	header := s.Text()
	if header != ninjaLogHeader {
		return nil, fmt.Errorf("incorrect header: %q", header)
	}
	var (
		start    int64
		end      int64
		ts       int64
		target   string
		checksum string
	)

	var entries []*LogEntry
	for s.Scan() {
		if len(s.Text()) == 0 || s.Text()[0] == '#' {
			continue
		}

		bb := bytes.NewBuffer(s.Bytes())
		n, err := fmt.Fscanf(bb, "%d\t%d\t%d\t%s\t%s",
			&start, &end, &ts, &target, &checksum)
		if err != nil {
			return nil, fmt.Errorf("could not parse %q: %w", bb.String(), err)
		}
		if n != fieldsCount {
			return nil, fmt.Errorf("expected %d arguments got %d", fieldsCount, n)
		}

		entries = append(entries, &LogEntry{
			Start:     time.Millisecond * time.Duration(start),
			End:       time.Millisecond * time.Duration(end),
			Timestamp: time.Unix(0, ts),
			Target:    target,
			Checksum:  checksum,
		})
	}
	if s.Err() != nil {
		return nil, s.Err()
	}

	return entries, nil
}
