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
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// missingFileMarker is the OnDisk text of a mismatch for a file that does not exist.
	missingFileMarker = "<file does not exist>"

	// generatedContentMarker is the Generated text of a mismatch for a missing file.
	generatedContentMarker = "(generated content)"
)

// Mismatch describes the first line at which a generated file differs from the file on
// disk.
type Mismatch struct {
	// File is the path to the file with the mismatch.
	File string

	// OnDisk is the line currently on disk at the mismatch point.
	OnDisk string

	// Generated is the line the generator produces at the mismatch point.
	Generated string

	// Line is the one-based line number of the first difference.
	Line int
}

// generatedFile is one fully generated output file waiting to be written or compared.
type generatedFile struct {
	// path is the output path relative to the project root.
	path string

	// data is the complete file content.
	data []byte
}

// GenerateFiles generates all .s, .h and (optionally) .go files for the given file groups
// and target architectures, then writes them via the provided writer.
//
// Every file is generated before any is written, so a failing handler leaves the files on
// disk untouched.
//
// Takes writer (FileSystemWriterPort) which writes the generated files.
// Takes architectures ([]A) which lists the target architectures.
// Takes groups ([]FileGroup[A]) which defines the handler file groups.
// Takes headers ([]HeaderFile) which defines the header files to generate.
// Takes goFiles ([]GoFile) which defines the Go source files to generate alongside the
// .s/.h outputs. Pass nil to skip Go-file generation.
// Takes options (...Option) which configure the generated-file header. Pass the same
// options to GenerateAndValidate so validation compares against byte-identical output.
//
// Returns error when the inputs are invalid, generation fails or a file write fails.
func GenerateFiles[A ArchitecturePort](
	writer FileSystemWriterPort,
	architectures []A,
	groups []FileGroup[A],
	headers []HeaderFile,
	goFiles []GoFile,
	options ...Option,
) error {
	if writer == nil {
		return fmt.Errorf("%w: writer is nil", ErrInvalidOption)
	}
	files, err := prepareFiles(architectures, groups, headers, goFiles, options)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := writer.WriteFile(file.path, file.data); err != nil {
			return fmt.Errorf("writing %s: %w", file.path, err)
		}
	}
	return nil
}

// GenerateAndValidate generates all files into memory and compares each against the
// existing file read through reader.
//
// Takes reader (FileSystemReaderPort) which reads the existing files.
// Takes architectures ([]A) which lists the target architectures.
// Takes groups ([]FileGroup[A]) which defines the handler file groups.
// Takes headers ([]HeaderFile) which defines the header files to validate.
// Takes goFiles ([]GoFile) which defines the generated Go source files to validate. Pass
// nil to skip Go-file validation.
// Takes options (...Option) which configure the generated-file header. Pass the same
// options used with GenerateFiles so validation compares against byte-identical output.
//
// Returns []Mismatch which lists the first difference in every differing file.
// Returns error when the inputs are invalid, generation fails or a file cannot be read.
func GenerateAndValidate[A ArchitecturePort](
	reader FileSystemReaderPort,
	architectures []A,
	groups []FileGroup[A],
	headers []HeaderFile,
	goFiles []GoFile,
	options ...Option,
) ([]Mismatch, error) {
	if reader == nil {
		return nil, fmt.Errorf("%w: reader is nil", ErrInvalidOption)
	}
	files, err := prepareFiles(architectures, groups, headers, goFiles, options)
	if err != nil {
		return nil, err
	}
	var mismatches []Mismatch
	for _, file := range files {
		mismatch, err := compareWithExisting(reader, file.path, file.data)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file.path, err)
		}
		if mismatch != nil {
			mismatches = append(mismatches, *mismatch)
		}
	}
	return mismatches, nil
}

// prepareFiles validates the generation inputs and generates every output file.
//
// Takes architectures ([]A) which lists the target architectures.
// Takes groups ([]FileGroup[A]) which defines the handler file groups.
// Takes headers ([]HeaderFile) which defines the header files.
// Takes goFiles ([]GoFile) which defines the Go source files.
// Takes options ([]Option) which configure the generated-file header.
//
// Returns []generatedFile which holds every output in generation order.
// Returns error when the inputs are invalid or any file fails to generate.
func prepareFiles[A ArchitecturePort](
	architectures []A,
	groups []FileGroup[A],
	headers []HeaderFile,
	goFiles []GoFile,
	options []Option,
) ([]generatedFile, error) {
	if err := validateGeneration(architectures, groups, headers, goFiles); err != nil {
		return nil, err
	}
	config := newGenerateConfig(options)
	files, err := generateAssemblyFiles(architectures, groups, config)
	if err != nil {
		return nil, err
	}
	architecturePorts := toArchitecturePorts(architectures)
	for _, header := range headers {
		files = append(files, generatedFile{
			path: filepath.Join(header.Dir, header.Name),
			data: []byte(header.Emit(architecturePorts)),
		})
	}
	for _, goFile := range goFiles {
		files = append(files, generatedFile{
			path: filepath.Join(goFile.Dir, goFile.Name),
			data: []byte(goFile.Emit(architecturePorts)),
		})
	}
	return files, nil
}

// generateAssemblyFiles generates the .s file of every group for every architecture the
// group targets, reporting every failing file rather than only the first.
//
// Takes architectures ([]A) which lists the target architectures.
// Takes groups ([]FileGroup[A]) which defines the handler file groups.
// Takes config (generateConfig) which is the resolved header configuration.
//
// Returns []generatedFile which holds the generated assembly files.
// Returns error which joins the failure of every file that could not be generated.
func generateAssemblyFiles[A ArchitecturePort](
	architectures []A,
	groups []FileGroup[A],
	config generateConfig,
) ([]generatedFile, error) {
	var files []generatedFile
	var failures []error
	for _, group := range groups {
		for _, architecture := range architectures {
			if !targetsArchitecture(group.Architectures, architecture.Arch()) {
				continue
			}
			outputPath := assemblyPath(group, architecture.Arch())
			data, err := generateFileGroup(group, architecture, config)
			if err != nil {
				failures = append(failures, fmt.Errorf("generating %s: %w", outputPath, err))
				continue
			}
			files = append(files, generatedFile{path: outputPath, data: data})
		}
	}
	if len(failures) > 0 {
		return nil, errors.Join(failures...)
	}
	return files, nil
}

// toArchitecturePorts widens a typed architecture list to the core port type that header
// and Go file emitters receive.
//
// Takes architectures ([]A) which lists the target architectures.
//
// Returns []ArchitecturePort which holds the same adapters.
func toArchitecturePorts[A ArchitecturePort](architectures []A) []ArchitecturePort {
	ports := make([]ArchitecturePort, len(architectures))
	for index, architecture := range architectures {
		ports[index] = architecture
	}
	return ports
}

// generateFileGroup produces the complete .s file content for a single file group and
// architecture.
//
// Takes group (FileGroup[A]) which defines the handler file group to generate.
// Takes architecture (A) which specifies the target architecture.
// Takes config (generateConfig) which supplies the generated-code marker and optional
// header.
//
// Returns []byte which contains the assembled file content.
// Returns error when a handler records a failure or fails the adapter's validation.
func generateFileGroup[A ArchitecturePort](group FileGroup[A], architecture A, config generateConfig) ([]byte, error) {
	emitter := NewEmitter()
	emitFilePreamble(emitter, group, architecture, config)

	headerComment := group.HeaderComment
	if group.HeaderCommentFunction != nil {
		headerComment = group.HeaderCommentFunction(architecture.Arch())
	}
	if headerComment != "" {
		emitter.Comment(headerComment)
	}

	var failures []error
	emitted := false
	for _, handler := range group.Handlers {
		if !targetsArchitecture(handler.Architectures, architecture.Arch()) {
			continue
		}
		if emitted || headerComment != "" {
			emitter.Blank()
		}
		if err := generateHandler(emitter, handler, architecture); err != nil {
			failures = append(failures, fmt.Errorf("handler %s: %w", handler.Name, err))
		}
		emitted = true
	}

	if len(failures) > 0 {
		return nil, errors.Join(failures...)
	}
	return emitter.Bytes(), nil
}

// emitFilePreamble writes the generated-code marker, optional file header, build
// constraint and include directives that open every generated .s file.
//
// Takes emitter (*Emitter) which receives the preamble.
// Takes group (FileGroup[A]) which supplies the constraint and includes.
// Takes architecture (A) which specifies the target architecture.
// Takes config (generateConfig) which supplies the marker tool name and file header.
func emitFilePreamble[A ArchitecturePort](emitter *Emitter, group FileGroup[A], architecture A, config generateConfig) {
	emitter.Line("// Code generated by " + config.generatedByTool + "; DO NOT EDIT.")
	if config.fileHeader != "" {
		emitter.Blank()
		emitter.Raw(config.fileHeader + "\n")
	}
	emitter.Blank()

	constraint := group.BuildConstraint
	if override, ok := group.ArchBuildConstraint[architecture.Arch()]; ok {
		constraint = override
	}
	emitter.Line("//go:build " + constraint + architecture.BuildConstraint())
	emitter.Blank()

	for _, include := range group.Includes {
		emitter.Line("#include \"" + resolveArchInclude(include, architecture.Arch()) + "\"")
	}
	emitter.Blank()
}

// generateHandler emits a single TEXT block for the given handler definition.
//
// Takes emitter (*Emitter) which receives the generated assembly.
// Takes handler (HandlerDefinition[A]) which defines the handler to emit.
// Takes architecture (A) which specifies the target architecture.
//
// Returns error when the handler records a failure on the emitter or, for an adapter
// that implements HandlerValidator, when its body fails validation.
func generateHandler[A ArchitecturePort](emitter *Emitter, handler HandlerDefinition[A], architecture A) error {
	comment := handler.Comment
	if handler.CommentFunction != nil {
		comment = handler.CommentFunction(architecture.Arch())
	}
	emitter.Comment(comment)

	frameSize := handler.FrameSize
	if override, ok := handler.ArchFrameSize[architecture.Arch()]; ok {
		frameSize = override
	}
	if frameSize == "" {
		frameSize = "$0"
	}

	flagsValue := handler.Flags
	if override, ok := handler.ArchFlags[architecture.Arch()]; ok {
		flagsValue = override
	}
	flags := ", "
	if flagsValue != "" {
		flags = ", " + flagsValue + ", "
	}

	emitter.Line("TEXT \xc2\xb7" + handler.Name + "(SB)" + flags + frameSize)

	bodyStart := emitter.Len()
	failuresBefore := len(emitter.failures)
	handler.Emit(emitter, architecture)
	if len(emitter.failures) > failuresBefore {
		return errors.Join(emitter.failures[failuresBefore:]...)
	}
	validator, ok := any(architecture).(HandlerValidator)
	if !ok {
		return nil
	}
	err := validator.ValidateHandler(handler.Name, string(emitter.Bytes()[bodyStart:]))
	if err != nil && !errors.Is(err, ErrHandlerValidation) {
		return fmt.Errorf("%w: %w", ErrHandlerValidation, err)
	}
	return err
}

// targetsArchitecture reports whether an Architectures restriction admits the given
// architecture.
//
// Takes restriction ([]Architecture) which is a group's or handler's Architectures field.
// Takes architecture (Architecture) which is the architecture to match.
//
// Returns bool which is true when the architecture is admitted.
func targetsArchitecture(restriction []Architecture, architecture Architecture) bool {
	if restriction == nil {
		return true
	}
	return slices.Contains(restriction, architecture)
}

// assemblyPath returns the output path of a group's .s file for one architecture.
//
// Takes group (FileGroup[A]) which supplies the base name and output directory.
// Takes architecture (Architecture) which supplies the file name suffix.
//
// Returns string which is the joined output path.
func assemblyPath[A ArchitecturePort](group FileGroup[A], architecture Architecture) string {
	return filepath.Join(group.OutputDir, group.BaseName+"_"+string(architecture)+".s")
}

// resolveArchInclude rewrites a file name's architecture suffix for the target.
//
// Only a "_<architecture>" suffix directly before the extension is rewritten.
//
// Takes include (string) which is the include path.
// Takes architecture (Architecture) which is the target architecture.
//
// Returns string which is the include path for the target architecture.
func resolveArchInclude(include string, architecture Architecture) string {
	directory, file := path.Split(include)
	extension := path.Ext(file)
	stem := strings.TrimSuffix(file, extension)
	for _, known := range knownArchitectures {
		if trimmed, ok := strings.CutSuffix(stem, "_"+string(known)); ok {
			return directory + trimmed + "_" + string(architecture) + extension
		}
	}
	return include
}

// compareWithExisting reads the file at filePath and compares it line by line with the
// generated content.
//
// Takes reader (FileSystemReaderPort) which reads the existing file.
// Takes filePath (string) which is the file path to read and compare against.
// Takes generated ([]byte) which contains the newly generated content.
//
// Returns *Mismatch which describes the first difference found, or nil if the files
// match.
// Returns error when the file exists but cannot be read.
func compareWithExisting(reader FileSystemReaderPort, filePath string, generated []byte) (*Mismatch, error) {
	existing, err := reader.ReadFile(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return &Mismatch{File: filePath, Line: 1, OnDisk: missingFileMarker, Generated: generatedContentMarker}, nil
	}
	if err != nil {
		return nil, err
	}

	existingLines := strings.Split(string(existing), "\n")
	generatedLines := strings.Split(string(generated), "\n")
	for index := range max(len(existingLines), len(generatedLines)) {
		existingLine := lineAt(existingLines, index)
		generatedLine := lineAt(generatedLines, index)
		if existingLine != generatedLine {
			return &Mismatch{File: filePath, Line: index + 1, OnDisk: existingLine, Generated: generatedLine}, nil
		}
	}
	return nil, nil
}

// lineAt returns the line at index, or an empty string past the end.
//
// Takes lines ([]string) which are the split lines.
// Takes index (int) which is the zero-based line index.
//
// Returns string which is the line or empty.
func lineAt(lines []string, index int) string {
	if index < len(lines) {
		return lines[index]
	}
	return ""
}
