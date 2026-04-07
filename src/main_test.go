package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildMonk builds the monk binary to a temp directory and returns its path.
func buildMonk(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "monk")
	// src/ is the module root; "go build ." from here builds the CLI
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = srcDir(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build monk: %s\n%s", err, out)
	}
	return bin
}

// srcDir returns the absolute path to src/ (the Go module root).
func srcDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// projectRoot returns the repo root (one level above src/).
func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// runMonkCmd runs the monk binary with args, returns stdout and exit code.
func runMonkCmd(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(), "MONK_RUNTIME_DIR="+filepath.Join(srcDir(t), "runtime"))
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return strings.TrimRight(stdout.String(), "\n"), strings.TrimRight(stderr.String(), "\n"), exitCode
}

// writeMonk writes a .monk file to a temp directory and returns its path.
func writeMonk(t *testing.T, name, source string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// === monk version ===

func TestVersion(t *testing.T) {
	bin := buildMonk(t)
	stdout, _, code := runMonkCmd(t, bin, "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "monk") {
		t.Errorf("expected version output, got %q", stdout)
	}
}

// === monk help ===

func TestHelp(t *testing.T) {
	bin := buildMonk(t)
	_, stderr, code := runMonkCmd(t, bin, "help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stderr, "Usage") {
		t.Errorf("expected usage text, got %q", stderr)
	}
}

func TestNoArgs(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin)
	if code == 0 {
		t.Error("expected non-zero exit for no args")
	}
}

func TestUnknownCommand(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin, "bogus")
	if code == 0 {
		t.Error("expected non-zero exit for unknown command")
	}
}

// === monk check ===

func TestCheckValid(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "valid.monk", `let x = 42`)
	_, stderr, code := runMonkCmd(t, bin, "check", src)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr)
	}
	if !strings.Contains(stderr, "ok") {
		t.Errorf("expected 'ok' in output, got %q", stderr)
	}
}

func TestCheckInvalid(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "invalid.monk", `let = 42`)
	_, _, code := runMonkCmd(t, bin, "check", src)
	if code == 0 {
		t.Error("expected non-zero exit for invalid source")
	}
}

func TestCheckMissingFile(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin, "check", "/nonexistent/file.monk")
	if code == 0 {
		t.Error("expected non-zero exit for missing file")
	}
}

func TestCheckNoFile(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin, "check")
	if code == 0 {
		t.Error("expected non-zero exit for missing argument")
	}
}

// === monk build ===

func TestBuildHello(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("Hello, World!")`)
	dir := filepath.Dir(src)
	outBin := filepath.Join(dir, "hello")

	_, stderr, code := runMonkCmd(t, bin, "build", src, "-o", outBin)
	if code != 0 {
		t.Fatalf("build failed (exit %d): %s", code, stderr)
	}

	// Check binary exists
	if _, err := os.Stat(outBin); err != nil {
		t.Fatalf("binary not created: %v", err)
	}

	// Run the binary
	cmd := exec.Command(outBin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", string(out))
	}
}

func TestBuildDefaultOutput(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "test.monk", `show("ok")`)
	dir := filepath.Dir(src)
	expectedBin := filepath.Join(dir, "test")

	_, _, code := runMonkCmd(t, bin, "build", src)
	if code != 0 {
		t.Fatalf("build failed (exit %d)", code)
	}

	// Default output should strip .monk extension
	if _, err := os.Stat(expectedBin); err != nil {
		t.Fatalf("default binary not created at %s: %v", expectedBin, err)
	}
}

func TestBuildOutputC(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("Hello!")`)
	dir := filepath.Dir(src)
	cOut := filepath.Join(dir, "hello.c")

	_, _, code := runMonkCmd(t, bin, "build", src, "-o", cOut)
	if code != 0 {
		t.Fatalf("build -o .c failed (exit %d)", code)
	}

	content, err := os.ReadFile(cOut)
	if err != nil {
		t.Fatalf("expected .c file: %v", err)
	}
	if !strings.Contains(string(content), "monk_show") {
		t.Error("generated C should contain monk_show")
	}
}

func TestBuildMissingFile(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin, "build", "/nonexistent.monk")
	if code == 0 {
		t.Error("expected non-zero exit for missing file")
	}
}

func TestBuildInvalidSource(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "bad.monk", `let = `)
	_, _, code := runMonkCmd(t, bin, "build", src)
	if code == 0 {
		t.Error("expected non-zero exit for invalid source")
	}
}

func TestBuildNoFile(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin, "build")
	if code == 0 {
		t.Error("expected non-zero exit for missing argument")
	}
}

// === monk run ===

func TestRunHello(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("Hello from run!")`)
	stdout, _, code := runMonkCmd(t, bin, "run", src)
	if code != 0 {
		t.Fatalf("run failed (exit %d)", code)
	}
	if stdout != "Hello from run!" {
		t.Errorf("expected 'Hello from run!', got %q", stdout)
	}
}

func TestRunArithmetic(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "math.monk", `show(to_string(2 + 3 * 4))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "14" {
		t.Errorf("expected '14', got %q", stdout)
	}
}

func TestRunVariables(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "vars.monk", `let x = 10
x += 5
show(to_string(x))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "15" {
		t.Errorf("expected '15', got %q", stdout)
	}
}

func TestRunFunction(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "func.monk", `let square = (x int) int { return x * x }
show(to_string(square(7)))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "49" {
		t.Errorf("expected '49', got %q", stdout)
	}
}

func TestRunForLoop(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "loop.monk", `let sum = 0
for i in range(5) { sum += i }
show(to_string(sum))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "10" {
		t.Errorf("expected '10', got %q", stdout)
	}
}

func TestRunGuardThrow(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "guard.monk", `let fail = () int { throw "oops" }
guard result = fail() against error {
    result = -1
}
show(to_string(result))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "-1" {
		t.Errorf("expected '-1', got %q", stdout)
	}
}

func TestRunRecords(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "records.monk", `let p = {name: "Alice", age: 30}
show(p.name)
show(to_string(p.age))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "Alice\n30" {
		t.Errorf("expected 'Alice\\n30', got %q", stdout)
	}
}

func TestRunRecursion(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "fib.monk", `let fib = (n int) int {
    if n <= 1 { return n }
    return fib(n - 1) + fib(n - 2)
}
show(to_string(fib(10)))`)
	stdout, _, _ := runMonkCmd(t, bin, "run", src)
	if stdout != "55" {
		t.Errorf("expected '55', got %q", stdout)
	}
}

func TestRunMissingFile(t *testing.T) {
	bin := buildMonk(t)
	_, _, code := runMonkCmd(t, bin, "run", "/nonexistent.monk")
	if code == 0 {
		t.Error("expected non-zero exit for missing file")
	}
}

func TestRunInvalidSource(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "bad.monk", `let = `)
	_, _, code := runMonkCmd(t, bin, "run", src)
	if code == 0 {
		t.Error("expected non-zero exit for invalid source")
	}
}

// === monk run with examples/ ===

func TestRunExampleHello(t *testing.T) {
	bin := buildMonk(t)
	examplePath := filepath.Join(projectRoot(t), "examples", "hello.monk")
	stdout, _, code := runMonkCmd(t, bin, "run", examplePath)
	if code != 0 {
		t.Fatalf("run failed (exit %d)", code)
	}
	if stdout != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", stdout)
	}
}

func TestRunExampleFibonacci(t *testing.T) {
	bin := buildMonk(t)
	examplePath := filepath.Join(projectRoot(t), "examples", "fibonacci.monk")
	stdout, _, code := runMonkCmd(t, bin, "run", examplePath)
	if code != 0 {
		t.Fatalf("run failed (exit %d)", code)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 lines, got %d", len(lines))
	}
	if lines[0] != "0" || lines[1] != "1" || lines[9] != "34" {
		t.Errorf("unexpected fibonacci output: %s", stdout)
	}
}

func TestRunExampleFizzBuzz(t *testing.T) {
	bin := buildMonk(t)
	examplePath := filepath.Join(projectRoot(t), "examples", "fizzbuzz.monk")
	stdout, _, code := runMonkCmd(t, bin, "run", examplePath)
	if code != 0 {
		t.Fatalf("run failed (exit %d)", code)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) != 20 {
		t.Fatalf("expected 20 lines, got %d", len(lines))
	}
	if lines[0] != "1" {
		t.Errorf("line 1: expected '1', got %q", lines[0])
	}
	if lines[2] != "Fizz" {
		t.Errorf("line 3: expected 'Fizz', got %q", lines[2])
	}
	if lines[4] != "Buzz" {
		t.Errorf("line 5: expected 'Buzz', got %q", lines[4])
	}
	if lines[14] != "FizzBuzz" {
		t.Errorf("line 15: expected 'FizzBuzz', got %q", lines[14])
	}
}

func TestRunExampleErrorHandling(t *testing.T) {
	bin := buildMonk(t)
	examplePath := filepath.Join(projectRoot(t), "examples", "error_handling.monk")
	stdout, _, code := runMonkCmd(t, bin, "run", examplePath)
	if code != 0 {
		t.Fatalf("run failed (exit %d)", code)
	}
	if !strings.Contains(stdout, "10 / 3 = 3") {
		t.Errorf("expected '10 / 3 = 3', got %q", stdout)
	}
	if !strings.Contains(stdout, "Caught: division by zero") {
		t.Errorf("expected catch message, got %q", stdout)
	}
}

// === Type checker integration ===
// monk check should reject programs that parse but fail type checking.

func TestCheckRejectsTypeMismatch(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "bad.monk", `let x = 42
x = "hi"`)
	_, stderr, code := runMonkCmd(t, bin, "check", src)
	if code == 0 {
		t.Fatal("expected type error, got exit 0")
	}
	if !strings.Contains(stderr, "cannot assign string to variable 'x'") {
		t.Errorf("expected type-mismatch message, got %q", stderr)
	}
}

func TestBuildRejectsTypeMismatch(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "bad.monk", `let nums int[] = [1, 2, 3]
nums[0] = "x"`)
	outBin := filepath.Join(filepath.Dir(src), "bad")
	_, stderr, code := runMonkCmd(t, bin, "build", src, "-o", outBin)
	if code == 0 {
		t.Fatal("expected type error, got exit 0")
	}
	if !strings.Contains(stderr, "type error") {
		t.Errorf("expected 'type error' in stderr, got %q", stderr)
	}
	// Binary must not be produced when the type check fails.
	if _, err := os.Stat(outBin); err == nil {
		t.Error("binary produced despite type error")
	}
}

// `monk run` must clean up its temp compile dir even when the child program
// exits non-zero. Previously os.Exit skipped the deferred RemoveAll, leaking
// /tmp/monk-run-* directories over time.
func TestRunCleansUpTempDirOnNonZeroExit(t *testing.T) {
	bin := buildMonk(t)
	// exit(7) from the child triggers the non-zero return path.
	src := writeMonk(t, "fail.monk", `exit(7)`)

	// Snapshot existing monk-run-* dirs in TMPDIR before running.
	tmpRoot := os.TempDir()
	before, _ := filepath.Glob(filepath.Join(tmpRoot, "monk-run-*"))
	beforeSet := make(map[string]bool, len(before))
	for _, d := range before {
		beforeSet[d] = true
	}

	_, _, code := runMonkCmd(t, bin, "run", src)
	if code != 7 {
		t.Fatalf("expected child exit code 7 to propagate, got %d", code)
	}

	// No new monk-run-* dir should remain.
	after, _ := filepath.Glob(filepath.Join(tmpRoot, "monk-run-*"))
	for _, d := range after {
		if !beforeSet[d] {
			t.Errorf("leaked temp dir: %s", d)
		}
	}
}

// === -o flag edge cases ===

// Dangling -o (no trailing value) should error, not silently ignore.
func TestBuildDashOMissingValue(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("hi")`)
	_, stderr, code := runMonkCmd(t, bin, "build", src, "-o")
	if code == 0 {
		t.Fatal("expected non-zero exit when -o has no value")
	}
	if !strings.Contains(stderr, "-o requires an output path") {
		t.Errorf("expected '-o requires an output path' error, got %q", stderr)
	}
}

// -o placed before the source file is a user error. The loop only scans from
// args[1:] for flags, so -o is treated as the source path and fails to read.
// This test documents that behavior so a future refactor doesn't regress it
// silently (e.g. by making args[0] = "-o" swallow "out" as the source).
func TestBuildDashOBeforeSource(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("hi")`)
	outBin := filepath.Join(filepath.Dir(src), "out")
	_, _, code := runMonkCmd(t, bin, "build", "-o", outBin, src)
	if code == 0 {
		t.Error("expected non-zero exit when -o precedes source file")
	}
	if _, err := os.Stat(outBin); err == nil {
		t.Error("binary should not be created when -o comes before source")
	}
	// Intermediate .c file must not leak either — a partial build that wrote
	// the .c but failed compilation would be a real bug to catch here.
	if _, err := os.Stat(outBin + ".c"); err == nil {
		t.Error("intermediate .c file should not be left behind")
	}
}

// A source file without the .monk extension must not have its default output
// path collide with the source itself (that would overwrite the input).
func TestBuildNonMonkExtension(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.txt", `show("ok")`)
	_, _, code := runMonkCmd(t, bin, "build", src)
	if code != 0 {
		t.Fatalf("expected build to succeed, got exit %d", code)
	}
	// Default output must be sourceFile + ".out", not the source itself.
	if _, err := os.Stat(src + ".out"); err != nil {
		t.Fatalf("expected %s.out to be created: %v", src, err)
	}
	// Source file must remain readable as the original .monk text.
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("source file missing: %v", err)
	}
	if !strings.Contains(string(content), `show("ok")`) {
		t.Errorf("source file was overwritten: %q", string(content))
	}
}

// -o with a nested path should create the parent directories, not fail.
func TestBuildNestedOutputPath(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("nested")`)
	outBin := filepath.Join(filepath.Dir(src), "build", "bin", "app")
	_, stderr, code := runMonkCmd(t, bin, "build", src, "-o", outBin)
	if code != 0 {
		t.Fatalf("build to nested path failed (exit %d): %s", code, stderr)
	}
	if _, err := os.Stat(outBin); err != nil {
		t.Fatalf("nested binary not created: %v", err)
	}
	out, err := exec.Command(outBin).Output()
	if err != nil {
		t.Fatalf("nested binary failed to run: %v", err)
	}
	if strings.TrimSpace(string(out)) != "nested" {
		t.Errorf("nested binary output wrong: %q", string(out))
	}
}

// Nested -o with .c extension should also create parent dirs.
func TestBuildNestedOutputCPath(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("hi")`)
	cOut := filepath.Join(filepath.Dir(src), "gen", "src", "hello.c")
	_, _, code := runMonkCmd(t, bin, "build", src, "-o", cOut)
	if code != 0 {
		t.Fatalf("build -o nested.c failed (exit %d)", code)
	}
	if _, err := os.Stat(cOut); err != nil {
		t.Fatalf("nested .c file not created: %v", err)
	}
}

// === MONK_RUNTIME_DIR handling ===

// runMonkCmdEnv is like runMonkCmd but lets the caller override env/workdir.
func runMonkCmdEnv(t *testing.T, bin, workdir string, env []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	cmd.Env = env
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return strings.TrimRight(stdout.String(), "\n"), strings.TrimRight(stderr.String(), "\n"), exitCode
}

// cleanEnv returns an env with MONK_RUNTIME_DIR stripped, HOME overridden, and
// PATH kept (we need cc). Used for testing fallback to embedded runtime.
func cleanEnv(home string) []string {
	out := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "MONK_RUNTIME_DIR=") {
			continue
		}
		if strings.HasPrefix(kv, "HOME=") {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "HOME="+home)
	return out
}

// If MONK_RUNTIME_DIR points at a directory without runtime.h, the CLI must
// fall through to the embedded runtime (not fail or use the bad dir).
// Tests the validation added in commit a4b639a. If a future change starts
// warning/erroring on the invalid value, update this test deliberately.
func TestMonkRuntimeDirInvalidFallsThrough(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("fallback")`)
	workdir := t.TempDir()     // no runtime/ here
	fakeRuntime := t.TempDir() // exists but empty — no runtime.h
	home := t.TempDir()

	env := cleanEnv(home)
	env = append(env, "MONK_RUNTIME_DIR="+fakeRuntime)

	outBin := filepath.Join(t.TempDir(), "app")
	_, stderr, code := runMonkCmdEnv(t, bin, workdir, env, "build", src, "-o", outBin)
	if code != 0 {
		t.Fatalf("build should fall through to embedded runtime, got exit %d: %s", code, stderr)
	}
	if _, err := os.Stat(outBin); err != nil {
		t.Fatalf("binary not created: %v", err)
	}
	// The stderr path the invalid MONK_RUNTIME_DIR took should not leak through
	// as a warning referencing the bad path (current contract: silent fallback).
	if strings.Contains(stderr, fakeRuntime) {
		t.Errorf("stderr leaked invalid MONK_RUNTIME_DIR path %q: %q", fakeRuntime, stderr)
	}
	// Embedded runtime should have been extracted to the fake HOME cache.
	cached := filepath.Join(home, ".cache", "monk", "runtime", "runtime.h")
	if _, err := os.Stat(cached); err != nil {
		t.Errorf("expected embedded runtime extracted to %s: %v", cached, err)
	}
}

// The cache writer must skip writes when content already matches, to avoid
// gratuitous mtime bumps and races between concurrent `monk build` invocations.
func TestEmbeddedRuntimeCacheIsIdempotent(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("cached")`)
	workdir := t.TempDir()
	home := t.TempDir()
	env := cleanEnv(home)

	outBin := filepath.Join(t.TempDir(), "app")

	// First build — extracts runtime to cache.
	if _, stderr, code := runMonkCmdEnv(t, bin, workdir, env, "build", src, "-o", outBin); code != 0 {
		t.Fatalf("first build failed (exit %d): %s", code, stderr)
	}
	cached := filepath.Join(home, ".cache", "monk", "runtime", "runtime.h")
	info1, err := os.Stat(cached)
	if err != nil {
		t.Fatalf("runtime.h not cached: %v", err)
	}

	// Force the mtime backward so we can detect a rewrite.
	past := info1.ModTime().Add(-2 * time.Hour)
	if err := os.Chtimes(cached, past, past); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// Second build with identical content — must not rewrite.
	if _, stderr, code := runMonkCmdEnv(t, bin, workdir, env, "build", src, "-o", outBin); code != 0 {
		t.Fatalf("second build failed (exit %d): %s", code, stderr)
	}
	info2, err := os.Stat(cached)
	if err != nil {
		t.Fatalf("runtime.h gone: %v", err)
	}
	if !info2.ModTime().Equal(past) {
		t.Errorf("runtime.h was rewritten; mtime changed from %v to %v", past, info2.ModTime())
	}

	// Now corrupt it — next build must rewrite.
	if err := os.WriteFile(cached, []byte("// corrupted"), 0644); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if _, _, code := runMonkCmdEnv(t, bin, workdir, env, "build", src, "-o", outBin); code != 0 {
		t.Fatal("rebuild after corruption failed")
	}
	content, err := os.ReadFile(cached)
	if err != nil || strings.Contains(string(content), "corrupted") {
		t.Error("corrupted runtime.h was not restored")
	}
}

// With no MONK_RUNTIME_DIR and no runtime/ nearby, the embedded runtime should
// be extracted to $HOME/.cache/monk/runtime/ on first build. This exercises the
// cold-start path that users hit on fresh installs.
func TestEmbeddedRuntimeExtraction(t *testing.T) {
	bin := buildMonk(t)
	src := writeMonk(t, "hello.monk", `show("embedded")`)
	workdir := t.TempDir() // no runtime/ subdirectory
	home := t.TempDir()    // empty cache

	env := cleanEnv(home) // strips MONK_RUNTIME_DIR

	outBin := filepath.Join(t.TempDir(), "app")
	_, stderr, code := runMonkCmdEnv(t, bin, workdir, env, "build", src, "-o", outBin)
	if code != 0 {
		t.Fatalf("build with embedded runtime failed (exit %d): %s", code, stderr)
	}

	// Cache should now contain both runtime files.
	cacheDir := filepath.Join(home, ".cache", "monk", "runtime")
	for _, name := range []string{"runtime.h", "internal.h", "value.c", "arith.c", "string.c", "container.c", "math.c", "builtins.c", "error.c"} {
		p := filepath.Join(cacheDir, name)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected extracted %s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("extracted %s is empty", name)
		}
	}

	// Compiled binary should run and print the expected output.
	out, err := exec.Command(outBin).Output()
	if err != nil {
		t.Fatalf("embedded-runtime binary failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "embedded" {
		t.Errorf("expected 'embedded', got %q", string(out))
	}
}

// ─── Module system integration tests ──────────────────────────────────────────

// writeMonkFiles writes multiple .monk files to a temp directory and returns
// the directory path. Keys are filenames (e.g. "main.monk", "lib/utils.monk").
func writeMonkFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, source := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunModuleBasicImport(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"math.monk": `let add = (a int, b int) int { return a + b }
export add`,
		"main.monk": `use add from "./math"
show(to_string(add(3, 4)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "7" {
		t.Errorf("expected '7', got %q", stdout)
	}
}

func TestRunModuleMultipleImports(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"math.monk": `let add = (a int, b int) int { return a + b }
export add`,
		"str.monk": `let greet = (name string) string { return "Hello, " + name }
export greet`,
		"main.monk": `use add from "./math"
use greet from "./str"
show(to_string(add(10, 20)))
show(greet("World"))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "30\nHello, World" {
		t.Errorf("expected '30\\nHello, World', got %q", stdout)
	}
}

func TestRunModuleStarImport(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `let a = 10
export a
let b = 20
export b`,
		"main.monk": `use * from "./lib"
show(to_string(a))
show(to_string(b))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "10\n20" {
		t.Errorf("expected '10\\n20', got %q", stdout)
	}
}

func TestRunModuleAliasImport(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"math.monk": `let multiply = (a int, b int) int { return a * b }
export multiply`,
		"main.monk": `use multiply as mul from "./math"
show(to_string(mul(6, 7)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "42" {
		t.Errorf("expected '42', got %q", stdout)
	}
}

func TestRunModuleDestructuredImport(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"ops.monk": `let add = (a int, b int) int { return a + b }
export add
let sub = (a int, b int) int { return a - b }
export sub`,
		"main.monk": `use { add, sub } from "./ops"
show(to_string(add(10, 3)))
show(to_string(sub(10, 3)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "13\n7" {
		t.Errorf("expected '13\\n7', got %q", stdout)
	}
}

func TestRunModuleInitOnce(t *testing.T) {
	// Module-level side effect (show) should run exactly once even when
	// imported by multiple modules.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"shared.monk": `show("init")
let X = 42
export X`,
		"a.monk": `use X from "./shared"
let aval = X
export aval`,
		"b.monk": `use X from "./shared"
let bval = X
export bval`,
		"main.monk": `use aval from "./a"
use bval from "./b"
show(to_string(aval))
show(to_string(bval))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	// "init" should appear exactly once.
	if stdout != "init\n42\n42" {
		t.Errorf("expected 'init\\n42\\n42', got %q", stdout)
	}
}

func TestRunModuleExportConst(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"constants.monk": `const PI = 3
export PI`,
		"main.monk": `use PI from "./constants"
show(to_string(PI))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "3" {
		t.Errorf("expected '3', got %q", stdout)
	}
}

func TestCheckModuleCircularError(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"a.monk": `use y from "./b"`,
		"b.monk": `use x from "./a"`,
	})
	_, stderr, code := runMonkCmd(t, bin, "check", filepath.Join(dir, "a.monk"))
	if code == 0 {
		t.Fatal("expected non-zero exit for circular import")
	}
	if !strings.Contains(stderr, "circular import") {
		t.Errorf("expected 'circular import' in error, got %q", stderr)
	}
}

func TestCheckModuleMissingExport(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk":  `let x = 1`,
		"main.monk": `use nope from "./lib"`,
	})
	_, stderr, code := runMonkCmd(t, bin, "check", filepath.Join(dir, "main.monk"))
	if code == 0 {
		t.Fatal("expected non-zero exit for missing export")
	}
	if !strings.Contains(stderr, "does not export") {
		t.Errorf("expected 'does not export' in error, got %q", stderr)
	}
}

func TestRunModuleSubdirectory(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib/utils.monk": `let double = (x int) int { return x * 2 }
export double`,
		"main.monk": `use double from "./lib/utils"
show(to_string(double(21)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "42" {
		t.Errorf("expected '42', got %q", stdout)
	}
}

func TestRunModuleExportFunction(t *testing.T) {
	// Test that exported functions with closures work across modules.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"counter.monk": `let make_adder = (base int) (int) -> int {
	let adder = (x int) int { return base + x }
	return adder
}
export make_adder`,
		"main.monk": `use make_adder from "./counter"
let add10 = make_adder(10)
show(to_string(add10(5)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "15" {
		t.Errorf("expected '15', got %q", stdout)
	}
}

func TestBuildModuleOutputC(t *testing.T) {
	// monk build with -o .c should produce valid C source for multi-module programs.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `let greet = () string { return "hello" }
export greet`,
		"main.monk": `use greet from "./lib"
show(greet())`,
	})
	outC := filepath.Join(t.TempDir(), "out.c")
	_, _, code := runMonkCmd(t, bin, "build", filepath.Join(dir, "main.monk"), "-o", outC)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	content, err := os.ReadFile(outC)
	if err != nil {
		t.Fatal(err)
	}
	// The C source should contain the module init function and main.
	if !strings.Contains(string(content), "_mod_0_init") {
		t.Error("expected _mod_0_init in generated C")
	}
	if !strings.Contains(string(content), "int main(void)") {
		t.Error("expected main() in generated C")
	}
}

func TestRunModuleExportInlineDecl(t *testing.T) {
	// export let x = ... should work (inline export of declaration).
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `export let double = (x int) int { return x * 2 }
export const MAGIC = 7`,
		"main.monk": `use { double, MAGIC } from "./lib"
show(to_string(double(MAGIC)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "14" {
		t.Errorf("expected '14', got %q", stdout)
	}
}

func TestRunModuleReExport(t *testing.T) {
	// A imports from B, then re-exports it so C can import from A.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"origin.monk": `let secret = 42
export secret`,
		"proxy.monk": `use secret from "./origin"
export secret`,
		"main.monk": `use secret from "./proxy"
show(to_string(secret))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "42" {
		t.Errorf("expected '42', got %q", stdout)
	}
}

func TestRunModuleTransitiveImport(t *testing.T) {
	// A -> B -> C. A uses C's export that B re-exports.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"base.monk": `let base_val = 100
export base_val`,
		"middle.monk": `use base_val from "./base"
let combined = base_val + 50
export combined`,
		"main.monk": `use combined from "./middle"
show(to_string(combined))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "150" {
		t.Errorf("expected '150', got %q", stdout)
	}
}

func TestRunModuleMultipleUseSameModule(t *testing.T) {
	// Two use statements from the same module — should not double-init.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `show("lib init")
let a = 1
export a
let b = 2
export b`,
		"main.monk": `use a from "./lib"
use b from "./lib"
show(to_string(a + b))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	// "lib init" should appear exactly once despite two use statements.
	if stdout != "lib init\n3" {
		t.Errorf("expected 'lib init\\n3', got %q", stdout)
	}
}

func TestRunModuleSideEffectsOnly(t *testing.T) {
	// Module imported for side effects only — no exports used.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"setup.monk": `show("setup done")
let x = 1
export x`,
		"main.monk": `use x from "./setup"
show("after setup")`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "setup done\nafter setup" {
		t.Errorf("expected 'setup done\\nafter setup', got %q", stdout)
	}
}

func TestRunModuleExportType(t *testing.T) {
	// Export a type definition and use it as an annotation in another module.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"types.monk": `type Point = { x: int, y: int }
export Point`,
		"main.monk": `use Point from "./types"
let p Point = {x: 3, y: 4}
show(to_string(p.x + p.y))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "7" {
		t.Errorf("expected '7', got %q", stdout)
	}
}

func TestCheckModuleNonRelativePath(t *testing.T) {
	// Non-relative module path should error.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"main.monk": `use x from "lib"`,
	})
	_, stderr, code := runMonkCmd(t, bin, "check", filepath.Join(dir, "main.monk"))
	if code == 0 {
		t.Fatal("expected non-zero exit for non-relative path")
	}
	if !strings.Contains(stderr, "must be relative") {
		t.Errorf("expected 'must be relative' in error, got %q", stderr)
	}
}

func TestRunModuleParentDirectory(t *testing.T) {
	// Import from parent directory with ../
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"shared.monk": `let val = 99
export val`,
		"sub/main.monk": `use val from "../shared"
show(to_string(val))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "sub", "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "99" {
		t.Errorf("expected '99', got %q", stdout)
	}
}

func TestRunModuleExportBareAndInline(t *testing.T) {
	// Mix of inline export (export let) and bare export (export name).
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `export let inline_fn = (x int) int { return x + 1 }
let bare_fn = (x int) int { return x * 2 }
export bare_fn`,
		"main.monk": `use { inline_fn, bare_fn } from "./lib"
show(to_string(inline_fn(5)))
show(to_string(bare_fn(5)))`,
	})
	stdout, _, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if stdout != "6\n10" {
		t.Errorf("expected '6\\n10', got %q", stdout)
	}
}

func TestCheckModuleSingleFile(t *testing.T) {
	// monk check on a single file with no imports should still work.
	bin := buildMonk(t)
	src := writeMonk(t, "solo.monk", `let x = 42
show(to_string(x))`)
	_, stderr, code := runMonkCmd(t, bin, "check", src)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
}

func TestCheckModuleValid(t *testing.T) {
	// monk check on a multi-file program should pass.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `let add = (a int, b int) int { return a + b }
export add`,
		"main.monk": `use add from "./lib"
show(to_string(add(1, 2)))`,
	})
	_, stderr, code := runMonkCmd(t, bin, "check", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "ok") {
		t.Errorf("expected 'ok' in stderr, got %q", stderr)
	}
}

func TestCheckModuleImportShadowError(t *testing.T) {
	// `let x = ...` after `use x from "./lib"` must be a compile error.
	// Without this check, codegen silently maps `x` to the imported C name,
	// making the local declaration dead with no warning.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `let value = 42
export value`,
		"main.monk": `use value from "./lib"
let value = 99
show(to_string(value))`,
	})
	_, stderr, code := runMonkCmd(t, bin, "check", filepath.Join(dir, "main.monk"))
	if code == 0 {
		t.Fatal("expected compile error for import shadowing, got exit 0")
	}
	if !strings.Contains(stderr, "already declared via import") {
		t.Errorf("expected shadowing error message, got: %s", stderr)
	}
}

func TestRunModuleCrossModuleClosure(t *testing.T) {
	// Exported function that captures a module-level variable.
	// The defining module's funcHasCapture must propagate to the importer so
	// emitCall routes through monk_call instead of a direct C call.
	// Without the fix: "too few arguments to function" C compile error.
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"counter.monk": `let n = 10
let adder = (x int) int { return x + n }
export adder`,
		"main.monk": `use adder from "./counter"
show(to_string(adder(5)))`,
	})
	stdout, stderr, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if stdout != "15" {
		t.Errorf("expected '15', got %q", stdout)
	}
}

// TestRunModuleLocalVarInsideFunction verifies that `let` declarations inside
// exported functions in non-entry modules are emitted as stack-local variables,
// NOT as static globals. Before the moduleInit save/restore fix, `result` would
// be declared `static int64_t mk_m0_result` at file scope — shared across calls.
// Pass: double(5) = 10, double(3) = 6  (each call gets its own result)
// Fail (before fix): static result corrupted by second call
func TestRunModuleLocalVarInsideFunction(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `
let double = (x int) int {
    let result = x * 2
    return result
}
export double
`,
		"main.monk": `
use double from "./lib"
show(to_string(double(5)))
show(to_string(double(3)))
`,
	})
	stdout, stderr, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	want := "10\n6"
	if stdout != want {
		t.Errorf("got %q, want %q", stdout, want)
	}
}

// TestRunModuleRecursiveLocalVar verifies that a recursive exported function
// with a local variable works correctly across multiple activations. Before
// the fix, the static global would be overwritten by each recursive call —
// factorial(5) would return 1 instead of 120.
// Pass: factorial(5) = 120
// Fail (before fix): shared static result corrupts the call stack
func TestRunModuleRecursiveLocalVar(t *testing.T) {
	bin := buildMonk(t)
	dir := writeMonkFiles(t, map[string]string{
		"lib.monk": `
let factorial = (n int) int {
    if n <= 1 { return 1 }
    let sub = factorial(n - 1)
    return n * sub
}
export factorial
`,
		"main.monk": `
use factorial from "./lib"
show(to_string(factorial(5)))
`,
	})
	stdout, stderr, code := runMonkCmd(t, bin, "run", filepath.Join(dir, "main.monk"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	want := "120"
	if stdout != want {
		t.Errorf("got %q, want %q", stdout, want)
	}
}
