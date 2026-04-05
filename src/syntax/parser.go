package syntax

import "fmt"

// Parser transforms a token stream into an AST.
type Parser struct {
	tokens    []Token
	pos       int
	loopDepth int   // tracks nesting depth inside loops (for break/continue validation)
	typeErr   error // sticky error from parseTypeExpr (which doesn't return error)
}

// Parse tokenizes source and returns the AST, or an error.
func Parse(source string) (*Program, error) {
	tokens := Scan(source)
	p := &Parser{tokens: tokens, pos: 0}
	return p.parseProgram()
}

// pos returns the current token's position as a Pos.
func (p *Parser) currentPos() Pos {
	tok := p.current()
	return Pos{Line: tok.Line, Column: tok.Column}
}

func (p *Parser) parseProgram() (*Program, error) {
	prog := &Program{}
	for !p.atEnd() {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		if p.typeErr != nil {
			return nil, p.typeErr
		}
		prog.Stmts = append(prog.Stmts, stmt)
	}
	// One last check — parseStmt may have finished successfully but left a
	// type-annotation error behind.
	if p.typeErr != nil {
		return nil, p.typeErr
	}
	return prog, nil
}

// --- Statement parsing ---

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

func (p *Parser) tryParseTypeExpr() (TypeExpr, bool) {
	if p.current().Kind != Identifier {
		return TypeExpr{}, false
	}

	saved := p.pos
	name := p.current().Text
	p.advance()

	te := TypeExpr{Name: name}

	if p.current().Kind == LeftBracket && p.peek().Kind == RightBracket {
		te.IsArray = true
		p.advance()
		p.advance()
	}

	if p.current().Kind == Question {
		te.Optional = true
		p.advance()
	}

	if p.current().Kind == Equal {
		return te, true
	}

	p.pos = saved
	return TypeExpr{}, false
}

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

func (p *Parser) parseTypeDef() (TypeDefExpr, error) {
	if p.current().Kind == LeftBrace {
		return p.parseRecordTypeDef()
	}

	if p.current().Kind == Identifier {
		te := p.parseTypeExpr()
		return TypeDefExpr{AliasOf: &te}, nil
	}

	if p.current().Kind == LeftParen {
		te := p.parseTypeExpr()
		return TypeDefExpr{AliasOf: &te}, nil
	}

	return TypeDefExpr{}, p.error("expected type definition")
}

func (p *Parser) parseRecordTypeDef() (TypeDefExpr, error) {
	p.advance() // skip {

	var fields []TypeField
	for p.current().Kind != RightBrace && !p.atEnd() {
		if p.current().Kind != Identifier {
			return TypeDefExpr{}, p.error("expected field name")
		}
		name := p.current().Text
		p.advance()

		if p.current().Kind != Colon {
			return TypeDefExpr{}, p.error("expected ':' after field name")
		}
		p.advance()

		te := p.parseTypeExpr()
		fields = append(fields, TypeField{Name: name, Type: te})

		if p.current().Kind == Comma {
			p.advance()
		}
	}

	if p.current().Kind != RightBrace {
		return TypeDefExpr{}, p.error("expected '}'")
	}
	p.advance()

	return TypeDefExpr{Fields: fields}, nil
}

func isTypeName(k TokenKind) bool {
	return k == Identifier || k == None
}

// matchRightParen returns the index of the ')' that matches the '(' at
// tokens[openIdx], respecting nesting. Returns -1 if unbalanced.
func matchRightParen(tokens []Token, openIdx int) int {
	depth := 0
	for i := openIdx; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case LeftParen:
			depth++
		case RightParen:
			depth--
			if depth == 0 {
				return i
			}
		case Eof:
			return -1
		}
	}
	return -1
}

func (p *Parser) parseTypeExpr() TypeExpr {
	// Function type: (T, T) -> T
	if p.current().Kind == LeftParen {
		return p.parseFuncType()
	}

	name := p.current().Text
	p.advance()

	te := TypeExpr{Name: name}

	if p.current().Kind == LeftBracket && p.peek().Kind == RightBracket {
		te.IsArray = true
		p.advance()
		p.advance()
	}

	if p.current().Kind == Question {
		te.Optional = true
		p.advance()
	}

	return te
}

// parseFuncType parses a function type annotation: `(T1, T2) -> T3` or `() -> T`.
// The caller has confirmed current token is `(`. Trailing `?` makes the
// function type itself optional: `(int) -> int?` is ambiguous between
// "function returning int?" and "optional function returning int" — per
// existing Monk convention (types read left-to-right, modifiers trail),
// we bind `?` to the RETURN type, so `(int) -> int?` is the former. To get
// an optional function, parenthesize the return type first — but Monk has
// no syntax for that today, so we simply don't support optional-function
// types.
func (p *Parser) parseFuncType() TypeExpr {
	p.advance() // skip (
	var params []TypeExpr
	for p.current().Kind != RightParen && !p.atEnd() {
		params = append(params, p.parseTypeExpr())
		if p.current().Kind == Comma {
			p.advance()
		}
	}
	p.advance() // skip )
	// Require ` -> ReturnType`.
	if p.current().Kind != Arrow {
		// parseTypeExpr doesn't return error (legacy signature, many callers).
		// Record the failure on the parser's sticky-error field so parseProgram
		// reports it instead of returning a nonsense AST. Advance to avoid an
		// infinite loop if the caller retries.
		if p.typeErr == nil {
			p.typeErr = p.error("expected '->' after function-type parameters")
		}
		return TypeExpr{}
	}
	p.advance() // skip ->
	ret := p.parseTypeExpr()
	return TypeExpr{
		IsFunc:     true,
		FuncParams: params,
		FuncReturn: &ret,
	}
}

// Fix 5: Parse use { X, Y } from "..." and use * from "..."
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

func (p *Parser) parseExport() (*ExportStmt, error) {
	pos := p.currentPos()
	p.advance() // skip 'export'

	stmt, err := p.parseStmt()
	if err != nil {
		return nil, err
	}

	return &ExportStmt{Pos: pos, Stmt: stmt}, nil
}

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

// --- Expression parsing (precedence climbing) ---
// Assignment is NOT in the expression chain — it's handled as a statement.

func (p *Parser) parseExpr() (Expr, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Or || p.current().Kind == PipePipe {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	left, err := p.parseBitwiseOr()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == And || p.current().Kind == AmpAmp {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseBitwiseOr()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseOr() (Expr, error) {
	left, err := p.parseBitwiseXor()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Pipe {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseBitwiseXor()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseXor() (Expr, error) {
	left, err := p.parseBitwiseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Caret {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseAnd() (Expr, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Amp {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseEquality() (Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == EqualEqual || p.current().Kind == BangEqual || p.current().Kind == Is {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseComparison() (Expr, error) {
	left, err := p.parseShift()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Less || p.current().Kind == Greater ||
		p.current().Kind == LessEqual || p.current().Kind == GreaterEqual {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseShift() (Expr, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == ShiftLeft || p.current().Kind == ShiftRight {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAddSub() (Expr, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Plus || p.current().Kind == Minus {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMulDiv() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Star || p.current().Kind == Slash || p.current().Kind == Percent {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Pos: pos, Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	if p.current().Kind == Minus || p.current().Kind == Not || p.current().Kind == Bang || p.current().Kind == Tilde {
		pos := p.currentPos()
		op := p.current().Kind
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Pos: pos, Op: op, Operand: operand}, nil
	}

	if p.current().Kind == Throw {
		pos := p.currentPos()
		p.advance()
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &ThrowExpr{Pos: pos, Value: value}, nil
	}

	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		switch p.current().Kind {
		case LeftParen:
			pos := p.currentPos()
			p.advance()
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			expr = &CallExpr{Pos: pos, Callee: expr, Args: args}

		case LeftBracket:
			pos := p.currentPos()
			p.advance()
			index, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if p.current().Kind != RightBracket {
				return nil, p.error("expected ']'")
			}
			p.advance()
			expr = &IndexExpr{Pos: pos, Object: expr, Index: index}

		case Dot:
			pos := p.currentPos()
			p.advance()
			if p.current().Kind != Identifier {
				return nil, p.error("expected property name after '.'")
			}
			name := p.current().Text
			p.advance()
			expr = &PropertyExpr{Pos: pos, Object: expr, Property: name}

		default:
			return expr, nil
		}
	}
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.current()
	pos := p.currentPos()

	switch tok.Kind {
	case IntLiteral:
		p.advance()
		return &NumberExpr{Pos: pos, Value: tok.Text, IsInt: true}, nil

	case FloatLiteral:
		p.advance()
		return &NumberExpr{Pos: pos, Value: tok.Text, IsInt: false}, nil

	case StringLiteral:
		p.advance()
		return &StringExpr{Pos: pos, Value: tok.Text}, nil

	case TemplateLiteral:
		p.advance()
		return &TemplateExpr{Pos: pos, Value: tok.Text}, nil

	case True:
		p.advance()
		return &BoolExpr{Pos: pos, Value: true}, nil

	case False:
		p.advance()
		return &BoolExpr{Pos: pos, Value: false}, nil

	case None:
		p.advance()
		return &NoneExpr{Pos: pos}, nil

	case Identifier:
		p.advance()
		return &IdentExpr{Pos: pos, Name: tok.Text}, nil

	case LeftParen:
		return p.parseParenOrFunc()

	case LeftBracket:
		return p.parseArrayLiteral()

	case LeftBrace:
		return p.parseRecordLiteral()

	default:
		return nil, p.error("unexpected token: %s", tok.Kind)
	}
}

func (p *Parser) parseParenOrFunc() (Expr, error) {
	// Fix 4: Improved disambiguation.
	// Function pattern: (name type, ...) returnType { body }
	// Grouping pattern: (expr)
	//
	// Strategy: save position, try to detect the function pattern by looking
	// at what follows '('. If we see ')' followed by a type/brace, or
	// 'ident ident' (param + type), it's a function. Otherwise it's grouping.

	saved := p.pos
	p.advance() // skip (

	// () followed by type name or { → function with no params
	if p.current().Kind == RightParen {
		p.advance()
		if isTypeName(p.current().Kind) || p.current().Kind == LeftBrace {
			p.pos = saved
			return p.parseFuncExpr()
		}
		p.pos = saved
		p.advance()
		return nil, p.error("unexpected ')'")
	}

	// Look for function pattern: identifier followed by another identifier (param type)
	// This is unambiguous because in an expression, ident is never followed by ident
	// (it would be ident operator ident, or ident ( for a call, etc.)
	if p.current().Kind == Identifier {
		afterIdent := p.pos + 1
		if afterIdent < len(p.tokens) {
			next := p.tokens[afterIdent].Kind
			// ident followed by ident = param type pair → function
			if next == Identifier {
				p.pos = saved
				return p.parseFuncExpr()
			}
			// ident followed by ( is ambiguous:
			//   function literal with fn-type param: `(cb (int) -> int) ...`
			//   grouped expression with a call:      `(to_float(y) / 2.0)`
			// Disambiguate by scanning to the matching ')' of the inner paren
			// and checking what follows. Only `->` indicates a function-type param.
			if next == LeftParen {
				closeIdx := matchRightParen(p.tokens, afterIdent)
				if closeIdx >= 0 && closeIdx+1 < len(p.tokens) && p.tokens[closeIdx+1].Kind == Arrow {
					p.pos = saved
					return p.parseFuncExpr()
				}
				// Otherwise it's a grouped expression; fall through.
			}
		}
	}

	// It's a grouped expression: (expr)
	p.pos = saved
	p.advance() // skip (
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.current().Kind != RightParen {
		return nil, p.error("expected ')'")
	}
	p.advance()
	return expr, nil
}

func (p *Parser) parseFuncExpr() (Expr, error) {
	pos := p.currentPos()
	p.advance() // skip (

	var params []Param
	for p.current().Kind != RightParen && !p.atEnd() {
		if p.current().Kind != Identifier {
			return nil, p.error("expected parameter name")
		}
		paramName := p.current().Text
		p.advance()

		paramType := p.parseTypeExpr()

		var defaultVal Expr
		if p.current().Kind == Equal {
			p.advance()
			var err error
			defaultVal, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}

		params = append(params, Param{Name: paramName, Type: paramType, Default: defaultVal})

		if p.current().Kind == Comma {
			p.advance()
		}
	}

	if p.current().Kind != RightParen {
		return nil, p.error("expected ')' after parameters")
	}
	p.advance()

	var returnType TypeExpr
	if isTypeName(p.current().Kind) {
		returnType = p.parseTypeExpr()
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &FuncExpr{Pos: pos, Params: params, ReturnType: returnType, Body: body}, nil
}

func (p *Parser) parseArrayLiteral() (Expr, error) {
	pos := p.currentPos()
	p.advance() // skip [

	var elements []Expr
	for p.current().Kind != RightBracket && !p.atEnd() {
		elem, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		if p.current().Kind == Comma {
			p.advance()
		}
	}

	if p.current().Kind != RightBracket {
		return nil, p.error("expected ']'")
	}
	p.advance()

	return &ArrayExpr{Pos: pos, Elements: elements}, nil
}

func (p *Parser) parseRecordLiteral() (Expr, error) {
	pos := p.currentPos()
	p.advance() // skip {

	var fields []RecordField
	for p.current().Kind != RightBrace && !p.atEnd() {
		if p.current().Kind != Identifier {
			return nil, p.error("expected field name")
		}
		key := p.current().Text
		p.advance()

		if p.current().Kind != Colon {
			return nil, p.error("expected ':' after field name")
		}
		p.advance()

		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		fields = append(fields, RecordField{Key: key, Value: value})

		if p.current().Kind == Comma {
			p.advance()
		}
	}

	if p.current().Kind != RightBrace {
		return nil, p.error("expected '}'")
	}
	p.advance()

	return &RecordExpr{Pos: pos, Fields: fields}, nil
}

func (p *Parser) parseArgList() ([]Expr, error) {
	var args []Expr
	for p.current().Kind != RightParen && !p.atEnd() {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.current().Kind == Comma {
			p.advance()
		}
	}

	if p.current().Kind != RightParen {
		return nil, p.error("expected ')'")
	}
	p.advance()

	return args, nil
}

// --- Helpers ---

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: Eof}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Kind: Eof}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) atEnd() bool {
	return p.current().Kind == Eof
}

func (p *Parser) error(format string, args ...any) error {
	tok := p.current()
	return fmt.Errorf("line %d, column %d: %s", tok.Line, tok.Column, fmt.Sprintf(format, args...))
}

func (p *Parser) isStmtStart() bool {
	switch p.current().Kind {
	case Let, Const, If, While, For, Return, Break, Continue, Guard, Type, Use, Export:
		return true
	}
	return false
}

func isAssignOp(k TokenKind) bool {
	switch k {
	case Equal, PlusEqual, MinusEqual, StarEqual, SlashEqual, PercentEqual:
		return true
	}
	return false
}
