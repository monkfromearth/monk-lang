/*
 * Math builtins. Each thin-wraps the C standard library.
 * Int/float mixing goes through monk_as_c_double (in value.c).
 */

#include "internal.h"
#include <math.h>
#include <stdint.h>

MonkValue monk_abs(MonkValue v) {
    if (v.kind == MONK_INT) {
        /* |INT64_MIN| overflows int64_t. Panic to match monk_neg. */
        if (v.int_val == INT64_MIN) monk_panic("integer overflow: abs(INT64_MIN)");
        return monk_int(v.int_val < 0 ? -v.int_val : v.int_val);
    }
    if (v.kind == MONK_FLOAT) return monk_float(fabs(v.float_val));
    monk_panic("abs: expected number"); return monk_none();
}

MonkValue monk_floor(MonkValue v) { return monk_int((int64_t)floor(monk_as_c_double(v))); }
MonkValue monk_ceil(MonkValue v)  { return monk_int((int64_t)ceil(monk_as_c_double(v))); }
MonkValue monk_round(MonkValue v) { return monk_int((int64_t)round(monk_as_c_double(v))); }

MonkValue monk_sqrt(MonkValue v) {
    double f = monk_as_c_double(v);
    if (f < 0) monk_panic("sqrt: cannot take square root of negative number");
    return monk_float(sqrt(f));
}

MonkValue monk_pow(MonkValue base, MonkValue exp) {
    return monk_float(pow(monk_as_c_double(base), monk_as_c_double(exp)));
}

MonkValue monk_log(MonkValue v) {
    double f = monk_as_c_double(v);
    if (f <= 0) monk_panic("log: argument must be positive");
    return monk_float(log(f));
}

MonkValue monk_log10(MonkValue v) {
    double f = monk_as_c_double(v);
    if (f <= 0) monk_panic("log10: argument must be positive");
    return monk_float(log10(f));
}

MonkValue monk_exp(MonkValue v)  { return monk_float(exp(monk_as_c_double(v))); }

MonkValue monk_min(MonkValue a, MonkValue b) {
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return a.int_val < b.int_val ? a : b;
    return monk_float(fmin(monk_as_c_double(a), monk_as_c_double(b)));
}

MonkValue monk_max(MonkValue a, MonkValue b) {
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return a.int_val > b.int_val ? a : b;
    return monk_float(fmax(monk_as_c_double(a), monk_as_c_double(b)));
}

MonkValue monk_sin(MonkValue v)  { return monk_float(sin(monk_as_c_double(v))); }
MonkValue monk_cos(MonkValue v)  { return monk_float(cos(monk_as_c_double(v))); }
MonkValue monk_tan(MonkValue v)  { return monk_float(tan(monk_as_c_double(v))); }
MonkValue monk_asin(MonkValue v) { return monk_float(asin(monk_as_c_double(v))); }
MonkValue monk_acos(MonkValue v) { return monk_float(acos(monk_as_c_double(v))); }
MonkValue monk_atan(MonkValue v) { return monk_float(atan(monk_as_c_double(v))); }
