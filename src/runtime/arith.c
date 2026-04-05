/*
 * Arithmetic, comparison, and equality operators.
 *
 * Every binary op dispatches on argument kind. Mixed int+float coerces
 * via monk_as_c_double. Division guards against zero. String concat is
 * the only operator that ever escapes this file (into string.c).
 */

#include "internal.h"
#include <stdint.h>
#include <string.h>

/* --- Equality & comparison --- */

MonkValue monk_equal(MonkValue a, MonkValue b) {
    if (a.kind == MONK_NONE && b.kind == MONK_NONE) return monk_bool(true);
    if (a.kind == MONK_NONE || b.kind == MONK_NONE) return monk_bool(false);
    if (a.kind != b.kind) monk_panic("cannot compare different types with ==");
    switch (a.kind) {
    case MONK_INT:    return monk_bool(a.int_val == b.int_val);
    case MONK_FLOAT:  return monk_bool(a.float_val == b.float_val);
    case MONK_STRING: return monk_bool(strcmp(a.str_val, b.str_val) == 0);
    case MONK_BOOL:   return monk_bool(a.bool_val == b.bool_val);
    default:          monk_panic("cannot compare this type"); return monk_none();
    }
}

MonkValue monk_not_equal(MonkValue a, MonkValue b) {
    MonkValue eq = monk_equal(a, b);
    return monk_bool(!eq.bool_val);
}

MonkValue monk_less(MonkValue a, MonkValue b) {
    if (a.kind == MONK_NONE || b.kind == MONK_NONE) monk_panic("cannot order none");
    if (a.kind != b.kind) monk_panic("cannot compare different types");
    switch (a.kind) {
    case MONK_INT:    return monk_bool(a.int_val < b.int_val);
    case MONK_FLOAT:  return monk_bool(a.float_val < b.float_val);
    case MONK_STRING: return monk_bool(strcmp(a.str_val, b.str_val) < 0);
    default:          monk_panic("cannot order this type"); return monk_none();
    }
}

MonkValue monk_greater(MonkValue a, MonkValue b) {
    if (a.kind == MONK_NONE || b.kind == MONK_NONE) monk_panic("cannot order none");
    if (a.kind != b.kind) monk_panic("cannot compare different types");
    switch (a.kind) {
    case MONK_INT:    return monk_bool(a.int_val > b.int_val);
    case MONK_FLOAT:  return monk_bool(a.float_val > b.float_val);
    case MONK_STRING: return monk_bool(strcmp(a.str_val, b.str_val) > 0);
    default:          monk_panic("cannot order this type"); return monk_none();
    }
}

MonkValue monk_less_equal(MonkValue a, MonkValue b) {
    return monk_bool(!monk_greater(a, b).bool_val);
}

MonkValue monk_greater_equal(MonkValue a, MonkValue b) {
    return monk_bool(!monk_less(a, b).bool_val);
}

/* --- Arithmetic --- */

MonkValue monk_add(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT)
        return monk_float(monk_as_c_double(a) + monk_as_c_double(b));
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return monk_int(a.int_val + b.int_val);
    monk_panic("cannot add these types");
    return monk_none();
}

MonkValue monk_sub(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT)
        return monk_float(monk_as_c_double(a) - monk_as_c_double(b));
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return monk_int(a.int_val - b.int_val);
    monk_panic("cannot subtract these types");
    return monk_none();
}

MonkValue monk_mul(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT)
        return monk_float(monk_as_c_double(a) * monk_as_c_double(b));
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return monk_int(a.int_val * b.int_val);
    monk_panic("cannot multiply these types");
    return monk_none();
}

MonkValue monk_div(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT) {
        double denom = monk_as_c_double(b);
        if (denom == 0) monk_panic("division by zero");
        return monk_float(monk_as_c_double(a) / denom);
    }
    if (a.kind == MONK_INT && b.kind == MONK_INT) {
        if (b.int_val == 0) monk_panic("division by zero");
        /* Design decision: int/int = int (truncates toward zero) */
        return monk_int(a.int_val / b.int_val);
    }
    monk_panic("cannot divide these types");
    return monk_none();
}

MonkValue monk_mod(MonkValue a, MonkValue b) {
    if (a.kind != MONK_INT || b.kind != MONK_INT) monk_panic("modulo requires integers");
    if (b.int_val == 0) monk_panic("modulo by zero");
    return monk_int(a.int_val % b.int_val);
}

MonkValue monk_neg(MonkValue v) {
    if (v.kind == MONK_INT) {
        /* Negating INT64_MIN is undefined behavior in C (result doesn't fit
         * in int64_t). Panic instead of silently wrapping — integer overflow
         * is a real bug, not a graceful edge case. */
        if (v.int_val == INT64_MIN) monk_panic("integer overflow: cannot negate INT64_MIN");
        return monk_int(-v.int_val);
    }
    if (v.kind == MONK_FLOAT) return monk_float(-v.float_val);
    monk_panic("cannot negate this type");
    return monk_none();
}
