package syntax

// Expression parsing by precedence climbing.
//
// Monk's 13 precedence levels, tightest binding last:
//   1. or       || or
//   2. and      && and
//   3. bit-or   |
//   4. bit-xor  ^
//   5. bit-and  &
//   6. equality == != is
//   7. compare  < > <= >=
//   8. shift    << >>
//   9. add/sub  + -
//  10. mul/div  * / %
//  11. unary    - not ! ~  (and throw)
//  12. postfix  call[...]  index[...]  .property
//  13. primary  literals / idents / parens / arrays / records
//
// Assignment is NOT in this chain — it is handled as a statement.
// See parseExprOrAssignStmt in parse_stmt.go.

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

// parseParenOrFunc disambiguates between a grouped expression `(expr)` and
// a function literal `(params) returnType { body }`.
//
// Strategy: save position, look at what follows `(`. Commit to function
// literal when we see a telltale sign INSIDE the parens (ident-typename
// pair, or ident-funcTypeReturn-arrow), or when the first token is `)`
// (zero-param function). Otherwise parse as a grouped expression.
//
// We deliberately only look INSIDE the parens, not after `)`, because a
// statement like `let x = (a + b)\nfoo int = 1` would otherwise be misread
// as a function literal once we see `foo` (an Identifier → type name) after
// the closing paren.
func (p *Parser) parseParenOrFunc() (Expr, error) {
	saved := p.pos
	p.advance() // skip (

	// () followed by type name, `{`, or `(` (func-type return) → zero-param function
	if p.current().Kind == RightParen {
		p.advance()
		if isTypeName(p.current().Kind) || p.current().Kind == LeftBrace || p.current().Kind == LeftParen {
			p.pos = saved
			return p.parseFuncExpr()
		}
		p.pos = saved
		p.advance()
		return nil, p.error("unexpected ')'")
	}

	// Look for function pattern: identifier followed by a type (param-name param-type).
	// This is unambiguous because in an expression, ident is never followed by a
	// type-name token (it would be ident operator ident, or ident ( for a call, etc.)
	if p.current().Kind == Identifier {
		afterIdent := p.pos + 1
		if afterIdent < len(p.tokens) {
			next := p.tokens[afterIdent].Kind
			// ident followed by a type-name (ident or `none`) → param-name type pair
			if isTypeName(next) {
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

	// Return type may be absent, a named type (ident/none), or a function type
	// beginning with LeftParen. Parameter types already accept all three via
	// parseTypeExpr — return types must too.
	var returnType TypeExpr
	if isTypeName(p.current().Kind) || p.current().Kind == LeftParen {
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
