# Comments With Examples

Every non-trivial code change MUST have an inline comment explaining what it does, with concrete examples showing pass/fail or before/after behavior.

Uncommented code — even correct code — is unacceptable.

## Example of a good comment

```typescript
// Boundary-aware domain check: exact match or proper subdomain only.
// "levo.so" → pass, "app.levo.so" → pass, "evil-levo.so" → fail
const is_allowed = origin === allowed || origin.endsWith(`.${allowed}`);
```

## Example of a bad (missing) comment

```typescript
const is_allowed = origin === allowed || origin.endsWith(`.${allowed}`);
```
