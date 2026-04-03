# Monk Lang Roadmap

> Ground-up rewrite. Same language, new implementation.
> No timelines. Each phase ships when it's solid.
> Methodology: Red-Green-Refactor TDD throughout.

The previous implementation (TypeScript/Bun tree-walking interpreter) is archived at [monk-lang-v1](https://github.com/monkfromearth/monk-lang-v1). This roadmap describes the ground-up rebuild.

---

## Phase 1: Lexer

Tokenize source code into a stream of tokens.

- [ ] Number literals: decimal (`42`, `3.14`, `1.23e5`), hex (`0xFF`), binary (`0b1010`), octal (`0o77`), underscores (`1_000_000`)
- [ ] String literals (double-quoted, escape sequences: `\n`, `\t`, `\"`, `\\`)
- [ ] Template literals (backtick strings, multiline)
- [ ] Identifiers
- [ ] All keywords: `let`, `const`, `if`, `else`, `for`, `in`, `while`, `break`, `continue`, `return`, `guard`, `against`, `throw`, `type`, `use`, `export`, `from`, `as`, `is`, `not`, `and`, `or`, `true`, `false`, `none`
- [ ] Reserved keywords: `ref`, `async`, `await`
- [ ] Arithmetic operators: `+`, `-`, `*`, `/`, `%`
- [ ] Assignment operators: `=`, `+=`, `-=`, `*=`, `/=`, `%=`
- [ ] Comparison operators: `==`, `!=`, `<`, `>`, `<=`, `>=`
- [ ] Symbolic logical operators: `&&`, `||`, `!`
- [ ] Bitwise operators: `&`, `|`, `^`, `~`, `<<`, `>>`
- [ ] Delimiters: `(`, `)`, `{`, `}`, `[`, `]`, `:`, `,`, `.`, `?`
- [ ] Arrow: `->`
- [ ] Single-line comments: `//`
- [ ] Newline handling (insignificant, with continuation heuristics)
- [ ] Trailing comma support
- [ ] Line and column tracking
- [ ] EOF token
- [ ] Error recovery for invalid tokens

**Done when:** Every token type from the spec is lexed correctly with position info. All lexer tests green.

---

## Phase 2: Parser

Transform token stream into an Abstract Syntax Tree (AST).

### Expressions
- [ ] Numeric, string, boolean, none literals
- [ ] Identifier expressions
- [ ] Unary expressions (`-`, `not`, `!`, `~`)
- [ ] Binary expressions with full operator precedence (13 levels)
- [ ] Parenthesized grouping
- [ ] Array literals (with trailing commas)
- [ ] Record literals (with trailing commas)
- [ ] Property access (`obj.prop`)
- [ ] Index access (`arr[i]`, `str[i]`)
- [ ] Function expressions (typed params, optional defaults, return type, block body)
- [ ] Call expressions (with trailing commas)
- [ ] Throw expressions
- [ ] Template literal expressions (multiline)

### Statements
- [ ] Variable declarations (`let`/`const`, optional type annotation)
- [ ] Assignment statements (NOT expressions — forbidden in conditions)
- [ ] Compound assignment (`+=`, `-=`, `*=`, `/=`, `%=`)
- [ ] Index assignment (`arr[i] = val`)
- [ ] Property assignment (`record.field = val`)
- [ ] If/else statements (including else-if chains)
- [ ] While loops
- [ ] For-in loops (arrays and strings only — no bare numbers)
- [ ] Break / continue statements
- [ ] Return statements
- [ ] Block statements (scoped `{ }`)
- [ ] Guard/against statements
- [ ] Type declarations (`type Name = ...` — record types, aliases, function types)
- [ ] Use/export statements

### Parser Quality
- [ ] Clear error messages with line/column
- [ ] Error recovery (don't stop at first error)
- [ ] Reject assignment in conditions
- [ ] Reject `break`/`continue` outside loops

**Done when:** Every syntactic construct from the spec parses into the correct AST. All parser tests green.

---

## Phase 3: Interpreter (Core Runtime)

Evaluate the AST. This is the heart of the language.

### Values and Evaluation
- [ ] Evaluate literals: int, float, string, boolean, none
- [ ] Arithmetic operations — numeric only, int/int=int (truncating), float involved=float
- [ ] String concatenation via `+` (string + string only, no auto-coercion)
- [ ] Comparison: `==`/`!=` on same-type primitives only (cross-type = error, collections = error)
- [ ] Ordering: `<`/`>`/`<=`/`>=` on numbers and strings (lexicographic). Cross-type = error. `none` = error.
- [ ] Logical operations with truthiness (`false`, `none`, `0` are falsy) — always return `boolean`
- [ ] Unary: `-` (numeric), `not`/`!` (truthiness), `~` (bitwise int)
- [ ] Bitwise operations: `&`, `|`, `^`, `<<`, `>>` (int only, two's complement)

### Variables and Scope
- [ ] Variable declaration and lookup (`let`/`const`)
- [ ] Assignment and compound assignment (`+=`, etc.)
- [ ] Deep const enforcement (no reassign, no element/field mutation)
- [ ] `let` = fully mutable (variable + contents)
- [ ] Value semantics: assignment copies arrays and records
- [ ] Scope chain: nested scopes, parent lookup
- [ ] Variable shadowing
- [ ] Block scope creation

### Control Flow
- [ ] If/else evaluation (truthiness-based conditions)
- [ ] While loops
- [ ] For-in loops: arrays and strings. Loop variable is `const`.
- [ ] Break signal (exits loop)
- [ ] Continue signal (skips to next iteration)

### Functions
- [ ] Function creation with closure capture (closures capture by copy, like C++ `[x]` lambdas)
- [ ] Function calls — arguments are copies (value semantics)
- [ ] Default parameter values
- [ ] Return signal
- [ ] Recursion (function name in scope inside own body)
- [ ] Higher-order functions (pass/return functions)
- [ ] Every code path must return matching type (enforce at runtime for now)

### Data Structures
- [ ] Array creation and indexing: read out-of-bounds → `none`, write out-of-bounds → error
- [ ] String indexing: `"hello"[0]` → `"h"`, out-of-bounds → `none`
- [ ] Record creation and property access (dot notation)
- [ ] Record shape is fixed at creation — no adding new fields
- [ ] Typed record: missing field read/write → error
- [ ] Untyped record: missing field read → `none`, missing field write → error
- [ ] Index assignment (`arr[i] = val`) on `let` arrays only
- [ ] Property assignment (`record.field = val`) on `let` records only

### Error Handling
- [ ] Throw expression (any value)
- [ ] Guard/against evaluation: declares variable in enclosing scope, defaults to `none` if unassigned
- [ ] `guard` with non-throwing expression is valid (against block = dead code)
- [ ] `break`/`continue`/`return` work inside `against` blocks
- [ ] Error propagation through call stack
- [ ] Unhandled `throw` at top level terminates program with error message

**Done when:** Every runtime behavior from the spec works. All interpreter tests green.

---

## Phase 4: Built-in Functions

Register native functions in the global scope.

### Output & Conversion
- [ ] `show` — with defined output format (JSON-ish: strings quoted inside collections, `<function>`)
- [ ] `to_string` — same format as `show`, returns string
- [ ] `to_int` — string→int only, strict (rejects "3.14"), throws on failure
- [ ] `to_float` — string→float, throws on failure

### Math
- [ ] `abs`, `floor`, `ceil`, `round`
- [ ] `sqrt` (throws if x < 0), `pow`, `log` (throws if x <= 0), `log10` (throws if x <= 0), `exp`
- [ ] `min`, `max`

### Trigonometry
- [ ] `sin`, `cos`, `tan`
- [ ] `asin`, `acos`, `atan`

### String
- [ ] `length` (overloaded: string, array, record)
- [ ] `substring` (indices clamp), `index_of`
- [ ] `split`, `trim`
- [ ] `to_upper_case`, `to_lower_case`

### Array
- [ ] `append`, `prepend`
- [ ] `pop` (`pop([])` = `[]`)
- [ ] `drop(arr, n=1)`, `take(arr, n=1)` — both clamp
- [ ] `slice` (indices clamp)
- [ ] `map`, `filter`, `reduce` (`reduce([], fn, x)` = `x`)
- [ ] `range` (`range(0)` = `[]`, `range(-5)` = `[]`)

### Type Checking
- [ ] `typeof` (returns base type strings: "int", "float", "string", "boolean", "none", "array", "record", "function")
- [ ] `is_number`, `is_string`, `is_boolean`
- [ ] `is_array`, `is_record`, `is_function`, `is_none`

### File System & Environment
- [ ] `file_read`, `file_write`, `file_exists`
- [ ] `env_get`, `exit`, `args`

**Done when:** Every built-in from the spec works with correct signatures. All built-in tests green.

---

## Phase 5: Type System

Static analysis pass over the AST, before or during evaluation.

- [ ] Type annotations on variables (space-separated syntax)
- [ ] Array type annotations (`int[]`, `string[]`)
- [ ] Function type signatures: `(int, int) -> int`
- [ ] Function parameter + return type enforcement
- [ ] First-assignment type inference (lock type on first assign)
- [ ] Type consistency on reassignment
- [ ] Optional types (`int?`, `string?`) — accepts base type or `none`. `int??` is invalid.
- [ ] Custom type definitions: record types + type aliases (structural, not nominal)
- [ ] Structural typing validation for records
- [ ] Element type enforcement in typed arrays
- [ ] Function return type validation (every code path)
- [ ] Definite assignment analysis (no reads before writes)
- [ ] Numeric widening: `int` → `float` implicit, reverse requires explicit conversion
- [ ] Empty array `[]` type inference from context
- [ ] Empty record `{}` handling

**Done when:** Every type rule from the spec is enforced. All type system tests green.

---

## Phase 6: Module System

- [ ] `use X from "./path"` — resolve and load `.monk` files
- [ ] `use { X, Y } from "./path"` — named imports
- [ ] `use * from "./path"` — wildcard imports
- [ ] `export` declarations (functions, constants, types)
- [ ] Import with alias (`use X as Y`)
- [ ] Module scope isolation
- [ ] Module-level code executes once on first import
- [ ] Circular import detection (compile error)

**Done when:** Modules can import/export functions, constants, and types across files.

---

## Phase 7: C FFI

> Syntax TBD — to be designed before implementation.

- [ ] Design FFI syntax (discuss options, pick one)
- [ ] Declare external C functions from Monk
- [ ] Type mapping at FFI boundary (int → int64_t, float → double, string → const char*, bool → bool)
- [ ] Emit `#include` directives in generated C
- [ ] Library linking flags passed to cc
- [ ] Compile-time type checking of extern call sites

**Done when:** Monk programs can call C standard library functions and link external C libraries.

---

## Phase 8: CLI

- [ ] `monk build <file>` — compile a `.monk` file to native binary
- [ ] `monk run <file>` — compile and run in one step (cache binary, recompile on change)
- [ ] `monk check <file>` — type check without compiling
- [ ] `monk lint <file>` — code quality checks
- [ ] `monk format <file>` — code formatting

### Linter Rules
- [ ] `snake_case_variables`
- [ ] `pascal_case_types`
- [ ] `const_case_constants`
- [ ] `descriptive_names`
- [ ] `consistent_indentation`
- [ ] `trailing_whitespace`
- [ ] `explicit_types_public`
- [ ] `consistent_optional_syntax`
- [ ] `max_parameters`
- [ ] `max_function_length`
- [ ] `const_vs_let_preference`

### Formatter
- [ ] Indentation normalization
- [ ] Operator spacing
- [ ] Brace style
- [ ] Trailing commas (multi-line)
- [ ] Newline at EOF
- [ ] Trailing whitespace removal
- [ ] Max consecutive empty lines

**Done when:** All CLI commands work. Linter and formatter match v1 behavior.

---

## Phase 9: LSP + Editor Support

- [ ] LSP server with TextDocument sync
- [ ] Completions (keywords, built-ins, scope variables)
- [ ] Hover (type info, signatures)
- [ ] Go to definition
- [ ] Find references
- [ ] Rename
- [ ] Signature help
- [ ] Document formatting
- [ ] Diagnostics (errors, warnings)
- [ ] VS Code extension (syntax highlighting, LSP client)

**Done when:** The VS Code extension provides a productive editing experience.

---

## Phase 10: Distribution

- [ ] Single-binary packaging
- [ ] Homebrew formula (update `homebrew-monk-lang` repo)
- [ ] Install script
- [ ] Docker image
- [ ] CI/CD pipeline

---

## Future (Not Scoped)

These are on the radar but not part of the current rebuild:

- Memory model finalization (`ref` parameters, borrowing, reference counting)
- Native compilation (LLVM backend or similar)
- Generics / parametric types
- Pattern matching
- Destructuring
- Async/await
- File I/O and system integration
- Standard library expansion
- Package manager and registry

---

**The spec lives at `spec/REFERENCE.md`. The implementation conforms to the spec. The tests prove it. This roadmap tracks what's done.**
