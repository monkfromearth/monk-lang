# Monk Lang

## What This Is

Ground-up rewrite of [Monk Lang](https://github.com/monkfromearth/monk-lang). Same language, new implementation.
The v1 (TypeScript/Bun tree-walking interpreter) is archived at [monk-lang-v1](https://github.com/monkfromearth/monk-lang-v1).

## Single Source of Truth

- **Language spec:** `spec/REFERENCE.md` — defines what Monk IS. If the code disagrees with the spec, one of them has a bug.
- **Roadmap:** `ROADMAP.md` — phased rebuild plan with checkboxes. No timelines.
- **Architecture decisions:** `spec/ARCHITECTURE_DECISIONS.md` — why Go, why compile-to-C, what can change later.
- **Memory model discussion:** `spec/MEMORY_MODEL_DISCUSSION.md` — ongoing design for ref/borrowing/memory.
- **This file:** project rules and decisions.

Never hallucinate syntax or behavior. Always check `spec/REFERENCE.md` before writing code or tests.

## Design Philosophy (from spec)

1. **Explicit over implicit.** No hidden coercion, no hidden errors, no hidden mutation. One exception: truthiness (`false`, `none`, `0` are falsy).
2. **Graceful on reads, strict on operations.** Reading missing data = `none` or `[]`. Operating on invalid data = error.
3. **Values, not references.** Assignment copies. Function args copy. Closures copy. Your data is yours. No exceptions.

When in doubt, apply these rules. If a proposed behavior violates them, the spec has a bug.

## Methodology

**Red-Green-Refactor TDD. No exceptions.**

1. Write a failing test (Red)
2. Write the minimum code to pass it (Green)
3. Refactor while keeping tests green (Refactor)

No code without a test. No test without a spec reference.

## Key Decisions

- **Implementation language:** Go. Chosen for fast builds (TDD), easy tree manipulation, first-class strings, single-binary output. Zig/Rust deferred to future backend work (LLVM, custom codegen).
- **Execution model:** Compile to C. Monk is a compiler, not an interpreter. `monk build` → native binary. `monk run` compiles and runs in one step.
- **Runtime:** Small C library (~2-5 KB) linked into every compiled program. Provides MonkValue, built-in functions, error handling.
- **Architecture restart:** Clean break from v1. No code carried over.
- **v1 archive:** `github.com/monkfromearth/monk-lang-v1` (renamed from monk-lang)
- **monk-examples:** Will be absorbed into `examples/` when ready. Currently separate at `projects/monk-examples`.
- **homebrew-monk-lang:** Separate repo (Homebrew requires `homebrew-*` naming). Formula updated in Phase 9.
- **monklore:** Docs site stays as separate repo. Independent deploy cycle.
- **Memory model:** Pure value semantics. Eager deep copy on assignment. Zero refcounting. Closures capture by copy. `ref` keyword reserved but deferred.
- **Value semantics:** Assignment, function args, and closures ALL copy. No exceptions. Zero refcounting.
- **Mixed arrays dropped:** All arrays must be homogeneous. Use records for mixed data.
- **Error handling impl:** setjmp/longjmp for guard/against/throw. Simple, correct, no per-call-site analysis needed.
- **Deep const:** `const` freezes variable AND contents. `let` = fully mutable.
- **Records are shapes:** Fixed set of fields from creation. Cannot add new fields.
- **No `is not`:** Dropped. Use `!=`.
- **No `function` type:** Dropped. Use typed signatures `(int, int) -> int`.
- **Truthiness:** `false`, `none`, `0` only. `""` and `[]` are truthy.

## Versioning and Releases

**Semantic versioning:** `MAJOR.MINOR.PATCH` (e.g., `0.1.0`, `0.2.0`, `1.0.0`)

- `0.x.y` — pre-1.0, breaking changes expected
- `1.0.0` — first stable release (language spec is frozen)

**Release names:** Each minor version gets a codename. Convention: **Urdu/Hindi single words** — personal, distinctive, meaningful.

Names are chosen when the release is ready. The word should reflect what the release achieves.

Word bank (pick when shipping, not planned ahead):
- *Safar* (journey) · *Noor* (light) · *Umeed* (hope)
- *Fikr* (thought) · *Sukoon* (peace) · *Irada* (will, intent)
- *Khoj* (search, discovery) · *Raasta* (path) · *Buniyaad* (foundation)
- *Dastak* (knock, arrival) · *Ehsaas* (feeling, awareness) · *Amal* (action)
- *Soch* (thought) · *Zariya* (means, medium) · *Wujood* (existence)

## Current Status

**Phases 1-4 complete.** Lexer, parser, interpreter, and 48 built-in functions working with 445 passing tests. Next step: Phase 5 (type system).

Remaining items from earlier phases (deferred):
- Default parameter values (parser + interpreter)
- Typed record field enforcement (requires type checker, Phase 5)
- Return type validation (requires type checker, Phase 5)
- Reject assignment in conditions (parser validation)
- Reject break/continue outside loops (parser validation)
- Function type signatures `(int, int) -> int` in parameter position (parser)
- File system builtins: file_read, file_write, file_exists (Phase 4 remainder)
- Environment builtins: env_get, exit, args (Phase 4 remainder)
- map/filter/reduce with user-defined Monk functions (needs evaluator integration)

Go 1.22+ required. Standard `go test` for TDD.

## Project Structure

```
spec/                    — Language specification
  REFERENCE.md               — THE source of truth for syntax and semantics
  ARCHITECTURE_DECISIONS.md  — Why Go, why compile-to-C, what can change later
  MEMORY_MODEL_DISCUSSION.md — Design history (contains rejected options, read only if needed)
knowledge/               — Learning course (Brilliant.org style, Astro + Solid)
  ASTRO_MIGRATION_PROMPT.md  — Full prompt for Astro migration work
src/                     — Go implementation
  syntax/                    — Lexer + Parser + AST (Phase 1-2, complete)
    token.go                     token types and keyword lookup
    scanner.go                   lexer: source text → tokens
    scanner_test.go              100 lexer tests
    ast.go                       29 AST node types
    parser.go                    recursive descent parser with 13-level precedence
    parser_test.go               119 parser tests
  interpreter/               — Tree-walking evaluator (Phase 3, complete)
    value.go                     MonkValue, DeepCopy, IsTruthy, String
    env.go                       Environment (scope chain, const enforcement)
    eval.go                      evaluator: AST → values
    eval_test.go                 144 interpreter tests
  cmd/monk/                  — CLI entry point (stubs)
    main.go
  monk.go                    — package-level documentation
runtime/                 — C runtime library (not yet created)
examples/                — .monk example programs
go.mod                   — github.com/monkfromearth/monk-lang
```

## Rules

- Spec-first: update `spec/REFERENCE.md` before implementing new features.
- Tests-first: write failing tests before writing implementation code.
- Check the ROADMAP: mark items done as you complete them.
- Don't add features not in the spec. If you want a new feature, add it to the spec first.
- Keep tests fast. If a test needs I/O or network, it's an integration test and should be marked as such.
- Philosophy-first: if an edge case isn't in the spec, resolve it using the three rules before deciding.

## Build Phase Order

Phases must be completed in order (each depends on the one before):

1. Lexer
2. Parser
3. Interpreter (core runtime)
4. Built-in functions
5. Type system
6. Module system
7. C FFI
8. CLI (build, run, lint, format, check)
9. LSP + editor
10. Distribution

Memory model / `ref` / borrowing will be inserted when the design is finalized.

## Reading Order for New Sessions

When starting a new session on this project, read files in this order:
1. **This file** (CLAUDE.md) — decisions, rules, current status
2. **spec/REFERENCE.md** — the language spec (read before writing ANY code or tests)
3. **ROADMAP.md** — what to build next (check the checkboxes)
4. **spec/ARCHITECTURE_DECISIONS.md** — only if working on codegen, runtime, or infrastructure

Do NOT read MEMORY_MODEL_DISCUSSION.md unless specifically discussing memory design — it contains historical options that were rejected.

## Design Constraints (knowledge/ pages)

If working on the learning course (`knowledge/`):
- NO emojis anywhere. Use SVG icons (Heroicons outline).
- NO purple, violet, or indigo colors.
- Dark code blocks (Shiki, `#1C1917` background, monk-dark theme in `astro.config.mjs`).
- See `knowledge/ASTRO_MIGRATION_PROMPT.md` for full design system.

## Keeping knowledge/ in sync

The learning course (`knowledge/`) teaches how the compiler is built. When any of the following change, update the affected knowledge pages:
- **spec/REFERENCE.md** — token set, syntax, semantics changes affect lessons 1.2 (Monk's Token Set) and any lesson referencing Monk syntax.
- **spec/ARCHITECTURE_DECISIONS.md** — changes to implementation language, execution model, or project structure affect lessons 0.1 (How Compilers Work), 0.2 (Setting Up), and the course map (index).
- **Build phase order** — adding/removing/reordering phases affects the course map and lesson navigation links.
- **Key decisions** — any decision change (e.g. error handling strategy, type system rules) that contradicts content in a lesson must be fixed in the lesson.

Rule: if you change the spec or architecture and a knowledge/ page now says something wrong, fix the page in the same commit.

## Related Repos

| Repo | Purpose | Status |
|------|---------|--------|
| `monkfromearth/monk-lang` | This repo — the compiler | Active |
| `monkfromearth/monk-lang-v1` | Archived v1 (TS/Bun interpreter) | Archived |
| `monkfromearth/homebrew-monk-lang` | Homebrew tap formula | Update in Phase 9 |
| `monkfromearth/monklore` | Docs site (Next.js) | Separate |
| `projects/monk-examples` | Starter examples (not a git repo) | Absorb later |
