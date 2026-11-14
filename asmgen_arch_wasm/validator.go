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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/asmgen"
	"piko.sh/asmgen/asmwasm"
)

const (
	// commentMarker starts a line or trailing comment in Go assembly.
	commentMarker = "//"

	// laneOperandCount is the number of operands a lane or V128Store instruction needs so
	// that its lane or offset reaches the encoder.
	laneOperandCount = 2
)

// openScope is a Block, Loop or If seen by the validator and not yet closed.
type openScope struct {
	// mnemonic is the instruction that opened the scope.
	mnemonic string

	// line is the one-based line within the body where it opened.
	line int

	// hasElse records whether an If has already seen its Else.
	hasElse bool
}

// validator walks a handler body line by line and collects structural problems.
type validator struct {
	// problems collects every problem found, in line order.
	problems []error

	// scopes is the stack of open scopes, outermost first.
	scopes []openScope
}

// instructionCheck checks one instruction with the given operands at the given line.
type instructionCheck func(v *validator, line int, mnemonic string, operands []string)

var (
	// instructionChecks maps a mnemonic to the check run when it appears.
	instructionChecks = map[string]instructionCheck{
		asmwasm.Block:     (*validator).openScope,
		asmwasm.Loop:      (*validator).openScope,
		asmwasm.If:        (*validator).openScope,
		asmwasm.Else:      (*validator).elseArm,
		asmwasm.End:       (*validator).closeScope,
		asmwasm.Call:      (*validator).topLevelCall,
		asmwasm.Br:        (*validator).branchDepth,
		asmwasm.BrIf:      (*validator).branchDepth,
		"BrTable":         (*validator).unparseable,
		asmwasm.V128Store: (*validator).twoOperands,
		asmwasm.F32Const:  (*validator).floatLiteral,
		asmwasm.F64Const:  (*validator).floatLiteral,
	}

	// misencodedInstructions lists instructions Go's encoder accepts but writes without
	// their memory or lane immediates, producing a module the engine rejects.
	misencodedInstructions = map[string]string{
		"V128Load8Splat":  "V128Load then I8x16Splat",
		"V128Load16Splat": "V128Load then I16x8Splat",
		"V128Load32Splat": "a scalar load then I32x4Splat or F32x4Splat",
		"V128Load64Splat": "a scalar load then I64x2Splat or F64x2Splat",
		"V128Load8x8S":    "V128Load then I16x8ExtendLowI8x16S",
		"V128Load8x8U":    "V128Load then I16x8ExtendLowI8x16U",
		"V128Load16x4S":   "V128Load then I32x4ExtendLowI16x8S",
		"V128Load16x4U":   "V128Load then I32x4ExtendLowI16x8U",
		"V128Load32x2S":   "V128Load then I64x2ExtendLowI32x4S",
		"V128Load32x2U":   "V128Load then I64x2ExtendLowI32x4U",
		"V128Load32Zero":  "a scalar load then a ReplaceLane",
		"V128Load64Zero":  "a scalar load then a ReplaceLane",
		"V128Load8Lane":   "a scalar load then I8x16ReplaceLane",
		"V128Load16Lane":  "a scalar load then I16x8ReplaceLane",
		"V128Load32Lane":  "a scalar load then I32x4ReplaceLane",
		"V128Load64Lane":  "a scalar load then I64x2ReplaceLane",
		"V128Store8Lane":  "an ExtractLane then a scalar store",
		"V128Store16Lane": "an ExtractLane then a scalar store",
		"V128Store32Lane": "an ExtractLane then a scalar store",
		"V128Store64Lane": "an ExtractLane then a scalar store",
		"I8x16Shuffle16":  "I8x16Swizzle with a V128Const index table",
	}

	// laneSuffixes are the mnemonic endings of instructions that take a lane index.
	laneSuffixes = []string{"ExtractLane", "ExtractLaneS", "ExtractLaneU", "ReplaceLane"}
)

// openScope records a Block, Loop or If.
//
// Takes line (int) which is the line of the instruction.
// Takes mnemonic (string) which is the opening instruction.
func (v *validator) openScope(line int, mnemonic string, _ []string) {
	v.scopes = append(v.scopes, openScope{mnemonic: mnemonic, line: line, hasElse: false})
}

// elseArm checks that an Else belongs to an open If without an Else.
//
// Takes line (int) which is the line of the instruction.
func (v *validator) elseArm(line int, _ string, _ []string) {
	if len(v.scopes) == 0 || v.scopes[len(v.scopes)-1].mnemonic != asmwasm.If {
		v.report(line, "Else outside an If")
		return
	}
	innermost := &v.scopes[len(v.scopes)-1]
	if innermost.hasElse {
		v.report(line, "second Else for the If opened at line %d", innermost.line)
		return
	}
	innermost.hasElse = true
}

// closeScope checks that an End has a scope to close and closes it.
//
// Takes line (int) which is the line of the instruction.
func (v *validator) closeScope(line int, _ string, _ []string) {
	if len(v.scopes) == 0 {
		v.report(line, "End with no open Block, Loop or If")
		return
	}
	v.scopes = v.scopes[:len(v.scopes)-1]
}

// topLevelCall checks that a CALL is not inside any scope.
//
// Takes line (int) which is the line of the instruction.
func (v *validator) topLevelCall(line int, _ string, _ []string) {
	if len(v.scopes) == 0 {
		return
	}
	innermost := v.scopes[len(v.scopes)-1]
	v.report(line, "CALL inside the %s opened at line %d: Go's wasm assembler allows CALL only at the top level",
		innermost.mnemonic, innermost.line)
}

// branchDepth checks that a numeric branch depth targets one of the body's own scopes.
//
// A depth at or beyond the body's nesting reaches the blocks Go wraps around every
// function, which jumps into the runtime's resume machinery instead of the author's code.
//
// Takes line (int) which is the line of the instruction.
// Takes mnemonic (string) which is Br or BrIf.
// Takes operands ([]string) which hold the depth.
func (v *validator) branchDepth(line int, mnemonic string, operands []string) {
	if len(operands) != 1 || !strings.HasPrefix(operands[0], immediatePrefix) {
		return
	}
	depth, err := strconv.Atoi(strings.TrimPrefix(operands[0], immediatePrefix))
	if err != nil || depth < 0 {
		v.report(line, "%s depth %q is not a non-negative integer", mnemonic, operands[0])
		return
	}
	if depth >= len(v.scopes) {
		v.report(line, "%s $%d escapes the %d enclosing scope(s) of this body", mnemonic, depth, len(v.scopes))
	}
}

// unparseable rejects an instruction Go's assembler cannot parse from text.
//
// Takes line (int) which is the line of the instruction.
// Takes mnemonic (string) which is the instruction.
func (v *validator) unparseable(line int, mnemonic string, _ []string) {
	v.report(line, "%s cannot be written in a .s file (the assembler panics); use Body.BrTable", mnemonic)
}

// twoOperands checks that a lane or V128Store instruction carries its immediate in the
// second operand, where the encoder reads it.
//
// Takes line (int) which is the line of the instruction.
// Takes mnemonic (string) which is the instruction.
// Takes operands ([]string) which are the instruction operands.
func (v *validator) twoOperands(line int, mnemonic string, operands []string) {
	if len(operands) != laneOperandCount {
		v.report(line, "%s needs two operands ($0, $immediate): with one the encoder silently uses 0", mnemonic)
	}
}

// floatLiteral checks that a float constant is written as a float literal.
//
// Takes line (int) which is the line of the instruction.
// Takes mnemonic (string) which is F32Const or F64Const.
// Takes operands ([]string) which hold the constant.
func (v *validator) floatLiteral(line int, mnemonic string, operands []string) {
	if len(operands) != 1 {
		v.report(line, "%s takes exactly one operand", mnemonic)
		return
	}
	if _, err := strconv.ParseInt(strings.TrimPrefix(operands[0], immediatePrefix), 10, 64); err == nil {
		v.report(line, "%s %s is an integer literal (the assembler panics); write $(%s.0)",
			mnemonic, operands[0], strings.TrimPrefix(operands[0], immediatePrefix))
	}
}

// check runs the checks that apply to one line of the body.
//
// Takes line (int) which is the one-based line number.
// Takes text (string) which is the raw line.
func (v *validator) check(line int, text string) {
	mnemonic, operands, ok := parseInstruction(text)
	if !ok {
		return
	}
	if instruction, found := instructionChecks[mnemonic]; found {
		instruction(v, line, mnemonic, operands)
		return
	}
	if replacement, found := misencodedInstructions[mnemonic]; found {
		v.report(line, "%s is encoded without its immediates and makes an invalid module; use %s",
			mnemonic, replacement)
		return
	}
	for _, suffix := range laneSuffixes {
		if strings.HasSuffix(mnemonic, suffix) {
			v.twoOperands(line, mnemonic, operands)
			return
		}
	}
}

// report records a problem at a line.
//
// Takes line (int) which is the one-based line number.
// Takes format (string) which is the fmt format of the message.
// Takes arguments (...any) which are the format arguments.
func (v *validator) report(line int, format string, arguments ...any) {
	v.problems = append(v.problems, fmt.Errorf("%w: line %d: %s", asmgen.ErrHandlerValidation, line, fmt.Sprintf(format, arguments...)))
}

// validateBody checks a handler body and returns every structural problem found.
//
// Takes body (string) which is the handler text after its TEXT directive.
//
// Returns error which joins the problems, or nil when there are none.
func validateBody(body string) error {
	walker := &validator{problems: nil, scopes: nil}
	for index, text := range strings.Split(body, "\n") {
		walker.check(index+1, text)
	}
	for _, unclosed := range walker.scopes {
		walker.report(unclosed.line, "%s is never closed with End", unclosed.mnemonic)
	}
	return errors.Join(walker.problems...)
}

// parseInstruction splits an instruction line into its mnemonic and operands.
//
// Takes text (string) which is one raw line of the body.
//
// Returns string which is the mnemonic.
// Returns []string which are the trimmed operands, nil when there are none.
// Returns bool which is false for blank lines, comments, labels and preprocessor lines.
func parseInstruction(text string) (string, []string, bool) {
	if index := strings.Index(text, commentMarker); index >= 0 {
		text = text[:index]
	}
	text = strings.TrimSpace(text)
	if text == "" || strings.HasSuffix(text, ":") || strings.HasPrefix(text, "#") {
		return "", nil, false
	}
	mnemonic, rest := text, ""
	if index := strings.IndexAny(text, " \t"); index >= 0 {
		mnemonic, rest = text[:index], strings.TrimSpace(text[index+1:])
	}
	if rest == "" {
		return mnemonic, nil, true
	}
	operands := strings.Split(rest, ",")
	for index := range operands {
		operands[index] = strings.TrimSpace(operands[index])
	}
	return mnemonic, operands, true
}
