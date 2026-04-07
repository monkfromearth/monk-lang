package codegen

import (
	"fmt"

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
	case *syntax.TypeDeclStmt:
		// Type declarations are compile-time only — no C code to emit.
	default:
		g.emitLine("    /* TODO: unhandled statement %T */\n", stmt)
	}
}

// emitVarDeclLine emits a single C variable declaration. In module mode it splits
// the declaration into a file-scope static global + an init-body assignment:
//
//	forModule=false: "    TYPE name = expr;\n"  → g.body  (stack-local)
//	forModule=true:  "static TYPE name;\n"       → g.globals (file scope)
//	                 "    name = expr;\n"         → g.body    (init assignment)
//
// This replaces the text-parsing approach in the old emitModuleVarDecl.
func (g *generator) emitVarDeclLine(name, typeName, expr string, forModule bool) {
	if forModule {
		fmt.Fprintf(&g.globals, "static %s %s;\n", typeName, name)
		g.emitLine("    %s = %s;\n", name, expr)
	} else {
		g.emitLine("    %s %s = %s;\n", typeName, name, expr)
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
		if !forModule && g.stackFuncDecls[s] {
			g.stackFuncValues[s] = g.emitStackFuncValueNamed(cFuncName, fnExpr)
			return
		}
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
						g.emitVarDeclLine(name, "MonkValue", "monk_range_int("+nCode+")", forModule)
					} else {
						g.emitVarDeclLine(name, "MonkValue", fmt.Sprintf("monk_range_int((%s).int_val)", g.emitExpr(call.Args[0])), forModule)
					}
					g.markArrayUniquenessFromExpr(name, store, s.Value)
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
					g.emitVarDeclLine(name, "MonkValue", fmt.Sprintf("monk_fill_bool(%s, %s)", nCode, valCode), forModule)
					g.markArrayUniquenessFromExpr(name, store, s.Value)
					g.recordArrayLen(s.Name, s.Value)
					return
				case store == storeIntArray && valStore == storeInt:
					g.emitVarDeclLine(name, "MonkValue", fmt.Sprintf("monk_fill_int(%s, %s)", nCode, valCode), forModule)
					g.markArrayUniquenessFromExpr(name, store, s.Value)
					g.recordArrayLen(s.Name, s.Value)
					return
				case store == storeFloatArray && valStore == storeFloat:
					g.emitVarDeclLine(name, "MonkValue", fmt.Sprintf("monk_fill_float(%s, %s)", nCode, valCode), forModule)
					g.markArrayUniquenessFromExpr(name, store, s.Value)
					g.recordArrayLen(s.Name, s.Value)
					return
				}
			}
		}
		value := g.emitExpr(s.Value)
		convFn := arrayConvFunc(store)
		g.emitVarDeclLine(name, "MonkValue", fmt.Sprintf("%s(%s)", convFn, value), forModule)
		g.markArrayUniquenessFromExpr(name, store, s.Value)
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
				g.emitVarDeclLine(name, cTypeName(store), rhsCode, forModule)
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
		init := "monk_deep_copy(" + rhsCode + ")"
		if isFreshValueExpr(s.Value) {
			// Move fresh temporaries into the binding instead of copying them.
			// Pass: `let s = to_upper_case(base)` avoids a duplicate string.
			// Fail: `let b = a` still uses deep_copy for value semantics.
			init = rhsCode
		}
		g.emitVarDeclLine(name, "MonkValue", init, forModule)
		return
	}

	// ── Pure scalar unboxed path ─────────────────────────────────────────
	g.storage[name] = store
	rhsCode, rhsStore := g.emitExprTyped(s.Value)
	init := coerce(rhsCode, rhsStore, store)
	g.emitVarDeclLine(name, cTypeName(store), init, forModule)
	g.recordConst(s.Name, s.Value) // bounds-check elision
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
					ensureFn := arrayEnsureFunc(objSt)
					idxCode, idxKind := g.emitExprTyped(indexTarget.Index)
					idxC := coerce(idxCode, idxKind, storeInt)
					rhsCode, rhsSt := g.emitExprTyped(s.Value)
					elemCode := coerce(rhsCode, rhsSt, elemSt)
					// Bounds-check elision: skip runtime check when statically provable.
					// idxC is a pure arithmetic expression (range analysis proved it),
					// so inlining it directly lets the compiler hoist and vectorize.
					if g.constVals != nil && g.isBoundedSafe(identObj, indexTarget.Index) {
						// Copy-on-write barrier for typed-array direct writes.
						// Pass: `let b = a; b[0]=99` detaches b before writing.
						// Fail: writing b mutates shared a backing storage.
						if g.arrayUnique[objName] {
							g.emitLine("    %s.%s->data[%s] = %s;\n", objName, ptrField, idxC, elemCode)
						} else {
							g.emitLine("    %s(&%s); %s.%s->data[%s] = %s;\n", ensureFn, objName, objName, ptrField, idxC, elemCode)
						}
					} else {
						tidx := g.newTemp()
						if g.arrayUnique[objName] {
							g.emitLine("    { int64_t %s = %s; if (%s < 0 || %s >= %s.%s->length) monk_panic(\"index out of bounds\"); %s.%s->data[%s] = %s; }\n",
								tidx, idxC, tidx, tidx, objName, ptrField, objName, ptrField, tidx, elemCode)
						} else {
							g.emitLine("    { int64_t %s = %s; if (%s < 0 || %s >= %s.%s->length) monk_panic(\"index out of bounds\"); %s(&%s); %s.%s->data[%s] = %s; }\n",
								tidx, idxC, tidx, tidx, objName, ptrField, ensureFn, objName, objName, ptrField, tidx, elemCode)
						}
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

	if rhsExpr := g.stringAppendRHS(s); rhsExpr != nil {
		target := s.Target.(*syntax.IdentExpr)
		name := g.mangledName(target.Name)
		rhs := g.emitExpr(rhsExpr)
		// In-place string concat assignment.
		// Pass: `s = s + "x"` / `s += "x"` grows s directly.
		// Fail: numeric `x += 1` must still use arithmetic assignment.
		g.emitLine("    monk_string_append_in_place(&%s, %s);\n", name, rhs)
		return
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
			g.markArrayUniquenessFromExpr(name, g.varStorage(name), s.Value)
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

// Control-flow emitters live in gen_flow.go.
