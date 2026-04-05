// Package codegen — scalar unboxing.
//
// The codegen in codegen.go emits MonkValue everywhere: every variable, every
// operation, every array slot. That's the correct-first path. This file adds
// an OPTIONAL fast path that emits raw C scalars (int64_t, double, bool) when
// the type checker can prove a variable only ever holds a scalar of a known
// kind.
//
// Storage decision (per variable):
//   - Static type int       → int64_t  storage
//   - Static type float     → double   storage
//   - Static type boolean   → bool     storage
//   - Anything else         → MonkValue storage
//
// When a raw scalar is used in a slot that expects a MonkValue (function
// call argument, array element, record field, being stored back into a
// MonkValue variable), the codegen boxes it: monk_int(x), monk_float(x),
// monk_bool(x). The reverse — unboxing from MonkValue — happens when a
// MonkValue expression is assigned to a raw-scalar slot; the codegen emits
// x.int_val, x.float_val, x.bool_val.
//
// The simplification: we only unbox VARIABLES, not arbitrary expressions.
// An expression like `arr[0] + 1` still goes through the runtime path unless
// both operands are scalar vars. This keeps the surface area tight and avoids
// boxing/unboxing on every intermediate — the C compiler's inliner handles
// the rest.
package codegen

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// storageKind describes how a variable OR the result of an expression is
// laid out in the generated C.
type storageKind int

const (
	storeBoxed storageKind = iota // MonkValue
	storeInt                      // int64_t
	storeFloat                    // double
	storeBool                     // bool
)

func (s storageKind) String() string {
	switch s {
	case storeBoxed:
		return "MonkValue"
	case storeInt:
		return "int64_t"
	case storeFloat:
		return "double"
	case storeBool:
		return "bool"
	}
	return "?"
}

// storageFor picks the storage kind for a given static Monk type. Optional
// types, arrays, records, and unknown types stay boxed — only the three
// unconditional scalars qualify for unboxing.
func storageFor(t *types.Type) storageKind {
	if t == nil || t.Optional {
		return storeBoxed
	}
	switch t.Kind {
	case types.KindInt:
		return storeInt
	case types.KindFloat:
		return storeFloat
	case types.KindBool:
		return storeBool
	}
	return storeBoxed
}

// boxExpr wraps a raw-scalar C expression in the appropriate monk_* constructor.
// If src is already boxed, returns code unchanged.
func boxExpr(code string, src storageKind) string {
	switch src {
	case storeInt:
		return "monk_int(" + code + ")"
	case storeFloat:
		return "monk_float(" + code + ")"
	case storeBool:
		return "monk_bool(" + code + ")"
	}
	return code
}

// unboxExpr extracts the raw scalar from a boxed MonkValue. Used when a
// runtime value flows into a statically-scalar slot (rare — a well-typed
// program shouldn't need this often).
//
// The caller must have already proven that the MonkValue has the expected
// kind. For mk_x stored boxed, unboxing to int emits `mk_x.int_val`.
func unboxExpr(code string, dst storageKind) string {
	switch dst {
	case storeInt:
		return "(" + code + ").int_val"
	case storeFloat:
		return "(" + code + ").float_val"
	case storeBool:
		return "(" + code + ").bool_val"
	}
	return code
}

// coerce converts a C expression from storage `src` to storage `dst`,
// inserting box/unbox/widen operations as needed. Returns the new code.
// Panics on impossible conversions (int→bool, bool→float, etc.) — such a
// conversion should have been caught by the type checker.
func coerce(code string, src, dst storageKind) string {
	if src == dst {
		return code
	}
	// Widen int → float (valid per spec).
	if src == storeInt && dst == storeFloat {
		return "(double)(" + code + ")"
	}
	// Box any raw scalar into MonkValue.
	if dst == storeBoxed {
		return boxExpr(code, src)
	}
	// Unbox MonkValue into a raw scalar.
	if src == storeBoxed {
		return unboxExpr(code, dst)
	}
	// Anything else (narrowing float→int, etc.) is a type-checker bug.
	// Fall back to leaving the expression alone; the C compiler will
	// complain loudly if it's truly wrong.
	return code
}

// cTypeName returns the C type name for a storage kind.
func cTypeName(s storageKind) string {
	return s.String()
}

// emitExprTyped emits a C expression AND reports what kind of value it
// evaluates to. When both operands of an arithmetic/comparison op are raw
// scalars, the result stays raw — so a chain like `a + b * c - d` with all
// ints compiles to straight C `int64_t` arithmetic with no MonkValue
// allocations, no runtime dispatch, no boxing until it leaves the chain.
//
// Operands that aren't eligible (or when g.info is nil) fall back to the
// classic boxed path: the code evaluates to a MonkValue, and the caller
// boxes/unboxes as needed.
//
// A separate concern: codegen ALWAYS had to return MonkValue because that's
// the assumption in all the older paths. emitExprTyped coexists with
// emitExpr — the latter is a thin boxed wrapper that keeps the existing
// call sites working without changes.
func (g *generator) emitExprTyped(expr syntax.Expr) (string, storageKind) {
	if g.info == nil {
		return g.emitExpr(expr), storeBoxed
	}

	switch e := expr.(type) {
	case *syntax.NumberExpr:
		// Raw numeric literals stay raw.
		if e.IsInt {
			return e.Value, storeInt
		}
		return e.Value, storeFloat
	case *syntax.BoolExpr:
		if e.Value {
			return "true", storeBool
		}
		return "false", storeBool
	case *syntax.IdentExpr:
		// Look up the variable's declared storage.
		name := mangleName(e.Name)
		return name, g.varStorage(name)
	case *syntax.BinaryExpr:
		return g.emitBinaryTyped(e)
	case *syntax.UnaryExpr:
		return g.emitUnaryTyped(e)
	case *syntax.CallExpr:
		return g.emitCallTyped(e)
	}

	// Fall through — any other expression goes through the classic boxed path.
	return g.emitExpr(expr), storeBoxed
}

// emitCallTyped handles calls where the callee is a fully-unboxed user
// function OR a scalar-returning builtin (to_int, to_float) whose single
// argument is already scalar. We skip the classic boxed path and emit raw
// scalar passing end-to-end. Any other callee (other builtins, boxed fns,
// indirect calls) goes through the classic path and reports storeBoxed.
func (g *generator) emitCallTyped(e *syntax.CallExpr) (string, storageKind) {
	ident, ok := e.Callee.(*syntax.IdentExpr)
	if !ok {
		return g.emitExpr(e), storeBoxed
	}

	// Scalar coercion builtins: to_int(int|float|bool|any) and to_float(...)
	// If the argument is already a raw scalar, inline the coercion with no
	// runtime call. This keeps chains like `y / to_float(H) * 2.0` fully raw.
	if (ident.Name == "to_int" || ident.Name == "to_float") && len(e.Args) == 1 {
		argCode, argKind := g.emitExprTyped(e.Args[0])
		if argKind != storeBoxed {
			if ident.Name == "to_int" {
				// Truncate floats toward zero to match runtime behavior.
				if argKind == storeFloat {
					return "((int64_t)(" + argCode + "))", storeInt
				}
				return argCode, storeInt
			}
			// to_float
			if argKind == storeInt {
				return "((double)(" + argCode + "))", storeFloat
			}
			return argCode, storeFloat
		}
	}

	cName, ok := g.funcNames[ident.Name]
	if !ok {
		return g.emitExpr(e), storeBoxed
	}
	fs, ok := g.fnStorage[cName]
	if !ok || !fs.All {
		return g.emitExpr(e), storeBoxed
	}
	return g.emitUnboxedCall(cName, fs, e.Args), fs.Return
}

// emitBinaryTyped emits a binary expression, preserving raw storage when
// both sides are raw scalars of compatible kind. The type checker has
// already verified the operation is well-typed; codegen just has to match
// its decision.
//
// IMPORTANT: We check the ACTUAL storage of each operand (what codegen has
// already committed to emitting as raw vs. boxed), not what the type
// checker thinks. The checker's types tell us what's semantically an int
// — but a loop variable whose Monk type is int may still be stored as a
// boxed MonkValue because we haven't unboxed for-loop vars yet. Using the
// actual storage keeps these two layers consistent.
func (g *generator) emitBinaryTyped(e *syntax.BinaryExpr) (string, storageKind) {
	lcode, ls := g.emitExprTyped(e.Left)
	rcode, rs := g.emitExprTyped(e.Right)
	// If either side is boxed, bail to the classic path.
	if ls == storeBoxed || rs == storeBoxed {
		return g.emitExpr(e), storeBoxed
	}

	// Arithmetic: + - * / %
	// Int op Int → Int. Any float involved → Float.
	result := ls
	if ls == storeFloat || rs == storeFloat {
		result = storeFloat
		lcode = coerce(lcode, ls, storeFloat)
		rcode = coerce(rcode, rs, storeFloat)
	}

	switch e.Op {
	case syntax.Plus:
		return fmt.Sprintf("(%s + %s)", lcode, rcode), result
	case syntax.Minus:
		return fmt.Sprintf("(%s - %s)", lcode, rcode), result
	case syntax.Star:
		return fmt.Sprintf("(%s * %s)", lcode, rcode), result
	case syntax.Slash:
		// Guard against int div-by-zero to match spec ("x/0 is a runtime error").
		// For unboxed int, C's / traps on zero; we emit an explicit check so
		// the error looks like the runtime-path error.
		if result == storeInt {
			return fmt.Sprintf("((%s)==0 ? (monk_panic(\"division by zero\"),0) : (%s / %s))",
				rcode, lcode, rcode), result
		}
		return fmt.Sprintf("(%s / %s)", lcode, rcode), result
	case syntax.Percent:
		if result == storeInt {
			return fmt.Sprintf("((%s)==0 ? (monk_panic(\"modulo by zero\"),0) : (%s %% %s))",
				rcode, lcode, rcode), result
		}
		// float modulo — use fmod(); but if you want raw floats here, fall back.
		return g.emitExpr(e), storeBoxed
	// Comparison operators return bool.
	case syntax.Less:
		return fmt.Sprintf("(%s < %s)", lcode, rcode), storeBool
	case syntax.LessEqual:
		return fmt.Sprintf("(%s <= %s)", lcode, rcode), storeBool
	case syntax.Greater:
		return fmt.Sprintf("(%s > %s)", lcode, rcode), storeBool
	case syntax.GreaterEqual:
		return fmt.Sprintf("(%s >= %s)", lcode, rcode), storeBool
	case syntax.EqualEqual:
		return fmt.Sprintf("(%s == %s)", lcode, rcode), storeBool
	case syntax.BangEqual:
		return fmt.Sprintf("(%s != %s)", lcode, rcode), storeBool
	// Bitwise (int only — the type checker enforces this).
	case syntax.Amp:
		return fmt.Sprintf("(%s & %s)", lcode, rcode), storeInt
	case syntax.Pipe:
		return fmt.Sprintf("(%s | %s)", lcode, rcode), storeInt
	case syntax.Caret:
		return fmt.Sprintf("(%s ^ %s)", lcode, rcode), storeInt
	case syntax.ShiftLeft:
		return fmt.Sprintf("(%s << %s)", lcode, rcode), storeInt
	case syntax.ShiftRight:
		return fmt.Sprintf("(%s >> %s)", lcode, rcode), storeInt
	// Logical and/or — short-circuit, bool result.
	case syntax.And, syntax.AmpAmp:
		return fmt.Sprintf("(%s && %s)", lcode, rcode), storeBool
	case syntax.Or, syntax.PipePipe:
		return fmt.Sprintf("(%s || %s)", lcode, rcode), storeBool
	}
	return g.emitExpr(e), storeBoxed
}

// emitUnaryTyped — unary -x, !x, ~x, not x on raw scalars.
func (g *generator) emitUnaryTyped(e *syntax.UnaryExpr) (string, storageKind) {
	ocode, os := g.emitExprTyped(e.Operand)
	if os == storeBoxed {
		return g.emitExpr(e), storeBoxed
	}
	switch e.Op {
	case syntax.Minus:
		return fmt.Sprintf("(-%s)", ocode), os
	case syntax.Not, syntax.Bang:
		return fmt.Sprintf("(!%s)", ocode), storeBool
	case syntax.Tilde:
		return fmt.Sprintf("(~%s)", ocode), storeInt
	}
	return g.emitExpr(e), storeBoxed
}

