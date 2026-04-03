package syntax

// TokenKind identifies the type of a lexical token.
type TokenKind int

const (
	// Special
	Illegal TokenKind = iota
	Eof
	Identifier

	// Literals
	IntLiteral
	FloatLiteral
	StringLiteral
	TemplateLiteral

	// Keywords
	Let
	Const
	If
	Else
	For
	In
	While
	Break
	Continue
	Return
	Guard
	Against
	Throw
	Type
	Use
	Export
	From
	As
	Is
	Not
	And
	Or
	True
	False
	None

	// Arithmetic
	Plus    // +
	Minus   // -
	Star    // *
	Slash   // /
	Percent // %

	// Comparison
	EqualEqual   // ==
	BangEqual    // !=
	Less         // <
	Greater      // >
	LessEqual    // <=
	GreaterEqual // >=

	// Logical
	AmpAmp   // &&
	PipePipe // ||
	Bang     // !

	// Bitwise
	Amp        // &
	Pipe       // |
	Caret      // ^
	Tilde      // ~
	ShiftLeft  // <<
	ShiftRight // >>

	// Assignment
	Equal        // =
	PlusEqual    // +=
	MinusEqual   // -=
	StarEqual    // *=
	SlashEqual   // /=
	PercentEqual // %=

	// Delimiters
	LeftParen    // (
	RightParen   // )
	LeftBrace    // {
	RightBrace   // }
	LeftBracket  // [
	RightBracket // ]
	Colon        // :
	Comma        // ,
	Dot          // .
	Question     // ?
	Arrow        // ->
)

// String returns a human-readable name for the token kind.
func (k TokenKind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return "UNKNOWN"
}

// kindNames provides display names only for non-obvious token kinds.
// Keyword and operator tokens use their source text directly via Token.Text.
var kindNames = map[TokenKind]string{
	Illegal:         "ILLEGAL",
	Eof:             "EOF",
	Identifier:      "IDENT",
	IntLiteral:      "INT",
	FloatLiteral:    "FLOAT",
	StringLiteral:   "STRING",
	TemplateLiteral: "TEMPLATE",
}

// keywords maps source text to keyword token kinds.
var keywords = map[string]TokenKind{
	"let":      Let,
	"const":    Const,
	"if":       If,
	"else":     Else,
	"for":      For,
	"in":       In,
	"while":    While,
	"break":    Break,
	"continue": Continue,
	"return":   Return,
	"guard":    Guard,
	"against":  Against,
	"throw":    Throw,
	"type":     Type,
	"use":      Use,
	"export":   Export,
	"from":     From,
	"as":       As,
	"is":       Is,
	"not":      Not,
	"and":      And,
	"or":       Or,
	"true":     True,
	"false":    False,
	"none":     None,
}

// LookupIdent returns the keyword TokenKind for ident if it is a keyword,
// or Identifier if it is a regular identifier.
func LookupIdent(ident string) TokenKind {
	if kind, ok := keywords[ident]; ok {
		return kind
	}
	return Identifier
}

// Token represents a single lexical token with its position in source.
type Token struct {
	Kind   TokenKind
	Text   string // the actual source text
	Line   int
	Column int
}
