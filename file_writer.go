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
	"os"
	"path/filepath"
)

const (
	// filePermission is the file mode used when writing output files.
	filePermission = 0o600

	// directoryPermission is the file mode used when creating parent directories.
	directoryPermission = 0o755
)

// DiskWriter implements FileSystemWriterPort by writing files to the local filesystem.
type DiskWriter struct{}

var (
	_ FileSystemWriterPort = (*DiskWriter)(nil)
)

// NewDiskWriter creates a new filesystem writer.
//
// Returns *DiskWriter ready for use.
func NewDiskWriter() *DiskWriter {
	return &DiskWriter{}
}

// WriteFile writes data to the given path atomically.
//
// It creates the parent directory if it does not yet exist, then stages the content in a
// temporary file in the same directory and renames it into place. The rename is atomic on
// POSIX filesystems, so an interrupted build cannot leave a half-written file on disk.
//
// Takes path (string) which is the output file path.
// Takes data ([]byte) which is the file content.
//
// Returns error when directory creation or file writing fails.
func (*DiskWriter) WriteFile(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, directoryPermission); err != nil {
		return fmt.Errorf("creating directory %q: %w", directory, err)
	}
	return writeFileAtomic(path, data, filePermission)
}

// writeFileAtomic writes data to path by staging it in a temporary file in the same
// directory and renaming it into place, so readers never observe a partially written file.
//
// Takes path (string) which is the destination file path.
// Takes data ([]byte) which is the file content.
// Takes perm (os.FileMode) which is the mode applied to the destination file.
//
// Returns error when the temporary file cannot be written or renamed into place.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary file in %q: %w", directory, err)
	}
	temporaryPath := temporary.Name()

	defer func() { _ = os.Remove(temporaryPath) }()

	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("writing %q: %w", temporaryPath, err)
	}
	if err := temporary.Chmod(perm); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("setting mode on %q: %w", temporaryPath, err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("syncing %q: %w", temporaryPath, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", temporaryPath, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("renaming %q to %q: %w", temporaryPath, path, err)
	}
	return nil
}
