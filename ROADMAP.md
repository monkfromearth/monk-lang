# Monk Lang Roadmap

> Monk is a compiler. Source → C → native binary.
> No interpreter, no VM, no REPL.
> Methodology: Red-Green-Refactor TDD throughout.

The previous implementation (TypeScript/Bun tree-walking interpreter) is archived at [monk-lang-v1](https://github.com/monkfromearth/monk-lang-v1).

---

## Phase 1: Lexer ✅

Tokenize source code into a stream of tokens.

- [x] Number literals: decimal, hex, binary, octal, underscores
- [x] String literals (double-quoted, escape sequences)
- [x] Template literals (backtick, multiline)
- [x] Identifiers and all keywords
- [x] All operators: arithmetic, comparison, logical, bitwise, assignment
- [x] All delimiters, arrow (`->`)
- [x] Single-line comments (`//`)
- [x] Line and column tracking
- [x] EOF and illegal token handling

**Status:** 112 tests passing. `src/syntax/scanner.go`

---

## Phase 2: Parser ✅

Transform token stream into an Abstract Syntax Tree (AST).

- [x] All expression types: literals, unary, binary, call, index, property, array, record, function, throw
- [x] All statement types: let/const, if/else/else-if, while, for-in, return, break, continue, guard/against, type declarations, use/export
- [x] Operator precedence (13 levels)
- [x] Trailing comma support
- [x] Type annotations on variables
- [x] Error messages with line/column

**Status:** 77 tests passing. `src/syntax/parser.go`, `src/syntax/ast.go`

---

## Phase 3: C Runtime Library ✅

The small C library linked into every compiled Monk program.

- [x] `MonkValue` tagged union (int, float, string, bool, none, array, record, function)
- [x] Value creation helpers (`monk_int`, `monk_string`, etc.)
- [x] `monk_show()` with spec-defined output format
- [x] Deep copy for value semantics
- [x] Truthiness check (false, none, 0 are falsy)
- [x] String operations: length, substring, index_of, split, trim, to_upper, to_lower
- [x] Array operations: append, prepend, pop, drop, take, slice, range (all return new arrays, all clamp)
- [x] Math functions: abs, floor, ceil, round, sqrt, pow, log, log10, exp, min, max, sin, cos, tan, asin, acos, atan
- [x] Conversion: to_string, to_int (strict), to_float
- [x] Type checking: typeof, is_number, is_string, is_boolean, is_array, is_record, is_function, is_none
- [x] File system: file_read, file_write, file_exists
- [x] Environment: env_get, exit, args
- [x] Error handling: setjmp/longjmp infrastructure for guard/against/throw
- [x] Memory: malloc/free wrappers, deep copy functions

**Done when:** `runtime.c` compiles standalone and all functions work in isolation.

---

## Phase 4: C Code Generation ✅

The compiler core. Walk the AST, emit C source code.

### Expressions
- [x] Numeric, string, boolean, none literals → C values
- [x] Arithmetic, comparison, logical, bitwise operators → C operators
- [x] String concatenation (string + string only)
- [x] Unary operators (-x, !x, ~x, not x)
- [x] Array literals → `monk_array_new(...)`
- [x] Record literals → `monk_record_new(...)`
- [x] Property access → field lookup in record struct
- [x] Index access → bounds-checked array/string access
- [x] Function calls → C function calls
- [x] Throw → `longjmp` to nearest guard

### Statements
- [x] Variable declarations → C variable declarations with deep copy
- [x] Assignment (simple + compound) → C assignment with const checking
- [x] Index/property assignment → mutation with const/bounds checking
- [x] If/else/else-if → C if/else
- [x] While loops → C while
- [x] For-in loops → iteration over array/string
- [x] Break/continue → C break/continue
- [x] Return → C return
- [x] Guard/against → setjmp/longjmp pattern
- [x] Block scoping → C block scoping

### Functions
- [x] Function expressions → C function definitions + closure struct
- [x] Closure capture by copy → snapshot environment into struct
- [x] Self-reference for recursion
- [x] Default parameter values

### Output
- [x] Generate valid, compilable `.c` file
- [x] Include `runtime.h` header
- [x] Generate `main()` that runs top-level statements
- [x] `#line` directives mapping back to `.monk` source

**Status:** 49 tests passing. `src/codegen/codegen.go`

---

## Phase 5: CLI ✅

- [x] `monk build <file>` — compile .monk → .c → native binary
- [x] `monk build <file> -o <out.c>` — emit C source (no compilation)
- [x] `monk run <file>` — compile and run in one step (compile, execute, delete temp files)
- [x] `monk check <file>` — parse and validate without compiling
- [x] Error reporting with source file, line, column

**Status:** 39 tests passing. `src/main.go`

---

## Phase 6: Type System ✅

Static analysis pass over the AST, before code generation.

- [x] Type annotations on variables (space-separated syntax)
- [x] Array type annotations (`int[]`, `string[]`)
- [x] Function type signatures: `(int, int) -> int`
- [x] First-assignment type inference
- [x] Type consistency on reassignment
- [x] Optional types (`int?`) — accepts base type or `none`
- [x] Custom type definitions: record types + type aliases (structural)
- [x] Structural typing validation for records
- [x] Element type enforcement in typed arrays
- [x] Function return type validation (every code path)
- [x] Numeric widening: int → float implicit
- [x] Typed record field enforcement (missing field = compile error)
- [x] Equality strictness: `5 == "5"` is a type error
- [x] Collections/functions cannot be compared with `==`
- [x] Loop variable is const
- [x] All-paths-return analysis

**Status:** 112 checker tests + 18 unboxing codegen tests. `src/types/` + `src/codegen/unbox.go`. Wired into `monk build/run/check`.

**Unboxing already delivered in Phase 6:**
- [x] Scalar variables (`int`/`float`/`bool`) stored as raw C types
- [x] Scalar arithmetic emits raw C (no `monk_add` dispatch)
- [x] Scalar function signatures (`static int64_t fib(int64_t n)`)
- [x] Raw conditions in if/while (no `monk_is_truthy`)
- [x] `to_int`/`to_float` inline as C casts when arg is already scalar

**Deferred (follow-up work, not a separate phase):**
- [x] Typed array inline access — `int[]`/`float[]`/`bool[]` element reads/writes emit direct `.int_val` struct-field access instead of `monk_array_get`/`monk_array_set`. Matmul: 11× C → 3× C. `storeIntArray` etc. in `unbox.go`.
- [x] Typed array backing store — back `int[]` with `int64_t*` instead of `MonkValue*`. New `MONK_INT_ARRAY` kind + `MonkIntArray { int64_t* data; int64_t length }` struct in runtime. `monk_int_array_from()` converts/copies. Matmul: 3× C → ~2× C. See `spec/ARCHITECTURE_DECISIONS.md §4A`.
- [ ] Copy-on-write for arrays — share backing storage on assign, copy only on mutation. Makes `let b = a` O(1) instead of O(n). No spec change needed. See `spec/ARCHITECTURE_DECISIONS.md §4B`.
- [ ] Bounds-check elision for typed arrays in provably-safe loops (`for i in range(0, arr.length)`). Requires spec clarification that `int[]` OOB is always a panic (strict, not graceful). See `spec/ARCHITECTURE_DECISIONS.md §4C`.
- [ ] Unboxed for-loop variables over typed iterables
- [ ] Runtime typeof/is_* inlined for known-type values
- [x] Default parameter values — arity-range check in type checker; `padDefaults()` fills defaults at call site in codegen.
- [x] First-class function values in codegen — closures, trampolines, `monk_make_function`, capture-by-copy with save-back.
- [x] Higher-order builtins wired into type checker — `map`, `filter`, `reduce` recognized as builtins; `funcExactMatch` accepts `Any` wildcard for callbacks.
- [x] `abs()` return type — type-preserving (int→int, float→float).
- [x] Hex/binary/octal underscore literals in codegen — stripped during emission.
- [x] Function parameter deep copy — `monk_deep_copy` emitted at entry for each boxed param.

---

## Phase 7: Module System

- [ ] `use X from "./path"` — resolve and load .monk files
- [ ] Named imports, wildcard imports
- [ ] `export` declarations
- [ ] Import with alias (`use X as Y`)
- [ ] Module scope isolation
- [ ] Circular import detection (compile error)
- [ ] Compile multi-file programs to a single .c file (or multiple linked .o files)

---

## Phase 8: C FFI

> Syntax TBD — to be designed before implementation.

- [ ] Design FFI syntax
- [ ] Declare external C functions from Monk
- [ ] Type mapping at FFI boundary
- [ ] Emit `#include` and linker flags

---

## Phase 9: Linter & Formatter

- [ ] `monk lint <file>` — code quality checks
- [ ] `monk format <file>` — code formatting
- [ ] Linter rules: naming conventions, unused variables, const preference, max parameters
- [ ] Formatter: indentation, spacing, trailing commas, newline at EOF

---

## Phase 10: LSP + Editor Support

- [ ] LSP server (diagnostics, completions, hover, go-to-definition)
- [ ] VS Code extension (syntax highlighting, LSP client)

---

## Phase 11: Distribution

- [x] Self-contained binary (runtime embedded via go:embed)
- [x] `make install` to ~/.local/bin
- [ ] Bundle zig cc (eliminate cc/gcc/clang dependency)
- [ ] Cross-compile targets: darwin-arm64, darwin-amd64, linux-arm64, linux-amd64
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Homebrew formula
- [ ] Install script

---

## Future (Not Scoped)

- Memory model: `ref` parameters, borrowing
- Native compilation (LLVM backend)
- Generics / parametric types
- Pattern matching, destructuring
- Async/await
- Package manager and registry

---

**The spec lives at `spec/REFERENCE.md`. The compiler conforms to the spec. The tests prove it.**
