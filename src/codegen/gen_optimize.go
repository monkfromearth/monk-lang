package codegen

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// arrayEnsureFunc returns the runtime COW write-barrier for a typed-array
// storage kind. Codegen calls it before direct backing-store writes.
func arrayEnsureFunc(s storageKind) string {
	switch s {
	case storeIntArray:
		return "monk_int_array_ensure_unique"
	case storeFloatArray:
		return "monk_float_array_ensure_unique"
	case storeBoolArray:
		return "monk_bool_array_ensure_unique"
	}
	return ""
}

// isFreshArrayExpr reports whether expr definitely creates a new array backing
// store rather than sharing an existing variable. It is deliberately narrow:
// unknown calls stay "maybe shared" so codegen emits the COW write barrier.
// Pass: `range(N)` creates a fresh backing store — skip the COW write barrier.
// Fail: `let b = a` stays false — b shares a's backing store until a mutation.
// Shadow guard: a user-defined `let append = ...` shadows the builtin; checking
// g.funcNames first ensures the user's function is never misclassified as fresh.
func (g *generator) isFreshArrayExpr(expr syntax.Expr) bool {
	switch e := expr.(type) {
	case *syntax.ArrayExpr:
		return true
	case *syntax.CallExpr:
		callee, ok := e.Callee.(*syntax.IdentExpr)
		if !ok {
			return false
		}
		// User-defined function shadows the builtin — its return value is not
		// guaranteed fresh, so conservatively require the COW write barrier.
		if _, userDefined := g.funcNames[callee.Name]; userDefined {
			return false
		}
		switch callee.Name {
		case "range", "fill", "append", "prepend", "pop", "drop", "take", "slice", "map", "filter":
			return true
		}
	}
	return false
}

// isFreshValueExpr reports whether an expression returns a newly-owned value
// that a `let` binding can take directly instead of deep-copying again.
// Only consulted in the boxed MonkValue path of emitVarDecl (store == storeBoxed).
// Scalar-promoted paths and typed-array paths never reach this check.
// Pass: `let s = to_upper_case(base)` owns the returned string — skip deep_copy.
// Fail: `let b = a` is not fresh and must copy/share to preserve value semantics.
// Shadow guard: if the user writes `let to_upper_case = (s string) string { ... }`,
// g.funcNames["to_upper_case"] exists and we conservatively return false so the
// binding still emits monk_deep_copy — preventing use-after-free via closure captures.
func (g *generator) isFreshValueExpr(expr syntax.Expr) bool {
	switch e := expr.(type) {
	case *syntax.NumberExpr, *syntax.StringExpr, *syntax.TemplateExpr,
		*syntax.BoolExpr, *syntax.NoneExpr, *syntax.ArrayExpr, *syntax.RecordExpr:
		return true
	case *syntax.BinaryExpr, *syntax.UnaryExpr, *syntax.IndexExpr, *syntax.PropertyExpr:
		return true
	case *syntax.CallExpr:
		callee, ok := e.Callee.(*syntax.IdentExpr)
		if !ok {
			return false
		}
		// User-defined function shadows the builtin — its return value may alias
		// closure capture heap data, so deep_copy is required.
		if _, userDefined := g.funcNames[callee.Name]; userDefined {
			return false
		}
		switch callee.Name {
		case "to_string", "to_int", "to_float", "length", "substring", "index_of",
			"split", "trim", "to_upper_case", "to_lower_case",
			"append", "prepend", "pop", "drop", "take", "slice", "range", "fill",
			"abs", "floor", "ceil", "round", "sqrt", "pow", "log", "log10", "exp",
			"min", "max", "sin", "cos", "tan", "asin", "acos", "atan",
			"typeof", "is_number", "is_string", "is_boolean", "is_array",
			"is_record", "is_function", "is_none", "file_read", "file_write",
			"file_exists", "env_get", "map", "filter":
			return true
		}
	}
	return false
}

// markArrayUniquenessFromExpr records whether a typed-array variable is proven
// to be unshared after initialization. Fresh arrays can skip the COW barrier in
// hot writes; identifier copies mark both variables maybe-shared.
func (g *generator) markArrayUniquenessFromExpr(name string, store storageKind, expr syntax.Expr) {
	if !isArrayStorage(store) {
		delete(g.arrayUnique, name)
		return
	}
	if ident, ok := expr.(*syntax.IdentExpr); ok {
		source := g.mangledName(ident.Name)
		if isArrayStorage(g.varStorage(source)) {
			// `let b = a` makes both variables share until first mutation.
			// Pass: later writes to either var emit a COW barrier.
			// Fail: treating a as unique lets `a[0]=...` mutate b too.
			g.arrayUnique[source] = false
		}
		g.arrayUnique[name] = false
		return
	}
	g.arrayUnique[name] = g.isFreshArrayExpr(expr)
}

func (g *generator) stringAppendRHS(s *syntax.AssignStmt) syntax.Expr {
	if g.info == nil {
		return nil
	}
	target, ok := s.Target.(*syntax.IdentExpr)
	if !ok {
		return nil
	}
	switch s.Op {
	case syntax.PlusEqual:
		// Compound string append: checker guarantees target is string when RHS
		// is string. Pass: `s += "x"`. Fail: `n += "x"` never reaches codegen.
		targetType := g.info.Types[s.Target]
		valueType := g.info.Types[s.Value]
		if targetType != nil && valueType != nil &&
			targetType.Kind == types.KindStr && valueType.Kind == types.KindStr &&
			!targetType.Optional && !valueType.Optional {
			return s.Value
		}
	case syntax.Equal:
		bin, ok := s.Value.(*syntax.BinaryExpr)
		if !ok || bin.Op != syntax.Plus {
			return nil
		}
		left, ok := bin.Left.(*syntax.IdentExpr)
		if !ok || left.Name != target.Name {
			return nil
		}
		lt := g.info.Types[bin.Left]
		rt := g.info.Types[bin.Right]
		if lt != nil && rt != nil && lt.Kind == types.KindStr && rt.Kind == types.KindStr && !lt.Optional && !rt.Optional {
			return bin.Right
		}
	}
	return nil
}

func (g *generator) emitKnownTypeBuiltin(e *syntax.CallExpr) (string, storageKind, bool) {
	if g.info == nil || len(e.Args) != 1 || !isKnownTypePureExpr(e.Args[0]) {
		return "", storeBoxed, false
	}
	ident, ok := e.Callee.(*syntax.IdentExpr)
	if !ok {
		return "", storeBoxed, false
	}
	t := g.info.Types[e.Args[0]]
	if t == nil || t.Optional || t.Kind == types.KindAny {
		return "", storeBoxed, false
	}
	typeName, ok := knownTypeName(t)
	if !ok {
		return "", storeBoxed, false
	}
	switch ident.Name {
	case "typeof":
		// Known-type builtin inline for pure args only.
		// Pass: `typeof(n)` where n:int -> "int".
		// Fail: `typeof(f())` must still call f() for its side effects.
		return fmt.Sprintf("monk_string(%s)", cString(typeName)), storeBoxed, true
	case "is_number":
		return boolLiteral(t.Kind == types.KindInt || t.Kind == types.KindFloat), storeBool, true
	case "is_string":
		return boolLiteral(t.Kind == types.KindStr), storeBool, true
	case "is_boolean":
		return boolLiteral(t.Kind == types.KindBool), storeBool, true
	case "is_array":
		return boolLiteral(t.Kind == types.KindArray), storeBool, true
	case "is_record":
		return boolLiteral(t.Kind == types.KindRecord), storeBool, true
	case "is_function":
		return boolLiteral(t.Kind == types.KindFunc), storeBool, true
	case "is_none":
		return boolLiteral(t.Kind == types.KindNone), storeBool, true
	}
	return "", storeBoxed, false
}

func (g *generator) emitLengthCaseFusion(e *syntax.CallExpr) (string, storageKind, bool) {
	if g.info == nil || len(e.Args) != 1 {
		return "", storeBoxed, false
	}
	ident, ok := e.Callee.(*syntax.IdentExpr)
	if !ok || ident.Name != "length" {
		return "", storeBoxed, false
	}
	inner, ok := e.Args[0].(*syntax.CallExpr)
	if !ok || len(inner.Args) != 1 || !isKnownTypePureExpr(inner.Args[0]) {
		return "", storeBoxed, false
	}
	innerCallee, ok := inner.Callee.(*syntax.IdentExpr)
	if !ok || (innerCallee.Name != "to_upper_case" && innerCallee.Name != "to_lower_case") {
		return "", storeBoxed, false
	}
	// Guard: user-defined function shadows the builtin — must not fuse.
	// Pass: `let to_upper_case = (s) string { "FIXED" }; length(to_upper_case("hi"))` → 5, not 2.
	// Fail (without guard): fusion emits monk_length("hi") skipping the user function entirely.
	if _, userDefined := g.funcNames[innerCallee.Name]; userDefined {
		return "", storeBoxed, false
	}
	argType := g.info.Types[inner.Args[0]]
	if argType == nil || argType.Kind != types.KindStr || argType.Optional {
		return "", storeBoxed, false
	}
	// ASCII case conversion preserves UTF-8 character count because it only
	// changes single-byte ASCII letters and passes non-ASCII bytes through.
	// WARNING: this fusion assumes ASCII-only case mapping. If the runtime ever
	// supports locale-aware conversion (e.g. ß→SS), length can change and this
	// optimization must be gated on a locale check or removed entirely.
	// Pass: `length(to_upper_case(s string))` -> `length(s)`.
	// Fail: `length(to_upper_case(f()))` must still call f() exactly once.
	return fmt.Sprintf("(monk_length(%s).int_val)", g.emitExpr(inner.Args[0])), storeInt, true
}

func boolLiteral(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func knownTypeName(t *types.Type) (string, bool) {
	switch t.Kind {
	case types.KindInt:
		return "int", true
	case types.KindFloat:
		return "float", true
	case types.KindStr:
		return "string", true
	case types.KindBool:
		return "boolean", true
	case types.KindNone:
		return "none", true
	case types.KindArray:
		return "array", true
	case types.KindRecord:
		return "record", true
	case types.KindFunc:
		return "function", true
	}
	return "", false
}

func isKnownTypePureExpr(expr syntax.Expr) bool {
	switch expr.(type) {
	case *syntax.NumberExpr, *syntax.StringExpr, *syntax.TemplateExpr,
		*syntax.BoolExpr, *syntax.NoneExpr, *syntax.IdentExpr:
		return true
	}
	return false
}
