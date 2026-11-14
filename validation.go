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
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// pathSeparators lists the characters a file or symbol name may not contain.
	pathSeparators = `/\`

	// nameWhitespace lists the whitespace characters a handler name may not contain.
	nameWhitespace = " \t\r\n"
)

var (
	// errNilArchitecture reports an adapter that is nil or cannot report its identifier.
	errNilArchitecture = errors.New("adapter is nil or unusable")
)

// outputRegistry records every output path and the definition that claimed it.
type outputRegistry map[string]string

// claim registers an output path for owner.
//
// Takes outputPath (string) which is the path being claimed.
// Takes owner (string) which describes the claimant for the error message.
//
// Returns error which wraps ErrDuplicateOutput when the path is already claimed.
func (r outputRegistry) claim(outputPath string, owner string) error {
	if previous, ok := r[outputPath]; ok {
		return fmt.Errorf("%w: %s is produced by both %s and %s", ErrDuplicateOutput, outputPath, previous, owner)
	}
	r[outputPath] = owner
	return nil
}

// validateGeneration checks every generation input before anything is generated.
//
// Takes architectures ([]A) which lists the target architectures.
// Takes groups ([]FileGroup[A]) which defines the handler file groups.
// Takes headers ([]HeaderFile) which defines the header files.
// Takes goFiles ([]GoFile) which defines the Go source files.
//
// Returns error which joins every problem found, or nil when the inputs are valid.
func validateGeneration[A ArchitecturePort](
	architectures []A,
	groups []FileGroup[A],
	headers []HeaderFile,
	goFiles []GoFile,
) error {
	targets, problems := validateTargetArchitectures(architectures)
	outputs := make(outputRegistry)
	for index, group := range groups {
		problems = append(problems, validateGroup(index, group, targets, outputs)...)
	}
	for _, header := range headers {
		problems = append(problems, validateGeneratedFile("header", header.Dir, header.Name, header.Emit == nil, outputs)...)
	}
	for _, goFile := range goFiles {
		problems = append(problems, validateGeneratedFile("Go file", goFile.Dir, goFile.Name, goFile.Emit == nil, outputs)...)
	}
	return errors.Join(problems...)
}

// validateTargetArchitectures checks that at least one architecture is given and that
// every one is known and listed once.
//
// Takes architectures ([]A) which lists the target architectures.
//
// Returns []Architecture which holds the valid target identifiers in order.
// Returns []error which lists the problems found.
func validateTargetArchitectures[A ArchitecturePort](architectures []A) ([]Architecture, []error) {
	if len(architectures) == 0 {
		return nil, []error{fmt.Errorf("%w: no target architectures", ErrEmptyArchitectures)}
	}
	var problems []error
	targets := make([]Architecture, 0, len(architectures))
	for index, architecture := range architectures {
		identifier, err := architectureIdentifier(architecture)
		if err != nil {
			problems = append(problems, fmt.Errorf("%w: architecture %d: %w", ErrUnknownArchitecture, index, err))
			continue
		}
		if !slices.Contains(knownArchitectures, identifier) || slices.Contains(targets, identifier) {
			problems = append(problems, fmt.Errorf("%w: %q", ErrUnknownArchitecture, identifier))
			continue
		}
		targets = append(targets, identifier)
	}
	return targets, problems
}

// architectureIdentifier asks an adapter for its identifier, turning a nil adapter or a
// panicking Arch method into an error instead of crashing the generator.
//
// Takes architecture (A) which is the adapter to query.
//
// Returns Architecture which is the adapter's identifier.
// Returns error when the adapter is nil or its Arch method panics.
func architectureIdentifier[A ArchitecturePort](architecture A) (identifier Architecture, err error) {
	if any(architecture) == nil {
		return "", errNilArchitecture
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: Arch panicked: %v", errNilArchitecture, recovered)
		}
	}()
	return architecture.Arch(), nil
}

// validateGroup checks one file group, its handlers and the output paths it produces.
//
// Takes index (int) which is the group's position, used when its name is invalid.
// Takes group (FileGroup[A]) which is the group to check.
// Takes targets ([]Architecture) which lists the valid target architectures.
// Takes outputs (outputRegistry) which records the output paths claimed so far.
//
// Returns []error which lists the problems found.
func validateGroup[A ArchitecturePort](index int, group FileGroup[A], targets []Architecture, outputs outputRegistry) []error {
	owner := fmt.Sprintf("group %q", group.BaseName)
	if group.BaseName == "" {
		owner = fmt.Sprintf("group %d", index)
	}
	var problems []error
	if err := validateName(group.BaseName); err != nil {
		problems = append(problems, fmt.Errorf("%s: base name: %w", owner, err))
	}
	if err := validateDirectory(group.OutputDir); err != nil {
		problems = append(problems, fmt.Errorf("%s: output directory: %w", owner, err))
	}
	problems = append(problems, validateRestriction(owner, group.Architectures)...)
	problems = append(problems, validateOverrideKeys(owner+" ArchBuildConstraint", group.ArchBuildConstraint)...)
	for _, handler := range group.Handlers {
		problems = append(problems, validateHandler(owner, handler)...)
	}
	for _, target := range targets {
		if !targetsArchitecture(group.Architectures, target) {
			continue
		}
		problems = append(problems, duplicateHandlers(owner, group.Handlers, target)...)
		if len(problems) == 0 {
			if err := outputs.claim(assemblyPath(group, target), owner); err != nil {
				problems = append(problems, err)
			}
		}
	}
	return problems
}

// validateHandler checks one handler definition.
//
// Takes owner (string) which describes the enclosing group for error messages.
// Takes handler (HandlerDefinition[A]) which is the handler to check.
//
// Returns []error which lists the problems found.
func validateHandler[A ArchitecturePort](owner string, handler HandlerDefinition[A]) []error {
	handlerOwner := fmt.Sprintf("%s handler %q", owner, handler.Name)
	var problems []error
	if handler.Name == "" || strings.ContainsAny(handler.Name, pathSeparators+nameWhitespace) {
		problems = append(problems, fmt.Errorf("%s: %w: handler name must be a non-empty symbol", handlerOwner, ErrInvalidName))
	}
	if handler.Emit == nil {
		problems = append(problems, fmt.Errorf("%s: %w", handlerOwner, ErrMissingEmit))
	}
	problems = append(problems, validateRestriction(handlerOwner, handler.Architectures)...)
	problems = append(problems, validateOverrideKeys(handlerOwner+" ArchFlags", handler.ArchFlags)...)
	problems = append(problems, validateOverrideKeys(handlerOwner+" ArchFrameSize", handler.ArchFrameSize)...)
	return problems
}

// duplicateHandlers reports handler names emitted more than once into the same file for
// one architecture.
//
// Takes owner (string) which describes the enclosing group for error messages.
// Takes handlers ([]HandlerDefinition[A]) which are the group's handlers.
// Takes target (Architecture) which is the architecture whose file is checked.
//
// Returns []error which lists one problem per duplicated name.
func duplicateHandlers[A ArchitecturePort](owner string, handlers []HandlerDefinition[A], target Architecture) []error {
	seen := make(map[string]struct{}, len(handlers))
	var problems []error
	for _, handler := range handlers {
		if !targetsArchitecture(handler.Architectures, target) {
			continue
		}
		if _, ok := seen[handler.Name]; ok {
			problems = append(problems, fmt.Errorf("%s: %w: %q on %s", owner, ErrDuplicateHandler, handler.Name, target))
			continue
		}
		seen[handler.Name] = struct{}{}
	}
	return problems
}

// validateRestriction checks that an architecture restriction is valid.
//
// Takes owner (string) which describes the restricted definition for error messages.
// Takes restriction ([]Architecture) which is the Architectures field.
//
// Returns []error which lists the problems found.
func validateRestriction(owner string, restriction []Architecture) []error {
	if restriction == nil {
		return nil
	}
	if len(restriction) == 0 {
		return []error{fmt.Errorf("%s: %w: use nil to target every architecture", owner, ErrEmptyArchitectures)}
	}
	var problems []error
	for _, architecture := range restriction {
		if !slices.Contains(knownArchitectures, architecture) {
			problems = append(problems, fmt.Errorf("%s: %w: %q", owner, ErrUnknownArchitecture, architecture))
		}
	}
	return problems
}

// validateOverrideKeys checks that every key of a per-architecture override map is a
// known architecture.
//
// Takes owner (string) which describes the override for error messages.
// Takes overrides (map[Architecture]string) which is the override map.
//
// Returns []error which lists one problem per unknown key, in sorted order.
func validateOverrideKeys(owner string, overrides map[Architecture]string) []error {
	var problems []error
	for _, architecture := range slices.Sorted(maps.Keys(overrides)) {
		if !slices.Contains(knownArchitectures, architecture) {
			problems = append(problems, fmt.Errorf("%s: %w: %q", owner, ErrUnknownArchitecture, architecture))
		}
	}
	return problems
}

// validateGeneratedFile checks a header or Go file definition and claims its output path.
//
// Takes kind (string) which names the definition kind for error messages.
// Takes directory (string) which is the output directory.
// Takes name (string) which is the output file name.
// Takes missingEmit (bool) which is true when the definition has no Emit function.
// Takes outputs (outputRegistry) which records the output paths claimed so far.
//
// Returns []error which lists the problems found.
func validateGeneratedFile(kind string, directory string, name string, missingEmit bool, outputs outputRegistry) []error {
	owner := fmt.Sprintf("%s %q", kind, name)
	var problems []error
	if err := validateName(name); err != nil {
		problems = append(problems, fmt.Errorf("%s: name: %w", owner, err))
	}
	if err := validateDirectory(directory); err != nil {
		problems = append(problems, fmt.Errorf("%s: directory: %w", owner, err))
	}
	if missingEmit {
		problems = append(problems, fmt.Errorf("%s: %w", owner, ErrMissingEmit))
	}
	if len(problems) == 0 {
		if err := outputs.claim(filepath.Join(directory, name), owner); err != nil {
			problems = append(problems, err)
		}
	}
	return problems
}

// validateName checks that a file name or stem is non-empty, contains no separator and
// is not "." or "..".
//
// Takes name (string) which is the name to check.
//
// Returns error which wraps ErrInvalidName when the name is unusable.
func validateName(name string) error {
	if name == "" || strings.ContainsAny(name, pathSeparators) || !filepath.IsLocal(name) || name == "." {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	return nil
}

// validateDirectory checks that an output directory stays inside the project root. An
// empty directory or "." means the root itself.
//
// Takes directory (string) which is the directory to check.
//
// Returns error which wraps ErrInvalidPath when the directory is absolute or escapes the
// root.
func validateDirectory(directory string) error {
	if directory == "" || filepath.IsLocal(directory) {
		return nil
	}
	return fmt.Errorf("%w: %q must be relative and inside the project root", ErrInvalidPath, directory)
}
