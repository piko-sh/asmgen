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
	"piko.sh/asmgen"
)

const (
	// mnemonicColumnWidth is the standard column width for arm64 Plan 9 assembly mnemonic
	// alignment.
	mnemonicColumnWidth = 5
)

// ARM64Arch implements ArchitecturePort for ARM 64-bit Plan 9 assembly generation. All
// methods are stateless text mappers.
type ARM64Arch struct{}

var (
	_ asmgen.ArchitecturePort = (*ARM64Arch)(nil)
)

// New creates a new ARM64 architecture adapter.
//
// Returns *ARM64Arch ready for use.
func New() *ARM64Arch {
	return &ARM64Arch{}
}

// Arch returns the architecture identifier for arm64.
//
// Returns asmgen.Architecture which is always ArchitectureARM64.
func (*ARM64Arch) Arch() asmgen.Architecture { return asmgen.ArchitectureARM64 }

// BuildConstraint returns the build constraint suffix for arm64.
//
// Returns string which is " && arm64".
func (*ARM64Arch) BuildConstraint() string { return " && arm64" }

// MoveRegister emits a MOVD instruction copying source to destination.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes source (string) which is the source register or operand.
// Takes destination (string) which is the destination register.
func (*ARM64Arch) MoveRegister(e *asmgen.Emitter, source, destination string) {
	e.Columned(mnemonicColumnWidth, "MOVD", source+", "+destination)
}

// LoadImmediate emits a MOVD instruction loading an immediate value.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes value (string) which is the immediate value to load.
// Takes destination (string) which is the destination register.
func (*ARM64Arch) LoadImmediate(e *asmgen.Emitter, value, destination string) {
	e.Columned(mnemonicColumnWidth, "MOVD", value+", "+destination)
}

// Return emits a RET instruction. Returns from the current function.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
func (*ARM64Arch) Return(e *asmgen.Emitter) {
	e.Instruction("RET")
}

// BranchOnCondition emits a conditional branch (B<condition>) to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes condition (string) which is the branch condition code.
// Takes label (string) which is the target label.
func (*ARM64Arch) BranchOnCondition(e *asmgen.Emitter, condition string, label string) {
	e.Columned(mnemonicColumnWidth, "B"+condition, label)
}

// UnconditionalBranch emits a B instruction to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes label (string) which is the target label.
func (*ARM64Arch) UnconditionalBranch(e *asmgen.Emitter, label string) {
	e.Columned(mnemonicColumnWidth, "B", label)
}

// TestAndBranch emits a compare-and-branch instruction for arm64.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes register (string) which is the register to test.
// Takes condition (string) which is "ZERO" or "NONZERO".
// Takes label (string) which is the target label.
func (*ARM64Arch) TestAndBranch(e *asmgen.Emitter, register, condition, label string) {
	switch condition {
	case "ZERO":
		e.Columned(mnemonicColumnWidth, "CBZ", register+", "+label)
	case "NONZERO":
		e.Columned(mnemonicColumnWidth, "CBNZ", register+", "+label)
	}
}
