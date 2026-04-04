# Prompt: Create Monk Language Guide as Astro Pages

> For a new agent session. Creates the Monk language learning guide as Astro pages inside knowledge/.

## Context

Monk Lang is a working compiled language. `monk build` compiles `.monk` files to native binaries via C. Read `CLAUDE.md` for full project context and `spec/REFERENCE.md` for the language spec.

The `knowledge/` folder is an Astro + Solid project. See `knowledge/ASTRO_MIGRATION_PROMPT.md` for the design system (colors, components, no emojis, no purple).

## What to Create

Create Astro pages inside `knowledge/src/pages/guide/` for the Monk language guide. This is a Monklore-style learning course — first principles, analogies before code, progressive disclosure.

### Pages to create:

1. `guide/index.astro` — Guide landing page with links to all sections
2. `guide/first-program.astro` — Hello world, monk run, monk build, -o flag
3. `guide/variables.astro` — let vs const, types, deep const, type inference
4. `guide/functions.astro` — Functions as expressions, mandatory types, recursion, higher-order
5. `guide/control-flow.astro` — if/else, while, for-in, break/continue, range()
6. `guide/arrays.astro` — Creation, indexing, built-in functions, value semantics
7. `guide/records.astro` — Creation, property access, fixed shape, records with functions
8. `guide/error-handling.astro` — guard/against/throw with analogy
9. `guide/value-semantics.astro` — Everything copies, why it matters, proof with examples
10. `guide/builtins.astro` — Table of the 20 most useful built-in functions

### Content Source

Use these example programs for code samples (they all compile and run):
- `examples/hello.monk`
- `examples/fibonacci.monk`
- `examples/fizzbuzz.monk`
- `examples/error_handling.monk`
- `examples/arrays.monk`
- `examples/newton_sqrt.monk`
- `examples/todo_list.monk`
- `examples/collatz.monk`
- `examples/sort.monk`

### Rules

- Use the existing Astro layout and components from `knowledge/src/layouts/` and `knowledge/src/components/`
- Follow the design system in `ASTRO_MIGRATION_PROMPT.md`
- Every code example must be valid Monk that compiles with `monk run`
- Explain concepts with analogies BEFORE showing code
- NO emojis, NO purple/violet/indigo
- Show `monk run` and `monk build` commands
- Include the `-o hello.c` flag for seeing generated C
- Link back to the course map (knowledge index) from each page
