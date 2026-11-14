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
	"errors"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryFileSystemKeepsPrivateCopies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		writes   map[string]string
		name     string
		lookup   string
		expected string
		found    bool
	}{
		{name: "written file", writes: map[string]string{"b.s": "two", "a.s": "one"}, lookup: "a.s", expected: "one", found: true},
		{name: "missing file", writes: map[string]string{"a.s": "one"}, lookup: "c.s", expected: "", found: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fileSystem := NewMemoryFileSystem()
			for path, content := range testCase.writes {
				data := []byte(content)
				require.NoError(t, fileSystem.WriteFile(path, data))
				data[0] = 'X'
			}
			data, ok := fileSystem.File(testCase.lookup)
			assert.Equal(t, testCase.found, ok)
			assert.Equal(t, testCase.expected, string(data))
			assert.Len(t, fileSystem.Paths(), len(testCase.writes))
			assert.IsNonDecreasing(t, fileSystem.Paths())
		})
	}
}

func TestMemoryFileSystemReadFile(t *testing.T) {
	t.Parallel()

	fileSystem := NewMemoryFileSystem()
	require.NoError(t, fileSystem.WriteFile("a.s", []byte("one")))

	data, err := fileSystem.ReadFile("a.s")
	require.NoError(t, err)
	assert.Equal(t, "one", string(data))
	data[0] = 'X'
	again, err := fileSystem.ReadFile("a.s")
	require.NoError(t, err)
	assert.Equal(t, "one", string(again), "ReadFile must return a private copy")

	_, err = fileSystem.ReadFile("missing.s")
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}
