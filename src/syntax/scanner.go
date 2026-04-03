package syntax

// Scanner reads source text and produces tokens.
type Scanner struct {
	source []byte
	pos    int
	line   int
	column int
}

// Scan tokenizes the entire source string and returns a slice of tokens.
// The last token is always Eof.
func Scan(source string) []Token {
	s := &Scanner{
		source: []byte(source),
		pos:    0,
		line:   1,
		column: 1,
	}

	var tokens []Token
	for {
		tok := s.next()
		tokens = append(tokens, tok)
		if tok.Kind == Eof {
			break
		}
	}
	return tokens
}

// next returns the next token from the source.
func (s *Scanner) next() Token {
	s.skipWhitespace()

	if s.atEnd() {
		return Token{Kind: Eof, Line: s.line, Column: s.column}
	}

	ch := s.current()
	line, col := s.line, s.column

	// String literal
	if ch == '"' {
		return s.scanString(line, col)
	}

	// Template literal
	if ch == '`' {
		return s.scanTemplate(line, col)
	}

	// Number
	if isDigit(ch) {
		return s.scanNumber(line, col)
	}

	// Identifier or keyword
	if isAlpha(ch) {
		return s.scanIdentifier(line, col)
	}

	// Operators and delimiters
	return s.scanOperator(line, col)
}

func (s *Scanner) scanString(line, col int) Token {
	s.advance() // skip opening "
	start := s.pos
	for !s.atEnd() && s.current() != '"' {
		if s.current() == '\\' {
			s.advance() // skip escape character
		}
		s.advance()
	}
	text := string(s.source[start:s.pos])
	if !s.atEnd() {
		s.advance() // skip closing "
	}
	return Token{Kind: StringLiteral, Text: text, Line: line, Column: col}
}

func (s *Scanner) scanTemplate(line, col int) Token {
	s.advance() // skip opening `
	start := s.pos
	for !s.atEnd() && s.current() != '`' {
		s.advance()
	}
	text := string(s.source[start:s.pos])
	if !s.atEnd() {
		s.advance() // skip closing `
	}
	return Token{Kind: TemplateLiteral, Text: text, Line: line, Column: col}
}

func (s *Scanner) scanNumber(line, col int) Token {
	start := s.pos
	kind := IntLiteral

	// Handle hex, binary, octal prefixes
	if s.current() == '0' && s.pos+1 < len(s.source) {
		next := s.source[s.pos+1]
		if next == 'x' || next == 'X' || next == 'b' || next == 'B' || next == 'o' || next == 'O' {
			s.advance() // 0
			s.advance() // prefix letter
			for !s.atEnd() && (isDigit(s.current()) || isHexDigit(s.current()) || s.current() == '_') {
				s.advance()
			}
			return Token{Kind: IntLiteral, Text: string(s.source[start:s.pos]), Line: line, Column: col}
		}
	}

	for !s.atEnd() && (isDigit(s.current()) || s.current() == '_') {
		s.advance()
	}

	// Float: decimal point
	if !s.atEnd() && s.current() == '.' && s.pos+1 < len(s.source) && isDigit(s.source[s.pos+1]) {
		kind = FloatLiteral
		s.advance() // .
		for !s.atEnd() && (isDigit(s.current()) || s.current() == '_') {
			s.advance()
		}
	}

	// Float: exponent
	if !s.atEnd() && (s.current() == 'e' || s.current() == 'E') {
		kind = FloatLiteral
		s.advance() // e/E
		if !s.atEnd() && (s.current() == '+' || s.current() == '-') {
			s.advance()
		}
		for !s.atEnd() && isDigit(s.current()) {
			s.advance()
		}
	}

	return Token{Kind: kind, Text: string(s.source[start:s.pos]), Line: line, Column: col}
}

func (s *Scanner) scanIdentifier(line, col int) Token {
	start := s.pos
	for !s.atEnd() && isAlphaNumeric(s.current()) {
		s.advance()
	}
	text := string(s.source[start:s.pos])
	kind := LookupIdent(text)
	return Token{Kind: kind, Text: text, Line: line, Column: col}
}

func (s *Scanner) scanOperator(line, col int) Token {
	ch := s.current()
	s.advance()

	// Two-character operators
	if !s.atEnd() {
		next := s.current()
		switch {
		case ch == '=' && next == '=':
			s.advance()
			return Token{Kind: EqualEqual, Text: "==", Line: line, Column: col}
		case ch == '!' && next == '=':
			s.advance()
			return Token{Kind: BangEqual, Text: "!=", Line: line, Column: col}
		case ch == '<' && next == '=':
			s.advance()
			return Token{Kind: LessEqual, Text: "<=", Line: line, Column: col}
		case ch == '>' && next == '=':
			s.advance()
			return Token{Kind: GreaterEqual, Text: ">=", Line: line, Column: col}
		case ch == '<' && next == '<':
			s.advance()
			return Token{Kind: ShiftLeft, Text: "<<", Line: line, Column: col}
		case ch == '>' && next == '>':
			s.advance()
			return Token{Kind: ShiftRight, Text: ">>", Line: line, Column: col}
		case ch == '&' && next == '&':
			s.advance()
			return Token{Kind: AmpAmp, Text: "&&", Line: line, Column: col}
		case ch == '|' && next == '|':
			s.advance()
			return Token{Kind: PipePipe, Text: "||", Line: line, Column: col}
		case ch == '+' && next == '=':
			s.advance()
			return Token{Kind: PlusEqual, Text: "+=", Line: line, Column: col}
		case ch == '-' && next == '=':
			s.advance()
			return Token{Kind: MinusEqual, Text: "-=", Line: line, Column: col}
		case ch == '*' && next == '=':
			s.advance()
			return Token{Kind: StarEqual, Text: "*=", Line: line, Column: col}
		case ch == '/' && next == '=':
			s.advance()
			return Token{Kind: SlashEqual, Text: "/=", Line: line, Column: col}
		case ch == '%' && next == '=':
			s.advance()
			return Token{Kind: PercentEqual, Text: "%=", Line: line, Column: col}
		case ch == '-' && next == '>':
			s.advance()
			return Token{Kind: Arrow, Text: "->", Line: line, Column: col}
		}
	}

	// Single-character operators and delimiters
	text := string(ch)
	switch ch {
	case '+':
		return Token{Kind: Plus, Text: text, Line: line, Column: col}
	case '-':
		return Token{Kind: Minus, Text: text, Line: line, Column: col}
	case '*':
		return Token{Kind: Star, Text: text, Line: line, Column: col}
	case '/':
		return Token{Kind: Slash, Text: text, Line: line, Column: col}
	case '%':
		return Token{Kind: Percent, Text: text, Line: line, Column: col}
	case '=':
		return Token{Kind: Equal, Text: text, Line: line, Column: col}
	case '!':
		return Token{Kind: Bang, Text: text, Line: line, Column: col}
	case '<':
		return Token{Kind: Less, Text: text, Line: line, Column: col}
	case '>':
		return Token{Kind: Greater, Text: text, Line: line, Column: col}
	case '&':
		return Token{Kind: Amp, Text: text, Line: line, Column: col}
	case '|':
		return Token{Kind: Pipe, Text: text, Line: line, Column: col}
	case '^':
		return Token{Kind: Caret, Text: text, Line: line, Column: col}
	case '~':
		return Token{Kind: Tilde, Text: text, Line: line, Column: col}
	case '(':
		return Token{Kind: LeftParen, Text: text, Line: line, Column: col}
	case ')':
		return Token{Kind: RightParen, Text: text, Line: line, Column: col}
	case '{':
		return Token{Kind: LeftBrace, Text: text, Line: line, Column: col}
	case '}':
		return Token{Kind: RightBrace, Text: text, Line: line, Column: col}
	case '[':
		return Token{Kind: LeftBracket, Text: text, Line: line, Column: col}
	case ']':
		return Token{Kind: RightBracket, Text: text, Line: line, Column: col}
	case ':':
		return Token{Kind: Colon, Text: text, Line: line, Column: col}
	case ',':
		return Token{Kind: Comma, Text: text, Line: line, Column: col}
	case '.':
		return Token{Kind: Dot, Text: text, Line: line, Column: col}
	case '?':
		return Token{Kind: Question, Text: text, Line: line, Column: col}
	default:
		return Token{Kind: Illegal, Text: text, Line: line, Column: col}
	}
}

// --- helpers ---

func (s *Scanner) current() byte    { return s.source[s.pos] }
func (s *Scanner) atEnd() bool      { return s.pos >= len(s.source) }

func (s *Scanner) advance() {
	if s.pos < len(s.source) {
		if s.source[s.pos] == '\n' {
			s.line++
			s.column = 1
		} else {
			s.column++
		}
		s.pos++
	}
}

func (s *Scanner) skipWhitespace() {
	for !s.atEnd() {
		ch := s.current()
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			s.advance()
			continue
		}
		// Single-line comment
		if ch == '/' && s.pos+1 < len(s.source) && s.source[s.pos+1] == '/' {
			for !s.atEnd() && s.current() != '\n' {
				s.advance()
			}
			continue
		}
		break
	}
}

func isDigit(c byte) bool        { return c >= '0' && c <= '9' }
func isHexDigit(c byte) bool     { return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') }
func isAlpha(c byte) bool        { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' }
func isAlphaNumeric(c byte) bool { return isAlpha(c) || isDigit(c) }
