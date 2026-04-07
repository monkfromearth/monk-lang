---
name: knowledge-update
description: >
  Update knowledge/ lessons and WALKTHROUGH.md after a feature or phase
  lands. Writes with the voice of a simplistic, knowledgeable teacher —
  "curious developer exploring how compilers work" energy, no assumed
  background, guided by concrete examples and design reasoning.
allowed-tools: Bash, Read, Grep, Glob, Edit, Write
---

# Knowledge Update

You are updating documentation for a developer who has never built a
compiler before, but is curious and smart. Your job is to make the
internals feel approachable — not dumbed down, but guided.

**The voice that already exists in WALKTHROUGH.md and knowledge/:**
- "Open `src/main.go` and find `generateC`."
- Table: file → what it does. One line. Concrete.
- "Key design decisions" — not just what, but *why*.
- Low jargon. When you use a term ("AST", "tagged union"), explain it
  inline the first time. No footnotes.
- Show actual code from the repo, not pseudocode.
- Encourage exploration: "Find the function. Read it. Come back."

**Never:** lecture, assume compiler background, list features without
context, write documentation that reads like an API reference.

---

## Step 1 — Understand what changed

Read `PROGRESS.md` (the last session entry) and `git log --oneline -10`.
Identify: what feature landed? What phase completed? What behavior changed?

If you're not sure what changed, ask before writing anything.

---

## Step 2 — Read what exists

Before writing a single word, read:

1. The existing `knowledge/` page(s) that cover the affected area
2. The relevant section of `WALKTHROUGH.md`
3. `spec/REFERENCE.md` — the part that covers the new feature

You cannot update in the established voice without reading what's already
there. The voice is consistent across pages — match it.

---

## Step 3 — Identify the gap

What does a new developer encounter now that didn't exist before?

Ask:
- "If I cloned the repo today and read WALKTHROUGH.md, would this new
  feature make sense?"
- "Is there a knowledge/ lesson that should exist but doesn't?"
- "Does an existing lesson now have a wrong example or a missing case?"

Write down the specific gaps before writing any content.

---

## Step 4 — Write (or update) the content

### For WALKTHROUGH.md

The walkthrough is a guided tour of the compiler source. It follows the
pipeline: syntax → types → codegen → runtime → CLI.

Add new sections in pipeline order. Structure per section:

```markdown
### `src/<package>/` — Short name (what it takes in → what it produces)

**Input:** what enters this stage
**Output:** what leaves this stage

| File | What it does |
|------|-------------|
| `file.go` | one-line description |

**Key design decisions:**
- Decision 1 — *why* in one sentence
- Decision 2 — *why* in one sentence
```

Do not add a section for every new file. Add sections when there is a
new architectural concept a reader needs to understand to follow the code.

### For knowledge/ lessons

Each lesson teaches one concept. The structure that works:

1. **The hook** — a question or scenario the reader already cares about.
   "Why does `let x = [1, 2, 3]; let y = x` give you two independent arrays?"

2. **The concept** — explain it simply. One analogy is worth a page of
   explanation. Use the analogy that already exists in the codebase's
   comments and docs — don't invent new ones unless the existing ones
   are absent.

3. **The code example** — actual Monk code. Not pseudocode. Show what works.
   If relevant, show what fails and what error you get.

4. **The why** — connect the concept back to the design philosophy
   (explicit over implicit / graceful reads, strict ops / values not
   references). One sentence.

5. **Try it yourself** — a small exercise or "now change X and see what
   happens." This is what makes it Brilliant-style, not just documentation.

---

## Step 5 — The teacher test

Before saving, read what you wrote as if you just cloned the repo today.

Ask:
- Can I follow this without going to Google?
- Is there a moment where I'd think "wait, why?" — and the text doesn't
  answer it?
- Is there a jargon term that isn't explained?
- Is the code example something I can actually run?

If the answer to any of these is "yes", fix it before moving on.

---

## Step 6 — Sync the chain

After updating knowledge/ or WALKTHROUGH.md, check:
- `PROGRESS.md` — was the doc update noted? (one-liner is fine)
- Cross-links — if another lesson or section references the feature you
  just documented, does the cross-link still make sense?
- Examples — if you wrote a code example, verify it actually runs:
  `./monk run <example_file>`

---

## What NOT to do

- Don't write in passive voice. "The value is copied" → "Monk copies the value."
- Don't add a lesson for every new function. Lessons are for *concepts*,
  not API surface.
- Don't document implementation details that could change. Document
  *behavior* (what the language does) and *architecture* (why the
  compiler is structured this way). Skip internal function names that
  a user of the language would never encounter.
- Don't update WALKTHROUGH.md for bug fixes. Only for structural changes
  a new developer reading the source would need to understand.
