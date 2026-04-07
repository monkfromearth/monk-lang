# Architecture Decisions — 2026-04-03

This document captures the full journey from "what language do we implement in" to "how does the compiler work." Every option considered, why it was rejected or chosen, and what can change later.

---

## Why Not Keep TypeScript/Bun (v1)?

Monk v1 was a tree-walking interpreter written in TypeScript, running on Bun.

**Why it hit a ceiling:**
- V8's garbage collector runs underneath — makes "no GC" a lie
- Tree-walking is the slowest execution model (pointer-chasing, cache misses, recursive dispatch)
- Can't produce native binaries — always needs the Bun runtime
- Binary distribution is heavy (Bun runtime is 50+ MB)
- Value semantics are impossible to enforce — JS has reference semantics for objects/arrays

**What was good:** fast iteration, great ecosystem, 599 tests passing. The language design and spec survived. The implementation didn't.

---

## Execution Model: What Kind of Language Implementation?

### Options Evaluated

| Approach | How it works | Speed | Complexity | Native output? |
|----------|-------------|-------|------------|----------------|
| Tree-walking interpreter | Walk AST, evaluate nodes | Slowest | Lowest | No |
| Bytecode VM | Compile to bytecodes, run in VM loop | 10-100x tree-walk | Medium | No |
| **Compile to C** | Generate C source, compile with cc | Native speed | Medium | **Yes** |
| LLVM backend | Generate LLVM IR, LLVM produces binary | Best optimized | High | Yes |
| Own machine code backend | Encode x86/ARM instructions directly | Native speed | Very high | Yes |

### Decision: Compile to C ✅

**Why:**
- Monk produces **native binaries**. No runtime, no VM, no interpreter needed on the user's machine.
- Zero dependencies beyond a C compiler (every system has one).
- C's value semantics (struct assignment copies) map directly to Monk's value semantics.
- Generated C is inspectable and debuggable.
- All of `cc`'s decades of optimization work — for free.
- Zig itself has a C backend at 97% test coverage as their universal fallback.
- Nim proves this approach scales to a real language with real users.

**What we skip:**
- No 100+ MB LLVM dependency
- No custom register allocator or instruction selector
- No bytecode format to design and maintain

**What we accept:**
- Two-phase compilation (Monk→C, then C→binary). Adds ~500ms-1s.
- Generated C has mangled names (debugging requires source maps via `#line` directives).
- Some optimizations are harder through C (tail call elimination, whole-program optimization).

### Future: LLVM as Optional Backend

The frontend (lexer → parser → AST → type checker) is identical regardless of backend. When Monk is mature, LLVM or Cranelift can be added as an alternative codegen target:

```
                  ┌─→ C Codegen → cc → native binary (default, portable)
AST → Type Check ─┤
                  └─→ LLVM Codegen → LLVM → native binary (future, optimized)
```

The C backend stays as the universal fallback. LLVM becomes the "release mode" optimization pass. This is exactly what Nim's `nlvm` does alongside Nim's default C backend.

---

## Implementation Language: The Full Journey

### Options Evaluated

| Language | Binary size | Build speed | Safety | Ecosystem | Deterministic memory |
|----------|------------|-------------|--------|-----------|---------------------|
| TypeScript/Bun | N/A (runtime) | Fast | N/A | Excellent | No (V8 GC) |
| Go | 5-10 MB | Fast | Good | Strong | No (Go GC) |
| Rust | 1-3 MB | Slow (10-20s) | Excellent | Excellent | Yes |
| C | 200-500 KB | Fast | None | Manual | Yes |
| C++ | 500 KB-2 MB | Slow | Partial | Fragmented | Yes |
| Nim | 200-600 KB | Fast | Good (ARC) | Small | Yes (ARC) |
| Odin | 300 KB-1 MB | Medium | None | Tiny | Yes |
| **Zig** | **100-400 KB** | **Fastest** | **Debug-time** | **Thin** | **Yes** |

### Eliminated

- **TypeScript/Bun** — GC underneath makes "no GC" impossible. Can't produce native binaries.
- **Go** — 5-10 MB binaries. Bundled GC contradicts philosophy. Reference semantics fight Monk's value semantics.
- **C++** — Rust's downsides (build speed) without its upsides (safety, ecosystem coherence).
- **Odin** — Too niche. Zero track record for language implementations.
- **C** — Proven (Lua, CPython, Ruby) but no memory safety net. No sum types. Verbose AST handling.
- **Nim** — ARC model is great. But small community, uncertain long-term momentum. Ecosystem risk.

### Final Two: Rust vs Zig

| Factor | Rust | Zig |
|--------|------|-----|
| Build speed (TDD) | 10-20 seconds | **Milliseconds** |
| Binary size | 1-3 MB | **100-400 KB** |
| Allocator control | Hidden (`Vec::push` allocates) | **Explicit (you choose)** |
| AST ergonomics | `Box<Rc<RefCell<T>>>` | Simpler tagged unions |
| LSP library | `tower-lsp` (saves weeks) | Build from scratch |
| Safety | Compile-time borrow checker | Debug-time leak detection |
| Stability | Stable, backward compatible | Pre-1.0, breaking changes |
| Community | Massive | Growing |
| Languages built in it | Gleam, Rhai, Deno | Bun (partial) |

### Decision: Go ✅ (revised from Zig)

Initial choice was Zig for minimal footprint and explicit memory. Revised after recognizing that the Monk compiler is fundamentally a **text-in, text-out translator** — read source, build trees, write C. The hard parts are tree manipulation and string generation, not hardware control.

**Why Go:**

1. **The language stays out of the way.** GC handles AST node allocation. Strings are first-class. Trees are trivial. You focus on compiler design, not memory management.

2. **Fast builds for TDD.** `go test ./...` runs in seconds for a 50K LOC project. Red-Green-Refactor stays in flow state.

3. **Single binary distribution.** `go build` produces a static binary. `monk build hello.monk` just works, no runtime install.

4. **You already know Go.** The goal is learning compiler design and C, not learning a new implementation language. Go is the tool, not the lesson.

5. **Strong ecosystem.** LSP libraries (`gopls` as reference, `go.lsp.dev/protocol`), testing built in, JSON built in, CLI libraries mature.

**What we accept:**
- 5-10 MB compiler binary (Go runtime + GC bundled). Acceptable for a CLI tool.
- Go's GC manages the *compiler's* memory, not Monk's. The generated C has no GC.
- Go's reference semantics for slices/maps internally. Doesn't affect Monk's value semantics — those are enforced in the generated C, not in the compiler.

**Zig/Rust deferred to:** future backend work — LLVM integration, custom machine code backend, or self-hosted Monk compiler. When the compiler needs to generate optimized native code directly, a systems language makes sense. For the current compile-to-C pipeline, Go is right.

---

## The Complete Architecture

```
Monk source (.monk)
    │
    ▼
┌──────────────────┐
│ Module Resolver  │  Go (DFS resolve, cycle detect, topo-sort)
│ (dependency graph)│  Single-file programs skip this step
└────────┬─────────┘
         │ per module:
         ▼
┌──────────┐
│  Lexer   │  Go
│  (tokens)│
└────┬─────┘
     │
     ▼
┌──────────┐
│  Parser  │  Go
│  (AST)   │
└────┬─────┘
     │
     ▼
┌──────────────┐
│ Type Checker │  Go (annotates AST with types, catches errors)
│              │  Cross-module imports resolved in dependency order
└──────┬───────┘
       │
       ▼
┌────────────┐
│ C Codegen  │  Go (AST → single .c file, all modules inlined)
└──────┬─────┘
       │
       ▼
┌──────────┐
│   cc     │  System C compiler (GCC, Clang, MSVC)
│ (binary) │
└──────────┘
       │
       ▼
  Native executable
```

### User-Facing Commands

```bash
monk build hello.monk       # Compile to native binary: hello.monk → hello.c → hello
monk run hello.monk         # Compile and run in one step
monk check hello.monk       # Type check without compiling
monk lint hello.monk        # Code quality
monk format hello.monk      # Code formatting
```

### What Lives Where

| Component | Written in | Purpose |
|-----------|-----------|---------|
| Monk compiler | Go | Lexer, parser, type checker, module resolver, C codegen, CLI |
| Monk runtime library | C | Built-in functions (show, math, string ops, array ops) |
| Generated code | C | The user's Monk program, compiled to C |
| Final binary | Native | Linked: generated code + runtime library |

The runtime library is a small C library (~2-5 KB) that provides:
- `MonkValue` tagged union type
- `monk_show()`, `monk_to_string()`, etc.
- Array/record/string operations with value semantics
- Error handling infrastructure (guard/against/throw)

This ships with the Monk compiler and gets linked into every compiled program.

---

## Known Shortcomings of Compile-to-C (From Research)

These are real problems other compile-to-C languages have hit. Our mitigations:

### 1. Debugging: mangled names in generated C
**Problem:** Nim appends random suffixes to identifiers. GDB shows `myVar_201190`.
**Our mitigation:** Emit readable C. Use Monk variable names directly where possible. Emit `#line` directives for every Monk source line. Generate a `.monk.map` source map for column-level debugging.

### 2. Closures in C
**Problem:** C has no closures. Need function pointer + environment struct pairs.
**Our approach:** Closures capture by copy (like C++ `[x]` lambdas). Generate a struct for each closure's captured variables. The closure owns its environment — no sharing, no refcounting. Freed when the closure goes out of scope.

### 3. Guard/against/throw
**Two options considered:**
- **Return-code propagation:** Every throwable function returns a result type. Every call site checks the tag. Zero happy-path overhead but requires transitive analysis of which functions can throw, and generates a lot of checking boilerplate.
- **setjmp/longjmp:** One `setjmp` per `guard` block. `throw` jumps back. No per-call-site checking. No analysis needed. Simpler codegen. Optimizer cost on `guard` blocks (saves/restores registers).

**Our approach:** `setjmp`/`longjmp`. Simpler to implement, no transitive analysis, no generated error-checking at every call site. The optimizer cost is localized to `guard` blocks, not spread across every function call. Can revisit with return-code propagation later if profiling shows setjmp overhead matters.

### 4. Value semantics (copy on assign) performance
**Problem:** Deep-copying a 10,000 element array on every `let b = a` is expensive.
**Our approach:** Copy-on-write for arrays; eager deep copy remains for records/strings/functions. Simple semantics, avoids O(n) array copies on assignment.
**Future optimizations:**
- **Copy elision:** When the source value is never used after assignment, turn the copy into a move at the Monk→C level.
- **Escape analysis:** If a value never leaves its scope, skip the heap allocation entirely.

### Performance Optimization Roadmap

The three remaining gaps between Monk and C performance — in order of impact:

#### A. Typed array backing store ✅ SHIPPED (matmul 11× → ~1.6× C)

**Current state:** `int[]` is a `MonkValue` whose `array_val->data` is `MonkValue[]` — a 16-byte struct per element (8 bytes tag + 8 bytes value). A 400×400 matrix uses 2.56 MB. C uses 1.28 MB. Every cache line holds half as many numbers.

**The fix:** New C runtime struct `MonkIntArray { int64_t* data; int64_t length; }` (and Float/Bool variants). `int[]` variables are backed by `int64_t*` instead of `MonkValue*`. Element access is `arr->data[i]` — no union, no tag, cache-friendly.

**What changes:**
- New tagged kind in `MonkValue` union: `int_array_val`, `float_array_val`, `bool_array_val`
- New allocation helpers in `runtime.c`: `monk_int_array_new(n)`, etc.
- Codegen emits `MonkValue x = monk_int_array_new(n)` for `int[]` declarations
- Element access `arr[i]` becomes `arr.int_array_val->data[i]` — same field-inline approach as current inline access, but now on a tighter struct
- At generic boundaries (passing to `map`, `filter`, `typeof`, builtins that take `MonkValue`), box the element: `monk_int(arr.int_array_val->data[i])`

**Spec impact:** None. `int[]` behaves identically — same type errors, same OOB semantics.

**Effort:** Medium. Touches runtime struct definition, 3-4 codegen paths, and array builtins (`append`, `prepend`, `range`, `slice`).

#### B. Copy-on-write for arrays ✅ SHIPPED (binary_trees ~63× → ~32× C)

**Previous state:** `let b = a` deep-copied every element. A 10,000-element array cost 160 KB of memcpy. This was spec-correct but wasteful when the copy was never mutated.

**The fix:** Reference-count the backing `data` pointer. On assign, increment refcount — no copy yet. On first mutation, check refcount > 1; if so, copy-then-write. If refcount == 1, write in place.

**What changed:**
- `MonkArray`, `MonkIntArray`, `MonkFloatArray`, and `MonkBoolArray` grew a `refcount` field.
- `monk_deep_copy` shares array backing stores (increment refcount, O(1)).
- `monk_free` decrements refcount; frees only when refcount hits 0
- `monk_array_set` and typed-array direct writes detach before mutation when needed.
- Codegen tracks typed arrays that are provably fresh (`range`, `fill`, literals) and skips the COW barrier in hot loops.

**Spec impact:** None. Value semantics are preserved — mutations don't bleed across copies. The spec says "assignment copies"; COW is an invisible optimization.

#### C. Bounds-check elision for typed arrays ✅ SHIPPED (matmul ~2× → ~1.6× C)

**Current state:** Every typed-array element access emits an OOB guard: `if (i < 0 || i >= arr.array_val->length) monk_panic(...)`. In a hot inner loop (e.g., matmul), this is 2 branches per access — branch predictor handles it, but it's noise.

**The fix:** Elide the bounds check when the type checker can prove safety. Specifically:
- `for i in range(0, arr.length)` — `i` is in `[0, length)` by construction; accesses `arr[i]` in the loop body don't need a check
- Literal index access `arr[0]` when array length is statically known

**What needs spec clarification:** Typed arrays (`int[]`, not `int?[]`) should be defined as **strict** — OOB is always a panic, not `none`. The current `graceful reads` principle applies only to untyped access. This is already implied by the inline access path (which panics on OOB), but the spec should say it explicitly.

**Effort:** Low for the spec clarification. Medium for dataflow analysis to track provably-safe index ranges.

### 5. Unicode strings
**Problem:** C's `char*` is bytes, not Unicode.
**Our approach:** Store strings as UTF-8 byte arrays internally. `length()` iterates UTF-8 sequences to count Unicode scalar values. String indexing is O(n) — acceptable for a first implementation, optimize with cached offsets later if needed.

### 6. Two-phase compile time
**Problem:** Monk→C is fast, but C→binary adds 500ms-1s.
**Our mitigation:** For `monk run`, cache the compiled binary. Re-compile only when source changes (like `go run`).

### 7. Optimization ceiling vs LLVM
**Problem:** C as an intermediate layer obscures language semantics from the optimizer.
**Our mitigation:** Accept it for now. Monk's value semantics actually help — C's alias analysis works better when values are copies, not pointers. For maximum performance later, add LLVM as an optional backend.

---

## Module System Architecture

Multi-file Monk programs compile to a single `.c` file. This is the simplest approach that works — no linker complexity, no extern declarations, no header generation.

### How it works

```
entry.monk ─┬─→ module.Build()     DFS resolve, cycle detect, topo-sort
             │   returns Graph{Order: [leaf.monk, mid.monk, entry.monk]}
             │
             ├─→ types.CheckModules()  Check each module in dependency order
             │   Imports inject bindings from already-checked modules
             │
             └─→ codegen.GenerateModules()  Emit single .c with:
                  - Module-prefixed names (mk_m0_x, _monk_m0_func_1)
                  - Static globals for non-entry module variables
                  - Init functions with once-guards
                  - Entry module code in main()
```

### Key design decisions

**Single `.c` output (not multiple `.o` files).** Functions are already `static` — making them visible across translation units would require `extern` declarations and header generation. A single file means one `cc` invocation, one set of `#line` directives, and the optimizer sees everything. Multi-`.o` linking can be added later as an incremental compilation optimization if compile times become a problem.

**Module-prefixed name mangling.** Entry module keeps `mk_NAME` (no prefix). Imported modules get `mk_m{N}_NAME` (e.g. `mk_m0_helper`). Prevents C-level name collisions between modules that both define a `helper` function. The numeric ID is assigned by topological order position.

**Static globals for module variables.** Non-entry module variables must be visible from both the init function (where they're assigned) and the entry module's `main()` (where they're used via imports). Declaring them as `static` at file scope solves this. The init function body handles assignment.

**Init functions with once-guards.** Each non-entry module gets `static void _mod_N_init(void)` with a `static int _initialized` flag. Diamond dependencies (A imports B and C, both import D) call D's init from both B and C, but only the first call executes.

**Storage propagation across modules.** When module A exports a scalar variable (e.g. `let x int = 42`, stored as `int64_t`), importing modules need to know it's `int64_t`, not `MonkValue`. The `storageKind` is tracked per-export and propagated to importers so `emitExpr` correctly boxes scalar values when passing to builtin functions.

### What the module system does NOT do

- **No package registry or remote imports.** Paths are always relative (`./`, `../`).
- **No conditional imports.** All imports are resolved statically at compile time.
- **No re-export syntax.** Re-exporting works by importing a name and then `export name` — but there's no `export { X } from "./mod"` shorthand.
- **No namespace objects.** `use * from "./mod"` dumps all exports into the current scope. There's no `mod.X` syntax.

---

## What Can Change Later

| Decision | Current | Future option | Trigger |
|----------|---------|--------------|---------|
| Backend | Compile to C | LLVM or Cranelift | When optimization matters more than build simplicity |
| Impl language | Go | Zig/Rust (for native backend) or self-hosted | When adding LLVM or custom codegen |
| Memory model | Value semantics + COW | Add `ref` parameters | When return-value-only style proves too limiting |
| String encoding | UTF-8 + O(n) indexing | Cached offsets or rope data structure | When string-heavy workloads show up |
| Module output | Single `.c` file | Multiple `.o` files | When compile times become a bottleneck for large programs |
| Module paths | Relative only (`./`, `../`) | Package registry, absolute imports | When ecosystem needs sharing |

---

## Summary

**Monk v2 is a compiler, not an interpreter.** It is written in Go, generates C, and produces native binaries with zero runtime dependencies beyond a C compiler. Multi-file programs compile to a single `.c` via the module resolver. The architecture is minimal, inspectable, and extensible — LLVM can be added later without changing the frontend.

The stack:
```
Go (compiler) → C (generated code + runtime) → cc → native binary
```

Go handles the translation (trees, strings, text processing). C handles the output (value semantics, explicit memory, no GC in the generated programs). Each language is used where it's strongest.
