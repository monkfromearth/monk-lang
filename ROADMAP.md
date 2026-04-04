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

**Status:** 100 tests passing. `src/syntax/scanner.go`

---

## Phase 2: Parser ✅

Transform token stream into an Abstract Syntax Tree (AST).

- [x] All expression types: literals, unary, binary, call, index, property, array, record, function, throw
- [x] All statement types: let/const, if/else/else-if, while, for-in, return, break, continue, guard/against, type declarations, use/export
- [x] Operator precedence (13 levels)
- [x] Trailing comma support
- [x] Type annotations on variables
- [x] Error messages with line/column

**Status:** 119 tests passing. `src/syntax/parser.go`, `src/syntax/ast.go`

---

## Phase 3: C Runtime Library

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

## Phase 4: C Code Generation

The compiler core. Walk the AST, emit C source code.

### Expressions
- [ ] Numeric, string, boolean, none literals → C values
- [ ] Arithmetic, comparison, logical, bitwise operators → C operators
- [ ] String concatenation (string + string only)
- [ ] Unary operators (-x, !x, ~x, not x)
- [ ] Array literals → `monk_array_new(...)`
- [ ] Record literals → `monk_record_new(...)`
- [ ] Property access → field lookup in record struct
- [ ] Index access → bounds-checked array/string access
- [ ] Function calls → C function calls
- [ ] Throw → `longjmp` to nearest guard

### Statements
- [ ] Variable declarations → C variable declarations with deep copy
- [ ] Assignment (simple + compound) → C assignment with const checking
- [ ] Index/property assignment → mutation with const/bounds checking
- [ ] If/else/else-if → C if/else
- [ ] While loops → C while
- [ ] For-in loops → iteration over array/string
- [ ] Break/continue → C break/continue
- [ ] Return → C return
- [ ] Guard/against → setjmp/longjmp pattern
- [ ] Block scoping → C block scoping

### Functions
- [ ] Function expressions → C function definitions + closure struct
- [ ] Closure capture by copy → snapshot environment into struct
- [ ] Self-reference for recursion
- [ ] Default parameter values

### Output
- [ ] Generate valid, compilable `.c` file
- [ ] Include `runtime.h` header
- [ ] Generate `main()` that runs top-level statements
- [ ] `#line` directives mapping back to `.monk` source

**Done when:** `monk build hello.monk` produces `hello.c` that compiles with `cc` to a working binary.

---

## Phase 5: CLI

- [x] `monk build <file>` — compile .monk → .c → native binary
- [x] `monk run <file>` — compile and run in one step (compile, execute, delete temp files)
- [x] `monk check <file>` — parse and validate without compiling
- [x] Error reporting with source file, line, column

**Done when:** You can write a .monk file and run it with `monk run hello.monk`.

---

## Phase 6: Type System

Static analysis pass over the AST, before code generation.

- [ ] Type annotations on variables (space-separated syntax)
- [ ] Array type annotations (`int[]`, `string[]`)
- [ ] Function type signatures: `(int, int) -> int`
- [ ] First-assignment type inference
- [ ] Type consistency on reassignment
- [ ] Optional types (`int?`) — accepts base type or `none`
- [ ] Custom type definitions: record types + type aliases (structural)
- [x] Structural typing validation for records
- [x] Element type enforcement in typed arrays
- [x] Function return type validation (every code path)
- [x] Definite assignment analysis
- [x] Numeric widening: int → float implicit
- [x] Typed record field enforcement (missing field = compile error)

**Done when:** Type errors are caught at compile time, not at runtime.

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

- [ ] Single-binary packaging for the compiler
- [ ] Homebrew formula
- [ ] Install script
- [ ] CI/CD pipeline

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
