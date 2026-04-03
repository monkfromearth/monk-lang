// Package builtins registers Monk's built-in functions into an interpreter Environment.
//
// Each built-in is a native Go function that receives evaluated arguments
// and returns a Monk value. Builtins follow the same error conventions as
// user code: they return errors for invalid arguments, which the interpreter
// surfaces as runtime errors.
package builtins

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/monkfromearth/monk-lang/src/interpreter"
)

// Register adds all built-in functions to the given environment.
func Register(env *interpreter.Environment) {
	register := func(name string, fn interpreter.BuiltinFunc) {
		env.Define(name, interpreter.Value{
			Kind: interpreter.FuncValue,
			Func: &interpreter.FuncDef{Builtin: fn},
		}, true) // builtins are const
	}

	// Output & conversion
	register("show", monkShow)
	register("to_string", monkToString)
	register("to_int", monkToInt)
	register("to_float", monkToFloat)

	// Math
	register("abs", monkAbs)
	register("floor", monkFloor)
	register("ceil", monkCeil)
	register("round", monkRound)
	register("sqrt", monkSqrt)
	register("pow", monkPow)
	register("log", monkLog)
	register("log10", monkLog10)
	register("exp", monkExp)
	register("min", monkMin)
	register("max", monkMax)

	// Trigonometry
	register("sin", monkSin)
	register("cos", monkCos)
	register("tan", monkTan)
	register("asin", monkAsin)
	register("acos", monkAcos)
	register("atan", monkAtan)

	// String
	register("length", monkLength)
	register("substring", monkSubstring)
	register("index_of", monkIndexOf)
	register("split", monkSplit)
	register("trim", monkTrim)
	register("to_upper_case", monkToUpperCase)
	register("to_lower_case", monkToLowerCase)

	// Array
	register("append", monkAppend)
	register("prepend", monkPrepend)
	register("pop", monkPop)
	register("drop", monkDrop)
	register("take", monkTake)
	register("slice", monkSlice)
	register("map", monkMap)
	register("filter", monkFilter)
	register("reduce", monkReduce)
	register("range", monkRange)

	// Type checking
	register("typeof", monkTypeof)
	register("is_number", monkIsNumber)
	register("is_string", monkIsString)
	register("is_boolean", monkIsBoolean)
	register("is_array", monkIsArray)
	register("is_record", monkIsRecord)
	register("is_function", monkIsFunction)
	register("is_none", monkIsNone)
}

// --- helpers ---

func expectArgs(name string, args []interpreter.Value, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s: expected %d argument(s), got %d", name, n, len(args))
	}
	return nil
}

func expectMinArgs(name string, args []interpreter.Value, n int) error {
	if len(args) < n {
		return fmt.Errorf("%s: expected at least %d argument(s), got %d", name, n, len(args))
	}
	return nil
}

func toGoFloat(v interpreter.Value) (float64, error) {
	switch v.Kind {
	case interpreter.IntValue:
		return float64(v.Int), nil
	case interpreter.FloatValue:
		return v.Float, nil
	default:
		return 0, fmt.Errorf("expected number, got %s", v.TypeName())
	}
}

// --- Output & Conversion ---

// Captured output for testing. If non-nil, show() appends here instead of printing.
var ShowOutput *[]string

func monkShow(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("show", args, 1); err != nil {
		return interpreter.MonkNone, err
	}
	s := args[0].String()
	if ShowOutput != nil {
		*ShowOutput = append(*ShowOutput, s)
	} else {
		fmt.Println(s)
	}
	return interpreter.MonkNone, nil
}

func monkToString(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("to_string", args, 1); err != nil {
		return interpreter.MonkNone, err
	}
	return interpreter.StringVal(args[0].String()), nil
}

func monkToInt(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("to_int", args, 1); err != nil {
		return interpreter.MonkNone, err
	}
	if args[0].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("to_int: expected string, got %s", args[0].TypeName())
	}
	// Design decision: strict. Only pure integer strings accepted.
	// "3.14" is rejected. Use floor(to_float("3.14")) instead.
	n, err := strconv.ParseInt(args[0].Str, 0, 64)
	if err != nil {
		return interpreter.MonkNone, fmt.Errorf("to_int: cannot parse %q as integer", args[0].Str)
	}
	return interpreter.IntVal(n), nil
}

func monkToFloat(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("to_float", args, 1); err != nil {
		return interpreter.MonkNone, err
	}
	if args[0].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("to_float: expected string, got %s", args[0].TypeName())
	}
	f, err := strconv.ParseFloat(args[0].Str, 64)
	if err != nil {
		return interpreter.MonkNone, fmt.Errorf("to_float: cannot parse %q as float", args[0].Str)
	}
	return interpreter.FloatVal(f), nil
}

// --- Math ---

func monkAbs(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("abs", args, 1); err != nil {
		return interpreter.MonkNone, err
	}
	switch args[0].Kind {
	case interpreter.IntValue:
		v := args[0].Int
		if v < 0 { v = -v }
		return interpreter.IntVal(v), nil
	case interpreter.FloatValue:
		return interpreter.FloatVal(math.Abs(args[0].Float)), nil
	default:
		return interpreter.MonkNone, fmt.Errorf("abs: expected number, got %s", args[0].TypeName())
	}
}

func monkFloor(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("floor", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("floor: %w", err) }
	return interpreter.IntVal(int64(math.Floor(f))), nil
}

func monkCeil(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("ceil", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("ceil: %w", err) }
	return interpreter.IntVal(int64(math.Ceil(f))), nil
}

func monkRound(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("round", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("round: %w", err) }
	return interpreter.IntVal(int64(math.Round(f))), nil
}

func monkSqrt(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("sqrt", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("sqrt: %w", err) }
	if f < 0 { return interpreter.MonkNone, fmt.Errorf("sqrt: cannot take square root of negative number") }
	return interpreter.FloatVal(math.Sqrt(f)), nil
}

func monkPow(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("pow", args, 2); err != nil { return interpreter.MonkNone, err }
	base, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("pow: base: %w", err) }
	exp, err := toGoFloat(args[1])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("pow: exponent: %w", err) }
	return interpreter.FloatVal(math.Pow(base, exp)), nil
}

func monkLog(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("log", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("log: %w", err) }
	if f <= 0 { return interpreter.MonkNone, fmt.Errorf("log: argument must be positive") }
	return interpreter.FloatVal(math.Log(f)), nil
}

func monkLog10(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("log10", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("log10: %w", err) }
	if f <= 0 { return interpreter.MonkNone, fmt.Errorf("log10: argument must be positive") }
	return interpreter.FloatVal(math.Log10(f)), nil
}

func monkExp(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("exp", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("exp: %w", err) }
	return interpreter.FloatVal(math.Exp(f)), nil
}

func monkMin(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("min", args, 2); err != nil { return interpreter.MonkNone, err }
	a, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("min: %w", err) }
	b, err := toGoFloat(args[1])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("min: %w", err) }
	if args[0].Kind == interpreter.IntValue && args[1].Kind == interpreter.IntValue {
		if args[0].Int < args[1].Int { return args[0], nil }
		return args[1], nil
	}
	return interpreter.FloatVal(math.Min(a, b)), nil
}

func monkMax(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("max", args, 2); err != nil { return interpreter.MonkNone, err }
	a, err := toGoFloat(args[0])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("max: %w", err) }
	b, err := toGoFloat(args[1])
	if err != nil { return interpreter.MonkNone, fmt.Errorf("max: %w", err) }
	if args[0].Kind == interpreter.IntValue && args[1].Kind == interpreter.IntValue {
		if args[0].Int > args[1].Int { return args[0], nil }
		return args[1], nil
	}
	return interpreter.FloatVal(math.Max(a, b)), nil
}

// --- Trigonometry ---

func monkSin(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("sin", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0]); if err != nil { return interpreter.MonkNone, err }
	return interpreter.FloatVal(math.Sin(f)), nil
}
func monkCos(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("cos", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0]); if err != nil { return interpreter.MonkNone, err }
	return interpreter.FloatVal(math.Cos(f)), nil
}
func monkTan(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("tan", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0]); if err != nil { return interpreter.MonkNone, err }
	return interpreter.FloatVal(math.Tan(f)), nil
}
func monkAsin(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("asin", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0]); if err != nil { return interpreter.MonkNone, err }
	return interpreter.FloatVal(math.Asin(f)), nil
}
func monkAcos(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("acos", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0]); if err != nil { return interpreter.MonkNone, err }
	return interpreter.FloatVal(math.Acos(f)), nil
}
func monkAtan(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("atan", args, 1); err != nil { return interpreter.MonkNone, err }
	f, err := toGoFloat(args[0]); if err != nil { return interpreter.MonkNone, err }
	return interpreter.FloatVal(math.Atan(f)), nil
}

// --- String ---

func monkLength(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("length", args, 1); err != nil { return interpreter.MonkNone, err }
	switch args[0].Kind {
	case interpreter.StringValue:
		// Design decision: counts Unicode scalar values, not bytes.
		// For ASCII this is the same. For multi-byte UTF-8, we count runes.
		return interpreter.IntVal(int64(len([]rune(args[0].Str)))), nil
	case interpreter.ArrayValue:
		return interpreter.IntVal(int64(len(args[0].Array))), nil
	case interpreter.RecordValue:
		return interpreter.IntVal(int64(len(args[0].Record))), nil
	default:
		return interpreter.MonkNone, fmt.Errorf("length: expected string, array, or record, got %s", args[0].TypeName())
	}
}

func monkSubstring(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("substring", args, 3); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.StringValue { return interpreter.MonkNone, fmt.Errorf("substring: expected string") }
	if args[1].Kind != interpreter.IntValue || args[2].Kind != interpreter.IntValue {
		return interpreter.MonkNone, fmt.Errorf("substring: expected int indices")
	}
	runes := []rune(args[0].Str)
	start, end := int(args[1].Int), int(args[2].Int)
	// Design decision: indices clamp (graceful on reads)
	if start < 0 { start = 0 }
	if end > len(runes) { end = len(runes) }
	if start >= end { return interpreter.StringVal(""), nil }
	return interpreter.StringVal(string(runes[start:end])), nil
}

func monkIndexOf(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("index_of", args, 2); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.StringValue || args[1].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("index_of: expected two strings")
	}
	idx := strings.Index(args[0].Str, args[1].Str)
	return interpreter.IntVal(int64(idx)), nil
}

func monkSplit(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("split", args, 2); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.StringValue || args[1].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("split: expected two strings")
	}
	parts := strings.Split(args[0].Str, args[1].Str)
	elems := make([]interpreter.Value, len(parts))
	for i, p := range parts {
		elems[i] = interpreter.StringVal(p)
	}
	return interpreter.ArrayVal(elems), nil
}

func monkTrim(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("trim", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("trim: expected string")
	}
	return interpreter.StringVal(strings.TrimSpace(args[0].Str)), nil
}

func monkToUpperCase(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("to_upper_case", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("to_upper_case: expected string")
	}
	return interpreter.StringVal(strings.ToUpper(args[0].Str)), nil
}

func monkToLowerCase(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("to_lower_case", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.StringValue {
		return interpreter.MonkNone, fmt.Errorf("to_lower_case: expected string")
	}
	return interpreter.StringVal(strings.ToLower(args[0].Str)), nil
}

// --- Array ---

func monkAppend(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("append", args, 2); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue {
		return interpreter.MonkNone, fmt.Errorf("append: expected array as first argument")
	}
	// Returns a NEW array (value semantics)
	newArr := make([]interpreter.Value, len(args[0].Array)+1)
	copy(newArr, args[0].Array)
	newArr[len(newArr)-1] = args[1].DeepCopy()
	return interpreter.ArrayVal(newArr), nil
}

func monkPrepend(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("prepend", args, 2); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue {
		return interpreter.MonkNone, fmt.Errorf("prepend: expected array as first argument")
	}
	newArr := make([]interpreter.Value, len(args[0].Array)+1)
	newArr[0] = args[1].DeepCopy()
	copy(newArr[1:], args[0].Array)
	return interpreter.ArrayVal(newArr), nil
}

func monkPop(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("pop", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue {
		return interpreter.MonkNone, fmt.Errorf("pop: expected array")
	}
	// Design decision: pop([]) returns [] (graceful, not error)
	if len(args[0].Array) == 0 { return interpreter.ArrayVal(nil), nil }
	return interpreter.ArrayVal(args[0].Array[:len(args[0].Array)-1]), nil
}

func monkDrop(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectMinArgs("drop", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue {
		return interpreter.MonkNone, fmt.Errorf("drop: expected array")
	}
	n := int64(1)
	if len(args) >= 2 {
		if args[1].Kind != interpreter.IntValue { return interpreter.MonkNone, fmt.Errorf("drop: n must be int") }
		n = args[1].Int
	}
	arr := args[0].Array
	// Design decision: clamps (graceful)
	if n >= int64(len(arr)) { return interpreter.ArrayVal(nil), nil }
	if n < 0 { n = 0 }
	return interpreter.ArrayVal(arr[n:]), nil
}

func monkTake(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectMinArgs("take", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue {
		return interpreter.MonkNone, fmt.Errorf("take: expected array")
	}
	n := int64(1)
	if len(args) >= 2 {
		if args[1].Kind != interpreter.IntValue { return interpreter.MonkNone, fmt.Errorf("take: n must be int") }
		n = args[1].Int
	}
	arr := args[0].Array
	// Design decision: clamps (graceful)
	if n >= int64(len(arr)) { return interpreter.ArrayVal(arr), nil }
	if n < 0 { n = 0 }
	return interpreter.ArrayVal(arr[:n]), nil
}

func monkSlice(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("slice", args, 3); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue {
		return interpreter.MonkNone, fmt.Errorf("slice: expected array")
	}
	if args[1].Kind != interpreter.IntValue || args[2].Kind != interpreter.IntValue {
		return interpreter.MonkNone, fmt.Errorf("slice: expected int indices")
	}
	arr := args[0].Array
	start, end := int(args[1].Int), int(args[2].Int)
	// Design decision: indices clamp (graceful)
	if start < 0 { start = 0 }
	if end > len(arr) { end = len(arr) }
	if start >= end { return interpreter.ArrayVal(nil), nil }
	return interpreter.ArrayVal(arr[start:end]), nil
}

func monkMap(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("map", args, 2); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue { return interpreter.MonkNone, fmt.Errorf("map: expected array") }
	if args[1].Kind != interpreter.FuncValue { return interpreter.MonkNone, fmt.Errorf("map: expected function") }
	fn := args[1].Func
	result := make([]interpreter.Value, len(args[0].Array))
	for i, elem := range args[0].Array {
		val, err := fn.Builtin([]interpreter.Value{elem})
		if err != nil { return interpreter.MonkNone, err }
		result[i] = val
	}
	return interpreter.ArrayVal(result), nil
}

func monkFilter(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("filter", args, 2); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue { return interpreter.MonkNone, fmt.Errorf("filter: expected array") }
	if args[1].Kind != interpreter.FuncValue { return interpreter.MonkNone, fmt.Errorf("filter: expected function") }
	fn := args[1].Func
	var result []interpreter.Value
	for _, elem := range args[0].Array {
		val, err := fn.Builtin([]interpreter.Value{elem})
		if err != nil { return interpreter.MonkNone, err }
		if val.IsTruthy() {
			result = append(result, elem)
		}
	}
	return interpreter.ArrayVal(result), nil
}

func monkReduce(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("reduce", args, 3); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.ArrayValue { return interpreter.MonkNone, fmt.Errorf("reduce: expected array") }
	if args[1].Kind != interpreter.FuncValue { return interpreter.MonkNone, fmt.Errorf("reduce: expected function") }
	fn := args[1].Func
	// Design decision: reduce([], fn, initial) returns initial
	acc := args[2]
	for _, elem := range args[0].Array {
		val, err := fn.Builtin([]interpreter.Value{acc, elem})
		if err != nil { return interpreter.MonkNone, err }
		acc = val
	}
	return acc, nil
}

func monkRange(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("range", args, 1); err != nil { return interpreter.MonkNone, err }
	if args[0].Kind != interpreter.IntValue {
		return interpreter.MonkNone, fmt.Errorf("range: expected int")
	}
	n := args[0].Int
	// Design decision: range(0) and range(-5) return [] (graceful)
	if n <= 0 { return interpreter.ArrayVal(nil), nil }
	result := make([]interpreter.Value, n)
	for i := int64(0); i < n; i++ {
		result[i] = interpreter.IntVal(i)
	}
	return interpreter.ArrayVal(result), nil
}

// --- Type Checking ---

func monkTypeof(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("typeof", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.StringVal(args[0].TypeName()), nil
}

func monkIsNumber(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_number", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.IntValue || args[0].Kind == interpreter.FloatValue), nil
}
func monkIsString(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_string", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.StringValue), nil
}
func monkIsBoolean(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_boolean", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.BoolValue), nil
}
func monkIsArray(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_array", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.ArrayValue), nil
}
func monkIsRecord(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_record", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.RecordValue), nil
}
func monkIsFunction(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_function", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.FuncValue), nil
}
func monkIsNone(args []interpreter.Value) (interpreter.Value, error) {
	if err := expectArgs("is_none", args, 1); err != nil { return interpreter.MonkNone, err }
	return interpreter.BoolVal(args[0].Kind == interpreter.NoneValue), nil
}
