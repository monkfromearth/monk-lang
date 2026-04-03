# Prompt: Migrate knowledge/ to Astro + Solid

> Copy-paste this into a new Claude Code session to continue the work.

---

## Context

I'm building **Monk Lang** — a programming language. The repo is at `projects/monk-lang/`. Read `CLAUDE.md` for the full project context.

Inside `knowledge/` there's a **learning course** (like Brilliant.org) that teaches how to build a compiler. It currently has 4 raw HTML pages using Tailwind CDN:

- `knowledge/index.html` — course map / landing page
- `knowledge/foundations/how-compilers-work/index.html` — lesson 0.1
- `knowledge/foundations/zig-first-30/index.html` — lesson 0.2 (note: will be reworked for Go, but content structure is the reference)
- `knowledge/foundations/first-program/index.html` — lesson 0.3

These work (open in browser) but are extremely verbose — each page is 400+ lines of HTML with duplicated layout, nav, footer, code block styling, and Tailwind config.

## The Task

Migrate `knowledge/` to an **Astro project** with **Solid** for any interactive components. Use **Bun** as the package manager and runtime.

### What to build

1. **Astro project setup** inside `knowledge/` (or rename to `knowledge-site/` if cleaner)
   - `bun create astro@latest`
   - Add Solid integration: `@astrojs/solid-js`
   - Tailwind integration: `@astrojs/tailwind`

2. **Shared layout component** — extract the common structure:
   - Sticky nav with back link, lesson number, section label
   - Max-width content container
   - Footer with GitHub link
   - Tailwind config (the monk color palette, fonts)
   
3. **Reusable components** — extract these from the existing HTML:
   - `<CodeBlock>` — light-theme code block with syntax highlighting classes (`.kw`, `.ty`, `.fn`, `.str`, `.num`, `.cm`, `.op`, `.builtin`). Takes a `lang` prop for future syntax highlighting.
   - `<InsightBox>` — orange left-border callout box
   - `<ExerciseBox>` — green left-border exercise prompt
   - `<StepNumber>` — numbered circle (used for walkthrough steps)
   - `<LessonNav>` — prev/next navigation at bottom of lessons
   - `<ChecklistItem>` — green checkmark + text (for checkpoint sections)
   - `<TagBadge>` — the small colored tags (concepts, go, c, monk, exercise, build, links)

4. **Content as Astro pages** (or MDX if it makes content authoring easier) — migrate the 4 existing pages. The content stays identical, just uses components instead of raw HTML.

5. **Course map (index)** stays as an Astro page with the pipeline visualization and unit cards.

### Folder structure suggestion

```
knowledge/
├── astro.config.mjs
├── package.json
├── tailwind.config.mjs
├── src/
│   ├── layouts/
│   │   └── LessonLayout.astro     ← nav, footer, container, meta
│   ├── components/
│   │   ├── CodeBlock.astro
│   │   ├── InsightBox.astro
│   │   ├── ExerciseBox.astro
│   │   ├── StepNumber.astro
│   │   ├── LessonNav.astro
│   │   ├── ChecklistItem.astro
│   │   ├── TagBadge.astro
│   │   └── Pipeline.astro          ← the animated pipeline diagram
│   ├── pages/
│   │   ├── index.astro              ← course map
│   │   └── foundations/
│   │       ├── how-compilers-work.astro
│   │       ├── go-first-30.astro
│   │       └── first-program.astro
│   └── styles/
│       └── global.css               ← code block colors, animations
├── public/
│   └── (static assets if any)
└── old/                              ← move raw HTML here for reference, delete after migration
```

## Design System — DO NOT CHANGE

### Colors (Tailwind config)

```js
monk: {
    bg: '#FAF8F5',        // page background (warm beige)
    card: '#FFFFFF',       // card background
    border: '#E8E4DF',    // borders
    text: '#1A1A1A',      // primary text
    muted: '#6B6560',     // secondary text
    accent: '#E8590C',    // primary accent (orange)
    accentLight: '#FFF4ED',
    green: '#16A34A',
    greenLight: '#F0FDF4',
    blue: '#2563EB',
    blueLight: '#EFF6FF',
    teal: '#0D9488',
    tealLight: '#F0FDFA',
    amber: '#D97706',
    amberLight: '#FFFBEB',
    locked: '#D1CDC8',
}
```

### Fonts

- Sans: Inter (400, 500, 600, 700, 800)
- Mono: JetBrains Mono (400, 500)
- Load from Google Fonts

### Hard rules

- **NO emojis.** Use inline SVGs (Heroicons outline style) for all icons.
- **NO purple, violet, or indigo** anywhere in the palette.
- **NO dark mode code blocks.** Code blocks have light background (`#FAFAF9`) with `1px solid #E8E4DF` border.
- **Code syntax colors** (applied via CSS classes, not a JS highlighter):
  - `.kw` (keywords): `#E8590C` (accent orange)
  - `.ty` (types): `#0D9488` (teal)
  - `.fn` (function names): `#2563EB` (blue)
  - `.str` (strings): `#16A34A` (green)
  - `.num` (numbers): `#2563EB` (blue)
  - `.cm` (comments): `#9C9590` (muted)
  - `.op` (operators): `#D97706` (amber)
  - `.builtin` (built-ins): `#E8590C` (accent)
- **Body background:** `#FAF8F5` (warm beige, not white)
- **Max content width:** `max-w-3xl` for lesson pages, `max-w-5xl` for course map
- **Trailing commas allowed** in code examples

### Component conventions

- Insight boxes: left border `3px solid #E8590C`, light orange background
- Exercise boxes: left border `3px solid #16A34A`, light green background
- Code blocks: `border-radius: 12px`, `padding: 1.25rem`, `font-mono text-sm leading-relaxed`
- Cards: `rounded-2xl border border-monk-border`, white background, subtle hover shadow
- Tags: tiny uppercase letters, colored pill backgrounds (see existing pages)

### Page structure convention

Every lesson page follows this pattern:
1. Sticky nav (back to course map + lesson number/section)
2. Header (lesson number, title, description, tags with read time)
3. Content sections (h2 headings, explanations, code blocks, diagrams, insight boxes)
4. Key takeaways (numbered list)
5. Bottom nav (prev/next lesson links)
6. Footer

### Tone of content

- First principles. Analogies before code.
- Concise. No filler words.
- Progressive disclosure — simple first, details revealed as needed.
- `<details>` tags for solutions (don't show the answer immediately).
- Every concept ties back to Monk — "this is what the lexer will use."

## Important: Implementation language changed

The compiler is written in **Go** (not Zig — that was changed). The knowledge/ course needs to teach Go basics instead of Zig basics. However, during this Astro migration, just migrate the existing HTML content as-is. The content will be reworked for Go in a separate pass. Focus on the component system, not the content words.

The generated output is still C. The course still teaches C (from Phase 7 onward).

## What NOT to do

- Don't change any content — the text, code examples, diagrams, and structure stay the same
- Don't add a syntax highlighter library (Prism, Shiki, etc.) — the manual CSS classes work and give us full control
- Don't add dark mode
- Don't add client-side routing or page transitions — static output is fine
- Don't add analytics or tracking
- Don't create a design system doc — the components ARE the design system

## After migration is done

- Verify all 4 pages render identically to the current raw HTML
- Delete `knowledge/old/` (the raw HTML backups)
- Run `bun run build` and confirm static output works
- Commit with message describing the migration

## Future pages (don't build these now, but the architecture should support them)

The course has 8 more units planned. Each unit will have 3-9 lesson pages. Some future pages will need:
- SVG diagrams (AST trees, scope chains, memory layouts)
- Interactive components (tokenizer step-through, AST builder) — this is where Solid comes in
- Side-by-side code comparisons (Monk source vs generated C)

The component system should make adding new lessons a matter of writing content in an Astro/MDX file with component imports, not duplicating 400 lines of HTML.
