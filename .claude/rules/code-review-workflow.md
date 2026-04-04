# Code Review Workflow

## When to Use

Apply this workflow whenever:
- User pastes review output (CodeRabbit, AI reviewer, human PR comment, linter output)
- Running `cr review` and acting on results
- Processing any batch of code review feedback

## The Workflow

### Step 1: Parse into Exhaustive List

Extract EVERY item — no collapsing, no skipping. Number them sequentially.

Format each item:
```
N. [FILE:LINE or SECTION] — [Issue summary in one line]
```

If the review has sections (nitpick / major / minor), preserve that grouping but still number globally.

### Step 2: Judge Each Item

For each numbered item, assign one of three verdicts:

| Verdict | Meaning |
|---------|---------|
| **ACT** | Clearly correct, matches our patterns, adds value — implement it |
| **SKIP** | Doesn't apply to our codebase, contradicts existing patterns, or is purely stylistic noise |
| **DISCUSS** | Ambiguous, architectural implication, or conflicts with existing code — needs user input before proceeding |

**Verdict format:**
```
N. ACT   — [one-line reason]
N. SKIP  — [one-line reason: why it doesn't apply]
N. DISCUSS — [one-line question or conflict to resolve]
```

### Step 3: Present the Full Verdict Table

Show the complete numbered list with verdicts BEFORE making any changes.
Do NOT silently skip items. If an item is borderline, surface it as DISCUSS.

Example output:
```
## Review Items — Judgment

1. ACT   — services/user.ts:42 — Missing null check on `account` before access
2. SKIP  — "Add workspace_id to post model" — post is MongoDB module, workspace isolation is per-DB
3. ACT   — Missing error handling in job trigger (consistent with code-patterns.md)
4. DISCUSS — Suggests extracting validation to separate file — only used in one place, YAGNI applies?
5. SKIP  — Suggests using `async/await` over `.then()` — both are valid here, not a real issue
6. ACT   — Variable name `data` is too generic — rename to `member_data`
```

### Step 3.5: Provide Context for DISCUSS Items

For each DISCUSS item, include enough context for the user to answer without going back to the code:

1. **What the old behavior was** vs **what the new behavior is** — concrete diff, not prose
2. **What failure mode you're guarding against** — be specific (e.g. "client disconnect mid-stream means `GeneratorExit` is thrown, not `Exception`, so the counter leaks")
3. **What question needs answering** — yes/no or a specific choice, not an open-ended "thoughts?"
4. **A verification path** — how the user can confirm without your help (e.g. "grep `reasoningEffort` in `node_modules/@ai-sdk/azure`")

Example DISCUSS block:
```
5. DISCUSS — `providerOptions.azure.reasoningEffort` vs old `reasoning.effort`
   Old: reasoning: { effort: "high" } (SDK v4 generic interface)
   New: providerOptions: { azure: { reasoningEffort: "high" } } (Azure-specific)
   Risk: If SDK doesn't translate `reasoningEffort` → `reasoning_effort` in the HTTP body,
         reasoning models silently get no effort level.
   Verification: grep "reasoningEffort" node_modules/@ai-sdk/azure/dist/index.js
   Question: Has this been tested against the staging Azure deployment?
```

### Step 3.7: Leave SKIP Comments for the Reviewer

When the reviewer's AI agent suggests fixes and we choose to SKIP, add a brief inline comment at the referenced location explaining WHY the item was skipped. This serves two purposes: (1) the reviewer understands the reasoning without re-analyzing, (2) future reviewers don't re-flag the same item.

Not every SKIP needs a comment — only add them when:
- The reviewer describes a plausible bug that doesn't exist in current code (already fixed, or misread)
- The reviewer suggests a change that contradicts an intentional design decision
- The item references code/files that don't exist (stale reference)

Comment format: `// REVIEW-SKIP: [one-line reason]` or `# REVIEW-SKIP: [one-line reason]`

Example:
```python
# REVIEW-SKIP: guard already includes account_id check — see cases in comment above.
# The None==None bypass was fixed in commit 4e79e37.
if existing and (
    (task_id is not None and existing.task_id == task_id)
    or (task_id is None and existing.task_id is None and existing.account_id == account_id)
):
```

Do NOT litter the codebase with REVIEW-SKIP comments for every item — use judgment. If the code is already well-commented and the skip reason is obvious from reading it, no additional comment is needed.

### Step 4: Confirm Then Execute

- If there are DISCUSS items: wait for user to resolve them before proceeding
- If all items are ACT/SKIP: present the list, then proceed to implement ACT items
- If user says "go ahead" or "proceed": implement all ACT items

### Step 5: Report Resolutions

After implementing, report per-item:
```
1. DONE  — Added null check on account before access
3. DONE  — Wrapped job trigger in try/catch, logs error without failing primary op
6. DONE  — Renamed `data` → `member_data` in services/user.ts:42
```

---

## Judgment Heuristics

**SKIP when:**
- Suggestion contradicts how 5+ other places in the codebase work
- Adding `workspace_id` to a MongoDB module (bevy, beacon, membership, blog, event, community)
- Moving job triggering outside a transaction when our pattern keeps it inside
- Generic advice that doesn't account for Bun/Lema ORM/our specific stack
- The reviewer flagged a pattern that is intentionally different here for a documented reason
- It's a style nitpick with no correctness/maintainability implication
- It duplicates what an external service already handles

**ACT when:**
- Null/undefined access without a guard
- Missing `workspace_id` filter in a Postgres module query
- Error thrown with a plain `Error` instead of `LevoError.Platform`
- Background job failure that would bring down the primary operation
- Naming that fails the 5-second clarity rule
- Actual security issue with a clear exploit path
- **Improvements that reduce future risk** — consistency fixes (e.g., sentinel check missing `tool_name`), defense-in-depth additions (e.g., Python-level validation before SQL safety net), documentation of known gaps with TODO for future fix (e.g., fire-and-forget indexing → SQS). If the fix is small (1-5 lines) and reduces ambiguity or prevents a future class of bug, ACT — don't defer it
- **Inconsistencies across language boundaries** — if Python does X and TS does Y for the same operation (e.g., byte count vs char count, `secrets.choice` vs `Math.random`), align them. Cross-language parity bugs are hard to catch later
- **Two sources of truth that can drift** — if a dict/constant and SQL/config encode the same rules, make one authoritative and the other derived or validated. Add a comment linking them at minimum, add a Python-level pre-check if possible

**DISCUSS when:**
- Architectural change (new file, new abstraction, extraction)
- Contradicts this rule set but the reviewer makes a compelling case
- Touches DB schema or migration
- Affects multiple modules or the public API surface
- **Product/design questions** — e.g., "should text/plain get semantic embeddings?" is a product decision, not a code bug. Present the trade-off and let the user decide

**Bias toward action.** When in doubt between SKIP and ACT, prefer ACT if the fix is small and the item improves consistency, defense-in-depth, or documentation of known gaps. A 1-line consistency fix or a TODO comment costs nothing; a deferred improvement may never happen. The bar is: "Would a future developer be confused or bitten by this?" If yes, fix it now.

---

## Key Safeguards

**Never blindly apply.** The full protocol is in `analysis-protocol.md` under "External Tool Suggestions Safeguard". In short:

- Verify the suggestion matches existing patterns (grep first)
- If you can't explain WHY the existing code is wrong, don't change it
- "Race condition" and other theoretical issues require a concrete failure scenario in OUR code
- A suggestion that makes code MORE complex is a red flag
