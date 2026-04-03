package syntax

import "testing"

// --- Helpers ---

func expectTokens(t *testing.T, source string, expected []TokenKind) {
	t.Helper()
	tokens := Scan(source)
	if len(tokens) != len(expected) {
		t.Fatalf("Scan(%q): expected %d tokens, got %d", source, len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("Scan(%q) token %d: expected %s, got %s (text: %q)", source, i, exp, tokens[i].Kind, tokens[i].Text)
		}
	}
}

func expectToken(t *testing.T, source string, index int, kind TokenKind, text string) {
	t.Helper()
	tokens := Scan(source)
	if index >= len(tokens) {
		t.Fatalf("Scan(%q): expected token at index %d, only got %d tokens", source, index, len(tokens))
	}
	tok := tokens[index]
	if tok.Kind != kind {
		t.Errorf("Scan(%q) token %d kind: expected %s, got %s", source, index, kind, tok.Kind)
	}
	if tok.Text != text {
		t.Errorf("Scan(%q) token %d text: expected %q, got %q", source, index, text, tok.Text)
	}
}

func expectPosition(t *testing.T, source string, index int, line, col int) {
	t.Helper()
	tokens := Scan(source)
	if index >= len(tokens) {
		t.Fatalf("Scan(%q): expected token at index %d, only got %d tokens", source, index, len(tokens))
	}
	tok := tokens[index]
	if tok.Line != line || tok.Column != col {
		t.Errorf("Scan(%q) token %d (%q): expected %d:%d, got %d:%d", source, index, tok.Text, line, col, tok.Line, tok.Column)
	}
}

// === EMPTY / WHITESPACE / COMMENTS ===

func TestScanEmpty(t *testing.T) {
	expectTokens(t, "", []TokenKind{Eof})
}

func TestScanWhitespaceOnly(t *testing.T) {
	expectTokens(t, "   \t\t  \n\n  ", []TokenKind{Eof})
}

func TestScanComment(t *testing.T) {
	expectTokens(t, "// this is a comment", []TokenKind{Eof})
}

func TestScanCommentBeforeCode(t *testing.T) {
	expectTokens(t, "// comment\n42", []TokenKind{IntLiteral, Eof})
}

func TestScanInlineComment(t *testing.T) {
	expectTokens(t, "42 // the answer", []TokenKind{IntLiteral, Eof})
}

func TestScanMultipleCommentLines(t *testing.T) {
	expectTokens(t, "// first\n// second\n42", []TokenKind{IntLiteral, Eof})
}

// === INTEGER LITERALS ===

func TestScanInteger(t *testing.T) {
	expectToken(t, "42", 0, IntLiteral, "42")
}

func TestScanIntegerZero(t *testing.T) {
	expectToken(t, "0", 0, IntLiteral, "0")
}

func TestScanIntegerLarge(t *testing.T) {
	expectToken(t, "999999999999999", 0, IntLiteral, "999999999999999")
}

func TestScanIntegerUnderscore(t *testing.T) {
	expectToken(t, "1_000_000", 0, IntLiteral, "1_000_000")
}

func TestScanIntegerHex(t *testing.T) {
	expectToken(t, "0xFF", 0, IntLiteral, "0xFF")
}

func TestScanIntegerHexUpper(t *testing.T) {
	expectToken(t, "0XAB", 0, IntLiteral, "0XAB")
}

func TestScanIntegerBinary(t *testing.T) {
	expectToken(t, "0b1010", 0, IntLiteral, "0b1010")
}

func TestScanIntegerOctal(t *testing.T) {
	expectToken(t, "0o77", 0, IntLiteral, "0o77")
}

func TestScanIntegerHexUnderscore(t *testing.T) {
	expectToken(t, "0xFF_FF", 0, IntLiteral, "0xFF_FF")
}

// === FLOAT LITERALS ===

func TestScanFloat(t *testing.T) {
	expectToken(t, "3.14", 0, FloatLiteral, "3.14")
}

func TestScanFloatLeadingZero(t *testing.T) {
	expectToken(t, "0.5", 0, FloatLiteral, "0.5")
}

func TestScanFloatScientific(t *testing.T) {
	expectToken(t, "1.23e5", 0, FloatLiteral, "1.23e5")
}

func TestScanFloatScientificNegative(t *testing.T) {
	expectToken(t, "1e-3", 0, FloatLiteral, "1e-3")
}

func TestScanFloatScientificPositive(t *testing.T) {
	expectToken(t, "2.5e+2", 0, FloatLiteral, "2.5e+2")
}

func TestScanFloatScientificUpperE(t *testing.T) {
	expectToken(t, "1E10", 0, FloatLiteral, "1E10")
}

// === STRING LITERALS ===

func TestScanString(t *testing.T) {
	expectToken(t, `"hello world"`, 0, StringLiteral, "hello world")
}

func TestScanStringEmpty(t *testing.T) {
	expectToken(t, `""`, 0, StringLiteral, "")
}

func TestScanStringEscapeNewline(t *testing.T) {
	expectToken(t, `"line1\nline2"`, 0, StringLiteral, `line1\nline2`)
}

func TestScanStringEscapeTab(t *testing.T) {
	expectToken(t, `"col1\tcol2"`, 0, StringLiteral, `col1\tcol2`)
}

func TestScanStringEscapeQuote(t *testing.T) {
	expectToken(t, `"say \"hi\""`, 0, StringLiteral, `say \"hi\"`)
}

func TestScanStringEscapeBackslash(t *testing.T) {
	expectToken(t, `"path\\to\\file"`, 0, StringLiteral, `path\\to\\file`)
}

func TestScanStringUnterminated(t *testing.T) {
	tokens := Scan(`"hello`)
	// Should still produce a token (the scanner doesn't crash)
	if tokens[0].Kind != StringLiteral {
		t.Fatalf("expected StringLiteral for unterminated string, got %s", tokens[0].Kind)
	}
}

// === TEMPLATE LITERALS ===

func TestScanTemplate(t *testing.T) {
	expectToken(t, "`hello`", 0, TemplateLiteral, "hello")
}

func TestScanTemplateWithQuotes(t *testing.T) {
	expectToken(t, "`has \"double\" and 'single'`", 0, TemplateLiteral, `has "double" and 'single'`)
}

func TestScanTemplateMultiline(t *testing.T) {
	expectToken(t, "`line1\nline2\nline3`", 0, TemplateLiteral, "line1\nline2\nline3")
}

func TestScanTemplateEmpty(t *testing.T) {
	expectToken(t, "``", 0, TemplateLiteral, "")
}

// === IDENTIFIERS ===

func TestScanIdentifier(t *testing.T) {
	expectToken(t, "foo", 0, Identifier, "foo")
}

func TestScanIdentifierUnderscore(t *testing.T) {
	expectToken(t, "bar_baz", 0, Identifier, "bar_baz")
}

func TestScanIdentifierLeadingUnderscore(t *testing.T) {
	expectToken(t, "_private", 0, Identifier, "_private")
}

func TestScanIdentifierWithDigits(t *testing.T) {
	expectToken(t, "x1", 0, Identifier, "x1")
}

func TestScanIdentifierAllUnderscores(t *testing.T) {
	expectToken(t, "___", 0, Identifier, "___")
}

func TestScanIdentifierCamelCase(t *testing.T) {
	expectToken(t, "myVariable", 0, Identifier, "myVariable")
}

// === KEYWORDS ===

func TestScanAllKeywords(t *testing.T) {
	tests := []struct {
		source string
		kind   TokenKind
	}{
		{"let", Let},
		{"const", Const},
		{"if", If},
		{"else", Else},
		{"for", For},
		{"in", In},
		{"while", While},
		{"break", Break},
		{"continue", Continue},
		{"return", Return},
		{"guard", Guard},
		{"against", Against},
		{"throw", Throw},
		{"type", Type},
		{"use", Use},
		{"export", Export},
		{"from", From},
		{"as", As},
		{"is", Is},
		{"not", Not},
		{"and", And},
		{"or", Or},
		{"true", True},
		{"false", False},
		{"none", None},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			expectToken(t, tt.source, 0, tt.kind, tt.source)
		})
	}
}

func TestScanKeywordPrefix(t *testing.T) {
	// "letter" starts with "let" but is an identifier, not a keyword
	expectToken(t, "letter", 0, Identifier, "letter")
}

func TestScanKeywordSuffix(t *testing.T) {
	// "notify" contains "not" but is an identifier
	expectToken(t, "notify", 0, Identifier, "notify")
}

func TestScanKeywordInMiddle(t *testing.T) {
	// "forest" contains "for" but is an identifier
	expectToken(t, "forest", 0, Identifier, "forest")
}

// === SINGLE-CHARACTER OPERATORS ===

func TestScanArithmeticOperators(t *testing.T) {
	expectTokens(t, "+ - * / %", []TokenKind{Plus, Minus, Star, Slash, Percent, Eof})
}

func TestScanComparisonSingleChar(t *testing.T) {
	expectTokens(t, "< >", []TokenKind{Less, Greater, Eof})
}

func TestScanLogicalBang(t *testing.T) {
	expectToken(t, "!", 0, Bang, "!")
}

func TestScanBitwiseOperators(t *testing.T) {
	expectTokens(t, "& | ^ ~", []TokenKind{Amp, Pipe, Caret, Tilde, Eof})
}

func TestScanAssignmentEqual(t *testing.T) {
	expectToken(t, "=", 0, Equal, "=")
}

// === TWO-CHARACTER OPERATORS ===

func TestScanEqualEqual(t *testing.T) {
	expectToken(t, "==", 0, EqualEqual, "==")
}

func TestScanBangEqual(t *testing.T) {
	expectToken(t, "!=", 0, BangEqual, "!=")
}

func TestScanLessEqual(t *testing.T) {
	expectToken(t, "<=", 0, LessEqual, "<=")
}

func TestScanGreaterEqual(t *testing.T) {
	expectToken(t, ">=", 0, GreaterEqual, ">=")
}

func TestScanShiftLeft(t *testing.T) {
	expectToken(t, "<<", 0, ShiftLeft, "<<")
}

func TestScanShiftRight(t *testing.T) {
	expectToken(t, ">>", 0, ShiftRight, ">>")
}

func TestScanAmpAmp(t *testing.T) {
	expectToken(t, "&&", 0, AmpAmp, "&&")
}

func TestScanPipePipe(t *testing.T) {
	expectToken(t, "||", 0, PipePipe, "||")
}

func TestScanPlusEqual(t *testing.T) {
	expectToken(t, "+=", 0, PlusEqual, "+=")
}

func TestScanMinusEqual(t *testing.T) {
	expectToken(t, "-=", 0, MinusEqual, "-=")
}

func TestScanStarEqual(t *testing.T) {
	expectToken(t, "*=", 0, StarEqual, "*=")
}

func TestScanSlashEqual(t *testing.T) {
	expectToken(t, "/=", 0, SlashEqual, "/=")
}

func TestScanPercentEqual(t *testing.T) {
	expectToken(t, "%=", 0, PercentEqual, "%=")
}

func TestScanArrow(t *testing.T) {
	expectToken(t, "->", 0, Arrow, "->")
}

// === DELIMITERS ===

func TestScanDelimiters(t *testing.T) {
	expectTokens(t, "( ) { } [ ] : , . ?", []TokenKind{
		LeftParen, RightParen, LeftBrace, RightBrace,
		LeftBracket, RightBracket, Colon, Comma, Dot, Question, Eof,
	})
}

// === OPERATOR DISAMBIGUATION ===

func TestScanEqualVsEqualEqual(t *testing.T) {
	expectTokens(t, "= ==", []TokenKind{Equal, EqualEqual, Eof})
}

func TestScanBangVsBangEqual(t *testing.T) {
	expectTokens(t, "! !=", []TokenKind{Bang, BangEqual, Eof})
}

func TestScanLessVsLessEqual(t *testing.T) {
	expectTokens(t, "< <=", []TokenKind{Less, LessEqual, Eof})
}

func TestScanGreaterVsGreaterEqual(t *testing.T) {
	expectTokens(t, "> >=", []TokenKind{Greater, GreaterEqual, Eof})
}

func TestScanLessVsShiftLeft(t *testing.T) {
	expectTokens(t, "< <<", []TokenKind{Less, ShiftLeft, Eof})
}

func TestScanPlusVsPlusEqual(t *testing.T) {
	expectTokens(t, "+ +=", []TokenKind{Plus, PlusEqual, Eof})
}

func TestScanMinusVsMinusEqualVsArrow(t *testing.T) {
	expectTokens(t, "- -= ->", []TokenKind{Minus, MinusEqual, Arrow, Eof})
}

func TestScanAmpVsAmpAmp(t *testing.T) {
	expectTokens(t, "& &&", []TokenKind{Amp, AmpAmp, Eof})
}

func TestScanPipeVsPipePipe(t *testing.T) {
	expectTokens(t, "| ||", []TokenKind{Pipe, PipePipe, Eof})
}

// === LINE AND COLUMN TRACKING ===

func TestScanPositionFirstToken(t *testing.T) {
	expectPosition(t, "let", 0, 1, 1)
}

func TestScanPositionSecondToken(t *testing.T) {
	expectPosition(t, "let x", 1, 1, 5)
}

func TestScanPositionMultiLine(t *testing.T) {
	source := "let x\nlet y"
	expectPosition(t, source, 0, 1, 1) // let (line 1)
	expectPosition(t, source, 1, 1, 5) // x (line 1)
	expectPosition(t, source, 2, 2, 1) // let (line 2)
	expectPosition(t, source, 3, 2, 5) // y (line 2)
}

func TestScanPositionAfterComment(t *testing.T) {
	source := "// comment\n42"
	expectPosition(t, source, 0, 2, 1) // 42 is on line 2
}

func TestScanPositionEof(t *testing.T) {
	tokens := Scan("x")
	eof := tokens[len(tokens)-1]
	if eof.Kind != Eof {
		t.Fatal("last token should be Eof")
	}
}

// === ILLEGAL TOKENS ===

func TestScanIllegalCharacter(t *testing.T) {
	expectToken(t, "@", 0, Illegal, "@")
}

func TestScanIllegalDoesNotCrash(t *testing.T) {
	// Scanner should handle any input without panicking
	inputs := []string{"@", "#", "$", "\\", "\x00"}
	for _, input := range inputs {
		tokens := Scan(input)
		if len(tokens) < 1 {
			t.Errorf("Scan(%q) returned no tokens", input)
		}
	}
}

// === FULL STATEMENTS ===

func TestScanLetStatement(t *testing.T) {
	expectTokens(t, "let x = 42", []TokenKind{Let, Identifier, Equal, IntLiteral, Eof})
}

func TestScanConstStatement(t *testing.T) {
	expectTokens(t, "const pi = 3.14", []TokenKind{Const, Identifier, Equal, FloatLiteral, Eof})
}

func TestScanTypedVariable(t *testing.T) {
	expectTokens(t, "let count int = 0", []TokenKind{Let, Identifier, Identifier, Equal, IntLiteral, Eof})
}

func TestScanFunctionDefinition(t *testing.T) {
	source := "let add = (a int, b int) int {"
	expectTokens(t, source, []TokenKind{
		Let, Identifier, Equal,
		LeftParen, Identifier, Identifier, Comma, Identifier, Identifier, RightParen,
		Identifier, LeftBrace, Eof,
	})
}

func TestScanReturnStatement(t *testing.T) {
	expectTokens(t, "return a + b", []TokenKind{Return, Identifier, Plus, Identifier, Eof})
}

func TestScanIfElse(t *testing.T) {
	source := "if x > 0 { } else { }"
	expectTokens(t, source, []TokenKind{
		If, Identifier, Greater, IntLiteral, LeftBrace, RightBrace,
		Else, LeftBrace, RightBrace, Eof,
	})
}

func TestScanForLoop(t *testing.T) {
	expectTokens(t, "for i in range(10) {", []TokenKind{
		For, Identifier, In, Identifier, LeftParen, IntLiteral, RightParen, LeftBrace, Eof,
	})
}

func TestScanArrayLiteral(t *testing.T) {
	expectTokens(t, "[1, 2, 3]", []TokenKind{
		LeftBracket, IntLiteral, Comma, IntLiteral, Comma, IntLiteral, RightBracket, Eof,
	})
}

func TestScanRecordLiteral(t *testing.T) {
	expectTokens(t, `{name: "Alice", age: 30}`, []TokenKind{
		LeftBrace, Identifier, Colon, StringLiteral, Comma,
		Identifier, Colon, IntLiteral, RightBrace, Eof,
	})
}

func TestScanGuardAgainst(t *testing.T) {
	expectTokens(t, "guard result = divide(10, 0) against error {", []TokenKind{
		Guard, Identifier, Equal, Identifier, LeftParen, IntLiteral, Comma, IntLiteral, RightParen,
		Against, Identifier, LeftBrace, Eof,
	})
}

func TestScanTypeDefinition(t *testing.T) {
	expectTokens(t, "type Point = { x: int, y: int }", []TokenKind{
		Type, Identifier, Equal, LeftBrace,
		Identifier, Colon, Identifier, Comma,
		Identifier, Colon, Identifier,
		RightBrace, Eof,
	})
}

func TestScanFunctionTypeSignature(t *testing.T) {
	expectTokens(t, "(int, int) -> int", []TokenKind{
		LeftParen, Identifier, Comma, Identifier, RightParen, Arrow, Identifier, Eof,
	})
}

func TestScanCompoundAssignment(t *testing.T) {
	expectTokens(t, "x += 1", []TokenKind{Identifier, PlusEqual, IntLiteral, Eof})
}

func TestScanPropertyAccess(t *testing.T) {
	expectTokens(t, "person.name", []TokenKind{Identifier, Dot, Identifier, Eof})
}

func TestScanIndexAccess(t *testing.T) {
	expectTokens(t, "arr[0]", []TokenKind{Identifier, LeftBracket, IntLiteral, RightBracket, Eof})
}

func TestScanOptionalType(t *testing.T) {
	expectTokens(t, "int?", []TokenKind{Identifier, Question, Eof})
}

func TestScanBooleanLogic(t *testing.T) {
	expectTokens(t, "true and false or not none", []TokenKind{
		True, And, False, Or, Not, None, Eof,
	})
}

func TestScanBitwiseExpression(t *testing.T) {
	expectTokens(t, "a & b | c ^ ~d", []TokenKind{
		Identifier, Amp, Identifier, Pipe, Identifier, Caret, Tilde, Identifier, Eof,
	})
}

func TestScanShiftExpression(t *testing.T) {
	expectTokens(t, "x << 2 >> 1", []TokenKind{
		Identifier, ShiftLeft, IntLiteral, ShiftRight, IntLiteral, Eof,
	})
}

func TestScanUseExport(t *testing.T) {
	expectTokens(t, `use helper from "./utils"`, []TokenKind{
		Use, Identifier, From, StringLiteral, Eof,
	})
}

func TestScanStringConcatenation(t *testing.T) {
	expectTokens(t, `"hello" + " " + "world"`, []TokenKind{
		StringLiteral, Plus, StringLiteral, Plus, StringLiteral, Eof,
	})
}

func TestScanNegativeNumber(t *testing.T) {
	// Negative numbers are unary minus + number, not a single token
	expectTokens(t, "-42", []TokenKind{Minus, IntLiteral, Eof})
}

func TestScanMultipleStatements(t *testing.T) {
	source := "let x = 1\nlet y = 2\nshow(x + y)"
	expectTokens(t, source, []TokenKind{
		Let, Identifier, Equal, IntLiteral,
		Let, Identifier, Equal, IntLiteral,
		Identifier, LeftParen, Identifier, Plus, Identifier, RightParen, Eof,
	})
}

func TestScanIsKeywordNotIdentifier(t *testing.T) {
	expectTokens(t, "x is 5", []TokenKind{Identifier, Is, IntLiteral, Eof})
}

func TestScanThrowExpression(t *testing.T) {
	expectTokens(t, `throw "error"`, []TokenKind{Throw, StringLiteral, Eof})
}

func TestScanTrailingComma(t *testing.T) {
	expectTokens(t, "[1, 2, 3,]", []TokenKind{
		LeftBracket, IntLiteral, Comma, IntLiteral, Comma, IntLiteral, Comma, RightBracket, Eof,
	})
}

func TestScanDefaultParameter(t *testing.T) {
	// let greet = (name string, greeting string = "Hello") string {
	expectTokens(t, `(name string, greeting string = "Hello")`, []TokenKind{
		LeftParen, Identifier, Identifier, Comma,
		Identifier, Identifier, Equal, StringLiteral, RightParen, Eof,
	})
}

// === EDGE CASES ===

func TestScanAdjacentOperators(t *testing.T) {
	// No whitespace between tokens
	expectTokens(t, "1+2", []TokenKind{IntLiteral, Plus, IntLiteral, Eof})
}

func TestScanDotNotNumber(t *testing.T) {
	// A dot after a non-number is a delimiter, not a decimal point
	expectTokens(t, "person.name", []TokenKind{Identifier, Dot, Identifier, Eof})
}

func TestScanConsecutiveStrings(t *testing.T) {
	expectTokens(t, `"a" "b"`, []TokenKind{StringLiteral, StringLiteral, Eof})
}

func TestScanOnlyComment(t *testing.T) {
	expectTokens(t, "// nothing here", []TokenKind{Eof})
}

func TestScanCommentDoesNotEatNextLine(t *testing.T) {
	expectTokens(t, "// comment\n42", []TokenKind{IntLiteral, Eof})
}

func TestScanSlashNotComment(t *testing.T) {
	// Single slash is division, not a comment
	expectTokens(t, "10 / 2", []TokenKind{IntLiteral, Slash, IntLiteral, Eof})
}

func TestScanManyWhitespaceTypes(t *testing.T) {
	expectTokens(t, "  \t\n\r\n  42  \t  ", []TokenKind{IntLiteral, Eof})
}
