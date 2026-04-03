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

### Decision: Zig ✅

**Why Zig over Rust:**

1. **TDD cycle speed.** Zig rebuilds in milliseconds. Rust takes 10-20 seconds. With Red-Green-Refactor methodology and hundreds of test runs per day, this is the difference between flow state and waiting.

2. **Binary size = Monk's identity.** A 200 KB compiler reflects Monk's minimalist philosophy. A 2 MB compiler doesn't.

3. **Explicit allocators match "explicit over implicit."** Zig's allocator model teaches memory patterns that directly inform Monk's memory model. You always know where memory comes from. No hidden allocations.

4. **The ecosystem gap is smaller than it looks.** Phases 1-7 (lexer through CLI) are all custom code — no ecosystem needed. LSP (Phase 8) is the only phase where Rust's `tower-lsp` saves real time, and that's months away.

5. **Zig's own C backend validates our approach.** Zig has a C backend at 97% test coverage. They consider compile-to-C a legitimate strategy. We're using the same pattern.

**What we accept:**
- Pre-1.0 risk (pin a stable version, don't chase latest)
- Building LSP protocol layer from scratch (or wrapping a C library)
- Smaller contributor pool
- Debug-time safety instead of compile-time safety (use `GeneralPurposeAllocator` in dev)

---

## The Complete Architecture

```
Monk source (.monk)
    │
    ▼
┌──────────┐
│  Lexer   │  Zig
│  (tokens)│
└────┬─────┘
     │
     ▼
┌──────────┐
│  Parser  │  Zig
│  (AST)   │
└────┬─────┘
     │
     ▼
┌──────────────┐
│ Type Checker │  Zig (annotates AST with types, catches errors)
└──────┬───────┘
       │
       ▼
┌────────────┐
│ C Codegen  │  Zig (AST → .c file)
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
monk repl                   # Interactive (interpreted mode, tree-walking for REPL only)
```

### What Lives Where

| Component | Written in | Purpose |
|-----------|-----------|---------|
| Monk compiler | Zig | Lexer, parser, type checker, C codegen, CLI |
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
**Our approach for v1:** Eager deep copy. Simple, correct, zero refcounting.
**Future optimizations (not in v1):**
- **Copy elision:** When the source value is never used after assignment, turn the copy into a move at the Monk→C level.
- **Escape analysis:** If a value never leaves its scope, skip the heap allocation entirely.
- **Copy-on-write (COW):** Share backing storage, copy only on mutation. Can be added as an invisible optimization later.

### 5. Unicode strings
**Problem:** C's `char*` is bytes, not Unicode.
**Our approach:** Store strings as UTF-8 byte arrays internally. `length()` iterates UTF-8 sequences to count Unicode scalar values. String indexing is O(n) — acceptable for a first implementation, optimize with cached offsets later if needed.

### 6. Two-phase compile time
**Problem:** Monk→C is fast, but C→binary adds 500ms-1s.
**Our mitigation:** For `monk run`, cache the compiled binary. Re-compile only when source changes (like `go run`). The REPL uses a tree-walking interpreter (no C compilation needed for interactive use).

### 7. Optimization ceiling vs LLVM
**Problem:** C as an intermediate layer obscures language semantics from the optimizer.
**Our mitigation:** Accept it for now. Monk's value semantics actually help — C's alias analysis works better when values are copies, not pointers. For maximum performance later, add LLVM as an optional backend.

---

## What Can Change Later

| Decision | Current | Future option | Trigger |
|----------|---------|--------------|---------|
| Backend | Compile to C | LLVM or Cranelift | When optimization matters more than build simplicity |
| REPL | Tree-walking interpreter | Bytecode VM | When REPL performance matters |
| Impl language | Zig | Self-hosted (Monk compiles Monk) | When Monk is mature enough |
| Memory model | Value semantics + COW | Add `ref` parameters | When return-value-only style proves too limiting |
| String encoding | UTF-8 + O(n) indexing | Cached offsets or rope data structure | When string-heavy workloads show up |

---

## Summary

**Monk v2 is a compiler, not an interpreter.** It is written in Zig, generates C, and produces native binaries with zero runtime dependencies beyond a C compiler. The architecture is minimal, inspectable, and extensible — LLVM can be added later without changing the frontend.

The stack:
```
Zig (compiler) → C (generated code + runtime) → cc → native binary
```

This matches Monk's three design principles:
1. **Explicit over implicit:** Zig's explicit allocators, C's explicit memory, no hidden runtime
2. **Graceful on reads, strict on operations:** error handling via return codes, not setjmp magic
3. **Values, not references:** C struct copies map directly to Monk's value semantics
