package builtins

import (
	"math"
	"testing"

	"github.com/monkfromearth/monk-lang/src/interpreter"
)

// --- Helpers ---

func run(t *testing.T, source string) interpreter.Value {
	t.Helper()
	env := interpreter.NewEnvironment()
	Register(env)
	result, err := interpreter.EvalWithEnv(source, env)
	if err != nil {
		t.Fatalf("Eval(%q) error: %v", source, err)
	}
	return result
}

func runError(t *testing.T, source string) {
	t.Helper()
	env := interpreter.NewEnvironment()
	Register(env)
	_, err := interpreter.EvalWithEnv(source, env)
	if err == nil {
		t.Fatalf("Eval(%q) expected error, got none", source)
	}
}

func expectInt(t *testing.T, source string, expected int64) {
	t.Helper()
	v := run(t, source)
	if v.Kind != interpreter.IntValue || v.Int != expected {
		t.Errorf("Eval(%q): expected int %d, got %s", source, expected, v.String())
	}
}

func expectFloat(t *testing.T, source string, expected float64) {
	t.Helper()
	v := run(t, source)
	if v.Kind != interpreter.FloatValue {
		t.Fatalf("Eval(%q): expected float, got %s", source, v.TypeName())
	}
	if math.Abs(v.Float-expected) > 0.0001 {
		t.Errorf("Eval(%q): expected %g, got %g", source, expected, v.Float)
	}
}

func expectString(t *testing.T, source string, expected string) {
	t.Helper()
	v := run(t, source)
	if v.Kind != interpreter.StringValue || v.Str != expected {
		t.Errorf("Eval(%q): expected string %q, got %s", source, expected, v.String())
	}
}

func expectBool(t *testing.T, source string, expected bool) {
	t.Helper()
	v := run(t, source)
	if v.Kind != interpreter.BoolValue || v.Bool != expected {
		t.Errorf("Eval(%q): expected %v, got %s", source, expected, v.String())
	}
}

func expectNone(t *testing.T, source string) {
	t.Helper()
	v := run(t, source)
	if v.Kind != interpreter.NoneValue {
		t.Errorf("Eval(%q): expected none, got %s", source, v.String())
	}
}

// === SHOW ===

func TestShow(t *testing.T) {
	var output []string
	ShowOutput = &output
	defer func() { ShowOutput = nil }()

	run(t, `show(42)`)
	run(t, `show("hello")`)
	run(t, `show(true)`)
	run(t, `show(none)`)
	run(t, `show([1, 2, 3])`)

	expected := []string{"42", "hello", "true", "none", "[1, 2, 3]"}
	for i, exp := range expected {
		if i >= len(output) {
			t.Fatalf("missing output at index %d", i)
		}
		if output[i] != exp {
			t.Errorf("show output %d: expected %q, got %q", i, exp, output[i])
		}
	}
}

// === TO_STRING ===

func TestToString(t *testing.T) {
	expectString(t, `to_string(42)`, "42")
	expectString(t, `to_string(3.14)`, "3.14")
	expectString(t, `to_string(true)`, "true")
	expectString(t, `to_string(none)`, "none")
}

// === TO_INT ===

func TestToInt(t *testing.T)                 { expectInt(t, `to_int("42")`, 42) }
func TestToIntNegative(t *testing.T)         { expectInt(t, `to_int("-5")`, -5) }
func TestToIntHex(t *testing.T)              { expectInt(t, `to_int("0xFF")`, 255) }
func TestToIntFloatStringError(t *testing.T) { runError(t, `to_int("3.14")`) }
func TestToIntNonNumericError(t *testing.T)  { runError(t, `to_int("abc")`) }
func TestToIntWrongTypeError(t *testing.T)   { runError(t, `to_int(42)`) }

// === TO_FLOAT ===

func TestToFloat(t *testing.T)      { expectFloat(t, `to_float("3.14")`, 3.14) }
func TestToFloatInt(t *testing.T)   { expectFloat(t, `to_float("42")`, 42.0) }
func TestToFloatError(t *testing.T) { runError(t, `to_float("abc")`) }

// === MATH ===

func TestAbs(t *testing.T)      { expectInt(t, `abs(-5)`, 5) }
func TestAbsFloat(t *testing.T) { expectFloat(t, `abs(-3.14)`, 3.14) }
func TestAbsPos(t *testing.T)   { expectInt(t, `abs(5)`, 5) }

func TestFloor(t *testing.T)     { expectInt(t, `floor(3.7)`, 3) }
func TestFloorNeg(t *testing.T)  { expectInt(t, `floor(-1.2)`, -2) }
func TestCeil(t *testing.T)      { expectInt(t, `ceil(3.2)`, 4) }
func TestCeilNeg(t *testing.T)   { expectInt(t, `ceil(-1.8)`, -1) }
func TestRound(t *testing.T)     { expectInt(t, `round(3.5)`, 4) }
func TestRoundDown(t *testing.T) { expectInt(t, `round(3.4)`, 3) }

func TestSqrt(t *testing.T)         { expectFloat(t, `sqrt(16)`, 4.0) }
func TestSqrtFloat(t *testing.T)    { expectFloat(t, `sqrt(2.0)`, math.Sqrt(2.0)) }
func TestSqrtNegError(t *testing.T) { runError(t, `sqrt(-1)`) }

func TestPow(t *testing.T)     { expectFloat(t, `pow(2, 10)`, 1024.0) }
func TestPowFrac(t *testing.T) { expectFloat(t, `pow(4, 0.5)`, 2.0) }

func TestLog(t *testing.T)          { expectFloat(t, `log(1)`, 0.0) }
func TestLogNegError(t *testing.T)  { runError(t, `log(-1)`) }
func TestLogZeroError(t *testing.T) { runError(t, `log(0)`) }
func TestLog10(t *testing.T)        { expectFloat(t, `log10(100)`, 2.0) }
func TestExp(t *testing.T)          { expectFloat(t, `exp(0)`, 1.0) }

func TestMin(t *testing.T)      { expectInt(t, `min(3, 7)`, 3) }
func TestMinFloat(t *testing.T) { expectFloat(t, `min(3.0, 7.0)`, 3.0) }
func TestMax(t *testing.T)      { expectInt(t, `max(3, 7)`, 7) }
func TestMaxFloat(t *testing.T) { expectFloat(t, `max(3.0, 7.0)`, 7.0) }

// === TRIGONOMETRY ===

func TestSin(t *testing.T)  { expectFloat(t, `sin(0)`, 0.0) }
func TestCos(t *testing.T)  { expectFloat(t, `cos(0)`, 1.0) }
func TestTan(t *testing.T)  { expectFloat(t, `tan(0)`, 0.0) }
func TestAsin(t *testing.T) { expectFloat(t, `asin(0)`, 0.0) }
func TestAcos(t *testing.T) { expectFloat(t, `acos(1)`, 0.0) }
func TestAtan(t *testing.T) { expectFloat(t, `atan(0)`, 0.0) }

// === LENGTH ===

func TestLengthString(t *testing.T)   { expectInt(t, `length("hello")`, 5) }
func TestLengthEmpty(t *testing.T)    { expectInt(t, `length("")`, 0) }
func TestLengthArray(t *testing.T)    { expectInt(t, `length([1, 2, 3])`, 3) }
func TestLengthRecord(t *testing.T)   { expectInt(t, `length({a: 1, b: 2})`, 2) }
func TestLengthEmptyArr(t *testing.T) { expectInt(t, `length([])`, 0) }

// === STRING FUNCTIONS ===

func TestSubstring(t *testing.T)      { expectString(t, `substring("hello", 1, 4)`, "ell") }
func TestSubstringClamp(t *testing.T) { expectString(t, `substring("hi", 0, 100)`, "hi") }
func TestSubstringEmpty(t *testing.T) { expectString(t, `substring("hi", 5, 10)`, "") }

func TestIndexOf(t *testing.T)         { expectInt(t, `index_of("hello", "ell")`, 1) }
func TestIndexOfNotFound(t *testing.T) { expectInt(t, `index_of("hello", "xyz")`, -1) }

func TestSplit(t *testing.T) {
	v := run(t, `split("a,b,c", ",")`)
	if v.Kind != interpreter.ArrayValue || len(v.Array) != 3 {
		t.Fatalf("expected array of 3, got %s", v.String())
	}
	if v.Array[0].Str != "a" || v.Array[1].Str != "b" || v.Array[2].Str != "c" {
		t.Errorf("unexpected split result: %s", v.String())
	}
}

func TestTrim(t *testing.T)        { expectString(t, `trim("  hello  ")`, "hello") }
func TestToUpperCase(t *testing.T) { expectString(t, `to_upper_case("hello")`, "HELLO") }
func TestToLowerCase(t *testing.T) { expectString(t, `to_lower_case("HELLO")`, "hello") }

// === ARRAY FUNCTIONS ===

func TestAppend(t *testing.T) {
	v := run(t, `append([1, 2], 3)`)
	if len(v.Array) != 3 || v.Array[2].Int != 3 {
		t.Errorf("expected [1,2,3], got %s", v.String())
	}
}

func TestPrepend(t *testing.T) {
	v := run(t, `prepend([2, 3], 1)`)
	if len(v.Array) != 3 || v.Array[0].Int != 1 {
		t.Errorf("expected [1,2,3], got %s", v.String())
	}
}

func TestPop(t *testing.T) {
	v := run(t, `pop([1, 2, 3])`)
	if len(v.Array) != 2 {
		t.Errorf("expected [1,2], got %s", v.String())
	}
}

func TestPopEmpty(t *testing.T) {
	v := run(t, `pop([])`)
	if len(v.Array) != 0 {
		t.Errorf("expected [], got %s", v.String())
	}
}

func TestDrop(t *testing.T) {
	v := run(t, `drop([1, 2, 3])`)
	if len(v.Array) != 2 || v.Array[0].Int != 2 {
		t.Errorf("expected [2,3], got %s", v.String())
	}
}

func TestDropN(t *testing.T) {
	v := run(t, `drop([1, 2, 3, 4], 2)`)
	if len(v.Array) != 2 || v.Array[0].Int != 3 {
		t.Errorf("expected [3,4], got %s", v.String())
	}
}

func TestDropClamp(t *testing.T) {
	v := run(t, `drop([1], 100)`)
	if len(v.Array) != 0 {
		t.Errorf("expected [], got %s", v.String())
	}
}

func TestTakeOne(t *testing.T) {
	v := run(t, `take([1, 2, 3])`)
	if len(v.Array) != 1 || v.Array[0].Int != 1 {
		t.Errorf("expected [1], got %s", v.String())
	}
}

func TestTakeN(t *testing.T) {
	v := run(t, `take([1, 2, 3, 4], 2)`)
	if len(v.Array) != 2 {
		t.Errorf("expected [1,2], got %s", v.String())
	}
}

func TestTakeClamp(t *testing.T) {
	v := run(t, `take([1], 100)`)
	if len(v.Array) != 1 {
		t.Errorf("expected [1], got %s", v.String())
	}
}

func TestSlice(t *testing.T) {
	v := run(t, `slice([10, 20, 30, 40], 1, 3)`)
	if len(v.Array) != 2 || v.Array[0].Int != 20 || v.Array[1].Int != 30 {
		t.Errorf("expected [20,30], got %s", v.String())
	}
}

func TestSliceClamp(t *testing.T) {
	v := run(t, `slice([1, 2, 3], 0, 100)`)
	if len(v.Array) != 3 {
		t.Errorf("expected [1,2,3], got %s", v.String())
	}
}

func TestRange(t *testing.T) {
	v := run(t, `range(5)`)
	if len(v.Array) != 5 || v.Array[0].Int != 0 || v.Array[4].Int != 4 {
		t.Errorf("expected [0,1,2,3,4], got %s", v.String())
	}
}

func TestRangeZero(t *testing.T) {
	v := run(t, `range(0)`)
	if len(v.Array) != 0 {
		t.Errorf("expected [], got %s", v.String())
	}
}

func TestRangeNeg(t *testing.T) {
	v := run(t, `range(-5)`)
	if len(v.Array) != 0 {
		t.Errorf("expected [], got %s", v.String())
	}
}

// === TYPE CHECKING ===

func TestTypeof(t *testing.T) {
	expectString(t, `typeof(42)`, "int")
	expectString(t, `typeof(3.14)`, "float")
	expectString(t, `typeof("hello")`, "string")
	expectString(t, `typeof(true)`, "boolean")
	expectString(t, `typeof(none)`, "none")
	expectString(t, `typeof([1, 2])`, "array")
	expectString(t, `typeof({a: 1})`, "record")
}

func TestIsNumber(t *testing.T) {
	expectBool(t, `is_number(42)`, true)
	expectBool(t, `is_number(3.14)`, true)
	expectBool(t, `is_number("42")`, false)
}

func TestIsString(t *testing.T)  { expectBool(t, `is_string("hello")`, true) }
func TestIsBoolean(t *testing.T) { expectBool(t, `is_boolean(true)`, true) }
func TestIsArray(t *testing.T)   { expectBool(t, `is_array([1, 2])`, true) }
func TestIsRecord(t *testing.T)  { expectBool(t, `is_record({a: 1})`, true) }
func TestIsNone(t *testing.T)    { expectBool(t, `is_none(none)`, true) }

func TestIsNotWrongType(t *testing.T) {
	expectBool(t, `is_string(42)`, false)
	expectBool(t, `is_boolean(0)`, false)
	expectBool(t, `is_array("hello")`, false)
	expectBool(t, `is_none(0)`, false)
}

// === REAL MONK PATTERNS ===

func TestRangeInForLoop(t *testing.T) {
	expectInt(t, `let sum = 0
for i in range(5) { sum += i }
sum`, 10)
}

func TestLengthInLoop(t *testing.T) {
	expectInt(t, `let arr = [10, 20, 30]
let sum = 0
for i in range(length(arr)) { sum += arr[i] }
sum`, 60)
}

func TestToStringConcat(t *testing.T) {
	expectString(t, `"value: " + to_string(42)`, "value: 42")
}

func TestAppendInLoop(t *testing.T) {
	v := run(t, `let arr = []
let i = 0
while i < 3 {
    arr = append(arr, i)
    i += 1
}
arr`)
	if len(v.Array) != 3 {
		t.Errorf("expected 3 elements, got %s", v.String())
	}
}

// === ARGUMENT COUNT ERRORS ===

func TestWrongArgCount(t *testing.T) {
	runError(t, `abs()`)
	runError(t, `abs(1, 2)`)
	runError(t, `length()`)
	runError(t, `min(1)`)
}
