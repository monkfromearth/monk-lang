# `src/syntax/` — lexer + parser + AST

Pipeline: **source text → Scanner → []Token → Parser → \*Program (AST)**

## Files

| File              | Contains                                                  |
| ----------------- | --------------------------------------------------------- |
| `token.go`        | `TokenKind`, `Token`, keyword table                       |
| `scanner.go`      | hand-written UTF-8 lexer (`Scan`)                         |
| `ast.go`          | `Pos`, `Node`/`Expr`/`Stmt` interfaces, `Program`, shared `TypeExpr`/`Param`/`TypeField`/`RecordField` |
| `ast_expr.go`     | expression nodes (`NumberExpr`, `BinaryExpr`, `FuncExpr`, …) |
| `ast_stmt.go`     | statement nodes (`VarDeclStmt`, `IfStmt`, `ForStmt`, …)   |
| `parser.go`       | `Parser` struct, entry points (`Parse`, `parseProgram`), token helpers |
| `parse_stmt.go`   | statement parsers (all 14 statement forms)                |
| `parse_expr.go`   | precedence-climbing expression parser + `parseParenOrFunc` disambiguator |
| `parse_type.go`   | type-expression parsers + `matchRightParen` nesting helper |

## Adding a new node type

1. Pick the right AST file (`ast_expr.go` or `ast_stmt.go`) and add the struct.
2. Add the `nodeKind()` + `exprNode()`/`stmtNode()` marker methods in the same file.
3. Add a parser in `parse_expr.go` or `parse_stmt.go`.
4. If introducing a new token, add it to `token.go`.
5. Handle the new node in `types/` and `codegen/`.

## Grammar notes

- Assignment is a **statement**, not an expression — prevents `if x = 5`.
- Parenthesized expressions vs function literals are disambiguated by looking
  at the token **inside** the `(`. See comments in `parseParenOrFunc`.
- `{` at statement start is always a record expression. Blocks only exist
  inside `if`/`while`/`for`/`guard`/function bodies (`parseBlock`).
