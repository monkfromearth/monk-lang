package codegen

import (
	"fmt"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Statement emission. Each emit* method writes directly into g.body.

// emitStmt dispatches to the appropriate emitter for each statement kind.
// Unrecognized nodes emit a C comment so the build still succeeds while
// making the gap visible.
func (g *generator) emitStmt(stmt syntax.Stmt) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		g.emitVarDecl(s, g.moduleInit)
	case *syntax.AssignStmt:
		g.emitAssign(s)
	case *syntax.ExprStmt:
		g.emitLine("    %s;\n", g.emitExpr(s.Expr))
	case *syntax.IfStmt:
		g.emitIf(s)
	case *syntax.WhileStmt:
		g.emitWhile(s)
	case *syntax.ForStmt:
		g.emitFor(s)
	case *syntax.ReturnStmt:
		g.emitReturn(s)
	case *syntax.BreakStmt:
		g.emitLine("    break;\n")
	case *syntax.ContinueStmt:
		g.emitLine("    continue;\n")
	case *syntax.GuardStmt:
		g.emitGuard(s)
	case *syntax.BlockStmt:
		for _, inner := range s.Stmts {
			g.emitStmt(inner)
		}
	case *syntax.ExportStmt:
		// Export is a visibility marker for the module system.
		// Bare "export name" (ExprStmt(IdentExpr)) emits nothing — the name
		// was already declared. Otherwise emit the inner declaration.
		if es, ok := s.Stmt.(*syntax.ExprStmt); ok {
			if _, ok := es.Expr.(*syntax.IdentExpr); ok {
				return // export-by-name, no code to emit
			}
		}
		g.emitStmt(s.Stmt)
	case *syntax.UseStmt:
		// Imports resolved at generator construction time — nothing to emit.
	default:
		g.emitLine("    /* TODO: unhandled statement %T */\n", stmt)
	}
}

func (g *generator) emitVarDecl(s *syntax.VarDeclStmt, forModule bool) {
	// Function declarations: hoist the C function, track the name mapping,
	// and emit a MonkValue wrapper so the function can be used as a value.
	if fnExpr, isFn := s.Value.(*syntax.FuncExpr); isFn {
		// Pre-register the function name so recursive self-references inside
		// the body can find it during hoisting.
		g.funcCount++
		cFuncName := fmt.Sprintf("_monk_%sfunc_%d", g.modulePrefix, g.funcCount)
		g.funcNames[s.Name] = cFuncName
		// emitFuncValueNamed uses the pre-allocated cName (doesn't increment funcCount again).
		funcVal := g.emitFuncValueNamed(cFuncName, fnExpr)
		name := g.mangledName(s.Name)
		if forModule {
			// Module mode: declare as static global, assign in init body.
			// static MonkValue mk_m0_add; (at file scope)
			// mk_m0_add = monk_make_function(...); (in init body)
			fmt.Fprintf(&g.globals, "static MonkValue %s;\n", name)
			g.emitLine("    %s = %s;\n", name, funcVal)
		} else {
			g.emitLine("    MonkValue %s = %s;\n", name, funcVal)
		}
		return
	}

	// Module mode for non-function variables: declare as static global,
	// assign in init body. This makes exported variables visible across C
	// functions (init function + caller's main).
	//
	// Strategy: temporarily capture the normal emitVarDecl output, then
	// split "TYPE name = expr;" into "static TYPE name;" (global) +
	// "name = expr;" (init body).
	if forModule {
		g.emitModuleVarDecl(s)
		return
	}

	name := g.mangledName(s.Name)

	// Decide this variable's storage based on the type checker's verdict.
	// Absent type info (legacy Generate path), stay boxed.
	var store storageKind
	if g.info != nil {
		if t, ok := g.info.Decls[s]; ok {
			store = storageFor(t)
		}
	}

	// ── Typed-array backing-store path ──────────────────────────────────
	// int[]/float[]/bool[] variables use a typed backing store (int64_t* /
	// double* / bool*) instead of MonkValue*. monk_int_array_from() handles
	// both conversion from generic MONK_ARRAY (e.g. range(N)) and deep-copy
	// from an existing typed array. It frees the MONK_ARRAY temporary after
	// conversion, so no separate deep_copy is needed.
	if isArrayStorage(store) {
		g.storage[name] = store
		// Specialized range: `range(N)` on int[] emits monk_range_int(N) directly,
		// avoiding 10M intermediate MonkValues. range(10M) → 80 MB vs 240 MB.
		if store == storeIntArray {
			if call, ok := s.Value.(*syntax.CallExpr); ok {
				if callee, ok := call.Callee.(*syntax.IdentExpr); ok && callee.Name == "range" && len(call.Args) == 1 {
					nCode, nStore := g.emitExprTyped(call.Args[0])
					if nStore == storeInt {
						g.emitLine("    MonkValue %s = monk_range_int(%s);\n", name, nCode)
					} else {
						g.emitLine("    MonkValue %s = monk_range_int((%s).int_val);\n", name, g.emitExpr(call.Args[0]))
					}
					g.recordArrayLen(s.Name, s.Value)
					return
				}
			}
		}
		// Specialized fill: `fill(n, val)` on a typed array emits a direct
		// monk_fill_bool/int/float call that allocates the typed backing store
		// directly — no intermediate MonkValue array + conversion.
		// fill(1M, true) → 1 MB memset instead of 16 MB alloc + convert + free.
		if call, ok := s.Value.(*syntax.CallExpr); ok {
			if callee, ok := call.Callee.(*syntax.IdentExpr); ok && callee.Name == "fill" && len(call.Args) == 2 {
				// Emit first arg (count) typed: if it's an int, wrap with
				// monk_int() directly to avoid the runtime string-check.
				// fill(N + 1, true) → monk_int(mk_N + 1) instead of monk_add(...)
				nCode, nStore := g.emitExprTyped(call.Args[0])
				if nStore == storeInt {
					nCode = "monk_int(" + nCode + ")"
				} else {
					nCode = g.emitExpr(call.Args[0])
				}
				valCode, valStore := g.emitExprTyped(call.Args[1])
				switch {
				case store == storeBoolArray && valStore == storeBool:
					g.emitLine("    MonkValue %s = monk_fill_bool(%s, %s);\n", name, nCode, valCode)
					g.recordArrayLen(s.Name, s.Value)
					return
				case store == storeIntArray && valStore == storeInt:
					g.emitLine("    MonkValue %s = monk_fill_int(%s, %s);\n", name, nCode, valCode)
					g.recordArrayLen(s.Name, s.Value)
					return
				case store == storeFloatArray && valStore == storeFloat:
					g.emitLine("    MonkValue %s = monk_fill_float(%s, %s);\n", name, nCode, valCode)
					g.recordArrayLen(s.Name, s.Value)
					return
				}
			}
		}
		value := g.emitExpr(s.Value)
		convFn := arrayConvFunc(store)
		g.emitLine("    MonkValue %s = %s(%s);\n", name, convFn, value)
		g.recordArrayLen(s.Name, s.Value) // bounds-check elision
		return
	}

	// ── Boxed path with scalar-promotion probe ───────────────────────────
	// When the declared type is boxed, try emitting the RHS typed ONLY when
	// the variable has NO explicit type annotation (s.Type == nil).
	//
	// Rationale: `let aik = A[i*N+k]` has no annotation — the inferred type
	// is int? but the user just wants an int. Promoting to int64_t (which
	// panics on OOB) is safe because they're going to use it in arithmetic
	// anyway (none in arithmetic → runtime panic regardless).
	//
	// `let tenth int? = nums[10]` has an explicit int? annotation — the user
	// expects graceful OOB (spec: reading missing data returns none). We
	// preserve that by staying on the classic monk_array_get path.
	if store == storeBoxed {
		var rhsCode string
		if s.Type == nil {
			var rhsStore storageKind
			rhsCode, rhsStore = g.emitExprTyped(s.Value)
			if isRawScalar(rhsStore) {
				store = rhsStore
				g.storage[name] = store
				g.emitLine("    %s %s = %s;\n", cTypeName(store), name, rhsCode)
				g.recordConst(s.Name, s.Value) // bounds-check elision
				return
			}
			// emitExprTyped fell through to emitExpr — rhsCode is already a
			// boxed MonkValue expression; reuse it below.
		} else {
			// Explicit type annotation: always use classic boxed path so that
			// graceful array OOB (→ none) and other spec-mandated behaviours
			// are preserved exactly.
			rhsCode = g.emitExpr(s.Value)
		}
		g.storage[name] = storeBoxed
		g.emitLine("    MonkValue %s = monk_deep_copy(%s);\n", name, rhsCode)
		return
	}

	// ── Pure scalar unboxed path ─────────────────────────────────────────
	g.storage[name] = store
	rhsCode, rhsStore := g.emitExprTyped(s.Value)
	init := coerce(rhsCode, rhsStore, store)
	g.emitLine("    %s %s = %s;\n", cTypeName(store), name, init)
	g.recordConst(s.Name, s.Value) // bounds-check elision
}

// emitModuleVarDecl handles top-level variable declarations in non-entry
// modules. Variables must be declared as file-scope `static` globals so they're
// visible from both the init function and the importing module's main().
//
// Strategy: run the normal emitVarDecl into a temporary body buffer, then
// transform each "    TYPE name = expr;\n" into:
//
//   - "static TYPE name;\n" in g.globals  (file scope)
//   - "    name = expr;\n"  in g.body     (init body)
//
// Example:
//
//	Module code: let x = 42
//	Globals:     static int64_t mk_m0_x;
//	Init body:       mk_m0_x = 42;
//
// ASSUMPTION: all C type tokens emitted by emitVarDecl are single words
// (MonkValue, int64_t, double, bool). Multi-word types (e.g. "unsigned long")
// would confuse the parts[0]/parts[1] split. If that ever happens, the panic
// guard below fires with a clear message rather than emitting corrupt C.
//
func (g *generator) emitModuleVarDecl(s *syntax.VarDeclStmt) {
	// Save the real body, swap in a temp buffer.
	saved := g.body
	g.body = strings.Builder{}

	// Call emitVarDecl with forModule=false so it takes the normal (non-module)
	// path — we handle the global/init split ourselves below.
	g.emitVarDecl(s, false)

	output := g.body.String()
	g.body = saved

	// Transform the captured output. Each line is "    TYPE name = expr;\n".
	// We split on the first " = " to get the declaration and initializer.
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Find "TYPE name = expr;" pattern.
		// Some lines may not be declarations (e.g. intermediate temp vars
		// or side-effect statements). Those go straight to init body.
		eqIdx := strings.Index(trimmed, " = ")
		if eqIdx == -1 || !strings.HasSuffix(trimmed, ";") {
			g.emitLine("%s\n", line)
			continue
		}

		declPart := trimmed[:eqIdx]   // "TYPE name" or just "name" for reassignment
		exprPart := trimmed[eqIdx+3:] // "expr;" (includes semicolon)

		// Check if this is a declaration (has a type prefix) vs an assignment.
		// Declarations have format "TYPE name" where TYPE is one of the single-token
		// C types emitted by Monk codegen: MonkValue, int64_t, double, bool.
		parts := strings.Fields(declPart)
		if len(parts) == 2 {
			typeName := parts[0]
			varName := parts[1]
			// Guard: if the type name contains a space, our parsing assumption has
			// broken. Panic early with a clear message rather than emitting corrupt C.
			if strings.Contains(typeName, " ") {
				panic("emitModuleVarDecl: multi-word C type '" + typeName + "' — update parser to use forModule bool")
			}
			// Emit as static global + init assignment.
			fmt.Fprintf(&g.globals, "static %s %s;\n", typeName, varName)
			g.emitLine("    %s = %s\n", varName, exprPart)
		} else {
			// Not a simple declaration — emit as-is in init body.
			g.emitLine("%s\n", line)
		}
	}
}

func (g *generator) emitAssign(s *syntax.AssignStmt) {
	// Unboxed scalar fast path: plain ident with raw scalar storage (int64_t,
	// double, bool). Typed arrays are excluded — they need free+reconvert on
	// reassignment (e.g. `arr = append(arr, x)` returns MONK_ARRAY, not
	// MONK_INT_ARRAY). Without this guard, typed arrays get a raw struct copy
	// that leaks the old backing store and reads the wrong union member.
	// Pass: `x = x + 1` (storeInt). Fail: `arr = append(arr, 5)` (storeIntArray).
	if target, ok := s.Target.(*syntax.IdentExpr); ok {
		name := g.mangledName(target.Name)
		if store := g.varStorage(name); isRawScalar(store) {
			rhsCode, rhsStore := g.emitExprTyped(s.Value)
			rhs := coerce(rhsCode, rhsStore, store)
			// Stash RHS in a temp for ops that need to evaluate it twice
			// (div/mod guards), so a side-effecting RHS like `a /= f()`
			// doesn't call f() both for the zero-check and the divide.
			switch s.Op {
			case syntax.Equal:
				g.emitLine("    %s = %s;\n", name, rhs)
			case syntax.PlusEqual:
				g.emitLine("    %s += %s;\n", name, rhs)
			case syntax.MinusEqual:
				g.emitLine("    %s -= %s;\n", name, rhs)
			case syntax.StarEqual:
				g.emitLine("    %s *= %s;\n", name, rhs)
			case syntax.SlashEqual:
				if store == storeInt {
					tmp := g.newTemp()
					g.emitLine("    { int64_t %s = %s; if (%s==0) monk_panic(\"division by zero\"); %s /= %s; }\n",
						tmp, rhs, tmp, name, tmp)
				} else {
					g.emitLine("    %s /= %s;\n", name, rhs)
				}
			case syntax.PercentEqual:
				if store == storeInt {
					tmp := g.newTemp()
					g.emitLine("    { int64_t %s = %s; if (%s==0) monk_panic(\"modulo by zero\"); %s %%= %s; }\n",
						tmp, rhs, tmp, name, tmp)
				} else {
					// float %=: C's `%` doesn't work on doubles, but spec
					// says `%` applies to numeric. Use fmod() from math.h
					// (already linked via -lm). Zero-check for consistency.
					tmp := g.newTemp()
					g.emitLine("    { double %s = %s; if (%s==0.0) monk_panic(\"modulo by zero\"); %s = fmod(%s, %s); }\n",
						tmp, rhs, tmp, name, name, tmp)
				}
			}
			return
		}
	}

	// ── Typed array element write fast path ─────────────────────────────
	// When the target is arr[i] where arr is a typed int[]/float[]/bool[],
	// emit a direct element write into the backing store instead of monk_array_set.
	// Backing-store path: arr.int_array_val->data[i] = rhs (no struct overhead).
	// Only for plain = (not +=, -= etc.) and only for direct ident targets.
	if s.Op == syntax.Equal {
		if indexTarget, ok := s.Target.(*syntax.IndexExpr); ok {
			if identObj, ok2 := indexTarget.Object.(*syntax.IdentExpr); ok2 {
				objName := g.mangledName(identObj.Name)
				objSt := g.varStorage(objName)
				elemSt := elemStorageFor(objSt)
				if elemSt != storeBoxed {
					ptrField := arrayPtrField(objSt)
					idxCode, idxKind := g.emitExprTyped(indexTarget.Index)
					idxC := coerce(idxCode, idxKind, storeInt)
					rhsCode, rhsSt := g.emitExprTyped(s.Value)
					elemCode := coerce(rhsCode, rhsSt, elemSt)
					// Bounds-check elision: skip runtime check when statically provable.
					// idxC is a pure arithmetic expression (range analysis proved it),
					// so inlining it directly lets the compiler hoist and vectorize.
					if g.constVals != nil && g.isBoundedSafe(identObj, indexTarget.Index) {
						g.emitLine("    %s.%s->data[%s] = %s;\n", objName, ptrField, idxC, elemCode)
					} else {
						tidx := g.newTemp()
						g.emitLine("    { int64_t %s = %s; if (%s < 0 || %s >= %s.%s->length) monk_panic(\"index out of bounds\"); %s.%s->data[%s] = %s; }\n",
							tidx, idxC, tidx, tidx, objName, ptrField, objName, ptrField, tidx, elemCode)
					}
					return
				}
			}
		}
	}

	// ── Typed record field write fast path ──────────────────────────────────
	// When target is rec.field where rec is a plain ident with a statically
	// known record type, emit a direct index write instead of monk_record_set.
	// For scalar fields: no free/copy needed (no heap allocation).
	// For boxed fields: free + copy using index (still avoids the strcmp loop).
	// Only for plain = (not +=, -= etc.) to match the typed-array fast path.
	if s.Op == syntax.Equal && g.info != nil {
		if propTarget, ok := s.Target.(*syntax.PropertyExpr); ok {
			if identObj, ok2 := propTarget.Object.(*syntax.IdentExpr); ok2 {
				objName := g.mangledName(identObj.Name)
				if objType, ok3 := g.info.Types[propTarget.Object]; ok3 && objType != nil {
					idx, fieldSt := recordField(objType, propTarget.Property)
					if idx >= 0 {
						rhsCode, rhsSt := g.emitExprTyped(s.Value)
						switch fieldSt {
						case storeInt:
							rhs := coerce(rhsCode, rhsSt, storeInt)
							g.emitLine("    %s.record_val->fields[%d].value = monk_int(%s);\n", objName, idx, rhs)
							return
						case storeFloat:
							rhs := coerce(rhsCode, rhsSt, storeFloat)
							g.emitLine("    %s.record_val->fields[%d].value = monk_float(%s);\n", objName, idx, rhs)
							return
						case storeBool:
							rhs := coerce(rhsCode, rhsSt, storeBool)
							g.emitLine("    %s.record_val->fields[%d].value = monk_bool(%s);\n", objName, idx, rhs)
							return
						default:
							// Boxed field (string, record, array): free old, copy new.
							rhs := coerce(rhsCode, rhsSt, storeBoxed)
							tmp := g.newTemp()
							g.emitLine("    { MonkValue %s = monk_deep_copy(%s); monk_free(%s.record_val->fields[%d].value); %s.record_val->fields[%d].value = %s; }\n",
								tmp, rhs, objName, idx, objName, idx, tmp)
							return
						}
					}
				}
			}
		}
	}
	value := g.emitExpr(s.Value)

	switch target := s.Target.(type) {
	case *syntax.IdentExpr:
		name := g.mangledName(target.Name)
		// Invalidate direct-call optimization on function reassignment.
		// `let f = (x) { x+1 }; f = (x) { x*2 }; f(5)` — without this,
		// f(5) still emits `_monk_func_1(5)` (the OLD function) because
		// funcNames["f"] hardcodes the hoisted C name. Deleting the entry
		// forces subsequent calls through monk_call(mk_f, ...) which reads
		// the variable and dispatches to whatever function it currently holds.
		// Pass: f(5) returns 10 after reassignment. Fail (before fix): returns 6.
		delete(g.funcNames, target.Name)
		tmp := g.newTemp()
		switch s.Op {
		case syntax.Equal:
			// Compute new value BEFORE freeing old (avoids use-after-free
			// when the new value expression references the variable)
			g.emitLine("    { MonkValue %s = monk_deep_copy(%s);\n", tmp, value)
			g.emitLine("      monk_free(%s);\n", name)
			g.emitLine("      %s = %s; }\n", name, tmp)
		case syntax.PlusEqual:
			// += mirrors Plus: dispatch on MONK_STRING for the concat overload
			// so `s += "world"` on a string routes through monk_string_concat
			// instead of monk_add (which would runtime-error).
			g.emitLine("    { MonkValue %s = (%s.kind==MONK_STRING ? monk_string_concat(%s,%s) : monk_add(%s,%s));\n",
				tmp, name, name, value, name, value)
			g.emitLine("      monk_free(%s);\n", name)
			g.emitLine("      %s = %s; }\n", name, tmp)
		default:
			op := compoundToArith(s.Op)
			g.emitLine("    { MonkValue %s = %s(%s, %s);\n", tmp, op, name, value)
			g.emitLine("      monk_free(%s);\n", name)
			g.emitLine("      %s = %s; }\n", name, tmp)
		}
	case *syntax.IndexExpr:
		// Common case — target.Object is a plain identifier, so we can take
		// its address directly and mutate in place. Nested targets like
		// `matrix[i][j]` produce an rvalue from emitExpr (`monk_array_get(…)`),
		// which cannot be the operand of &. Route those through a temp so
		// the generated C compiles; mutating the intermediate copy is a
		// no-op per Monk's value semantics, which is the intended behavior —
		// nested index assignment does NOT propagate to the outer container.
		// Users who need deep mutation must rebind: `let row = m[i]; row[j] = x; m[i] = row`.
		idx := g.emitExpr(target.Index)
		if _, ok := target.Object.(*syntax.IdentExpr); ok {
			obj := g.emitExpr(target.Object)
			g.emitLine("    monk_array_set(&%s, %s, %s);\n", obj, idx, value)
		} else {
			obj := g.emitExpr(target.Object)
			tmp := g.newTemp()
			g.emitLine("    { MonkValue %s = %s; monk_array_set(&%s, %s, %s); }\n",
				tmp, obj, tmp, idx, value)
		}
	case *syntax.PropertyExpr:
		// Same rationale as IndexExpr above.
		if _, ok := target.Object.(*syntax.IdentExpr); ok {
			obj := g.emitExpr(target.Object)
			g.emitLine("    monk_record_set(&%s, \"%s\", %s);\n", obj, target.Property, value)
		} else {
			obj := g.emitExpr(target.Object)
			tmp := g.newTemp()
			g.emitLine("    { MonkValue %s = %s; monk_record_set(&%s, \"%s\", %s); }\n",
				tmp, obj, tmp, target.Property, value)
		}
	}
}

// emitCondition generates the C expression for an `if`/`while` condition.
// Uses typed emission when possible so `while (i < N)` stays as raw C
// comparison instead of going through monk_is_truthy(monk_less(...)).
func (g *generator) emitCondition(e syntax.Expr) string {
	if g.info != nil {
		code, kind := g.emitExprTyped(e)
		// Strip ONE layer of outer parens when the whole expression is wrapped
		// in a single balanced pair. This avoids cc's -Wparentheses-equality
		// warning on code like `if ((a == b))`. Unsafe strips would change
		// meaning (e.g. `(a)+(b)` → `a)+(b`), so we verify balance.
		code = stripOuterParens(code)
		switch kind {
		case storeBool:
			return code
		case storeInt, storeFloat:
			// Truthiness: 0 is falsy, everything else is truthy.
			return "(" + code + ") != 0"
		}
	}
	return "monk_is_truthy(" + g.emitExpr(e) + ")"
}

// stripOuterParens removes ONE layer of enclosing parens if the first `(`
// matches the last `)` directly (i.e. the entire expression is wrapped).
// Returns s unchanged for everything else.
// IMPORTANT: never strip parens from GCC statement expressions `({...})` —
// removing the outer `()` turns a valid expression into a bare `{...}` block.
// Pass: `((a == b))` → `(a == b)`. Fail: `({int64_t t=x; t;})` → unchanged.
func stripOuterParens(s string) string {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return s
	}
	// Statement expression: ({...}) — must keep outer parens.
	if len(s) >= 3 && s[1] == '{' {
		return s
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i < len(s)-1 {
				// The matching `)` for the opening `(` is not at the end —
				// meaning the expression is NOT wrapped in a single balanced
				// pair. Example: "(a)+(b)" — depth hits 0 at index 2.
				return s
			}
		}
	}
	return s[1 : len(s)-1]
}

// emitIf handles if/else-if/else chains recursively.
func (g *generator) emitIf(s *syntax.IfStmt) {
	g.emitLine("    if (%s) {\n", g.emitCondition(s.Condition))
	thenSnap := g.saveStorage()
	for _, stmt := range s.Then.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(thenSnap)
	if s.Else != nil {
		switch e := s.Else.(type) {
		case *syntax.BlockStmt:
			g.emitLine("    } else {\n")
			elseSnap := g.saveStorage()
			for _, stmt := range e.Stmts {
				g.emitStmt(stmt)
			}
			g.restoreStorage(elseSnap)
			g.emitLine("    }\n")
			return
		case *syntax.IfStmt:
			// Recursive: handles arbitrary else-if chain depth
			g.body.WriteString("    } else ")
			g.emitIfInline(e)
			return
		}
	}
	g.emitLine("    }\n")
}

// emitIfInline emits an if statement without the leading indent (for else-if chains).
func (g *generator) emitIfInline(s *syntax.IfStmt) {
	fmt.Fprintf(&g.body, "if (%s) {\n", g.emitCondition(s.Condition))
	thenSnap := g.saveStorage()
	for _, stmt := range s.Then.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(thenSnap)
	if s.Else != nil {
		switch e := s.Else.(type) {
		case *syntax.BlockStmt:
			g.emitLine("    } else {\n")
			elseSnap := g.saveStorage()
			for _, stmt := range e.Stmts {
				g.emitStmt(stmt)
			}
			g.restoreStorage(elseSnap)
			g.emitLine("    }\n")
			return
		case *syntax.IfStmt:
			g.body.WriteString("    } else ")
			g.emitIfInline(e)
			return
		}
	}
	g.emitLine("    }\n")
}

// emitWhile emits a C `while` loop. The storage snapshot is taken before the
// body and restored after so that variables declared inside the loop don't
// pollute the outer storage map.
func (g *generator) emitWhile(s *syntax.WhileStmt) {
	g.emitLine("    while (%s) {\n", g.emitCondition(s.Condition))
	snap := g.saveStorage()
	// Bounds-check elision: track the loop counter's range so that array
	// accesses inside the body can be proven in-bounds.
	var boundVar string
	var boundPrev [2]int64
	var boundHad bool
	if g.constVals != nil {
		if varName, lo, hi, found := g.whileBoundsEntry(s.Condition); found {
			boundVar = varName
			boundPrev, boundHad = g.setVarBound(varName, lo, hi)
		}
	}
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	if boundVar != "" {
		g.restoreVarBound(boundVar, boundPrev, boundHad)
	}
	g.restoreStorage(snap)
	g.emitLine("    }\n")
}

func (g *generator) emitFor(s *syntax.ForStmt) {
	varName := g.mangledName(s.VarName)

	// Counter-loop fast path: `for x in range(N)` → `for(int64_t x=0; x<N; x++)`.
	// Avoids allocating an N-element array entirely.
	// Pass: `for i in range(10)` → counter. Fail: `for x in arr` → iterate.
	if call, ok := s.Iterable.(*syntax.CallExpr); ok {
		if callee, ok := call.Callee.(*syntax.IdentExpr); ok && callee.Name == "range" && len(call.Args) == 1 {
			nCode, nStore := g.emitExprTyped(call.Args[0])
			var nExpr string
			if nStore == storeInt {
				nExpr = nCode
			} else {
				nExpr = "(" + g.emitExpr(call.Args[0]) + ").int_val"
			}
			g.emitLine("    {\n")
			g.emitLine("        int64_t _n = %s;\n", nExpr)
			g.emitLine("        for (int64_t %s = 0; %s < _n; %s++) {\n", varName, varName, varName)
			g.emitLine("            {\n")
			snap := g.saveStorage()
			g.storage[varName] = storeInt
			for _, stmt := range s.Body.Stmts {
				g.emitStmt(stmt)
			}
			g.restoreStorage(snap)
			g.emitLine("            }\n")
			g.emitLine("        }\n")
			g.emitLine("    }\n")
			return
		}
	}

	iter := g.emitExpr(s.Iterable)

	// Fast path: when we know the iterable is a typed array at compile time,
	// emit a direct loop with a raw scalar loop variable — no boxing, no
	// runtime kind-dispatch. This is what makes for-in over int[] match C.
	if g.info != nil {
		if iterType, ok := g.info.Types[s.Iterable]; ok && iterType != nil {
			iterStore := storageFor(iterType)
			if isArrayStorage(iterStore) {
				ptrField := arrayPtrField(iterStore)
				var elemStore storageKind
				var cType string
				switch iterStore {
				case storeIntArray:
					elemStore = storeInt
					cType = "int64_t"
				case storeFloatArray:
					elemStore = storeFloat
					cType = "double"
				case storeBoolArray:
					elemStore = storeBool
					cType = "bool"
				}
				// Extract raw element from the iterable without conversion.
				// Handles both typed backing store (MONK_INT_ARRAY) and generic
				// MONK_ARRAY (reads .int_val from each MonkValue element).
				// No allocation, no copy — just a branch at loop setup.
				var kindName, genericField string
				switch iterStore {
				case storeIntArray:
					kindName = "MONK_INT_ARRAY"
					genericField = "int_val"
				case storeFloatArray:
					kindName = "MONK_FLOAT_ARRAY"
					genericField = "float_val"
				case storeBoolArray:
					kindName = "MONK_BOOL_ARRAY"
					genericField = "bool_val"
				}
				g.emitLine("    {\n")
				g.emitLine("        MonkValue _iter = %s;\n", iter)
				g.emitLine("        int64_t _len = (_iter.kind == %s) ? _iter.%s->length : _iter.array_val->length;\n",
					kindName, ptrField)
				g.emitLine("        for (int64_t _i = 0; _i < _len; _i++) {\n")
				g.emitLine("            %s %s = (_iter.kind == %s) ? _iter.%s->data[_i] : _iter.array_val->data[_i].%s;\n",
					cType, varName, kindName, ptrField, genericField)
				g.emitLine("            {\n")
				snap := g.saveStorage()
				g.storage[varName] = elemStore
				for _, stmt := range s.Body.Stmts {
					g.emitStmt(stmt)
				}
				g.restoreStorage(snap)
				g.emitLine("            }\n")
				g.emitLine("        }\n")
				g.emitLine("    }\n")
				return
			}
		}
	}

	g.emitLine("    {\n")
	g.emitLine("        MonkValue _iter = %s;\n", iter)
	// Generic MONK_ARRAY path
	g.emitLine("        if (_iter.kind == MONK_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_deep_copy(_iter.array_val->data[_i]);\n", varName)
	// Inner block so user code can safely shadow the loop variable.
	g.emitLine("                {\n")
	arrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(arrSnap)
	g.emitLine("                }\n")
	g.emitLine("                monk_free(%s);\n", varName)
	g.emitLine("            }\n")
	// Typed backing-store array paths: iterate raw scalars, box each into MonkValue
	// so the loop body always sees a MonkValue (consistent with MONK_ARRAY path).
	g.emitLine("        } else if (_iter.kind == MONK_INT_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.int_array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_int(_iter.int_array_val->data[_i]);\n", varName)
	g.emitLine("                {\n")
	intArrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(intArrSnap)
	g.emitLine("                }\n")
	g.emitLine("            }\n")
	g.emitLine("        } else if (_iter.kind == MONK_FLOAT_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.float_array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_float(_iter.float_array_val->data[_i]);\n", varName)
	g.emitLine("                {\n")
	floatArrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(floatArrSnap)
	g.emitLine("                }\n")
	g.emitLine("            }\n")
	g.emitLine("        } else if (_iter.kind == MONK_BOOL_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.bool_array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_bool(_iter.bool_array_val->data[_i]);\n", varName)
	g.emitLine("                {\n")
	boolArrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(boolArrSnap)
	g.emitLine("                }\n")
	g.emitLine("            }\n")
	g.emitLine("        } else if (_iter.kind == MONK_STRING) {\n")
	g.emitLine("            for (int64_t _i = 0; _iter.str_val[_i]; ) {\n")
	g.emitLine("                int64_t _clen = 1;\n")
	// Fix 6: bounds check for malformed UTF-8
	g.emitLine("                while (_iter.str_val[_i + _clen] && (_iter.str_val[_i + _clen] & 0xC0) == 0x80) _clen++;\n")
	g.emitLine("                char *_ch = malloc(_clen + 1);\n")
	g.emitLine("                if (!_ch) monk_panic(\"out of memory\");\n")
	g.emitLine("                memcpy(_ch, _iter.str_val + _i, _clen);\n")
	g.emitLine("                _ch[_clen] = '\\0';\n")
	// Note: value-semantics keeps this safe even though _ch is freed after
	// the body runs. Any downstream consumer (emitVarDecl, monk_append,
	// monk_record_set, …) deep-copies the loop variable on store, so a
	// captured reference owns its own string by the time we hit free(_ch).
	g.emitLine("                MonkValue %s = (MonkValue){.kind = MONK_STRING, .str_val = _ch};\n", varName)
	g.emitLine("                {\n")
	strSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(strSnap)
	g.emitLine("                }\n")
	g.emitLine("                free(_ch);\n")
	g.emitLine("                _i += _clen;\n")
	g.emitLine("            }\n")
	g.emitLine("        }\n")
	g.emitLine("    }\n")
}

// emitReturn saves captured variables back to _self->captures (via
// emitCaptureSaveBack) then emits the C return. For unboxed functions the
// return value is coerced to the declared raw scalar type.
func (g *generator) emitReturn(s *syntax.ReturnStmt) {
	// Save captured variables back to _self->captures before returning.
	g.emitCaptureSaveBack()

	if s.Value == nil {
		if g.retStorage == storeBoxed {
			g.emitLine("    return monk_none();\n")
		} else {
			g.emitLine("    return 0;\n")
		}
		return
	}
	if g.retStorage == storeBoxed {
		g.emitLine("    return %s;\n", g.emitExpr(s.Value))
		return
	}
	// Unboxed return: evaluate in typed form and coerce.
	code, kind := g.emitExprTyped(s.Value)
	g.emitLine("    return %s;\n", coerce(code, kind, g.retStorage))
}

// emitCaptureSaveBack writes local capture variables back to _self->captures
// so mutations persist across calls. Only emits if the current function has captures.
func (g *generator) emitCaptureSaveBack() {
	for i, name := range g.currentCaptures {
		mn := g.mangledName(name)
		g.emitLine("    _self->captures[%d] = %s;\n", i, mn)
	}
}

// emitGuard emits the setjmp-based guard construct. The guarded expression runs
// inside monk_guard_begin's if-branch; on throw the else-branch runs with the
// error value bound to errName. The guard variable is pre-initialised to none
// so it has a safe default even if the against block doesn't assign it.
func (g *generator) emitGuard(s *syntax.GuardStmt) {
	varName := g.mangledName(s.VarName)
	errName := g.mangledName(s.ErrorName)

	g.emitLine("    MonkValue %s = monk_none();\n", varName)
	g.emitLine("    {\n")
	g.emitLine("        MonkGuardContext _guard_ctx;\n")
	g.emitLine("        if (monk_guard_begin(&_guard_ctx) == 0) {\n")
	g.emitLine("            %s = %s;\n", varName, g.emitExpr(s.Expr))
	g.emitLine("            monk_guard_end(&_guard_ctx);\n")
	g.emitLine("        } else {\n")
	g.emitLine("            MonkValue %s = monk_current_error();\n", errName)
	for _, stmt := range s.Against.Stmts {
		g.emitStmt(stmt)
	}
	g.emitLine("            (void)%s;\n", errName)
	g.emitLine("        }\n")
	g.emitLine("    }\n")
}
