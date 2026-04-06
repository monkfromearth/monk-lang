package syntax

// Statement parsers. Each function consumes tokens from the parser's current
// position and returns a typed Stmt. Blocks are handled by parseBlock, which
// is the only parser that consumes matching `{` ... `}`.

// parseStmt dispatches to the appropriate statement parser based on the
// current token. break/continue legality is checked here against loopDepth.
func (p *Parser) parseStmt() (Stmt, error) {
	switch p.current().Kind {
	case Let, Const:
		return p.parseVarDecl()
	case If:
		return p.parseIf()
	case While:
		return p.parseWhile()
	case For:
		return p.parseFor()
	case Return:
		return p.parseReturn()
	case Break:
		if p.loopDepth == 0 {
			return nil, p.error("'break' can only be used inside a loop")
		}
		pos := p.currentPos()
		p.advance()
		return &BreakStmt{Pos: pos}, nil
	case Continue:
		if p.loopDepth == 0 {
			return nil, p.error("'continue' can only be used inside a loop")
		}
		pos := p.currentPos()
		p.advance()
		return &ContinueStmt{Pos: pos}, nil
	case Guard:
		return p.parseGuard()
	case Type:
		return p.parseTypeDecl()
	case Use:
		return p.parseUse()
	case Export:
		return p.parseExport()
	// Design decision: bare { at statement level is a record expression, not a block.
	// Blocks only exist inside if/while/for/guard/function bodies (parseBlock is
	// called explicitly by those parsers). This avoids the block-vs-record ambiguity
	// without fragile peek-ahead heuristics. Standalone blocks removed from language.
	default:
		return p.parseExprOrAssignStmt()
	}
}

// parseExprOrAssignStmt parses either an expression statement or an assignment statement.
// Design decision: assignment is a STATEMENT, not an expression.
// It cannot appear inside conditions, function args, or other expressions.
func (p *Parser) parseExprOrAssignStmt() (Stmt, error) {
	pos := p.currentPos()
	expr, err := p.parseOr() // parse left side (NOT full parseExpr, to avoid nested assignment)
	if err != nil {
		return nil, err
	}

	// Check for assignment operator
	if isAssignOp(p.current().Kind) {
		op := p.current().Kind
		p.advance()
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &AssignStmt{Pos: pos, Target: expr, Op: op, Value: value}, nil
	}

	return &ExprStmt{Pos: pos, Expr: expr}, nil
}

// parseVarDecl parses `let name [type] = expr` or `const name [type] = expr`.
// The optional type annotation is attempted via tryParseTypeExpr, which rolls
// back cleanly if the next token is not a valid type name.
func (p *Parser) parseVarDecl() (*VarDeclStmt, error) {
	pos := p.currentPos()
	isConst := p.current().Kind == Const
	p.advance() // skip let/const

	if p.current().Kind != Identifier {
		return nil, p.error("expected identifier after %s", map[bool]string{true: "const", false: "let"}[isConst])
	}
	name := p.current().Text
	p.advance()

	// Optional type annotation
	var typeAnnotation *TypeExpr
	if p.current().Kind == Identifier && p.current().Kind != Equal {
		te, ok := p.tryParseTypeExpr()
		if ok {
			typeAnnotation = &te
		}
	}

	if p.current().Kind != Equal {
		return nil, p.error("expected '=' in variable declaration")
	}
	p.advance() // skip =

	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &VarDeclStmt{
		Pos:     pos,
		Name:    name,
		Type:    typeAnnotation,
		Value:   value,
		IsConst: isConst,
	}, nil
}

// parseIf parses `if expr { ... } [else { ... } | else if ...]` recursively.
func (p *Parser) parseIf() (*IfStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'if'

	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	then, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	var elseStmt Stmt
	if p.current().Kind == Else {
		p.advance()
		if p.current().Kind == If {
			elseIf, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			elseStmt = elseIf
		} else {
			block, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			elseStmt = block
		}
	}

	return &IfStmt{Pos: pos, Condition: condition, Then: then, Else: elseStmt}, nil
}

// parseWhile parses `while expr { ... }`, bumping loopDepth so nested
// break/continue are permitted inside the body.
func (p *Parser) parseWhile() (*WhileStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'while'

	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	p.loopDepth++
	body, err := p.parseBlock()
	p.loopDepth--
	if err != nil {
		return nil, err
	}

	return &WhileStmt{Pos: pos, Condition: condition, Body: body}, nil
}

// parseFor parses `for varName in expr { ... }`, bumping loopDepth so
// break/continue are valid inside the body.
func (p *Parser) parseFor() (*ForStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'for'

	if p.current().Kind != Identifier {
		return nil, p.error("expected identifier after 'for'")
	}
	varName := p.current().Text
	p.advance()

	if p.current().Kind != In {
		return nil, p.error("expected 'in' after for variable")
	}
	p.advance()

	iterable, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	p.loopDepth++
	body, err := p.parseBlock()
	p.loopDepth--
	if err != nil {
		return nil, err
	}

	return &ForStmt{Pos: pos, VarName: varName, Iterable: iterable, Body: body}, nil
}

// parseReturn parses `return [expr]`. The value is optional: a bare `return`
// at end-of-input, before `}`, or before another statement keyword is a
// value-less return.
func (p *Parser) parseReturn() (*ReturnStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'return'

	if p.atEnd() || p.current().Kind == RightBrace || p.isStmtStart() {
		return &ReturnStmt{Pos: pos, Value: nil}, nil
	}

	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Pos: pos, Value: value}, nil
}

// parseGuard parses `guard varName = expr against errName { ... }`.
// The guard evaluates expr inside a setjmp context; if it throws, the
// against block runs with errName bound to the thrown value.
func (p *Parser) parseGuard() (*GuardStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'guard'

	if p.current().Kind != Identifier {
		return nil, p.error("expected identifier after 'guard'")
	}
	varName := p.current().Text
	p.advance()

	if p.current().Kind != Equal {
		return nil, p.error("expected '=' after guard variable")
	}
	p.advance()

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != Against {
		return nil, p.error("expected 'against' in guard statement")
	}
	p.advance()

	if p.current().Kind != Identifier {
		return nil, p.error("expected error variable name after 'against'")
	}
	errorName := p.current().Text
	p.advance()

	block, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &GuardStmt{
		Pos:       pos,
		VarName:   varName,
		Expr:      expr,
		ErrorName: errorName,
		Against:   block,
	}, nil
}

// parseTypeDecl parses `type Name = typeDef`, creating a named type alias or
// record-type definition.
func (p *Parser) parseTypeDecl() (*TypeDeclStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'type'

	if p.current().Kind != Identifier {
		return nil, p.error("expected type name after 'type'")
	}
	name := p.current().Text
	p.advance()

	if p.current().Kind != Equal {
		return nil, p.error("expected '=' after type name")
	}
	p.advance()

	def, err := p.parseTypeDef()
	if err != nil {
		return nil, err
	}

	return &TypeDeclStmt{Pos: pos, Name: name, Definition: def}, nil
}

// parseUse handles all four import forms:
//
//	use X from "path"
//	use X as Y from "path"
//	use { X, Y } from "path"
//	use * from "path"
func (p *Parser) parseUse() (*UseStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'use'

	u := &UseStmt{Pos: pos}

	// use * from "..."
	if p.current().Kind == Star {
		u.Star = true
		p.advance()
		if p.current().Kind != From {
			return nil, p.error("expected 'from' after '*'")
		}
		p.advance()
		if p.current().Kind != StringLiteral {
			return nil, p.error("expected module path string after 'from'")
		}
		u.Source = p.current().Text
		p.advance()
		return u, nil
	}

	// use { X, Y } from "..."
	if p.current().Kind == LeftBrace {
		p.advance()
		for p.current().Kind != RightBrace && !p.atEnd() {
			if p.current().Kind != Identifier {
				return nil, p.error("expected identifier in import list")
			}
			u.Names = append(u.Names, p.current().Text)
			p.advance()
			if p.current().Kind == Comma {
				p.advance()
			}
		}
		if p.current().Kind != RightBrace {
			return nil, p.error("expected '}'")
		}
		p.advance()
		if p.current().Kind != From {
			return nil, p.error("expected 'from' after import list")
		}
		p.advance()
		if p.current().Kind != StringLiteral {
			return nil, p.error("expected module path string after 'from'")
		}
		u.Source = p.current().Text
		p.advance()
		return u, nil
	}

	// use X from "..." or use X as Y from "..."
	if p.current().Kind != Identifier {
		return nil, p.error("expected identifier, '{', or '*' after 'use'")
	}
	firstName := p.current().Text
	p.advance()

	if p.current().Kind == As {
		p.advance()
		if p.current().Kind != Identifier {
			return nil, p.error("expected alias name after 'as'")
		}
		u.Alias = p.current().Text
		p.advance()
	}

	u.Names = []string{firstName}

	if p.current().Kind != From {
		return nil, p.error("expected 'from' in use statement")
	}
	p.advance()

	if p.current().Kind != StringLiteral {
		return nil, p.error("expected module path string after 'from'")
	}
	u.Source = p.current().Text
	p.advance()

	return u, nil
}

// parseExport wraps any statement in an ExportStmt, marking it as part of the
// module's public surface. Only valid at the top level (enforced by the module
// system in a later phase).
func (p *Parser) parseExport() (*ExportStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'export'

	stmt, err := p.parseStmt()
	if err != nil {
		return nil, err
	}

	return &ExportStmt{Pos: pos, Stmt: stmt}, nil
}

// parseBlock consumes `{ stmt* }` and returns a BlockStmt. It is the only
// parser that handles matching braces — individual statement parsers call
// parseBlock rather than consuming braces themselves.
func (p *Parser) parseBlock() (*BlockStmt, error) {
	pos := p.currentPos()
	if p.current().Kind != LeftBrace {
		return nil, p.error("expected '{'")
	}
	p.advance()

	var stmts []Stmt
	for p.current().Kind != RightBrace && !p.atEnd() {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}

	if p.current().Kind != RightBrace {
		return nil, p.error("expected '}'")
	}
	p.advance()

	return &BlockStmt{Pos: pos, Stmts: stmts}, nil
}
