package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildMonk builds the monk binary to a temp directory and returns its path.
func buildMonk(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "monk")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(projectRoot(t), "src", "cmd", "monk")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build monk: %s\n%s", err, out)
	}
	return bin
}

func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// runMonkCmd runs the monk binary with args, returns stdout and exit code.
func runMonkCmd(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = projectRoot(t) // run from project root so runtime/ is found
	cmd.Env = append(os.Environ(), "MONK_RUNTIME_DIR="+filepath.Join(projectRoot(t), "runtime"))
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
	if lines[0] != "1" { t.Errorf("line 1: expected '1', got %q", lines[0]) }
	if lines[2] != "Fizz" { t.Errorf("line 3: expected 'Fizz', got %q", lines[2]) }
	if lines[4] != "Buzz" { t.Errorf("line 5: expected 'Buzz', got %q", lines[4]) }
	if lines[14] != "FizzBuzz" { t.Errorf("line 15: expected 'FizzBuzz', got %q", lines[14]) }
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
