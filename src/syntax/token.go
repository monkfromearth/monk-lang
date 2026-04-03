package syntax

// TokenKind identifies the type of a lexical token.
type TokenKind int

const (
	// Special
	ILLEGAL TokenKind = iota
	EOF

	// Literals
	INT    // 42, 0xFF, 0b1010
	FLOAT  // 3.14, 1.23e5
	STRING // "hello"
	TMPL   // `template literal`

	// Identifiers
	IDENT // variable_name

	// Keywords
	LET
	CONST
	IF
	ELSE
	FOR
	IN
	WHILE
	BREAK
	CONTINUE
	RETURN
	GUARD
	AGAINST
	THROW
	TYPE
	USE
	EXPORT
	FROM
	AS
	IS
	NOT
	AND
	OR
	TRUE
	FALSE
	NONE

	// Arithmetic operators
	PLUS     // +
	MINUS    // -
	STAR     // *
	SLASH    // /
	PERCENT  // %

	// Comparison operators
	EQ    // ==
	NEQ   // !=
	LT    // <
	GT    // >
	LTEQ  // <=
	GTEQ  // >=

	// Logical operators
	AMPAMP  // &&
	PIPEPIPE // ||
	BANG    // !

	// Bitwise operators
	AMP   // &
	PIPE  // |
	CARET // ^
	TILDE // ~
	SHL   // <<
	SHR   // >>

	// Assignment operators
	ASSIGN     // =
	PLUSEQ     // +=
	MINUSEQ    // -=
	STAREQ     // *=
	SLASHEQ    // /=
	PERCENTEQ  // %=

	// Delimiters
	LPAREN // (
	RPAREN // )
	LBRACE // {
	RBRACE // }
	LBRACK // [
	RBRACK // ]
	COLON  // :
	COMMA  // ,
	DOT    // .
	QMARK  // ?
	ARROW  // ->
)

var tokenNames = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",

	INT:    "INT",
	FLOAT:  "FLOAT",
	STRING: "STRING",
	TMPL:   "TMPL",

	IDENT: "IDENT",

	LET:      "let",
	CONST:    "const",
	IF:       "if",
	ELSE:     "else",
	FOR:      "for",
	IN:       "in",
	WHILE:    "while",
	BREAK:    "break",
	CONTINUE: "continue",
	RETURN:   "return",
	GUARD:    "guard",
	AGAINST:  "against",
	THROW:    "throw",
	TYPE:     "type",
	USE:      "use",
	EXPORT:   "export",
	FROM:     "from",
	AS:       "as",
	IS:       "is",
	NOT:      "not",
	AND:      "and",
	OR:       "or",
	TRUE:     "true",
	FALSE:    "false",
	NONE:     "none",

	PLUS:    "+",
	MINUS:   "-",
	STAR:    "*",
	SLASH:   "/",
	PERCENT: "%",

	EQ:   "==",
	NEQ:  "!=",
	LT:   "<",
	GT:   ">",
	LTEQ: "<=",
	GTEQ: ">=",

	AMPAMP:   "&&",
	PIPEPIPE: "||",
	BANG:     "!",

	AMP:   "&",
	PIPE:  "|",
	CARET: "^",
	TILDE: "~",
	SHL:   "<<",
	SHR:   ">>",

	ASSIGN:    "=",
	PLUSEQ:    "+=",
	MINUSEQ:   "-=",
	STAREQ:    "*=",
	SLASHEQ:   "/=",
	PERCENTEQ: "%=",

	LPAREN: "(",
	RPAREN: ")",
	LBRACE: "{",
	RBRACE: "}",
	LBRACK: "[",
	RBRACK: "]",
	COLON:  ":",
	COMMA:  ",",
	DOT:    ".",
	QMARK:  "?",
	ARROW:  "->",
}

func (k TokenKind) String() string {
	if int(k) < len(tokenNames) {
		return tokenNames[k]
	}
	return "UNKNOWN"
}

// keywords maps identifier strings to keyword token kinds.
var keywords = map[string]TokenKind{
	"let":      LET,
	"const":    CONST,
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"in":       IN,
	"while":    WHILE,
	"break":    BREAK,
	"continue": CONTINUE,
	"return":   RETURN,
	"guard":    GUARD,
	"against":  AGAINST,
	"throw":    THROW,
	"type":     TYPE,
	"use":      USE,
	"export":   EXPORT,
	"from":     FROM,
	"as":       AS,
	"is":       IS,
	"not":      NOT,
	"and":      AND,
	"or":       OR,
	"true":     TRUE,
	"false":    FALSE,
	"none":     NONE,
}

// LookupIdent returns the keyword TokenKind for ident if it is a keyword,
// or IDENT if it is a regular identifier.
func LookupIdent(ident string) TokenKind {
	if kind, ok := keywords[ident]; ok {
		return kind
	}
	return IDENT
}

// Token represents a single lexical token with its position in the source.
type Token struct {
	Kind   TokenKind
	Text   string
	Line   int
	Column int
}
