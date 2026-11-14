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
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// dataWordSize is the width in bytes of each DATA directive emitted by DataTable.
	dataWordSize = 8
)

// Emitter writes Plan 9 assembly text to an internal buffer. It provides convenience
// methods for common assembly formatting patterns but performs no validation or
// transformation; callers are responsible for producing correctly formatted text.
//
// The emitter intentionally has no formatting intelligence. It writes exactly what it is
// told. This is critical for character-by-character matching against existing
// hand-written assembly files.
type Emitter struct {
	// failures collects the errors recorded with Fail, in the order they were recorded.
	failures []error

	// buffer accumulates the emitted assembly text.
	buffer bytes.Buffer
}

// NewEmitter creates a new assembly text emitter with an empty buffer.
//
// Returns *Emitter ready for use.
func NewEmitter() *Emitter {
	return &Emitter{}
}

// Instruction emits a tab-indented instruction line.
//
// Takes parts (...string) which are concatenated to form the instruction text.
func (e *Emitter) Instruction(parts ...string) {
	_ = e.buffer.WriteByte('\t')
	for _, part := range parts {
		_, _ = e.buffer.WriteString(part)
	}
	_ = e.buffer.WriteByte('\n')
}

// Columned emits a tab-indented instruction whose operands start at a fixed column.
//
// The mnemonic is followed by enough spaces to reach width characters, and always at
// least one, so a mnemonic as long as the column is still separated from its operands.
// The mnemonic is measured in runes, and a width of zero or less yields the single
// separating space. Operands are omitted, along with the padding, when empty.
//
// Takes width (int) which is the column at which operands start.
// Takes mnemonic (string) which is the instruction name.
// Takes operands (string) which is the operand text, or empty for none.
func (e *Emitter) Columned(width int, mnemonic string, operands string) {
	if operands == "" {
		e.Instruction(mnemonic)
		return
	}
	padding := max(1, width-utf8.RuneCountInString(mnemonic))
	e.Instruction(mnemonic, strings.Repeat(" ", padding), operands)
}

// DataTable emits a read-only table of 64-bit words as DATA directives followed by its
// GLOBL directive.
//
// Takes symbol (string) which names the table, with a "<>" suffix for a file-local symbol.
// Takes flags (string) which is the GLOBL flags, such as "RODATA|NOPTR".
// Takes values ([]uint64) which are the table words in order.
func (e *Emitter) DataTable(symbol string, flags string, values []uint64) {
	for index, value := range values {
		offset := strconv.Itoa(index * dataWordSize)
		e.Line("DATA " + symbol + "+" + offset + "(SB)/8, $0x" + strconv.FormatUint(value, 16))
	}
	size := strconv.Itoa(len(values) * dataWordSize)
	e.Line("GLOBL " + symbol + "(SB), " + flags + ", $" + size)
}

// Fail records a generation error against the text being emitted.
//
// Every recorded error wraps ErrHandlerEmission, and a %w verb in format wraps its
// argument as fmt.Errorf does.
//
// Takes format (string) which is the fmt format of the error message.
// Takes arguments (...any) which are the format arguments.
func (e *Emitter) Fail(format string, arguments ...any) {
	e.failures = append(e.failures, fmt.Errorf("%w: %w", ErrHandlerEmission, fmt.Errorf(format, arguments...)))
}

// Err returns the errors recorded with Fail, joined, or nil when there are none.
//
// Returns error which joins every recorded failure.
func (e *Emitter) Err() error {
	return errors.Join(e.failures...)
}

// Label emits a label on its own line, followed by a newline.
//
// Takes name (string) which is the label identifier.
func (e *Emitter) Label(name string) {
	_, _ = e.buffer.WriteString(name)
	_, _ = e.buffer.WriteString(":\n")
}

// Comment emits a non-indented comment line.
//
// Takes text (string) which is the comment body.
func (e *Emitter) Comment(text string) {
	for index, line := range strings.Split(text, "\n") {
		if index > 0 {
			_ = e.buffer.WriteByte('\n')
		}
		switch {
		case line == "":
			_, _ = e.buffer.WriteString("//")
		case strings.HasPrefix(line, "//"):
			_, _ = e.buffer.WriteString(line)
		default:
			_, _ = e.buffer.WriteString("// ")
			_, _ = e.buffer.WriteString(line)
		}
	}
	_ = e.buffer.WriteByte('\n')
}

// IndentedComment emits a tab-indented comment line.
//
// Takes text (string) which is the comment body.
func (e *Emitter) IndentedComment(text string) {
	_, _ = e.buffer.WriteString("\t// ")
	_, _ = e.buffer.WriteString(text)
	_ = e.buffer.WriteByte('\n')
}

// Blank emits a single blank line.
func (e *Emitter) Blank() {
	_ = e.buffer.WriteByte('\n')
}

// Line emits a raw line of text followed by a newline.
//
// Takes text (string) which is written verbatim before the newline.
func (e *Emitter) Line(text string) {
	_, _ = e.buffer.WriteString(text)
	_ = e.buffer.WriteByte('\n')
}

// Raw writes text to the buffer with no additional formatting.
//
// Takes text (string) which is written verbatim.
func (e *Emitter) Raw(text string) {
	_, _ = e.buffer.WriteString(text)
}

// Bytes returns the accumulated assembly text as a byte slice.
//
// Returns []byte containing all emitted text.
func (e *Emitter) Bytes() []byte {
	return e.buffer.Bytes()
}

// String returns the accumulated assembly text as a string.
//
// Returns string containing all emitted text.
func (e *Emitter) String() string {
	return e.buffer.String()
}

// Len returns the number of bytes written so far.
//
// Returns int which is the current buffer length.
func (e *Emitter) Len() int {
	return e.buffer.Len()
}

// Reset clears the buffer and any recorded failures for reuse.
func (e *Emitter) Reset() {
	e.buffer.Reset()
	e.failures = nil
}
