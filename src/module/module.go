// Package module resolves multi-file Monk programs into a dependency graph.
// Given an entry .monk file, it recursively discovers all imported modules,
// detects circular imports, collects export names, and produces a topological
// ordering suitable for type-checking and code generation.
package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Module represents one parsed .monk file in the dependency graph.
type Module struct {
	Path     string          // absolute file path (canonical key)
	AST      *syntax.Program // parsed AST
	Exports  map[string]bool // names declared via `export` (populated from AST walk)
	DepPaths []string        // absolute paths of modules this one imports
}

// Graph is the complete set of modules reachable from the entry file.
type Graph struct {
	Entry   string             // absolute path of entry module
	Modules map[string]*Module // path -> Module
	Order   []string           // topological: deps first, entry last
}

// Build resolves, parses, and topologically sorts all modules reachable from
// entryPath. Returns an error on circular imports, missing files, or missing
// exports.
//
//	graph, err := module.Build("main.monk")
//	// graph.Order = ["utils.monk", "math.monk", "main.monk"]
func Build(entryPath string) (*Graph, error) {
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve entry path: %w", err)
	}

	g := &Graph{
		Entry:   absEntry,
		Modules: make(map[string]*Module),
	}

	// DFS state: gray = currently visiting (cycle detection), black = done.
	gray := make(map[string]bool)
	black := make(map[string]bool)

	// chain tracks the import path for cycle error messages.
	// e.g. ["main.monk", "utils.monk", "math.monk"]
	var chain []string

	var visit func(path string) error
	visit = func(path string) error {
		if black[path] {
			return nil // already fully processed
		}
		if gray[path] {
			// Build cycle description: from the first occurrence of path in
			// chain to the current path.
			// e.g. "circular import: a.monk -> b.monk -> a.monk"
			start := 0
			for i, p := range chain {
				if p == path {
					start = i
					break
				}
			}
			// Use append([]string(nil), ...) to make a clean copy of the slice
			// before appending. chain[start:] shares the backing array with chain,
			// so a plain append could overwrite chain past its logical end if
			// capacity exists — safe here (we return immediately), but misleading.
			cycle := append(append([]string(nil), chain[start:]...), path)
			names := make([]string, len(cycle))
			for i, p := range cycle {
				names[i] = filepath.Base(p)
			}
			return fmt.Errorf("circular import: %s", strings.Join(names, " -> "))
		}

		gray[path] = true
		chain = append(chain, path)
		defer func() {
			chain = chain[:len(chain)-1]
			delete(gray, path)
			black[path] = true
		}()

		// Read and parse the module.
		source, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read module %q: %w", filepath.Base(path), err)
		}
		prog, err := syntax.Parse(string(source))
		if err != nil {
			return fmt.Errorf("parse error in %s: %w", filepath.Base(path), err)
		}

		mod := &Module{
			Path:    path,
			AST:     prog,
			Exports: collectExportNames(prog),
		}

		// Resolve imports and validate exported names.
		for _, stmt := range prog.Stmts {
			use, ok := stmt.(*syntax.UseStmt)
			if !ok {
				continue
			}
			depPath, err := ResolvePath(use.Source, path)
			if err != nil {
				return fmt.Errorf("%s:%d: %w", filepath.Base(path), use.Line, err)
			}

			// Recurse into the dependency first so its exports are available.
			if err := visit(depPath); err != nil {
				return err
			}

			dep := g.Modules[depPath]

			// Validate imported names exist in the dependency's exports.
			if !use.Star {
				for _, name := range use.Names {
					importName := name
					if use.Alias != "" && len(use.Names) == 1 {
						importName = use.Names[0] // the original name, not alias
					}
					if !dep.Exports[importName] {
						return fmt.Errorf("%s:%d: module %q does not export %q",
							filepath.Base(path), use.Line, use.Source, importName)
					}
				}
			}

			mod.DepPaths = append(mod.DepPaths, depPath)
		}

		g.Modules[path] = mod
		g.Order = append(g.Order, path) // post-order = deps before dependents
		return nil
	}

	if err := visit(absEntry); err != nil {
		return nil, err
	}
	return g, nil
}

// ResolvePath resolves a module source string relative to the importing file.
//
//	ResolvePath("./utils", "/project/main.monk") -> "/project/utils.monk"
//	ResolvePath("./lib/math", "/project/main.monk") -> "/project/lib/math.monk"
//	ResolvePath("./utils.monk", "/project/main.monk") -> "/project/utils.monk"
func ResolvePath(source, importerPath string) (string, error) {
	if !strings.HasPrefix(source, ".") {
		return "", fmt.Errorf("module path %q must be relative (start with '.' or '..')", source)
	}

	// REVIEW-SKIP: No project-root boundary check for ../ traversal. Intentional:
	// Monk's threat model is a local developer on their own machine (see security.md).
	// ../ imports are a supported feature (tested: TestRunModuleParentDirectory).
	// Revisit if/when Monk gains sandboxed or server-side execution.
	dir := filepath.Dir(importerPath)
	resolved := filepath.Join(dir, source)

	// Append .monk extension if not already present.
	if !strings.HasSuffix(resolved, ".monk") {
		resolved += ".monk"
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("cannot resolve module path %q: %w", source, err)
	}

	// Verify the file exists.
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("module %q not found (resolved to %s)", source, filepath.Base(abs))
	}

	return abs, nil
}

// collectExportNames walks a program's AST and returns the set of names that
// are exported via `export` statements.
//
//	export let helper = ...  -> "helper"
//	export PI                -> "PI"   (ExprStmt wrapping IdentExpr)
//	export type Point = ...  -> "Point"
func collectExportNames(prog *syntax.Program) map[string]bool {
	exports := make(map[string]bool)
	for _, stmt := range prog.Stmts {
		es, ok := stmt.(*syntax.ExportStmt)
		if !ok {
			continue
		}
		switch inner := es.Stmt.(type) {
		case *syntax.VarDeclStmt:
			exports[inner.Name] = true
		case *syntax.TypeDeclStmt:
			exports[inner.Name] = true
		case *syntax.ExprStmt:
			// Bare "export name" — the inner expression is an IdentExpr.
			if id, ok := inner.Expr.(*syntax.IdentExpr); ok {
				exports[id.Name] = true
			}
		}
	}
	return exports
}
