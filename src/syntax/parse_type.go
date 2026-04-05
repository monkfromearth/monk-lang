package syntax

// Type-expression parsing.
//
// Types appear in three places:
//   - Variable declarations: `let x int = 42`
//   - Function params/returns: `(x int) int { ... }`
//   - Type declarations: `type Point = { x: int, y: int }`
//
// Function types (`(int, int) -> int`) are parsed by parseFuncType.

// isTypeName reports whether a token kind can start a (plain) type name.
// `none` is its own keyword token, not an Identifier, so it needs special
// handling — it is both a value literal and a valid type name.
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

// tryParseTypeExpr is used by parseVarDecl to detect an optional type
// annotation between the variable name and the `=`. It rewinds if it can't
// commit, so the caller falls back to "no annotation".
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

// parseTypeExpr parses any type expression: named, array, optional, or function.
// Does NOT return an error directly — the (legacy) signature has many callers.
// Hard failures (like a function type missing its `->`) are stored on
// Parser.typeErr so parseProgram reports them.
func (p *Parser) parseTypeExpr() TypeExpr {
	// Function type: (T, T) -> T
	if p.current().Kind == LeftParen {
		return p.parseFuncType()
	}

	// Defensive: callers are expected to have verified isTypeName before
	// calling, but we guard anyway so malformed input like a missing type
	// after a param name produces a proper "expected type name" error
	// instead of silently swallowing an arbitrary token's text.
	//
	// CRITICAL: we MUST advance past the offending token before returning,
	// otherwise loops in parseFuncType / parseRecordTypeDef that only
	// terminate on ')' / '}' / EOF would spin forever on malformed input
	// like `(42) -> int`.
	if !isTypeName(p.current().Kind) {
		if p.typeErr == nil {
			p.typeErr = p.error("expected type name")
		}
		p.advance()
		return TypeExpr{}
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
		// Bail out immediately if parseTypeExpr signaled an error — its
		// advance is good for loop progress but we don't want to emit
		// additional spurious errors on the trailing garbage.
		if p.typeErr != nil {
			return TypeExpr{}
		}
		params = append(params, p.parseTypeExpr())
		if p.current().Kind == Comma {
			p.advance()
		}
	}
	p.advance() // skip )
	// Require ` -> ReturnType`.
	if p.current().Kind != Arrow {
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

// parseTypeDef parses the right-hand side of a `type X = ...` declaration.
// Either a record type (`{ field: type, ... }`) or a type alias.
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
