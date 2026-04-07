# Plan: Language Guide + Documentation Site + Knowledge Units 6+

> Finalized 2026-04-06. Three workstreams, prioritized.

---

## Overview — Three Sites, Three Audiences

| Site | Location | Framework | Audience | Analogy |
|------|----------|-----------|----------|---------|
| **Build a Compiler** | `knowledge/` | Astro + Solid (existing) | Devs learning compiler design | Brilliant.org |
| **Learn Monk** | `knowledge/src/pages/guide/` | Same Astro site, new section | Users who want to write Monk | The Rust Book |
| **Monk Docs** | `docs/` | Fumadocs (Next.js) | Reference lookup while coding | MDN / Rust Reference |

---

## Workstream 1: Language Guide ("Learn Monk")

**Where:** `knowledge/src/pages/guide/` — new section in existing Astro site.

**Why in knowledge/:** Reuses components (`InsightBox`, `LessonNav`, `ExerciseBox`, etc.), design system (no emojis, no purple, monk-dark code theme), and GitHub Pages deployment. The landing page gets two clear tracks: "Build a Compiler" and "Learn Monk".

### Pages (10)

| # | Route | Topic | Content Source |
|---|-------|-------|----------------|
| 1 | `/guide/` | Landing page | Links to all sections, what Monk is, install steps |
| 2 | `/guide/first-program` | Hello world | `monk run`, `monk build`, `-o` flag, `examples/hello.monk` |
| 3 | `/guide/variables` | Variables & constants | `let` vs `const`, types, deep const, inference |
| 4 | `/guide/functions` | Functions | Expressions, mandatory types, recursion, higher-order. `examples/fibonacci.monk` |
| 5 | `/guide/control-flow` | Control flow | if/else, while, for-in, break/continue, range(). `examples/fizzbuzz.monk`, `examples/collatz.monk` |
| 6 | `/guide/arrays` | Arrays | Creation, indexing, builtins, value semantics. `examples/arrays.monk`, `examples/sort.monk` |
| 7 | `/guide/records` | Records | Creation, property access, fixed shape, records with functions. `examples/todo_list.monk` |
| 8 | `/guide/error-handling` | Error handling | guard/against/throw with analogy. `examples/error_handling.monk` |
| 9 | `/guide/value-semantics` | Value semantics | Everything copies, why it matters, proof with examples |
| 10 | `/guide/builtins` | Built-in functions | Table of the 20 most useful builtins with examples |

### Page Structure (each page)

1. **Analogy/concept** — explain the idea before any code
2. **First code example** — minimal, runs with `monk run`
3. **Deep dive** — edge cases, type system interaction, design philosophy
4. **Interactive element** — visualization or exercise (e.g., type inference step-through, value copy animation)
5. **Try it** — `monk run` / `monk build -o` commands to run yourself
6. **Navigation** — prev/next lesson links

### Install Page (in landing or first-program)

```
## Install Monk

### From source (requires Go 1.26+ and a C compiler)
git clone https://github.com/monkfromearth/monk-lang
cd monk-lang
make install    # installs to ~/.local/bin/monk

### Verify
monk version
```

Homebrew tap exists (`monkfromearth/homebrew-monk-lang`) but isn't wired up yet — mention it as "coming soon".

### Interactive Elements (planned)

| Page | Interactive Element |
|------|-------------------|
| `variables` | Type inference visualizer — shows how the checker infers types from first assignment |
| `functions` | Closure capture diagram — what gets copied into the closure struct |
| `value-semantics` | Before/after memory diagram — assignment copies, mutation doesn't propagate |
| `arrays` | Array operation playground — shows input → operation → new array |
| `error-handling` | Guard/against flow diagram — how execution jumps on throw |

Build these as Solid.js components (same as existing knowledge site interactive elements).

### Landing Page Update

The existing `knowledge/src/pages/index.astro` needs a second track:

```
┌─────────────────────────────────────────┐
│           Monk Lang                     │
│   A compiled language that thinks       │
│   in values, not references             │
├───────────────────┬─────────────────────┤
│  🔧 Build a       │  📖 Learn Monk      │
│  Compiler          │                     │
│                    │  Write programs in   │
│  How Monk works    │  Monk. From hello    │
│  under the hood.   │  world to error      │
│  26 lessons.       │  handling.           │
│                    │  10 lessons.         │
│  [Start Course →]  │  [Start Guide →]    │
└───────────────────┴─────────────────────┘
```

(No actual emojis in implementation — those are placeholders for layout.)

---

## Workstream 2: Documentation Site ("Monk Docs")

**Where:** `docs/` — new Fumadocs (Next.js) project.

**Why separate:** Docs are reference material, not narrative. Different information architecture (sidebar nav, search, API tables) than the tutorial sites. Fumadocs gives us search, sidebar, MDX, and versioning out of the box.

### Setup

```
docs/
├── content/docs/          # MDX content files
│   ├── language/          # Friendly spec rewrite
│   ├── api/               # Builtins API reference
│   └── cli/               # CLI reference
├── src/app/
│   ├── docs/              # Doc layout + catch-all page
│   └── api/search/        # Built-in search route
├── source.config.ts       # MDX source config
├── next.config.mjs
├── package.json
└── tailwind.config.ts
```

**Dependencies:** `fumadocs-core`, `fumadocs-ui`, `fumadocs-mdx`, `next`, `react`, `tailwindcss`

### Content Sections

#### Section 1: Language Reference (friendly spec rewrite)

Rewrite `spec/REFERENCE.md` as navigable pages with more examples and less formalism.

| Page | Source (from spec) |
|------|-------------------|
| `language/design-philosophy` | Design Philosophy section |
| `language/comments` | Comments |
| `language/data-types` | Data Types — int, float, string, bool, none, array, record, function |
| `language/variables` | Variables and Constants — let, const, deep const, type annotations |
| `language/operators` | Operators — arithmetic, comparison, logical, bitwise, precedence table |
| `language/functions` | Functions — syntax, closures, recursion, higher-order, hoisting |
| `language/control-flow` | Control Flow — if/else, while, for-in, break/continue |
| `language/arrays` | Arrays — homogeneous, value semantics, all operations |
| `language/records` | Records — creation, access, mutation rules |
| `language/error-handling` | Error Handling — guard/against/throw, nested guards, rethrow |
| `language/type-system` | Type System — annotations, inference, optional types, custom types, structural typing |
| `language/template-literals` | Template Literals |
| `language/edge-cases` | Edge Cases and Special Behaviors |

Each page: definition → syntax → examples → edge cases → "See also" links.

#### Section 2: API Reference (builtins)

One page per category, with a table per function.

| Page | Functions |
|------|-----------|
| `api/output` | `show`, `show_error` |
| `api/type-checking` | `typeof`, `is_number`, `is_string`, `is_boolean`, `is_array`, `is_record`, `is_function`, `is_none` |
| `api/conversion` | `to_string`, `to_int`, `to_float` |
| `api/string` | `length`, `substring`, `index_of`, `split`, `trim`, `to_upper`, `to_lower`, `replace`, `starts_with`, `ends_with`, `contains` |
| `api/array` | `length`, `append`, `prepend`, `pop`, `drop`, `take`, `slice`, `range`, `keys` |
| `api/record` | `keys`, `has` |
| `api/math` | `abs`, `floor`, `ceil`, `round`, `sqrt`, `pow`, `log`, `log10`, `exp`, `min`, `max`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan` |
| `api/file-system` | `file_read`, `file_write`, `file_exists` |
| `api/system` | `env_get`, `exit`, `args` |

Each function entry: signature, description, return type, example, edge cases.

#### Section 3: CLI Reference

| Page | Content |
|------|---------|
| `cli/build` | `monk build <file>` — all flags, `-o` behavior (extension decides output) |
| `cli/run` | `monk run <file>` — compile + execute + cleanup |
| `cli/check` | `monk check <file>` — parse + type-check without compiling |
| `cli/version` | `monk version` |
| `cli/overview` | Summary table of all commands |

### Design System

Match the knowledge site's aesthetic:
- **Dark code blocks** with monk-dark theme (same Shiki config)
- **Warm palette** — stone/amber tones, no purple/violet
- **No emojis** in content
- Fumadocs supports Tailwind customization — override their default theme to match

### Deployment

Options (decide during implementation):
- **GitHub Pages** alongside knowledge site (different base path: `/monk-lang/docs/`)
- **Vercel** (free tier, automatic from `docs/` directory)
- **Subdomain** if we get a custom domain later

---

## Workstream 3: Knowledge Site Unit 6 — Type System

**Where:** `knowledge/src/pages/phase-6-types/`

### Pages (6)

| # | Route | Topic | Teaching Approach |
|---|-------|-------|-------------------|
| 1 | `what-is-a-type-system` | Why types exist | Analogy: types as contracts. Static vs dynamic typing. What Monk chose and why. |
| 2 | `type-inference` | First-assignment inference | Walk through how the checker figures out types without annotations. The "detective" analogy — clues from usage. |
| 3 | `optional-types` | `int?` and none-safety | The "maybe it's there, maybe it's not" problem. How other languages handle null. Why `int?` is better than null everywhere. |
| 4 | `structural-typing` | Record types and type aliases | Analogy: "if it has the right shape, it fits." Nominal vs structural — why Monk picks structural. Real examples with records. |
| 5 | `scalar-unboxing` | The performance optimization | Before/after: MonkValue wrapper vs raw C int. Show the generated C diff. Benchmark numbers (fibonacci, mandelbrot). Not "how to implement" — "what the compiler does for you." |
| 6 | `references` | Links and further reading | Spec sections, academic papers on type inference, related compiler courses. |

### Teaching Philosophy for Unit 6

- **Not showing Go code.** Units 1-5 showed compiler internals. Unit 6 can still show what the type checker does, but through Monk examples and generated C output, not through `types/checker.go` listings.
- **Analogies first, always.** Type inference = detective. Structural typing = duck typing with proof. Unboxing = removing the gift wrapping.
- **Interactive elements:**
  - Type inference step-through: input a Monk snippet, see the checker assign types variable by variable
  - Unboxing diff viewer: toggle between boxed/unboxed C output for the same Monk function

---

## Priority & Dependencies

```
Priority 1 (start now):
  ├── Language Guide (10 pages)     — no blockers
  └── Knowledge Unit 6 (6 pages)    — no blockers

Priority 2 (after guide is live):
  └── Docs site scaffolding         — Fumadocs setup, design system, deploy pipeline

Priority 3 (after scaffold):
  ├── Language Reference pages       — rewrite from spec
  ├── API Reference pages            — from spec builtins section + runtime source
  └── CLI Reference pages            — from main.go flag handling

Priority 4 (blocked on compiler):
  ├── Knowledge Unit 7 (Modules)     — blocked on Phase 7
  ├── Knowledge Unit 8 (FFI)         — blocked on Phase 8
  ├── Docs: Module System pages      — blocked on Phase 7
  └── Docs: FFI pages                — blocked on Phase 8
```

---

## Estimated Scope

| Workstream | Pages | New Components | Framework Work |
|------------|:-----:|:--------------:|:--------------:|
| Language Guide | 10 | 3-5 Solid.js interactive | Landing page redesign |
| Knowledge Unit 6 | 6 | 2 Solid.js interactive | None (existing infra) |
| Docs site | ~25 | 0 (Fumadocs built-in) | Full Fumadocs scaffold + theme |
| **Total** | **~41** | **5-7** | **1 new site** |

---

## Open Questions (resolved)

- ~~Framework for docs/~~ → Fumadocs (Next.js)
- ~~Guide location~~ → `knowledge/src/pages/guide/`
- ~~Docs scope~~ → Friendly spec + API reference + CLI reference
- ~~Unit 6 depth~~ → Analogies and examples, not compiler code

## Remaining Decisions (decide during implementation)

1. **Docs deployment** — GitHub Pages vs Vercel vs custom domain
2. **Monk syntax highlighting** — need a Shiki grammar for `.monk` files in Fumadocs (knowledge site already has monk-dark theme — can we extract the grammar?)
3. **Cross-linking** — how do guide pages link to docs reference? Relative URLs or absolute?
4. **Versioning** — Fumadocs supports versioned docs. Start with unversioned, add when Monk hits 1.0?

---

_This plan lives at `DOCS_AND_GUIDE_PLAN.md`. Delete when all three workstreams are complete._
