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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/safedisk"
)

type shortReadSandbox struct {
	*safedisk.MockSandbox
}

func (s *shortReadSandbox) ReadFileLimit(string, int64) ([]byte, int64, error) {
	return []byte("ab"), 4, nil
}

func newMockFileSystem(t *testing.T, sandbox safedisk.Sandbox, options ...DiskOption) *DiskFileSystem {
	t.Helper()
	fileSystem, err := NewDiskFileSystemFromSandbox(sandbox, options...)
	require.NoError(t, err)
	return fileSystem
}

func TestDiskFileSystemRoundTrip(t *testing.T) {
	t.Parallel()

	fileSystem, root := newDiskFileSystem(t)
	target := filepath.Join("sub", "out.s")
	content := []byte("line1\nline2\n")

	require.NoError(t, fileSystem.WriteFile(target, content))
	require.NoError(t, fileSystem.WriteFile(target, []byte("replaced\n")))
	require.NoError(t, fileSystem.WriteFile(target, content))

	info, err := os.Stat(filepath.Join(root, target))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(generatedFilePermission), info.Mode().Perm())

	data, err := fileSystem.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	entries, err := os.ReadDir(filepath.Join(root, "sub"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "atomic writes must not leave temporary files")
	assert.Equal(t, "out.s", entries[0].Name())

	_, err = fileSystem.ReadFile("missing.s")
	assert.True(t, errors.Is(err, fs.ErrNotExist), "missing files must wrap fs.ErrNotExist, got %v", err)
}

func TestDiskFileSystemRejectsPathsOutsideTheRoot(t *testing.T) {
	t.Parallel()

	fileSystem, root := newDiskFileSystem(t)
	for _, path := range []string{"", "../escape.s", "/tmp/absolute.s", filepath.Join("a", "..", "..", "b.s")} {
		require.ErrorIs(t, fileSystem.WriteFile(path, []byte("x")), ErrInvalidPath, path)
		_, err := fileSystem.ReadFile(path)
		require.ErrorIs(t, err, ErrInvalidPath, path)
	}
	_, err := os.Stat(filepath.Join(filepath.Dir(root), "escape.s"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestDiskFileSystemEnforcesTheReadLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fileSystem, err := NewDiskFileSystem(root, WithMaxReadBytes(4))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fileSystem.Close()) })

	require.NoError(t, fileSystem.WriteFile("big.s", []byte(strings.Repeat("x", 5))))
	_, err = fileSystem.ReadFile("big.s")
	require.ErrorIs(t, err, ErrExistingFileTooLarge)

	short := newMockFileSystem(t, &shortReadSandbox{MockSandbox: safedisk.NewMockSandbox("/root", safedisk.ModeReadWrite)})
	_, err = short.ReadFile("cut.s")
	require.ErrorIs(t, err, ErrExistingFileTooLarge, "a read shorter than the reported size must not pass silently")
}

func TestDiskFileSystemOptionsAndConstruction(t *testing.T) {
	t.Parallel()

	_, err := NewDiskFileSystem(t.TempDir(), WithMaxReadBytes(0))
	require.ErrorIs(t, err, ErrInvalidOption)

	_, err = NewDiskFileSystemFromSandbox(nil)
	require.ErrorIs(t, err, ErrInvalidOption)

	_, err = NewDiskFileSystemFromSandbox(safedisk.NewMockSandbox("/root", safedisk.ModeReadWrite), WithMaxReadBytes(-1))
	require.ErrorIs(t, err, ErrInvalidOption)

	_, err = NewDiskFileSystem("")
	require.Error(t, err)

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	_, err = NewDiskFileSystem(filepath.Join(blocker, "child"))
	require.Error(t, err, "a root beneath a regular file cannot be opened")
}

func TestDiskFileSystemPropagatesSandboxErrors(t *testing.T) {
	t.Parallel()

	sandboxError := errors.New("sandbox failure")
	testCases := []struct {
		configure func(sandbox *safedisk.MockSandbox)
		operation func(fileSystem *DiskFileSystem) error
		name      string
	}{
		{
			name:      "mkdir failure",
			configure: func(sandbox *safedisk.MockSandbox) { sandbox.MkdirAllErr = sandboxError },
			operation: func(fileSystem *DiskFileSystem) error { return fileSystem.WriteFile("a/b.s", nil) },
		},
		{
			name:      "atomic write failure",
			configure: func(sandbox *safedisk.MockSandbox) { sandbox.WriteFileAtomicErr = sandboxError },
			operation: func(fileSystem *DiskFileSystem) error { return fileSystem.WriteFile("b.s", nil) },
		},
		{
			name:      "read failure",
			configure: func(sandbox *safedisk.MockSandbox) { sandbox.ReadFileLimitErr = sandboxError },
			operation: func(fileSystem *DiskFileSystem) error {
				_, err := fileSystem.ReadFile("b.s")
				return err
			},
		},
		{
			name:      "close failure",
			configure: func(sandbox *safedisk.MockSandbox) { sandbox.CloseErr = sandboxError },
			operation: func(fileSystem *DiskFileSystem) error { return fileSystem.Close() },
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			sandbox := safedisk.NewMockSandbox("/root", safedisk.ModeReadWrite)
			testCase.configure(sandbox)
			require.ErrorIs(t, testCase.operation(newMockFileSystem(t, sandbox)), sandboxError)
		})
	}
}

func TestDiskFileSystemCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/root", safedisk.ModeReadWrite)
	fileSystem := newMockFileSystem(t, sandbox)
	require.NoError(t, fileSystem.Close())
	require.NoError(t, fileSystem.Close())
	assert.Equal(t, 1, sandbox.CallCounts["Close"])
}
