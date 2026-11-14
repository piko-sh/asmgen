# asmgen

A library for **generating** Plan 9 assembly (`.s`) files for Go, architecture-agnostically.

Go ships with its own assembler. When you write a performance-critical routine in assembly,
you write one `.s` file per architecture - `foo_amd64.s`, `foo_arm64.s`, and so on - and keep
them in sync by hand. `asmgen` removes that mechanical toil. You describe a handler once, in
ordinary Go, and emit it for every target architecture through adapters you write. You commit
the output to your repository and diff it in CI, so it can never silently drift.

`asmgen` treats assembly generation as a **string-generation problem**. It performs no
instruction encoding, no operand validation, and no register allocation. You emit text lines,
and Go's own assembler (`go tool asm`) does the rest. If your text is wrong, `go tool asm`
tells you, not `asmgen`.

Module `piko.sh/asmgen`. Go 1.27. Apache-2.0.

## Install

```sh
go get piko.sh/asmgen
```

## What it is, and what it is not

`asmgen` occupies a deliberately narrow slot, and it is easy to expect the wrong thing of it.

**It is not a replacement for knowing assembly.** You still write the assembly logic yourself.
Every instruction, every register choice, every stack-frame decision is yours. `asmgen` never
encodes an instruction, never validates an operand, and never allocates a register. If you do
not already understand Plan 9 assembly and Go's calling convention, this library does not help
you write it. It removes the toil of *maintaining* assembly you already know how to write, not
the knowledge required to write it.

**It is not locked to amd64.** A tool that assumes x86-64 locks you to one architecture.
`asmgen` imposes no instruction set. Reference adapters for `amd64`, `arm64` and `wasm`
ship in the module, but the core knows nothing about any architecture. You target any
architecture Go's assembler supports by writing an adapter for it. Nothing in the core
knows or cares which ISA you mean.

**It is not an assembler, a validator, or a DSL.** There is no instruction vocabulary to learn,
no operand grammar, no register model to satisfy. Because the output is plain text, you compose
it with ordinary Go: loops, conditionals, helper functions, tables.

In return you get four things:

1. **One description, N files.** Express a handler once and emit it for every target
   architecture through adapters.
2. **Reusable cross-architecture helpers.** Because a routine is Go code, you factor shared
   structure into functions instead of copy-paste-and-tweak.
3. **Parameterised by your Go.** A handler is a Go function, so constants, struct field
   offsets, and tables from your Go types flow straight into the emitted assembly. Regenerating
   keeps the two sides in sync, and you never align them by hand.
4. **Golden-tested output.** The generator is deterministic and byte-exact, so you commit the
   generated `.s` files and let CI regenerate them in memory and diff line by line. The
   assembly can never drift from its generator.

### Who it is for

- Projects that maintain near-identical per-architecture `.s` files that must stay in
  lockstep - a VM dispatch loop, SIMD kernels, syscall shims, ABI trampolines.
- Projects that want generated assembly golden-tested so a stale `.s` file fails CI.
- Engineers who already write Plan 9 assembly by hand and want to escape the per-architecture
  copy-and-tweak cycle.

### Who it is not for

- Anyone hoping to write assembly without learning assembly. `asmgen` does not help there.
- Anyone wanting an assembler, an encoder, a register allocator, an operand validator, or
  instruction selection. `asmgen` does none of these, by design.

## Reference adapters

| Package | Target | Offers |
| --- | --- | --- |
| `asmgen_arch_amd64`, `asmamd64` | x86-64 | register moves, branches, mnemonic constants |
| `asmgen_arch_arm64`, `asmarm64` | ARM 64-bit | register moves, branches, mnemonic constants |
| `asmgen_arch_wasm`, `asmwasm` | WebAssembly | structured control, ABI0 frame accessors, a body validator, mnemonic constants |

`Signature` names argument and result slots for assembly functions. `Labels` supplies
unique labels, and `Emitter` provides alignment and constant-table helpers.
`MemoryFileSystem` captures generated files in memory for tests.

## Generating and validating

`GenerateFiles` writes generated files. `GenerateAndValidate` reports the first differing
line in each file that differs from the generated output.

`DiskFileSystem` confines file access to a project root and writes files atomically.
`WithMaxReadBytes` controls the validation read limit, which defaults to 64 MiB.

```go
fileSystem, err := asmgen.NewDiskFileSystem(repositoryRoot)
if err != nil {
    return err
}
defer fileSystem.Close()

if err := asmgen.GenerateFiles(fileSystem, architectures, groups, nil, nil); err != nil {
    return err
}
mismatches, err := asmgen.GenerateAndValidate(fileSystem, architectures, groups, nil, nil)
```

Invalid inputs return errors such as `ErrInvalidPath` or `ErrDuplicateOutput`. Every file
is generated before any is written, so a failing handler leaves the files on disk untouched.

### WebAssembly

Use the wasm adapter for whole kernels, such as vector operations or bulk byte loops.
It provides structured control flow and ABI0 argument and result access. Go's wasm
assembler allows `CALL` only at the top level of a function.

WebAssembly is a stack machine, not a register machine, so wasm handlers are written
differently from amd64 and arm64 ones. Instructions take their operands from a stack and
push their result, the `R`, `F` and `V` registers are local variables moved with `Get` and
`Set`, and control flow uses nested `Block` and `Loop` scopes with depth-based branches
instead of jumps to labels. The engine maps the stack and locals onto real registers when
it compiles the module.

```go
body := asmgen_arch_wasm.NewBody(emitter)
body.Block("done", func() {
    body.Loop("next", func() {
        body.Op(asmwasm.Get, "R1")
        body.Op(asmwasm.I64Eqz)
        body.BrIf("done")
        // ...
        body.Br("next")
    })
})
body.StoreResult(signature, "ret", func() { body.Op(asmwasm.Get, "R2") })
```

`Block`, `Loop` and `If` close their scopes automatically, and branches use labels.
`WasmArch` validates emitted handlers, including raw `Instruction` lines.

## Adapter checks

`HandlerValidator` lets an adapter check each emitted handler. Helpers can report errors
with `Emitter.Fail`. Generation and validation return these errors with the file and
handler name.
