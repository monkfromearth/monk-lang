# Pre-Completion Checks — Mandatory Process

Before declaring any non-trivial code change "done" (refactor, bug fix,
feature work, anything touching compiled code paths), run the full check
battery below. Do NOT tell the user the work is complete until every step
has passed — or you have explicitly called out which steps failed and why.

This exists because: (a) Monk compiles to C via generated code, so bugs
surface at three layers (Go → C compilation → runtime behavior), and (b)
the harness re-embeds runtime files into the binary, so cross-cutting
changes silently break unless every channel is verified.

## The Battery (in order)

### 1. Formatting
```bash
gofmt -l src/
```
Expected output: empty, OR a list of pre-existing repo-style files
(test files using single-line `if x { do }` statements — those are the
project's own style, accept them). If the list includes a file you just
wrote fresh, reformat it.

### 2. Build
```bash
make build                # or: cd src && go build ./...
```
Expected: binary produced, no errors.

### 3. Static analysis — Go
```bash
cd src
go vet ./...              # stdlib analyzer (typed printf etc.)
staticcheck ./...         # honnef.co/go/tools — deadcode, ineffective assigns, etc.
golangci-lint run ./...   # aggregate linter suite
govulncheck ./...         # CVE scan against Go stdlib + deps
```
All four must print zero issues. If a linter flags something in code you
did NOT touch, grep git blame — if pre-existing, note it and move on. If
it's in your diff, fix it.

### 4. Full Go test suite
```bash
cd src && go test ./...
```
Expected: all packages `ok`. Not `cached` counting — the codegen tests
call `cc` under the hood, so a green `(cached)` means the last run passed
against the current sources. If you changed codegen or runtime, clear the
cache with `go clean -testcache` first.

### 5. C runtime tests
```bash
cd src/runtime
cc -std=c11 value.c arith.c string.c container.c math.c builtins.c error.c \
    runtime_test.c -lm -o /tmp/rt_test
/tmp/rt_test           # expect: "151/151 tests passed"
rm /tmp/rt_test
```
If the runtime file list changes, update this command AND the three sync
points: `src/embed.go`, `runtimeSources`+`embeddedRuntimeFiles` in
`src/main.go`, `runtimeTestSources` in `src/codegen/codegen_test.go`.

### 6. All examples
```bash
for f in examples/*.monk; do
  ./monk run "$f" > /tmp/monk_out 2>&1 && echo "ok  $f" || echo "FAIL $f"
done
```
All 13 must print `ok`.

### 7. Benchmark smoke test
```bash
for b in fibonacci mandelbrot trial_primes leibniz matmul; do
  ./monk run bench/benchmarks/$b/$b.monk
done
```
Each prints a single integer. Compare with `expected.txt` in that
benchmark's directory:
- fibonacci: 9227465
- mandelbrot: 169799
- trial_primes: 17984
- leibniz: 31415926
- matmul: 21333200

### 8. CodeRabbit review of uncommitted changes
```bash
coderabbit review --plain -t uncommitted > /tmp/cr_review.txt
cat /tmp/cr_review.txt
```
Apply the `.claude/rules/code-review-workflow.md` protocol to every
finding: build the verdict table (ACT / SKIP / DISCUSS), present it,
then implement the ACT items. Add regression tests for every ACT item
that fixes a real bug.

### 9. Update `INDEX.md` for every touched directory
If files were added, moved, deleted, or had their public surface
materially change in `src/syntax/`, `src/codegen/`, `src/runtime/`,
update the corresponding `INDEX.md` in the same set of changes.

## Reporting

When you declare the work complete, report the battery results:

```
Checks:
  gofmt            clean (N pre-existing style noise — not mine)
  build            ok
  go vet           clean
  staticcheck      clean
  golangci-lint    0 issues
  govulncheck      clean
  go test ./...    all packages ok
  C runtime        151/151
  examples         13/13
  benchmarks       5/5 matched expected
  coderabbit       N findings → M ACT (fixed) / S SKIP / D DISCUSS
```

If any step fails, say so explicitly before declaring done. Never
silently drop failures.

## Exceptions

- **Trivial doc-only edits** (typo fix, comment tweak): skip steps 3–8.
  Always run 1 and 9.
- **Tests-only edits**: run 1, 2, 4, 5. Skip 6–8 unless you changed a
  test harness that other tests use.
- **Under active iteration** (explicit "WIP, not done yet" messages):
  use step 2 + 4 in the inner loop. Run the full battery before
  declaring done.
