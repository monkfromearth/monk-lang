package codegen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/monkfromearth/monk-lang/module"
	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// GenerateModules produces a single C source string from a checked module graph.
// All modules are inlined into one .c file: non-entry modules become init
// functions with once-guards, the entry module becomes main().
//
//	graph, _ := module.Build("main.monk")
//	modInfo, _ := types.CheckModules(graph)
//	cSource := codegen.GenerateModules(graph, modInfo)
func GenerateModules(graph *module.Graph, modInfo *types.ModuleInfo) string {
	var out strings.Builder

	// Assign each non-entry module a numeric ID for name prefixing.
	// Entry module gets no prefix (keeps mk_NAME, _monk_func_N).
	modIndex := make(map[string]int)
	idx := 0
	for _, path := range graph.Order {
		if path != graph.Entry {
			modIndex[path] = idx
			idx++
		}
	}

	// Per-module export tracking, populated as we generate each module:
	// exportCNames[modPath][monkName]         = C variable name ("mk_m0_add")
	// exportCFuncNames[modPath][monkName]     = C function name ("_monk_m0_func_1")
	// exportFnStorage[modPath][monkName]      = funcStorage (for cross-module unboxed calls)
	// exportVarStorage[modPath][monkName]     = storageKind (so importers know int64_t vs MonkValue)
	// exportFuncHasCapture[modPath][cFuncName] = true if function closes over variables
	//   Required so importers emit monk_call(fn, args) instead of direct fn(args) —
	//   closures need _self passed implicitly; direct calls omit it → C compile error.
	exportCNames := make(map[string]map[string]string)
	exportCFuncNames := make(map[string]map[string]string)
	exportFnStorage := make(map[string]map[string]funcStorage)
	exportVarStorage := make(map[string]map[string]storageKind)
	exportFuncHasCapture := make(map[string]map[string]bool)
	exportFuncDefaults := make(map[string]map[string][]syntax.Expr)

	// Global function counter shared across all modules to avoid name collisions.
	globalFuncCount := 0

	type moduleOutput struct {
		globals string // static global variable declarations (module mode)
		funcs   string // hoisted function definitions
		body    string // module-level statements (init body for non-entry)
	}
	outputs := make(map[string]*moduleOutput)

	for _, modPath := range graph.Order {
		mod := graph.Modules[modPath]
		isEntry := modPath == graph.Entry

		// Module prefix: "" for entry, "m0_"/"m1_" for imports.
		var modPrefix string
		if !isEntry {
			modPrefix = fmt.Sprintf("m%d_", modIndex[modPath])
		}

		info := modInfo.Info[modPath]

		g := &generator{
			filename:        modPath,
			funcCount:       globalFuncCount,
			tmpCount:        0,
			funcNames:       make(map[string]string),
			funcDefaults:    make(map[string][]syntax.Expr),
			funcHasCapture:  make(map[string]bool),
			info:            info,
			storage:         make(map[string]storageKind),
			arrayUnique:     make(map[string]bool), // REVIEW-SKIP: initialized here — not a nil map. GenerateWithTypes does the same.

			fnStorage:       make(map[string]funcStorage),
			stackFuncValues: make(map[*syntax.VarDeclStmt]stackFuncInfo),
			modulePrefix:    modPrefix,
			importMap:       make(map[string]string),
			moduleInit:      !isEntry, // non-entry modules split vars into static globals + init body
		}
		if info != nil {
			g.initBounds()
		}
		g.stackFuncDecls, g.stackFuncCalls = analyzeStackFuncDecls(mod.AST)

		// Wire up imports from already-generated dependencies.
		for _, stmt := range mod.AST.Stmts {
			use, ok := stmt.(*syntax.UseStmt)
			if !ok {
				continue
			}
			// module.Build already validated this path; ignore the error (can't happen).
			depPath, _ := module.ResolvePath(use.Source, modPath)

			// Build the list of (localName, origName) pairs to import.
			type ie struct{ local, orig string }
			var imports []ie

			if use.Star {
				// REVIEW-SKIP: Go map iteration is non-deterministic, so star imports
				// are wired in arbitrary order. This is safe — each name is wired
				// independently into importMap/funcNames/storage, and ordering has no
				// effect on the generated C semantics. Sorting would add cost with
				// no correctness benefit.
				for name := range exportCNames[depPath] {
					imports = append(imports, ie{name, name})
				}
			} else if use.Alias != "" && len(use.Names) == 1 {
				imports = append(imports, ie{use.Alias, use.Names[0]})
			} else {
				for _, name := range use.Names {
					imports = append(imports, ie{name, name})
				}
			}

			for _, imp := range imports {
				// Wire function imports for direct call resolution.
				if cFuncName, ok := exportCFuncNames[depPath][imp.orig]; ok {
					g.funcNames[imp.local] = cFuncName
					// Copy fnStorage so unboxed calls work across module boundaries.
					if fs, ok2 := exportFnStorage[depPath][imp.orig]; ok2 {
						g.fnStorage[cFuncName] = fs
					}
					// Propagate closure flag so emitCall uses monk_call(fn, args)
					// instead of a direct fn(args) call. Closures need _self passed
					// implicitly; omitting it produces a C "too few arguments" error.
					// e.g. let n=10; let f=(x int) int { return x+n }; export f
					//   → _monk_m0_func_N(MonkFunction *_self, int64_t mk_x)
					//   → importer must emit: monk_call(mk_m0_f, args)
					if exportFuncHasCapture[depPath][cFuncName] {
						g.funcHasCapture[cFuncName] = true
					}
					// Propagate default parameter expressions so padDefaults can fill omitted
					// trailing args at call sites in the importing module.
					if defs, ok := exportFuncDefaults[depPath][cFuncName]; ok {
						g.funcDefaults[cFuncName] = defs
					}
				}
				// Wire variable imports so IdentExpr resolves to the foreign C name.
				if cName, ok := exportCNames[depPath][imp.orig]; ok {
					g.importMap[imp.local] = cName
					// Propagate storage kind so the importer knows if the
					// variable is int64_t/double/bool vs MonkValue.
					// e.g. imported `value` (int64_t) must be boxed before
					// passing to monk_to_string(MonkValue).
					if sk, ok2 := exportVarStorage[depPath][imp.orig]; ok2 {
						g.storage[cName] = sk
					}
				}
			}
		}

		// Emit all statements.
		for _, stmt := range mod.AST.Stmts {
			g.emitStmt(stmt)
		}

		globalFuncCount = g.funcCount

		outputs[modPath] = &moduleOutput{
			globals: g.globals.String(),
			funcs:   g.funcs.String(),
			body:    g.body.String(),
		}

		// Record this module's export C names and storage kinds for downstream importers.
		cNames := make(map[string]string)
		cFuncNames := make(map[string]string)
		fnStorageMap := make(map[string]funcStorage)
		varStorageMap := make(map[string]storageKind)
		hasCapture := make(map[string]bool)
		funcDefaultsMap := make(map[string][]syntax.Expr)
		// REVIEW-SKIP: Type-only exports (e.g. `export type Point = ...`) add entries
		// to exportCNames even though no C variable `mk_m0_Point` exists. Harmless —
		// the type checker prevents type names from appearing in value positions, so
		// importMap["Point"] is never consulted by any codegen expression emitter.
		for exportName := range mod.Exports {
			// Re-export check: if this name was imported (in importMap),
			// propagate the original C name instead of generating a new one.
			// e.g. proxy.monk imports secret from origin.monk, then exports it.
			// proxy's exportCNames["secret"] should be "mk_m0_secret" (origin's name).
			cVarName := "mk_" + modPrefix + exportName
			if imported, ok := g.importMap[exportName]; ok {
				cVarName = imported
			}
			cNames[exportName] = cVarName
			// Track variable storage so importers know if it's int64_t vs MonkValue.
			if sk, ok := g.storage[cVarName]; ok {
				varStorageMap[exportName] = sk
			}
			if cFuncName, ok := g.funcNames[exportName]; ok {
				cFuncNames[exportName] = cFuncName
				if fs, ok2 := g.fnStorage[cFuncName]; ok2 {
					fnStorageMap[exportName] = fs
				}
				// Record closure flag so importers can route through monk_call.
				if g.funcHasCapture[cFuncName] {
					hasCapture[cFuncName] = true
				}
				// Propagate default param expressions so importers can pad omitted args.
				// e.g. greet("world") omits suffix → padDefaults fills it with "!"
				if defs, ok := g.funcDefaults[cFuncName]; ok {
					funcDefaultsMap[cFuncName] = defs
				}
			}
		}
		exportCNames[modPath] = cNames
		exportCFuncNames[modPath] = cFuncNames
		exportFnStorage[modPath] = fnStorageMap
		exportVarStorage[modPath] = varStorageMap
		exportFuncHasCapture[modPath] = hasCapture
		exportFuncDefaults[modPath] = funcDefaultsMap
	}

	// Assemble the single .c file.

	// Header.
	out.WriteString("/* Generated by Monk Lang compiler */\n")
	fmt.Fprintf(&out, "#line 1 %s\n", cString(graph.Entry))
	out.WriteString("#include \"runtime.h\"\n")
	out.WriteString("#include <stdlib.h>\n")
	out.WriteString("#include <string.h>\n")
	out.WriteString("#include <math.h>\n\n")

	// Non-entry modules: hoisted functions + init function with once-guard.
	for _, modPath := range graph.Order {
		if modPath == graph.Entry {
			continue
		}
		mo := outputs[modPath]
		modID := modIndex[modPath]

		fmt.Fprintf(&out, "/* Module: %s */\n", filepath.Base(modPath))
		fmt.Fprintf(&out, "#line 1 %s\n", cString(modPath))
		// Static globals for module-level variables (visible from init + main).
		out.WriteString(mo.globals)
		out.WriteString(mo.funcs)

		// Init function: module-level code runs exactly once.
		// static void _mod_0_init(void) { static int _init=0; if(_init) return; ... }
		fmt.Fprintf(&out, "static void _mod_%d_init(void) {\n", modID)
		fmt.Fprintf(&out, "    static int _initialized = 0;\n")
		fmt.Fprintf(&out, "    if (_initialized) return;\n")
		fmt.Fprintf(&out, "    _initialized = 1;\n")
		out.WriteString(mo.body)
		out.WriteString("}\n\n")
	}

	// Entry module: hoisted functions + main().
	entryOutput := outputs[graph.Entry]
	fmt.Fprintf(&out, "/* Entry: %s */\n", filepath.Base(graph.Entry))
	fmt.Fprintf(&out, "#line 1 %s\n", cString(graph.Entry))
	out.WriteString(entryOutput.funcs)

	out.WriteString("int main(void) {\n")
	// Call dependency init functions in topological order before entry code.
	for _, modPath := range graph.Order {
		if modPath == graph.Entry {
			continue
		}
		fmt.Fprintf(&out, "    _mod_%d_init();\n", modIndex[modPath])
	}
	out.WriteString(entryOutput.body)
	out.WriteString("    return 0;\n")
	out.WriteString("}\n")

	return out.String()
}
