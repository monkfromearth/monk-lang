---
name: release
description: >
  End-to-end release workflow for Monk. Determines version and name,
  runs the pre-flight battery, writes CHANGELOG, bumps version string,
  tags, builds cross-platform binaries, and creates the GitHub release.
  Follows the process in release.md exactly.
allowed-tools: Bash, Read, Grep, Glob, Edit, Write
---

# Release

You are cutting a Monk release. Be deliberate. A release is permanent.

Read `CLAUDE.md` (release names word bank), `CHANGELOG.md` (existing
format), and `PROGRESS.md` (what changed since last release) before
doing anything else.

---

## Step 1 — Determine version and name

Ask:

1. What changed since the last release?
   ```bash
   git log $(git describe --tags --abbrev=0)..HEAD --oneline
   ```

2. Is this a phase completion (`0.MINOR.0`) or a fix/hardening pass (`0.MINOR.PATCH`)?
   - New language features or compiler phases → `0.MINOR.0`
   - Bug fixes, tests, documentation, hardening → `0.MINOR.PATCH`

3. If `0.MINOR.0`: what is the theme of this release? Present 2-3 name
   options from the word bank with a one-line rationale for each.
   Wait for the user to choose.

4. Confirm version + name with the user before proceeding.

---

## Step 2 — Pre-flight

All of this must be green before touching any files:

```bash
# Build
make build

# Full test suite (clear cache — we're measuring the current state)
cd src && go clean -testcache && go test ./...

# C runtime
RT_SRCS=$(ls src/runtime/*.c | xargs)
cc -std=c11 $RT_SRCS -lm -o /tmp/rt_release && /tmp/rt_release && rm /tmp/rt_release

# All examples
PASS=0; FAIL=0
for f in examples/*.monk; do
  ./monk run "$f" > /tmp/out 2>&1 && { echo "ok $f"; PASS=$((PASS+1)); } || { echo "FAIL $f"; cat /tmp/out; FAIL=$((FAIL+1)); }
done
echo "$PASS/$(ls examples/*.monk | wc -l | tr -d ' ') passed"

# All benchmarks match expected
PASS=0; FAIL=0
for dir in bench/benchmarks/*/; do
  name=$(basename "$dir"); monk_file="$dir$name.monk"; expected="${dir}expected.txt"
  [ -f "$monk_file" ] || continue
  exp=$(cat "$expected" 2>/dev/null | tr -d '[:space:]')
  got=$(./monk run "$monk_file" 2>&1 | tr -d '[:space:]')
  [ "$got" = "$exp" ] && { echo "ok $name"; PASS=$((PASS+1)); } || { echo "FAIL $name"; FAIL=$((FAIL+1)); }
done
echo "$PASS/$((PASS+FAIL)) benchmarks matched"
```

Do NOT proceed if anything fails. Fix it first.

---

## Step 3 — Update files (in this order)

### 3a. Version string (`src/main.go:55`)
```go
// Change this line:
fmt.Println("monk 0.OLD — OldName")
// To:
fmt.Println("monk 0.X.Y — NewName")
```

### 3b. CHANGELOG.md
Add a new section at the top, ABOVE "Unreleased":

```markdown
## 0.X.Y — ReleaseName (YYYY-MM-DD)

### Language
[what users can now write]

### Performance  
[before → after → vs C, with hardware note]

### Compiler
[internal improvements affecting reliability/speed]

### Fixes
[bug fixes, one-liner each]
```

Write for a developer reading a year from now. No commit-log noise.
Move items out of "Unreleased" into this section.

### 3c. ROADMAP.md
Check off any items that are complete in this release.

### 3d. CLAUDE.md word bank
Remove the used release name from the word bank.

---

## Step 4 — Verify the version string

```bash
make build && ./monk version
# Expected: monk 0.X.Y — ReleaseName
```

---

## Step 5 — Commit and tag

```bash
git add src/main.go CHANGELOG.md ROADMAP.md CLAUDE.md PROGRESS.md
git status   # confirm only expected files staged
git commit -m "release: v0.X.Y — ReleaseName"
git tag v0.X.Y
```

Show the user the commit and tag before pushing. Wait for "go ahead".

```bash
git push && git push --tags
```

---

## Step 6 — Build release binaries

```bash
mkdir -p dist

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o dist/monk-darwin-arm64 ./...
# macOS amd64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o dist/monk-darwin-amd64 ./...
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o dist/monk-linux-amd64 ./...
```

Run from `src/`:
```bash
cd src
GOOS=darwin GOARCH=arm64 go build -o ../dist/monk-darwin-arm64 .
# etc.
```

Verify each binary prints the right version:
```bash
./dist/monk-darwin-arm64 version
./dist/monk-darwin-amd64 version
./dist/monk-linux-amd64 version
```

---

## Step 7 — GitHub release

Extract the CHANGELOG section for this version:
```bash
# Read CHANGELOG.md and extract the section for this version
```

Create the release:
```bash
gh release create v0.X.Y \
  dist/monk-darwin-arm64 \
  dist/monk-darwin-amd64 \
  dist/monk-linux-amd64 \
  --title "monk 0.X.Y — ReleaseName" \
  --notes-file /tmp/release_notes.md
```

---

## Step 8 — Post-release

```bash
# Confirm the release is live
gh release view v0.X.Y
```

Update PROGRESS.md with one line: "Released v0.X.Y — ReleaseName on YYYY-MM-DD."

**Homebrew tap** (Phase 9+ only):
If Phase 9 is complete, update `monkfromearth/homebrew-monk-lang` with
new SHA256 hashes and version. Test: `brew upgrade monk-lang && monk version`.

---

## What to do if something fails mid-process

- **Pre-flight fails:** Do not proceed. Fix the failure, re-run the battery.
- **Build fails during cross-compile:** Check `GOOS`/`GOARCH` values.
  Ensure no `cgo` in the binary (`CGO_ENABLED=0` if needed).
- **Tag already exists:** `git tag -d v0.X.Y` to delete local, check
  if remote needs it too. Don't overwrite a pushed tag — it confuses
  anyone who pulled it.
- **GitHub release fails:** The tag is pushed, so the code is safe.
  Retry `gh release create` with the binaries.
