package syntax

import "testing"

func TestScanEmpty(t *testing.T) {
	tokens := Scan("")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token (EOF), got %d", len(tokens))
	}
	if tokens[0].Kind != Eof {
		t.Fatalf("expected EOF, got %s", tokens[0].Kind)
	}
}

func TestScanInteger(t *testing.T) {
	tokens := Scan("42")
	if len(tokens) != 2 { // INT, EOF
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != IntLiteral {
		t.Fatalf("expected IntLiteral, got %s", tokens[0].Kind)
	}
	if tokens[0].Text != "42" {
		t.Fatalf("expected text '42', got '%s'", tokens[0].Text)
	}
}

func TestScanOperators(t *testing.T) {
	tokens := Scan("+ - * / %")
	expected := []TokenKind{Plus, Minus, Star, Slash, Percent, Eof}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestScanKeywords(t *testing.T) {
	tokens := Scan("let const if else")
	expected := []TokenKind{Let, Const, If, Else, Eof}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestScanIdentifier(t *testing.T) {
	tokens := Scan("foo bar_baz")
	if tokens[0].Kind != Identifier {
		t.Fatalf("expected Identifier, got %s", tokens[0].Kind)
	}
	if tokens[0].Text != "foo" {
		t.Fatalf("expected 'foo', got '%s'", tokens[0].Text)
	}
	if tokens[1].Kind != Identifier {
		t.Fatalf("expected Identifier, got %s", tokens[1].Kind)
	}
	if tokens[1].Text != "bar_baz" {
		t.Fatalf("expected 'bar_baz', got '%s'", tokens[1].Text)
	}
}

func TestScanString(t *testing.T) {
	tokens := Scan(`"hello world"`)
	if tokens[0].Kind != StringLiteral {
		t.Fatalf("expected StringLiteral, got %s", tokens[0].Kind)
	}
	if tokens[0].Text != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", tokens[0].Text)
	}
}

func TestScanLineColumn(t *testing.T) {
	tokens := Scan("let x = 42")
	// let starts at column 1
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("'let': expected 1:1, got %d:%d", tokens[0].Line, tokens[0].Column)
	}
	// x starts at column 5
	if tokens[1].Line != 1 || tokens[1].Column != 5 {
		t.Errorf("'x': expected 1:5, got %d:%d", tokens[1].Line, tokens[1].Column)
	}
}

func TestScanLetStatement(t *testing.T) {
	tokens := Scan("let x = 42")
	expected := []TokenKind{Let, Identifier, Equal, IntLiteral, Eof}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}
