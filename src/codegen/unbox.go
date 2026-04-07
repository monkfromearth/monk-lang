// Package codegen — scalar and typed-array unboxing.
//
// The codegen in codegen.go emits MonkValue everywhere: every variable, every
// operation, every array slot. That's the correct-first path. This file adds
// OPTIONAL fast paths that emit raw C types when the type checker can prove a
// variable's kind at compile time.
//
// Storage decision (per variable):
//   - Static type int       → int64_t  storage
//   - Static type float     → double   storage
//   - Static type boolean   → bool     storage
//   - Static type int[]     → MonkValue storage, BUT element accesses inline
//   - Static type float[]   → MonkValue storage, BUT element accesses inline
//   - Static type bool[]    → MonkValue storage, BUT element accesses inline
//   - Anything else         → MonkValue storage
//
// Scalar variables (storeInt/Float/Bool): the C variable IS the raw value.
// Boxing/unboxing wraps/unwraps at MonkValue call boundaries.
//
// Typed-array variables (storeIntArray etc.): the C variable is still
// MonkValue (a pointer to the runtime MonkArray), so boxing is free at
// function call boundaries. But element reads/writes inline directly as
// arr.array_val->data[i].int_val instead of calling monk_array_get/set.
// Out-of-bounds access panics (strict) rather than returning none (graceful),
// consistent with the spec's "operating on invalid data = error" rule.
package codegen

import (
	"fmt"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// storageKind describes how a variable OR the result of an expression is
// laid out in the generated C.
type storageKind int

const (
	storeBoxed      storageKind = iota // MonkValue (generic fallback)
	storeInt                           // int64_t
	storeFloat                         // double
	storeBool                          // bool
	storeIntArray                      // MonkValue (int[]) — element accesses inlined as int64_t
	storeFloatArray                    // MonkValue (float[]) — element accesses inlined as double
	storeBoolArray                     // MonkValue (bool[]) — element accesses inlined as bool
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
	case storeIntArray, storeFloatArray, storeBoolArray:
		// Typed arrays are still MonkValue at the C level; only element
		// accesses are inlined. The kind tag tells codegen HOW to access them.
		return "MonkValue"
	}
	return "?"
}

// isRawScalar reports whether s is a pure C scalar (int64_t / double / bool)
// with no MonkValue wrapper. Used to gate the All-scalar fast path in
// function signatures — arrays are MonkValue even when typed.
func isRawScalar(s storageKind) bool {
	return s == storeInt || s == storeFloat || s == storeBool
}

// isArrayStorage reports whether s is one of the typed-array storage kinds.
func isArrayStorage(s storageKind) bool {
	return s == storeIntArray || s == storeFloatArray || s == storeBoolArray
}

// elemStorageFor returns the element's storage kind for a typed-array storage.
// Returns storeBoxed for non-array or untyped storage (caller should fall back).
func elemStorageFor(s storageKind) storageKind {
	switch s {
	case storeIntArray:
		return storeInt
	case storeFloatArray:
		return storeFloat
	case storeBoolArray:
		return storeBool
	}
	return storeBoxed
}

// arrayConvFunc returns the C runtime function name that converts or deep-copies
// a value into the given typed-array kind (e.g. storeIntArray → "monk_int_array_from").
func arrayConvFunc(s storageKind) string {
	switch s {
	case storeIntArray:
		return "monk_int_array_from"
	case storeFloatArray:
		return "monk_float_array_from"
	case storeBoolArray:
		return "monk_bool_array_from"
	}
	return "monk_deep_copy"
}

// arrayPtrField returns the MonkValue union field name for a typed-array storage
// kind. Used in the backing-store path: arr.{field}->data[i].
func arrayPtrField(s storageKind) string {
	switch s {
	case storeIntArray:
		return "int_array_val"
	case storeFloatArray:
		return "float_array_val"
	case storeBoolArray:
		return "bool_array_val"
	}
	return "array_val"
}

// elemZero returns the C zero literal for a typed-array element kind.
func elemZero(s storageKind) string {
	switch s {
	case storeInt:
		return "(int64_t)0"
	case storeFloat:
		return "(double)0.0"
	case storeBool:
		return "false"
	}
	return "monk_none()"
}

// recordField looks up a field in a typed record and returns its index (into
// the runtime fields[] array) and the corresponding storageKind for the field
// value. Returns (-1, storeBoxed) when the type is unknown, not a record, or
// the field name is not present.
//
// The index is ONLY valid if record literals for this type are emitted with
// fields in type-declaration order (enforced by emitExpr's RecordExpr branch).
func recordField(t *types.Type, name string) (int, storageKind) {
	if t == nil || t.Kind != types.KindRecord {
		return -1, storeBoxed
	}
	for i, f := range t.Fields {
		if f.Name == name {
			return i, storageFor(f.Type)
		}
	}
	return -1, storeBoxed
}

// storageFor picks the storage kind for a given static Monk type.
//   - Non-optional int/float/bool → raw scalar storage
//   - Non-optional int[]/float[]/bool[] → typed-array storage (MonkValue
//     container, but element accesses are inlined)
//   - Optional types, records, functions, untyped arrays → storeBoxed
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
	case types.KindArray:
		// Only unbox element access for homogeneous scalar arrays.
		if t.Elem != nil && !t.Elem.Optional {
			switch t.Elem.Kind {
			case types.KindInt:
				return storeIntArray
			case types.KindFloat:
				return storeFloatArray
			case types.KindBool:
				return storeBoolArray
			}
		}
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
	// Typed-array kinds and storeBoxed both live in MonkValue — no-op.
	srcIsMonkValue := src == storeBoxed || isArrayStorage(src)
	dstIsMonkValue := dst == storeBoxed || isArrayStorage(dst)
	if srcIsMonkValue && dstIsMonkValue {
		return code
	}
	// Widen int → float (valid per spec).
	if src == storeInt && dst == storeFloat {
		return "(double)(" + code + ")"
	}
	// Box any raw scalar into MonkValue.
	if dstIsMonkValue {
		return boxExpr(code, src)
	}
	// Unbox MonkValue into a raw scalar — only valid if src could actually
	// hold the scalar type. Array storage kinds can't be unboxed to scalars.
	if srcIsMonkValue && !isArrayStorage(src) {
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
		// Raw numeric literals stay raw. Strip underscores for C.
		lit := strings.ReplaceAll(e.Value, "_", "")
		if e.IsInt {
			return lit, storeInt
		}
		return lit, storeFloat
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
	case *syntax.IndexExpr:
		return g.emitIndexTyped(e)
	case *syntax.PropertyExpr:
		return g.emitPropertyTyped(e)
	}

	// Fall through — any other expression goes through the classic boxed path.
	return g.emitExpr(expr), storeBoxed
}

// emitIndexTyped handles arr[i] when arr is a typed array (storeIntArray etc.).
// Emits a bounds-checked inline element read returning a raw scalar, using the
// typed backing-store path: arr.int_array_val->data[i] — a direct pointer
// dereference with no MonkValue union overhead.
// Out-of-bounds panics (strict), consistent with "operating on invalid data = error".
//
// Falls back to the classic boxed path when the object's storage is unknown.
func (g *generator) emitIndexTyped(e *syntax.IndexExpr) (string, storageKind) {
	objCode, objKind := g.emitExprTyped(e.Object)
	elemSt := elemStorageFor(objKind)
	if elemSt == storeBoxed {
		return g.emitExpr(e), storeBoxed
	}

	ptrField := arrayPtrField(objKind)
	zero := elemZero(elemSt)
	idxCode, idxKind := g.emitExprTyped(e.Index)
	idxC := coerce(idxCode, idxKind, storeInt)

	// Simple ident: use directly — no temp needed for the object.
	if arrIdent, isIdent := e.Object.(*syntax.IdentExpr); isIdent {
		// Bounds-check elision: if we can statically prove 0 <= idx < length,
		// skip the runtime check and emit a direct data[] access.
		// The index is guaranteed to be a pure arithmetic expression (loop
		// counters + constants) by isBoundedSafe's range analysis, so we can
		// inline it directly — no statement-expression wrapper, no temp.
		// This lets the compiler hoist, CSE, and vectorize freely.
		if g.constVals != nil && g.isBoundedSafe(arrIdent, e.Index) {
			return fmt.Sprintf("%s.%s->data[%s]", objCode, ptrField, idxC), elemSt
		}
		tidx := g.newTemp()
		return fmt.Sprintf(
			"({int64_t %s=%s; (%s<0||%s>=%s.%s->length)?(monk_panic(\"index out of bounds\"),%s):%s.%s->data[%s];})",
			tidx, idxC, tidx, tidx, objCode, ptrField, zero, objCode, ptrField, tidx,
		), elemSt
	}
	// Complex expression: stash in a MonkValue temp to avoid double evaluation.
	tobj := g.newTemp()
	tidx := g.newTemp()
	return fmt.Sprintf(
		"({MonkValue %s=%s; int64_t %s=%s; (%s<0||%s>=%s.%s->length)?(monk_panic(\"index out of bounds\"),%s):%s.%s->data[%s];})",
		tobj, objCode, tidx, idxC, tidx, tidx, tobj, ptrField, zero, tobj, ptrField, tidx,
	), elemSt
}

// emitPropertyTyped handles record.field access when the object's type is
// statically known. For scalar fields (int/float/bool) it extracts the raw C
// value directly from the fields array by index, avoiding the runtime
// monk_record_get strcmp loop entirely. For boxed fields it returns the
// MonkValue via the same index path but with a deep copy (storeBoxed).
//
// Falls back to the classic boxed path when type info is unavailable.
func (g *generator) emitPropertyTyped(e *syntax.PropertyExpr) (string, storageKind) {
	objType, ok := g.info.Types[e.Object]
	if !ok || objType == nil {
		return g.emitExpr(e), storeBoxed
	}
	idx, fieldSt := recordField(objType, e.Property)
	if idx < 0 {
		return g.emitExpr(e), storeBoxed
	}

	// Simple ident object: access directly, no temp.
	if _, isIdent := e.Object.(*syntax.IdentExpr); isIdent {
		obj := g.emitExpr(e.Object)
		fieldVal := fmt.Sprintf("%s.record_val->fields[%d].value", obj, idx)
		switch fieldSt {
		case storeInt:
			return fieldVal + ".int_val", storeInt
		case storeFloat:
			return fieldVal + ".float_val", storeFloat
		case storeBool:
			return fieldVal + ".bool_val", storeBool
		}
		// Boxed field: deep copy so caller can free/store safely.
		return fmt.Sprintf("monk_deep_copy(%s)", fieldVal), storeBoxed
	}

	// Complex object: stash in a temp to avoid double evaluation.
	obj := g.emitExpr(e.Object)
	tobj := g.newTemp()
	fieldVal := fmt.Sprintf("%s.record_val->fields[%d].value", tobj, idx)
	switch fieldSt {
	case storeInt:
		return fmt.Sprintf("({MonkValue %s=%s; %s.int_val;})", tobj, obj, fieldVal), storeInt
	case storeFloat:
		return fmt.Sprintf("({MonkValue %s=%s; %s.float_val;})", tobj, obj, fieldVal), storeFloat
	case storeBool:
		return fmt.Sprintf("({MonkValue %s=%s; %s.bool_val;})", tobj, obj, fieldVal), storeBool
	}
	return fmt.Sprintf("({MonkValue %s=%s; monk_deep_copy(%s);})", tobj, obj, fieldVal), storeBoxed
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

	// Scalar coercion builtins: to_int / to_float.
	// If the argument is already a raw scalar int or float, inline the
	// coercion with no runtime call. This keeps chains like
	// `y / to_float(H) * 2.0` fully raw.
	//
	// We deliberately do NOT inline for storeBool — the runtime's
	// monk_to_int/monk_to_float reject bool with "expected int, float, or
	// string". Inlining would silently return true->1 / false->0, changing
	// observable program behavior.
	if (ident.Name == "to_int" || ident.Name == "to_float") && len(e.Args) == 1 {
		argCode, argKind := g.emitExprTyped(e.Args[0])
		if argKind == storeInt || argKind == storeFloat {
			if ident.Name == "to_int" {
				if argKind == storeFloat {
					// Truncate toward zero to match runtime behavior.
					return "((int64_t)(" + argCode + "))", storeInt
				}
				return argCode, storeInt // int -> int (identity)
			}
			// to_float
			if argKind == storeInt {
				return "((double)(" + argCode + "))", storeFloat
			}
			return argCode, storeFloat // float -> float (identity)
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
	// Pad defaults before emitting — emitCall does the same; without this,
	// calls with omitted trailing args generate a C call with too few arguments.
	fullArgs := g.padDefaults(cName, e.Args)
	return g.emitUnboxedCall(cName, fs, fullArgs), fs.Return
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
	// Bail unless BOTH sides are raw scalars (int64_t / double / bool).
	// Typed-array storage kinds (storeIntArray etc.) are MonkValue structs —
	// emitting C arithmetic on them would produce nonsensical code.
	// Pass: `i + j` (both storeInt). Fail: `arr + arr` (storeIntArray).
	if !isRawScalar(ls) || !isRawScalar(rs) {
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
		// Use a statement-expression to evaluate rcode ONCE — otherwise a
		// side-effecting divisor like f() would execute twice.
		if result == storeInt {
			return fmt.Sprintf("({ int64_t _d=%s; _d==0 ? (monk_panic(\"division by zero\"),(int64_t)0) : ((int64_t)(%s) / _d); })",
				rcode, lcode), result
		}
		return fmt.Sprintf("(%s / %s)", lcode, rcode), result
	case syntax.Percent:
		if result == storeInt {
			return fmt.Sprintf("({ int64_t _m=%s; _m==0 ? (monk_panic(\"modulo by zero\"),(int64_t)0) : ((int64_t)(%s) %% _m); })",
				rcode, lcode), result
		}
		// float modulo — inline via fmod() to match the compound-assign
		// path (`x %= 2.0`). Zero-check for parity with int modulo.
		return fmt.Sprintf("({ double _m=%s; _m==0.0 ? (monk_panic(\"modulo by zero\"),0.0) : fmod((double)(%s), _m); })",
			rcode, lcode), result
	// Comparison operators return bool.
	case syntax.Less:
		return fmt.Sprintf("(%s < %s)", lcode, rcode), storeBool
	case syntax.LessEqual:
		return fmt.Sprintf("(%s <= %s)", lcode, rcode), storeBool
	case syntax.Greater:
		return fmt.Sprintf("(%s > %s)", lcode, rcode), storeBool
	case syntax.GreaterEqual:
		return fmt.Sprintf("(%s >= %s)", lcode, rcode), storeBool
	case syntax.EqualEqual, syntax.Is:
		// `is` is semantically `==` per emitBinary's boxed path.
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
