// Package codegen generates C source code from a Monk AST.
//
// The generated C #includes runtime.h and calls runtime functions for all
// Monk operations. The output is a complete, compilable .c file.
//
// The implementation is split across several files:
//   - gen.go         — this file: entry points, generator struct, driver loop
//   - gen_stmt.go    — statement emission (var decls, assignments, control flow)
//   - gen_expr.go    — expression emission (boxed MonkValue path)
//   - gen_func.go    — function hoisting + call lowering
//   - gen_helpers.go  — small utilities (mangleName, cString, compoundToArith, builtinMap)
//   - unbox.go        — scalar-unboxing path (raw int64_t/double/bool codegen)
//   - gen_access.go   — typed-array and record-field access fast paths
//   - gen_optimize.go — small typed optimization detectors (COW, string append, known-type builtins)
//   - gen_escape.go   — conservative escape analysis for stack-allocated closures
//   - gen_bounds.go   — static bounds analysis for bounds-check elision on typed arrays
//   - capture.go      — free-variable analysis for closure capture
package codegen

import (
	"fmt"
	"maps"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// Generate takes a parsed Monk program and returns a complete C source string.
// The filename is used for #line directives in the generated C.
//
// Unboxing: when called via GenerateWithTypes, the generator uses the type
// Info to emit raw C scalars (int64_t, double, bool) for scalar-typed
// variables instead of boxing everything as MonkValue. This is a pure
// performance win with no semantic change — the generated binary computes
// the same answer.
func Generate(prog *syntax.Program, filename string) string {
	return GenerateWithTypes(prog, filename, nil)
}

// GenerateWithTypes is like Generate but threads the type checker's Info
// through so scalar variables can be emitted unboxed.
func GenerateWithTypes(prog *syntax.Program, filename string, info *types.Info) string {
	g := &generator{
		filename:        filename,
		funcCount:       0,
		tmpCount:        0,
		funcNames:       make(map[string]string),
		funcDefaults:    make(map[string][]syntax.Expr),
		funcHasCapture:  make(map[string]bool),
		info:            info,
		storage:         make(map[string]storageKind),
		arrayUnique:     make(map[string]bool),
		fnStorage:       make(map[string]funcStorage),
		stackFuncValues: make(map[*syntax.VarDeclStmt]stackFuncInfo),
	}
	if info != nil {
		g.initBounds()
	}
	g.stackFuncDecls, g.stackFuncCalls = analyzeStackFuncDecls(prog)
	return g.generate(prog)
}

type generator struct {
	filename       string
	funcCount      int                      // counter for unique function names
	tmpCount       int                      // counter for unique temp variable names
	funcNames      map[string]string        // maps Monk variable name → hoisted C function name
	funcDefaults   map[string][]syntax.Expr // maps C function name → default exprs (nil for required params)
	funcHasCapture map[string]bool          // true if the C function has captures (needs monk_call)
	funcs          strings.Builder          // collected function definitions (hoisted above main)
	body           strings.Builder          // main body statements
	// Unboxing support — nil when Generate was called without type info.
	info            *types.Info            // per-expression types from the checker
	storage         map[string]storageKind // per-variable storage decision (Monk name → kind)
	arrayUnique     map[string]bool        // typed-array vars proven unshared; false/absent means emit COW barrier
	fnStorage       map[string]funcStorage // per-Monk-function storage decision (Monk name → params/ret)
	retStorage      storageKind            // expected return storage of the current function body
	currentCaptures []string               // capture variable names for the function being emitted (empty = no closure)
	stackFuncDecls  map[*syntax.VarDeclStmt]bool
	stackFuncCalls  map[*syntax.CallExpr]*syntax.VarDeclStmt
	stackFuncValues map[*syntax.VarDeclStmt]stackFuncInfo
	// Bounds-check elision — populated only when info != nil.
	constVals map[string]int64    // compile-time constant variable values (e.g. let N = 400)
	arrayLens map[string]int64    // statically known lengths of typed array variables
	varBounds map[string][2]int64 // inclusive [lo, hi] bounds for while-loop counters
	// Module system — populated only when GenerateModules is used.
	modulePrefix string            // "" for entry module, "m0_"/"m1_" for imports
	importMap    map[string]string // Monk name -> foreign C variable name (for imported non-function values)
	// Stack closure capture cleanup — heap values inside stack MonkValue[] arrays
	// must be freed when the enclosing scope exits, otherwise loops leak per iteration.
	// Each entry is a (capArrayName, capCount) pair pushed by emitStackFuncValueNamed.
	pendingCapCleanups []capCleanup
	// moduleInit is true for non-entry modules: variable declarations are split
	// into static globals (in g.globals) and assignments (in g.body/init function).
	moduleInit bool
	globals    strings.Builder // static global variable declarations (module mode only)
}

type capCleanup struct {
	arrayName string
	count     int
}

// funcStorage captures the unboxed C signature of a Monk function, so call
// sites can pass raw scalars and consume raw return values instead of
// boxing/unboxing. A function is fully unboxed only when every param AND
// the return type qualifies as a scalar storage kind.
type funcStorage struct {
	Params []storageKind
	Return storageKind
	All    bool // true when every param + return is scalar (fully unboxed)
}

// varStorage returns the storage kind for a mangled variable name, or
// storeBoxed if unknown (defensive default — always correct, just slower).
func (g *generator) varStorage(monkName string) storageKind {
	if k, ok := g.storage[monkName]; ok {
		return k
	}
	return storeBoxed
}

// REVIEW-SKIP: arrayUnique IS saved and restored across scope boundaries —
// it is a field in this struct and copied in saveStorage/restoreStorage below.
type generatorSnapshot struct {
	storage        map[string]storageKind
	arrayUnique    map[string]bool
	capCleanupMark int // len(pendingCapCleanups) at snapshot time
}

// saveStorage takes a snapshot of g.storage and flow-sensitive optimization
// facts that can be restored later.
// Used around scope boundaries (function bodies, loop bodies, if/else
// branches) so a `let x = ...` inside a nested scope that shadows an outer
// `x` doesn't leak its storage decision to the outer scope on exit.
//
// The returned value is an opaque snapshot; pass it to restoreStorage.
func (g *generator) saveStorage() generatorSnapshot {
	snap := generatorSnapshot{
		storage:        make(map[string]storageKind, len(g.storage)),
		arrayUnique:    make(map[string]bool, len(g.arrayUnique)),
		capCleanupMark: len(g.pendingCapCleanups),
	}
	maps.Copy(snap.storage, g.storage)
	maps.Copy(snap.arrayUnique, g.arrayUnique)
	return snap
}

// restoreStorage replaces g.storage with a previously saved snapshot,
// discarding any storage decisions made since the snapshot was taken.
func (g *generator) restoreStorage(snap generatorSnapshot) {
	// Emit cleanup for stack closure captures allocated since the snapshot.
	// Without this, heap values (strings, arrays) inside stack MonkValue[]
	// capture arrays leak every time the scope re-enters (e.g. loop iterations).
	// Pass: `while ... { let f=(x){x+name}; f(1) }` frees deep-copied name each iter.
	// Fail: omitting cleanup leaks one deep-copy per captured heap value per iteration.
	for i := snap.capCleanupMark; i < len(g.pendingCapCleanups); i++ {
		c := g.pendingCapCleanups[i]
		g.emitLine("    for (int _ci = 0; _ci < %d; _ci++) monk_free(%s[_ci]);\n", c.count, c.arrayName)
	}
	g.pendingCapCleanups = g.pendingCapCleanups[:snap.capCleanupMark]

	restoredUnique := make(map[string]bool, len(snap.arrayUnique))
	maps.Copy(restoredUnique, snap.arrayUnique)
	for name, nowUnique := range g.arrayUnique {
		thenUnique, existed := snap.arrayUnique[name]
		if existed && nowUnique != thenUnique {
			// Restore conservatively for flow-sensitive COW facts.
			// Pass: shadowed inner `arr` cannot leave stale true on outer `arr`.
			// Fail: branch assignment from shared array restores old true and skips detach.
			restoredUnique[name] = false
		}
	}
	g.storage = snap.storage
	g.arrayUnique = restoredUnique
}

// newTemp returns a unique C temp variable name.
func (g *generator) newTemp() string {
	g.tmpCount++
	return fmt.Sprintf("_tmp_%d", g.tmpCount)
}

// generate drives the whole emission: header, statements (which hoist
// functions as a side-effect), then wraps the body in main().
func (g *generator) generate(prog *syntax.Program) string {
	var out strings.Builder

	// Header with #line for source mapping
	out.WriteString("/* Generated by Monk Lang compiler */\n")
	// cString() handles escaping for Windows paths, quotes, etc.
	fmt.Fprintf(&out, "#line 1 %s\n", cString(g.filename))
	out.WriteString("#include \"runtime.h\"\n")
	out.WriteString("#include <stdlib.h>\n")
	out.WriteString("#include <string.h>\n")
	out.WriteString("#include <math.h>\n\n")

	// Emit all statements (functions get hoisted, rest goes to main body)
	for _, stmt := range prog.Stmts {
		g.emitStmt(stmt)
	}

	// Emit hoisted functions
	out.WriteString(g.funcs.String())

	// Emit main
	out.WriteString("int main(void) {\n")
	out.WriteString(g.body.String())
	out.WriteString("    return 0;\n")
	out.WriteString("}\n")

	return out.String()
}
