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

package asmgen_arch_wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/asmgen"
	"piko.sh/asmgen/asmwasm"
)

func TestBodyEmitsBalancedStructuredControl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		emit     func(body *Body)
		name     string
		expected string
	}{
		{
			name: "branches resolve to relative depths",
			emit: func(body *Body) {
				body.Block("done", func() {
					body.Loop("next", func() {
						body.BrIf("done")
						body.Br("next")
					})
				})
			},
			expected: "\tBlock\n\t\tLoop\n\t\t\tBrIf $1\n\t\t\tBr $0\n\t\tEnd\n\tEnd\n",
		},
		{
			name: "if with else arm",
			emit: func(body *Body) {
				body.IfElse("choice", func() { body.Op(asmwasm.Nop) }, func() { body.Br("choice") })
			},
			expected: "\tIf\n\t\tNop\n\tElse\n\t\tBr $0\n\tEnd\n",
		},
		{
			name: "if without else",
			emit: func(body *Body) {
				body.If("", func() { body.Op(asmwasm.I64Const, "$1") })
			},
			expected: "\tIf\n\t\tI64Const $1\n\tEnd\n",
		},
		{
			name: "branch table lowers to compare chains",
			emit: func(body *Body) {
				body.Block("other", func() {
					body.Block("zero", func() { body.BrTable("R3", "other", "zero") })
				})
			},
			expected: "\tBlock\n\t\tBlock\n\t\t\tGet R3\n\t\t\tI64Const $0\n\t\t\tI64Eq\n\t\t\tBrIf $0\n\t\t\tBr $1\n\t\tEnd\n\tEnd\n",
		},
		{
			name:     "call and return at the top level",
			emit:     func(body *Body) { body.Call("runtime·memmove(SB)"); body.Return() },
			expected: "\tCALL runtime·memmove(SB)\n\tRET\n",
		},
		{
			name:     "operands are comma separated",
			emit:     func(body *Body) { body.Op(asmwasm.F64x2ExtractLane, "$0", "$1") },
			expected: "\tF64x2ExtractLane $0, $1\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			emitter := asmgen.NewEmitter()
			testCase.emit(NewBody(emitter))
			require.NoError(t, emitter.Err())
			assert.Equal(t, testCase.expected, emitter.String())
			require.NoError(t, validateBody(emitter.String()))
		})
	}
}

func TestBodyRecordsMistakesOnTheEmitter(t *testing.T) {
	t.Parallel()

	signature, err := asmgen.NewSignature([]asmgen.Parameter{{Name: "a", Kind: asmgen.ParameterSlice}}, nil)
	require.NoError(t, err)
	testCases := []struct {
		emit    func(body *Body)
		name    string
		wantErr string
	}{
		{
			name:    "branch to a label that is not open",
			emit:    func(body *Body) { body.Block("done", func() { body.Br("missing") }) },
			wantErr: `Br to "missing": no enclosing Block, Loop or If has that label (open: Block done)`,
		},
		{
			name:    "branch at the top level",
			emit:    func(body *Body) { body.BrIf("done") },
			wantErr: "open: the top level",
		},
		{
			name:    "an unlabelled if is not a branch target",
			emit:    func(body *Body) { body.If("", func() { body.Br("") }) },
			wantErr: `Br to ""`,
		},
		{
			name:    "call inside a loop",
			emit:    func(body *Body) { body.Loop("next", func() { body.Call("f(SB)") }) },
			wantErr: "CALL f(SB) inside Loop next",
		},
		{
			name:    "unknown argument slot",
			emit:    func(body *Body) { body.LoadArgument(signature, "b_base") },
			wantErr: `loading argument: asmgen: unknown signature slot: "b_base"`,
		},
		{
			name:    "unknown result slot",
			emit:    func(body *Body) { body.StoreResult(signature, "ret", func() {}) },
			wantErr: `storing result: asmgen: unknown signature slot: "ret"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			emitter := asmgen.NewEmitter()
			testCase.emit(NewBody(emitter))
			require.ErrorContains(t, emitter.Err(), testCase.wantErr)
		})
	}
}

func TestFrameAccessorsPickTheMnemonicForTheSlotKind(t *testing.T) {
	t.Parallel()

	signature, err := asmgen.NewSignature(
		[]asmgen.Parameter{
			{Name: "word", Kind: asmgen.ParameterWord}, {Name: "narrow", Kind: asmgen.ParameterWord32},
			{Name: "flag", Kind: asmgen.ParameterByte}, {Name: "single", Kind: asmgen.ParameterFloat32},
			{Name: "double", Kind: asmgen.ParameterFloat64},
		},
		[]asmgen.Parameter{
			{Name: "r1", Kind: asmgen.ParameterWord}, {Name: "r2", Kind: asmgen.ParameterWord32},
			{Name: "r3", Kind: asmgen.ParameterByte}, {Name: "r4", Kind: asmgen.ParameterFloat32},
			{Name: "r5", Kind: asmgen.ParameterFloat64},
		},
	)
	require.NoError(t, err)
	testCases := []struct {
		name     string
		argument string
		load     string
		result   string
		store    string
	}{
		{name: "word", argument: "word", load: "\tI64Load word+0(FP)\n", result: "r1", store: "\tI64Store r1+32(FP)\n"},
		{name: "word32", argument: "narrow", load: "\tI32Load narrow+8(FP)\n", result: "r2", store: "\tI32Store r2+40(FP)\n"},
		{name: "byte", argument: "flag", load: "\tI32Load8U flag+12(FP)\n", result: "r3", store: "\tI32Store8 r3+44(FP)\n"},
		{name: "float32", argument: "single", load: "\tF32Load single+16(FP)\n", result: "r4", store: "\tF32Store r4+48(FP)\n"},
		{name: "float64", argument: "double", load: "\tF64Load double+24(FP)\n", result: "r5", store: "\tF64Store r5+56(FP)\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			loads := asmgen.NewEmitter()
			NewBody(loads).LoadArgument(signature, testCase.argument)
			assert.Equal(t, testCase.load, loads.String())

			stores := asmgen.NewEmitter()
			body := NewBody(stores)
			body.StoreResult(signature, testCase.result, func() { body.Op(asmwasm.Get, "R0") })
			assert.Equal(t, "\tGet SP\n\tGet R0\n"+testCase.store, stores.String())
		})
	}
}
