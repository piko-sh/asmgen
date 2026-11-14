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
	"fmt"
	"slices"
	"strconv"
)

const (
	// abi0WordSize is the width of an ABI0 pointer or register slot, in bytes.
	abi0WordSize = 8

	// abi0Word32Size is the size of a 32-bit integer slot.
	abi0Word32Size = 4

	// abi0Float32Size is the size of a float32 slot.
	abi0Float32Size = 4

	// abi0ByteSize is the size of a bool, byte or int8 slot.
	abi0ByteSize = 1
)

// ParameterKind identifies the Go type class of an ABI0 argument or result.
type ParameterKind int

const (
	// ParameterWord is a 64-bit integer, uintptr or pointer.
	ParameterWord ParameterKind = iota

	// ParameterWord32 is a 32-bit signed or unsigned integer.
	ParameterWord32

	// ParameterByte is a bool, byte or int8.
	ParameterByte

	// ParameterFloat64 is a float64.
	ParameterFloat64

	// ParameterFloat32 is a float32.
	ParameterFloat32

	// ParameterSlice is a slice, laid out as the _base, _len and _cap words.
	ParameterSlice

	// ParameterString is a string, laid out as the _base and _len words.
	ParameterString
)

// Parameter names one Go argument or result and its type class.
type Parameter struct {
	// Name is the Go parameter name, which prefixes every slot name derived from it.
	Name string

	// Kind is the parameter's type class.
	Kind ParameterKind
}

// Slot is one scalar position in an ABI0 argument frame.
type Slot struct {
	// Name is the slot name used before the offset in an FP operand.
	Name string

	// Kind is the scalar type class, excluding ParameterSlice and ParameterString.
	Kind ParameterKind

	// Offset is the byte offset of the slot from FP.
	Offset int

	// Size is the slot width in bytes.
	Size int
}

// Signature is the ABI0 frame layout of an assembly function's Go declaration.
type Signature struct {
	// slots holds every argument slot followed by every result slot, in frame order.
	slots []Slot

	// argumentSize is the total size of arguments and results in bytes.
	argumentSize int
}

// Operand returns the FP-relative operand naming the slot, such as "a_base+0(FP)".
//
// Returns string which is the operand text.
func (s Slot) Operand() string {
	return s.Name + "+" + strconv.Itoa(s.Offset) + "(FP)"
}

// NewSignature lays out the ABI0 frame for a function's arguments and results.
//
// Takes arguments ([]Parameter) which are the Go arguments in declaration order.
// Takes results ([]Parameter) which are the Go results in declaration order. Name a single
// unnamed result "ret" to match go vet.
//
// Returns Signature which holds the computed layout.
// Returns error which wraps ErrInvalidParameter when a parameter has an empty name, an
// unknown kind, or a slot name that collides with another parameter's slot.
func NewSignature(arguments []Parameter, results []Parameter) (Signature, error) {
	var slots []Slot
	var err error
	offset := 0
	for _, parameter := range arguments {
		if slots, offset, err = appendParameterSlots(slots, parameter, offset); err != nil {
			return Signature{}, err
		}
	}
	offset = alignUp(offset, abi0WordSize)
	for _, parameter := range results {
		if slots, offset, err = appendParameterSlots(slots, parameter, offset); err != nil {
			return Signature{}, err
		}
	}
	return Signature{slots: slots, argumentSize: offset}, nil
}

// Slot finds a slot by name.
//
// Takes name (string) which is the slot name, such as "a_len" or "ret".
//
// Returns Slot which is the named slot.
// Returns error which wraps ErrUnknownSlot when no slot has that name.
func (s Signature) Slot(name string) (Slot, error) {
	for _, slot := range s.slots {
		if slot.Name == name {
			return slot, nil
		}
	}
	return Slot{}, fmt.Errorf("%w: %q", ErrUnknownSlot, name)
}

// Slots returns every slot in frame order, arguments first.
//
// Returns []Slot which is a copy of the layout.
func (s Signature) Slots() []Slot {
	return append([]Slot(nil), s.slots...)
}

// ArgumentSize returns the combined size of the arguments and results in bytes.
//
// Returns int which is the size used after the dash in a TEXT frame.
func (s Signature) ArgumentSize() int {
	return s.argumentSize
}

// FrameSize returns the TEXT directive frame operand for the given local frame size.
//
// Takes locals (int) which is the local frame size in bytes, usually 0 for a leaf.
//
// Returns string which is the frame operand, such as "$0-56".
// Returns error which wraps ErrInvalidFrame when locals is negative.
func (s Signature) FrameSize(locals int) (string, error) {
	if locals < 0 {
		return "", fmt.Errorf("%w: locals %d is negative", ErrInvalidFrame, locals)
	}
	return "$" + strconv.Itoa(locals) + "-" + strconv.Itoa(s.argumentSize), nil
}

// appendParameterSlots appends the slots of one parameter at the next aligned offset.
//
// Takes slots ([]Slot) which is the layout so far.
// Takes parameter (Parameter) which is the parameter to lay out.
// Takes offset (int) which is the first free byte offset.
//
// Returns []Slot which is the extended layout.
// Returns int which is the first free byte offset after the parameter.
// Returns error which wraps ErrInvalidParameter when the parameter is invalid.
func appendParameterSlots(slots []Slot, parameter Parameter, offset int) ([]Slot, int, error) {
	parts, err := parameterSlots(parameter)
	if err != nil {
		return nil, 0, err
	}
	for _, part := range parts {
		if slices.ContainsFunc(slots, func(existing Slot) bool { return existing.Name == part.Name }) {
			return nil, 0, fmt.Errorf("%w: slot %q is defined twice", ErrInvalidParameter, part.Name)
		}
		offset = alignUp(offset, part.Size)
		part.Offset = offset
		slots = append(slots, part)
		offset += part.Size
	}
	return slots, offset, nil
}

// parameterSlots returns the unplaced slots one parameter occupies.
//
// Takes parameter (Parameter) which is the parameter to expand.
//
// Returns []Slot which holds the slots with zero offsets.
// Returns error which wraps ErrInvalidParameter when the name is empty or the kind unknown.
func parameterSlots(parameter Parameter) ([]Slot, error) {
	if parameter.Name == "" {
		return nil, fmt.Errorf("%w: parameter name is empty", ErrInvalidParameter)
	}
	if parameter.Kind == ParameterSlice || parameter.Kind == ParameterString {
		slots := []Slot{
			{Name: parameter.Name + "_base", Kind: ParameterWord, Offset: 0, Size: abi0WordSize},
			{Name: parameter.Name + "_len", Kind: ParameterWord, Offset: 0, Size: abi0WordSize},
		}
		if parameter.Kind == ParameterSlice {
			slots = append(slots, Slot{Name: parameter.Name + "_cap", Kind: ParameterWord, Offset: 0, Size: abi0WordSize})
		}
		return slots, nil
	}
	size, ok := scalarSize(parameter.Kind)
	if !ok {
		return nil, fmt.Errorf("%w: parameter %q has unknown kind %d", ErrInvalidParameter, parameter.Name, parameter.Kind)
	}
	return []Slot{{Name: parameter.Name, Kind: parameter.Kind, Offset: 0, Size: size}}, nil
}

// scalarSize returns the byte width of a scalar parameter kind.
//
// Takes kind (ParameterKind) which is a scalar kind.
//
// Returns int which is the slot size in bytes.
// Returns bool which is false when kind is not a scalar kind.
func scalarSize(kind ParameterKind) (int, bool) {
	switch kind {
	case ParameterWord32:
		return abi0Word32Size, true
	case ParameterFloat32:
		return abi0Float32Size, true
	case ParameterByte:
		return abi0ByteSize, true
	case ParameterWord, ParameterFloat64:
		return abi0WordSize, true
	case ParameterSlice, ParameterString:
		return 0, false
	}
	return 0, false
}

// alignUp rounds offset up to the next multiple of alignment.
//
// Takes offset (int) which is the offset to round.
// Takes alignment (int) which is a power of two.
//
// Returns int which is the aligned offset.
func alignUp(offset int, alignment int) int {
	return (offset + alignment - 1) &^ (alignment - 1)
}
