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

// ArchitecturePort defines the minimal driven port that each architecture adapter must
// implement. It provides only the identity information needed by the file generation
// framework to produce correctly named and tagged .s files.
//
// Domain-specific operations are defined as separate interfaces in their respective
// consumer packages, embedding ArchitecturePort. Adapters implement both ArchitecturePort
// and whichever domain interfaces they support. The framework does not prescribe any
// particular assembly operations; consumers bring their own.
type ArchitecturePort interface {
	// Arch returns the target architecture identifier.
	//
	// Returns Architecture which identifies the target.
	Arch() Architecture

	// BuildConstraint returns the build constraint suffix for this architecture.
	//
	// Returns string which is appended to a file group's build constraint.
	BuildConstraint() string
}

// HandlerValidator is an optional interface an architecture adapter implements to check
// each emitted handler body before it is written.
type HandlerValidator interface {
	// ValidateHandler checks the emitted body of the named handler.
	//
	// Takes name (string) which is the handler name.
	// Takes body (string) which is the emitted text without its comment or TEXT directive.
	//
	// Returns error which reports problems in the handler body.
	ValidateHandler(name string, body string) error
}

// FileSystemWriterPort writes generated assembly, header and Go files.
type FileSystemWriterPort interface {
	// WriteFile writes data to the given path, creating or overwriting the file.
	//
	// Takes path (string) which is the output file path.
	// Takes data ([]byte) which is the content to write.
	//
	// Returns error when the file cannot be written.
	WriteFile(path string, data []byte) error
}

// FileSystemReaderPort reads existing files so generated output can be compared with
// them.
type FileSystemReaderPort interface {
	// ReadFile returns the content of the file at path.
	//
	// Takes path (string) which is the file path.
	//
	// Returns []byte which is the file content.
	// Returns error which wraps fs.ErrNotExist when the file does not exist.
	ReadFile(path string) ([]byte, error)
}
