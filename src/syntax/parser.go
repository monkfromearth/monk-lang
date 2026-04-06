package syntax

import "fmt"

// Parser transforms a token stream into an AST.
//
// The implementation is split across several files:
//   - parser.go       — this file: Parser struct, entry points, token helpers
//   - parse_stmt.go   — statement parsers
//   - parse_expr.go   — expression parsers (precedence climbing)
//   - parse_type.go   — type-expression parsers + matchRightParen
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

// --- Token helpers ---

// currentPos returns the current token's position as a Pos.
func (p *Parser) currentPos() Pos {
	tok := p.current()
	return Pos{Line: tok.Line, Column: tok.Column}
}

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

// isStmtStart reports whether the current token begins a statement. Used by
// parseReturn to detect a bare `return` that is followed by another statement.
func (p *Parser) isStmtStart() bool {
	switch p.current().Kind {
	case Let, Const, If, While, For, Return, Break, Continue, Guard, Type, Use, Export:
		return true
	}
	return false
}

// isAssignOp reports whether a token is one of the assignment operators.
func isAssignOp(k TokenKind) bool {
	switch k {
	case Equal, PlusEqual, MinusEqual, StarEqual, SlashEqual, PercentEqual:
		return true
	}
	return false
}
