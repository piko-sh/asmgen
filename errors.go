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
)

var (
	// ErrInvalidPath is returned when an output directory or file path is empty, absolute
	// or escapes the project root.
	ErrInvalidPath = errors.New("asmgen: invalid path")

	// ErrInvalidName is returned when a file base name, handler name or generated file
	// name is empty or contains a path separator.
	ErrInvalidName = errors.New("asmgen: invalid name")

	// ErrMissingEmit is returned when a handler, header file or Go file has no Emit
	// function.
	ErrMissingEmit = errors.New("asmgen: missing emit function")

	// ErrDuplicateOutput is returned when two generated files resolve to the same path.
	ErrDuplicateOutput = errors.New("asmgen: duplicate output path")

	// ErrDuplicateHandler is returned when two handlers in one file share a name for the
	// same architecture.
	ErrDuplicateHandler = errors.New("asmgen: duplicate handler name")

	// ErrEmptyArchitectures is returned when no target architectures are given, or when an
	// Architectures restriction is non-nil but empty.
	ErrEmptyArchitectures = errors.New("asmgen: empty architecture list")

	// ErrUnknownArchitecture is returned when an adapter or override names an architecture
	// asmgen does not know, or when a target architecture is listed twice.
	ErrUnknownArchitecture = errors.New("asmgen: unknown or duplicate architecture")

	// ErrHandlerEmission wraps every failure recorded with Emitter.Fail.
	ErrHandlerEmission = errors.New("asmgen: handler emission failed")

	// ErrHandlerValidation wraps every problem an architecture adapter reports while
	// validating an emitted handler body.
	ErrHandlerValidation = errors.New("asmgen: handler validation failed")

	// ErrUnknownSlot is returned when a signature has no slot with the requested name.
	ErrUnknownSlot = errors.New("asmgen: unknown signature slot")

	// ErrInvalidParameter is returned when a signature parameter has an empty or duplicate
	// name, or an unknown kind.
	ErrInvalidParameter = errors.New("asmgen: invalid signature parameter")

	// ErrInvalidFrame is returned when a TEXT frame is given a negative local size.
	ErrInvalidFrame = errors.New("asmgen: invalid frame size")

	// ErrExistingFileTooLarge is returned when an on-disk file being validated is larger
	// than the configured read limit.
	ErrExistingFileTooLarge = errors.New("asmgen: existing file exceeds read limit")

	// ErrInvalidOption is returned when a configuration option has an invalid value.
	ErrInvalidOption = errors.New("asmgen: invalid option")
)
