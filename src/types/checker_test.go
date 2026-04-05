package types

import (
	"strings"
	"testing"

	"github.com/monkfromearth/monk-lang/syntax"
)

// checkSrc parses source and runs the type checker. Returns nil on success.
func checkSrc(t *testing.T, src string) error {
	t.Helper()
	prog, err := syntax.Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return Check(prog)
}

// expectOk asserts the program type-checks cleanly.
func expectOk(t *testing.T, src string) {
	t.Helper()
	if err := checkSrc(t, src); err != nil {
		t.Fatalf("expected ok, got type error:\n  %v\n  source:\n%s", err, src)
	}
}

// expectErr asserts the program fails type-check with a message containing
// `want` (substring match, so tests stay stable if we tweak wording).
func expectErr(t *testing.T, src, want string) {
	t.Helper()
	err := checkSrc(t, src)
	if err == nil {
		t.Fatalf("expected type error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error message mismatch:\n  got:  %s\n  want: contains %q", err, want)
	}
}

// ─── Primitives & literals ────────────────────────────────────────────────

func TestCheckIntLiteral(t *testing.T) {
	expectOk(t, `let x = 42`)
}

func TestCheckFloatLiteral(t *testing.T) {
	expectOk(t, `let x = 3.14`)
}

func TestCheckStringLiteral(t *testing.T) {
	expectOk(t, `let x = "hello"`)
}

// ─── First-assignment inference ───────────────────────────────────────────

func TestCheckInferIntThenReassign(t *testing.T) {
	expectOk(t, `let x = 42
x = 7`)
}

func TestCheckInferIntRejectsString(t *testing.T) {
	expectErr(t, `let x = 42
x = "hi"`, "cannot assign string to variable 'x' of type int")
}

func TestCheckInferArrayElementType(t *testing.T) {
	expectErr(t, `let nums = [1, 2, 3]
nums[0] = "oops"`, "cannot assign string to int[] element")
}

// ─── Explicit annotations ─────────────────────────────────────────────────

func TestCheckAnnotatedInt(t *testing.T) {
	expectOk(t, `let x int = 5`)
}

func TestCheckAnnotatedMismatch(t *testing.T) {
	expectErr(t, `let x int = "hi"`,
		"cannot assign string to int in declaration of 'x'")
}

func TestCheckAnnotatedFloatAcceptsInt(t *testing.T) {
	// Numeric widening: int flows into float slot.
	expectOk(t, `let x float = 5`)
}

func TestCheckAnnotatedIntRejectsFloat(t *testing.T) {
	// No narrowing.
	expectErr(t, `let x int = 5.0`, "cannot assign float to int")
}

// ─── Optional types ───────────────────────────────────────────────────────

func TestCheckOptionalAcceptsNone(t *testing.T) {
	expectOk(t, `let x int? = none`)
}

func TestCheckOptionalAcceptsBase(t *testing.T) {
	expectOk(t, `let x int? = 42`)
}

func TestCheckNonOptionalRejectsNone(t *testing.T) {
	expectErr(t, `let x int = none`, "cannot assign none to int")
}

// Regression (CodeRabbit): T? cannot flow into T. At runtime int? may be none,
// which would silently appear as an int — defeats the whole point of optional.
func TestCheckOptionalCannotFlowToRequired(t *testing.T) {
	expectErr(t, `let x int? = none
let y int = x`, "cannot assign int? to int")
}

// Regression: AssignableTo recursion was only stripping Optional from dst, not
// src, so int? -> int? fell through to the "src.Optional rejected" branch.
func TestCheckOptionalFlowsToOptional(t *testing.T) {
	expectOk(t, `let arr int[] = [10, 20, 30]
let first int? = arr[0]
let second int? = first`)
}

func TestCheckIndexReadIsOptional(t *testing.T) {
	// Array index always returns T? (graceful read — may be none).
	// Assigning it to a non-optional slot must be rejected.
	expectErr(t, `let xs = [1, 2, 3]
let first int = xs[0]`, "cannot assign")
}

// ─── Const ─────────────────────────────────────────────────────────────────

func TestCheckConstReassignError(t *testing.T) {
	expectErr(t, `const x = 5
x = 6`, "cannot assign to const 'x'")
}

// ─── Arithmetic ────────────────────────────────────────────────────────────

func TestCheckArithIntInt(t *testing.T) {
	expectOk(t, `let s = 1 + 2`)
}

func TestCheckArithIntFloat(t *testing.T) {
	expectOk(t, `let s float = 1 + 2.0`)
}

func TestCheckStringConcat(t *testing.T) {
	// Plus has a string-concat overload: string + string is OK.
	expectOk(t, `let greeting = "Hello, " + "World"`)
}

func TestCheckStringConcatMixed(t *testing.T) {
	expectErr(t, `let s = "a" + 1`,
		"cannot mix string and int")
}

func TestCheckStringConcatWithToString(t *testing.T) {
	// The canonical pattern from the spec.
	expectOk(t, `let s = "Number: " + to_string(42)`)
}

func TestCheckArithRejectsBool(t *testing.T) {
	expectErr(t, `let s = true + false`,
		"requires numeric operands")
}

func TestCheckUntypedArrayParam(t *testing.T) {
	// Per examples/sort.monk — unparameterized 'array' annotation.
	expectOk(t, `let sort = (arr array) array { return arr }
show(to_string(sort([1, 2, 3])))`)
}

func TestCheckBitwiseRequiresInt(t *testing.T) {
	expectErr(t, `let s = 1.0 & 2`, "bitwise operator requires int operands")
}

// ─── Arrays ────────────────────────────────────────────────────────────────

func TestCheckArrayHomogeneous(t *testing.T) {
	expectOk(t, `let a = [1, 2, 3]`)
}

func TestCheckArrayMixedTypeError(t *testing.T) {
	expectErr(t, `let a = [1, "hi"]`, "array elements must be same type")
}

func TestCheckArrayIntFloatWidens(t *testing.T) {
	// [1, 2.0] becomes float[] via numeric widening.
	expectOk(t, `let a = [1, 2.0, 3]`)
}

func TestCheckTypedArrayRejectsWrongElement(t *testing.T) {
	expectErr(t, `let nums int[] = [1, "x", 3]`, "array elements must be same type")
}

func TestCheckIndexAssignmentTypeEnforcement(t *testing.T) {
	expectErr(t, `let nums int[] = [1, 2, 3]
nums[0] = "x"`, "cannot assign string to int[] element")
}

// ─── Records ───────────────────────────────────────────────────────────────

func TestCheckRecordLiteral(t *testing.T) {
	expectOk(t, `let p = {x: 1, y: 2}`)
}

func TestCheckRecordDuplicateField(t *testing.T) {
	expectErr(t, `let p = {x: 1, x: 2}`, "duplicate field 'x'")
}

func TestCheckTypedRecord(t *testing.T) {
	expectOk(t, `type Point = { x: int, y: int }
let p Point = {x: 1, y: 2}`)
}

func TestCheckTypedRecordMissingField(t *testing.T) {
	expectErr(t, `type Point = { x: int, y: int }
let p Point = {x: 1}`, "cannot assign")
}

func TestCheckTypedRecordExtraField(t *testing.T) {
	expectErr(t, `type Point = { x: int, y: int }
let p Point = {x: 1, y: 2, z: 3}`, "cannot assign")
}

func TestCheckTypedRecordWrongFieldType(t *testing.T) {
	expectErr(t, `type Point = { x: int, y: int }
let p Point = {x: 1, y: "oops"}`, "cannot assign")
}

func TestCheckTypedRecordNoDynamicFieldWrite(t *testing.T) {
	expectErr(t, `type Point = { x: int, y: int }
let p Point = {x: 1, y: 2}
p.z = 3`, "has no field 'z'")
}

// ─── Functions ────────────────────────────────────────────────────────────

func TestCheckFuncReturnsDeclared(t *testing.T) {
	expectOk(t, `let add = (a int, b int) int { return a + b }
show(to_string(add(1, 2)))`)
}

func TestCheckFuncReturnMismatch(t *testing.T) {
	expectErr(t, `let bad = (a int) int { return "x" }`,
		"cannot return string from function returning int")
}

func TestCheckFuncCallWrongArgCount(t *testing.T) {
	expectErr(t, `let add = (a int, b int) int { return a + b }
show(to_string(add(1)))`, "wrong number of arguments")
}

func TestCheckFuncCallWrongArgType(t *testing.T) {
	expectErr(t, `let add = (a int, b int) int { return a + b }
show(to_string(add(1, "x")))`, "cannot pass string to parameter of type int")
}

func TestCheckFuncRecursion(t *testing.T) {
	expectOk(t, `let fib = (n int) int {
    if n < 2 { return n }
    return fib(n - 1) + fib(n - 2)
}
show(to_string(fib(10)))`)
}

// ─── Function-type parameters ─────────────────────────────────────────────

func TestCheckFuncTypeParam(t *testing.T) {
	// Higher-order fn with function-type parameter annotation.
	expectOk(t, `let apply = (f (int) -> int, x int) int { return f(x) }
let double = (n int) int { return n * 2 }
show(to_string(apply(double, 21)))`)
}

func TestCheckFuncTypeParamMismatchArg(t *testing.T) {
	expectErr(t, `let apply = (f (int) -> int, x int) int { return f(x) }
let sq = (n int, m int) int { return n * m }
show(to_string(apply(sq, 21)))`, "cannot pass")
}

func TestCheckFuncTypeMultipleArgs(t *testing.T) {
	expectOk(t, `let combine = (op (int, int) -> int, a int, b int) int {
  return op(a, b)
}
let add = (x int, y int) int { return x + y }
show(to_string(combine(add, 3, 4)))`)
}

func TestCheckFuncTypeInTypeDecl(t *testing.T) {
	expectOk(t, `type BinaryOp = (int, int) -> int
let add = (a int, b int) int { return a + b }
let op BinaryOp = add`)
}

// Regression: before CodeRabbit CR-2, funcExactMatch used AssignableTo on
// params, which allowed int->float widening. That's unsound: a fn typed
// (float) -> int accessed via a (int) -> int slot would let callers pass
// a float where the underlying fn expects an int.
func TestCheckFuncTypeParamWidenRejected(t *testing.T) {
	expectErr(t, `type FloatFn = (float) -> int
let intFn = (n int) int { return n }
let f FloatFn = intFn`, "cannot assign")
}

// ─── Control flow ─────────────────────────────────────────────────────────

func TestCheckForOverArray(t *testing.T) {
	expectOk(t, `let nums = [1, 2, 3]
for n in nums { show(to_string(n)) }`)
}

func TestCheckForOverString(t *testing.T) {
	expectOk(t, `for c in "hello" { show(c) }`)
}

func TestCheckForOverInt(t *testing.T) {
	expectErr(t, `for x in 42 { show(to_string(x)) }`, "cannot iterate over int")
}

// Break/continue outside loops are caught by the parser, not the type checker.

func TestCheckReturnOutsideFunc(t *testing.T) {
	expectErr(t, `return 42`, "return outside of function")
}

// ─── Undefined / scoping ──────────────────────────────────────────────────

func TestCheckUndefinedVariable(t *testing.T) {
	expectErr(t, `show(to_string(x))`, "undefined variable 'x'")
}

func TestCheckScopeIsolation(t *testing.T) {
	expectErr(t, `if true { let y = 5 }
show(to_string(y))`, "undefined variable 'y'")
}

// ─── All-paths-return ──────────────────────────────────────────────────────

func TestCheckNoReturnErrors(t *testing.T) {
	expectErr(t, `let add = (a int, b int) int {
  let x = a + b
}`, "function may exit without returning int")
}

func TestCheckIfWithoutElseErrors(t *testing.T) {
	expectErr(t, `let pick = (x int) int {
  if x > 0 { return 1 }
}`, "function may exit without returning int")
}

func TestCheckIfElseBothReturnOk(t *testing.T) {
	expectOk(t, `let pick = (x int) int {
  if x > 0 { return 1 } else { return 0 }
}`)
}

func TestCheckElseIfChainOk(t *testing.T) {
	expectOk(t, `let classify = (n int) string {
  if n < 0 { return "neg" }
  else if n == 0 { return "zero" }
  else { return "pos" }
}`)
}

func TestCheckThrowTerminatesPath(t *testing.T) {
	expectOk(t, `let guard_pos = (n int) int {
  if n < 0 { throw "negative" }
  return n
}`)
}

func TestCheckNoneReturnAllowed(t *testing.T) {
	// Functions returning none don't need explicit return.
	expectOk(t, `let greet = (name string) none {
  show("hi " + name)
}`)
}

func TestCheckAnyReturnAllowed(t *testing.T) {
	// Functions without explicit return type default to Any — permissive.
	expectOk(t, `let maybe = () {
  if true { return 1 }
}`)
}

func TestCheckLoopBodyMissingReturnErrors(t *testing.T) {
	// A while loop may not execute — can't rely on it to return.
	expectErr(t, `let mystery = (n int) int {
  while n > 0 { return n }
}`, "function may exit without returning int")
}

// ─── Loop variable is const ────────────────────────────────────────────────

func TestCheckLoopVarIsConst(t *testing.T) {
	expectErr(t, `for i in range(5) {
  i = 99
}`, "cannot assign to const 'i'")
}

// ─── Equality rules ────────────────────────────────────────────────────────

func TestCheckEqualitySameTypes(t *testing.T) {
	expectOk(t, `let a = 1 == 2
let b = "x" == "y"
let c = true == false`)
}

func TestCheckEqualityIntFloat(t *testing.T) {
	// int/float can be compared (numeric widening).
	expectOk(t, `let eq = 1 == 2.0`)
}

func TestCheckEqualityCrossTypeError(t *testing.T) {
	expectErr(t, `let eq = 5 == "5"`, "no implicit cross-type comparison")
}

func TestCheckEqualityNoneAlwaysOk(t *testing.T) {
	// Per spec: comparing to none is always allowed.
	expectOk(t, `let a = 42 == none
let b = "x" == none`)
}

func TestCheckEqualityOptionalToBase(t *testing.T) {
	expectOk(t, `let x int? = 5
let b = x == 42`)
}

func TestCheckEqualityArraysError(t *testing.T) {
	expectErr(t, `let eq = [1, 2] == [1, 2]`, "cannot compare")
}

func TestCheckEqualityRecordsError(t *testing.T) {
	expectErr(t, `let a = {x: 1}
let b = {x: 1}
let eq = a == b`, "cannot compare")
}

// ─── Real-world examples — the test suite's examples/ programs ────────────

func TestCheckFibonacciExample(t *testing.T) {
	expectOk(t, `let fibonacci = (n int) int {
    if n <= 1 { return n }
    return fibonacci(n - 1) + fibonacci(n - 2)
}

for i in range(10) {
    show(to_string(fibonacci(i)))
}`)
}

func TestCheckArraysExample(t *testing.T) {
	expectOk(t, `let nums = [1, 2, 3, 4, 5]
show(to_string(nums))
show(to_string(length(nums)))
let doubled = append(nums, 6)
show(to_string(doubled))`)
}
