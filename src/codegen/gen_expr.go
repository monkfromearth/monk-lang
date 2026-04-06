package codegen

import (
	"fmt"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Expression emission — BOXED path.
//
// Each function returns a C expression string that evaluates to a MonkValue.
// When the type checker told us a scalar storage is safe, the caller should
// use the typed variants in unbox.go (emitExprTyped) instead.

func (g *generator) emitExpr(expr syntax.Expr) string {
	switch e := expr.(type) {
	case *syntax.NumberExpr:
		if e.IsInt {
			return fmt.Sprintf("monk_int(%s)", e.Value)
		}
		return fmt.Sprintf("monk_float(%s)", e.Value)

	case *syntax.StringExpr:
		return fmt.Sprintf("monk_string(%s)", cString(e.Value))

	case *syntax.TemplateExpr:
		return fmt.Sprintf("monk_string(%s)", cString(e.Value))

	case *syntax.BoolExpr:
		if e.Value {
			return "monk_bool(true)"
		}
		return "monk_bool(false)"

	case *syntax.NoneExpr:
		return "monk_none()"

	case *syntax.IdentExpr:
		name := mangleName(e.Name)
		// If this variable is stored as a raw scalar, box it up so the
		// classic emit-path (which assumes MonkValue) stays correct.
		if store := g.varStorage(name); store != storeBoxed {
			return boxExpr(name, store)
		}
		return name

	case *syntax.UnaryExpr:
		operand := g.emitExpr(e.Operand)
		switch e.Op {
		case syntax.Minus:
			return fmt.Sprintf("monk_neg(%s)", operand)
		case syntax.Not, syntax.Bang:
			return fmt.Sprintf("monk_bool(!monk_is_truthy(%s))", operand)
		case syntax.Tilde:
			return fmt.Sprintf("monk_int(~(%s).int_val)", operand)
		}

	case *syntax.BinaryExpr:
		return g.emitBinary(e)

	case *syntax.CallExpr:
		return g.emitCall(e)

	case *syntax.IndexExpr:
		// Fix 7: evaluate object once to avoid double evaluation
		obj := g.emitExpr(e.Object)
		idx := g.emitExpr(e.Index)
		tmpObj := g.newTemp()
		tmpIdx := g.newTemp()
		return fmt.Sprintf("({MonkValue %s=%s; MonkValue %s=%s; "+
			"%s.kind==MONK_STRING ? monk_string_index(%s,%s) : monk_array_get(%s,%s);})",
			tmpObj, obj, tmpIdx, idx,
			tmpObj, tmpObj, tmpIdx, tmpObj, tmpIdx)

	case *syntax.PropertyExpr:
		obj := g.emitExpr(e.Object)
		return fmt.Sprintf("monk_record_get(%s, \"%s\")", obj, e.Property)

	case *syntax.ArrayExpr:
		if len(e.Elements) == 0 {
			return "monk_array(NULL, 0)"
		}
		elems := make([]string, len(e.Elements))
		for i, elem := range e.Elements {
			elems[i] = g.emitExpr(elem)
		}
		return fmt.Sprintf("monk_array((MonkValue[]){%s}, %d)", strings.Join(elems, ", "), len(elems))

	case *syntax.RecordExpr:
		if len(e.Fields) == 0 {
			return "monk_record(NULL, 0)"
		}
		fields := make([]string, len(e.Fields))
		for i, f := range e.Fields {
			fields[i] = fmt.Sprintf("{.key = %s, .value = %s}", cString(f.Key), g.emitExpr(f.Value))
		}
		return fmt.Sprintf("monk_record((MonkRecordField[]){%s}, %d)", strings.Join(fields, ", "), len(fields))

	case *syntax.FuncExpr:
		return g.emitFuncExpr(e)

	case *syntax.ThrowExpr:
		val := g.emitExpr(e.Value)
		return fmt.Sprintf("(monk_throw(%s), monk_none())", val)
	}

	return "monk_none() /* unhandled expr */"
}

func (g *generator) emitBinary(e *syntax.BinaryExpr) string {
	left := g.emitExpr(e.Left)
	right := g.emitExpr(e.Right)

	switch e.Op {
	// Arithmetic
	case syntax.Plus:
		// Fix 2: evaluate left once to avoid double evaluation
		tmp := g.newTemp()
		return fmt.Sprintf("({MonkValue %s=%s; %s.kind==MONK_STRING ? monk_string_concat(%s,%s) : monk_add(%s,%s);})",
			tmp, left, tmp, tmp, right, tmp, right)
	case syntax.Minus:
		return fmt.Sprintf("monk_sub(%s, %s)", left, right)
	case syntax.Star:
		return fmt.Sprintf("monk_mul(%s, %s)", left, right)
	case syntax.Slash:
		return fmt.Sprintf("monk_div(%s, %s)", left, right)
	case syntax.Percent:
		return fmt.Sprintf("monk_mod(%s, %s)", left, right)

	// Comparison
	case syntax.EqualEqual, syntax.Is:
		return fmt.Sprintf("monk_equal(%s, %s)", left, right)
	case syntax.BangEqual:
		return fmt.Sprintf("monk_not_equal(%s, %s)", left, right)
	case syntax.Less:
		return fmt.Sprintf("monk_less(%s, %s)", left, right)
	case syntax.Greater:
		return fmt.Sprintf("monk_greater(%s, %s)", left, right)
	case syntax.LessEqual:
		return fmt.Sprintf("monk_less_equal(%s, %s)", left, right)
	case syntax.GreaterEqual:
		return fmt.Sprintf("monk_greater_equal(%s, %s)", left, right)

	// Logical (short-circuit via ternary)
	case syntax.And, syntax.AmpAmp:
		return fmt.Sprintf("(monk_is_truthy(%s) ? monk_bool(monk_is_truthy(%s)) : monk_bool(false))", left, right)
	case syntax.Or, syntax.PipePipe:
		return fmt.Sprintf("(monk_is_truthy(%s) ? monk_bool(true) : monk_bool(monk_is_truthy(%s)))", left, right)

	// Bitwise — direct C operators on int values
	case syntax.Amp:
		return fmt.Sprintf("monk_int((%s).int_val & (%s).int_val)", left, right)
	case syntax.Pipe:
		return fmt.Sprintf("monk_int((%s).int_val | (%s).int_val)", left, right)
	case syntax.Caret:
		return fmt.Sprintf("monk_int((%s).int_val ^ (%s).int_val)", left, right)
	case syntax.ShiftLeft:
		return fmt.Sprintf("monk_int((%s).int_val << (%s).int_val)", left, right)
	case syntax.ShiftRight:
		return fmt.Sprintf("monk_int((%s).int_val >> (%s).int_val)", left, right)
	}

	return fmt.Sprintf("monk_none() /* unhandled op %s */", e.Op)
}

func (g *generator) emitCall(e *syntax.CallExpr) string {
	// If the callee is an unboxed user function, call it with raw args and
	// box the return. A typed call site (emitCallTyped) stays unboxed, but
	// emitExpr always returns MonkValue for compatibility with the classic
	// emission paths.
	if ident, ok := e.Callee.(*syntax.IdentExpr); ok {
		if cName, ok := g.funcNames[ident.Name]; ok {
			if fs, unboxed := g.fnStorage[cName]; unboxed && fs.All {
				rawCall := g.emitUnboxedCall(cName, fs, e.Args)
				return boxExpr(rawCall, fs.Return)
			}
		}
	}

	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = g.emitExpr(arg)
	}
	argStr := strings.Join(args, ", ")

	if ident, ok := e.Callee.(*syntax.IdentExpr); ok {
		// Known builtin → direct C runtime call
		if fn, ok := builtinMap[ident.Name]; ok {
			return fmt.Sprintf("%s(%s)", fn, argStr)
		}

		// User-defined function → call the hoisted C function
		if cName, ok := g.funcNames[ident.Name]; ok {
			return fmt.Sprintf("%s(%s)", cName, argStr)
		}

		// Unknown — forward reference or passed-in function
		return fmt.Sprintf("%s(%s)", mangleName(ident.Name), argStr)
	}

	return "monk_none() /* indirect call TODO */"
}

// emitUnboxedCall emits a call to a fully-unboxed user function, coercing
// each argument to the parameter's expected storage. Returns the raw call
// expression; the caller decides whether to box it.
func (g *generator) emitUnboxedCall(cName string, fs funcStorage, argExprs []syntax.Expr) string {
	args := make([]string, len(argExprs))
	for i, a := range argExprs {
		code, kind := g.emitExprTyped(a)
		args[i] = coerce(code, kind, fs.Params[i])
	}
	return fmt.Sprintf("%s(%s)", cName, strings.Join(args, ", "))
}
