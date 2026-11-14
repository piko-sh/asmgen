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

package asmgen_arch_amd64

import (
	"piko.sh/asmgen"
)

const (
	// mnemonicColumnWidth is the standard column width for amd64 Plan 9 assembly mnemonic
	// alignment.
	mnemonicColumnWidth = 8
)

// AMD64Arch implements ArchitecturePort for x86-64 Plan 9 assembly generation. All
// methods are stateless text mappers.
type AMD64Arch struct{}

var (
	_ asmgen.ArchitecturePort = (*AMD64Arch)(nil)
)

// New creates a new AMD64 architecture adapter.
//
// Returns *AMD64Arch ready for use.
func New() *AMD64Arch {
	return &AMD64Arch{}
}

// Arch returns the architecture identifier for amd64.
//
// Returns asmgen.Architecture which is always ArchitectureAMD64.
func (*AMD64Arch) Arch() asmgen.Architecture { return asmgen.ArchitectureAMD64 }

// BuildConstraint returns the build constraint suffix for amd64.
//
// Returns string which is " && amd64".
func (*AMD64Arch) BuildConstraint() string { return " && amd64" }

// MoveRegister emits a MOVQ instruction copying source to destination.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes source (string) which is the source register or operand.
// Takes destination (string) which is the destination register.
func (*AMD64Arch) MoveRegister(e *asmgen.Emitter, source, destination string) {
	e.Columned(mnemonicColumnWidth, "MOVQ", source+", "+destination)
}

// LoadImmediate emits a MOVQ instruction loading an immediate value.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes value (string) which is the immediate value to load.
// Takes destination (string) which is the destination register.
func (*AMD64Arch) LoadImmediate(e *asmgen.Emitter, value, destination string) {
	e.Columned(mnemonicColumnWidth, "MOVQ", value+", "+destination)
}

// Return emits a RET instruction. Returns from the current function.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
func (*AMD64Arch) Return(e *asmgen.Emitter) { e.Instruction("RET") }

// BranchOnCondition emits a conditional branch (J<condition>) to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes condition (string) which is the branch condition code.
// Takes label (string) which is the target label.
func (*AMD64Arch) BranchOnCondition(e *asmgen.Emitter, condition string, label string) {
	e.Columned(mnemonicColumnWidth, "J"+condition, label)
}

// UnconditionalBranch emits a JMP instruction to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes label (string) which is the target label.
func (*AMD64Arch) UnconditionalBranch(e *asmgen.Emitter, label string) {
	e.Columned(mnemonicColumnWidth, "JMP", label)
}

// TestAndBranch emits a TESTQ followed by a conditional branch.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes register (string) which is the register to test.
// Takes condition (string) which is "ZERO" or "NONZERO".
// Takes label (string) which is the target label.
func (*AMD64Arch) TestAndBranch(e *asmgen.Emitter, register, condition, label string) {
	e.Columned(mnemonicColumnWidth, "TESTQ", register+", "+register)
	switch condition {
	case "ZERO":
		e.Columned(mnemonicColumnWidth, "JZ", label)
	case "NONZERO":
		e.Columned(mnemonicColumnWidth, "JNZ", label)
	}
}
