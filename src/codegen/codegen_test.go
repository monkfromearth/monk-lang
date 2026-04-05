package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkfromearth/monk-lang/syntax"
)

// runMonk compiles a Monk source string to C, compiles the C with cc,
// runs the binary, and returns stdout. This is a full integration test.
func runMonk(t *testing.T, source string) string {
	t.Helper()

	// Parse
	prog, err := syntax.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Generate C
	cSource := Generate(prog, "test.monk")

	// Write to temp files
	dir := t.TempDir()
	cFile := filepath.Join(dir, "test.c")
	binFile := filepath.Join(dir, "test")

	// Find runtime path (relative to this test file)
	runtimeDir, err := filepath.Abs("../runtime")
	if err != nil {
		t.Fatalf("failed to resolve runtime path: %v", err)
	}
	runtimeC := filepath.Join(runtimeDir, "runtime.c")

	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		t.Fatalf("write .c: %v", err)
	}

	// Compile: cc -std=c11 -I<runtime_dir> test.c runtime.c -lm -o test
	cmd := exec.Command("cc", "-std=c11", "-Wall",
		"-I"+runtimeDir,
		cFile, runtimeC,
		"-lm", "-o", binFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed:\n%s\n\nGenerated C:\n%s", string(out), cSource)
	}

	// Run
	cmd = exec.Command(binFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runtime error:\n%s\n\nGenerated C:\n%s", string(out), cSource)
	}

	return strings.TrimRight(string(out), "\n")
}

func expectOutput(t *testing.T, source string, expected string) {
	t.Helper()
	got := runMonk(t, source)
	if got != expected {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

// === LITERALS ===

func TestCodegenIntLiteral(t *testing.T) {
	expectOutput(t, `show(to_string(42))`, "42")
}

func TestCodegenFloatLiteral(t *testing.T) {
	expectOutput(t, `show(to_string(3.14))`, "3.14")
}

func TestCodegenStringLiteral(t *testing.T) {
	expectOutput(t, `show("hello")`, "hello")
}

func TestCodegenBoolLiteral(t *testing.T) {
	expectOutput(t, `show(to_string(true))`, "true")
	expectOutput(t, `show(to_string(false))`, "false")
}

func TestCodegenNoneLiteral(t *testing.T) {
	expectOutput(t, `show(to_string(none))`, "none")
}

// === ARITHMETIC ===

func TestCodegenAdd(t *testing.T) {
	expectOutput(t, `show(to_string(1 + 2))`, "3")
}

func TestCodegenIntDiv(t *testing.T) {
	expectOutput(t, `show(to_string(10 / 3))`, "3")
}

func TestCodegenFloatPromotion(t *testing.T) {
	expectOutput(t, `show(to_string(1 + 2.5))`, "3.5")
}

func TestCodegenPrecedence(t *testing.T) {
	expectOutput(t, `show(to_string(1 + 2 * 3))`, "7")
}

func TestCodegenParens(t *testing.T) {
	expectOutput(t, `show(to_string((1 + 2) * 3))`, "9")
}

func TestCodegenNegation(t *testing.T) {
	expectOutput(t, `show(to_string(-42))`, "-42")
}

// === STRING ===

func TestCodegenStringConcat(t *testing.T) {
	expectOutput(t, `show("hello" + " " + "world")`, "hello world")
}

// === COMPARISON ===

func TestCodegenEqual(t *testing.T) {
	expectOutput(t, `show(to_string(5 == 5))`, "true")
	expectOutput(t, `show(to_string(5 == 3))`, "false")
}

func TestCodegenLess(t *testing.T) {
	expectOutput(t, `show(to_string(3 < 5))`, "true")
}

// === LOGICAL ===

func TestCodegenAnd(t *testing.T) {
	expectOutput(t, `show(to_string(true and true))`, "true")
	expectOutput(t, `show(to_string(true and false))`, "false")
}

func TestCodegenOr(t *testing.T) {
	expectOutput(t, `show(to_string(false or true))`, "true")
}

func TestCodegenNot(t *testing.T) {
	expectOutput(t, `show(to_string(not true))`, "false")
	expectOutput(t, `show(to_string(not 0))`, "true")
}

// === VARIABLES ===

func TestCodegenLetDecl(t *testing.T) {
	expectOutput(t, "let x = 42\nshow(to_string(x))", "42")
}

func TestCodegenLetReassign(t *testing.T) {
	expectOutput(t, "let x = 1\nx = 2\nshow(to_string(x))", "2")
}

func TestCodegenCompoundAssign(t *testing.T) {
	expectOutput(t, "let x = 10\nx += 5\nshow(to_string(x))", "15")
}

// === IF/ELSE ===

func TestCodegenIf(t *testing.T) {
	expectOutput(t, `let x = 0
if true { x = 1 }
show(to_string(x))`, "1")
}

func TestCodegenIfElse(t *testing.T) {
	expectOutput(t, `let x = 0
if false { x = 1 } else { x = 2 }
show(to_string(x))`, "2")
}

// === WHILE ===

func TestCodegenWhile(t *testing.T) {
	expectOutput(t, `let x = 0
while x < 5 { x += 1 }
show(to_string(x))`, "5")
}

func TestCodegenWhileBreak(t *testing.T) {
	expectOutput(t, `let x = 0
while true {
    x += 1
    if x == 3 { break }
}
show(to_string(x))`, "3")
}

// === FOR ===

func TestCodegenForArray(t *testing.T) {
	expectOutput(t, `let sum = 0
for x in [1, 2, 3] { sum += x }
show(to_string(sum))`, "6")
}

func TestCodegenForString(t *testing.T) {
	expectOutput(t, `let count = 0
for c in "hello" { count += 1 }
show(to_string(count))`, "5")
}

// === ARRAYS ===

func TestCodegenArray(t *testing.T) {
	expectOutput(t, `let arr = [10, 20, 30]
show(to_string(arr))`, "[10, 20, 30]")
}

func TestCodegenArrayIndex(t *testing.T) {
	expectOutput(t, `let arr = [10, 20, 30]
show(to_string(arr[1]))`, "20")
}

func TestCodegenArrayOutOfBounds(t *testing.T) {
	expectOutput(t, `let arr = [1, 2, 3]
show(to_string(arr[10]))`, "none")
}

// === RECORDS ===

func TestCodegenRecord(t *testing.T) {
	expectOutput(t, `let r = {name: "Alice", age: 30}
show(r.name)`, "Alice")
}

// === FUNCTIONS ===

func TestCodegenFunction(t *testing.T) {
	expectOutput(t, `let add = (a int, b int) int {
    return a + b
}
show(to_string(add(3, 4)))`, "7")
}

func TestCodegenRecursion(t *testing.T) {
	expectOutput(t, `let factorial = (n int) int {
    if n <= 1 { return 1 }
    return n * factorial(n - 1)
}
show(to_string(factorial(5)))`, "120")
}

// === GUARD / THROW ===

func TestCodegenGuardSuccess(t *testing.T) {
	expectOutput(t, `let divide = (a int, b int) int {
    if b == 0 { throw "division by zero" }
    return a / b
}
guard result = divide(10, 2) against error {
    result = -1
}
show(to_string(result))`, "5")
}

func TestCodegenGuardError(t *testing.T) {
	expectOutput(t, `let divide = (a int, b int) int {
    if b == 0 { throw "division by zero" }
    return a / b
}
guard result = divide(10, 0) against error {
    result = -1
}
show(to_string(result))`, "-1")
}

// === BUILTINS ===

func TestCodegenRange(t *testing.T) {
	expectOutput(t, `let sum = 0
for i in range(5) { sum += i }
show(to_string(sum))`, "10")
}

func TestCodegenLength(t *testing.T) {
	expectOutput(t, `show(to_string(length("hello")))`, "5")
	expectOutput(t, `show(to_string(length([1, 2, 3])))`, "3")
}

func TestCodegenTypeof(t *testing.T) {
	expectOutput(t, `show(typeof(42))`, "int")
	expectOutput(t, `show(typeof("hi"))`, "string")
}

func TestCodegenAppend(t *testing.T) {
	expectOutput(t, `let arr = append([1, 2], 3)
show(to_string(arr))`, "[1, 2, 3]")
}

func TestCodegenToString(t *testing.T) {
	expectOutput(t, `show("value: " + to_string(42))`, "value: 42")
}

// to_int / to_float now accept int | float | string (not just string).
// Previously, converting an int variable to a float required `x * 1.0`.
func TestCodegenToFloatFromInt(t *testing.T) {
	expectOutput(t, `let i = 5
show(to_string(to_float(i)))`, "5")
}

func TestCodegenToIntFromFloatTruncates(t *testing.T) {
	expectOutput(t, `show(to_string(to_int(3.9)))
show(to_string(to_int(0.0 - 3.9)))`, "3\n-3")
}

func TestCodegenToIntFromIntIdentity(t *testing.T) {
	expectOutput(t, `show(to_string(to_int(42)))`, "42")
}

func TestCodegenToFloatFromString(t *testing.T) {
	// Existing behavior — must still work after the signature widening.
	expectOutput(t, `show(to_string(to_float("2.5")))`, "2.5")
}

// cString must produce valid C string literals for any input. Used for
// user-provided strings AND filenames in #line directives — both can contain
// backslashes, quotes, or newlines that would otherwise produce broken C.
func TestCString(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`plain.monk`, `"plain.monk"`},
		{`C:\path\file.monk`, `"C:\\path\\file.monk"`},
		{`with"quote.monk`, `"with\"quote.monk"`},
		{"with\nnewline.monk", `"with\nnewline.monk"`},
		{"tab\there.monk", `"tab\there.monk"`},
		{"carriage\rreturn", `"carriage\rreturn"`},
		{"", `""`},
	}
	for _, c := range cases {
		if got := cString(c.in); got != c.want {
			t.Errorf("cString(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A filename containing backslashes/quotes must produce C that still compiles.
// Before #line was run through cString(), the directive emitted raw
// backslashes, producing invalid C.
func TestCodegenFilenameWithBackslashes(t *testing.T) {
	prog, err := syntax.Parse(`show("ok")`)
	if err != nil {
		t.Fatal(err)
	}
	cSource := Generate(prog, `C:\users\test\hello.monk`)
	if !strings.Contains(cSource, `#line 1 "C:\\users\\test\\hello.monk"`) {
		t.Errorf("#line directive not escaped correctly:\n%s", cSource)
	}

	// And the generated C must actually compile.
	dir := t.TempDir()
	cFile := filepath.Join(dir, "test.c")
	runtimeDir, _ := filepath.Abs("../runtime")
	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("cc", "-std=c11", "-Wall", "-I"+runtimeDir,
		cFile, filepath.Join(runtimeDir, "runtime.c"), "-lm",
		"-o", filepath.Join(dir, "test"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed on escaped path:\n%s\n\n%s", out, cSource)
	}
}
