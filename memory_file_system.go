// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package asmgen

import (
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"sync"
)

// MemoryFileSystem keeps generated files in memory and is safe for concurrent use.
type MemoryFileSystem struct {
	// files maps each written path to a private copy of its content.
	files map[string][]byte

	// mu guards files.
	mu sync.Mutex
}

var (
	_ FileSystemWriterPort = (*MemoryFileSystem)(nil)

	_ FileSystemReaderPort = (*MemoryFileSystem)(nil)
)

// NewMemoryFileSystem creates an empty in-memory file system.
//
// Returns *MemoryFileSystem ready for use.
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{files: make(map[string][]byte), mu: sync.Mutex{}}
}

// WriteFile stores a copy of data under path, replacing any earlier content.
//
// Takes path (string) which is the output file path.
// Takes data ([]byte) which is the file content.
//
// Returns error which is always nil.
//
// Concurrency: Concurrent calls are protected by the file-map mutex.
func (m *MemoryFileSystem) WriteFile(path string, data []byte) error {
	clone := slices.Clone(data)
	m.mu.Lock()
	m.files[path] = clone
	m.mu.Unlock()
	return nil
}

// ReadFile returns a copy of the content stored under path.
//
// Takes path (string) which is the file path.
//
// Returns []byte which is a copy of the content.
// Returns error which wraps fs.ErrNotExist when nothing was written to path.
func (m *MemoryFileSystem) ReadFile(path string) ([]byte, error) {
	data, ok := m.File(path)
	if !ok {
		return nil, fmt.Errorf("reading %q: %w", path, fs.ErrNotExist)
	}
	return data, nil
}

// File returns the content written to path.
//
// Takes path (string) which is the output file path.
//
// Returns []byte which is a copy of the content.
// Returns bool which is false when nothing was written to path.
//
// Concurrency: Concurrent calls are protected by the file-map mutex.
func (m *MemoryFileSystem) File(path string) ([]byte, bool) {
	m.mu.Lock()
	data, ok := m.files[path]
	m.mu.Unlock()
	return slices.Clone(data), ok
}

// Paths returns every written path in sorted order.
//
// Returns []string which lists the paths.
//
// Concurrency: Concurrent calls are protected by the file-map mutex.
func (m *MemoryFileSystem) Paths() []string {
	m.mu.Lock()
	paths := slices.Collect(maps.Keys(m.files))
	m.mu.Unlock()
	slices.Sort(paths)
	return paths
}
