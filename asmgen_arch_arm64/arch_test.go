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

package asmgen_arch_arm64

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/asmgen"
)

func TestArch(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, asmgen.ArchitectureARM64, arch.Arch())
}

func TestBuildConstraint(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, " && arm64", arch.BuildConstraint())
}

func TestMoveRegister(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.MoveRegister(e, "R3", "R4")

	assert.Equal(t, "\tMOVD R3, R4\n", e.String())
}

func TestLoadImmediate(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadImmediate(e, "$99", "R5")

	assert.Equal(t, "\tMOVD $99, R5\n", e.String())
}

func TestReturn(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.Return(e)

	assert.Equal(t, "\tRET\n", e.String())
}

func TestBranchOnCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition string
		expected  string
	}{
		{name: "LE", condition: "LE", expected: "\tBLE  done\n"},
		{name: "LT", condition: "LT", expected: "\tBLT  done\n"},
		{name: "EQ", condition: "EQ", expected: "\tBEQ  done\n"},
		{name: "NE", condition: "NE", expected: "\tBNE  done\n"},
		{name: "GE", condition: "GE", expected: "\tBGE  done\n"},
		{name: "GT", condition: "GT", expected: "\tBGT  done\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arch := New()
			e := asmgen.NewEmitter()
			arch.BranchOnCondition(e, tt.condition, "done")

			assert.Equal(t, tt.expected, e.String())
		})
	}
}

func TestUnconditionalBranch(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.UnconditionalBranch(e, "loop")

	assert.Equal(t, "\tB    loop\n", e.String())
}

func TestTestAndBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition string
		expected  string
	}{
		{
			name:      "ZERO",
			condition: "ZERO",
			expected:  "\tCBZ  R3, skip\n",
		},
		{
			name:      "NONZERO",
			condition: "NONZERO",
			expected:  "\tCBNZ R3, skip\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arch := New()
			e := asmgen.NewEmitter()
			arch.TestAndBranch(e, "R3", tt.condition, "skip")

			assert.Equal(t, tt.expected, e.String())
		})
	}
}
