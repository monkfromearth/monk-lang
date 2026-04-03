package syntax

import "fmt"

// Parser transforms a token stream into an AST.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse tokenizes source and returns the AST, or an error.
func Parse(source string) (*Program, error) {
	tokens := Scan(source)
	p := &Parser{tokens: tokens, pos: 0}
	return p.parseProgram()
}

func (p *Parser) parseProgram() (*Program, error) {
	prog := &Program{}
	for !p.atEnd() {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		prog.Stmts = append(prog.Stmts, stmt)
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
		p.advance()
		return &BreakStmt{}, nil
	case Continue:
		p.advance()
		return &ContinueStmt{}, nil
	case Guard:
		return p.parseGuard()
	case Type:
		return p.parseTypeDecl()
	case Use:
		return p.parseUse()
	case Export:
		return p.parseExport()
	// No case for LeftBrace here — bare { at statement level is a record expression.
	// Blocks only exist inside if/while/for/guard/function bodies (parseBlock is
	// called explicitly by those parsers). This avoids the block-vs-record ambiguity.
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseVarDecl() (*VarDeclStmt, error) {
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
		// Could be a type annotation: let count int = 0
		// But only if followed by = or [] or ?
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

	// Look ahead: is the next token after the name '=', '[]', or '?'?
	// If it's '=' then this IS a type annotation (name = value after it)
	saved := p.pos
	name := p.current().Text
	p.advance()

	te := TypeExpr{Name: name}

	// Check for array suffix []
	if p.current().Kind == LeftBracket && p.peek().Kind == RightBracket {
		te.IsArray = true
		p.advance() // [
		p.advance() // ]
	}

	// Check for optional suffix ?
	if p.current().Kind == Question {
		te.Optional = true
		p.advance()
	}

	// If next is '=' then this was a type annotation
	if p.current().Kind == Equal {
		return te, true
	}

	// Not a type annotation, restore position
	p.pos = saved
	return TypeExpr{}, false
}

func (p *Parser) parseIf() (*IfStmt, error) {
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
		p.advance() // skip 'else'
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

	return &IfStmt{Condition: condition, Then: then, Else: elseStmt}, nil
}

func (p *Parser) parseWhile() (*WhileStmt, error) {
	p.advance() // skip 'while'

	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &WhileStmt{Condition: condition, Body: body}, nil
}

func (p *Parser) parseFor() (*ForStmt, error) {
	p.advance() // skip 'for'

	if p.current().Kind != Identifier {
		return nil, p.error("expected identifier after 'for'")
	}
	varName := p.current().Text
	p.advance()

	if p.current().Kind != In {
		return nil, p.error("expected 'in' after for variable")
	}
	p.advance() // skip 'in'

	iterable, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ForStmt{VarName: varName, Iterable: iterable, Body: body}, nil
}

func (p *Parser) parseReturn() (*ReturnStmt, error) {
	p.advance() // skip 'return'

	// Bare return: next token is } or EOF or another statement keyword
	if p.atEnd() || p.current().Kind == RightBrace || p.isStmtStart() {
		return &ReturnStmt{Value: nil}, nil
	}

	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Value: value}, nil
}

func (p *Parser) parseGuard() (*GuardStmt, error) {
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
		VarName:   varName,
		Expr:      expr,
		ErrorName: errorName,
		Against:   block,
	}, nil
}

func (p *Parser) parseTypeDecl() (*TypeDeclStmt, error) {
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

	return &TypeDeclStmt{Name: name, Definition: def}, nil
}

func (p *Parser) parseTypeDef() (TypeDefExpr, error) {
	// Record type: { x: int, y: int }
	if p.current().Kind == LeftBrace {
		return p.parseRecordTypeDef()
	}

	// Alias type: int, string, etc.
	if p.current().Kind == Identifier {
		te := p.parseTypeExpr()
		return TypeDefExpr{AliasOf: &te}, nil
	}

	// Function type: (int, int) -> int
	if p.current().Kind == LeftParen {
		te := p.parseTypeExpr()
		return TypeDefExpr{AliasOf: &te}, nil
	}

	return TypeDefExpr{}, p.error("expected type definition")
}

func (p *Parser) parseRecordTypeDef() (TypeDefExpr, error) {
	p.advance() // skip {

	var fields []RecordField
	for p.current().Kind != RightBrace && !p.atEnd() {
		if p.current().Kind != Identifier {
			return TypeDefExpr{}, p.error("expected field name")
		}
		key := p.current().Text
		p.advance()

		if p.current().Kind != Colon {
			return TypeDefExpr{}, p.error("expected ':' after field name")
		}
		p.advance()

		te := p.parseTypeExpr()
		fields = append(fields, RecordField{Key: key, Value: &IdentExpr{Name: te.Name}})

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

func (p *Parser) parseTypeExpr() TypeExpr {
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

func (p *Parser) parseUse() (*UseStmt, error) {
	p.advance() // skip 'use'

	u := &UseStmt{}

	if p.current().Kind != Identifier {
		return nil, p.error("expected identifier after 'use'")
	}
	firstName := p.current().Text
	p.advance()

	// Check for alias: use X as Y from "..."
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
	p.advance() // skip 'export'

	stmt, err := p.parseStmt()
	if err != nil {
		return nil, err
	}

	return &ExportStmt{Stmt: stmt}, nil
}

func (p *Parser) parseExprStmt() (*ExprStmt, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ExprStmt{Expr: expr}, nil
}

func (p *Parser) parseBlock() (*BlockStmt, error) {
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

	return &BlockStmt{Stmts: stmts}, nil
}

// --- Expression parsing (Pratt / precedence climbing) ---

func (p *Parser) parseExpr() (Expr, error) {
	return p.parseAssignment()
}

func (p *Parser) parseAssignment() (Expr, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	// Assignment operators
	if isAssignOp(p.current().Kind) {
		op := p.current().Kind
		p.advance()
		value, err := p.parseAssignment() // right-associative
		if err != nil {
			return nil, err
		}
		return &AssignExpr{Target: expr, Op: op, Value: value}, nil
	}

	return expr, nil
}

func (p *Parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Or || p.current().Kind == PipePipe {
		op := p.current().Kind
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	left, err := p.parseBitwiseOr()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == And || p.current().Kind == AmpAmp {
		op := p.current().Kind
		p.advance()
		right, err := p.parseBitwiseOr()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseOr() (Expr, error) {
	left, err := p.parseBitwiseXor()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Pipe {
		op := p.current().Kind
		p.advance()
		right, err := p.parseBitwiseXor()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseXor() (Expr, error) {
	left, err := p.parseBitwiseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Caret {
		op := p.current().Kind
		p.advance()
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseAnd() (Expr, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Amp {
		op := p.current().Kind
		p.advance()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseEquality() (Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == EqualEqual || p.current().Kind == BangEqual || p.current().Kind == Is {
		op := p.current().Kind
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
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
		op := p.current().Kind
		p.advance()
		right, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseShift() (Expr, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == ShiftLeft || p.current().Kind == ShiftRight {
		op := p.current().Kind
		p.advance()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAddSub() (Expr, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Plus || p.current().Kind == Minus {
		op := p.current().Kind
		p.advance()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMulDiv() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.current().Kind == Star || p.current().Kind == Slash || p.current().Kind == Percent {
		op := p.current().Kind
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	if p.current().Kind == Minus || p.current().Kind == Not || p.current().Kind == Bang || p.current().Kind == Tilde {
		op := p.current().Kind
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op, Operand: operand}, nil
	}

	if p.current().Kind == Throw {
		p.advance()
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &ThrowExpr{Value: value}, nil
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
			// Function call
			p.advance()
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			expr = &CallExpr{Callee: expr, Args: args}

		case LeftBracket:
			// Index access
			p.advance()
			index, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if p.current().Kind != RightBracket {
				return nil, p.error("expected ']'")
			}
			p.advance()
			expr = &IndexExpr{Object: expr, Index: index}

		case Dot:
			// Property access
			p.advance()
			if p.current().Kind != Identifier {
				return nil, p.error("expected property name after '.'")
			}
			name := p.current().Text
			p.advance()
			expr = &PropertyExpr{Object: expr, Property: name}

		default:
			return expr, nil
		}
	}
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.current()

	switch tok.Kind {
	case IntLiteral:
		p.advance()
		return &NumberExpr{Value: tok.Text, IsInt: true}, nil

	case FloatLiteral:
		p.advance()
		return &NumberExpr{Value: tok.Text, IsInt: false}, nil

	case StringLiteral:
		p.advance()
		return &StringExpr{Value: tok.Text}, nil

	case TemplateLiteral:
		p.advance()
		return &TemplateExpr{Value: tok.Text}, nil

	case True:
		p.advance()
		return &BoolExpr{Value: true}, nil

	case False:
		p.advance()
		return &BoolExpr{Value: false}, nil

	case None:
		p.advance()
		return &NoneExpr{}, nil

	case Identifier:
		p.advance()
		return &IdentExpr{Name: tok.Text}, nil

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
	// Distinguish between (expr) grouping and (params) returnType { body } function
	// Heuristic: if after '(' we see 'ident ident' or ')' followed by ident/'{', it's a function

	saved := p.pos
	p.advance() // skip (

	// () -> definitely a function (no params)
	if p.current().Kind == RightParen {
		p.advance()
		// If followed by a type name (return type) or '{', it's a function
		if isTypeName(p.current().Kind) || p.current().Kind == LeftBrace || p.current().Kind == LeftParen {
			p.pos = saved
			return p.parseFuncExpr()
		}
		// Empty parens with nothing after — error
		p.pos = saved
		p.advance() // skip (
		// Actually this might be a grouping of nothing, which is invalid
		return nil, p.error("unexpected ')'")
	}

	// Look for function pattern: (name type, ...)
	if p.current().Kind == Identifier {
		afterIdent := p.pos + 1
		if afterIdent < len(p.tokens) {
			next := p.tokens[afterIdent].Kind
			if next == Identifier || next == LeftParen {
				// Looks like (param type, ...) — it's a function
				p.pos = saved
				return p.parseFuncExpr()
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

	// Return type (before the block)
	var returnType TypeExpr
	if isTypeName(p.current().Kind) {
		returnType = p.parseTypeExpr()
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &FuncExpr{Params: params, ReturnType: returnType, Body: body}, nil
}

func (p *Parser) parseArrayLiteral() (Expr, error) {
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

	return &ArrayExpr{Elements: elements}, nil
}

func (p *Parser) parseRecordLiteral() (Expr, error) {
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

	return &RecordExpr{Fields: fields}, nil
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
