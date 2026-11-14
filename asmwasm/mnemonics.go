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

// Package asmwasm exports the Go assembler's WebAssembly mnemonics used by asmgen
// consumers, plus the register names Go maps onto wasm locals.
package asmwasm

const (
	// RegisterStackPointer is the Go stack pointer, an i32 global cached in a local.
	RegisterStackPointer = "SP"
)

const (
	// Call is the Go CALL pseudo-instruction, allowed only outside Block, Loop and If scopes.
	Call = "CALL"

	// Return is the Go RET pseudo-instruction, which pops the Go frame and returns to the
	// caller.
	Return = "RET"

	// MoveWord is the Go MOVD pseudo-instruction, which copies 8 bytes between a register
	// and memory.
	MoveWord = "MOVD"
)

const (
	// Block opens a block whose branches continue after its End.
	Block = "Block"

	// Loop opens a loop whose branches restart it.
	Loop = "Loop"

	// If pops an i32 and runs its body when the value is non-zero.
	If = "If"

	// Else starts the alternative body of the innermost If.
	Else = "Else"

	// End closes the innermost Block, Loop or If.
	End = "End"

	// Br branches unconditionally to the enclosing scope at the given relative depth.
	Br = "Br"

	// BrIf pops an i32 and branches to the enclosing scope at the given depth when it is
	// non-zero.
	BrIf = "BrIf"

	// Unreachable traps unconditionally.
	Unreachable = "Unreachable"

	// Nop does nothing.
	Nop = "Nop"

	// Drop discards the value on top of the stack.
	Drop = "Drop"

	// Select pops a condition and two values and keeps the first when the condition is
	// non-zero.
	Select = "Select"
)

const (
	// Get pushes the value of a register local (R0 to R15 are i64, F0 to F15 f32, F16 to
	// F31 f64, V0 to V15 v128, SP is i32).
	Get = "Get"

	// Set pops the top of the stack into a register local.
	Set = "Set"

	// Tee stores the top of the stack into a register local without popping it.
	Tee = "Tee"
)

const (
	// I32Const pushes a 32-bit integer immediate.
	I32Const = "I32Const"

	// I64Const pushes a 64-bit integer immediate.
	I64Const = "I64Const"

	// F32Const pushes a float32 immediate written as a float literal, such as $(0.0).
	F32Const = "F32Const"

	// F64Const pushes a float64 immediate written as a float literal, such as $(0.0).
	F64Const = "F64Const"

	// V128Const pushes a 128-bit immediate written as two operands, the low then the high
	// 64 bits.
	V128Const = "V128Const"
)

const (
	// I32Load loads 4 bytes as an i32.
	I32Load = "I32Load"

	// I64Load loads 8 bytes as an i64.
	I64Load = "I64Load"

	// F32Load loads a float32.
	F32Load = "F32Load"

	// F64Load loads a float64.
	F64Load = "F64Load"

	// I32Load8U loads 1 byte zero-extended to i32.
	I32Load8U = "I32Load8U"

	// I32Load8S loads 1 byte sign-extended to i32.
	I32Load8S = "I32Load8S"

	// I32Load16U loads 2 bytes zero-extended to i32.
	I32Load16U = "I32Load16U"

	// I32Load16S loads 2 bytes sign-extended to i32.
	I32Load16S = "I32Load16S"

	// I64Load8U loads 1 byte zero-extended to i64.
	I64Load8U = "I64Load8U"

	// I64Load8S loads 1 byte sign-extended to i64.
	I64Load8S = "I64Load8S"

	// I64Load32U loads 4 bytes zero-extended to i64.
	I64Load32U = "I64Load32U"

	// I64Load32S loads 4 bytes sign-extended to i64.
	I64Load32S = "I64Load32S"

	// I32Store stores an i32 as 4 bytes.
	I32Store = "I32Store"

	// I64Store stores an i64 as 8 bytes.
	I64Store = "I64Store"

	// F32Store stores a float32.
	F32Store = "F32Store"

	// F64Store stores a float64.
	F64Store = "F64Store"

	// I32Store8 stores the low byte of an i32.
	I32Store8 = "I32Store8"

	// I32Store16 stores the low 2 bytes of an i32.
	I32Store16 = "I32Store16"

	// I64Store8 stores the low byte of an i64.
	I64Store8 = "I64Store8"

	// I64Store32 stores the low 4 bytes of an i64.
	I64Store32 = "I64Store32"

	// MemoryCopy copies a byte range within linear memory (destination, source and length
	// on the stack).
	MemoryCopy = "MemoryCopy"

	// MemoryFill fills a byte range of linear memory with a byte value.
	MemoryFill = "MemoryFill"
)

const (
	// I32Add adds i32 values.
	I32Add = "I32Add"

	// I32Sub subtracts i32 values.
	I32Sub = "I32Sub"

	// I32Mul multiplies i32 values.
	I32Mul = "I32Mul"

	// I32DivS divides signed i32 values.
	I32DivS = "I32DivS"

	// I32DivU divides unsigned i32 values.
	I32DivU = "I32DivU"

	// I32RemS takes the signed remainder of i32 values.
	I32RemS = "I32RemS"

	// I32RemU takes the unsigned remainder of i32 values.
	I32RemU = "I32RemU"

	// I32And ands i32 values.
	I32And = "I32And"

	// I32Or ors i32 values.
	I32Or = "I32Or"

	// I32Xor exclusive-ors i32 values.
	I32Xor = "I32Xor"

	// I32Shl shifts left i32 values.
	I32Shl = "I32Shl"

	// I32ShrS shifts right arithmetically i32 values.
	I32ShrS = "I32ShrS"

	// I32ShrU shifts right logically i32 values.
	I32ShrU = "I32ShrU"

	// I32Eqz tests for zero, giving an i32, i32 values.
	I32Eqz = "I32Eqz"

	// I32Eq compares for equality, giving an i32, i32 values.
	I32Eq = "I32Eq"

	// I32Ne compares for inequality, giving an i32, i32 values.
	I32Ne = "I32Ne"

	// I32LtS compares signed less-than, giving an i32, i32 values.
	I32LtS = "I32LtS"

	// I32LtU compares unsigned less-than, giving an i32, i32 values.
	I32LtU = "I32LtU"

	// I32GtS compares signed greater-than, giving an i32, i32 values.
	I32GtS = "I32GtS"

	// I32GtU compares unsigned greater-than, giving an i32, i32 values.
	I32GtU = "I32GtU"

	// I32LeS compares signed less-or-equal, giving an i32, i32 values.
	I32LeS = "I32LeS"

	// I32LeU compares unsigned less-or-equal, giving an i32, i32 values.
	I32LeU = "I32LeU"

	// I32GeS compares signed greater-or-equal, giving an i32, i32 values.
	I32GeS = "I32GeS"

	// I32GeU compares unsigned greater-or-equal, giving an i32, i32 values.
	I32GeU = "I32GeU"

	// I64Add adds i64 values.
	I64Add = "I64Add"

	// I64Sub subtracts i64 values.
	I64Sub = "I64Sub"

	// I64Mul multiplies i64 values.
	I64Mul = "I64Mul"

	// I64DivS divides signed i64 values.
	I64DivS = "I64DivS"

	// I64DivU divides unsigned i64 values.
	I64DivU = "I64DivU"

	// I64RemS takes the signed remainder of i64 values.
	I64RemS = "I64RemS"

	// I64RemU takes the unsigned remainder of i64 values.
	I64RemU = "I64RemU"

	// I64And ands i64 values.
	I64And = "I64And"

	// I64Or ors i64 values.
	I64Or = "I64Or"

	// I64Xor exclusive-ors i64 values.
	I64Xor = "I64Xor"

	// I64Shl shifts left i64 values.
	I64Shl = "I64Shl"

	// I64ShrS shifts right arithmetically i64 values.
	I64ShrS = "I64ShrS"

	// I64ShrU shifts right logically i64 values.
	I64ShrU = "I64ShrU"

	// I64Eqz tests for zero, giving an i32, i64 values.
	I64Eqz = "I64Eqz"

	// I64Eq compares for equality, giving an i32, i64 values.
	I64Eq = "I64Eq"

	// I64Ne compares for inequality, giving an i32, i64 values.
	I64Ne = "I64Ne"

	// I64LtS compares signed less-than, giving an i32, i64 values.
	I64LtS = "I64LtS"

	// I64LtU compares unsigned less-than, giving an i32, i64 values.
	I64LtU = "I64LtU"

	// I64GtS compares signed greater-than, giving an i32, i64 values.
	I64GtS = "I64GtS"

	// I64GtU compares unsigned greater-than, giving an i32, i64 values.
	I64GtU = "I64GtU"

	// I64LeS compares signed less-or-equal, giving an i32, i64 values.
	I64LeS = "I64LeS"

	// I64LeU compares unsigned less-or-equal, giving an i32, i64 values.
	I64LeU = "I64LeU"

	// I64GeS compares signed greater-or-equal, giving an i32, i64 values.
	I64GeS = "I64GeS"

	// I64GeU compares unsigned greater-or-equal, giving an i32, i64 values.
	I64GeU = "I64GeU"
)

const (
	// F32Add adds f32 values.
	F32Add = "F32Add"

	// F32Sub subtracts f32 values.
	F32Sub = "F32Sub"

	// F32Mul multiplies f32 values.
	F32Mul = "F32Mul"

	// F32Div divides f32 values.
	F32Div = "F32Div"

	// F32Min takes the minimum of f32 values.
	F32Min = "F32Min"

	// F32Max takes the maximum of f32 values.
	F32Max = "F32Max"

	// F32Sqrt takes the square root of f32 values.
	F32Sqrt = "F32Sqrt"

	// F32Abs takes the absolute value of f32 values.
	F32Abs = "F32Abs"

	// F32Neg negates f32 values.
	F32Neg = "F32Neg"

	// F32Eq compares for equality, giving an i32, f32 values.
	F32Eq = "F32Eq"

	// F32Ne compares for inequality, giving an i32, f32 values.
	F32Ne = "F32Ne"

	// F32Lt compares less-than, giving an i32, f32 values.
	F32Lt = "F32Lt"

	// F32Gt compares greater-than, giving an i32, f32 values.
	F32Gt = "F32Gt"

	// F32Le compares less-or-equal, giving an i32, f32 values.
	F32Le = "F32Le"

	// F32Ge compares greater-or-equal, giving an i32, f32 values.
	F32Ge = "F32Ge"

	// F64Add adds f64 values.
	F64Add = "F64Add"

	// F64Sub subtracts f64 values.
	F64Sub = "F64Sub"

	// F64Mul multiplies f64 values.
	F64Mul = "F64Mul"

	// F64Div divides f64 values.
	F64Div = "F64Div"

	// F64Min takes the minimum of f64 values.
	F64Min = "F64Min"

	// F64Max takes the maximum of f64 values.
	F64Max = "F64Max"

	// F64Sqrt takes the square root of f64 values.
	F64Sqrt = "F64Sqrt"

	// F64Abs takes the absolute value of f64 values.
	F64Abs = "F64Abs"

	// F64Neg negates f64 values.
	F64Neg = "F64Neg"

	// F64Eq compares for equality, giving an i32, f64 values.
	F64Eq = "F64Eq"

	// F64Ne compares for inequality, giving an i32, f64 values.
	F64Ne = "F64Ne"

	// F64Lt compares less-than, giving an i32, f64 values.
	F64Lt = "F64Lt"

	// F64Gt compares greater-than, giving an i32, f64 values.
	F64Gt = "F64Gt"

	// F64Le compares less-or-equal, giving an i32, f64 values.
	F64Le = "F64Le"

	// F64Ge compares greater-or-equal, giving an i32, f64 values.
	F64Ge = "F64Ge"
)

const (
	// I32WrapI64 truncates an i64 to its low 32 bits, as needed for every memory address.
	I32WrapI64 = "I32WrapI64"

	// I64ExtendI32U zero-extends an i32 to i64.
	I64ExtendI32U = "I64ExtendI32U"

	// I64ExtendI32S sign-extends an i32 to i64.
	I64ExtendI32S = "I64ExtendI32S"

	// F64ConvertI64S converts a signed i64 to float64.
	F64ConvertI64S = "F64ConvertI64S"

	// F64PromoteF32 widens a float32 to float64.
	F64PromoteF32 = "F64PromoteF32"

	// F32DemoteF64 narrows a float64 to float32.
	F32DemoteF64 = "F32DemoteF64"

	// I64TruncSatF64S converts a float64 to a signed i64, saturating instead of trapping.
	I64TruncSatF64S = "I64TruncSatF64S"
)

const (
	// V128Load loads 16 bytes using a single offset operand.
	V128Load = "V128Load"

	// V128Store stores 16 bytes using operands $0, $offset.
	V128Store = "V128Store"

	// V128And ands two v128 values.
	V128And = "V128And"

	// V128Or ors two v128 values.
	V128Or = "V128Or"

	// V128Xor exclusive-ors two v128 values.
	V128Xor = "V128Xor"

	// V128Not inverts a v128 value.
	V128Not = "V128Not"

	// V128AnyTrue gives an i32 that is 1 when any bit of a v128 is set.
	V128AnyTrue = "V128AnyTrue"

	// I8x16Splat fills 16 byte lanes from an i32.
	I8x16Splat = "I8x16Splat"

	// I8x16Eq compares 16 byte lanes for equality.
	I8x16Eq = "I8x16Eq"

	// I8x16Bitmask gathers the top bit of each byte lane into an i32.
	I8x16Bitmask = "I8x16Bitmask"

	// I8x16Swizzle selects byte lanes by index from a second v128.
	I8x16Swizzle = "I8x16Swizzle"

	// I32x4Splat fills four i32 lanes.
	I32x4Splat = "I32x4Splat"

	// I32x4Add adds four i32 lanes.
	I32x4Add = "I32x4Add"

	// I32x4ExtractLane extracts an i32 lane using operands $0, $lane.
	I32x4ExtractLane = "I32x4ExtractLane"

	// I64x2Splat fills two i64 lanes.
	I64x2Splat = "I64x2Splat"

	// I64x2Add adds two i64 lanes.
	I64x2Add = "I64x2Add"

	// I64x2ExtractLane extracts an i64 lane using operands $0, $lane.
	I64x2ExtractLane = "I64x2ExtractLane"

	// F32x4Splat fills four float32 lanes.
	F32x4Splat = "F32x4Splat"

	// F32x4Add adds four float32 lanes.
	F32x4Add = "F32x4Add"

	// F32x4Sub subtracts four float32 lanes.
	F32x4Sub = "F32x4Sub"

	// F32x4Mul multiplies four float32 lanes.
	F32x4Mul = "F32x4Mul"

	// F32x4ExtractLane extracts a float32 lane using operands $0, $lane.
	F32x4ExtractLane = "F32x4ExtractLane"

	// F64x2Splat fills two float64 lanes.
	F64x2Splat = "F64x2Splat"

	// F64x2Add adds two float64 lanes.
	F64x2Add = "F64x2Add"

	// F64x2Sub subtracts two float64 lanes.
	F64x2Sub = "F64x2Sub"

	// F64x2Mul multiplies two float64 lanes.
	F64x2Mul = "F64x2Mul"

	// F64x2Div divides two float64 lanes.
	F64x2Div = "F64x2Div"

	// F64x2ExtractLane extracts a float64 lane using operands $0, $lane.
	F64x2ExtractLane = "F64x2ExtractLane"
)
