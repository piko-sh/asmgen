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

package asmgen_arch_wasm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/asmgen"
	"piko.sh/asmgen/asmgen_arch_wasm"
	"piko.sh/asmgen/asmwasm"
)

func newSumSignature() (asmgen.Signature, error) {
	return asmgen.NewSignature(
		[]asmgen.Parameter{{Name: "values", Kind: asmgen.ParameterSlice}},
		[]asmgen.Parameter{{Name: "ret", Kind: asmgen.ParameterWord}},
	)
}

func emitSumInt64(e *asmgen.Emitter, _ asmgen.ArchitecturePort) {
	sumSignature, err := newSumSignature()
	if err != nil {
		e.Fail("sum signature: %w", err)
		return
	}
	body := asmgen_arch_wasm.NewBody(e)
	body.LoadArgument(sumSignature, "values_base")
	body.Op(asmwasm.Set, "R0")
	body.LoadArgument(sumSignature, "values_len")
	body.Op(asmwasm.Set, "R1")
	body.Op(asmwasm.I64Const, "$0")
	body.Op(asmwasm.Set, "R2")
	body.Block("done", func() {
		body.Loop("next", func() {
			body.Op(asmwasm.Get, "R1")
			body.Op(asmwasm.I64Eqz)
			body.BrIf("done")
			body.Op(asmwasm.Get, "R2")
			body.Op(asmwasm.Get, "R0")
			body.Op(asmwasm.I32WrapI64)
			body.Op(asmwasm.I64Load, "$0")
			body.Op(asmwasm.I64Add)
			body.Op(asmwasm.Set, "R2")
			body.Op(asmwasm.Get, "R0")
			body.Op(asmwasm.I64Const, "$8")
			body.Op(asmwasm.I64Add)
			body.Op(asmwasm.Set, "R0")
			body.Op(asmwasm.Get, "R1")
			body.Op(asmwasm.I64Const, "$1")
			body.Op(asmwasm.I64Sub)
			body.Op(asmwasm.Set, "R1")
			body.Br("next")
		})
	})
	body.StoreResult(sumSignature, "ret", func() { body.Op(asmwasm.Get, "R2") })
	body.Return()
}

func TestGeneratedKernelMatchesTheGoldenFile(t *testing.T) {
	t.Parallel()

	sumSignature, err := newSumSignature()
	require.NoError(t, err)
	sumFrame, err := sumSignature.FrameSize(0)
	require.NoError(t, err)
	testCases := []struct {
		name   string
		golden string
		group  asmgen.FileGroup[asmgen.ArchitecturePort]
	}{
		{
			name:   "sum of an int64 slice",
			golden: "sum_int64_wasm.s",
			group: asmgen.FileGroup[asmgen.ArchitecturePort]{
				BaseName:              "sum_int64",
				OutputDir:             "out",
				BuildConstraint:       "!safe",
				ArchBuildConstraint:   map[asmgen.Architecture]string{asmgen.ArchitectureWASM: "!safe && js"},
				HeaderComment:         "",
				HeaderCommentFunction: nil,
				Includes:              []string{"textflag.h"},
				Architectures:         []asmgen.Architecture{asmgen.ArchitectureWASM},
				Handlers: []asmgen.HandlerDefinition[asmgen.ArchitecturePort]{{
					CommentFunction: nil,
					Emit:            emitSumInt64,
					Name:            "sumInt64",
					Comment:         "sumInt64 returns the sum of values.",
					FrameSize:       sumFrame,
					Flags:           "NOSPLIT",
					ArchFlags:       nil,
					ArchFrameSize:   nil,
					Architectures:   nil,
				}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			writer := asmgen.NewMemoryFileSystem()
			architectures := []asmgen.ArchitecturePort{asmgen_arch_wasm.New()}
			groups := []asmgen.FileGroup[asmgen.ArchitecturePort]{testCase.group}
			require.NoError(t, asmgen.GenerateFiles(writer, architectures, groups, nil, nil))
			generated, ok := writer.File(filepath.Join("out", testCase.golden))
			require.True(t, ok)
			golden, err := os.ReadFile(filepath.Join("testdata", testCase.golden))
			require.NoError(t, err)
			assert.Equal(t, string(golden), string(generated))
		})
	}
}
