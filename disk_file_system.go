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
	"fmt"
	"path/filepath"
	"sync"

	"piko.sh/safedisk"
)

const (
	// generatedFilePermission is the file mode of every generated source file.
	generatedFilePermission = 0o644

	// directoryPermission is the file mode used when creating parent directories.
	directoryPermission = 0o755

	// defaultMaxReadBytes is the default cap on the size of an on-disk file read during
	// validation.
	defaultMaxReadBytes int64 = 64 << 20
)

// DiskFileSystem implements FileSystemWriterPort and FileSystemReaderPort on the local
// filesystem, confined to a project root by a safedisk sandbox.
type DiskFileSystem struct {
	// sandbox confines every read and write to the project root.
	sandbox safedisk.Sandbox

	// maxReadBytes caps the size of a file read back for validation.
	maxReadBytes int64

	// closeOnce makes Close idempotent.
	closeOnce sync.Once
}

// DiskOption configures a DiskFileSystem.
type DiskOption func(*diskConfig)

// diskConfig holds the resolved DiskFileSystem options.
type diskConfig struct {
	// maxReadBytes caps the size of a file read back for validation.
	maxReadBytes int64
}

var (
	_ FileSystemWriterPort = (*DiskFileSystem)(nil)

	_ FileSystemReaderPort = (*DiskFileSystem)(nil)
)

// NewDiskFileSystem opens a read-write sandbox rooted at root, creating the directory if
// it does not exist.
//
// Takes root (string) which is the project root every generated path is relative to.
// Takes options (...DiskOption) which configure the file system.
//
// Returns *DiskFileSystem which must be closed when no longer needed.
// Returns error when an option is invalid or the sandbox cannot be opened.
func NewDiskFileSystem(root string, options ...DiskOption) (*DiskFileSystem, error) {
	config, err := newDiskConfig(options)
	if err != nil {
		return nil, err
	}
	sandbox, err := safedisk.NewSandbox(root, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("opening project root %q: %w", root, err)
	}
	return &DiskFileSystem{sandbox: sandbox, maxReadBytes: config.maxReadBytes, closeOnce: sync.Once{}}, nil
}

// NewDiskFileSystemFromSandbox takes ownership of an existing sandbox.
//
// Takes sandbox (safedisk.Sandbox) which confines every read and write.
// Takes options (...DiskOption) which configure the file system.
//
// Returns *DiskFileSystem which must be closed when no longer needed.
// Returns error when the sandbox is nil or an option is invalid.
func NewDiskFileSystemFromSandbox(sandbox safedisk.Sandbox, options ...DiskOption) (*DiskFileSystem, error) {
	if sandbox == nil {
		return nil, fmt.Errorf("%w: sandbox is nil", ErrInvalidOption)
	}
	config, err := newDiskConfig(options)
	if err != nil {
		return nil, err
	}
	return &DiskFileSystem{sandbox: sandbox, maxReadBytes: config.maxReadBytes, closeOnce: sync.Once{}}, nil
}

// WriteFile writes data to path atomically, creating parent directories as needed.
//
// Takes path (string) which is the output path, relative to the project root.
// Takes data ([]byte) which is the file content.
//
// Returns error when the path is invalid or the directory or file cannot be written.
func (d *DiskFileSystem) WriteFile(path string, data []byte) error {
	if err := validateRelativePath(path); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := d.sandbox.MkdirAll(directory, directoryPermission); err != nil {
		return fmt.Errorf("creating directory %q: %w", directory, err)
	}
	if err := d.sandbox.WriteFileAtomic(path, data, generatedFilePermission); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}

// ReadFile reads the file at path, refusing files larger than the read limit.
//
// Takes path (string) which is the file path, relative to the project root.
//
// Returns []byte which is the file content.
// Returns error which wraps fs.ErrNotExist when the file does not exist, or
// ErrExistingFileTooLarge when it exceeds the read limit.
func (d *DiskFileSystem) ReadFile(path string) ([]byte, error) {
	if err := validateRelativePath(path); err != nil {
		return nil, err
	}
	data, size, err := d.sandbox.ReadFileLimit(path, d.maxReadBytes)
	if errors.Is(err, safedisk.ErrFileExceedsLimit) {
		return nil, fmt.Errorf("%w: %q is %d bytes, limit %d", ErrExistingFileTooLarge, path, size, d.maxReadBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	if int64(len(data)) < size {
		return nil, fmt.Errorf("%w: %q read %d of %d bytes", ErrExistingFileTooLarge, path, len(data), size)
	}
	return data, nil
}

// Close releases the sandbox and returns nil if it was already closed.
//
// Returns error when the sandbox fails to close.
func (d *DiskFileSystem) Close() error {
	var err error
	d.closeOnce.Do(func() {
		if closeErr := d.sandbox.Close(); closeErr != nil {
			err = fmt.Errorf("closing project root: %w", closeErr)
		}
	})
	return err
}

// WithMaxReadBytes sets the validation read limit, which defaults to 64 MiB.
//
// Takes maxBytes (int64) which is the positive size cap in bytes.
//
// Returns DiskOption which applies the cap.
func WithMaxReadBytes(maxBytes int64) DiskOption {
	return func(config *diskConfig) { config.maxReadBytes = maxBytes }
}

// newDiskConfig resolves DiskFileSystem options over the defaults.
//
// Takes options ([]DiskOption) which are applied in order.
//
// Returns diskConfig which holds the resolved options.
// Returns error when an option value is invalid.
func newDiskConfig(options []DiskOption) (diskConfig, error) {
	config := diskConfig{maxReadBytes: defaultMaxReadBytes}
	for _, option := range options {
		option(&config)
	}
	if config.maxReadBytes <= 0 {
		return diskConfig{}, fmt.Errorf("%w: max read bytes must be positive, got %d", ErrInvalidOption, config.maxReadBytes)
	}
	return config, nil
}

// validateRelativePath checks that path is a non-empty local path that stays inside the
// project root.
//
// Takes path (string) which is the path to check.
//
// Returns error which wraps ErrInvalidPath when the path is empty, absolute or escapes the
// root.
func validateRelativePath(path string) error {
	if path == "" || !filepath.IsLocal(path) {
		return fmt.Errorf("%w: %q must be a relative path inside the project root", ErrInvalidPath, path)
	}
	return nil
}
