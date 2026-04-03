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
- **Execution model:** Compile to C. Monk is a compiler, not an interpreter. `monk build` → native binary. REPL uses tree-walking for interactivity.
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

**Release names:** Each minor version gets a codename. Convention: **Sanskrit/Pali words related to clarity, knowledge, and simplicity** — fitting for a language called Monk.

Examples:
- `0.1.0` — *"Bodhi"* (awakening)
- `0.2.0` — *"Dharma"* (truth, natural law)
- `0.3.0` — *"Prajna"* (wisdom, insight)
- `0.4.0` — *"Karuna"* (compassion)
- `0.5.0` — *"Shunyata"* (emptiness, simplicity)
- `1.0.0` — *"Nirvana"* (liberation)

Names are chosen when the release is ready, not planned ahead. The name should reflect what the release achieves.

## Current Status

**No code written yet.** The spec, architecture, and learning materials are complete. Next step: set up the Go project and write the first failing lexer test.

Go 1.22+ required. Standard `go test` for TDD.

## Project Structure

```
spec/                — Language specification
  REFERENCE.md           — THE source of truth for syntax and semantics
  ARCHITECTURE_DECISIONS.md — Why Go, why compile-to-C, implementation plans
  MEMORY_MODEL_DISCUSSION.md — Design history for memory management
knowledge/           — Learning course (Brilliant.org style, migrating to Astro)
  ASTRO_MIGRATION_PROMPT.md — Full prompt for migrating to Astro + Solid
  index.html             — Course map landing page
  foundations/           — Unit 0 lessons (how compilers work, Go/C basics)
tests/               — Test suite (contract for the implementation)
examples/            — Example .monk programs
src/                 — Go implementation (not yet created)
runtime/             — C runtime library (not yet created)
editor/              — VS Code extension, LSP
scripts/             — Build and automation
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
7. CLI (run, repl, lint, format, check)
8. LSP + editor
9. Distribution

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
- Light-theme code blocks only (bg: `#FAFAF9`, border: `#E8E4DF`).
- See `knowledge/ASTRO_MIGRATION_PROMPT.md` for full design system.

## Related Repos

| Repo | Purpose | Status |
|------|---------|--------|
| `monkfromearth/monk-lang` | This repo — the compiler | Active |
| `monkfromearth/monk-lang-v1` | Archived v1 (TS/Bun interpreter) | Archived |
| `monkfromearth/homebrew-monk-lang` | Homebrew tap formula | Update in Phase 9 |
| `monkfromearth/monklore` | Docs site (Next.js) | Separate |
| `projects/monk-examples` | Starter examples (not a git repo) | Absorb later |
