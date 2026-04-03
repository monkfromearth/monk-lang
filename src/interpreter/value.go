package interpreter

import (
	"fmt"
	"strings"
)

// ValueKind identifies the runtime type of a Monk value.
type ValueKind int

const (
	IntValue ValueKind = iota
	FloatValue
	StringValue
	BoolValue
	NoneValue
	ArrayValue
	RecordValue
	FuncValue
)

// Value is a Monk runtime value. All values are immutable from the outside;
// mutation goes through the Environment.
type Value struct {
	Kind   ValueKind
	Int    int64
	Float  float64
	Str    string
	Bool   bool
	Array  []Value
	Record []RecordEntry
	Func   *FuncDef
}

// RecordEntry is a key-value pair in a record.
type RecordEntry struct {
	Key   string
	Value Value
}

// FuncDef is a runtime function value.
type FuncDef struct {
	Params  []string
	Env     *Environment // captured environment (closure snapshot)
	// Body is stored as an interface{} to avoid circular import with syntax.
	// The evaluator casts it to *syntax.BlockStmt.
	Body    any
}

// Predefined constants.
var (
	MonkNone  = Value{Kind: NoneValue}
	MonkTrue  = Value{Kind: BoolValue, Bool: true}
	MonkFalse = Value{Kind: BoolValue, Bool: false}
)

func IntVal(n int64) Value      { return Value{Kind: IntValue, Int: n} }
func FloatVal(f float64) Value  { return Value{Kind: FloatValue, Float: f} }
func StringVal(s string) Value  { return Value{Kind: StringValue, Str: s} }
func BoolVal(b bool) Value      { if b { return MonkTrue }; return MonkFalse }
func ArrayVal(elems []Value) Value { return Value{Kind: ArrayValue, Array: elems} }

func RecordVal(entries []RecordEntry) Value {
	return Value{Kind: RecordValue, Record: entries}
}

// DeepCopy returns an independent copy of the value (value semantics).
func (v Value) DeepCopy() Value {
	switch v.Kind {
	case ArrayValue:
		copied := make([]Value, len(v.Array))
		for i, elem := range v.Array {
			copied[i] = elem.DeepCopy()
		}
		return Value{Kind: ArrayValue, Array: copied}
	case RecordValue:
		copied := make([]RecordEntry, len(v.Record))
		for i, entry := range v.Record {
			copied[i] = RecordEntry{Key: entry.Key, Value: entry.Value.DeepCopy()}
		}
		return Value{Kind: RecordValue, Record: copied}
	default:
		return v // primitives are already values (no pointers)
	}
}

// IsTruthy implements Monk's truthiness rules: false, none, 0 are falsy.
func (v Value) IsTruthy() bool {
	switch v.Kind {
	case BoolValue:
		return v.Bool
	case NoneValue:
		return false
	case IntValue:
		return v.Int != 0
	default:
		return true
	}
}

// TypeName returns the Monk type name for typeof().
func (v Value) TypeName() string {
	switch v.Kind {
	case IntValue:
		return "int"
	case FloatValue:
		return "float"
	case StringValue:
		return "string"
	case BoolValue:
		return "boolean"
	case NoneValue:
		return "none"
	case ArrayValue:
		return "array"
	case RecordValue:
		return "record"
	case FuncValue:
		return "function"
	default:
		return "unknown"
	}
}

// String returns the show/to_string representation.
func (v Value) String() string {
	switch v.Kind {
	case IntValue:
		return fmt.Sprintf("%d", v.Int)
	case FloatValue:
		return fmt.Sprintf("%g", v.Float)
	case StringValue:
		return v.Str
	case BoolValue:
		if v.Bool {
			return "true"
		}
		return "false"
	case NoneValue:
		return "none"
	case ArrayValue:
		parts := make([]string, len(v.Array))
		for i, elem := range v.Array {
			if elem.Kind == StringValue {
				parts[i] = fmt.Sprintf("%q", elem.Str)
			} else {
				parts[i] = elem.String()
			}
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case RecordValue:
		parts := make([]string, len(v.Record))
		for i, entry := range v.Record {
			val := entry.Value.String()
			if entry.Value.Kind == StringValue {
				val = fmt.Sprintf("%q", entry.Value.Str)
			}
			parts[i] = entry.Key + ": " + val
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case FuncValue:
		return "<function>"
	default:
		return "<unknown>"
	}
}
