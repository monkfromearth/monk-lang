package codegen

import (
	"fmt"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Function hoisting and anonymous function lowering.

// hoistFunction emits a C function definition into the hoisted funcs buffer.
//
// When the checker's Info tells us every parameter AND the return type is a
// scalar (int/float/bool), we emit an unboxed signature:
//
//	static int64_t _monk_func_1(int64_t mk_n) { ... }
//
// Otherwise we fall back to the classic boxed signature:
//
//	static MonkValue _monk_func_1(MonkValue mk_n) { ... }
//
// The cName-to-funcStorage mapping lets call sites know which form to use.
// deriveFuncStorage consults the checker's Info to figure out each param's
// and the return's storage kind. If any slot is boxed, the function as a
// whole stays boxed (All=false); call sites still call it as MonkValue.
func (g *generator) deriveFuncStorage(e *syntax.FuncExpr) funcStorage {
	fs := funcStorage{Params: make([]storageKind, len(e.Params))}
	if g.info == nil {
		return fs
	}
	sig := g.info.Funcs[e]
	if sig == nil {
		return fs
	}
	// Require a known return type and all known param types to be RAW scalars
	// (int64_t / double / bool). Typed-array kinds (storeIntArray etc.) are
	// still MonkValue at the C level — they can't participate in an unboxed
	// function signature, so isRawScalar guards both the return and params.
	fs.Return = storageFor(sig.Return)
	if !isRawScalar(fs.Return) {
		return fs
	}
	allScalar := true
	for i, p := range sig.Params {
		ps := storageFor(p)
		fs.Params[i] = ps
		if !isRawScalar(ps) {
			allScalar = false
		}
	}
	fs.All = allScalar
	return fs
}

// emitTrampoline emits a thunk function matching MonkFuncPtr signature that
// unpacks positional args and calls the real hoisted function.
// captureNames lists captured variables (empty for non-closures). The thunk
// extracts them from _self->captures and passes them as extra trailing args.
func (g *generator) emitTrampoline(cName string, paramCount int, sig funcStorage, captureNames []string) {
	thunkName := cName + "_thunk"
	fmt.Fprintf(&g.funcs, "static MonkValue %s(MonkFunction *_self, MonkValue *_args, int64_t _argc) {\n", thunkName)
	// Unpack args and call the real function
	args := make([]string, paramCount+len(captureNames))
	for i := 0; i < paramCount; i++ {
		if sig.All && i < len(sig.Params) {
			args[i] = fmt.Sprintf("(%s)%s",
				cTypeName(sig.Params[i]),
				unboxExpr(fmt.Sprintf("_args[%d]", i), sig.Params[i]))
		} else {
			args[i] = fmt.Sprintf("_args[%d]", i)
		}
	}
	// For closures with captures, pass _self as the first argument.
	// The hoisted function has _self as its first param and loads captures from it.
	var allArgs []string
	if len(captureNames) > 0 {
		allArgs = append([]string{"_self"}, args[:paramCount]...)
	} else {
		allArgs = args[:paramCount]
	}
	callExpr := fmt.Sprintf("%s(%s)", cName, strings.Join(allArgs, ", "))
	if sig.All && len(captureNames) == 0 {
		fmt.Fprintf(&g.funcs, "    return %s;\n", boxExpr(callExpr, sig.Return))
	} else {
		fmt.Fprintf(&g.funcs, "    return %s;\n", callExpr)
	}
	g.funcs.WriteString("}\n\n")
}

// emitFuncExpr handles an anonymous function literal by allocating a fresh
// C name and delegating to emitFuncValueNamed.
func (g *generator) emitFuncExpr(e *syntax.FuncExpr) string {
	// Anonymous function — allocate a fresh cName.
	g.funcCount++
	cName := fmt.Sprintf("_monk_func_%d", g.funcCount)
	return g.emitFuncValueNamed(cName, e)
}

// emitFuncValueNamed hoists a function under the given cName, emits its
// trampoline, and returns a C expression creating a MonkFunction value.
// The caller must have already allocated cName (and possibly pre-registered
// it in funcNames for recursive self-reference).
func (g *generator) emitFuncValueNamed(cName string, e *syntax.FuncExpr) string {
	// Compute captures — free variables referenced but not declared locally.
	captures := freeVars(e)
	// Filter to only regular variables in the current scope. Hoisted function
	// names are NOT captures — they're available as MonkValue locals (mk_name)
	// and can be called directly or referenced by value.
	var validCaptures []string
	for _, name := range captures {
		mn := mangleName(name)
		if _, isFunc := g.funcNames[name]; isFunc {
			continue // hoisted functions are already MonkValue locals, not captures
		}
		if _, inStorage := g.storage[mn]; inStorage {
			validCaptures = append(validCaptures, name)
		}
	}

	g.hoistFunctionWithCaptures(cName, e, validCaptures)
	sig := g.fnStorage[cName]
	g.emitTrampoline(cName, len(e.Params), sig, validCaptures)
	thunkName := cName + "_thunk"

	if len(validCaptures) == 0 {
		return fmt.Sprintf("monk_make_function(%s, NULL, 0)", thunkName)
	}

	// Build captures array
	captureExprs := make([]string, len(validCaptures))
	for i, name := range validCaptures {
		mn := mangleName(name)
		if store := g.varStorage(mn); store != storeBoxed {
			captureExprs[i] = boxExpr(mn, store)
		} else {
			captureExprs[i] = mn
		}
	}
	return fmt.Sprintf("monk_make_function(%s, (MonkValue[]){%s}, %d)",
		thunkName, strings.Join(captureExprs, ", "), len(validCaptures))
}

// hoistFunctionWithCaptures is like hoistFunction but adds extra MonkValue
// parameters for captured variables from the enclosing scope.
func (g *generator) hoistFunctionWithCaptures(cName string, e *syntax.FuncExpr, captures []string) {
	g.funcHasCapture[cName] = len(captures) > 0

	// Record default expressions
	defaults := make([]syntax.Expr, len(e.Params))
	for i, p := range e.Params {
		defaults[i] = p.Default
	}
	g.funcDefaults[cName] = defaults

	sig := g.deriveFuncStorage(e)
	// If we have captures, force the function to be boxed (captures are MonkValue).
	if len(captures) > 0 {
		sig.All = false
	}
	g.fnStorage[cName] = sig

	savedBody := g.body
	savedRet := g.retStorage
	savedStorage := g.saveStorage()

	// Build parameter list: _self (if captures) + regular params
	var paramParts []string
	if len(captures) > 0 {
		paramParts = append(paramParts, "MonkFunction *_self")
	}
	for i, p := range e.Params {
		store := storeBoxed
		if sig.All {
			store = sig.Params[i]
		}
		g.storage[mangleName(p.Name)] = store
		paramParts = append(paramParts, fmt.Sprintf("%s %s", cTypeName(store), mangleName(p.Name)))
	}
	// Register captured vars in storage so body emission can use them.
	for _, name := range captures {
		g.storage[mangleName(name)] = storeBoxed
	}

	paramStr := strings.Join(paramParts, ", ")
	if paramStr == "" {
		paramStr = "void"
	}

	retType := "MonkValue"
	if sig.All {
		retType = cTypeName(sig.Return)
	}

	// Track captures so emitReturn can save them back before returning.
	savedCaptures := g.currentCaptures
	g.currentCaptures = captures

	g.body = strings.Builder{}
	g.retStorage = storeBoxed
	if sig.All {
		g.retStorage = sig.Return
	}
	g.body.WriteString("    {\n")

	// Load captures from _self into local variables at function entry.
	for i, name := range captures {
		mn := mangleName(name)
		fmt.Fprintf(&g.body, "    MonkValue %s = _self->captures[%d];\n", mn, i)
	}

	if !sig.All {
		for _, p := range e.Params {
			mn := mangleName(p.Name)
			fmt.Fprintf(&g.body, "    %s = monk_deep_copy(%s);\n", mn, mn)
		}
	}

	for _, stmt := range e.Body.Stmts {
		g.emitStmt(stmt)
	}
	// Fallback save-back: emitted inside the block so captured variables are
	// still in scope. Handles implicit-return (fall-through) closures that
	// never hit an explicit `return` statement.
	// Explicit returns go through emitReturn → emitCaptureSaveBack which also
	// writes inside the block, so this is unreachable after them (dead code, harmless).
	for i, name := range captures {
		mn := mangleName(name)
		fmt.Fprintf(&g.body, "    _self->captures[%d] = %s;\n", i, mn)
	}
	g.body.WriteString("    }\n")
	bodyStr := g.body.String()
	g.body = savedBody
	g.retStorage = savedRet
	g.currentCaptures = savedCaptures
	g.restoreStorage(savedStorage)

	fmt.Fprintf(&g.funcs, "static %s %s(%s) {\n", retType, cName, paramStr)
	g.funcs.WriteString(bodyStr)

	if sig.All {
		switch sig.Return {
		case storeInt:
			g.funcs.WriteString("    return 0;\n")
		case storeFloat:
			g.funcs.WriteString("    return 0.0;\n")
		case storeBool:
			g.funcs.WriteString("    return false;\n")
		}
	} else {
		g.funcs.WriteString("    return monk_none();\n")
	}
	g.funcs.WriteString("}\n\n")
}
