# `src/types/` — static type checker

Entry: `Check(prog) (*Info, error)` in `checker.go`.
Multi-module: `CheckModules(graph) (*ModuleInfo, error)` — checks all modules in topological order, resolving cross-module imports.

The checker is a single AST walk that:
1. Resolves named type declarations (`type Point = ...`)
2. Hoists top-level function signatures (enables forward references + recursion)
3. Infers and validates every expression and statement

On success it returns `*Info`, which codegen uses to decide between raw C scalars
and the `MonkValue` tagged union.

## Files

| File           | Contains                                                                    |
| -------------- | --------------------------------------------------------------------------- |
| `types.go`     | `Kind` enum (9 kinds), `Type` struct, `AssignableTo`, constructors (`Int`, `ArrayOf`, …) |
| `checker.go`   | `Info` struct, `Check` entry, `CheckModules` + `ModuleInfo` (multi-module), `checker` + `scope` + `Binding`, `checkProgram` / `checkStmt` / `checkBlock`, builtin declarations |
| `stmts.go`     | Statement checks: `checkVarDecl`, `checkAssign`, `checkCompoundOp`, `checkIf`, `checkWhile`, `checkFor`, `checkReturn`, `checkGuard`, type resolver (`resolveTypeDef` / `resolveTypeExpr`) |
| `exprs.go`     | Expression inference: `inferExpr` / `inferExprInner`, one `infer*` function per expression kind, helper predicates (`isNumericOrAny`, `isIntOrAny`, `isNonComparable`) |
| `returns.go`   | All-paths-return analysis (`stmtsAlwaysReturn` / `stmtAlwaysReturns`) used to verify typed return functions |

## Key types

- **`Type`** — the single type representation. Has a `Kind` field (KindInt, KindStr, etc.),
  an optional `Optional` flag, an `Elem *Type` for arrays, `Fields []RecordTypeField` for records,
  and `Params []*Type` + `Return *Type` for functions. All types are pointers; singleton
  `Int`, `Float`, `Str`, `Bool`, `None`, `Any` are pre-allocated globals.

- **`Info`** — carries all type results back to codegen:
  - `Types map[Expr]*Type` — every expression node mapped to its type
  - `Decls map[*VarDeclStmt]*Type` — every variable declaration mapped to its resolved type
  - `Funcs map[*FuncExpr]*Type` — every function expression mapped to its full signature

- **`scope`** — singly-linked chain of `map[string]*Binding`. `declare` adds to the
  current frame; `lookup` walks the chain. `newScopeOf` creates a child frame.

## Adding a new type kind

1. Add the `Kind` constant to `types.go`.
2. Add a constructor or singleton as appropriate.
3. Handle the new kind in `AssignableTo` and `Equal`.
4. Handle it in the relevant `infer*` / `check*` functions.
5. Handle it in `codegen/unbox.go`'s `storageFor` if it has a raw C representation.
