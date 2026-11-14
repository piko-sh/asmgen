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
	"slices"
	"strconv"
	"strings"

	"piko.sh/asmgen"
	"piko.sh/asmgen/asmwasm"
)

const (
	// immediatePrefix marks an immediate operand in Go assembly.
	immediatePrefix = "$"
)

// scopeKind identifies the structured control instruction that opened a scope.
type scopeKind int

const (
	// scopeBlock is a Block, which a branch leaves.
	scopeBlock scopeKind = iota

	// scopeLoop is a Loop, which a branch restarts.
	scopeLoop

	// scopeIf is an If, which a branch leaves.
	scopeIf
)

// scope is one open structured control instruction.
type scope struct {
	// label is the name branches use to target the scope, or empty when untargetable.
	label string

	// kind is the instruction that opened the scope.
	kind scopeKind
}

// Body emits the instructions of one wasm handler with structured control flow.
type Body struct {
	// emitter receives the instruction text and any failures.
	emitter *asmgen.Emitter

	// scopes is the stack of open scopes, outermost first.
	scopes []scope
}

// NewBody creates a Body that writes to the given emitter at the top level.
//
// Takes emitter (*asmgen.Emitter) which receives the handler text.
//
// Returns *Body ready for use.
func NewBody(emitter *asmgen.Emitter) *Body {
	return &Body{emitter: emitter, scopes: nil}
}

// Op emits one instruction, indented by the current nesting depth.
//
// Takes mnemonic (string) which is the instruction, usually an asmwasm constant.
// Takes operands (...string) which are joined with ", " after a single space.
func (b *Body) Op(mnemonic string, operands ...string) {
	indent := strings.Repeat("\t", len(b.scopes))
	if len(operands) == 0 {
		b.emitter.Instruction(indent, mnemonic)
		return
	}
	b.emitter.Instruction(indent, mnemonic, " ", strings.Join(operands, ", "))
}

// Block emits a Block scope around contents.
//
// Takes label (string) which names the scope whose branches continue after the block.
// Takes contents (func()) which emits the block's instructions.
func (b *Body) Block(label string, contents func()) {
	b.open(asmwasm.Block, scope{label: label, kind: scopeBlock}, contents)
	b.Op(asmwasm.End)
}

// Loop emits a Loop scope around contents.
//
// Takes label (string) which names the scope whose branches restart the loop.
// Takes contents (func()) which emits the loop's instructions.
func (b *Body) Loop(label string, contents func()) {
	b.open(asmwasm.Loop, scope{label: label, kind: scopeLoop}, contents)
	b.Op(asmwasm.End)
}

// If emits an If scope that pops an i32 and runs contents when it is non-zero.
//
// Takes label (string) which names the scope for Br and BrIf, or empty for none.
// Takes contents (func()) which emits the instructions run when the condition holds.
func (b *Body) If(label string, contents func()) {
	b.open(asmwasm.If, scope{label: label, kind: scopeIf}, contents)
	b.Op(asmwasm.End)
}

// IfElse emits an If scope with an Else arm.
//
// Takes label (string) which names the scope for Br and BrIf, or empty for none.
// Takes then (func()) which emits the instructions run when the condition holds.
// Takes otherwise (func()) which emits the instructions run when it does not.
func (b *Body) IfElse(label string, then func(), otherwise func()) {
	b.open(asmwasm.If, scope{label: label, kind: scopeIf}, then)
	b.scopes = append(b.scopes, scope{label: label, kind: scopeIf})
	b.emitter.Instruction(strings.Repeat("\t", len(b.scopes)-1), asmwasm.Else)
	otherwise()
	b.scopes = b.scopes[:len(b.scopes)-1]
	b.Op(asmwasm.End)
}

// Br emits an unconditional branch to the named enclosing scope.
//
// Takes label (string) which names an open Block, Loop or If.
func (b *Body) Br(label string) {
	if depth, ok := b.depthOf(asmwasm.Br, label); ok {
		b.Op(asmwasm.Br, immediatePrefix+strconv.Itoa(depth))
	}
}

// BrIf emits a branch to the named enclosing scope taken when the popped i32 is non-zero.
//
// Takes label (string) which names an open Block, Loop or If.
func (b *Body) BrIf(label string) {
	if depth, ok := b.depthOf(asmwasm.BrIf, label); ok {
		b.Op(asmwasm.BrIf, immediatePrefix+strconv.Itoa(depth))
	}
}

// BrTable emits a multi-way branch on the value of an i64 register local.
//
// Go's assembler cannot parse a BrTable operand list, so the table is lowered to one
// compare-and-BrIf per case followed by a Br to the default. Case k is taken when the
// register holds k.
//
// Takes indexRegister (string) which is the R register holding the case index.
// Takes defaultLabel (string) which names the scope taken for any other value.
// Takes caseLabels (...string) which name the scope for index 0, 1, and so on.
func (b *Body) BrTable(indexRegister string, defaultLabel string, caseLabels ...string) {
	for index, label := range caseLabels {
		b.Op(asmwasm.Get, indexRegister)
		b.Op(asmwasm.I64Const, immediatePrefix+strconv.Itoa(index))
		b.Op(asmwasm.I64Eq)
		b.BrIf(label)
	}
	b.Br(defaultLabel)
}

// Call emits a Go CALL, which the assembler allows only at the top level.
//
// Takes symbol (string) which is the callee's assembly symbol.
func (b *Body) Call(symbol string) {
	if len(b.scopes) > 0 {
		b.emitter.Fail("CALL %s inside %s: Go's wasm assembler allows CALL only at the top level",
			symbol, b.describeScopes())
		return
	}
	b.Op(asmwasm.Call, symbol)
}

// Return emits the Go RET pseudo-instruction.
func (b *Body) Return() {
	b.Op(asmwasm.Return)
}

// LoadArgument pushes an ABI0 argument slot onto the wasm stack.
//
// Word slots push an i64, Word32 and Byte slots an i32 (bytes zero-extended), and float
// slots an f32 or f64.
//
// Takes signature (asmgen.Signature) which is the function's frame layout.
// Takes name (string) which is the slot name, such as "a_base".
func (b *Body) LoadArgument(signature asmgen.Signature, name string) {
	slot, err := signature.Slot(name)
	if err != nil {
		b.emitter.Fail("loading argument: %w", err)
		return
	}
	b.Op(loadMnemonic(slot.Kind), slot.Operand())
}

// StoreResult stores a value into an ABI0 result slot.
//
// A wasm store takes its address before its value, so StoreResult pushes SP, runs value
// to push the result, then emits the store that matches the slot's kind.
//
// Takes signature (asmgen.Signature) which is the function's frame layout.
// Takes name (string) which is the slot name, such as "ret".
// Takes value (func()) which pushes exactly one value of the slot's type.
func (b *Body) StoreResult(signature asmgen.Signature, name string, value func()) {
	slot, err := signature.Slot(name)
	if err != nil {
		b.emitter.Fail("storing result: %w", err)
		return
	}
	b.Op(asmwasm.Get, asmwasm.RegisterStackPointer)
	value()
	b.Op(storeMnemonic(slot.Kind), slot.Operand())
}

// open emits the opening instruction of a scope, then its contents one level deeper.
//
// Takes mnemonic (string) which is Block, Loop or If.
// Takes opened (scope) which describes the scope.
// Takes contents (func()) which emits the scope's instructions.
func (b *Body) open(mnemonic string, opened scope, contents func()) {
	b.Op(mnemonic)
	b.scopes = append(b.scopes, opened)
	contents()
	b.scopes = b.scopes[:len(b.scopes)-1]
}

// depthOf resolves a label to the relative branch depth of its scope.
//
// Takes mnemonic (string) which is the branch instruction, for the error message.
// Takes label (string) which names the target scope.
//
// Returns int which is the relative depth, 0 for the innermost scope.
// Returns bool which is false, after recording a failure, when no open scope has the
// label.
func (b *Body) depthOf(mnemonic string, label string) (int, bool) {
	for index, candidate := range slices.Backward(b.scopes) {
		if label != "" && candidate.label == label {
			return len(b.scopes) - 1 - index, true
		}
	}
	b.emitter.Fail("%s to %q: no enclosing Block, Loop or If has that label (open: %s)",
		mnemonic, label, b.describeScopes())
	return 0, false
}

// describeScopes lists the open scopes, outermost first, for error messages.
//
// Returns string such as "Block done > Loop next", or "the top level".
func (b *Body) describeScopes() string {
	if len(b.scopes) == 0 {
		return "the top level"
	}
	parts := make([]string, 0, len(b.scopes))
	for _, scoped := range b.scopes {
		parts = append(parts, strings.TrimSpace(scopeMnemonic(scoped.kind)+" "+scoped.label))
	}
	return strings.Join(parts, " > ")
}

// scopeMnemonic returns the instruction that opens a scope kind.
//
// Takes kind (scopeKind) which is the scope kind.
//
// Returns string which is Block, Loop or If.
func scopeMnemonic(kind scopeKind) string {
	switch kind {
	case scopeLoop:
		return asmwasm.Loop
	case scopeIf:
		return asmwasm.If
	case scopeBlock:
		return asmwasm.Block
	}
	return asmwasm.Block
}

// loadMnemonic returns the load instruction that pushes a slot of the given kind.
//
// Takes kind (asmgen.ParameterKind) which is a scalar slot kind.
//
// Returns string which is the load mnemonic.
func loadMnemonic(kind asmgen.ParameterKind) string {
	switch kind {
	case asmgen.ParameterWord32:
		return asmwasm.I32Load
	case asmgen.ParameterByte:
		return asmwasm.I32Load8U
	case asmgen.ParameterFloat64:
		return asmwasm.F64Load
	case asmgen.ParameterFloat32:
		return asmwasm.F32Load
	case asmgen.ParameterWord, asmgen.ParameterSlice, asmgen.ParameterString:
		return asmwasm.I64Load
	}
	return asmwasm.I64Load
}

// storeMnemonic returns the store instruction that writes a slot of the given kind.
//
// Takes kind (asmgen.ParameterKind) which is a scalar slot kind.
//
// Returns string which is the store mnemonic.
func storeMnemonic(kind asmgen.ParameterKind) string {
	switch kind {
	case asmgen.ParameterWord32:
		return asmwasm.I32Store
	case asmgen.ParameterByte:
		return asmwasm.I32Store8
	case asmgen.ParameterFloat64:
		return asmwasm.F64Store
	case asmgen.ParameterFloat32:
		return asmwasm.F32Store
	case asmgen.ParameterWord, asmgen.ParameterSlice, asmgen.ParameterString:
		return asmwasm.I64Store
	}
	return asmwasm.I64Store
}
