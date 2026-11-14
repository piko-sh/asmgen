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
)

func TestValidateHandlerAcceptsSoundBodies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "comments, labels and includes are ignored", body: "// note\nloop:\n#define X 1\n\tRET\n"},
		{name: "call at the top level", body: "\tBlock\n\tEnd\n\tCALL f(SB)\n\tRET\n"},
		{name: "nested scopes with in-range branches", body: "\tBlock\n\tLoop\n\tBrIf $1\n\tBr $0\n\tEnd\n\tEnd\n"},
		{name: "if with one else", body: "\tIf\n\tNop\n\tElse\n\tNop\n\tEnd\n"},
		{name: "two-operand lane and store forms", body: "\tF64x2ExtractLane $0, $1\n\tV128Store $0, $16\n\tI8x16ExtractLaneU\t$0, $3\n"},
		{name: "float literals", body: "\tF64Const $(0.0)\n\tF32Const $1.5\n"},
		{name: "branch to a label operand is left to the assembler", body: "\tBr done\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, New().ValidateHandler("kernel", testCase.body))
		})
	}
}

func TestValidateHandlerRejectsStructuralMistakes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{name: "end without a scope", body: "\tEnd\n", wantErr: "line 1: End with no open Block, Loop or If"},
		{name: "scope never closed", body: "\tRET\n\tLoop\n", wantErr: "line 2: Loop is never closed with End"},
		{name: "else outside if", body: "\tBlock\n\tElse\n\tEnd\n", wantErr: "line 2: Else outside an If"},
		{name: "second else", body: "\tIf\n\tElse\n\tElse\n\tEnd\n", wantErr: "line 3: second Else for the If opened at line 1"},
		{name: "call inside a loop", body: "\tLoop\n\tCALL f(SB)\n\tEnd\n", wantErr: "line 2: CALL inside the Loop opened at line 1"},
		{name: "branch escaping the body", body: "\tBlock\n\tBrIf $1\n\tEnd\n", wantErr: "line 2: BrIf $1 escapes the 1 enclosing scope(s)"},
		{name: "branch at the top level", body: "\tBr $0\n", wantErr: "line 1: Br $0 escapes the 0 enclosing scope(s)"},
		{name: "malformed branch depth", body: "\tBlock\n\tBr $x\n\tEnd\n", wantErr: `line 2: Br depth "$x" is not a non-negative integer`},
		{name: "branch table", body: "\tBrTable $0, $1\n", wantErr: "line 1: BrTable cannot be written in a .s file"},
		{name: "one-operand lane extract", body: "\tF64x2ExtractLane $1\n", wantErr: "line 1: F64x2ExtractLane needs two operands"},
		{name: "one-operand replace lane", body: "\tI32x4ReplaceLane $2\n", wantErr: "line 1: I32x4ReplaceLane needs two operands"},
		{name: "one-operand v128 store", body: "\tV128Store $16\n", wantErr: "line 1: V128Store needs two operands"},
		{name: "splat load", body: "\tV128Load64Splat $0\n", wantErr: "line 1: V128Load64Splat is encoded without its immediates"},
		{name: "shuffle", body: "\tI8x16Shuffle16\n", wantErr: "use I8x16Swizzle with a V128Const index table"},
		{name: "integer float constant", body: "\tF64Const $0\n", wantErr: "line 1: F64Const $0 is an integer literal (the assembler panics); write $(0.0)"},
		{name: "float constant without operand", body: "\tF32Const\n", wantErr: "line 1: F32Const takes exactly one operand"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorContains(t, New().ValidateHandler("kernel", testCase.body), testCase.wantErr)
		})
	}
}

func TestWasmArchIdentity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		got      string
		expected string
	}{
		{name: "architecture", got: string(New().Arch()), expected: string(asmgen.ArchitectureWASM)},
		{name: "build constraint", got: New().BuildConstraint(), expected: " && wasm"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, testCase.got)
		})
	}
}
