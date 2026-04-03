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
}

func TestParseFloatLiteral(t *testing.T) {
	expr := parseExpr(t, "3.14")
	num, ok := expr.(*NumberExpr)
	if !ok {
		t.Fatalf("expected NumberExpr, got %T", expr)
	}
	if num.Value != "3.14" || num.IsInt {
		t.Errorf("expected float 3.14, got %s (isInt=%v)", num.Value, num.IsInt)
	}
}

func TestParseStringLiteral(t *testing.T) {
	expr := parseExpr(t, `"hello"`)
	str, ok := expr.(*StringExpr)
	if !ok {
		t.Fatalf("expected StringExpr, got %T", expr)
	}
	if str.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", str.Value)
	}
}

func TestParseTemplateLiteral(t *testing.T) {
	expr := parseExpr(t, "`world`")
	tmpl, ok := expr.(*TemplateExpr)
	if !ok {
		t.Fatalf("expected TemplateExpr, got %T", expr)
	}
	if tmpl.Value != "world" {
		t.Errorf("expected 'world', got '%s'", tmpl.Value)
	}
}

func TestParseTrueLiteral(t *testing.T) {
	expr := parseExpr(t, "true")
	b, ok := expr.(*BoolExpr)
	if !ok {
		t.Fatalf("expected BoolExpr, got %T", expr)
	}
	if !b.Value {
		t.Error("expected true")
	}
}

func TestParseFalseLiteral(t *testing.T) {
	expr := parseExpr(t, "false")
	b, ok := expr.(*BoolExpr)
	if !ok {
		t.Fatalf("expected BoolExpr, got %T", expr)
	}
	if b.Value {
		t.Error("expected false")
	}
}

func TestParseNoneLiteral(t *testing.T) {
	expr := parseExpr(t, "none")
	_, ok := expr.(*NoneExpr)
	if !ok {
		t.Fatalf("expected NoneExpr, got %T", expr)
	}
}

func TestParseIdentifier(t *testing.T) {
	expr := parseExpr(t, "foo")
	id, ok := expr.(*IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr, got %T", expr)
	}
	if id.Name != "foo" {
		t.Errorf("expected 'foo', got '%s'", id.Name)
	}
}

// === UNARY EXPRESSIONS ===

func TestParseUnaryMinus(t *testing.T) {
	expr := parseExpr(t, "-42")
	un, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}
	if un.Op != Minus {
		t.Errorf("expected Minus, got %s", un.Op)
	}
	num, ok := un.Operand.(*NumberExpr)
	if !ok {
		t.Fatalf("operand: expected NumberExpr, got %T", un.Operand)
	}
	if num.Value != "42" {
		t.Errorf("expected '42', got '%s'", num.Value)
	}
}

func TestParseUnaryNot(t *testing.T) {
	expr := parseExpr(t, "not true")
	un, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}
	if un.Op != Not {
		t.Errorf("expected Not, got %s", un.Op)
	}
}

func TestParseUnaryBang(t *testing.T) {
	expr := parseExpr(t, "!false")
	un, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}
	if un.Op != Bang {
		t.Errorf("expected Bang, got %s", un.Op)
	}
}

func TestParseUnaryTilde(t *testing.T) {
	expr := parseExpr(t, "~x")
	un, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}
	if un.Op != Tilde {
		t.Errorf("expected Tilde, got %s", un.Op)
	}
}

// === BINARY EXPRESSIONS ===

func TestParseBinaryAdd(t *testing.T) {
	expr := parseExpr(t, "1 + 2")
	bin, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", expr)
	}
	if bin.Op != Plus {
		t.Errorf("expected Plus, got %s", bin.Op)
	}
}

func TestParseBinarySub(t *testing.T) {
	expr := parseExpr(t, "5 - 3")
	bin, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", expr)
	}
	if bin.Op != Minus {
		t.Errorf("expected Minus, got %s", bin.Op)
	}
}

func TestParseBinaryMul(t *testing.T) {
	expr := parseExpr(t, "2 * 3")
	bin := expr.(*BinaryExpr)
	if bin.Op != Star {
		t.Errorf("expected Star, got %s", bin.Op)
	}
}

func TestParseBinaryDiv(t *testing.T) {
	expr := parseExpr(t, "10 / 2")
	bin := expr.(*BinaryExpr)
	if bin.Op != Slash {
		t.Errorf("expected Slash, got %s", bin.Op)
	}
}

func TestParseBinaryMod(t *testing.T) {
	expr := parseExpr(t, "7 % 3")
	bin := expr.(*BinaryExpr)
	if bin.Op != Percent {
		t.Errorf("expected Percent, got %s", bin.Op)
	}
}

func TestParseBinaryComparison(t *testing.T) {
	tests := []struct {
		source string
		op     TokenKind
	}{
		{"a == b", EqualEqual},
		{"a != b", BangEqual},
		{"a < b", Less},
		{"a > b", Greater},
		{"a <= b", LessEqual},
		{"a >= b", GreaterEqual},
		{"a is b", Is},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			bin := parseExpr(t, tt.source).(*BinaryExpr)
			if bin.Op != tt.op {
				t.Errorf("expected %s, got %s", tt.op, bin.Op)
			}
		})
	}
}

func TestParseBinaryLogical(t *testing.T) {
	tests := []struct {
		source string
		op     TokenKind
	}{
		{"a and b", And},
		{"a or b", Or},
		{"a && b", AmpAmp},
		{"a || b", PipePipe},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			bin := parseExpr(t, tt.source).(*BinaryExpr)
			if bin.Op != tt.op {
				t.Errorf("expected %s, got %s", tt.op, bin.Op)
			}
		})
	}
}

func TestParseBinaryBitwise(t *testing.T) {
	tests := []struct {
		source string
		op     TokenKind
	}{
		{"a & b", Amp},
		{"a | b", Pipe},
		{"a ^ b", Caret},
		{"a << b", ShiftLeft},
		{"a >> b", ShiftRight},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			bin := parseExpr(t, tt.source).(*BinaryExpr)
			if bin.Op != tt.op {
				t.Errorf("expected %s, got %s", tt.op, bin.Op)
			}
		})
	}
}

// === OPERATOR PRECEDENCE ===

func TestPrecedenceMulBeforeAdd(t *testing.T) {
	// 1 + 2 * 3 should parse as 1 + (2 * 3)
	expr := parseExpr(t, "1 + 2 * 3")
	bin := expr.(*BinaryExpr)
	if bin.Op != Plus {
		t.Fatalf("top-level op should be Plus, got %s", bin.Op)
	}
	right := bin.Right.(*BinaryExpr)
	if right.Op != Star {
		t.Errorf("right op should be Star, got %s", right.Op)
	}
}

func TestPrecedenceParensOverride(t *testing.T) {
	// (1 + 2) * 3 should parse as (1 + 2) * 3
	expr := parseExpr(t, "(1 + 2) * 3")
	bin := expr.(*BinaryExpr)
	if bin.Op != Star {
		t.Fatalf("top-level op should be Star, got %s", bin.Op)
	}
	left := bin.Left.(*BinaryExpr)
	if left.Op != Plus {
		t.Errorf("left op should be Plus, got %s", left.Op)
	}
}

func TestPrecedenceComparisonBeforeLogical(t *testing.T) {
	// a > 0 and b < 10 should parse as (a > 0) and (b < 10)
	expr := parseExpr(t, "a > 0 and b < 10")
	bin := expr.(*BinaryExpr)
	if bin.Op != And {
		t.Fatalf("top-level op should be And, got %s", bin.Op)
	}
	left := bin.Left.(*BinaryExpr)
	if left.Op != Greater {
		t.Errorf("left op should be Greater, got %s", left.Op)
	}
	right := bin.Right.(*BinaryExpr)
	if right.Op != Less {
		t.Errorf("right op should be Less, got %s", right.Op)
	}
}

func TestPrecedenceAndBeforeOr(t *testing.T) {
	// a or b and c should parse as a or (b and c)
	expr := parseExpr(t, "a or b and c")
	bin := expr.(*BinaryExpr)
	if bin.Op != Or {
		t.Fatalf("top-level op should be Or, got %s", bin.Op)
	}
	right := bin.Right.(*BinaryExpr)
	if right.Op != And {
		t.Errorf("right op should be And, got %s", right.Op)
	}
}

func TestPrecedenceBitwiseBeforeLogical(t *testing.T) {
	// a & b and c should parse as (a & b) and c
	expr := parseExpr(t, "a & b and c")
	bin := expr.(*BinaryExpr)
	if bin.Op != And {
		t.Fatalf("top-level op should be And, got %s", bin.Op)
	}
	left := bin.Left.(*BinaryExpr)
	if left.Op != Amp {
		t.Errorf("left op should be Amp, got %s", left.Op)
	}
}

func TestPrecedenceEqualityBeforeBitwise(t *testing.T) {
	// a == b & c should parse as (a == b) & c
	expr := parseExpr(t, "a == b & c")
	bin := expr.(*BinaryExpr)
	if bin.Op != Amp {
		t.Fatalf("top-level op should be Amp, got %s", bin.Op)
	}
	left := bin.Left.(*BinaryExpr)
	if left.Op != EqualEqual {
		t.Errorf("left op should be EqualEqual, got %s", left.Op)
	}
}

// === CALL EXPRESSIONS ===

func TestParseCallNoArgs(t *testing.T) {
	expr := parseExpr(t, "foo()")
	call, ok := expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", expr)
	}
	if len(call.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(call.Args))
	}
}

func TestParseCallWithArgs(t *testing.T) {
	expr := parseExpr(t, "add(1, 2)")
	call := expr.(*CallExpr)
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Args))
	}
}

func TestParseCallTrailingComma(t *testing.T) {
	expr := parseExpr(t, "add(1, 2,)")
	call := expr.(*CallExpr)
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Args))
	}
}

func TestParseCallChain(t *testing.T) {
	// foo()() — call the result of a call
	expr := parseExpr(t, "foo()()")
	call, ok := expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", expr)
	}
	inner, ok := call.Callee.(*CallExpr)
	if !ok {
		t.Fatalf("callee: expected CallExpr, got %T", call.Callee)
	}
	_, ok = inner.Callee.(*IdentExpr)
	if !ok {
		t.Fatalf("inner callee: expected IdentExpr, got %T", inner.Callee)
	}
}

// === INDEX AND PROPERTY ACCESS ===

func TestParseIndexAccess(t *testing.T) {
	expr := parseExpr(t, "arr[0]")
	idx, ok := expr.(*IndexExpr)
	if !ok {
		t.Fatalf("expected IndexExpr, got %T", expr)
	}
	obj := idx.Object.(*IdentExpr)
	if obj.Name != "arr" {
		t.Errorf("expected 'arr', got '%s'", obj.Name)
	}
}

func TestParsePropertyAccess(t *testing.T) {
	expr := parseExpr(t, "person.name")
	prop, ok := expr.(*PropertyExpr)
	if !ok {
		t.Fatalf("expected PropertyExpr, got %T", expr)
	}
	if prop.Property != "name" {
		t.Errorf("expected 'name', got '%s'", prop.Property)
	}
}

func TestParseChainedPropertyAccess(t *testing.T) {
	expr := parseExpr(t, "a.b.c")
	outer := expr.(*PropertyExpr)
	if outer.Property != "c" {
		t.Errorf("expected 'c', got '%s'", outer.Property)
	}
	inner := outer.Object.(*PropertyExpr)
	if inner.Property != "b" {
		t.Errorf("expected 'b', got '%s'", inner.Property)
	}
}

func TestParseMethodCall(t *testing.T) {
	expr := parseExpr(t, "obj.method(1)")
	call := expr.(*CallExpr)
	prop := call.Callee.(*PropertyExpr)
	if prop.Property != "method" {
		t.Errorf("expected 'method', got '%s'", prop.Property)
	}
}

// === ARRAY AND RECORD LITERALS ===

func TestParseArrayEmpty(t *testing.T) {
	expr := parseExpr(t, "[]")
	arr, ok := expr.(*ArrayExpr)
	if !ok {
		t.Fatalf("expected ArrayExpr, got %T", expr)
	}
	if len(arr.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(arr.Elements))
	}
}

func TestParseArray(t *testing.T) {
	expr := parseExpr(t, "[1, 2, 3]")
	arr := expr.(*ArrayExpr)
	if len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestParseArrayTrailingComma(t *testing.T) {
	expr := parseExpr(t, "[1, 2,]")
	arr := expr.(*ArrayExpr)
	if len(arr.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr.Elements))
	}
}

func TestParseRecordEmpty(t *testing.T) {
	expr := parseExpr(t, "{}")
	rec, ok := expr.(*RecordExpr)
	if !ok {
		t.Fatalf("expected RecordExpr, got %T", expr)
	}
	if len(rec.Fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(rec.Fields))
	}
}

func TestParseRecord(t *testing.T) {
	expr := parseExpr(t, `{name: "Alice", age: 30}`)
	rec := expr.(*RecordExpr)
	if len(rec.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(rec.Fields))
	}
	if rec.Fields[0].Key != "name" {
		t.Errorf("field 0: expected 'name', got '%s'", rec.Fields[0].Key)
	}
	if rec.Fields[1].Key != "age" {
		t.Errorf("field 1: expected 'age', got '%s'", rec.Fields[1].Key)
	}
}

func TestParseRecordTrailingComma(t *testing.T) {
	expr := parseExpr(t, `{x: 1, y: 2,}`)
	rec := expr.(*RecordExpr)
	if len(rec.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(rec.Fields))
	}
}

// === VARIABLE DECLARATIONS ===

func TestParseLetDecl(t *testing.T) {
	stmt := parseStmt(t, "let x = 42")
	decl, ok := stmt.(*VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", stmt)
	}
	if decl.Name != "x" {
		t.Errorf("expected 'x', got '%s'", decl.Name)
	}
	if decl.IsConst {
		t.Error("expected let (not const)")
	}
}

func TestParseConstDecl(t *testing.T) {
	stmt := parseStmt(t, "const pi = 3.14")
	decl := stmt.(*VarDeclStmt)
	if !decl.IsConst {
		t.Error("expected const")
	}
	if decl.Name != "pi" {
		t.Errorf("expected 'pi', got '%s'", decl.Name)
	}
}

func TestParseTypedDecl(t *testing.T) {
	stmt := parseStmt(t, "let count int = 0")
	decl := stmt.(*VarDeclStmt)
	if decl.Type == nil {
		t.Fatal("expected type annotation")
	}
	if decl.Type.Name != "int" {
		t.Errorf("expected type 'int', got '%s'", decl.Type.Name)
	}
}

func TestParseArrayTypeDecl(t *testing.T) {
	stmt := parseStmt(t, "let nums int[] = [1, 2, 3]")
	decl := stmt.(*VarDeclStmt)
	if decl.Type == nil {
		t.Fatal("expected type annotation")
	}
	if !decl.Type.IsArray {
		t.Error("expected array type")
	}
	if decl.Type.Name != "int" {
		t.Errorf("expected 'int', got '%s'", decl.Type.Name)
	}
}

func TestParseOptionalTypeDecl(t *testing.T) {
	stmt := parseStmt(t, "let maybe int? = none")
	decl := stmt.(*VarDeclStmt)
	if decl.Type == nil {
		t.Fatal("expected type annotation")
	}
	if !decl.Type.Optional {
		t.Error("expected optional type")
	}
}

// === IF STATEMENTS ===

func TestParseIf(t *testing.T) {
	stmt := parseStmt(t, "if x > 0 { }")
	ifStmt, ok := stmt.(*IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt, got %T", stmt)
	}
	if ifStmt.Else != nil {
		t.Error("expected no else")
	}
}

func TestParseIfElse(t *testing.T) {
	stmt := parseStmt(t, "if x > 0 { } else { }")
	ifStmt := stmt.(*IfStmt)
	if ifStmt.Else == nil {
		t.Fatal("expected else block")
	}
	_, ok := ifStmt.Else.(*BlockStmt)
	if !ok {
		t.Fatalf("else: expected BlockStmt, got %T", ifStmt.Else)
	}
}

func TestParseIfElseIf(t *testing.T) {
	stmt := parseStmt(t, "if x > 0 { } else if x < 0 { } else { }")
	ifStmt := stmt.(*IfStmt)
	elseIf, ok := ifStmt.Else.(*IfStmt)
	if !ok {
		t.Fatalf("else: expected IfStmt (else-if), got %T", ifStmt.Else)
	}
	if elseIf.Else == nil {
		t.Fatal("expected final else block")
	}
}

// === LOOPS ===

func TestParseWhile(t *testing.T) {
	stmt := parseStmt(t, "while x > 0 { }")
	_, ok := stmt.(*WhileStmt)
	if !ok {
		t.Fatalf("expected WhileStmt, got %T", stmt)
	}
}

func TestParseFor(t *testing.T) {
	stmt := parseStmt(t, "for i in items { }")
	forStmt, ok := stmt.(*ForStmt)
	if !ok {
		t.Fatalf("expected ForStmt, got %T", stmt)
	}
	if forStmt.VarName != "i" {
		t.Errorf("expected 'i', got '%s'", forStmt.VarName)
	}
}

func TestParseBreak(t *testing.T) {
	prog := parse(t, "while true { break }")
	whileStmt := prog.Stmts[0].(*WhileStmt)
	_, ok := whileStmt.Body.Stmts[0].(*BreakStmt)
	if !ok {
		t.Fatalf("expected BreakStmt, got %T", whileStmt.Body.Stmts[0])
	}
}

func TestParseContinue(t *testing.T) {
	prog := parse(t, "while true { continue }")
	whileStmt := prog.Stmts[0].(*WhileStmt)
	_, ok := whileStmt.Body.Stmts[0].(*ContinueStmt)
	if !ok {
		t.Fatalf("expected ContinueStmt, got %T", whileStmt.Body.Stmts[0])
	}
}

// === RETURN ===

func TestParseReturnValue(t *testing.T) {
	prog := parse(t, "return 42")
	ret, ok := prog.Stmts[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected ReturnStmt, got %T", prog.Stmts[0])
	}
	if ret.Value == nil {
		t.Fatal("expected return value")
	}
}

func TestParseReturnBare(t *testing.T) {
	prog := parse(t, "return")
	ret := prog.Stmts[0].(*ReturnStmt)
	if ret.Value != nil {
		t.Error("expected bare return (no value)")
	}
}

// === GUARD / AGAINST ===

func TestParseGuard(t *testing.T) {
	stmt := parseStmt(t, "guard result = divide(10, 0) against error { }")
	g, ok := stmt.(*GuardStmt)
	if !ok {
		t.Fatalf("expected GuardStmt, got %T", stmt)
	}
	if g.VarName != "result" {
		t.Errorf("expected 'result', got '%s'", g.VarName)
	}
	if g.ErrorName != "error" {
		t.Errorf("expected 'error', got '%s'", g.ErrorName)
	}
}

// === THROW ===

func TestParseThrow(t *testing.T) {
	expr := parseExpr(t, `throw "error"`)
	th, ok := expr.(*ThrowExpr)
	if !ok {
		t.Fatalf("expected ThrowExpr, got %T", expr)
	}
	str := th.Value.(*StringExpr)
	if str.Value != "error" {
		t.Errorf("expected 'error', got '%s'", str.Value)
	}
}

// === TYPE DECLARATIONS ===

func TestParseTypeAlias(t *testing.T) {
	stmt := parseStmt(t, "type UserId = int")
	td, ok := stmt.(*TypeDeclStmt)
	if !ok {
		t.Fatalf("expected TypeDeclStmt, got %T", stmt)
	}
	if td.Name != "UserId" {
		t.Errorf("expected 'UserId', got '%s'", td.Name)
	}
	if td.Definition.AliasOf == nil {
		t.Fatal("expected alias type")
	}
	if td.Definition.AliasOf.Name != "int" {
		t.Errorf("expected 'int', got '%s'", td.Definition.AliasOf.Name)
	}
}

func TestParseTypeRecord(t *testing.T) {
	stmt := parseStmt(t, "type Point = { x: int, y: int }")
	td := stmt.(*TypeDeclStmt)
	if td.Name != "Point" {
		t.Errorf("expected 'Point', got '%s'", td.Name)
	}
	if len(td.Definition.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(td.Definition.Fields))
	}
}

// === FUNCTION EXPRESSIONS ===

func TestParseFuncExpr(t *testing.T) {
	expr := parseExpr(t, "(a int, b int) int { return a + b }")
	fn, ok := expr.(*FuncExpr)
	if !ok {
		t.Fatalf("expected FuncExpr, got %T", expr)
	}
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "a" {
		t.Errorf("param 0: expected 'a', got '%s'", fn.Params[0].Name)
	}
	if fn.ReturnType.Name != "int" {
		t.Errorf("return type: expected 'int', got '%s'", fn.ReturnType.Name)
	}
}

func TestParseFuncNoParams(t *testing.T) {
	expr := parseExpr(t, "() string { return \"hello\" }")
	fn := expr.(*FuncExpr)
	if len(fn.Params) != 0 {
		t.Errorf("expected 0 params, got %d", len(fn.Params))
	}
}

func TestParseFuncAsValue(t *testing.T) {
	stmt := parseStmt(t, `let add = (a int, b int) int { return a + b }`)
	decl := stmt.(*VarDeclStmt)
	if decl.Name != "add" {
		t.Errorf("expected 'add', got '%s'", decl.Name)
	}
	_, ok := decl.Value.(*FuncExpr)
	if !ok {
		t.Fatalf("value: expected FuncExpr, got %T", decl.Value)
	}
}

// === USE / EXPORT ===

func TestParseUseFrom(t *testing.T) {
	stmt := parseStmt(t, `use helper from "./utils"`)
	u, ok := stmt.(*UseStmt)
	if !ok {
		t.Fatalf("expected UseStmt, got %T", stmt)
	}
	if len(u.Names) != 1 || u.Names[0] != "helper" {
		t.Errorf("expected [helper], got %v", u.Names)
	}
	if u.Source != "./utils" {
		t.Errorf("expected './utils', got '%s'", u.Source)
	}
}

func TestParseUseAlias(t *testing.T) {
	stmt := parseStmt(t, `use longName as short from "./mod"`)
	u := stmt.(*UseStmt)
	if u.Alias != "short" {
		t.Errorf("expected alias 'short', got '%s'", u.Alias)
	}
}

// === MULTIPLE STATEMENTS ===

func TestParseMultipleStatements(t *testing.T) {
	prog := parse(t, "let x = 1\nlet y = 2")
	if len(prog.Stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(prog.Stmts))
	}
}

// === ERROR CASES ===

func TestParseError(t *testing.T) {
	_, err := Parse("let = 42") // missing identifier
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorUnmatchedParen(t *testing.T) {
	_, err := Parse("(1 + 2")
	if err == nil {
		t.Fatal("expected parse error for unmatched paren")
	}
}

func TestParseErrorAssignInIfCondition(t *testing.T) {
	_, err := Parse("if x = 5 { }")
	if err == nil {
		t.Fatal("expected error for assignment in if condition")
	}
}

func TestParseErrorAssignInWhileCondition(t *testing.T) {
	_, err := Parse("while x = 5 { }")
	if err == nil {
		t.Fatal("expected error for assignment in while condition")
	}
}

func TestParseErrorBreakOutsideLoop(t *testing.T) {
	_, err := Parse("break")
	if err == nil {
		t.Fatal("expected error for break outside loop")
	}
}

func TestParseErrorContinueOutsideLoop(t *testing.T) {
	_, err := Parse("continue")
	if err == nil {
		t.Fatal("expected error for continue outside loop")
	}
}

func TestParseBreakInsideLoopOk(t *testing.T) {
	_, err := Parse("while true { break }")
	if err != nil {
		t.Fatalf("break inside loop should be valid: %v", err)
	}
}

func TestParseContinueInsideForOk(t *testing.T) {
	_, err := Parse("for x in items { continue }")
	if err != nil {
		t.Fatalf("continue inside for should be valid: %v", err)
	}
}

func TestParseBreakInNestedIfInsideLoop(t *testing.T) {
	_, err := Parse("while true { if x { break } }")
	if err != nil {
		t.Fatalf("break in if inside loop should be valid: %v", err)
	}
}
