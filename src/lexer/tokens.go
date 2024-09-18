package lexer

type TokenKind int

const (
	EOFToken TokenKind = iota
	NumberToken
	StringToken
	IdentifierToken
	OpenParenthesisToken   // (
	CloseParenthesisToken  // )
	OpenCurlyBraceToken    // {
	OpenBracketToken       // [
	CloseCurlyBraceToken   // }
	CloseBracketToken      // ]
	ColonToken             // :
	CommaToken             // ,
	DotToken               // .
	AssignmentToken        // =
	IsToken                // is
	IsAltToken             // ==
	NotToken               // not
	NotAltToken            // !
	NotEqualsToken         // is not
	NotEqualsAltToken      // !=
	LessThanToken          // <
	LessThanEqualsToken    // <=
	GreaterThanToken       // >
	GreaterThanEqualsToken // >=
	OrToken                // or
	OrAltToken             // ||
	AndToken               // and
	AndAltToken            // &&
	PlusEqualsToken        // +=
	MinusEqualsToken       // -=
	MultiplyEqualsToken    // *=
	DivideEqualsToken      // /=
	ModuloEqualsToken      // %=
	PlusToken              // +
	MinusToken             // -
	MultiplyToken          // *
	DivideToken            // /
	ModuloToken            // %
	QuestionMarkToken      // ?
	ColonColonToken        // ::
	ReferenceToken         // &
	LetToken               // let
	ConstToken             // const
	UseToken               // use
	ExportToken            // export
	RefToken               // ref
	FromToken              // from
	IfToken                // if
	ElseToken              // else
	ForToken               // for
	InToken                // in
	WhileToken             // while
	BreakToken             // break
	ContinueToken          // continue
	ReturnToken            // return
)

var ReservedWords = map[string]TokenKind{
	"let":      LetToken,
	"const":    ConstToken,
	"use":      UseToken,
	"export":   ExportToken,
	"ref":      RefToken,
	"from":     FromToken,
	"if":       IfToken,
	"else":     ElseToken,
	"for":      ForToken,
	"in":       InToken,
	"while":    WhileToken,
	"break":    BreakToken,
	"continue": ContinueToken,
	"return":   ReturnToken,
	"is":       IsToken,
	"not":      NotToken,
	"and":      AndToken,
	"or":       OrToken,
}

type Token struct {
	Value  string
	Kind   TokenKind
	Line   int
	Column int
}
