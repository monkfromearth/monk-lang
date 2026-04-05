// Package types — expression-level type inference.
package types

import (
	"github.com/monkfromearth/monk-lang/syntax"
)

// inferExpr returns the type of an expression, or an error.
func (c *checker) inferExpr(e syntax.Expr) (*Type, error) {
	switch expr := e.(type) {
	case *syntax.NumberExpr:
		if expr.IsInt {
			return Int, nil
		}
		return Float, nil
	case *syntax.StringExpr:
		return Str, nil
	case *syntax.TemplateExpr:
		return Str, nil
	case *syntax.BoolExpr:
		return Bool, nil
	case *syntax.NoneExpr:
		return None, nil
	case *syntax.IdentExpr:
		return c.inferIdent(expr)
	case *syntax.UnaryExpr:
		return c.inferUnary(expr)
	case *syntax.BinaryExpr:
		return c.inferBinary(expr)
	case *syntax.CallExpr:
		return c.inferCall(expr)
	case *syntax.IndexExpr:
		return c.inferIndex(expr)
	case *syntax.PropertyExpr:
		return c.inferProperty(expr)
	case *syntax.ArrayExpr:
		return c.inferArray(expr)
	case *syntax.RecordExpr:
		return c.inferRecord(expr)
	case *syntax.FuncExpr:
		return c.inferFunc(expr)
	case *syntax.ThrowExpr:
		// throw never returns normally, but the expression type has to be SOMETHING
		// so code using it flows. Any is the honest answer — the throw doesn't
		// produce a value.
		if _, err := c.inferExpr(expr.Value); err != nil {
			return nil, err
		}
		return Any, nil
	}
	return Any, nil
}

func (c *checker) inferIdent(e *syntax.IdentExpr) (*Type, error) {
	b := c.scope.lookup(e.Name)
	if b == nil {
		return nil, newTypeError(e.Pos, "undefined variable '%s'", e.Name)
	}
	return b.Type, nil
}

func (c *checker) inferUnary(e *syntax.UnaryExpr) (*Type, error) {
	t, err := c.inferExpr(e.Operand)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case syntax.Minus:
		if !isNumericOrAny(t) {
			return nil, newTypeError(e.Pos, "unary - expects numeric, got %s", t)
		}
		return t, nil
	case syntax.Not, syntax.Bang:
		// Truthiness applies to any type.
		return Bool, nil
	case syntax.Tilde:
		if t.Kind != KindInt && t.Kind != KindAny {
			return nil, newTypeError(e.Pos, "~ expects int, got %s", t)
		}
		return Int, nil
	}
	return Any, nil
}

func (c *checker) inferBinary(e *syntax.BinaryExpr) (*Type, error) {
	lt, err := c.inferExpr(e.Left)
	if err != nil {
		return nil, err
	}
	rt, err := c.inferExpr(e.Right)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case syntax.Plus, syntax.Minus, syntax.Star, syntax.Slash, syntax.Percent:
		return c.inferArith(e, lt, rt)
	case syntax.EqualEqual, syntax.BangEqual:
		return c.inferEquality(e, lt, rt)
	case syntax.Less, syntax.LessEqual, syntax.Greater, syntax.GreaterEqual:
		if !isNumericOrAny(lt) || !isNumericOrAny(rt) {
			// Strings are also comparable per spec.
			if lt.Kind != KindStr || rt.Kind != KindStr {
				return nil, newTypeError(e.Pos,
					"cannot compare %s and %s with ordering operators", lt, rt)
			}
		}
		return Bool, nil
	case syntax.And, syntax.Or:
		return Bool, nil
	case syntax.Is:
		return Bool, nil
	case syntax.Amp, syntax.Pipe, syntax.Caret, syntax.ShiftLeft, syntax.ShiftRight:
		if !isIntOrAny(lt) || !isIntOrAny(rt) {
			return nil, newTypeError(e.Pos,
				"bitwise operator requires int operands, got %s and %s", lt, rt)
		}
		return Int, nil
	}
	return Any, nil
}

// inferArith handles + - * / %. Per spec:
//   - `+` accepts string+string (concat) OR numeric+numeric. Mixed is an error.
//   - `- * / %` are numeric only.
//   - int op int → int (except / with floats anywhere → float)
//   - int op float or float op float → float
func (c *checker) inferArith(e *syntax.BinaryExpr, lt, rt *Type) (*Type, error) {
	// Plus has a string-concat overload. Handle it before the numeric check.
	if e.Op == syntax.Plus {
		if lt.Kind == KindStr && rt.Kind == KindStr {
			return Str, nil
		}
		if lt.Kind == KindAny || rt.Kind == KindAny {
			// One side unknown — we can't tell concat from addition. Allow and
			// let the runtime figure it out. Codegen already branches on kind.
			return Any, nil
		}
		if lt.Kind == KindStr || rt.Kind == KindStr {
			return nil, newTypeError(e.Pos,
				"operator + cannot mix string and %s — use to_string() to concatenate",
				otherOp(lt, rt))
		}
	}
	if !isNumericOrAny(lt) || !isNumericOrAny(rt) {
		return nil, newTypeError(e.Pos,
			"operator %s requires numeric operands, got %s and %s",
			tokenOpString(e.Op), lt, rt)
	}
	if lt.Kind == KindAny || rt.Kind == KindAny {
		return Any, nil
	}
	if lt.Kind == KindFloat || rt.Kind == KindFloat {
		return Float, nil
	}
	return Int, nil
}

// inferEquality handles == and !=. Per spec (Cross-Type Operations):
//   - Comparing different types with == is a type error (except none checks)
//   - Arrays/records/functions cannot be compared with == — use is_none or
//     a deep-compare builtin
//   - int and float may be compared (widening)
//   - Comparing to 'none' is always allowed (and yields bool)
func (c *checker) inferEquality(e *syntax.BinaryExpr, lt, rt *Type) (*Type, error) {
	// Any side Any → allow; runtime will decide.
	if lt.Kind == KindAny || rt.Kind == KindAny {
		return Bool, nil
	}
	// Either side is none: allowed.
	if lt.Kind == KindNone || rt.Kind == KindNone {
		return Bool, nil
	}
	// Optional on either side comparing to its base type: allowed.
	if lt.Optional {
		stripped := *lt
		stripped.Optional = false
		if Equal(&stripped, rt) {
			return Bool, nil
		}
	}
	if rt.Optional {
		stripped := *rt
		stripped.Optional = false
		if Equal(lt, &stripped) {
			return Bool, nil
		}
	}
	// Collections and functions cannot be compared.
	if lt.Kind == KindArray || lt.Kind == KindRecord || lt.Kind == KindFunc {
		return nil, newTypeError(e.Pos,
			"cannot compare %s with == (collections and functions have no equality)", lt)
	}
	// Numeric mixing (int/float) is allowed.
	if isNumericOrAny(lt) && isNumericOrAny(rt) {
		return Bool, nil
	}
	// Same primitive kind.
	if lt.Kind == rt.Kind {
		return Bool, nil
	}
	return nil, newTypeError(e.Pos,
		"cannot compare %s and %s (no implicit cross-type comparison)", lt, rt)
}

// otherOp returns the non-string side's type name, for error messages.
func otherOp(l, r *Type) string {
	if l.Kind == KindStr {
		return r.String()
	}
	return l.String()
}

func (c *checker) inferCall(e *syntax.CallExpr) (*Type, error) {
	calleeType, err := c.inferExpr(e.Callee)
	if err != nil {
		return nil, err
	}
	if calleeType.Kind == KindAny {
		// We genuinely don't know the signature — check args only for validity.
		for _, a := range e.Args {
			if _, err := c.inferExpr(a); err != nil {
				return nil, err
			}
		}
		return Any, nil
	}
	if calleeType.Kind != KindFunc {
		return nil, newTypeError(e.Pos, "cannot call non-function of type %s", calleeType)
	}
	if len(e.Args) != len(calleeType.Params) {
		return nil, newTypeError(e.Pos,
			"wrong number of arguments: expected %d, got %d",
			len(calleeType.Params), len(e.Args))
	}
	for i, a := range e.Args {
		at, err := c.inferExpr(a)
		if err != nil {
			return nil, err
		}
		if !AssignableTo(at, calleeType.Params[i]) {
			return nil, newTypeError(e.Pos,
				"argument %d: cannot pass %s to parameter of type %s",
				i+1, at, calleeType.Params[i])
		}
	}
	return calleeType.Return, nil
}

func (c *checker) inferIndex(e *syntax.IndexExpr) (*Type, error) {
	objType, err := c.inferExpr(e.Object)
	if err != nil {
		return nil, err
	}
	idxType, err := c.inferExpr(e.Index)
	if err != nil {
		return nil, err
	}
	if !isIntOrAny(idxType) {
		return nil, newTypeError(e.Pos, "index must be int, got %s", idxType)
	}
	switch objType.Kind {
	case KindArray:
		// Graceful on reads: element or none.
		return OptionalOf(objType.Elem), nil
	case KindStr:
		return OptionalOf(Str), nil
	case KindAny:
		return Any, nil
	}
	return nil, newTypeError(e.Pos, "cannot index %s", objType)
}

func (c *checker) inferProperty(e *syntax.PropertyExpr) (*Type, error) {
	objType, err := c.inferExpr(e.Object)
	if err != nil {
		return nil, err
	}
	switch objType.Kind {
	case KindRecord:
		for i := range objType.Fields {
			if objType.Fields[i].Name == e.Property {
				return objType.Fields[i].Type, nil
			}
		}
		// Typed record: unknown field is a compile error.
		// Untyped record: return none (graceful on reads).
		if objType.RecordName != "" {
			return nil, newTypeError(e.Pos,
				"%s has no field '%s'", objType, e.Property)
		}
		return None, nil
	case KindAny:
		return Any, nil
	}
	return nil, newTypeError(e.Pos,
		"cannot access property '%s' on %s", e.Property, objType)
}

func (c *checker) inferArray(e *syntax.ArrayExpr) (*Type, error) {
	if len(e.Elements) == 0 {
		// Empty array — element type unknown, will be inferred from context
		// (typed declaration or first use).
		return ArrayOf(Any), nil
	}
	first, err := c.inferExpr(e.Elements[0])
	if err != nil {
		return nil, err
	}
	elemType := first
	for i := 1; i < len(e.Elements); i++ {
		et, err := c.inferExpr(e.Elements[i])
		if err != nil {
			return nil, err
		}
		// Widen int→float if mixing (so [1, 2.0] becomes float[]).
		if elemType.Kind == KindInt && et.Kind == KindFloat {
			elemType = Float
			continue
		}
		if elemType.Kind == KindFloat && et.Kind == KindInt {
			continue
		}
		if !Equal(elemType, et) {
			return nil, newTypeError(e.Pos,
				"array elements must be same type: %s and %s", elemType, et)
		}
	}
	return ArrayOf(elemType), nil
}

func (c *checker) inferRecord(e *syntax.RecordExpr) (*Type, error) {
	fields := make([]RecordTypeField, len(e.Fields))
	seen := make(map[string]bool, len(e.Fields))
	for i, f := range e.Fields {
		if seen[f.Key] {
			return nil, newTypeError(e.Pos, "duplicate field '%s'", f.Key)
		}
		seen[f.Key] = true
		ft, err := c.inferExpr(f.Value)
		if err != nil {
			return nil, err
		}
		fields[i] = RecordTypeField{Name: f.Key, Type: ft}
	}
	return &Type{Kind: KindRecord, Fields: fields}, nil
}

func (c *checker) inferFunc(fn *syntax.FuncExpr) (*Type, error) {
	sig, err := c.funcSignature(fn)
	if err != nil {
		return nil, err
	}
	// Check the body in a fresh scope with params bound.
	savedScope := c.scope
	savedReturn := c.returnType
	c.scope = newScopeOf(c.scope)
	c.returnType = sig.Return
	for i, p := range fn.Params {
		c.scope.declare(p.Name, sig.Params[i], false)
	}
	// Function bodies can reference themselves by outer name (handled in
	// checkProgram's hoist pass). Nested/lambda funcs have no self-ref.
	for _, stmt := range fn.Body.Stmts {
		if err := c.checkStmt(stmt); err != nil {
			c.scope = savedScope
			c.returnType = savedReturn
			return nil, err
		}
	}
	c.scope = savedScope
	c.returnType = savedReturn
	// All-paths-return check: functions declared to return a non-none,
	// non-any type must return on every execution path.
	if sig.Return.Kind != KindNone && sig.Return.Kind != KindAny &&
		!stmtsAlwaysReturn(fn.Body.Stmts) {
		return nil, newTypeError(fn.Pos,
			"function may exit without returning %s", sig.Return)
	}
	return sig, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func isNumericOrAny(t *Type) bool {
	return t.Kind == KindInt || t.Kind == KindFloat || t.Kind == KindAny
}

func isIntOrAny(t *Type) bool {
	return t.Kind == KindInt || t.Kind == KindAny
}

// tokenOpString returns "+" / "-" etc. for binary operators. Only used in
// error messages, so unknown tokens fall through to "<op>".
func tokenOpString(k syntax.TokenKind) string {
	switch k {
	case syntax.Plus:
		return "+"
	case syntax.Minus:
		return "-"
	case syntax.Star:
		return "*"
	case syntax.Slash:
		return "/"
	case syntax.Percent:
		return "%"
	}
	return "<op>"
}
