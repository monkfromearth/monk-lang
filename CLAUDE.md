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

**Phases 1-5 complete.** Monk is a working compiler. 446 tests total. `monk build` compiles .monk to native binaries. `monk run` compiles and runs. `monk check` validates. Next step: Phase 6 (type system).

**Interpreter/builtins deleted.** We built a tree-walking interpreter (Phase 3) and Go builtins (Phase 4) before realizing: Monk is a compiler, not an interpreter. There is no REPL. Those packages don't ship. They were deleted. The language semantics they validated will be re-tested as integration tests (compile .monk → run binary → check output).

Remaining parser items (deferred):
- Default parameter values (parser handles syntax, codegen will use them)
- Function type signatures `(int, int) -> int` in parameter position
- Reject assignment in conditions (added to parser)
- Reject break/continue outside loops (added to parser)

Go 1.22+ required. Standard `go test` for TDD.

## Project Structure

```
spec/                    — Language specification
  REFERENCE.md               — THE source of truth for syntax and semantics
  ARCHITECTURE_DECISIONS.md  — Why Go, why compile-to-C, what can change later
  MEMORY_MODEL_DISCUSSION.md — Design history (contains rejected options, read only if needed)
knowledge/               — Learning course (Brilliant.org style, Astro + Solid)
src/                     — Go compiler
  syntax/                    — Lexer + Parser + AST (Phases 1-2, complete)
    token.go                     token types and keyword lookup
    scanner.go                   lexer: source text → tokens
    scanner_test.go              100 lexer tests
    ast.go                       29 AST node types
    parser.go                    recursive descent parser with 13-level precedence
    parser_test.go               119 parser tests
  codegen/                   — C code generator (Phase 4, complete)
    codegen.go                   AST → C source emitter
    codegen_test.go              35 integration tests (compile + run)
  cmd/monk/                  — CLI (Phase 5, complete)
    main.go                      monk build/run/check/version/help
    main_test.go                 28 CLI tests
  monk.go                    — package-level documentation
runtime/                 — C runtime library (Phase 3, not yet created)
  runtime.h                  — header for generated C to include
  runtime.c                  — MonkValue, builtins, error handling
examples/                — 9 working .monk programs (hello, fibonacci, fizzbuzz,
                           error_handling, arrays, newton_sqrt, todo_list, collatz, sort)
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

1. Lexer ✅
2. Parser ✅
3. C Runtime Library ✅
4. C Code Generation ✅
5. CLI ✅ (monk build, monk run, monk check, --emit-c)
6. Type System (static analysis)
7. Module System
8. C FFI (syntax TBD)
9. Linter & Formatter
10. LSP + Editor
11. Distribution

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

## Workflow Rules (learned from building Phases 1-5)

### Git Workflow
- **Every phase gets its own branch.** `phase-N-name` branched from main.
- **CodeRabbit review before merging.** Run `coderabbit review --plain -t committed --base main` on each branch.
- **Fix ALL CodeRabbit findings** before merging. Don't skip any.
- **Merge with `--no-ff`** so the merge commit is visible in history.
- **Future: use PRs** instead of direct merges. Current direct-merge workflow is for first iteration speed only.

### Code Quality
- **Test before code.** Write the failing test first (Red), then the code (Green), then clean up (Refactor).
- **Integration tests for codegen.** Codegen tests should compile AND run the generated binary, checking stdout. Not just "does the C look right."
- **Design decision comments.** Every non-obvious choice in the code must have a comment linking back to the spec section.
- **Update docs EVERY phase.** ROADMAP checkboxes, CLAUDE.md status, file tree. Not after — during the same commit.

### Mistakes Made (don't repeat)
- **Built a tree-walking interpreter** (Phases 3-4 originally) before realizing Monk is a compiler with no REPL. 2,792 lines deleted. Lesson: read the architecture doc before writing code.
- **Chose Zig, then switched to Go.** The compiler is a text-in/text-out translator — Go's tree manipulation and strings are right for this. Zig/Rust are for the future native backend.
- **AssignExpr instead of AssignStmt.** Assignment was an expression (interpreter thinking). Fixed: assignment is a statement in both the spec and the code.
- **AST nodes had no position info.** Could not emit #line directives for source mapping. Fixed: every node embeds Pos{Line, Column}.
- **Use-after-free in codegen.** `monk_free(old); old = monk_deep_copy(expr_using_old)` — freed before computing the new value. Fixed: compute first, then free.
- **Made creative decisions without asking.** Sanskrit release names, FFI syntax, etc. Lesson: always present options for naming/branding/creative decisions.

### Architecture Lessons
- **Monk is a compiler.** Source → C → binary. No interpreter, no VM, no REPL.
- **The C runtime is the foundation.** Generated code calls runtime functions. The runtime must be complete before codegen starts.
- **Builtins are C functions,** not Go functions. The Go compiler generates calls to them; it doesn't implement them.
- **Functions are hoisted** as static C functions above main(). Called directly by name, not through function pointers (closures deferred).
- **`-o` flag controls output format.** `-o hello` = binary. `-o hello.c` = C source. Extension decides.

## Related Repos

| Repo | Purpose | Status |
|------|---------|--------|
| `monkfromearth/monk-lang` | This repo — the compiler | Active |
| `monkfromearth/monk-lang-v1` | Archived v1 (TS/Bun interpreter) | Archived |
| `monkfromearth/homebrew-monk-lang` | Homebrew tap formula | Update in Phase 9 |
| `monkfromearth/monklore` | Docs site (Next.js) | Separate |
| `projects/monk-examples` | Starter examples (not a git repo) | Absorb later |
