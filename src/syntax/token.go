// Package syntax provides the lexer, parser, and AST for the Monk language.
//
// The pipeline: source text → Scanner → []Token → Parser → *Program (AST).
// See spec/REFERENCE.md for the language specification.
package syntax

// TokenKind identifies the type of a lexical token.
type TokenKind int

const (
	// Special tokens
	Illegal    TokenKind = iota // unrecognized character
	Eof                         // end of input
	Identifier                  // user-defined name (variable, function, type)

	// Literal tokens
	IntLiteral      // integer: 42, 0xFF, 0b1010, 0o77, 1_000
	FloatLiteral    // float: 3.14, 1.23e5, 1e-3
	StringLiteral   // double-quoted: "hello"
	TemplateLiteral // backtick: `hello`

	// Keyword tokens — each maps to a reserved word in the source
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

	// Arithmetic operators
	Plus    // +
	Minus   // -
	Star    // *
	Slash   // /
	Percent // %

	// Comparison operators
	EqualEqual   // ==
	BangEqual    // !=
	Less         // <
	Greater      // >
	LessEqual    // <=
	GreaterEqual // >=

	// Logical operators
	AmpAmp   // &&
	PipePipe // ||
	Bang     // !

	// Bitwise operators
	Amp        // &
	Pipe       // |
	Caret      // ^
	Tilde      // ~
	ShiftLeft  // <<
	ShiftRight // >>

	// Assignment operators
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

// kindNames provides display names for non-obvious token kinds.
// Keywords and operators use their source text directly via Token.Text.
var kindNames = map[TokenKind]string{
	Illegal:         "ILLEGAL",
	Eof:             "EOF",
	Identifier:      "IDENT",
	IntLiteral:      "INT",
	FloatLiteral:    "FLOAT",
	StringLiteral:   "STRING",
	TemplateLiteral: "TEMPLATE",
}

// String returns a human-readable name for the token kind.
func (k TokenKind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return "UNKNOWN"
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
