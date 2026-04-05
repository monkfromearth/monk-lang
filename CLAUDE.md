# Monk Lang

## Rules

All guidelines are organized in `.claude/rules/`:

- @.claude/rules/persona.md — Conversation style, behavior, concision
- @.claude/rules/code-review-workflow.md — How to process external review feedback
- @.claude/rules/pre-completion-checks.md — **MANDATORY** check battery before declaring work done

Ground-up rewrite of [Monk Lang](https://github.com/monkfromearth/monk-lang). Same language, new implementation.
The v1 (TypeScript/Bun tree-walking interpreter) is archived at [monk-lang-v1](https://github.com/monkfromearth/monk-lang-v1).

## Single Source of Truth

- **Language spec:** `spec/REFERENCE.md` — defines what Monk IS. If the code disagrees with the spec, one of them has a bug.
- **Roadmap:** `ROADMAP.md` — phased rebuild plan with checkboxes. No timelines.
- **Architecture decisions:** `spec/ARCHITECTURE_DECISIONS.md` — why Go, why compile-to-C, what can change later.
- **Progress:** `PROGRESS.md` — what's been built, pivots, mistakes, what's next. **Always update this after meaningful work.**

Never hallucinate syntax or behavior. Always check `spec/REFERENCE.md` before writing code or tests.

## Design Philosophy (from spec)

1. **Explicit over implicit.** No hidden coercion, no hidden errors, no hidden mutation. One exception: truthiness (`false`, `none`, `0` are falsy).
2. **Graceful on reads, strict on operations.** Reading missing data = `none` or `[]`. Operating on invalid data = error.
3. **Values, not references.** Assignment copies. Function args copy. Closures copy. Your data is yours. No exceptions.

When in doubt, apply these rules. If a proposed behavior violates them, the spec has a bug.

## Reading Order for New Sessions

1. **This file** (CLAUDE.md) — rules, philosophy, how to work
2. **PROGRESS.md** — current status, what exists, what's next
3. **spec/REFERENCE.md** — the language spec (read before writing ANY code or tests)
4. **ROADMAP.md** — what to build next (check the checkboxes)
5. **spec/ARCHITECTURE_DECISIONS.md** — only if working on codegen, runtime, or infrastructure

Do NOT read `spec/MEMORY_MODEL_DISCUSSION.md` unless specifically discussing memory design — it contains rejected options.

## Rules

### Development
- **Spec-first.** Update `spec/REFERENCE.md` before implementing new features.
- **Tests-first.** Red-Green-Refactor TDD. No exceptions. No code without a test.
- **Philosophy-first.** If an edge case isn't in the spec, resolve it using the three design rules.
- Don't add features not in the spec. If you want a new feature, add it to the spec first.

### Documentation
- **Always update `PROGRESS.md`** after completing work. This is the project's memory. If it's not in PROGRESS.md, it didn't happen.
- Update ROADMAP checkboxes when phases complete.
- If a spec or architecture change contradicts a knowledge/ page, fix the page in the same commit.
- Don't update THIS file (CLAUDE.md) unless rules, philosophy, or workflow guidance changes. Status and history go in PROGRESS.md.

### Git Workflow
- **Every phase gets its own branch.** `phase-N-name` branched from main.
- **CodeRabbit review before merging.** Run `coderabbit review --plain -t committed --base main` on each branch.
- **Fix ALL CodeRabbit findings** before merging.
- **Merge with `--no-ff`** so the merge commit is visible in history.

### Code Quality
- **Integration tests for codegen.** Compile AND run the generated binary, check stdout.
- **Design decision comments.** Non-obvious choices must have a comment linking to the spec.

### File Organization & INDEX.md
Every source directory (`src/syntax/`, `src/codegen/`, `src/runtime/`) has an `INDEX.md` listing each file and what lives in it. This is a navigation aid, not a spec.

- **Read before searching.** When looking for a function or type, open the directory's `INDEX.md` first. It's faster than grep for "where does this live".
- **Update when files move.** If you add a new file, delete a file, or move a function from one file to another, update the dir's `INDEX.md` in the same commit.
- **Keep it one-line-per-file.** Each row: filename → what it contains. Don't duplicate doc comments.
- **Keep files small and focused.** If a single file exceeds ~500 lines, consider splitting. One topic per file. Go makes this free — same package, multiple files.
- **Two sync points for the C runtime.** The list of runtime `.c` files is duplicated in: (1) `src/embed.go` `//go:embed` directives, (2) `runtimeSources`/`embeddedRuntimeFiles` in `src/main.go`, (3) `runtimeTestSources` in `src/codegen/codegen_test.go`. When you add a runtime file, update ALL THREE plus `runtime/INDEX.md`.

## Key Decisions

- **Compile to C.** No interpreter, no VM, no REPL. Source → C → binary.
- **Go for the compiler.** Fast TDD builds, good tree manipulation, single-binary output.
- **Small C runtime** (~2-5 KB) linked into every binary. MonkValue tagged union, builtins, setjmp/longjmp error handling.
- **Pure value semantics.** Deep copy on assignment. No refcounting, no GC. `ref` keyword reserved but deferred.
- **Functions are hoisted** as static C functions above main(). Direct calls, not function pointers.
- **`-o` flag controls output.** `-o hello` = binary. `-o hello.c` = C source. Extension decides.
- **Homogeneous arrays only.** Use records for mixed data.
- **Deep const.** `const` freezes variable AND contents. `let` = fully mutable.
- **Assignment is a statement**, not an expression. Prevents `if x = 5` bugs.
- **Truthiness:** `false`, `none`, `0` only. `""` and `[]` are truthy.

## Versioning

**Semantic versioning:** `MAJOR.MINOR.PATCH`. `0.x.y` = pre-1.0, breaking changes expected.

**Release names:** Urdu/Hindi single words. Chosen when shipping, not planned ahead.

Word bank: *Safar* (journey) · *Noor* (light) · *Umeed* (hope) · *Fikr* (thought) · *Sukoon* (peace) · *Irada* (will) · *Khoj* (discovery) · *Raasta* (path) · *Buniyaad* (foundation) · *Dastak* (arrival) · *Ehsaas* (awareness) · *Amal* (action)

## Build Phase Order

1. Lexer ✅
2. Parser ✅
3. C Runtime Library ✅
4. C Code Generation ✅
5. CLI ✅ (monk build, monk run, monk check, -o for C output)
6. Type System + scalar unboxing codegen ✅
7. Module System
8. C FFI (syntax TBD)
9. Linter & Formatter
10. LSP + Editor
11. Distribution

## Project Structure

```
src/                     — Go compiler (module root)
  main.go                    CLI: monk build/run/check/version/help
  main_test.go               CLI integration tests
  embed.go                   go:embed for runtime (self-contained binary)
  go.mod                     github.com/monkfromearth/monk-lang
  syntax/                    Lexer + Parser + AST (see INDEX.md)
    token.go, scanner.go
    ast.go, ast_expr.go, ast_stmt.go
    parser.go, parse_stmt.go, parse_expr.go, parse_type.go
  types/                     Static type checker
  codegen/                   AST → C emitter + scalar unboxing (see INDEX.md)
    gen.go, gen_stmt.go, gen_expr.go, gen_func.go, gen_helpers.go, unbox.go
  runtime/                   C runtime library, linked into every binary (see INDEX.md)
    runtime.h                    public API (MonkValue + function declarations)
    internal.h                   shared helpers (not for generated code)
    value.c, arith.c, string.c, container.c, math.c, builtins.c, error.c
spec/                    — Language specification
  REFERENCE.md               THE source of truth for syntax and semantics
  ARCHITECTURE_DECISIONS.md  Why Go, why compile-to-C, what can change later
  MEMORY_MODEL_DISCUSSION.md Design history (rejected options, read only if needed)
knowledge/               — Learning course (Brilliant.org style, Astro + Solid)
examples/                — 9 working .monk programs
Makefile                 — make build/install/test/clean
CHANGELOG.md             — Release history
```

## Building

```bash
make              # build → ./monk
make install      # build + copy to ~/.local/bin/monk
make test         # run all tests
make clean        # remove binary
```

Go 1.26.1+ required (matches `src/go.mod`). Tests need a C compiler (cc/gcc/clang).

## Knowledge Site Constraints

If working on `knowledge/`:
- NO emojis. NO purple/violet/indigo.
- Dark code blocks (Shiki, `#1C1917`, monk-dark theme).
- Inline `<code>` with `{`/`}` must use `&#123;`/`&#125;` (JSX escaping). `<Code code={...}>` handles this automatically.
- All links use `import.meta.env.BASE_URL` for GitHub Pages base path (`/monk-lang/`).
- Deployed at https://monkfromearth.github.io/monk-lang/ (auto-deploys via `.github/workflows/deploy-knowledge.yml`).

## Mistakes to Avoid

- Don't build an interpreter. Monk is a compiler. No REPL.
- Don't make creative decisions (naming, branding) without presenting options.
- Compute new values BEFORE freeing old ones in codegen (use-after-free).
- Don't skip position tracking on AST nodes — codegen needs `#line` directives.
- Don't double-evaluate expressions in generated C — use GCC/clang statement-expressions `({ int64_t _t=expr; ... _t ...; })` or emit a named temp before the expression. A side-effecting RHS like `a / f()` must call `f()` exactly once.
- When writing benchmark `.monk` code, use `break` and `continue` explicitly. Setting a loop variable to sentinel values (`d = n` to exit) works but runs extra iterations and changes perf by 3-5×.

## Related Repos

| Repo / URL                           | Purpose                          |
| ------------------------------------ | -------------------------------- |
| `monkfromearth/monk-lang`            | This repo — the compiler         |
| `monkfromearth.github.io/monk-lang/` | Knowledge site (GitHub Pages)    |
| `monkfromearth/monk-lang-v1`         | Archived v1 (TS/Bun interpreter) |
| `monkfromearth/homebrew-monk-lang`   | Homebrew tap (update in Phase 9) |
| `monkfromearth/monklore`             | Docs site (Next.js, separate)    |
