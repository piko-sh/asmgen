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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstruction(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Instruction("MOVQ DX, AX")

	assert.Equal(t, "\tMOVQ DX, AX\n", e.String())
}

func TestInstruction_MultiplePartsJoined(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Instruction("MOVQ", "    DX, AX")

	assert.Equal(t, "\tMOVQ    DX, AX\n", e.String())
}

func TestLabel(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Label("done")

	assert.Equal(t, "done:\n", e.String())
}

func TestComment(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Comment("text")

	assert.Equal(t, "// text\n", e.String())
}

func TestIndentedComment(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.IndentedComment("text")

	assert.Equal(t, "\t// text\n", e.String())
}

func TestBlank(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Blank()

	assert.Equal(t, "\n", e.String())
}

func TestLine(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Line("text")

	assert.Equal(t, "text\n", e.String())
}

func TestRaw(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Raw("raw")

	assert.Equal(t, "raw", e.String())
}

func TestStringAndBytes(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Line("hello")
	e.Line("world")

	expected := "hello\nworld\n"

	assert.Equal(t, expected, e.String())
	assert.Equal(t, []byte(expected), e.Bytes())
}

func TestLen(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	assert.Equal(t, 0, e.Len())

	e.Line("abc")

	assert.Equal(t, 4, e.Len())

	e.Instruction("RET")

	assert.Equal(t, 9, e.Len())
}

func TestReset(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Line("something")
	require.Greater(t, e.Len(), 0)

	e.Reset()
	assert.Equal(t, 0, e.Len())
	assert.Equal(t, "", e.String())
}

func TestAccumulation(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Comment("header")
	e.Label("start")
	e.Instruction("MOVQ    AX, BX")
	e.Blank()
	e.IndentedComment("return")
	e.Instruction("RET")

	expected := "// header\n" +
		"start:\n" +
		"\tMOVQ    AX, BX\n" +
		"\n" +
		"\t// return\n" +
		"\tRET\n"

	assert.Equal(t, expected, e.String())
}

func TestNewEmitter(t *testing.T) {
	t.Parallel()

	e := NewEmitter()

	assert.Equal(t, 0, e.Len())
	assert.Equal(t, "", e.String())
	assert.Empty(t, e.Bytes())
}

func TestColumnedPadsOperandsToTheColumn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		mnemonic string
		operands string
		expected string
		width    int
	}{
		{name: "short mnemonic", width: 8, mnemonic: "MOVQ", operands: "AX, BX", expected: "\tMOVQ    AX, BX\n"},
		{name: "mnemonic at the column keeps one space", width: 5, mnemonic: "MOVD", operands: "R1, R2", expected: "\tMOVD R1, R2\n"},
		{name: "mnemonic past the column keeps one space", width: 5, mnemonic: "VMOVDQU", operands: "Y0, Y1", expected: "\tVMOVDQU Y0, Y1\n"},
		{name: "no operands has no padding", width: 8, mnemonic: "RET", operands: "", expected: "\tRET\n"},
		{name: "negative width keeps one space", width: -4, mnemonic: "RET", operands: "R0", expected: "\tRET R0\n"},
		{name: "multi-byte mnemonic is measured in runes", width: 6, mnemonic: "MOV\u00b7", operands: "X", expected: "\tMOV\u00b7  X\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			emitter := NewEmitter()
			emitter.Columned(testCase.width, testCase.mnemonic, testCase.operands)
			assert.Equal(t, testCase.expected, emitter.String())
		})
	}
}

func TestDataTableEmitsWordsAndGlobl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		symbol   string
		flags    string
		expected string
		values   []uint64
	}{
		{
			name:   "two words",
			symbol: "mask<>",
			flags:  "RODATA|NOPTR",
			values: []uint64{0xff, 0x8000000000000000},
			expected: "DATA mask<>+0(SB)/8, $0xff\n" +
				"DATA mask<>+8(SB)/8, $0x8000000000000000\n" +
				"GLOBL mask<>(SB), RODATA|NOPTR, $16\n",
		},
		{
			name:     "empty table still declares the symbol",
			symbol:   "empty<>",
			flags:    "RODATA",
			values:   nil,
			expected: "GLOBL empty<>(SB), RODATA, $0\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			emitter := NewEmitter()
			emitter.DataTable(testCase.symbol, testCase.flags, testCase.values)
			assert.Equal(t, testCase.expected, emitter.String())
		})
	}
}

func TestFailRecordsErrorsUntilReset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		messages []string
		reset    bool
		wantErr  bool
	}{
		{name: "no failures", messages: nil, reset: false, wantErr: false},
		{name: "one failure", messages: []string{"bad label"}, reset: false, wantErr: true},
		{name: "two failures are joined", messages: []string{"first", "second"}, reset: false, wantErr: true},
		{name: "reset clears failures", messages: []string{"bad"}, reset: true, wantErr: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			emitter := NewEmitter()
			for _, message := range testCase.messages {
				emitter.Fail("handler: %s", message)
			}
			if testCase.reset {
				emitter.Reset()
			}
			err := emitter.Err()
			if !testCase.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrHandlerEmission)
			for _, message := range testCase.messages {
				assert.Contains(t, err.Error(), "handler: "+message)
			}
		})
	}
}

func TestFailWrapsErrorArguments(t *testing.T) {
	t.Parallel()

	emitter := NewEmitter()
	emitter.Fail("loading: %w", ErrUnknownSlot)
	err := emitter.Err()
	require.ErrorIs(t, err, ErrHandlerEmission)
	require.ErrorIs(t, err, ErrUnknownSlot)
}

func TestCommentFormatsEachLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{name: "single line", text: "text", expected: "// text\n"},
		{name: "multiple lines", text: "first\nsecond", expected: "// first\n// second\n"},
		{name: "blank line becomes bare marker", text: "first\n\nthird", expected: "// first\n//\n// third\n"},
		{name: "already commented line is kept", text: "// kept\nadded", expected: "// kept\n// added\n"},
		{name: "empty text", text: "", expected: "//\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			emitter := NewEmitter()
			emitter.Comment(testCase.text)
			assert.Equal(t, testCase.expected, emitter.String())
		})
	}
}
