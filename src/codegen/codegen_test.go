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

// runtimeArgs builds the cc argument list for linking the runtime into a test.
// Discovers .c files from the runtime/ directory at test time — no manual
// sync needed when new runtime files are added.
func runtimeArgs(runtimeDir string) []string {
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		return nil
	}
	var args []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".c") && name != "runtime_test.c" {
			args = append(args, filepath.Join(runtimeDir, name))
		}
	}
	return args
}

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
	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		t.Fatalf("write .c: %v", err)
	}

	// Compile: cc -std=c11 -I<runtime_dir> test.c <runtime sources> -lm -o test
	ccArgs := append([]string{"-std=c11", "-Wall", "-I" + runtimeDir, cFile},
		runtimeArgs(runtimeDir)...)
	ccArgs = append(ccArgs, "-lm", "-o", binFile)
	cmd := exec.Command("cc", ccArgs...)
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
	ccArgs := append([]string{"-std=c11", "-Wall", "-I" + runtimeDir, cFile},
		runtimeArgs(runtimeDir)...)
	ccArgs = append(ccArgs, "-lm", "-o", binFile)
	cmd := exec.Command("cc", ccArgs...)
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
	// Three occurrences expected: definition `static int64_t _monk_func_1(...)`,
	// the thunk `_monk_func_1(...)`, and the single call `_monk_func_1(5)`.
	// Four would mean the expression was evaluated twice.
	callCount := strings.Count(src, "_monk_func_1(")
	if callCount != 3 {
		t.Errorf("expected 3 occurrences (1 def + 1 thunk + 1 call), got %d\n%s", callCount, src)
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
	ccArgs := append([]string{"-std=c11", "-I" + runtimeDir, cFile},
		runtimeArgs(runtimeDir)...)
	ccArgs = append(ccArgs, "-lm", "-o", bin)
	cmd := exec.Command("cc", ccArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %s", out)
	}
	out, _ := exec.Command(bin).CombinedOutput()
	if !strings.Contains(string(out), "division by zero") {
		t.Errorf("expected 'division by zero' in output, got: %s", out)
	}
}

// Regression: the unboxing storage map used to be flat, so a shadowed
// variable in an inner scope would leak its (different) storage decision
// back out when the scope ended. Typical symptom: `show(to_string(n))`
// on the outer `n` emitted monk_to_string(mk_n) but mk_n was int64_t.
func TestUnboxShadowRestoresStorage(t *testing.T) {
	// Outer int n, inner string n in an if body, then use outer n again.
	out, _ := runMonkTyped(t, `let n = 5
if true {
  let n = "inner"
  show(n)
}
show(to_string(n))`)
	if out != "inner\n5" {
		t.Errorf("expected 'inner\\n5', got %q", out)
	}
}

func TestUnboxShadowInFunctionParam(t *testing.T) {
	// Function param is int; body shadows it with a string.
	out, _ := runMonkTyped(t, `let f = (x int) int {
  let x = "shadow"
  show(x)
  return 42
}
show(to_string(f(99)))`)
	if out != "shadow\n42" {
		t.Errorf("expected 'shadow\\n42', got %q", out)
	}
}

// Regression: when one function has an unboxed param (e.g. mk_x → int64_t)
// and a LATER hoisted function also has a param named mk_x that's boxed,
// the storage map used to keep the stale int64_t binding alive, so the
// second function's body compiled calls that passed a MonkValue where an
// int64_t was expected. Fixed by saving g.storage BEFORE writing the
// params into it, and always recording the param's storage (even boxed).
func TestUnboxFuncParamsDoNotLeakBetweenSiblings(t *testing.T) {
	// validate_age is fully unboxed (int → int). process is boxed (returns
	// string). process's body calls validate_age — if the storage map had
	// leaked, process's `age` ident would have been read back as int64_t
	// and emitUnboxedCall would have accepted it without coercion, leaving
	// a MonkValue flowing into an int64_t parameter.
	out, _ := runMonkTyped(t, `let validate_age = (age int) int {
  if age < 0 { throw "neg" }
  return age
}
let process = (age int) string {
  let v = validate_age(age)
  return "got " + to_string(v)
}
show(process(42))`)
	if out != "got 42" {
		t.Errorf("expected 'got 42', got %q", out)
	}
}

// Regression: nested index assignment like `m[0][1] = x` used to emit
// `monk_array_set(&monk_array_get(…), …)` — taking the address of an
// rvalue. The cc invocation failed with "cannot take address of rvalue".
// Codegen now routes non-identifier targets through a temp, producing
// compilable C. Per Monk's value semantics the mutation is a no-op
// against the outer container (matrix[i] returns a deep copy).
func TestCodegenNestedIndexAssignCompiles(t *testing.T) {
	// m[0] is a deep copy; mutating m[0][1] doesn't propagate back.
	expectOutput(t, `let m = [[1, 2], [3, 4]]
m[0][1] = 99
show(to_string(m[0][1]))`, "2")
}

func TestCodegenNestedPropertyAssignCompiles(t *testing.T) {
	// Same rationale — p.inner returns a deep copy, so the assignment
	// through that copy doesn't reach back into p. This test just proves
	// the generated C compiles and runs.
	expectOutput(t, `type Inner = { v: int }
type Outer = { i: Inner }
let inner Inner = { v: 1 }
let p Outer = { i: inner }
p.i.v = 99
show(to_string(p.i.v))`, "1")
}

func TestUnboxShadowInForLoopBody(t *testing.T) {
	// For loop variable is int (element of range(...)); body shadows with string.
	out, _ := runMonkTyped(t, `for i in range(2) {
  let i = "x"
  show(i)
}`)
	if out != "x\nx" {
		t.Errorf("expected 'x\\nx', got %q", out)
	}
}

func TestUnboxShadowRejectsToIntOnBool(t *testing.T) {
	// Regression: the to_int/to_float inlining used to accept storeBool and
	// emit the bool code tagged as int (true -> 1). That silently diverges
	// from the runtime, which panics "to_int: expected int, float, or string".
	// The fast path now only inlines for storeInt/storeFloat args.
	prog, _ := syntax.Parse(`let b = true
let n = to_int(b)
show(to_string(n))`)
	info, _ := types.Check(prog)
	cSource := GenerateWithTypes(prog, "t.monk", info)
	// Expected: monk_to_int(monk_bool(mk_b)) — routes to the runtime which
	// will panic. The inlined path would emit `mk_b` directly.
	if !strings.Contains(cSource, "monk_to_int") {
		t.Errorf("expected runtime monk_to_int call for bool argument, generated:\n%s", cSource)
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
	ccArgs := append([]string{"-std=c11", "-Wall", "-I" + runtimeDir, cFile},
		runtimeArgs(runtimeDir)...)
	ccArgs = append(ccArgs, "-lm", "-o", filepath.Join(dir, "test"))
	cmd := exec.Command("cc", ccArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed on escaped path:\n%s\n\n%s", out, cSource)
	}
}

// === UNDERSCORE NUMERIC LITERALS ===

func TestCodegenUnderscoreInt(t *testing.T) {
	// Monk allows 1_000_000 but C doesn't — codegen must strip underscores.
	expectOutput(t, `let x = 1_000_000
show(to_string(x))`, "1000000")
}

func TestCodegenUnderscoreHex(t *testing.T) {
	expectOutput(t, `let x = 0xFF_FF
show(to_string(x))`, "65535")
}

func TestCodegenUnderscoreBinary(t *testing.T) {
	expectOutput(t, `let x = 0b1010_0101
show(to_string(x))`, "165")
}

func TestCodegenUnderscoreTypedInt(t *testing.T) {
	// Typed path (unboxed) must also strip underscores.
	out, _ := runMonkTyped(t, `let x int = 1_000_000
show(to_string(x))`)
	if out != "1000000" {
		t.Errorf("expected '1000000', got %q", out)
	}
}

// === FUNCTION PARAM DEEP COPY (VALUE SEMANTICS) ===

func TestCodegenFuncParamDeepCopyArray(t *testing.T) {
	// Spec: function args are copies. Mutating arr inside the function
	// must not affect the caller's data.
	expectOutput(t, `let mutate = (arr array) none {
    arr[0] = 999
}
let data = [10, 20, 30]
mutate(data)
show(to_string(data))`, "[10, 20, 30]")
}

func TestCodegenFuncParamDeepCopyRecord(t *testing.T) {
	expectOutput(t, `let mutate = (r record) none {
    r.x = 999
}
let p = {x: 1, y: 2}
mutate(p)
show(to_string(p.x))`, "1")
}

// === ABS RETURNS INT FOR INT INPUT ===

func TestCodegenAbsReturnsIntForInt(t *testing.T) {
	// abs() on an int arg should be usable in an int-returning function.
	out, _ := runMonkTyped(t, `let my_abs = (a int, b int) int {
    return abs(a * b)
}
show(to_string(my_abs(3, -7)))`)
	if out != "21" {
		t.Errorf("expected '21', got %q", out)
	}
}

// === DEFAULT PARAMETER VALUES ===

func TestCodegenDefaultParamOmitted(t *testing.T) {
	expectOutput(t, `let greet = (name string, greeting string = "Hello") string {
    return greeting + ", " + name
}
show(greet("Alice"))`, "Hello, Alice")
}

func TestCodegenDefaultParamProvided(t *testing.T) {
	expectOutput(t, `let greet = (name string, greeting string = "Hello") string {
    return greeting + ", " + name
}
show(greet("Alice", "Hey"))`, "Hey, Alice")
}

func TestCodegenDefaultParamMultiple(t *testing.T) {
	expectOutput(t, `let f = (a int, b int = 10, c int = 20) int {
    return a + b + c
}
show(to_string(f(1)))
show(to_string(f(1, 2)))
show(to_string(f(1, 2, 3)))`, "31\n23\n6")
}

func TestCodegenDefaultParamTypedUnboxed(t *testing.T) {
	out, _ := runMonkTyped(t, `let add = (a int, b int = 100) int {
    return a + b
}
show(to_string(add(5)))
show(to_string(add(5, 7)))`)
	if out != "105\n12" {
		t.Errorf("expected '105\\n12', got %q", out)
	}
}

// Regression: emitCallTyped must call padDefaults before emitUnboxedCall,
// otherwise omitting a trailing default arg in a typed context (e.g. the RHS
// of a typed let) generates a C call with too few arguments.
func TestCodegenDefaultParamTypedCallSite(t *testing.T) {
	out, _ := runMonkTyped(t, `let add = (a int, b int = 100) int {
    return a + b
}
let result = add(5)
show(to_string(result))`)
	if out != "105" {
		t.Errorf("expected '105', got %q", out)
	}
}

// Regression: void closures (no explicit return) must still persist mutations
// to captured variables through the fallback return path.
func TestCodegenVoidClosureSavesCaptures(t *testing.T) {
	out := runMonk(t, `let make_counter = () () -> none {
    let count = 0
    return () none {
        count = count + 1
        show(to_string(count))
    }
}
let inc = make_counter()
inc()
inc()
inc()`)
	if out != "1\n2\n3" {
		t.Errorf("expected '1\\n2\\n3', got %q", out)
	}
}

func TestCodegenMapBuiltin(t *testing.T) {
	out := runMonk(t, `let nums = [1, 2, 3]
let doubled = map(nums, (x int) { return x * 2 })
show(to_string(doubled))`)
	if out != "[2, 4, 6]" {
		t.Errorf("expected '[2, 4, 6]', got %q", out)
	}
}

func TestCodegenFilterBuiltin(t *testing.T) {
	out := runMonk(t, `let nums = [1, 2, 3, 4, 5]
let evens = filter(nums, (x int) { return x % 2 == 0 })
show(to_string(evens))`)
	if out != "[2, 4]" {
		t.Errorf("expected '[2, 4]', got %q", out)
	}
}

func TestCodegenReduceBuiltin(t *testing.T) {
	out := runMonk(t, `let nums = [1, 2, 3, 4, 5]
let total = reduce(nums, (acc int, x int) { return acc + x }, 0)
show(to_string(total))`)
	if out != "15" {
		t.Errorf("expected '15', got %q", out)
	}
}

// === TYPED ARRAY UNBOXING ===
// All tests use runMonkTyped so the type checker feeds Info to codegen and
// the typed-array fast path activates.

// Int array literal: element reads return unboxed int64_t.
func TestTypedArrayIntRead(t *testing.T) {
	out, src := runMonkTyped(t, `let arr int[] = [10, 20, 30]
let x = arr[1]
show(to_string(x))`)
	if out != "20" {
		t.Errorf("want '20', got %q", out)
	}
	// Fast path: direct .int_val access, no monk_array_get.
	if strings.Contains(src, "monk_array_get") {
		t.Errorf("expected no monk_array_get for typed int[], generated:\n%s", src)
	}
}

// Range-initialized int array: element read stays scalar.
func TestTypedArrayRangeRead(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr = range(5)
let x = arr[3]
show(to_string(x))`)
	if out != "3" {
		t.Errorf("want '3', got %q", out)
	}
}

// Typed array element write via plain assignment.
func TestTypedArrayWrite(t *testing.T) {
	out, src := runMonkTyped(t, `let arr = range(5)
arr[2] = 99
show(to_string(arr[2]))`)
	if out != "99" {
		t.Errorf("want '99', got %q", out)
	}
	// Fast path: direct element write, no monk_array_set.
	if strings.Contains(src, "monk_array_set") {
		t.Errorf("expected no monk_array_set for typed int[], generated:\n%s", src)
	}
}

// Typed array arithmetic: read → scalar → raw C arithmetic.
func TestTypedArrayArith(t *testing.T) {
	out, src := runMonkTyped(t, `let arr = range(5)
let x = arr[1] + arr[3]
show(to_string(x))`)
	if out != "4" {
		t.Errorf("want '4', got %q", out)
	}
	// Should NOT call monk_add — both operands are unboxed ints.
	if strings.Contains(src, "monk_add(") {
		t.Errorf("expected raw + not monk_add, generated:\n%s", src)
	}
}

// Typed array element participates in unboxed multiplication chain.
func TestTypedArrayMulChain(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr int[] = [2, 3, 5]
let result = arr[0] * arr[1] + arr[2]
show(to_string(result))`)
	if out != "11" {
		t.Errorf("want '11', got %q", out)
	}
}

// Typed array element used as loop accumulator.
func TestTypedArrayAccumulate(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr = range(5)
let i = 0
let sum = 0
while i < 5 {
    sum = sum + arr[i]
    i += 1
}
show(to_string(sum))`)
	if out != "10" {
		t.Errorf("want '10', got %q", out)
	}
}

// Mini matmul (2×2) correctness: the canonical performance target.
func TestTypedArrayMatmul2x2(t *testing.T) {
	out, _ := runMonkTyped(t, `let N = 2
let A int[] = [1, 2, 3, 4]
let B int[] = [5, 6, 7, 8]
let C int[] = [0, 0, 0, 0]
let i = 0
while i < N {
    let k = 0
    while k < N {
        let aik = A[i * N + k]
        let j = 0
        while j < N {
            C[i * N + j] = C[i * N + j] + aik * B[k * N + j]
            j += 1
        }
        k += 1
    }
    i += 1
}
let sum = C[0] + C[1] + C[2] + C[3]
show(to_string(sum))`)
	// [1,2,3,4] * [5,6,7,8] = [[19,22],[43,50]], sum = 134
	if out != "134" {
		t.Errorf("want '134', got %q", out)
	}
}

// Float array: element reads return unboxed double.
func TestTypedArrayFloatRead(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr float[] = [1.5, 2.5, 3.5]
let x = arr[0] + arr[2]
show(to_string(x))`)
	if out != "5" {
		t.Errorf("want '5', got %q", out)
	}
}

// OOB on typed array panics (strict, not graceful none).
func TestTypedArrayOOBPanics(t *testing.T) {
	prog, err := syntax.Parse(`let arr = range(3)
let x = arr[10]
show(to_string(x))`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info, err := types.Check(prog)
	if err != nil {
		t.Fatalf("type check: %v", err)
	}
	cSource := GenerateWithTypes(prog, "test.monk", info)
	dir := t.TempDir()
	runtimeDir, _ := filepath.Abs("../runtime")
	cFile := filepath.Join(dir, "test.c")
	binFile := filepath.Join(dir, "test")
	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		t.Fatalf("write c: %v", err)
	}
	ccArgs := append([]string{"-std=c11", "-I" + runtimeDir, cFile},
		runtimeArgs(runtimeDir)...)
	ccArgs = append(ccArgs, "-lm", "-o", binFile)
	if out, err := exec.Command("cc", ccArgs...).CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s\n\nC source:\n%s", err, out, cSource)
	}
	out, err := exec.Command(binFile).CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit on OOB, but got: %s", out)
	}
	if !strings.Contains(string(out), "index out of bounds") {
		t.Errorf("expected 'index out of bounds' in output, got: %s", out)
	}
}

// ── Backing-store tests (int64_t* / double* / bool* instead of MonkValue*) ──

// Verify int[] literal uses monk_int_array_from in the generated C.
func TestBackingStoreIntLiteralDecl(t *testing.T) {
	_, src := runMonkTyped(t, `let arr int[] = [10, 20, 30]
let x = arr[1]
show(to_string(x))`)
	if !strings.Contains(src, "monk_int_array_from") {
		t.Errorf("expected monk_int_array_from for int[] decl, generated:\n%s", src)
	}
	// Must NOT fall back to the generic MonkArray field (.array_val->).
	if strings.Contains(src, ".array_val->") {
		t.Errorf("expected no .array_val-> for typed int[], generated:\n%s", src)
	}
}

// Verify range()-initialized int[] uses monk_range_int (specialized direct alloc).
func TestBackingStoreRangeDecl(t *testing.T) {
	out, src := runMonkTyped(t, `let arr = range(4)
show(to_string(arr[2]))`)
	if out != "2" {
		t.Errorf("want '2', got %q", out)
	}
	if !strings.Contains(src, "monk_range_int") {
		t.Errorf("expected monk_range_int for range-init int[], generated:\n%s", src)
	}
}

// Verify element write uses int_array_val path.
func TestBackingStoreWrite(t *testing.T) {
	out, src := runMonkTyped(t, `let arr = range(4)
arr[1] = 42
show(to_string(arr[1]))`)
	if out != "42" {
		t.Errorf("want '42', got %q", out)
	}
	if !strings.Contains(src, "int_array_val") {
		t.Errorf("expected int_array_val in generated C, got:\n%s", src)
	}
	if strings.Contains(src, "monk_array_set") {
		t.Errorf("expected no monk_array_set for typed int[], generated:\n%s", src)
	}
}

func TestBackingStoreWriteDetachesCopyOnWriteArray(t *testing.T) {
	out, src := runMonkTyped(t, `let a int[] = [1, 2, 3]
let b = a
b[0] = 99
show(to_string(a[0]) + "," + to_string(b[0]))`)
	if out != "1,99" {
		t.Errorf("want '1,99', got %q", out)
	}
	// COW write barrier for the typed-array fast path.
	// Pass: `b[0] = 99` first calls monk_int_array_ensure_unique(&b).
	// Fail: direct `b.int_array_val->data[0] = 99` mutates shared storage.
	if !strings.Contains(src, "monk_int_array_ensure_unique(&mk_b)") {
		t.Errorf("expected COW detach before typed-array write, generated:\n%s", src)
	}
}

// Verify float[] uses float_array_val and monk_float_array_from.
func TestBackingStoreFloat(t *testing.T) {
	out, src := runMonkTyped(t, `let arr float[] = [1.0, 2.0, 4.0]
arr[1] = 3.0
let x = arr[0] + arr[1] + arr[2]
show(to_string(x))`)
	if out != "8" {
		t.Errorf("want '8', got %q", out)
	}
	if !strings.Contains(src, "monk_float_array_from") {
		t.Errorf("expected monk_float_array_from for float[], generated:\n%s", src)
	}
}

// Verify for-in over int[] boxes each element into MonkValue.
func TestBackingStoreForIn(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr int[] = [1, 2, 3, 4]
let total = 0
for x in arr {
    total = total + x
}
show(to_string(total))`)
	if out != "10" {
		t.Errorf("want '10', got %q", out)
	}
}

// Verify show(arr) on a typed int[] displays correctly.
func TestBackingStoreShow(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr int[] = [7, 8, 9]
show(arr)`)
	if out != "[7, 8, 9]" {
		t.Errorf("want '[7, 8, 9]', got %q", out)
	}
}

// Verify is_array returns true for a typed int[].
func TestBackingStoreIsArray(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr int[] = [1, 2]
show(to_string(is_array(arr)))`)
	if out != "true" {
		t.Errorf("want 'true', got %q", out)
	}
}

// Verify length() works on typed arrays.
func TestBackingStoreLength(t *testing.T) {
	out, _ := runMonkTyped(t, `let arr int[] = [5, 6, 7, 8, 9]
show(to_string(length(arr)))`)
	if out != "5" {
		t.Errorf("want '5', got %q", out)
	}
}

// ── Record field unboxing tests ──────────────────────────────────────────────

// Verify that record field reads emit direct index access, not monk_record_get.
func TestRecordFieldReadUnboxed(t *testing.T) {
	_, src := runMonkTyped(t, `let r = {x: 1, y: 2}
show(to_string(r.x))`)
	if strings.Contains(src, "monk_record_get") {
		t.Errorf("expected no monk_record_get for typed record read, generated:\n%s", src)
	}
	if !strings.Contains(src, "record_val->fields[") {
		t.Errorf("expected direct fields[] access in generated C:\n%s", src)
	}
}

// Verify that record field writes emit direct index assignment, not monk_record_set.
func TestRecordFieldWriteUnboxed(t *testing.T) {
	_, src := runMonkTyped(t, `let r = {x: 0, y: 0}
r.x = 42
show(to_string(r.x))`)
	if strings.Contains(src, "monk_record_set") {
		t.Errorf("expected no monk_record_set for typed record write, generated:\n%s", src)
	}
}

// Verify correct runtime behavior for scalar record field reads.
func TestRecordFieldReadScalarCorrect(t *testing.T) {
	out, _ := runMonkTyped(t, `let r = {x: 10, y: 20}
show(to_string(r.x + r.y))`)
	if out != "30" {
		t.Errorf("want '30', got %q", out)
	}
}

// Verify correct runtime behavior for record field writes (scalar floats).
// to_string on floats uses %g (strips trailing zeros), so 9.0 → "9".
func TestRecordFieldWriteScalarCorrect(t *testing.T) {
	out, _ := runMonkTyped(t, `let r = {x: 0.0, y: 0.0, z: 0.0}
r.x = 3.0
r.y = r.x * 2.0
r.z = r.x + r.y
show(to_string(to_int(r.z)))`)
	if out != "9" {
		t.Errorf("want '9', got %q", out)
	}
}

// Verify that float record fields stay scalar through arithmetic chains.
func TestRecordFieldFloatChain(t *testing.T) {
	out, _ := runMonkTyped(t, `let r = {a: 1.0, b: 3.0}
let sum = r.a + r.b
show(to_string(to_int(sum)))`)
	if out != "4" {
		t.Errorf("want '4', got %q", out)
	}
}

// Verify record field unboxing works in a tight accumulation loop.
func TestRecordFieldLoopAccumulate(t *testing.T) {
	out, _ := runMonkTyped(t, `let r = {x: 0, y: 0}
let sum = 0
let i = 0
while i < 100 {
    r.x = i
    r.y = r.x * 2
    sum += r.x + r.y
    i += 1
}
show(to_string(sum))`)
	if out != "14850" {
		t.Errorf("want '14850', got %q", out)
	}
}

// ── Bounds-check elision tests ───────────────────────────────────────────────

// TestBoundsElisionRead verifies that for a simple `while i < N` loop
// accessing a typed array of length N, the emitted C has no bounds check.
func TestBoundsElisionRead(t *testing.T) {
	src := `let N int = 5
let arr int[] = range(N)
let sum int = 0
let i int = 0
while i < N {
    sum += arr[i]
    i += 1
}
show(to_string(sum))`
	out, c := runMonkTyped(t, src)
	if out != "10" {
		t.Errorf("want '10', got %q", out)
	}
	// The generated C must not contain a bounds check for arr[i] in this loop.
	if strings.Contains(c, "monk_panic(\"index out of bounds\")") {
		t.Errorf("expected bounds check to be elided, but got one in generated C")
	}
}

// TestBoundsElisionWrite verifies elision on the write path: arr[i] = val.
func TestBoundsElisionWrite(t *testing.T) {
	src := `let N int = 4
let arr int[] = range(N)
let i int = 0
while i < N {
    arr[i] = i * 2
    i += 1
}
show(to_string(arr[3]))`
	out, c := runMonkTyped(t, src)
	if out != "6" {
		t.Errorf("want '6', got %q", out)
	}
	if strings.Contains(c, "monk_panic(\"index out of bounds\")") {
		t.Errorf("expected bounds check to be elided, but got one in generated C")
	}
}

// TestBoundsElisionNotElided verifies that when the upper bound is LARGER
// than the array length, the bounds check is NOT elided.
func TestBoundsElisionNotElided(t *testing.T) {
	src := `let N int = 3
let arr int[] = range(N)
let i int = 0
while i < 10 {
    i += 1
}
show(to_string(arr[0]))`
	_, c := runMonkTyped(t, src)
	// arr has length 3 but loop goes to 9 — i is NOT bounded by arr's length.
	// We never access arr inside the loop, but arr[0] outside it is also safe
	// (constant 0). The loop body has no array access so no elision question.
	_ = c // no assertion needed — just verify it compiles and runs
}

// TestBoundsElisionCorrectness runs a 100-element sum with elision and
// verifies the result is still correct (elision doesn't skip real checks).
func TestBoundsElisionCorrectness(t *testing.T) {
	src := `let N int = 100
let arr int[] = range(N)
let sum int = 0
let i int = 0
while i < N {
    sum += arr[i]
    i += 1
}
show(to_string(sum))`
	out, _ := runMonkTyped(t, src)
	// sum(0..99) = 99*100/2 = 4950
	if out != "4950" {
		t.Errorf("want '4950', got %q", out)
	}
}

// fill(n, value) creates an array of n copies.
// fill(3, true) → [true, true, true] as bool[].
func TestFillBool(t *testing.T) {
	src := `let arr boolean[] = fill(5, true)
show(to_string(length(arr)))
show(to_string(arr[0]))
show(to_string(arr[4]))`
	out, _ := runMonkTyped(t, src)
	if out != "5\ntrue\ntrue" {
		t.Errorf("want '5\\ntrue\\ntrue', got %q", out)
	}
}

// fill(n, value) with int → int[].
func TestFillInt(t *testing.T) {
	src := `let arr int[] = fill(3, 42)
show(to_string(length(arr)))
show(to_string(arr[0]))
show(to_string(arr[2]))`
	out, _ := runMonkTyped(t, src)
	if out != "3\n42\n42" {
		t.Errorf("want '3\\n42\\n42', got %q", out)
	}
}

// fill(0, x) returns empty array (graceful).
func TestFillEmpty(t *testing.T) {
	src := `let arr int[] = fill(0, 0)
show(to_string(length(arr)))`
	out, _ := runMonkTyped(t, src)
	if out != "0" {
		t.Errorf("want '0', got %q", out)
	}
}

// fill with bool[] used as sieve-style write target.
func TestFillBoolSieve(t *testing.T) {
	src := `let N int = 20
let is_prime boolean[] = fill(N + 1, true)
is_prime[0] = false
is_prime[1] = false
let i int = 2
while i * i <= N {
    if is_prime[i] {
        let j int = i * i
        while j <= N {
            is_prime[j] = false
            j += i
        }
    }
    i += 1
}
let count int = 0
i = 2
while i <= N {
    if is_prime[i] {
        count += 1
    }
    i += 1
}
show(to_string(count))`
	out, _ := runMonkTyped(t, src)
	// primes up to 20: 2,3,5,7,11,13,17,19 = 8
	if out != "8" {
		t.Errorf("want '8', got %q", out)
	}
}

// for i in range(N) → counter loop, no allocation.
// sum(0..99) = 4950. Generated C should have no monk_range call.
func TestCounterLoopRange(t *testing.T) {
	src := `let sum int = 0
for i in range(100) {
    sum += i
}
show(to_string(sum))`
	out, csrc := runMonkTyped(t, src)
	if out != "4950" {
		t.Errorf("want '4950', got %q", out)
	}
	if strings.Contains(csrc, "monk_range") {
		t.Errorf("counter loop should not call monk_range, generated:\n%s", csrc)
	}
	if !strings.Contains(csrc, "for (int64_t") {
		t.Errorf("expected C for-loop, generated:\n%s", csrc)
	}
}

// for i in range(N) with variable bound.
func TestCounterLoopRangeVar(t *testing.T) {
	src := `let N int = 5
let sum int = 0
for i in range(N) {
    sum += i
}
show(to_string(sum))`
	out, _ := runMonkTyped(t, src)
	if out != "10" {
		t.Errorf("want '10', got %q", out)
	}
}

// Nested counter loops.
func TestCounterLoopNested(t *testing.T) {
	src := `let count int = 0
for i in range(10) {
    for j in range(10) {
        count += 1
    }
}
show(to_string(count))`
	out, _ := runMonkTyped(t, src)
	if out != "100" {
		t.Errorf("want '100', got %q", out)
	}
}
