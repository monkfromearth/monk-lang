package syntax

import "testing"

// --- Helpers ---

func parse(t *testing.T, source string) *Program {
	t.Helper()
	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", source, err)
	}
	return prog
}

func parseExpr(t *testing.T, source string) Expr {
	t.Helper()
	prog := parse(t, source)
	if len(prog.Stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Stmts))
	}
	es, ok := prog.Stmts[0].(*ExprStmt)
	if !ok {
		t.Fatalf("expected ExprStmt, got %T", prog.Stmts[0])
	}
	return es.Expr
}

func parseStmt(t *testing.T, source string) Stmt {
	t.Helper()
	prog := parse(t, source)
	if len(prog.Stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Stmts))
	}
	return prog.Stmts[0]
}

// === LITERALS ===

func TestParseIntLiteral(t *testing.T) {
	expr := parseExpr(t, "42")
	num, ok := expr.(*NumberExpr)
	if !ok {
		t.Fatalf("expected NumberExpr, got %T", expr)
	}
	if num.Value != "42" || !num.IsInt {
		t.Errorf("expected int 42, got %s (isInt=%v)", num.Value, num.IsInt)
	}
	if num.Line != 1 || num.Column != 1 {
		t.Errorf("expected pos 1:1, got %d:%d", num.Line, num.Column)
	}
}

func TestParseFloatLiteral(t *testing.T) {
	num := parseExpr(t, "3.14").(*NumberExpr)
	if num.Value != "3.14" || num.IsInt {
		t.Errorf("expected float 3.14, got %s (isInt=%v)", num.Value, num.IsInt)
	}
}

func TestParseStringLiteral(t *testing.T) {
	str := parseExpr(t, `"hello"`).(*StringExpr)
	if str.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", str.Value)
	}
}

func TestParseTemplateLiteral(t *testing.T) {
	tmpl := parseExpr(t, "`world`").(*TemplateExpr)
	if tmpl.Value != "world" {
		t.Errorf("expected 'world', got '%s'", tmpl.Value)
	}
}

func TestParseTrueLiteral(t *testing.T) {
	b := parseExpr(t, "true").(*BoolExpr)
	if !b.Value { t.Error("expected true") }
}

func TestParseFalseLiteral(t *testing.T) {
	b := parseExpr(t, "false").(*BoolExpr)
	if b.Value { t.Error("expected false") }
}

func TestParseNoneLiteral(t *testing.T) {
	_, ok := parseExpr(t, "none").(*NoneExpr)
	if !ok { t.Fatal("expected NoneExpr") }
}

func TestParseIdentifier(t *testing.T) {
	id := parseExpr(t, "foo").(*IdentExpr)
	if id.Name != "foo" {
		t.Errorf("expected 'foo', got '%s'", id.Name)
	}
}

// === UNARY EXPRESSIONS ===

func TestParseUnaryMinus(t *testing.T) {
	un := parseExpr(t, "-42").(*UnaryExpr)
	if un.Op != Minus { t.Errorf("expected Minus, got %s", un.Op) }
	num := un.Operand.(*NumberExpr)
	if num.Value != "42" { t.Errorf("expected '42', got '%s'", num.Value) }
}

func TestParseUnaryNot(t *testing.T) {
	un := parseExpr(t, "not true").(*UnaryExpr)
	if un.Op != Not { t.Errorf("expected Not, got %s", un.Op) }
}

func TestParseUnaryBang(t *testing.T) {
	un := parseExpr(t, "!false").(*UnaryExpr)
	if un.Op != Bang { t.Errorf("expected Bang, got %s", un.Op) }
}

func TestParseUnaryTilde(t *testing.T) {
	un := parseExpr(t, "~x").(*UnaryExpr)
	if un.Op != Tilde { t.Errorf("expected Tilde, got %s", un.Op) }
}

// === BINARY EXPRESSIONS ===

func TestParseBinaryAdd(t *testing.T) {
	bin := parseExpr(t, "1 + 2").(*BinaryExpr)
	if bin.Op != Plus { t.Errorf("expected Plus, got %s", bin.Op) }
}

func TestParseBinarySub(t *testing.T) {
	bin := parseExpr(t, "5 - 3").(*BinaryExpr)
	if bin.Op != Minus { t.Errorf("expected Minus, got %s", bin.Op) }
}

func TestParseBinaryMul(t *testing.T) {
	bin := parseExpr(t, "2 * 3").(*BinaryExpr)
	if bin.Op != Star { t.Errorf("expected Star, got %s", bin.Op) }
}

func TestParseBinaryDiv(t *testing.T) {
	bin := parseExpr(t, "10 / 2").(*BinaryExpr)
	if bin.Op != Slash { t.Errorf("expected Slash, got %s", bin.Op) }
}

func TestParseBinaryMod(t *testing.T) {
	bin := parseExpr(t, "7 % 3").(*BinaryExpr)
	if bin.Op != Percent { t.Errorf("expected Percent, got %s", bin.Op) }
}

func TestParseBinaryComparison(t *testing.T) {
	tests := []struct{ source string; op TokenKind }{
		{"a == b", EqualEqual}, {"a != b", BangEqual},
		{"a < b", Less}, {"a > b", Greater},
		{"a <= b", LessEqual}, {"a >= b", GreaterEqual},
		{"a is b", Is},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			bin := parseExpr(t, tt.source).(*BinaryExpr)
			if bin.Op != tt.op { t.Errorf("expected %s, got %s", tt.op, bin.Op) }
		})
	}
}

func TestParseBinaryLogical(t *testing.T) {
	tests := []struct{ source string; op TokenKind }{
		{"a and b", And}, {"a or b", Or},
		{"a && b", AmpAmp}, {"a || b", PipePipe},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			bin := parseExpr(t, tt.source).(*BinaryExpr)
			if bin.Op != tt.op { t.Errorf("expected %s, got %s", tt.op, bin.Op) }
		})
	}
}

func TestParseBinaryBitwise(t *testing.T) {
	tests := []struct{ source string; op TokenKind }{
		{"a & b", Amp}, {"a | b", Pipe}, {"a ^ b", Caret},
		{"a << b", ShiftLeft}, {"a >> b", ShiftRight},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			bin := parseExpr(t, tt.source).(*BinaryExpr)
			if bin.Op != tt.op { t.Errorf("expected %s, got %s", tt.op, bin.Op) }
		})
	}
}

// === OPERATOR PRECEDENCE ===

func TestPrecedenceMulBeforeAdd(t *testing.T) {
	bin := parseExpr(t, "1 + 2 * 3").(*BinaryExpr)
	if bin.Op != Plus { t.Fatalf("top should be Plus, got %s", bin.Op) }
	right := bin.Right.(*BinaryExpr)
	if right.Op != Star { t.Errorf("right should be Star, got %s", right.Op) }
}

func TestPrecedenceParensOverride(t *testing.T) {
	bin := parseExpr(t, "(1 + 2) * 3").(*BinaryExpr)
	if bin.Op != Star { t.Fatalf("top should be Star, got %s", bin.Op) }
	left := bin.Left.(*BinaryExpr)
	if left.Op != Plus { t.Errorf("left should be Plus, got %s", left.Op) }
}

func TestPrecedenceComparisonBeforeLogical(t *testing.T) {
	bin := parseExpr(t, "a > 0 and b < 10").(*BinaryExpr)
	if bin.Op != And { t.Fatalf("top should be And, got %s", bin.Op) }
}

func TestPrecedenceAndBeforeOr(t *testing.T) {
	bin := parseExpr(t, "a or b and c").(*BinaryExpr)
	if bin.Op != Or { t.Fatalf("top should be Or, got %s", bin.Op) }
	right := bin.Right.(*BinaryExpr)
	if right.Op != And { t.Errorf("right should be And, got %s", right.Op) }
}

func TestPrecedenceBitwiseBeforeLogical(t *testing.T) {
	bin := parseExpr(t, "a & b and c").(*BinaryExpr)
	if bin.Op != And { t.Fatalf("top should be And, got %s", bin.Op) }
}

func TestPrecedenceEqualityBeforeBitwise(t *testing.T) {
	bin := parseExpr(t, "a == b & c").(*BinaryExpr)
	if bin.Op != Amp { t.Fatalf("top should be Amp, got %s", bin.Op) }
}

// === CALL EXPRESSIONS ===

func TestParseCallNoArgs(t *testing.T) {
	call := parseExpr(t, "foo()").(*CallExpr)
	if len(call.Args) != 0 { t.Errorf("expected 0 args, got %d", len(call.Args)) }
}

func TestParseCallWithArgs(t *testing.T) {
	call := parseExpr(t, "add(1, 2)").(*CallExpr)
	if len(call.Args) != 2 { t.Fatalf("expected 2 args, got %d", len(call.Args)) }
}

func TestParseCallTrailingComma(t *testing.T) {
	call := parseExpr(t, "add(1, 2,)").(*CallExpr)
	if len(call.Args) != 2 { t.Fatalf("expected 2 args, got %d", len(call.Args)) }
}

func TestParseCallChain(t *testing.T) {
	call := parseExpr(t, "foo()()").(*CallExpr)
	inner := call.Callee.(*CallExpr)
	_ = inner.Callee.(*IdentExpr)
}

// === INDEX AND PROPERTY ACCESS ===

func TestParseIndexAccess(t *testing.T) {
	idx := parseExpr(t, "arr[0]").(*IndexExpr)
	obj := idx.Object.(*IdentExpr)
	if obj.Name != "arr" { t.Errorf("expected 'arr', got '%s'", obj.Name) }
}

func TestParsePropertyAccess(t *testing.T) {
	prop := parseExpr(t, "person.name").(*PropertyExpr)
	if prop.Property != "name" { t.Errorf("expected 'name', got '%s'", prop.Property) }
}

func TestParseChainedPropertyAccess(t *testing.T) {
	outer := parseExpr(t, "a.b.c").(*PropertyExpr)
	if outer.Property != "c" { t.Errorf("expected 'c', got '%s'", outer.Property) }
	inner := outer.Object.(*PropertyExpr)
	if inner.Property != "b" { t.Errorf("expected 'b', got '%s'", inner.Property) }
}

func TestParseMethodCall(t *testing.T) {
	call := parseExpr(t, "obj.method(1)").(*CallExpr)
	prop := call.Callee.(*PropertyExpr)
	if prop.Property != "method" { t.Errorf("expected 'method', got '%s'", prop.Property) }
}

// === ARRAY AND RECORD LITERALS ===

func TestParseArrayEmpty(t *testing.T) {
	arr := parseExpr(t, "[]").(*ArrayExpr)
	if len(arr.Elements) != 0 { t.Errorf("expected 0 elements, got %d", len(arr.Elements)) }
}

func TestParseArray(t *testing.T) {
	arr := parseExpr(t, "[1, 2, 3]").(*ArrayExpr)
	if len(arr.Elements) != 3 { t.Fatalf("expected 3 elements, got %d", len(arr.Elements)) }
}

func TestParseArrayTrailingComma(t *testing.T) {
	arr := parseExpr(t, "[1, 2,]").(*ArrayExpr)
	if len(arr.Elements) != 2 { t.Fatalf("expected 2 elements, got %d", len(arr.Elements)) }
}

func TestParseRecordEmpty(t *testing.T) {
	rec := parseExpr(t, "{}").(*RecordExpr)
	if len(rec.Fields) != 0 { t.Errorf("expected 0 fields, got %d", len(rec.Fields)) }
}

func TestParseRecord(t *testing.T) {
	rec := parseExpr(t, `{name: "Alice", age: 30}`).(*RecordExpr)
	if len(rec.Fields) != 2 { t.Fatalf("expected 2 fields, got %d", len(rec.Fields)) }
	if rec.Fields[0].Key != "name" { t.Errorf("field 0: expected 'name', got '%s'", rec.Fields[0].Key) }
}

func TestParseRecordTrailingComma(t *testing.T) {
	rec := parseExpr(t, `{x: 1, y: 2,}`).(*RecordExpr)
	if len(rec.Fields) != 2 { t.Fatalf("expected 2 fields, got %d", len(rec.Fields)) }
}

// === VARIABLE DECLARATIONS ===

func TestParseLetDecl(t *testing.T) {
	decl := parseStmt(t, "let x = 42").(*VarDeclStmt)
	if decl.Name != "x" { t.Errorf("expected 'x', got '%s'", decl.Name) }
	if decl.IsConst { t.Error("expected let") }
	if decl.Line != 1 { t.Errorf("expected line 1, got %d", decl.Line) }
}

func TestParseConstDecl(t *testing.T) {
	decl := parseStmt(t, "const pi = 3.14").(*VarDeclStmt)
	if !decl.IsConst { t.Error("expected const") }
}

func TestParseTypedDecl(t *testing.T) {
	decl := parseStmt(t, "let count int = 0").(*VarDeclStmt)
	if decl.Type == nil { t.Fatal("expected type annotation") }
	if decl.Type.Name != "int" { t.Errorf("expected 'int', got '%s'", decl.Type.Name) }
}

func TestParseArrayTypeDecl(t *testing.T) {
	decl := parseStmt(t, "let nums int[] = [1, 2, 3]").(*VarDeclStmt)
	if decl.Type == nil { t.Fatal("expected type annotation") }
	if !decl.Type.IsArray { t.Error("expected array type") }
}

func TestParseOptionalTypeDecl(t *testing.T) {
	decl := parseStmt(t, "let maybe int? = none").(*VarDeclStmt)
	if decl.Type == nil { t.Fatal("expected type annotation") }
	if !decl.Type.Optional { t.Error("expected optional type") }
}

// === ASSIGNMENT STATEMENTS ===

func TestParseAssignment(t *testing.T) {
	stmt := parseStmt(t, "x = 42").(*AssignStmt)
	if stmt.Op != Equal { t.Errorf("expected Equal, got %s", stmt.Op) }
	target := stmt.Target.(*IdentExpr)
	if target.Name != "x" { t.Errorf("expected 'x', got '%s'", target.Name) }
}

func TestParseCompoundAssignment(t *testing.T) {
	tests := []struct{ source string; op TokenKind }{
		{"x += 1", PlusEqual}, {"x -= 1", MinusEqual},
		{"x *= 2", StarEqual}, {"x /= 2", SlashEqual},
		{"x %= 3", PercentEqual},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			stmt := parseStmt(t, tt.source).(*AssignStmt)
			if stmt.Op != tt.op { t.Errorf("expected %s, got %s", tt.op, stmt.Op) }
		})
	}
}

func TestParseIndexAssignment(t *testing.T) {
	stmt := parseStmt(t, "arr[0] = 99").(*AssignStmt)
	_ = stmt.Target.(*IndexExpr)
}

func TestParsePropertyAssignment(t *testing.T) {
	stmt := parseStmt(t, "person.name = \"Bob\"").(*AssignStmt)
	_ = stmt.Target.(*PropertyExpr)
}

// === IF STATEMENTS ===

func TestParseIf(t *testing.T) {
	ifStmt := parseStmt(t, "if x > 0 { }").(*IfStmt)
	if ifStmt.Else != nil { t.Error("expected no else") }
}

func TestParseIfElse(t *testing.T) {
	ifStmt := parseStmt(t, "if x > 0 { } else { }").(*IfStmt)
	if ifStmt.Else == nil { t.Fatal("expected else block") }
	_ = ifStmt.Else.(*BlockStmt)
}

func TestParseIfElseIf(t *testing.T) {
	ifStmt := parseStmt(t, "if x > 0 { } else if x < 0 { } else { }").(*IfStmt)
	elseIf := ifStmt.Else.(*IfStmt)
	if elseIf.Else == nil { t.Fatal("expected final else") }
}

// === LOOPS ===

func TestParseWhile(t *testing.T) {
	_ = parseStmt(t, "while x > 0 { }").(*WhileStmt)
}

func TestParseFor(t *testing.T) {
	forStmt := parseStmt(t, "for i in items { }").(*ForStmt)
	if forStmt.VarName != "i" { t.Errorf("expected 'i', got '%s'", forStmt.VarName) }
}

func TestParseBreak(t *testing.T) {
	prog := parse(t, "while true { break }")
	whileStmt := prog.Stmts[0].(*WhileStmt)
	_ = whileStmt.Body.Stmts[0].(*BreakStmt)
}

func TestParseContinue(t *testing.T) {
	prog := parse(t, "while true { continue }")
	whileStmt := prog.Stmts[0].(*WhileStmt)
	_ = whileStmt.Body.Stmts[0].(*ContinueStmt)
}

// === RETURN ===

func TestParseReturnValue(t *testing.T) {
	ret := parse(t, "return 42").Stmts[0].(*ReturnStmt)
	if ret.Value == nil { t.Fatal("expected return value") }
}

func TestParseReturnBare(t *testing.T) {
	ret := parse(t, "return").Stmts[0].(*ReturnStmt)
	if ret.Value != nil { t.Error("expected bare return") }
}

// === GUARD / AGAINST ===

func TestParseGuard(t *testing.T) {
	g := parseStmt(t, "guard result = divide(10, 0) against error { }").(*GuardStmt)
	if g.VarName != "result" { t.Errorf("expected 'result', got '%s'", g.VarName) }
	if g.ErrorName != "error" { t.Errorf("expected 'error', got '%s'", g.ErrorName) }
}

// === THROW ===

func TestParseThrow(t *testing.T) {
	th := parseExpr(t, `throw "error"`).(*ThrowExpr)
	str := th.Value.(*StringExpr)
	if str.Value != "error" { t.Errorf("expected 'error', got '%s'", str.Value) }
}

// === TYPE DECLARATIONS ===

func TestParseTypeAlias(t *testing.T) {
	td := parseStmt(t, "type UserId = int").(*TypeDeclStmt)
	if td.Name != "UserId" { t.Errorf("expected 'UserId', got '%s'", td.Name) }
	if td.Definition.AliasOf == nil { t.Fatal("expected alias") }
	if td.Definition.AliasOf.Name != "int" { t.Errorf("expected 'int', got '%s'", td.Definition.AliasOf.Name) }
}

func TestParseTypeRecord(t *testing.T) {
	td := parseStmt(t, "type Point = { x: int, y: int }").(*TypeDeclStmt)
	if len(td.Definition.Fields) != 2 { t.Fatalf("expected 2 fields, got %d", len(td.Definition.Fields)) }
	if td.Definition.Fields[0].Name != "x" { t.Errorf("field 0: expected 'x', got '%s'", td.Definition.Fields[0].Name) }
	if td.Definition.Fields[0].Type.Name != "int" { t.Errorf("field 0 type: expected 'int', got '%s'", td.Definition.Fields[0].Type.Name) }
}

// === FUNCTION EXPRESSIONS ===

func TestParseFuncExpr(t *testing.T) {
	fn := parseExpr(t, "(a int, b int) int { return a + b }").(*FuncExpr)
	if len(fn.Params) != 2 { t.Fatalf("expected 2 params, got %d", len(fn.Params)) }
	if fn.Params[0].Name != "a" { t.Errorf("param 0: expected 'a', got '%s'", fn.Params[0].Name) }
	if fn.ReturnType.Name != "int" { t.Errorf("return type: expected 'int', got '%s'", fn.ReturnType.Name) }
}

func TestParseFuncNoParams(t *testing.T) {
	fn := parseExpr(t, `() string { return "hello" }`).(*FuncExpr)
	if len(fn.Params) != 0 { t.Errorf("expected 0 params, got %d", len(fn.Params)) }
}

func TestParseFuncAsValue(t *testing.T) {
	decl := parseStmt(t, `let add = (a int, b int) int { return a + b }`).(*VarDeclStmt)
	if decl.Name != "add" { t.Errorf("expected 'add', got '%s'", decl.Name) }
	_ = decl.Value.(*FuncExpr)
}

func TestParseFuncNoneReturn(t *testing.T) {
	fn := parseExpr(t, "() none { }").(*FuncExpr)
	if fn.ReturnType.Name != "none" { t.Errorf("expected 'none', got '%s'", fn.ReturnType.Name) }
}

// === GROUPED EXPRESSIONS WITH CALLS ===
// Regression: parser used to commit to "function literal" on any `(ident(`,
// misreading grouped expressions like `(to_float(y) / 2.0)` as the start of
// a function with a function-typed parameter. Now disambiguates by looking
// for `->` after the closing `)` of the nested paren.

func TestParseGroupedCall(t *testing.T) {
	// Simple grouped call — must parse as BinaryExpr(Call, ...).
	e := parseExpr(t, `(f(x) + 1)`)
	if _, ok := e.(*BinaryExpr); !ok {
		t.Fatalf("expected BinaryExpr, got %T", e)
	}
}

func TestParseGroupedCallDivided(t *testing.T) {
	// The exact shape that broke during benchmark authoring.
	e := parseExpr(t, `(to_float(y) / 2.0)`)
	if _, ok := e.(*BinaryExpr); !ok {
		t.Fatalf("expected BinaryExpr, got %T", e)
	}
}

func TestParseNestedGroupedCalls(t *testing.T) {
	// Nested parens with calls should still parse as expressions, not fn literal.
	e := parseExpr(t, `((f(a) + g(b)) * h(c))`)
	if _, ok := e.(*BinaryExpr); !ok {
		t.Fatalf("expected BinaryExpr, got %T", e)
	}
}

func TestParseMatchRightParenNesting(t *testing.T) {
	// matchRightParen must handle nesting. Inner `(x+1)` has its own pair.
	e := parseExpr(t, `(f((x+1)) - 3)`)
	if _, ok := e.(*BinaryExpr); !ok {
		t.Fatalf("expected BinaryExpr, got %T", e)
	}
}

// === USE / EXPORT ===

func TestParseUseFrom(t *testing.T) {
	u := parseStmt(t, `use helper from "./utils"`).(*UseStmt)
	if len(u.Names) != 1 || u.Names[0] != "helper" { t.Errorf("expected [helper], got %v", u.Names) }
	if u.Source != "./utils" { t.Errorf("expected './utils', got '%s'", u.Source) }
}

func TestParseUseAlias(t *testing.T) {
	u := parseStmt(t, `use longName as short from "./mod"`).(*UseStmt)
	if u.Alias != "short" { t.Errorf("expected 'short', got '%s'", u.Alias) }
}

func TestParseUseNamed(t *testing.T) {
	u := parseStmt(t, `use { foo, bar } from "./mod"`).(*UseStmt)
	if len(u.Names) != 2 { t.Fatalf("expected 2 names, got %d", len(u.Names)) }
	if u.Names[0] != "foo" || u.Names[1] != "bar" {
		t.Errorf("expected [foo, bar], got %v", u.Names)
	}
}

func TestParseUseStar(t *testing.T) {
	u := parseStmt(t, `use * from "./all"`).(*UseStmt)
	if !u.Star { t.Error("expected Star import") }
	if u.Source != "./all" { t.Errorf("expected './all', got '%s'", u.Source) }
}

// === MULTIPLE STATEMENTS ===

func TestParseMultipleStatements(t *testing.T) {
	prog := parse(t, "let x = 1\nlet y = 2")
	if len(prog.Stmts) != 2 { t.Fatalf("expected 2 statements, got %d", len(prog.Stmts)) }
}

// === POSITION TRACKING ===

func TestParsePositionMultiLine(t *testing.T) {
	prog := parse(t, "let x = 1\nlet y = 2")
	s1 := prog.Stmts[0].(*VarDeclStmt)
	s2 := prog.Stmts[1].(*VarDeclStmt)
	if s1.Line != 1 { t.Errorf("stmt 1: expected line 1, got %d", s1.Line) }
	if s2.Line != 2 { t.Errorf("stmt 2: expected line 2, got %d", s2.Line) }
}

// === ERROR CASES ===

func TestParseError(t *testing.T) {
	_, err := Parse("let = 42")
	if err == nil { t.Fatal("expected parse error") }
}

func TestParseErrorUnmatchedParen(t *testing.T) {
	_, err := Parse("(1 + 2")
	if err == nil { t.Fatal("expected parse error") }
}

func TestParseErrorBreakOutsideLoop(t *testing.T) {
	_, err := Parse("break")
	if err == nil { t.Fatal("expected error for break outside loop") }
}

func TestParseErrorContinueOutsideLoop(t *testing.T) {
	_, err := Parse("continue")
	if err == nil { t.Fatal("expected error for continue outside loop") }
}

func TestParseBreakInsideLoopOk(t *testing.T) {
	_, err := Parse("while true { break }")
	if err != nil { t.Fatalf("break inside loop should be valid: %v", err) }
}

func TestParseContinueInsideForOk(t *testing.T) {
	_, err := Parse("for x in items { continue }")
	if err != nil { t.Fatalf("continue inside for should be valid: %v", err) }
}

func TestParseBreakInNestedIfInsideLoop(t *testing.T) {
	_, err := Parse("while true { if x { break } }")
	if err != nil { t.Fatalf("break in if inside loop should be valid: %v", err) }
}
