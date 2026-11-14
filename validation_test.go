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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateGenerationRejectsMalformedInputs(t *testing.T) {
	t.Parallel()

	emitHeader := func([]ArchitecturePort) string { return "" }
	testCases := []struct {
		wantIs        error
		mutate        func(group *FileGroup[*testArch])
		name          string
		architectures []*testArch
		headers       []HeaderFile
		goFiles       []GoFile
	}{
		{name: "no architectures", architectures: []*testArch{}, wantIs: ErrEmptyArchitectures},
		{name: "nil architecture", architectures: []*testArch{nil}, wantIs: ErrUnknownArchitecture},
		{
			name:          "unknown architecture",
			architectures: []*testArch{{architecture: "riscv64", constraint: ""}},
			wantIs:        ErrUnknownArchitecture,
		},
		{name: "duplicate architecture", architectures: []*testArch{newAMD64Arch(), newAMD64Arch()}, wantIs: ErrUnknownArchitecture},
		{name: "empty base name", mutate: func(group *FileGroup[*testArch]) { group.BaseName = "" }, wantIs: ErrInvalidName},
		{name: "base name with separator", mutate: func(group *FileGroup[*testArch]) { group.BaseName = "a/b" }, wantIs: ErrInvalidName},
		{name: "base name climbing out", mutate: func(group *FileGroup[*testArch]) { group.BaseName = ".." }, wantIs: ErrInvalidName},
		{name: "absolute output directory", mutate: func(group *FileGroup[*testArch]) { group.OutputDir = "/etc" }, wantIs: ErrInvalidPath},
		{name: "escaping output directory", mutate: func(group *FileGroup[*testArch]) { group.OutputDir = "../x" }, wantIs: ErrInvalidPath},
		{
			name:   "empty group restriction",
			mutate: func(group *FileGroup[*testArch]) { group.Architectures = []Architecture{} },
			wantIs: ErrEmptyArchitectures,
		},
		{
			name:   "unknown group restriction",
			mutate: func(group *FileGroup[*testArch]) { group.Architectures = []Architecture{"mips"} },
			wantIs: ErrUnknownArchitecture,
		},
		{
			name:   "unknown constraint override",
			mutate: func(group *FileGroup[*testArch]) { group.ArchBuildConstraint = map[Architecture]string{"mips": "x"} },
			wantIs: ErrUnknownArchitecture,
		},
		{name: "empty handler name", mutate: func(group *FileGroup[*testArch]) { group.Handlers[0].Name = "" }, wantIs: ErrInvalidName},
		{name: "handler name with space", mutate: func(group *FileGroup[*testArch]) { group.Handlers[0].Name = "a b" }, wantIs: ErrInvalidName},
		{name: "missing handler emit", mutate: func(group *FileGroup[*testArch]) { group.Handlers[0].Emit = nil }, wantIs: ErrMissingEmit},
		{
			name:   "empty handler restriction",
			mutate: func(group *FileGroup[*testArch]) { group.Handlers[0].Architectures = []Architecture{} },
			wantIs: ErrEmptyArchitectures,
		},
		{
			name:   "unknown flags override",
			mutate: func(group *FileGroup[*testArch]) { group.Handlers[0].ArchFlags = map[Architecture]string{"mips": ""} },
			wantIs: ErrUnknownArchitecture,
		},
		{
			name: "unknown frame override",
			mutate: func(group *FileGroup[*testArch]) {
				group.Handlers[0].ArchFrameSize = map[Architecture]string{"mips": ""}
			},
			wantIs: ErrUnknownArchitecture,
		},
		{
			name:   "duplicate handler",
			mutate: func(group *FileGroup[*testArch]) { group.Handlers = append(group.Handlers, group.Handlers[0]) },
			wantIs: ErrDuplicateHandler,
		},
		{
			name:    "header claims an assembly path",
			headers: []HeaderFile{{Name: "foo_amd64.s", Dir: "out", Emit: emitHeader}},
			wantIs:  ErrDuplicateOutput,
		},
		{name: "header without emit", headers: []HeaderFile{{Name: "a.h", Dir: "out", Emit: nil}}, wantIs: ErrMissingEmit},
		{name: "header with bad name", headers: []HeaderFile{{Name: "", Dir: "out", Emit: emitHeader}}, wantIs: ErrInvalidName},
		{name: "go file escaping", goFiles: []GoFile{{Name: "a.go", Dir: "../up", Emit: emitHeader}}, wantIs: ErrInvalidPath},
		{
			name: "two go files share a path",
			goFiles: []GoFile{
				{Name: "a.go", Dir: "out", Emit: emitHeader},
				{Name: "a.go", Dir: "out", Emit: emitHeader},
			},
			wantIs: ErrDuplicateOutput,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			group := minimalGroup("foo")
			if testCase.mutate != nil {
				testCase.mutate(&group)
			}
			architectures := testCase.architectures
			if architectures == nil {
				architectures = []*testArch{newAMD64Arch()}
			}
			err := validateGeneration(architectures, []FileGroup[*testArch]{group}, testCase.headers, testCase.goFiles)
			require.ErrorIs(t, err, testCase.wantIs)
			require.ErrorIs(t, GenerateFiles(NewMemoryFileSystem(), architectures, []FileGroup[*testArch]{group}, testCase.headers, testCase.goFiles), testCase.wantIs)
		})
	}
}

func TestValidateGenerationRejectsGroupsSharingAnOutput(t *testing.T) {
	t.Parallel()

	groups := []FileGroup[*testArch]{minimalGroup("same"), minimalGroup("same")}
	err := validateGeneration([]*testArch{newAMD64Arch()}, groups, nil, nil)
	require.ErrorIs(t, err, ErrDuplicateOutput)
}

func TestValidateGenerationAcceptsValidInputs(t *testing.T) {
	t.Parallel()

	first := minimalGroup("first")
	first.OutputDir = ""
	second := minimalGroup("second")
	second.OutputDir = "."
	second.Handlers = append(second.Handlers, HandlerDefinition[*testArch]{
		Name:          "noop",
		Emit:          func(e *Emitter, _ *testArch) { e.Instruction("RET") },
		Architectures: []Architecture{ArchitectureARM64},
	})
	second.Handlers[0].Architectures = []Architecture{ArchitectureAMD64}
	headers := []HeaderFile{{Name: "a.h", Dir: "", Emit: func([]ArchitecturePort) string { return "" }}}
	err := validateGeneration([]*testArch{newAMD64Arch(), newARM64Arch()}, []FileGroup[*testArch]{first, second}, headers, nil)
	require.NoError(t, err, "a handler name may repeat when each copy targets a different architecture")
}
