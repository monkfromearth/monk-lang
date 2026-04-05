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
func (g *generator) hoistFunction(cName string, e *syntax.FuncExpr) {
	// Decide per-param + return storage. Only unbox when every slot qualifies.
	sig := g.deriveFuncStorage(e)
	g.fnStorage[cName] = sig

	// Snapshot caller's storage BEFORE touching g.storage for this function's
	// params. Otherwise a sibling function's unboxed param binding (e.g.
	// validate_age's mk_age → storeInt) would leak into this function's body
	// and we'd emit a call with a MonkValue where an int64_t is expected.
	savedBody := g.body
	savedRet := g.retStorage
	savedStorage := g.saveStorage()

	params := make([]string, len(e.Params))
	for i, p := range e.Params {
		store := storeBoxed
		if sig.All {
			store = sig.Params[i]
		}
		// Always record the param's storage — even storeBoxed — so that an
		// inner lookup cannot inherit a sibling function's prior binding for
		// the same mangled name.
		g.storage[mangleName(p.Name)] = store
		params[i] = fmt.Sprintf("%s %s", cTypeName(store), mangleName(p.Name))
	}

	paramStr := strings.Join(params, ", ")
	if paramStr == "" {
		paramStr = "void"
	}

	retType := "MonkValue"
	if sig.All {
		retType = cTypeName(sig.Return)
	}
	fmt.Fprintf(&g.funcs, "static %s %s(%s) {\n", retType, cName, paramStr)

	// Emit body into funcs buffer (swap body temporarily). Track the expected
	// return storage so emitReturn can coerce.
	// Wrap the body in an inner block so user code can safely shadow params:
	//   let f = (x int) int { let x = "y"; ... }
	// would otherwise redeclare mk_x at the same C scope.
	g.body = strings.Builder{}
	if sig.All {
		g.retStorage = sig.Return
	} else {
		g.retStorage = storeBoxed
	}
	g.body.WriteString("    {\n")
	for _, stmt := range e.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.body.WriteString("    }\n")
	g.funcs.WriteString(g.body.String())
	g.body = savedBody
	g.retStorage = savedRet
	g.restoreStorage(savedStorage)

	// Fallback return at the end of function body.
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

	// Param storage bindings are already cleared by restoreStorage above.
}

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
	// Require a known return type and all known param types to be scalar.
	fs.Return = storageFor(sig.Return)
	if fs.Return == storeBoxed {
		return fs
	}
	allScalar := true
	for i, p := range sig.Params {
		ps := storageFor(p)
		fs.Params[i] = ps
		if ps == storeBoxed {
			allScalar = false
		}
	}
	fs.All = allScalar
	return fs
}

func (g *generator) emitFuncExpr(e *syntax.FuncExpr) string {
	// Anonymous function expression — hoist it
	g.funcCount++
	cName := fmt.Sprintf("_monk_func_%d", g.funcCount)
	g.hoistFunction(cName, e)
	return "monk_none() /* anonymous func */"
}
