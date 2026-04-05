package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
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

// ─── Scalar unboxing (Phase 6) ────────────────────────────────────────────
// The unboxing path emits raw int64_t/double/bool for scalar variables and
// raw C arithmetic for operations between scalars. These tests verify both
// the GENERATED SHAPE (string checks) and the RUNTIME RESULT (compile+run).

// runMonkTyped is like runMonk but runs the type checker first and feeds
// the Info to GenerateWithTypes. This is what `monk build` does in production.
func runMonkTyped(t *testing.T, source string) (string, string) {
	t.Helper()
	prog, err := syntax.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	info, err := types.Check(prog)
	if err != nil {
		t.Fatalf("type error: %v", err)
	}
	cSource := GenerateWithTypes(prog, "test.monk", info)

	dir := t.TempDir()
	cFile := filepath.Join(dir, "test.c")
	binFile := filepath.Join(dir, "test")
	runtimeDir, _ := filepath.Abs("../runtime")
	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("cc", "-std=c11", "-Wall", "-I"+runtimeDir,
		cFile, filepath.Join(runtimeDir, "runtime.c"), "-lm", "-o", binFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed:\n%s\n\nGenerated C:\n%s", string(out), cSource)
	}
	cmd = exec.Command(binFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runtime error:\n%s\n\nGenerated C:\n%s", string(out), cSource)
	}
	return strings.TrimRight(string(out), "\n"), cSource
}

func TestUnboxIntVarDecl(t *testing.T) {
	out, src := runMonkTyped(t, `let x int = 42
show(to_string(x))`)
	if out != "42" {
		t.Errorf("expected '42', got %q", out)
	}
	if !strings.Contains(src, "int64_t mk_x = 42") {
		t.Errorf("expected unboxed int declaration, generated:\n%s", src)
	}
}

func TestUnboxFloatVarDecl(t *testing.T) {
	out, src := runMonkTyped(t, `let pi float = 3.14
show(to_string(pi))`)
	if out != "3.14" {
		t.Errorf("expected '3.14', got %q", out)
	}
	if !strings.Contains(src, "double mk_pi = 3.14") {
		t.Errorf("expected unboxed double declaration, generated:\n%s", src)
	}
}

func TestUnboxBoolVarDecl(t *testing.T) {
	out, src := runMonkTyped(t, `let flag boolean = true
show(to_string(flag))`)
	if out != "true" {
		t.Errorf("expected 'true', got %q", out)
	}
	if !strings.Contains(src, "bool mk_flag = true") {
		t.Errorf("expected unboxed bool declaration, generated:\n%s", src)
	}
}

func TestUnboxIntArithmeticStaysRaw(t *testing.T) {
	out, src := runMonkTyped(t, `let a = 10
let b = 3
let c = a * b + 5
show(to_string(c))`)
	if out != "35" {
		t.Errorf("expected '35', got %q", out)
	}
	// Should emit `(a * b) + 5` style raw arithmetic, NOT monk_add / monk_mul.
	if strings.Contains(src, "monk_mul(") || strings.Contains(src, "monk_add(") {
		t.Errorf("expected raw C arithmetic, got runtime calls:\n%s", src)
	}
}

func TestUnboxMixedIntFloatWidens(t *testing.T) {
	out, src := runMonkTyped(t, `let i = 5
let f float = 2.5
let result = i * f
show(to_string(result))`)
	if out != "12.5" {
		t.Errorf("expected '12.5', got %q", out)
	}
	if !strings.Contains(src, "(double)") {
		t.Errorf("expected explicit int->float widening, generated:\n%s", src)
	}
}

func TestUnboxConditionIsRaw(t *testing.T) {
	out, src := runMonkTyped(t, `let x = 10
if x > 5 { show("big") } else { show("small") }`)
	if out != "big" {
		t.Errorf("expected 'big', got %q", out)
	}
	// `if (mk_x > 5)` — no monk_less/monk_is_truthy wrapper.
	if strings.Contains(src, "monk_is_truthy") {
		t.Errorf("expected raw C condition, got monk_is_truthy wrap:\n%s", src)
	}
}

func TestUnboxWhileLoopRaw(t *testing.T) {
	out, src := runMonkTyped(t, `let n = 0
let sum = 0
while n < 10 {
  sum += n
  n += 1
}
show(to_string(sum))`)
	if out != "45" {
		t.Errorf("expected '45', got %q", out)
	}
	if strings.Contains(src, "monk_is_truthy") || strings.Contains(src, "monk_add") {
		t.Errorf("expected raw C loop, generated:\n%s", src)
	}
}

func TestUnboxUserFuncFullScalar(t *testing.T) {
	out, src := runMonkTyped(t, `let add = (a int, b int) int { return a + b }
show(to_string(add(7, 5)))`)
	if out != "12" {
		t.Errorf("expected '12', got %q", out)
	}
	if !strings.Contains(src, "static int64_t") {
		t.Errorf("expected unboxed function signature, generated:\n%s", src)
	}
}

func TestUnboxRecursiveInt(t *testing.T) {
	// Fibonacci — classic unboxable recursion.
	out, src := runMonkTyped(t, `let fib = (n int) int {
  if n < 2 { return n }
  return fib(n - 1) + fib(n - 2)
}
show(to_string(fib(10)))`)
	if out != "55" {
		t.Errorf("expected '55', got %q", out)
	}
	if !strings.Contains(src, "static int64_t") {
		t.Errorf("expected unboxed fib signature, generated:\n%s", src)
	}
	// The recursive call should be raw, not boxing args and unboxing return.
	if strings.Count(src, "monk_int(") > 2 {
		// Allow the final show() boxing, but not more.
		t.Errorf("expected raw recursive calls, too many monk_int(...) in:\n%s", src)
	}
}

func TestUnboxToFloatInlined(t *testing.T) {
	// to_float(int_var) should become a direct (double) cast with no runtime call.
	out, src := runMonkTyped(t, `let n = 10
let f = to_float(n)
show(to_string(f))`)
	if out != "10" {
		t.Errorf("expected '10', got %q", out)
	}
	if strings.Contains(src, "monk_to_float(") {
		t.Errorf("expected to_float to inline as (double), got runtime call:\n%s", src)
	}
}

// Regression (CodeRabbit): the unboxed SlashEqual/PercentEqual used to double-
// evaluate the RHS, so `a /= f()` called f() twice. With a stateful f(), that
// changes semantics.
func TestUnboxDivEvaluatesRHSOnce(t *testing.T) {
	_, src := runMonkTyped(t, `let get_divisor = (n int) int { return n * 2 }
let a = 100
let b = a / get_divisor(5)
show(to_string(b))`)
	// The call must appear exactly once in the generated expression.
	// Two occurrences expected: definition `static int64_t _monk_func_1(...)`
	// and the single call `_monk_func_1(5)` — total 2. Three would mean the
	// expression was evaluated twice.
	callCount := strings.Count(src, "_monk_func_1(")
	if callCount != 2 {
		t.Errorf("expected 2 occurrences (1 def + 1 call), got %d\n%s", callCount, src)
	}
}

func TestUnboxFloatModuloUsesFmod(t *testing.T) {
	// Float %= used to silently do nothing. Now it uses fmod().
	out, src := runMonkTyped(t, `let f float = 7.5
f %= 2.0
show(to_string(f))`)
	if out != "1.5" {
		t.Errorf("expected '1.5', got %q", out)
	}
	if !strings.Contains(src, "fmod(") {
		t.Errorf("expected fmod() for float %%=, generated:\n%s", src)
	}
}

func TestUnboxIntDivByZeroPanics(t *testing.T) {
	// Unboxed int div-by-zero should panic at runtime with the same message
	// the boxed path produces.
	prog, _ := syntax.Parse(`let a = 10
let b = 0
show(to_string(a / b))`)
	info, _ := types.Check(prog)
	cSource := GenerateWithTypes(prog, "t.monk", info)
	dir := t.TempDir()
	cFile := filepath.Join(dir, "t.c")
	bin := filepath.Join(dir, "t")
	runtimeDir, _ := filepath.Abs("../runtime")
	_ = os.WriteFile(cFile, []byte(cSource), 0644)
	cmd := exec.Command("cc", "-std=c11", "-I"+runtimeDir, cFile,
		filepath.Join(runtimeDir, "runtime.c"), "-lm", "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %s", out)
	}
	out, _ := exec.Command(bin).CombinedOutput()
	if !strings.Contains(string(out), "division by zero") {
		t.Errorf("expected 'division by zero' in output, got: %s", out)
	}
}

func TestUnboxBoxedMixedPath(t *testing.T) {
	// A value flowing between unboxed and boxed worlds — should box/unbox
	// cleanly with no crashes.
	out, _ := runMonkTyped(t, `let n = 42
let s = "value: " + to_string(n)
show(s)`)
	if out != "value: 42" {
		t.Errorf("expected 'value: 42', got %q", out)
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
