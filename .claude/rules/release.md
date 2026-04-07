# Release Process

Every release is a named, versioned snapshot of a working compiler.
Releases are small, frequent, and always in a releasable state —
not big-bang drops after months of work.

---

## Versioning

`MAJOR.MINOR.PATCH` — semantic versioning. We are in `0.x.y` (pre-1.0).
Breaking changes are expected and don't require a MAJOR bump until 1.0.

| Increment | When |
|-----------|------|
| `0.MINOR.0` | A phase completes, new language features land |
| `0.MINOR.PATCH` | Bug fixes, doc improvements, hardening passes |
| `1.0.0` | Language spec is stable, no more planned breaking changes |

---

## Release names

Every `0.MINOR.0` release gets an Urdu/Hindi codename. Single word.
Chosen to reflect the character of the release — not assigned in advance.

**Word bank** (unused):
*Safar* (journey) · *Noor* (light) · *Umeed* (hope) · *Fikr* (thought) ·
*Sukoon* (peace) · *Irada* (will) · *Khoj* (discovery) · *Raasta* (path) ·
*Dastak* (arrival) · *Ehsaas* (awareness) · *Amal* (action)

**Selection criteria:**
- What is the theme of this release? (foundation, discovery, hope...)
- Does the word's meaning fit? (Buniyaad = foundation → v0.0.1, the first build)
- Is it easy to say and remember?
- Don't overthink it. Ask: "If I told someone 'this is the Noor release',
  does it evoke the right feeling?"

Patch releases (`0.MINOR.PATCH`) do NOT get a new name. They carry
the name of their minor version.

---

## Release checklist

### Before branching

- [ ] Full pre-completion battery passes (see `pre-completion-checks.md`)
- [ ] All `ROADMAP.md` items for this phase are checked
- [ ] No open DISCUSS items from the last senior review
- [ ] `PROGRESS.md` is up to date

### Version and name

1. Decide the version: `0.MINOR.0` or `0.MINOR.PATCH`
2. If `0.MINOR.0`: choose the release name from the word bank above
3. Update the version string in `src/main.go`:
   ```go
   fmt.Println("monk 0.X.Y — ReleaseName")
   ```
4. Remove the used name from the word bank in `CLAUDE.md`

### Changelog

Write the CHANGELOG entry in `CHANGELOG.md`. Format:

```markdown
## 0.X.Y — ReleaseName (YYYY-MM-DD)

### Language
- What the user can now write that they couldn't before

### Performance
- Before → After → vs C numbers (include hardware)

### Compiler
- Internal improvements that affect reliability or speed

### Fixes
- Bug fixes with one-line description each
```

Write for a developer reading a year from now. Not for the commit log.
Skip internal refactors unless they affect observable behavior.

### Commit and tag

```bash
git add src/main.go CHANGELOG.md PROGRESS.md ROADMAP.md
git commit -m "release: v0.X.Y — ReleaseName"
git tag v0.X.Y
git push && git push --tags
```

### Build the release binary

```bash
make clean && make build
# macOS arm64 (default)
GOOS=darwin GOARCH=arm64 go build -o dist/monk-darwin-arm64 ./src/...
# macOS amd64
GOOS=darwin GOARCH=amd64 go build -o dist/monk-darwin-amd64 ./src/...
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o dist/monk-linux-amd64 ./src/...
```

Verify each binary:
```bash
./dist/monk-darwin-arm64 version   # should print "monk 0.X.Y — ReleaseName"
```

### GitHub release

```bash
gh release create v0.X.Y \
  dist/monk-darwin-arm64 \
  dist/monk-darwin-amd64 \
  dist/monk-linux-amd64 \
  --title "monk 0.X.Y — ReleaseName" \
  --notes "$(sed -n '/## 0.X.Y/,/## 0\./p' CHANGELOG.md | head -n -1)"
```

### Homebrew tap (Phase 9+)

Until Phase 9, Homebrew tap is not maintained. After Phase 9:
1. Update `monkfromearth/homebrew-monk-lang` Formula
2. Update SHA256 for each binary
3. Update version string in formula
4. Test: `brew upgrade monk-lang && monk version`

---

## What makes a good release

- **Working end to end.** Every example runs. Every test passes.
  No known regressions.
- **A clear story.** The release notes tell one story: "This release
  adds X. Here's what that means." Not a list of commits.
- **Binary ships.** Someone with no Go toolchain can download and run it.
- **Docs are current.** `spec/REFERENCE.md`, WALKTHROUGH.md, knowledge/
  all reflect what the binary actually does.

---

## What is NOT a release blocker

- Known bugs that have workarounds (document them in the release notes)
- Missing future-phase features
- Performance gaps that are acknowledged and documented
- knowledge/ pages that haven't been updated yet for a niche feature
