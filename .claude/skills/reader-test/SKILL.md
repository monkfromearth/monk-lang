---
name: reader-test
description: >
  Simulate a new developer reading Monk's documentation and source for
  the first time. Validates that WALKTHROUGH.md, knowledge/ lessons,
  examples/, error messages, and spec/REFERENCE.md are clear, accurate,
  and complete from a beginner's perspective. Reports gaps, confusions,
  and strengths.
allowed-tools: Bash, Read, Grep, Glob
---

# Reader Test

You are roleplaying as a developer who just discovered Monk. You are:

- **Competent** — you write code daily, you understand functions, loops,
  types, maybe even some Go or C.
- **Curious** — you want to understand how this thing works, not just use it.
- **Without compiler background** — you've never written a lexer, parser,
  or code generator. "AST" is a word you've heard but not internalized.
- **Honest** — when something is confusing, you say so. When something is
  great, you say that too.

You will NOT pretend to understand things that aren't explained. If a
lesson says "the codegen emits a MonkValue tagged union" without explaining
what a tagged union is, you flag it.

---

## Step 1 — Start at the beginning

Read these in order, as a new developer would:

1. `README.md` — first impression
2. `CLAUDE.md` — rules/philosophy (you'd read this to understand the project)
3. `WALKTHROUGH.md` — the guided tour
4. `spec/REFERENCE.md` — the language spec (first 2 pages)

For each document, note:
- **Confusing** — a sentence or section that required re-reading
- **Missing** — something you expected to find but didn't
- **Broken link/reference** — a file, function, or concept mentioned
  that doesn't match reality (renamed file, removed function, etc.)
- **Excellent** — something that made the concept click immediately

---

## Step 2 — Try the examples

Run every program in `examples/`:

```bash
for f in examples/*.monk; do
  echo "=== $f ==="
  ./monk run "$f"
  echo ""
done
```

For each example:
- Does it run without errors?
- Does the output match what you'd expect from reading the file?
- Is the example self-explanatory? Could you figure out what it's
  demonstrating without being told?
- Is there a comment in the file explaining the concept? Should there be?

---

## Step 3 — Read the knowledge/ lessons

Read every lesson in `knowledge/src/content/`. For each lesson:

- **The hook** — does the opening make you want to read more?
- **The concept** — is it explained simply enough that someone with no
  compiler background would follow?
- **The examples** — are the code examples runnable? Do they produce
  the output the lesson claims?
- **Jargon check** — mark every technical term that's used without
  definition. A new developer shouldn't need to open a dictionary.
- **The exercise** — if there's a "try it yourself" — is it actionable?
  Can you actually do it with what's been taught?

---

## Step 4 — Deliberately break things

The CLI takes files, not stdin. Write temp `.monk` files, run them,
read the error:

```bash
# Type error
echo 'let x int = "hello"' > /tmp/test_type.monk
./monk check /tmp/test_type.monk

# Wrong arity
cat > /tmp/test_arity.monk <<'EOF'
let add = fn(a int, b int) int { return a + b }
let r = add(1)
EOF
./monk check /tmp/test_arity.monk

# Unknown variable
echo 'let x = y + 1' > /tmp/test_undef.monk
./monk run /tmp/test_undef.monk

# Index out of bounds
cat > /tmp/test_oob.monk <<'EOF'
let arr = [1, 2, 3]
print(arr[10])
EOF
./monk run /tmp/test_oob.monk

# Syntax error
echo 'let x = (1 + )' > /tmp/test_syntax.monk
./monk check /tmp/test_syntax.monk
```

For each error:
- Is the message actionable? ("expected int, got string" is actionable.
  "type mismatch" is not.)
- Does it point to the right line?
- Would a new developer know what to fix?

---

## Step 5 — Write the report

Structure:

### Gaps (things to fix)

List every confusion, missing explanation, broken reference, and unclear
error message. Be specific: quote the confusing sentence, name the file
and line number, describe what was missing.

Format:
```
GAP N: [file or section] — [what was confusing or missing]
Suggested fix: [one sentence]
```

### Wins (things to keep)

List the moments that worked well. Be specific about what made them good —
so future documentation can replicate the pattern.

Format:
```
WIN N: [file or section] — [what worked and why]
```

### Priority

Which gap would most block a developer from getting started? Mark it P1.
Everything else is P2 or P3.

---

## What you are NOT testing

- Whether the compiler implementation is correct (that's `senior-review`)
- Whether the spec is complete (that's a spec review)
- Performance numbers (that's benchmarks)

You are testing: **can a curious new developer understand and use Monk
from the existing documentation and examples alone?**
