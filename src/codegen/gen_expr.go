package codegen

import (
	"fmt"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// Expression emission — BOXED path.
//
// Each function returns a C expression string that evaluates to a MonkValue.
// When the type checker told us a scalar storage is safe, the caller should
// use the typed variants in unbox.go (emitExprTyped) instead.

// emitExpr returns a C expression string that evaluates to a MonkValue.
// This is the classic boxed path; for scalar-unboxed variants call emitExprTyped.
func (g *generator) emitExpr(expr syntax.Expr) string {
	switch e := expr.(type) {
	case *syntax.NumberExpr:
		// Strip underscores — Monk allows 1_000_000 but C doesn't.
		lit := strings.ReplaceAll(e.Value, "_", "")
		if e.IsInt {
			return fmt.Sprintf("monk_int(%s)", lit)
		}
		return fmt.Sprintf("monk_float(%s)", lit)

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
		name := g.mangledName(e.Name)
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
		// Fast path: if the object's record type is statically known, use direct
		// index access (obj.record_val->fields[N].value) instead of the runtime
		// monk_record_get strcmp loop. For scalar fields, skip the deep copy too.
		if g.info != nil {
			if objType, ok := g.info.Types[e.Object]; ok && objType != nil {
				idx, fieldSt := recordField(objType, e.Property)
				if idx >= 0 {
					obj := g.emitExpr(e.Object)
					fieldVal := fmt.Sprintf("(%s).record_val->fields[%d].value", obj, idx)
					// Scalars: no heap allocation, deep copy is a no-op — skip it.
					if isRawScalar(fieldSt) {
						return fieldVal
					}
					return fmt.Sprintf("monk_deep_copy(%s)", fieldVal)
				}
			}
		}
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
		// Normalize field order to match the type-checker's Fields slice order.
		// This is required for index-based field access (recordField) to be
		// correct: index N in the type must align with runtime fields[N]. For
		// anonymous records the order already matches; for named records the
		// literal source order may differ from the type declaration order.
		if g.info != nil {
			if recType, ok := g.info.Types[e]; ok && recType != nil && recType.Kind == types.KindRecord {
				valMap := make(map[string]string, len(e.Fields))
				for _, f := range e.Fields {
					valMap[f.Key] = g.emitExpr(f.Value)
				}
				fields := make([]string, 0, len(recType.Fields))
				for _, tf := range recType.Fields {
					if v, ok2 := valMap[tf.Name]; ok2 {
						fields = append(fields, fmt.Sprintf("{.key = %s, .value = %s}", cString(tf.Name), v))
					}
				}
				return fmt.Sprintf("monk_record((MonkRecordField[]){%s}, %d)", strings.Join(fields, ", "), len(fields))
			}
		}
		// Fallback: emit in source order (no type info available).
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

// emitBinary emits a binary expression as a boxed MonkValue. For the typed
// (unboxed) path, callers use emitExprTyped which short-circuits to raw C ops.
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
		// User-defined function — pad defaults, then decide call form.
		if cName, ok := g.funcNames[ident.Name]; ok {
			fullArgs := g.padDefaults(cName, e.Args)
			// Unboxed-all path: call with raw scalars and box the return.
			if fs, unboxed := g.fnStorage[cName]; unboxed && fs.All {
				rawCall := g.emitUnboxedCall(cName, fs, fullArgs)
				return boxExpr(rawCall, fs.Return)
			}
			// Boxed path: emit args as MonkValue, route through monk_call if
			// the function has captures (needs _self for capture state).
			args := make([]string, len(fullArgs))
			for i, arg := range fullArgs {
				args[i] = g.emitExpr(arg)
			}
			if g.funcHasCapture[cName] {
				mn := g.mangledName(ident.Name)
				return fmt.Sprintf("monk_call(%s, %s)",
					mn, monkValArray(args, len(args)))
			}
			return fmt.Sprintf("%s(%s)", cName, strings.Join(args, ", "))
		}
		if code, store, ok := g.emitKnownTypeBuiltin(e); ok {
			if store != storeBoxed {
				return boxExpr(code, store)
			}
			return code
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

		// Function value (parameter or variable) — indirect call via monk_call
		name := g.mangledName(ident.Name)
		return fmt.Sprintf("monk_call(%s, %s)", name, monkValArray(args, len(e.Args)))
	}

	// Indirect call — call through the function value's fn pointer.
	callee := g.emitExpr(e.Callee)
	return fmt.Sprintf("monk_call(%s, %s)", callee, monkValArray(args, len(e.Args)))
}

// padDefaults returns a full argument list, appending default expressions
// for any trailing parameters the caller omitted.
func (g *generator) padDefaults(cName string, args []syntax.Expr) []syntax.Expr {
	defs, ok := g.funcDefaults[cName]
	if !ok || len(args) >= len(defs) {
		return args
	}
	full := make([]syntax.Expr, len(defs))
	copy(full, args)
	for i := len(args); i < len(defs); i++ {
		full[i] = defs[i] // the default expression from the FuncExpr
	}
	return full
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
