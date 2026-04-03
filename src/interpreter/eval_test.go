package interpreter

import "testing"

// --- Helpers ---

func run(t *testing.T, source string) Value {
	t.Helper()
	result, err := Eval(source)
	if err != nil {
		t.Fatalf("Eval(%q) error: %v", source, err)
	}
	return result
}

func runError(t *testing.T, source string) error {
	t.Helper()
	_, err := Eval(source)
	if err == nil {
		t.Fatalf("Eval(%q) expected error, got none", source)
	}
	return err
}

func expectInt(t *testing.T, source string, expected int64) {
	t.Helper()
	v := run(t, source)
	if v.Kind != IntValue {
		t.Fatalf("Eval(%q): expected int, got %s", source, v.TypeName())
	}
	if v.Int != expected {
		t.Errorf("Eval(%q): expected %d, got %d", source, expected, v.Int)
	}
}

func expectFloat(t *testing.T, source string, expected float64) {
	t.Helper()
	v := run(t, source)
	if v.Kind != FloatValue {
		t.Fatalf("Eval(%q): expected float, got %s", source, v.TypeName())
	}
	if v.Float != expected {
		t.Errorf("Eval(%q): expected %g, got %g", source, expected, v.Float)
	}
}

func expectString(t *testing.T, source string, expected string) {
	t.Helper()
	v := run(t, source)
	if v.Kind != StringValue {
		t.Fatalf("Eval(%q): expected string, got %s", source, v.TypeName())
	}
	if v.Str != expected {
		t.Errorf("Eval(%q): expected %q, got %q", source, expected, v.Str)
	}
}

func expectBool(t *testing.T, source string, expected bool) {
	t.Helper()
	v := run(t, source)
	if v.Kind != BoolValue {
		t.Fatalf("Eval(%q): expected boolean, got %s", source, v.TypeName())
	}
	if v.Bool != expected {
		t.Errorf("Eval(%q): expected %v, got %v", source, expected, v.Bool)
	}
}

func expectNone(t *testing.T, source string) {
	t.Helper()
	v := run(t, source)
	if v.Kind != NoneValue {
		t.Fatalf("Eval(%q): expected none, got %s (%s)", source, v.TypeName(), v.String())
	}
}

// === INTEGER ARITHMETIC ===

func TestEvalIntLiteral(t *testing.T)  { expectInt(t, "42", 42) }
func TestEvalIntZero(t *testing.T)     { expectInt(t, "0", 0) }
func TestEvalIntAdd(t *testing.T)      { expectInt(t, "1 + 2", 3) }
func TestEvalIntSub(t *testing.T)      { expectInt(t, "10 - 3", 7) }
func TestEvalIntMul(t *testing.T)      { expectInt(t, "4 * 5", 20) }
func TestEvalIntDiv(t *testing.T)      { expectInt(t, "10 / 3", 3) }
func TestEvalIntDivExact(t *testing.T) { expectInt(t, "10 / 2", 5) }
func TestEvalIntMod(t *testing.T)      { expectInt(t, "7 % 3", 1) }
func TestEvalIntNeg(t *testing.T)      { expectInt(t, "-42", -42) }
func TestEvalIntDoubleNeg(t *testing.T) { expectInt(t, "--5", 5) }

func TestEvalIntPrecedence(t *testing.T) { expectInt(t, "1 + 2 * 3", 7) }
func TestEvalIntParens(t *testing.T)     { expectInt(t, "(1 + 2) * 3", 9) }
func TestEvalIntComplex(t *testing.T)    { expectInt(t, "2 + 3 * 4 - 1", 13) }
func TestEvalIntChainedAdd(t *testing.T) { expectInt(t, "1 + 2 + 3 + 4", 10) }
func TestEvalIntNestedParens(t *testing.T) { expectInt(t, "((1 + 2) * (3 + 4))", 21) }

// === FLOAT ARITHMETIC ===

func TestEvalFloatLiteral(t *testing.T) { expectFloat(t, "3.14", 3.14) }
func TestEvalFloatAdd(t *testing.T)     { expectFloat(t, "1.5 + 2.5", 4.0) }
func TestEvalFloatMixedAdd(t *testing.T) { expectFloat(t, "1 + 2.0", 3.0) }
func TestEvalFloatMixedMul(t *testing.T) { expectFloat(t, "3 * 2.5", 7.5) }
func TestEvalFloatDiv(t *testing.T)      { expectFloat(t, "10.0 / 3.0", 10.0/3.0) }
func TestEvalIntFloatDiv(t *testing.T)   { expectFloat(t, "10.0 / 3", 10.0/3.0) }

// === DIVISION AND MODULO ERRORS ===

func TestEvalDivByZero(t *testing.T) { runError(t, "10 / 0") }
func TestEvalModByZero(t *testing.T) { runError(t, "10 % 0") }

// === STRING OPERATIONS ===

func TestEvalStringLiteral(t *testing.T)  { expectString(t, `"hello"`, "hello") }
func TestEvalStringEmpty(t *testing.T)    { expectString(t, `""`, "") }
func TestEvalStringConcat(t *testing.T)   { expectString(t, `"hello" + " " + "world"`, "hello world") }
func TestEvalTemplateLiteral(t *testing.T) { expectString(t, "`template`", "template") }

func TestEvalStringPlusIntError(t *testing.T)  { runError(t, `"hello" + 42`) }
func TestEvalStringPlusBoolError(t *testing.T) { runError(t, `"hello" + true`) }

// === BOOLEAN / COMPARISON ===

func TestEvalTrue(t *testing.T)  { expectBool(t, "true", true) }
func TestEvalFalse(t *testing.T) { expectBool(t, "false", false) }
func TestEvalNone(t *testing.T)  { expectNone(t, "none") }

func TestEvalEqualInt(t *testing.T)       { expectBool(t, "5 == 5", true) }
func TestEvalNotEqualInt(t *testing.T)    { expectBool(t, "5 != 3", true) }
func TestEvalLess(t *testing.T)           { expectBool(t, "3 < 5", true) }
func TestEvalGreater(t *testing.T)        { expectBool(t, "5 > 3", true) }
func TestEvalLessEqual(t *testing.T)      { expectBool(t, "3 <= 3", true) }
func TestEvalGreaterEqual(t *testing.T)   { expectBool(t, "5 >= 5", true) }
func TestEvalLessFalse(t *testing.T)      { expectBool(t, "5 < 3", false) }
func TestEvalEqualString(t *testing.T)    { expectBool(t, `"abc" == "abc"`, true) }
func TestEvalNotEqualString(t *testing.T) { expectBool(t, `"abc" != "def"`, true) }
func TestEvalStringLess(t *testing.T)     { expectBool(t, `"abc" < "abd"`, true) }
func TestEvalStringGreater(t *testing.T)  { expectBool(t, `"b" > "a"`, true) }
func TestEvalIsKeyword(t *testing.T)      { expectBool(t, "5 is 5", true) }
func TestEvalNoneEqNone(t *testing.T)     { expectBool(t, "none == none", true) }

func TestEvalCrossTypeEqualError(t *testing.T) { runError(t, `5 == "5"`) }
func TestEvalCrossTypeLessError(t *testing.T)  { runError(t, `5 < "hello"`) }
func TestEvalNoneOrderError(t *testing.T)      { runError(t, "none < 5") }

// === LOGICAL / TRUTHINESS ===

func TestEvalNotTrue(t *testing.T)     { expectBool(t, "not true", false) }
func TestEvalNotFalse(t *testing.T)    { expectBool(t, "not false", true) }
func TestEvalBangTrue(t *testing.T)    { expectBool(t, "!true", false) }
func TestEvalNotNone(t *testing.T)     { expectBool(t, "not none", true) }
func TestEvalNotZero(t *testing.T)     { expectBool(t, "not 0", true) }
func TestEvalNotOne(t *testing.T)      { expectBool(t, "not 1", false) }
func TestEvalNotString(t *testing.T)   { expectBool(t, `not ""`, false) } // "" is truthy
func TestEvalNotEmptyArr(t *testing.T) { expectBool(t, "not []", false) } // [] is truthy

func TestEvalAndTrueTrue(t *testing.T)   { expectBool(t, "true and true", true) }
func TestEvalAndTrueFalse(t *testing.T)  { expectBool(t, "true and false", false) }
func TestEvalAndFalseShort(t *testing.T) { expectBool(t, "false and true", false) }
func TestEvalOrFalseTrue(t *testing.T)   { expectBool(t, "false or true", true) }
func TestEvalOrTrueShort(t *testing.T)   { expectBool(t, "true or false", true) }
func TestEvalOrFalseFalse(t *testing.T)  { expectBool(t, "false or false", false) }

func TestEvalAndReturnsBoolean(t *testing.T) { expectBool(t, "1 and 2", true) }
func TestEvalOrReturnsBoolean(t *testing.T)  { expectBool(t, "0 or 5", true) }
func TestEvalOrZeroZero(t *testing.T)        { expectBool(t, "0 or 0", false) }

// === BITWISE ===

func TestEvalBitwiseAnd(t *testing.T) { expectInt(t, "0xFF & 0x0F", 0x0F) }
func TestEvalBitwiseOr(t *testing.T)  { expectInt(t, "0xF0 | 0x0F", 0xFF) }
func TestEvalBitwiseXor(t *testing.T) { expectInt(t, "0xFF ^ 0x0F", 0xF0) }
func TestEvalBitwiseNot(t *testing.T) { expectInt(t, "~0", -1) }
func TestEvalShiftLeft(t *testing.T)  { expectInt(t, "1 << 4", 16) }
func TestEvalShiftRight(t *testing.T) { expectInt(t, "16 >> 4", 1) }

// === VARIABLES ===

func TestEvalLetDecl(t *testing.T) { expectInt(t, "let x = 42\nx", 42) }
func TestEvalConstDecl(t *testing.T) { expectInt(t, "const x = 42\nx", 42) }
func TestEvalLetReassign(t *testing.T) { expectInt(t, "let x = 1\nx = 2\nx", 2) }
func TestEvalConstReassignError(t *testing.T) { runError(t, "const x = 1\nx = 2") }
func TestEvalUndefinedVarError(t *testing.T) { runError(t, "x") }
func TestEvalCompoundAssign(t *testing.T) { expectInt(t, "let x = 10\nx += 5\nx", 15) }
func TestEvalCompoundSub(t *testing.T) { expectInt(t, "let x = 10\nx -= 3\nx", 7) }
func TestEvalCompoundMul(t *testing.T) { expectInt(t, "let x = 4\nx *= 3\nx", 12) }
func TestEvalCompoundDiv(t *testing.T) { expectInt(t, "let x = 20\nx /= 4\nx", 5) }
func TestEvalCompoundMod(t *testing.T) { expectInt(t, "let x = 7\nx %= 3\nx", 1) }

// === DEEP CONST ===

func TestEvalConstArrayMutateError(t *testing.T) {
	runError(t, "const arr = [1, 2, 3]\narr[0] = 99")
}

func TestEvalConstRecordMutateError(t *testing.T) {
	runError(t, `const person = {name: "Alice"}`+"\n"+`person.name = "Bob"`)
}

func TestEvalLetArrayMutate(t *testing.T) {
	expectInt(t, "let arr = [1, 2, 3]\narr[0] = 99\narr[0]", 99)
}

func TestEvalLetRecordMutate(t *testing.T) {
	expectString(t, `let r = {name: "Alice"}`+"\n"+`r.name = "Bob"`+"\n"+`r.name`, "Bob")
}

// === VALUE SEMANTICS (COPY ON ASSIGN) ===

func TestEvalArrayCopyOnAssign(t *testing.T) {
	expectInt(t, "let a = [1, 2, 3]\nlet b = a\nb[0] = 99\na[0]", 1)
}

func TestEvalRecordCopyOnAssign(t *testing.T) {
	expectString(t, `let a = {name: "Alice"}`+"\n"+`let b = a`+"\n"+`b.name = "Bob"`+"\n"+`a.name`, "Alice")
}

// === ARRAYS ===

func TestEvalArrayLiteral(t *testing.T) {
	v := run(t, "[1, 2, 3]")
	if v.Kind != ArrayValue || len(v.Array) != 3 {
		t.Fatalf("expected array of 3, got %s", v.String())
	}
}

func TestEvalArrayEmpty(t *testing.T) {
	v := run(t, "[]")
	if v.Kind != ArrayValue || len(v.Array) != 0 {
		t.Fatalf("expected empty array, got %s", v.String())
	}
}

func TestEvalArrayIndex(t *testing.T)     { expectInt(t, "let a = [10, 20, 30]\na[0]", 10) }
func TestEvalArrayIndexLast(t *testing.T) { expectInt(t, "let a = [10, 20, 30]\na[2]", 30) }
func TestEvalArrayIndexOutOfBounds(t *testing.T) { expectNone(t, "let a = [1, 2, 3]\na[10]") }
func TestEvalArrayIndexNegative(t *testing.T) { expectNone(t, "let a = [1, 2, 3]\na[-1]") }

func TestEvalArrayIndexAssign(t *testing.T) {
	expectInt(t, "let a = [1, 2, 3]\na[1] = 99\na[1]", 99)
}

func TestEvalArrayIndexAssignOutOfBoundsError(t *testing.T) {
	runError(t, "let a = [1, 2, 3]\na[10] = 99")
}

// === RECORDS ===

func TestEvalRecordLiteral(t *testing.T) {
	v := run(t, `{name: "Alice", age: 30}`)
	if v.Kind != RecordValue || len(v.Record) != 2 {
		t.Fatalf("expected record with 2 fields, got %s", v.String())
	}
}

func TestEvalRecordAccess(t *testing.T) {
	expectString(t, `let r = {name: "Alice"}`+"\n"+`r.name`, "Alice")
}

func TestEvalRecordMissingFieldNone(t *testing.T) {
	expectNone(t, `let r = {a: 1}`+"\n"+`r.missing`)
}

func TestEvalRecordNewFieldError(t *testing.T) {
	runError(t, `let r = {a: 1}`+"\n"+`r.b = 2`)
}

func TestEvalRecordNested(t *testing.T) {
	expectString(t, `let addr = {city: "SF"}`+"\n"+`let c = {addr: addr}`+"\n"+`c.addr.city`, "SF")
}

// === IF / ELSE ===

func TestEvalIfTrue(t *testing.T) {
	expectInt(t, "let x = 0\nif true { x = 1 }\nx", 1)
}

func TestEvalIfFalse(t *testing.T) {
	expectInt(t, "let x = 0\nif false { x = 1 }\nx", 0)
}

func TestEvalIfElse(t *testing.T) {
	expectInt(t, "let x = 0\nif false { x = 1 } else { x = 2 }\nx", 2)
}

func TestEvalIfElseIf(t *testing.T) {
	expectInt(t, "let x = 0\nif false { x = 1 } else if true { x = 2 } else { x = 3 }\nx", 2)
}

func TestEvalIfTruthyZero(t *testing.T) {
	expectInt(t, "let x = 0\nif 0 { x = 1 }\nx", 0) // 0 is falsy
}

func TestEvalIfTruthyNone(t *testing.T) {
	expectInt(t, "let x = 0\nif none { x = 1 }\nx", 0) // none is falsy
}

func TestEvalIfTruthyOne(t *testing.T) {
	expectInt(t, "let x = 0\nif 1 { x = 1 }\nx", 1) // 1 is truthy
}

func TestEvalIfTruthyEmptyString(t *testing.T) {
	expectInt(t, `let x = 0`+"\n"+`if "" { x = 1 }`+"\n"+`x`, 1) // "" is truthy
}

// === WHILE LOOPS ===

func TestEvalWhile(t *testing.T) {
	expectInt(t, "let x = 0\nwhile x < 5 { x += 1 }\nx", 5)
}

func TestEvalWhileFalse(t *testing.T) {
	expectInt(t, "let x = 0\nwhile false { x = 1 }\nx", 0)
}

func TestEvalWhileBreak(t *testing.T) {
	expectInt(t, "let x = 0\nwhile true { x += 1\nif x == 3 { break } }\nx", 3)
}

func TestEvalWhileContinue(t *testing.T) {
	// Count only odd iterations
	source := `let sum = 0
let i = 0
while i < 6 {
    i += 1
    if i % 2 == 0 { continue }
    sum += i
}
sum`
	expectInt(t, source, 9) // 1 + 3 + 5
}

// === FOR LOOPS ===

func TestEvalForArray(t *testing.T) {
	expectInt(t, "let sum = 0\nfor x in [1, 2, 3] { sum += x }\nsum", 6)
}

func TestEvalForString(t *testing.T) {
	source := `let count = 0
for c in "hello" { count += 1 }
count`
	expectInt(t, source, 5)
}

func TestEvalForBreak(t *testing.T) {
	source := `let sum = 0
for x in [1, 2, 3, 4, 5] {
    if x == 4 { break }
    sum += x
}
sum`
	expectInt(t, source, 6) // 1 + 2 + 3
}

func TestEvalForContinue(t *testing.T) {
	source := `let sum = 0
for x in [1, 2, 3, 4, 5] {
    if x % 2 == 0 { continue }
    sum += x
}
sum`
	expectInt(t, source, 9) // 1 + 3 + 5
}

func TestEvalForLoopVarConst(t *testing.T) {
	// Loop variable is const — cannot reassign
	runError(t, "for x in [1, 2, 3] { x = 99 }")
}

func TestEvalForEmpty(t *testing.T) {
	expectInt(t, "let sum = 0\nfor x in [] { sum += 1 }\nsum", 0)
}

// === FUNCTIONS ===

func TestEvalFuncCall(t *testing.T) {
	expectInt(t, "let add = (a int, b int) int { return a + b }\nadd(3, 4)", 7)
}

func TestEvalFuncNoParams(t *testing.T) {
	expectInt(t, "let f = () int { return 42 }\nf()", 42)
}

func TestEvalFuncReturnNone(t *testing.T) {
	expectNone(t, "let f = () none { }\nf()")
}

func TestEvalFuncRecursion(t *testing.T) {
	source := `let factorial = (n int) int {
    if n <= 1 { return 1 }
    return n * factorial(n - 1)
}
factorial(5)`
	expectInt(t, source, 120)
}

func TestEvalFuncClosure(t *testing.T) {
	source := `let make_counter = (start int) int {
    let count = start
    let inner = () int {
        count = count + 1
        return count
    }
    return inner()
}
make_counter(10)`
	expectInt(t, source, 11)
}

func TestEvalFuncClosureCopies(t *testing.T) {
	// Closure captures by copy — outer scope unchanged
	source := `let x = 0
let f = () int {
    x = x + 1
    return x
}
f()
x`
	expectInt(t, source, 0) // x unchanged because closure captured a copy
}

func TestEvalFuncHigherOrder(t *testing.T) {
	// TODO: full function type signatures (int, int) -> int not yet parsed
	// For now, test higher-order by passing function directly
	source := `let mul = (a int, b int) int { return a * b }
let double = (x int) int { return mul(x, 2) }
double(6)`
	expectInt(t, source, 12)
}

func TestEvalFuncArgsCopy(t *testing.T) {
	// Function arguments are copies — modifying inside doesn't affect caller
	source := `let modify = (arr int[]) none {
    arr[0] = 99
}
let data = [1, 2, 3]
modify(data)
data[0]`
	expectInt(t, source, 1) // unchanged
}

// === SCOPE ===

func TestEvalBlockScopeViaIf(t *testing.T) {
	source := `let x = 1
if true {
    let y = 2
    x = x + y
}
x`
	expectInt(t, source, 3)
}

func TestEvalBlockScopeLeakError(t *testing.T) {
	runError(t, "if true { let x = 1 }\nx") // x not accessible outside block
}

func TestEvalShadowing(t *testing.T) {
	source := `let x = 1
if true {
    let x = 99
}
x`
	expectInt(t, source, 1) // outer x unchanged
}

// === GUARD / AGAINST / THROW ===

func TestEvalThrowCaught(t *testing.T) {
	source := `let divide = (a int, b int) int {
    if b == 0 { throw "division by zero" }
    return a / b
}
guard result = divide(10, 0) against error {
    result = -1
}
result`
	expectInt(t, source, -1)
}

func TestEvalThrowSuccess(t *testing.T) {
	source := `let divide = (a int, b int) int {
    if b == 0 { throw "division by zero" }
    return a / b
}
guard result = divide(10, 2) against error {
    result = -1
}
result`
	expectInt(t, source, 5)
}

func TestEvalThrowUnhandled(t *testing.T) {
	runError(t, `throw "crash"`)
}

func TestEvalGuardNoAssignDefaultsNone(t *testing.T) {
	source := `let fail = () int { throw "oops" }
guard result = fail() against error {
}
result`
	expectNone(t, source)
}

func TestEvalThrowPropagates(t *testing.T) {
	source := `let inner = () int { throw "deep error" }
let outer = () int { return inner() }
guard result = outer() against error {
    result = -1
}
result`
	expectInt(t, source, -1)
}

// === NONE ARITHMETIC ERRORS ===

func TestEvalNonePlusInt(t *testing.T)  { runError(t, "none + 1") }
func TestEvalNoneMulInt(t *testing.T)   { runError(t, "none * 2") }
func TestEvalIntMinusNone(t *testing.T) { runError(t, "1 - none") }

// === STRING INDEXING ===

func TestEvalStringIndex(t *testing.T) {
	expectString(t, `"hello"[0]`, "h")
}

func TestEvalStringIndexLast(t *testing.T) {
	expectString(t, `"hello"[4]`, "o")
}

func TestEvalStringIndexOutOfBounds(t *testing.T) {
	expectNone(t, `"hello"[10]`)
}

// === EDGE CASES ===

func TestEvalEmptyProgram(t *testing.T) { expectNone(t, "") }

func TestEvalMultipleExpressions(t *testing.T) {
	// Last expression's value is the result
	expectInt(t, "1\n2\n3", 3)
}

func TestEvalNestedFuncCalls(t *testing.T) {
	source := `let double = (x int) int { return x * 2 }
let inc = (x int) int { return x + 1 }
double(inc(3))`
	expectInt(t, source, 8)
}

func TestEvalReturnFromNested(t *testing.T) {
	source := `let f = () int {
    if true {
        return 42
    }
    return 0
}
f()`
	expectInt(t, source, 42)
}
