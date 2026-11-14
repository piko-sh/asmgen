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
	"piko.sh/asmgen"
)

// WasmArch implements ArchitecturePort and HandlerValidator for Go's wasm assembler.
type WasmArch struct{}

var (
	_ asmgen.ArchitecturePort = (*WasmArch)(nil)

	_ asmgen.HandlerValidator = (*WasmArch)(nil)
)

// New creates a new wasm architecture adapter.
//
// Returns *WasmArch ready for use.
func New() *WasmArch {
	return &WasmArch{}
}

// Arch returns the architecture identifier for wasm.
//
// Returns asmgen.Architecture which is always ArchitectureWASM.
func (*WasmArch) Arch() asmgen.Architecture { return asmgen.ArchitectureWASM }

// BuildConstraint returns the build constraint suffix for wasm.
//
// Returns string which is " && wasm".
func (*WasmArch) BuildConstraint() string { return " && wasm" }

// ValidateHandler checks the emitted body of a handler for structural mistakes.
//
// Takes body (string) which is the handler text after its TEXT directive.
//
// Returns error which joins every problem found, each prefixed with its line number
// within the body, or nil when the body is sound.
func (*WasmArch) ValidateHandler(_ string, body string) error {
	return validateBody(body)
}
