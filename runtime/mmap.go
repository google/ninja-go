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
	"os"
	"syscall"
)

// CloseFunc releases resources associated with a mapped or opened file.
type CloseFunc = func() error

// ReadMMapFile memory-maps the named file for read-only access. The returned
// CloseFunc unmaps and closes the underlying file.
func ReadMMapFile(fileName string) ([]byte, CloseFunc, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	size := stat.Size()
	if size == 0 {
		file.Close()
		return []byte{}, func() error {
			return nil
		}, nil
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return data, func() error {
		if err := syscall.Munmap(data); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	}, nil
}
