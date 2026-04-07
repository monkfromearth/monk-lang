package codegen

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
)

// arrayPtrField returns the MonkValue union field name for a typed-array storage
// kind. Used in the backing-store path: arr.{field}->data[i].
func arrayPtrField(s storageKind) string {
	switch s {
	case storeIntArray:
		return "int_array_val"
	case storeFloatArray:
		return "float_array_val"
	case storeBoolArray:
		return "bool_array_val"
	}
	return "array_val"
}

// elemZero returns the C zero literal for a typed-array element kind.
func elemZero(s storageKind) string {
	switch s {
	case storeInt:
		return "(int64_t)0"
	case storeFloat:
		return "(double)0.0"
	case storeBool:
		return "false"
	}
	return "monk_none()"
}

// recordField looks up a field in a typed record and returns its index (into
// the runtime fields[] array) and the corresponding storageKind for the field
// value. Returns (-1, storeBoxed) when the type is unknown, not a record, or
// the field name is not present.
//
// The index is ONLY valid if record literals for this type are emitted with
// fields in type-declaration order (enforced by emitExpr's RecordExpr branch).
func recordField(t *types.Type, name string) (int, storageKind) {
	if t == nil || t.Kind != types.KindRecord {
		return -1, storeBoxed
	}
	for i, f := range t.Fields {
		if f.Name == name {
			return i, storageFor(f.Type)
		}
	}
	return -1, storeBoxed
}

// emitIndexTyped handles arr[i] when arr is a typed array (storeIntArray etc.).
// Emits a bounds-checked inline element read returning a raw scalar, using the
// typed backing-store path: arr.int_array_val->data[i] — a direct pointer
// dereference with no MonkValue union overhead.
// Out-of-bounds panics (strict), consistent with "operating on invalid data = error".
//
// Falls back to the classic boxed path when the object's storage is unknown.
func (g *generator) emitIndexTyped(e *syntax.IndexExpr) (string, storageKind) {
	objCode, objKind := g.emitExprTyped(e.Object)
	elemSt := elemStorageFor(objKind)
	if elemSt == storeBoxed {
		return g.emitExpr(e), storeBoxed
	}

	ptrField := arrayPtrField(objKind)
	zero := elemZero(elemSt)
	idxCode, idxKind := g.emitExprTyped(e.Index)
	idxC := coerce(idxCode, idxKind, storeInt)

	// Simple ident: use directly — no temp needed for the object.
	if arrIdent, isIdent := e.Object.(*syntax.IdentExpr); isIdent {
		// Bounds-check elision: if we can statically prove 0 <= idx < length,
		// skip the runtime check and emit a direct data[] access.
		// The index is guaranteed to be a pure arithmetic expression (loop
		// counters + constants) by isBoundedSafe's range analysis, so we can
		// inline it directly — no statement-expression wrapper, no temp.
		// This lets the compiler hoist, CSE, and vectorize freely.
		if g.constVals != nil && g.isBoundedSafe(arrIdent, e.Index) {
			return fmt.Sprintf("%s.%s->data[%s]", objCode, ptrField, idxC), elemSt
		}
		tidx := g.newTemp()
		return fmt.Sprintf(
			"({int64_t %s=%s; (%s<0||%s>=%s.%s->length)?(monk_panic(\"index out of bounds\"),%s):%s.%s->data[%s];})",
			tidx, idxC, tidx, tidx, objCode, ptrField, zero, objCode, ptrField, tidx,
		), elemSt
	}
	// Complex expression: stash in a MonkValue temp to avoid double evaluation.
	tobj := g.newTemp()
	tidx := g.newTemp()
	return fmt.Sprintf(
		"({MonkValue %s=%s; int64_t %s=%s; (%s<0||%s>=%s.%s->length)?(monk_panic(\"index out of bounds\"),%s):%s.%s->data[%s];})",
		tobj, objCode, tidx, idxC, tidx, tidx, tobj, ptrField, zero, tobj, ptrField, tidx,
	), elemSt
}

// emitPropertyTyped handles record.field access when the object's type is
// statically known. For scalar fields (int/float/bool) it extracts the raw C
// value directly from the fields array by index, avoiding the runtime
// monk_record_get strcmp loop entirely. For boxed fields it returns the
// MonkValue via the same index path but with a deep copy (storeBoxed).
//
// Falls back to the classic boxed path when type info is unavailable.
func (g *generator) emitPropertyTyped(e *syntax.PropertyExpr) (string, storageKind) {
	objType, ok := g.info.Types[e.Object]
	if !ok || objType == nil {
		return g.emitExpr(e), storeBoxed
	}
	idx, fieldSt := recordField(objType, e.Property)
	if idx < 0 {
		return g.emitExpr(e), storeBoxed
	}

	// Simple ident object: access directly, no temp.
	if _, isIdent := e.Object.(*syntax.IdentExpr); isIdent {
		obj := g.emitExpr(e.Object)
		fieldVal := fmt.Sprintf("%s.record_val->fields[%d].value", obj, idx)
		switch fieldSt {
		case storeInt:
			return fieldVal + ".int_val", storeInt
		case storeFloat:
			return fieldVal + ".float_val", storeFloat
		case storeBool:
			return fieldVal + ".bool_val", storeBool
		}
		// Boxed field: deep copy so caller can free/store safely.
		return fmt.Sprintf("monk_deep_copy(%s)", fieldVal), storeBoxed
	}

	// Complex object: stash in a temp to avoid double evaluation.
	obj := g.emitExpr(e.Object)
	tobj := g.newTemp()
	fieldVal := fmt.Sprintf("%s.record_val->fields[%d].value", tobj, idx)
	switch fieldSt {
	case storeInt:
		return fmt.Sprintf("({MonkValue %s=%s; %s.int_val;})", tobj, obj, fieldVal), storeInt
	case storeFloat:
		return fmt.Sprintf("({MonkValue %s=%s; %s.float_val;})", tobj, obj, fieldVal), storeFloat
	case storeBool:
		return fmt.Sprintf("({MonkValue %s=%s; %s.bool_val;})", tobj, obj, fieldVal), storeBool
	}
	return fmt.Sprintf("({MonkValue %s=%s; monk_deep_copy(%s);})", tobj, obj, fieldVal), storeBoxed
}
