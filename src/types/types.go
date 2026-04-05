// Package types implements static type checking for Monk Lang.
//
// The checker runs between parse and codegen. It walks the AST, assigns a Type
// to every expression, and reports TypeError on mismatch. Codegen remains
// type-agnostic (still emits tagged-union MonkValue everywhere) — the checker
// only catches bugs at compile time. Unboxed codegen comes in a later phase.
//
// Type representation is a small sum type (via the Kind field). We don't use
// Go interfaces because every Monk type fits a fixed set of shapes.
//
// Core rules enforced here (from spec/REFERENCE.md):
//
//  1. First-assignment type inference: `let x = 42` locks x to int.
//  2. Consistency on reassignment: `x = "hi"` on an int variable errors.
//  3. Arrays are homogeneous (`[1, "x"]` errors at the literal, and
//     `append(int[], "x")` errors at the call).
//  4. Numeric widening: int can flow into a float slot, but not the reverse.
//  5. Optional (T?): accepts T or none.
//  6. Typed records: shape must match exactly, field types checked.
//  7. Function params and returns: checked against declared signature.
package types

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Kind is the tag of a Monk type.
type Kind int

const (
	KindAny    Kind = iota // untyped / "we don't know yet" — accepts anything
	KindInt                // int
	KindFloat              // float
	KindStr                // string
	KindBool               // boolean
	KindNone               // none
	KindArray              // T[]
	KindRecord             // { field: T, ... }
	KindFunc               // (T, T) -> T
)

// Type represents a Monk type. Heap types carry their shape inline — no
// pointer indirection, no interface dispatch. Two types are equal if they
// agree on Kind AND shape.
type Type struct {
	Kind     Kind
	Optional bool // T? — accepts T or none
	// Kind-specific fields (only the relevant one is populated):
	Elem   *Type            // Array element type
	Fields []RecordTypeField // Record fields (ordered)
	Params []*Type          // Func params
	Return *Type            // Func return
	// RecordName is "" for anonymous records (literals like {x: 1, y: 2}).
	// For `type Point = {x: int, y: int}; let p Point = ...`, the name is "Point".
	// Typed records reject unknown fields; anonymous records read-graceful, write-strict.
	RecordName string
}

// RecordTypeField is a name-type pair inside a Record type.
type RecordTypeField struct {
	Name string
	Type *Type
}

// Primitive type constructors (no allocation for common cases).
var (
	Any    = &Type{Kind: KindAny}
	Int    = &Type{Kind: KindInt}
	Float  = &Type{Kind: KindFloat}
	Str    = &Type{Kind: KindStr}
	Bool   = &Type{Kind: KindBool}
	None   = &Type{Kind: KindNone}
	IntOpt = &Type{Kind: KindInt, Optional: true}
)

// ArrayOf returns T[].
func ArrayOf(t *Type) *Type { return &Type{Kind: KindArray, Elem: t} }

// OptionalOf returns T? (clone, so we don't mutate the shared singleton).
func OptionalOf(t *Type) *Type {
	c := *t
	c.Optional = true
	return &c
}

// FuncType returns (params) -> ret.
func FuncType(params []*Type, ret *Type) *Type {
	return &Type{Kind: KindFunc, Params: params, Return: ret}
}

// String returns a human-readable type name for error messages.
func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	base := t.kindString()
	if t.Kind == KindArray {
		base = t.Elem.String() + "[]"
	}
	if t.Kind == KindFunc {
		ps := ""
		for i, p := range t.Params {
			if i > 0 {
				ps += ", "
			}
			ps += p.String()
		}
		base = "(" + ps + ") -> " + t.Return.String()
	}
	if t.Kind == KindRecord {
		if t.RecordName != "" {
			base = t.RecordName
		} else {
			base = "{...}"
		}
	}
	if t.Optional {
		return base + "?"
	}
	return base
}

func (t *Type) kindString() string {
	switch t.Kind {
	case KindAny:
		return "any"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindStr:
		return "string"
	case KindBool:
		return "boolean"
	case KindNone:
		return "none"
	case KindArray:
		return "array"
	case KindRecord:
		return "record"
	case KindFunc:
		return "function"
	}
	return "<unknown>"
}

// AssignableTo reports whether a value of type `src` can flow into a slot of
// type `dst`. This encodes Monk's compatibility rules:
//
//   - Any on either side = compatible (unknown/untyped flow).
//   - Identical primitives = compatible.
//   - int → float (numeric widening).
//   - none → T? (optional acceptance).
//   - T → T? (non-none value in optional slot).
//   - Arrays: element types assignable.
//   - Records: structural — dst's fields must all be present and type-assignable.
//     If dst has a name, extra fields in src are rejected. For anonymous dst
//     (literal assigned to untyped slot), extras are allowed.
//   - Functions: contravariant params, covariant return is overkill pre-1.0.
//     We require exact match for now.
func AssignableTo(src, dst *Type) bool {
	if src == nil || dst == nil {
		return false
	}
	if src.Kind == KindAny || dst.Kind == KindAny {
		return true
	}

	// Optional acceptance: none fits into any T?; T fits into T? if T fits T.
	if dst.Optional {
		if src.Kind == KindNone {
			return true
		}
		// strip optional from dst for further comparison
		dstStripped := *dst
		dstStripped.Optional = false
		return AssignableTo(src, &dstStripped)
	}
	// Non-optional dst cannot accept none.
	if src.Kind == KindNone {
		return false
	}

	// Numeric widening: int flows into float.
	if src.Kind == KindInt && dst.Kind == KindFloat {
		return true
	}

	if src.Kind != dst.Kind {
		return false
	}

	switch src.Kind {
	case KindArray:
		return AssignableTo(src.Elem, dst.Elem)
	case KindRecord:
		return recordAssignable(src, dst)
	case KindFunc:
		return funcExactMatch(src, dst)
	}
	return true // primitives already matched on kind
}

func recordAssignable(src, dst *Type) bool {
	// Every field the dst expects must exist in src with an assignable type.
	srcFields := make(map[string]*Type, len(src.Fields))
	for _, f := range src.Fields {
		srcFields[f.Name] = f.Type
	}
	for _, df := range dst.Fields {
		st, ok := srcFields[df.Name]
		if !ok {
			return false
		}
		if !AssignableTo(st, df.Type) {
			return false
		}
	}
	// If dst is a named (typed) record, src must NOT have extra fields.
	if dst.RecordName != "" && len(src.Fields) != len(dst.Fields) {
		return false
	}
	return true
}

// funcExactMatch enforces exact equality on params and return type.
// We deliberately don't use AssignableTo (which permits int->float widening)
// because a function typed (float) -> int being called via a (int) -> int
// slot would let callers pass a float where the callee expects an int.
// Proper contravariance requires dst.Params[i] to be assignable to
// src.Params[i], not the other way round — but that's overkill for pre-1.0,
// so we require exact matching until we have a concrete need for variance.
func funcExactMatch(src, dst *Type) bool {
	if len(src.Params) != len(dst.Params) {
		return false
	}
	for i := range src.Params {
		if !Equal(src.Params[i], dst.Params[i]) {
			return false
		}
	}
	return Equal(src.Return, dst.Return)
}

// Equal reports whether two types are structurally identical.
func Equal(a, b *Type) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind || a.Optional != b.Optional {
		return false
	}
	switch a.Kind {
	case KindArray:
		return Equal(a.Elem, b.Elem)
	case KindRecord:
		if len(a.Fields) != len(b.Fields) {
			return false
		}
		for i := range a.Fields {
			if a.Fields[i].Name != b.Fields[i].Name ||
				!Equal(a.Fields[i].Type, b.Fields[i].Type) {
				return false
			}
		}
		return true
	case KindFunc:
		if len(a.Params) != len(b.Params) {
			return false
		}
		for i := range a.Params {
			if !Equal(a.Params[i], b.Params[i]) {
				return false
			}
		}
		return Equal(a.Return, b.Return)
	}
	return true
}

// TypeError is a type-checking failure with source position.
type TypeError struct {
	Line, Column int
	Msg          string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("type error at line %d, column %d: %s", e.Line, e.Column, e.Msg)
}

func newTypeError(pos syntax.Pos, format string, args ...any) *TypeError {
	return &TypeError{
		Line:   pos.Line,
		Column: pos.Column,
		Msg:    fmt.Sprintf(format, args...),
	}
}
