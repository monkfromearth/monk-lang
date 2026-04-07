# Security

Monk is a local compiler — not a server, not a service. The threat model
is a single developer running `monk build` on their own machine. Attack
surface is small but non-zero.

Know the threat model. Don't over-engineer it. But don't ignore the
real exposures either.

---

## The actual threat model

**In scope:**
- Malformed `.monk` source crashing the compiler (affects every user)
- Shell injection if user input ever reaches `cc` invocation
- Temp file predictability (race conditions, cross-user issues)
- Module import path traversal escaping the project tree
- Generated C that silently does the wrong thing (security by correctness)

**Out of scope (for now):**
- Server-side use (Monk has no server mode)
- Supply chain attacks on the embedded runtime (it's go:embed, immutable after build)
- `cc` binary substitution via PATH (user-controlled; not our threat)

---

## 1. Panic on malformed input = compiler bug

The compiler must never panic, crash with a Go stack trace, or produce
broken C given any input a user could type.

If a `.monk` file causes a Go panic, that is a bug in the compiler —
not a user error. Write a test for it:
`TestParserNoPanicOnX`, `TestCheckNoPanicOnY`.

Add a fuzzing note: any input that triggers a panic should become a
regression test immediately. Don't wait for a fuzzer to find it.

See `error-message-quality.md` for the full rule.

---

## 2. Shell injection in `cc` invocation

The `cc` command in `src/main.go` is assembled as a string slice:

```go
cmd := exec.Command("cc", "-O3", "-flto", "-o", outPath, cFile, "-lm")
```

This is safe **as long as all arguments are controlled by the compiler,
not by the user**. If you ever add:
- A pragma that lets `.monk` source pass flags to `cc`
- An env var like `MONK_CFLAGS` that's passed through
- A config file read at build time

...you must sanitize those values. Shell metacharacters (`$`, `;`, `&`,
`` ` ``, `|`, `(`, `)`) must not reach `cc` arguments. Use `exec.Command`
with separate args (already done — not `exec.Command("sh", "-c", ...)`)
and validate each user-supplied argument before passing it.

Rule: if you add user-controllable `cc` flags in the future, add a
whitelist of valid flag patterns and reject everything else.

---

## 3. Temp file hygiene

`monk run` creates a temp directory, compiles into it, runs the binary,
then cleans up. This must:
- Use `os.MkdirTemp("", "monk-run-*")` — random suffix, not predictable
- Clean up with `defer os.RemoveAll(dir)` that runs even on non-zero exit
  (the fix for the early `os.Exit` bug is already in — maintain it)
- Never put temp files in a fixed location like `/tmp/monk.c`

A predictable temp path is exploitable: another process can pre-create
the file and influence what the compiler writes.

---

## 4. Module import path traversal

`use "../../../etc/passwd" from somewhere` should never reach the
filesystem with a resolved absolute path outside the project tree.

`module.ResolvePath` canonicalizes import paths. Verify it:
- Resolves relative to the importing file's directory (correct)
- Does NOT allow `..` to escape the project root (check this)
- Rejects non-relative paths (paths not starting with `./` or `../`)
  with a clean error, not a filesystem lookup

If a future version adds URL-style imports or registry imports,
the resolver must be re-audited.

---

## 5. Generated C identifier safety

Monk variable names become C symbol names via `mangledName()`. The
scanner only allows identifiers matching `[a-zA-Z_][a-zA-Z0-9_]*`.
This means:
- No spaces, no special characters, no compiler extensions
- No `__attribute__`, `__asm__`, or other C extension prefixes
- No `/` or `.` that could form an include path

This is already enforced by the scanner. **Never relax this.** If a
future feature (e.g., string interpolation, raw identifiers) allows
unusual characters in names, audit the `mangledName()` function to
ensure the output is still a valid, safe C identifier.

---

## 6. Embedded runtime integrity

The C runtime (`src/runtime/`) is embedded into the compiler binary
via `go:embed`. It cannot be tampered with after the binary is built.
Users cannot inject into the runtime by placing files in expected
locations.

Maintain this property:
- Never download runtime files at build time or run time
- Never allow user code to specify an alternate runtime path
  (except for development workflows that are clearly opt-in)
- If `MONK_RUNTIME_DIR` env var support is added in the future,
  validate that the directory contains expected files and sizes
  before using it

---

## 7. Security in the C runtime itself

The embedded C runtime is trusted code. It must not:
- Call `system()` with user-supplied strings
- Call `dlopen()` / `dlsym()` on user-supplied paths
- Execute arbitrary code from a Monk value (there's no `eval`)
- Read files outside of explicit `read_file` / `write_file` builtins

If you add a new builtin that does I/O, it must:
- Take explicit paths from the Monk program (never construct paths internally)
- Use POSIX-safe functions (`open()` with `O_CREAT | O_EXCL` for new files)
- Emit an error (not crash) on permission denied or path not found

---

## Reporting findings

If a security issue is found:
1. Write a regression test immediately (even before fixing)
2. Fix it in the same session
3. Note it in PROGRESS.md under "Fixes" with the word "security:"
   so it's searchable in history

We don't have a formal CVE process at 0.x. That changes at 1.0.
