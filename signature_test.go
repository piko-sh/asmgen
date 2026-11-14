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

func TestNewSignatureLaysOutTheABI0Frame(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		operands  map[string]string
		name      string
		frame     string
		arguments []Parameter
		results   []Parameter
	}{
		{
			name:      "two slices to a float64",
			arguments: []Parameter{{Name: "a", Kind: ParameterSlice}, {Name: "b", Kind: ParameterSlice}},
			results:   []Parameter{{Name: "ret", Kind: ParameterFloat64}},
			frame:     "$0-56",
			operands: map[string]string{
				"a_base": "a_base+0(FP)", "a_len": "a_len+8(FP)", "a_cap": "a_cap+16(FP)",
				"b_base": "b_base+24(FP)", "ret": "ret+48(FP)",
			},
		},
		{
			name:      "slice and byte to int, results start word aligned",
			arguments: []Parameter{{Name: "b", Kind: ParameterSlice}, {Name: "c", Kind: ParameterByte}},
			results:   []Parameter{{Name: "ret", Kind: ParameterWord}},
			frame:     "$0-40",
			operands:  map[string]string{"c": "c+24(FP)", "ret": "ret+32(FP)"},
		},
		{
			name:      "bool result is not padded",
			arguments: []Parameter{{Name: "a", Kind: ParameterString}, {Name: "b", Kind: ParameterString}},
			results:   []Parameter{{Name: "ret", Kind: ParameterByte}},
			frame:     "$0-33",
			operands:  map[string]string{"b_len": "b_len+24(FP)", "ret": "ret+32(FP)"},
		},
		{
			name: "small scalars align naturally",
			arguments: []Parameter{
				{Name: "flag", Kind: ParameterByte}, {Name: "count", Kind: ParameterWord32},
				{Name: "scale", Kind: ParameterFloat32}, {Name: "value", Kind: ParameterFloat64},
			},
			results:  nil,
			frame:    "$0-24",
			operands: map[string]string{"flag": "flag+0(FP)", "count": "count+4(FP)", "scale": "scale+8(FP)", "value": "value+16(FP)"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			signature, err := NewSignature(testCase.arguments, testCase.results)
			require.NoError(t, err)
			frame, err := signature.FrameSize(0)
			require.NoError(t, err)
			assert.Equal(t, testCase.frame, frame)
			for name, operand := range testCase.operands {
				slot, err := signature.Slot(name)
				require.NoError(t, err)
				assert.Equal(t, operand, slot.Operand(), name)
			}
		})
	}
}

func TestSignatureReportsUnknownSlotsAndCopiesItsLayout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		slot string
	}{
		{name: "missing slot", slot: "c_base"},
		{name: "slice parent name is not a slot", slot: "a"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			signature, err := NewSignature([]Parameter{{Name: "a", Kind: ParameterSlice}}, nil)
			require.NoError(t, err)
			_, err = signature.Slot(testCase.slot)
			require.ErrorIs(t, err, ErrUnknownSlot)
			require.ErrorContains(t, err, testCase.slot)

			slots := signature.Slots()
			slots[0].Name = "changed"
			first, err := signature.Slot("a_base")
			require.NoError(t, err)
			assert.Equal(t, 0, first.Offset)
			assert.Equal(t, 24, signature.ArgumentSize())
			frame, err := signature.FrameSize(16)
			require.NoError(t, err)
			assert.Equal(t, "$16-24", frame)
		})
	}
}

func TestNewSignatureRejectsInvalidParameters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		arguments []Parameter
		results   []Parameter
	}{
		{name: "empty argument name", arguments: []Parameter{{Name: "", Kind: ParameterWord}}, results: nil},
		{name: "empty result name", arguments: nil, results: []Parameter{{Name: "", Kind: ParameterWord}}},
		{name: "unknown kind", arguments: []Parameter{{Name: "a", Kind: ParameterKind(99)}}, results: nil},
		{name: "duplicate name", arguments: []Parameter{{Name: "a", Kind: ParameterWord}, {Name: "a", Kind: ParameterByte}}, results: nil},
		{
			name:      "derived slot collides with a scalar",
			arguments: []Parameter{{Name: "a", Kind: ParameterSlice}, {Name: "a_len", Kind: ParameterWord}},
			results:   nil,
		},
		{name: "result collides with argument", arguments: []Parameter{{Name: "ret", Kind: ParameterWord}}, results: []Parameter{{Name: "ret", Kind: ParameterWord}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewSignature(testCase.arguments, testCase.results)
			require.ErrorIs(t, err, ErrInvalidParameter)
		})
	}
}

func TestSignatureFrameSizeRejectsNegativeLocals(t *testing.T) {
	t.Parallel()

	signature, err := NewSignature(nil, nil)
	require.NoError(t, err)
	_, err = signature.FrameSize(-8)
	require.ErrorIs(t, err, ErrInvalidFrame)
}

func TestScalarSizeRejectsCompositeKinds(t *testing.T) {
	t.Parallel()

	for _, kind := range []ParameterKind{ParameterSlice, ParameterString, ParameterKind(-1)} {
		_, ok := scalarSize(kind)
		assert.False(t, ok, "kind %d", kind)
	}
}
