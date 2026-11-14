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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testArch struct {
	architecture Architecture
	constraint   string
}

func (a *testArch) Arch() Architecture { return a.architecture }

func (a *testArch) BuildConstraint() string { return a.constraint }

type memWriter struct {
	files map[string][]byte
}

func (w *memWriter) WriteFile(path string, data []byte) error {
	w.files[path] = data
	return nil
}

type errWriter struct {
	err error
}

func (w *errWriter) WriteFile(_ string, _ []byte) error {
	return w.err
}

type failingReader struct {
	err error
}

func (r *failingReader) ReadFile(string) ([]byte, error) {
	return nil, r.err
}

type validatingArch struct {
	validate func(name string, body string) error
	testArch
}

func (a *validatingArch) ValidateHandler(name string, body string) error {
	return a.validate(name, body)
}

func newAMD64Arch() *testArch {
	return &testArch{
		architecture: ArchitectureAMD64,
		constraint:   " && amd64",
	}
}

func newARM64Arch() *testArch {
	return &testArch{
		architecture: ArchitectureARM64,
		constraint:   " && arm64",
	}
}

func newMemWriter() *memWriter {
	return &memWriter{files: make(map[string][]byte)}
}

func newDiskFileSystem(t *testing.T) (*DiskFileSystem, string) {
	t.Helper()
	root := t.TempDir()
	fileSystem, err := NewDiskFileSystem(root)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fileSystem.Close()) })
	return fileSystem, root
}

func minimalGroup(baseName string) FileGroup[*testArch] {
	return FileGroup[*testArch]{
		BaseName:        baseName,
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "noop",
				Comment: "noop does nothing.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
		},
	}
}

func TestGenerateFiles_ProducesCorrectFilename(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	expectedPath := filepath.Join("out", "foo_amd64.s")
	_, exists := writer.files[expectedPath]
	assert.True(t, exists, "expected file %s to be written", expectedPath)
}

func TestGenerateFiles_LicenseHeaderPresent(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.True(t, strings.HasPrefix(content, "// Code generated"), "output should start with licence header")
}

func TestGenerateFiles_BuildConstraintPresent(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "//go:build !safe && amd64")
}

func TestGenerateFiles_IncludesPresent(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")
	group.Includes = []string{"textflag.h", "offsets.h"}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "#include \"textflag.h\"")
	assert.Contains(t, content, "#include \"offsets.h\"")
}

func TestGenerateFiles_HeaderComment(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")
	group.HeaderComment = "Arithmetic handlers for the dispatch loop."

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "// Arithmetic handlers for the dispatch loop.")
}

func TestGenerateFiles_HeaderCommentFunction(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")
	group.HeaderComment = "should be overridden"
	group.HeaderCommentFunction = func(arch Architecture) string {
		return "dynamic header for " + string(arch)
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "// dynamic header for amd64")
	assert.NotContains(t, content, "should be overridden")
}

func TestGenerateFiles_HandlerEmitted(t *testing.T) {
	t.Parallel()

	emitCalled := false
	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "bar",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "myHandler",
				Comment: "myHandler does a thing.",
				Emit: func(e *Emitter, _ *testArch) {
					emitCalled = true
					e.Instruction("MOVQ    AX, BX")
					e.Instruction("RET")
				},
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	assert.True(t, emitCalled, "handler Emit function should have been called")

	content := string(writer.files[filepath.Join("out", "bar_amd64.s")])
	assert.Contains(t, content, "// myHandler does a thing.")
	assert.Contains(t, content, "\tMOVQ    AX, BX\n")
	assert.Contains(t, content, "\tRET\n")
}

func TestGenerateFiles_TextDirectiveFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		handler  HandlerDefinition[*testArch]
	}{
		{
			name: "with flags and frame size",
			handler: HandlerDefinition[*testArch]{
				Name:      "myFunc",
				Comment:   "myFunc comment.",
				Flags:     "NOSPLIT",
				FrameSize: "$0-8",
				Emit:      func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},

			expected: "TEXT \xc2\xb7myFunc(SB), NOSPLIT, $0-8",
		},
		{
			name: "no flags default frame size",
			handler: HandlerDefinition[*testArch]{
				Name:    "simple",
				Comment: "simple comment.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
			expected: "TEXT \xc2\xb7simple(SB), $0",
		},
		{
			name: "no flags custom frame size",
			handler: HandlerDefinition[*testArch]{
				Name:      "withFrame",
				Comment:   "withFrame comment.",
				FrameSize: "$16-24",
				Emit:      func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
			expected: "TEXT \xc2\xb7withFrame(SB), $16-24",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			writer := newMemWriter()
			group := FileGroup[*testArch]{
				BaseName:        "txt",
				OutputDir:       "out",
				BuildConstraint: "!safe",
				Includes:        []string{"textflag.h"},
				Handlers:        []HandlerDefinition[*testArch]{tc.handler},
			}

			err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
			require.NoError(t, err)

			content := string(writer.files[filepath.Join("out", "txt_amd64.s")])
			assert.Contains(t, content, tc.expected)
		})
	}
}

func TestGenerateFiles_PerArchFlagAndFrameSizeOverride(t *testing.T) {
	t.Parallel()

	handler := HandlerDefinition[*testArch]{
		Name:      "myShim",
		Comment:   "myShim is a BL-bearing shim.",
		Flags:     "NOSPLIT|NOFRAME",
		FrameSize: "$0",
		ArchFlags: map[Architecture]string{
			ArchitectureARM64: "NOSPLIT",
		},
		ArchFrameSize: map[Architecture]string{
			ArchitectureARM64: "$32-0",
		},
		Emit: func(e *Emitter, _ *testArch) { e.Instruction("RET") },
	}

	group := FileGroup[*testArch]{
		BaseName:        "ovr",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers:        []HandlerDefinition[*testArch]{handler},
	}

	writer := newMemWriter()
	err := GenerateFiles(writer, []*testArch{newAMD64Arch(), newARM64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	amd64Content := string(writer.files[filepath.Join("out", "ovr_amd64.s")])
	assert.Contains(t, amd64Content, "TEXT \xc2\xb7myShim(SB), NOSPLIT|NOFRAME, $0",
		"amd64 falls back to default Flags/FrameSize")

	arm64Content := string(writer.files[filepath.Join("out", "ovr_arm64.s")])
	assert.Contains(t, arm64Content, "TEXT \xc2\xb7myShim(SB), NOSPLIT, $32-0",
		"arm64 picks up ArchFlags+ArchFrameSize override")
}

func TestGenerateFiles_PerArchOverridePartialFallsBack(t *testing.T) {
	t.Parallel()

	handler := HandlerDefinition[*testArch]{
		Name:      "halfOverride",
		Comment:   "halfOverride exercises the partial override path.",
		Flags:     "NOSPLIT|NOFRAME",
		FrameSize: "$16",
		ArchFlags: map[Architecture]string{
			ArchitectureARM64: "NOSPLIT",
		},
		Emit: func(e *Emitter, _ *testArch) { e.Instruction("RET") },
	}

	group := FileGroup[*testArch]{
		BaseName:        "half",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers:        []HandlerDefinition[*testArch]{handler},
	}

	writer := newMemWriter()
	err := GenerateFiles(writer, []*testArch{newARM64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	arm64Content := string(writer.files[filepath.Join("out", "half_arm64.s")])
	assert.Contains(t, arm64Content, "TEXT \xc2\xb7halfOverride(SB), NOSPLIT, $16",
		"arm64 takes flag override but inherits FrameSize=$16 from default")
}

func TestGenerateFiles_ArchitectureFilter(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "filtered",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:          "arm64Only",
				Comment:       "arm64Only is ARM64 exclusive.",
				Architectures: []Architecture{ArchitectureARM64},
				Emit: func(e *Emitter, _ *testArch) {
					e.Instruction("RET")
				},
			},
			{
				Name:    "universal",
				Comment: "universal runs everywhere.",
				Emit: func(e *Emitter, _ *testArch) {
					e.Instruction("RET")
				},
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "filtered_amd64.s")])
	assert.NotContains(t, content, "arm64Only")
	assert.Contains(t, content, "universal")
}

func TestGenerateFiles_CommentFunction(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "cf",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "dynComment",
				Comment: "should be overridden",
				CommentFunction: func(arch Architecture) string {
					return "generated for " + string(arch)
				},
				Emit: func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "cf_amd64.s")])
	assert.Contains(t, content, "// generated for amd64")
	assert.NotContains(t, content, "should be overridden")
}

func TestGenerateFiles_BlankLinesBetweenHandlers(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "multi",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "first",
				Comment: "first handler.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
			{
				Name:    "second",
				Comment: "second handler.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "multi_amd64.s")])

	assert.Contains(t, content, "\tRET\n\n// second handler.")
}

func TestGenerateFiles_HeaderFileGenerated(t *testing.T) {
	t.Parallel()

	emitCalled := false
	writer := newMemWriter()

	header := HeaderFile{
		Name: "constants.h",
		Dir:  "out",
		Emit: func(archs []ArchitecturePort) string {
			emitCalled = true
			return "// generated header\n#define FOO 42\n"
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, nil, []HeaderFile{header}, nil)
	require.NoError(t, err)

	assert.True(t, emitCalled, "HeaderFile Emit should have been called")

	content := string(writer.files[filepath.Join("out", "constants.h")])
	assert.Contains(t, content, "#define FOO 42")
}

func TestGenerateFiles_WriterErrorPropagates(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("disk full")
	writer := &errWriter{err: expectedError}
	group := minimalGroup("fail")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedError)
}

func TestGenerateAndValidate_NoMismatches(t *testing.T) {
	t.Parallel()

	fileSystem, _ := newDiskFileSystem(t)
	arch := newAMD64Arch()
	group := minimalGroup("check")

	err := GenerateFiles(fileSystem, []*testArch{arch}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	mismatches, err := GenerateAndValidate(fileSystem, []*testArch{arch}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, mismatches)
}

func TestGenerateAndValidate_DetectsDifference(t *testing.T) {
	t.Parallel()

	fileSystem, root := newDiskFileSystem(t)
	arch := newAMD64Arch()
	group := minimalGroup("drift")

	err := GenerateFiles(fileSystem, []*testArch{arch}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)

	relativePath := filepath.Join("out", "drift_amd64.s")
	filePath := filepath.Join(root, relativePath)
	original, err := os.ReadFile(filePath)
	require.NoError(t, err)

	tampered := strings.Replace(string(original), "noop does nothing.", "TAMPERED LINE", 1)
	require.NotEqual(t, string(original), tampered, "replacement should have changed the content")

	err = os.WriteFile(filePath, []byte(tampered), 0o600)
	require.NoError(t, err)

	mismatches, err := GenerateAndValidate(fileSystem, []*testArch{arch}, []FileGroup[*testArch]{group}, nil, nil)
	require.NoError(t, err)
	require.Len(t, mismatches, 1)

	mismatch := mismatches[0]
	assert.Equal(t, relativePath, mismatch.File)
	assert.Positive(t, mismatch.Line, "line number should be positive")
	assert.Contains(t, mismatch.OnDisk, "TAMPERED LINE")
	assert.Contains(t, mismatch.Generated, "noop does nothing.")
}

func TestGenerateAndValidate_ComparesHeadersAndGoFiles(t *testing.T) {
	t.Parallel()

	header := HeaderFile{Name: "offsets.h", Dir: "out", Emit: func([]ArchitecturePort) string { return "#define A 1\n" }}
	goFile := GoFile{Name: "glue.go", Dir: "out", Emit: func([]ArchitecturePort) string { return "package out\n" }}
	architectures := []*testArch{newAMD64Arch()}
	groups := []FileGroup[*testArch]{minimalGroup("glue")}

	testCases := []struct {
		files    map[string]string
		name     string
		expected []Mismatch
	}{
		{
			name:     "missing files are reported",
			files:    nil,
			expected: nil,
		},
		{
			name: "stale header and longer go file are reported",
			files: map[string]string{
				filepath.Join("out", "offsets.h"): "#define A 2\n",
				filepath.Join("out", "glue.go"):   "package out\n// extra\n",
			},
			expected: []Mismatch{
				{File: filepath.Join("out", "offsets.h"), OnDisk: "#define A 2", Generated: "#define A 1", Line: 1},
				{File: filepath.Join("out", "glue.go"), OnDisk: "// extra", Generated: "", Line: 2},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fileSystem := NewMemoryFileSystem()
			require.NoError(t, GenerateFiles(fileSystem, architectures, groups, nil, nil))
			for path, content := range testCase.files {
				require.NoError(t, fileSystem.WriteFile(path, []byte(content)))
			}
			mismatches, err := GenerateAndValidate(fileSystem, architectures, groups, []HeaderFile{header}, []GoFile{goFile})
			require.NoError(t, err)
			if testCase.expected == nil {
				require.Len(t, mismatches, 2)
				for _, mismatch := range mismatches {
					assert.Equal(t, missingFileMarker, mismatch.OnDisk)
					assert.Equal(t, generatedContentMarker, mismatch.Generated)
				}
				return
			}
			assert.Equal(t, testCase.expected, mismatches)
		})
	}
}

func TestGenerateAndValidate_ReadErrorPropagates(t *testing.T) {
	t.Parallel()

	readError := errors.New("disk on fire")
	_, err := GenerateAndValidate(&failingReader{err: readError}, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{minimalGroup("x")}, nil, nil)
	require.ErrorIs(t, err, readError)
}

func TestGenerate_RejectsNilPorts(t *testing.T) {
	t.Parallel()

	architectures := []*testArch{newAMD64Arch()}
	groups := []FileGroup[*testArch]{minimalGroup("x")}
	require.ErrorIs(t, GenerateFiles(nil, architectures, groups, nil, nil), ErrInvalidOption)
	_, err := GenerateAndValidate(nil, architectures, groups, nil, nil)
	require.ErrorIs(t, err, ErrInvalidOption)
}

func TestGenerateFiles_FailureWritesNothing(t *testing.T) {
	t.Parallel()

	failing := minimalGroup("bad")
	failing.Handlers[0].Emit = func(e *Emitter, _ *testArch) { e.Fail("broken") }
	fileSystem := NewMemoryFileSystem()
	header := HeaderFile{Name: "h.h", Dir: "out", Emit: func([]ArchitecturePort) string { return "" }}

	err := GenerateFiles(fileSystem, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{minimalGroup("good"), failing}, []HeaderFile{header}, nil)
	require.ErrorIs(t, err, ErrHandlerEmission)
	assert.Empty(t, fileSystem.Paths(), "no file may be written when any file fails to generate")
}

func TestResolveArchInclude(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		include      string
		architecture Architecture
		expected     string
	}{
		{name: "plain include is untouched", include: "textflag.h", architecture: ArchitectureARM64, expected: "textflag.h"},
		{name: "suffix is rewritten", include: "offsets_amd64.h", architecture: ArchitectureARM64, expected: "offsets_arm64.h"},
		{name: "wasm suffix is rewritten", include: "offsets_wasm.h", architecture: ArchitectureAMD64, expected: "offsets_amd64.h"},
		{name: "directory naming an architecture is untouched", include: "wasmtime/x_amd64.h", architecture: ArchitectureARM64, expected: "wasmtime/x_arm64.h"},
		{name: "architecture inside the stem is untouched", include: "amd64_tables.h", architecture: ArchitectureARM64, expected: "amd64_tables.h"},
		{name: "same architecture is unchanged", include: "offsets_amd64.h", architecture: ArchitectureAMD64, expected: "offsets_amd64.h"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, resolveArchInclude(testCase.include, testCase.architecture))
		})
	}
}

func newWASMArch(validate func(name string, body string) error) *validatingArch {
	return &validatingArch{
		architecture: ArchitectureWASM, constraint: " && wasm",
		validate: validate,
	}
}

func portGroup(emit func(e *Emitter, arch ArchitecturePort)) FileGroup[ArchitecturePort] {
	return FileGroup[ArchitecturePort]{
		BaseName:        "kernel",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h", "offsets_amd64.h"},
		Handlers: []HandlerDefinition[ArchitecturePort]{
			{Name: "kernel", Comment: "kernel is a test kernel.", Emit: emit},
		},
	}
}

func TestGenerateFiles_GroupArchitecturesAndConstraints(t *testing.T) {
	t.Parallel()

	accept := func(string, string) error { return nil }
	testCases := []struct {
		overrides     map[Architecture]string
		expectedFiles map[string]string
		name          string
		architectures []Architecture
	}{
		{
			name:          "unrestricted group writes every architecture",
			architectures: nil,
			overrides:     nil,
			expectedFiles: map[string]string{"kernel_amd64.s": "//go:build !safe && amd64", "kernel_wasm.s": "//go:build !safe && wasm"},
		},
		{
			name:          "restricted group skips other architectures",
			architectures: []Architecture{ArchitectureWASM},
			overrides:     nil,
			expectedFiles: map[string]string{"kernel_wasm.s": "//go:build !safe && wasm"},
		},
		{
			name:          "per architecture constraint override",
			architectures: nil,
			overrides:     map[Architecture]string{ArchitectureWASM: "!safe && js"},
			expectedFiles: map[string]string{"kernel_amd64.s": "//go:build !safe && amd64", "kernel_wasm.s": "//go:build !safe && js && wasm"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			group := portGroup(func(e *Emitter, _ ArchitecturePort) { e.Instruction("RET") })
			group.Architectures = testCase.architectures
			group.ArchBuildConstraint = testCase.overrides
			writer := NewMemoryFileSystem()
			architectures := []ArchitecturePort{newAMD64Arch(), newWASMArch(accept)}
			require.NoError(t, GenerateFiles(writer, architectures, []FileGroup[ArchitecturePort]{group}, nil, nil))
			require.Len(t, writer.Paths(), len(testCase.expectedFiles))
			for name, constraint := range testCase.expectedFiles {
				data, ok := writer.File(filepath.Join("out", name))
				require.True(t, ok, name)
				assert.Contains(t, string(data), constraint+"\n", name)
			}
			wasm, ok := writer.File(filepath.Join("out", "kernel_wasm.s"))
			require.True(t, ok)
			assert.Contains(t, string(wasm), "#include \"offsets_wasm.h\"")
		})
	}
}

func TestGenerateFiles_HandlerFailuresStopGeneration(t *testing.T) {
	t.Parallel()

	rejectCall := func(name string, body string) error {
		if strings.Contains(body, "CALL") {
			return errors.New(name + " calls out")
		}
		return nil
	}
	testCases := []struct {
		wantIs  error
		emit    func(e *Emitter, arch ArchitecturePort)
		name    string
		wantErr string
	}{
		{
			name:    "validator accepts the body",
			emit:    func(e *Emitter, _ ArchitecturePort) { e.Instruction("RET") },
			wantErr: "",
			wantIs:  nil,
		},
		{
			name:    "validator rejects the body",
			emit:    func(e *Emitter, _ ArchitecturePort) { e.Instruction("CALL x(SB)") },
			wantErr: "kernel calls out",
			wantIs:  ErrHandlerValidation,
		},
		{
			name:    "emitter failure is reported",
			emit:    func(e *Emitter, _ ArchitecturePort) { e.Fail("branch to %s is out of scope", "done") },
			wantErr: "branch to done is out of scope",
			wantIs:  ErrHandlerEmission,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			groups := []FileGroup[ArchitecturePort]{portGroup(testCase.emit)}
			architectures := []ArchitecturePort{newWASMArch(rejectCall)}
			err := GenerateFiles(NewMemoryFileSystem(), architectures, groups, nil, nil)
			_, validateErr := GenerateAndValidate(NewMemoryFileSystem(), architectures, groups, nil, nil)
			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.wantErr)
			require.ErrorContains(t, err, "handler kernel: ")
			require.ErrorIs(t, err, testCase.wantIs)
			require.ErrorContains(t, err, filepath.Join("out", "kernel_wasm.s"))
			require.ErrorContains(t, validateErr, testCase.wantErr)
		})
	}
}
