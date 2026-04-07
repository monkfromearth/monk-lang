# File Organization

## The Core Rule

One topic per file. Go makes this free — same package, multiple files.
When a file exceeds ~500 lines, that's a signal it's holding two concerns.
Split it.

## When to Split

Split when a file:
- Exceeds ~500 lines (soft limit — use judgment, not a linter)
- Mixes parsing logic with AST node definitions
- Mixes statement handling with expression handling
- Mixes helper utilities with primary logic
- Has two clearly separable responsibilities that a reader would have to
  mentally juggle at once

Don't split just to stay under 500 lines. Split when the file has two
stories to tell.

## Naming Conventions (from existing patterns)

| Pattern | Meaning | Example |
|---------|---------|---------|
| `noun.go` | Data types, structs, constants | `ast.go`, `token.go` |
| `noun_kind.go` | Specialized subset of data | `ast_expr.go`, `ast_stmt.go` |
| `verb_noun.go` | Operations on a thing | `parse_expr.go`, `parse_stmt.go`, `gen_stmt.go` |
| `verb.go` | Single-concern behavior | `unbox.go`, `embed.go` |
| `noun_test.go` | Tests for `noun.go` | `parser_test.go` |

Keep the name self-documenting. If you can't name a file without using
"and" or "misc", it's doing too many things.

## Runtime C Files

Same principle applies. Each `.c` file handles one concern:

- `value.c` — MonkValue construction, tag inspection
- `arith.c` — arithmetic operations
- `string.c` — string builtins
- `container.c` — array and record operations

Don't add a string utility to `arith.c` because it's convenient.
Create a new file or add to `string.c`.

## INDEX.md Is Mandatory

Every source directory (`src/syntax/`, `src/codegen/`, `src/runtime/`)
has an `INDEX.md` listing each file and what lives in it.

**Update INDEX.md in the same commit as the file change.** Never let it
drift. A stale INDEX.md is worse than no INDEX.md — it misdirects readers.

One-line-per-file format: `filename.go` — what it contains.

## When You Add a New File

Checklist:
1. Name it according to the conventions above
2. Add it to the directory's `INDEX.md` immediately
3. If it's a runtime `.c` file — no extra wiring needed (auto-discovered),
   but still add it to `runtime/INDEX.md`
4. Keep the new file focused. If you find yourself adding a second concern
   to it before the first PR is merged, that's a split waiting to happen

## Anti-Patterns

- `helpers.go` — helpers for what? Split by the thing being helped.
- `utils.go` — same problem.
- A 900-line `parser.go` with both the Parser struct and all statement parsers.
  (We already fixed this — `parse_stmt.go`, `parse_expr.go`, `parse_type.go`.)
- A `gen.go` that handles both statements and expressions.
  (We already fixed this too — split was worth it.)

Don't undo past splits by letting files grow back together.
