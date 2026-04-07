/*
 * Runtime test harness. Compile and run:
 *   cc -std=c11 value.c arith.c string.c container.c math.c builtins.c error.c \
 *       runtime_test.c -lm -o runtime_test && ./runtime_test
 */

#include "runtime.h"
#include <stdio.h>
#include <string.h>
#include <math.h>

static int tests_run = 0;
static int tests_passed = 0;

#define ASSERT(cond, msg) do { \
    tests_run++; \
    if (!(cond)) { \
        printf("FAIL: %s (line %d)\n", msg, __LINE__); \
    } else { \
        tests_passed++; \
    } \
} while(0)

#define ASSERT_INT(v, expected) do { \
    tests_run++; \
    if ((v).kind != MONK_INT || (v).int_val != (expected)) { \
        printf("FAIL: expected int %lld, got %s (line %d)\n", \
               (long long)(expected), monk_type_name(v), __LINE__); \
    } else { tests_passed++; } \
} while(0)

#define ASSERT_FLOAT(v, expected) do { \
    tests_run++; \
    if ((v).kind != MONK_FLOAT || fabs((v).float_val - (expected)) > 0.0001) { \
        printf("FAIL: expected float %g (line %d)\n", (expected), __LINE__); \
    } else { tests_passed++; } \
} while(0)

#define ASSERT_BOOL(v, expected) do { \
    tests_run++; \
    if ((v).kind != MONK_BOOL || (v).bool_val != (expected)) { \
        printf("FAIL: expected bool %s (line %d)\n", (expected) ? "true" : "false", __LINE__); \
    } else { tests_passed++; } \
} while(0)

#define ASSERT_STR(v, expected) do { \
    tests_run++; \
    if ((v).kind != MONK_STRING || strcmp((v).str_val, (expected)) != 0) { \
        printf("FAIL: expected string \"%s\" (line %d)\n", (expected), __LINE__); \
    } else { tests_passed++; } \
} while(0)

#define ASSERT_NONE(v) do { \
    tests_run++; \
    if ((v).kind != MONK_NONE) { \
        printf("FAIL: expected none (line %d)\n", __LINE__); \
    } else { tests_passed++; } \
} while(0)

void test_constructors(void) {
    ASSERT_INT(monk_int(42), 42);
    ASSERT_INT(monk_int(-5), -5);
    ASSERT_INT(monk_int(0), 0);
    ASSERT_FLOAT(monk_float(3.14), 3.14);
    ASSERT_STR(monk_string("hello"), "hello");
    ASSERT_STR(monk_string(""), "");
    ASSERT_BOOL(monk_bool(true), true);
    ASSERT_BOOL(monk_bool(false), false);
    ASSERT_NONE(monk_none());
}

void test_truthiness(void) {
    ASSERT(monk_is_truthy(monk_bool(true)) == true, "true is truthy");
    ASSERT(monk_is_truthy(monk_bool(false)) == false, "false is falsy");
    ASSERT(monk_is_truthy(monk_none()) == false, "none is falsy");
    ASSERT(monk_is_truthy(monk_int(0)) == false, "0 is falsy");
    ASSERT(monk_is_truthy(monk_int(1)) == true, "1 is truthy");
    ASSERT(monk_is_truthy(monk_int(-1)) == true, "-1 is truthy");
    ASSERT(monk_is_truthy(monk_string("")) == true, "\"\" is truthy");
    ASSERT(monk_is_truthy(monk_string("hello")) == true, "\"hello\" is truthy");
    ASSERT(monk_is_truthy(monk_float(0.0)) == true, "0.0 is truthy (only int 0 is falsy)");
}

void test_arithmetic(void) {
    ASSERT_INT(monk_add(monk_int(1), monk_int(2)), 3);
    ASSERT_INT(monk_sub(monk_int(10), monk_int(3)), 7);
    ASSERT_INT(monk_mul(monk_int(4), monk_int(5)), 20);
    ASSERT_INT(monk_div(monk_int(10), monk_int(3)), 3); /* truncates */
    ASSERT_INT(monk_div(monk_int(10), monk_int(2)), 5);
    ASSERT_INT(monk_mod(monk_int(7), monk_int(3)), 1);
    ASSERT_INT(monk_neg(monk_int(42)), -42);

    /* Float promotion */
    ASSERT_FLOAT(monk_add(monk_int(1), monk_float(2.5)), 3.5);
    ASSERT_FLOAT(monk_mul(monk_float(3.0), monk_int(2)), 6.0);
    ASSERT_FLOAT(monk_div(monk_float(10.0), monk_int(3)), 10.0/3.0);
}

void test_comparison(void) {
    ASSERT_BOOL(monk_equal(monk_int(5), monk_int(5)), true);
    ASSERT_BOOL(monk_equal(monk_int(5), monk_int(3)), false);
    ASSERT_BOOL(monk_not_equal(monk_int(5), monk_int(3)), true);
    ASSERT_BOOL(monk_less(monk_int(3), monk_int(5)), true);
    ASSERT_BOOL(monk_greater(monk_int(5), monk_int(3)), true);
    ASSERT_BOOL(monk_less_equal(monk_int(3), monk_int(3)), true);
    ASSERT_BOOL(monk_greater_equal(monk_int(5), monk_int(5)), true);

    /* String comparison (lexicographic) */
    ASSERT_BOOL(monk_equal(monk_string("abc"), monk_string("abc")), true);
    ASSERT_BOOL(monk_less(monk_string("abc"), monk_string("abd")), true);
    ASSERT_BOOL(monk_greater(monk_string("b"), monk_string("a")), true);

    /* none == none */
    ASSERT_BOOL(monk_equal(monk_none(), monk_none()), true);
    ASSERT_BOOL(monk_equal(monk_none(), monk_int(0)), false);
}

void test_string_ops(void) {
    ASSERT_INT(monk_length(monk_string("hello")), 5);
    ASSERT_INT(monk_length(monk_string("")), 0);

    ASSERT_STR(monk_substring(monk_string("hello"), monk_int(1), monk_int(4)), "ell");
    ASSERT_STR(monk_substring(monk_string("hi"), monk_int(0), monk_int(100)), "hi"); /* clamps */
    ASSERT_STR(monk_substring(monk_string("hi"), monk_int(5), monk_int(10)), "");

    ASSERT_INT(monk_index_of(monk_string("hello"), monk_string("ell")), 1);
    ASSERT_INT(monk_index_of(monk_string("hello"), monk_string("xyz")), -1);

    ASSERT_STR(monk_trim(monk_string("  hello  ")), "hello");
    ASSERT_STR(monk_to_upper_case(monk_string("hello")), "HELLO");
    ASSERT_STR(monk_to_lower_case(monk_string("HELLO")), "hello");

    ASSERT_STR(monk_string_concat(monk_string("hello"), monk_string(" world")), "hello world");
}

void test_string_index(void) {
    ASSERT_STR(monk_string_index(monk_string("hello"), monk_int(0)), "h");
    ASSERT_STR(monk_string_index(monk_string("hello"), monk_int(4)), "o");
    ASSERT_NONE(monk_string_index(monk_string("hello"), monk_int(10)));
    ASSERT_NONE(monk_string_index(monk_string("hello"), monk_int(-1)));
}

void test_arrays(void) {
    MonkValue elems[] = {monk_int(10), monk_int(20), monk_int(30)};
    MonkValue arr = monk_array(elems, 3);

    ASSERT_INT(monk_length(arr), 3);
    ASSERT_INT(monk_array_get(arr, monk_int(0)), 10);
    ASSERT_INT(monk_array_get(arr, monk_int(2)), 30);
    ASSERT_NONE(monk_array_get(arr, monk_int(10))); /* out of bounds = none */
    ASSERT_NONE(monk_array_get(arr, monk_int(-1)));

    /* append returns new array */
    MonkValue appended = monk_append(arr, monk_int(40));
    ASSERT_INT(monk_length(appended), 4);
    ASSERT_INT(monk_length(arr), 3); /* original unchanged */

    /* prepend */
    MonkValue prepended = monk_prepend(arr, monk_int(5));
    ASSERT_INT(monk_length(prepended), 4);
    ASSERT_INT(monk_array_get(prepended, monk_int(0)), 5);

    /* pop */
    MonkValue popped = monk_pop(arr);
    ASSERT_INT(monk_length(popped), 2);
    MonkValue pop_empty = monk_pop(monk_array(NULL, 0));
    ASSERT_INT(monk_length(pop_empty), 0); /* pop([]) = [] */

    /* drop/take */
    MonkValue dropped = monk_drop(arr, monk_int(1));
    ASSERT_INT(monk_length(dropped), 2);
    ASSERT_INT(monk_array_get(dropped, monk_int(0)), 20);

    MonkValue taken = monk_take(arr, monk_int(2));
    ASSERT_INT(monk_length(taken), 2);
    ASSERT_INT(monk_array_get(taken, monk_int(1)), 20);

    /* clamp */
    MonkValue drop_all = monk_drop(arr, monk_int(100));
    ASSERT_INT(monk_length(drop_all), 0);
    MonkValue take_all = monk_take(arr, monk_int(100));
    ASSERT_INT(monk_length(take_all), 3);

    /* slice */
    MonkValue sliced = monk_slice(arr, monk_int(1), monk_int(3));
    ASSERT_INT(monk_length(sliced), 2);
    ASSERT_INT(monk_array_get(sliced, monk_int(0)), 20);

    /* range */
    MonkValue range5 = monk_range(monk_int(5));
    ASSERT_INT(monk_length(range5), 5);
    ASSERT_INT(monk_array_get(range5, monk_int(0)), 0);
    ASSERT_INT(monk_array_get(range5, monk_int(4)), 4);
    ASSERT_INT(monk_length(monk_range(monk_int(0))), 0);
    ASSERT_INT(monk_length(monk_range(monk_int(-5))), 0);

    /* fill(n, value) — create array of n copies */
    MonkValue fill3 = monk_fill(monk_int(3), monk_bool(true));
    ASSERT_INT(monk_length(fill3), 3);
    /* fill(0, x) and fill(-1, x) return empty array (graceful) */
    ASSERT_INT(monk_length(monk_fill(monk_int(0), monk_int(0))), 0);
    ASSERT_INT(monk_length(monk_fill(monk_int(-1), monk_int(0))), 0);
}

void test_records(void) {
    MonkRecordField fields[] = {
        {.key = "name", .value = monk_string("Alice")},
        {.key = "age", .value = monk_int(30)},
    };
    MonkValue rec = monk_record(fields, 2);

    ASSERT_INT(monk_length(rec), 2);
    ASSERT_STR(monk_record_get(rec, "name"), "Alice");
    ASSERT_INT(monk_record_get(rec, "age"), 30);
    ASSERT_NONE(monk_record_get(rec, "missing")); /* graceful on reads */
}

void test_deep_copy(void) {
    MonkValue elems[] = {monk_int(1), monk_int(2), monk_int(3)};
    MonkValue a = monk_array(elems, 3);
    MonkValue b = monk_deep_copy(a);

    /* Modify b, a should be unchanged (value semantics) */
    monk_array_set(&b, monk_int(0), monk_int(99));
    ASSERT_INT(monk_array_get(a, monk_int(0)), 1); /* unchanged */
    ASSERT_INT(monk_array_get(b, monk_int(0)), 99);
}

void test_math(void) {
    ASSERT_INT(monk_abs(monk_int(-5)), 5);
    ASSERT_INT(monk_abs(monk_int(5)), 5);
    ASSERT_FLOAT(monk_abs(monk_float(-3.14)), 3.14);

    ASSERT_INT(monk_floor(monk_float(3.7)), 3);
    ASSERT_INT(monk_ceil(monk_float(3.2)), 4);
    ASSERT_INT(monk_round(monk_float(3.5)), 4);

    ASSERT_FLOAT(monk_sqrt(monk_int(16)), 4.0);
    ASSERT_FLOAT(monk_pow(monk_int(2), monk_int(10)), 1024.0);
    ASSERT_FLOAT(monk_log(monk_int(1)), 0.0);
    ASSERT_FLOAT(monk_log10(monk_int(100)), 2.0);
    ASSERT_FLOAT(monk_exp(monk_int(0)), 1.0);

    ASSERT_INT(monk_min(monk_int(3), monk_int(7)), 3);
    ASSERT_INT(monk_max(monk_int(3), monk_int(7)), 7);

    ASSERT_FLOAT(monk_sin(monk_int(0)), 0.0);
    ASSERT_FLOAT(monk_cos(monk_int(0)), 1.0);
}

void test_type_checking(void) {
    ASSERT_STR(monk_typeof(monk_int(42)), "int");
    ASSERT_STR(monk_typeof(monk_float(3.14)), "float");
    ASSERT_STR(monk_typeof(monk_string("hi")), "string");
    ASSERT_STR(monk_typeof(monk_bool(true)), "boolean");
    ASSERT_STR(monk_typeof(monk_none()), "none");

    ASSERT_BOOL(monk_is_number(monk_int(42)), true);
    ASSERT_BOOL(monk_is_number(monk_float(1.0)), true);
    ASSERT_BOOL(monk_is_number(monk_string("42")), false);
    ASSERT_BOOL(monk_is_string(monk_string("hi")), true);
    ASSERT_BOOL(monk_is_none(monk_none()), true);
    ASSERT_BOOL(monk_is_none(monk_int(0)), false);
}

void test_conversion(void) {
    /* String parsing (strict) */
    ASSERT_INT(monk_to_int(monk_string("42")), 42);
    ASSERT_INT(monk_to_int(monk_string("-5")), -5);
    ASSERT_FLOAT(monk_to_float(monk_string("3.14")), 3.14);

    /* Numeric coercion — added 2026-04-05, see spec/REFERENCE.md */
    ASSERT_INT(monk_to_int(monk_int(7)), 7);            /* int -> int (identity) */
    ASSERT_INT(monk_to_int(monk_float(3.9)), 3);        /* float -> int (truncate toward zero) */
    ASSERT_INT(monk_to_int(monk_float(-3.9)), -3);      /* negative truncates toward zero */
    ASSERT_FLOAT(monk_to_float(monk_int(5)), 5.0);      /* int -> float (widen) */
    ASSERT_FLOAT(monk_to_float(monk_float(2.5)), 2.5);  /* float -> float (identity) */

    /* to_string */
    ASSERT_STR(monk_to_string(monk_int(42)), "42");
    ASSERT_STR(monk_to_string(monk_bool(true)), "true");
    ASSERT_STR(monk_to_string(monk_none()), "none");
}

void test_guard_throw(void) {
    /* Test guard/against/throw pattern */
    MonkGuardContext ctx;
    MonkValue result;

    /* Success path */
    if (monk_guard_begin(&ctx) == 0) {
        result = monk_int(42);
        monk_guard_end(&ctx);
    } else {
        result = monk_int(-1);
    }
    ASSERT_INT(result, 42);

    /* Error path */
    if (monk_guard_begin(&ctx) == 0) {
        monk_throw(monk_string("test error"));
        result = monk_int(999); /* unreachable */
        monk_guard_end(&ctx);
    } else {
        MonkValue err = monk_current_error();
        ASSERT_STR(err, "test error");
        result = monk_int(-1);
    }
    ASSERT_INT(result, -1);
}

void test_split(void) {
    MonkValue result = monk_split(monk_string("a,b,c"), monk_string(","));
    ASSERT_INT(monk_length(result), 3);
    ASSERT_STR(monk_array_get(result, monk_int(0)), "a");
    ASSERT_STR(monk_array_get(result, monk_int(1)), "b");
    ASSERT_STR(monk_array_get(result, monk_int(2)), "c");

    /* Split with no delimiter found */
    MonkValue no_split = monk_split(monk_string("hello"), monk_string(","));
    ASSERT_INT(monk_length(no_split), 1);
    ASSERT_STR(monk_array_get(no_split, monk_int(0)), "hello");

    /* Split empty string */
    MonkValue empty_split = monk_split(monk_string(""), monk_string(","));
    ASSERT_INT(monk_length(empty_split), 1);
    ASSERT_STR(monk_array_get(empty_split, monk_int(0)), "");
}

void test_array_set(void) {
    MonkValue elems[] = {monk_int(1), monk_int(2), monk_int(3)};
    MonkValue arr = monk_array(elems, 3);
    monk_array_set(&arr, monk_int(1), monk_int(99));
    ASSERT_INT(monk_array_get(arr, monk_int(1)), 99);
    /* Other elements unchanged */
    ASSERT_INT(monk_array_get(arr, monk_int(0)), 1);
    ASSERT_INT(monk_array_get(arr, monk_int(2)), 3);
}

void test_record_set(void) {
    MonkRecordField fields[] = {
        {.key = "x", .value = monk_int(10)},
        {.key = "y", .value = monk_int(20)},
    };
    MonkValue rec = monk_record(fields, 2);
    monk_record_set(&rec, "x", monk_int(99));
    ASSERT_INT(monk_record_get(rec, "x"), 99);
    ASSERT_INT(monk_record_get(rec, "y"), 20); /* unchanged */
}

void test_edge_cases(void) {
    /* Nested arrays deep copy */
    MonkValue inner_elems[] = {monk_int(1), monk_int(2)};
    MonkValue inner = monk_array(inner_elems, 2);
    MonkValue outer_elems[] = {inner};
    MonkValue outer = monk_array(outer_elems, 1);
    MonkValue copy = monk_deep_copy(outer);
    /* Modify inner of copy, outer should be unchanged */
    MonkValue copied_inner = monk_array_get(copy, monk_int(0));
    ASSERT_INT(monk_array_get(copied_inner, monk_int(0)), 1);

    /* String concat with empty */
    ASSERT_STR(monk_string_concat(monk_string(""), monk_string("")), "");
    ASSERT_STR(monk_string_concat(monk_string("a"), monk_string("")), "a");

    /* show/to_string format for arrays with strings */
    MonkValue str_arr_elems[] = {monk_string("a"), monk_string("b")};
    MonkValue str_arr = monk_array(str_arr_elems, 2);
    MonkValue str_output = monk_to_string(str_arr);
    ASSERT_STR(str_output, "[\"a\", \"b\"]");

    /* show/to_string format for records */
    MonkRecordField rec_fields[] = {
        {.key = "name", .value = monk_string("Alice")},
    };
    MonkValue rec = monk_record(rec_fields, 1);
    MonkValue rec_output = monk_to_string(rec);
    ASSERT_STR(rec_output, "{name: \"Alice\"}");

    /* Nested guard/throw */
    MonkGuardContext ctx1, ctx2;
    MonkValue r1;

    if (monk_guard_begin(&ctx1) == 0) {
        if (monk_guard_begin(&ctx2) == 0) {
            monk_throw(monk_string("inner error"));
            monk_guard_end(&ctx2);
        } else {
            /* Inner catch — rethrow */
            monk_throw(monk_string("rethrown"));
            r1 = monk_int(-2);
        }
        monk_guard_end(&ctx1);
    } else {
        ASSERT_STR(monk_current_error(), "rethrown");
        r1 = monk_int(-1);
    }
    ASSERT_INT(r1, -1);

    /* Negation of negative */
    ASSERT_INT(monk_neg(monk_neg(monk_int(42))), 42);
    ASSERT_FLOAT(monk_neg(monk_float(-3.14)), 3.14);

    /* min/max with same values */
    ASSERT_INT(monk_min(monk_int(5), monk_int(5)), 5);
    ASSERT_INT(monk_max(monk_int(5), monk_int(5)), 5);

    /* Boolean equality */
    ASSERT_BOOL(monk_equal(monk_bool(true), monk_bool(true)), true);
    ASSERT_BOOL(monk_equal(monk_bool(true), monk_bool(false)), false);
}

/* ── Higher-order function tests ───────────────────────────────────────── */

static MonkValue cb_double(MonkFunction *self, MonkValue *args, int64_t argc) {
    (void)self; (void)argc;
    return monk_int(args[0].int_val * 2);
}
static MonkValue cb_is_even(MonkFunction *self, MonkValue *args, int64_t argc) {
    (void)self; (void)argc;
    return monk_bool(args[0].int_val % 2 == 0);
}
static MonkValue cb_add(MonkFunction *self, MonkValue *args, int64_t argc) {
    (void)self; (void)argc;
    return monk_int(args[0].int_val + args[1].int_val);
}

void test_higher_order(void) {
    MonkValue fn_double = monk_make_function(cb_double, NULL, 0);
    MonkValue fn_even   = monk_make_function(cb_is_even, NULL, 0);
    MonkValue fn_add    = monk_make_function(cb_add, NULL, 0);

    /* map: double each element */
    MonkValue arr = monk_array((MonkValue[]){monk_int(1), monk_int(2), monk_int(3)}, 3);
    MonkValue mapped = monk_map(arr, fn_double);
    ASSERT(mapped.kind == MONK_ARRAY, "map returns array");
    ASSERT(mapped.array_val->length == 3, "map preserves length");
    ASSERT_INT(mapped.array_val->data[0], 2);
    ASSERT_INT(mapped.array_val->data[1], 4);
    ASSERT_INT(mapped.array_val->data[2], 6);
    monk_free(mapped);

    /* map: empty array */
    MonkValue empty = monk_array(NULL, 0);
    MonkValue mapped_empty = monk_map(empty, fn_double);
    ASSERT(mapped_empty.array_val->length == 0, "map empty array");
    monk_free(mapped_empty);

    /* filter: keep even elements */
    MonkValue filtered = monk_filter(arr, fn_even);
    ASSERT(filtered.kind == MONK_ARRAY, "filter returns array");
    ASSERT(filtered.array_val->length == 1, "filter keeps 1 of 3");
    ASSERT_INT(filtered.array_val->data[0], 2);
    monk_free(filtered);

    /* filter: none pass */
    MonkValue odds = monk_array((MonkValue[]){monk_int(1), monk_int(3), monk_int(5)}, 3);
    MonkValue no_evens = monk_filter(odds, fn_even);
    ASSERT(no_evens.array_val->length == 0, "filter returns empty when none pass");
    monk_free(no_evens);
    monk_free(odds);

    /* reduce: sum */
    MonkValue sum = monk_reduce(arr, fn_add, monk_int(0));
    ASSERT_INT(sum, 6);
    monk_free(sum);

    /* reduce: empty array returns initial */
    MonkValue sum_empty = monk_reduce(empty, fn_add, monk_int(42));
    ASSERT_INT(sum_empty, 42);
    monk_free(sum_empty);
    monk_free(empty);

    monk_free(arr);
    monk_free(fn_double);
    monk_free(fn_even);
    monk_free(fn_add);
}

int main(void) {
    test_constructors();
    test_truthiness();
    test_arithmetic();
    test_comparison();
    test_string_ops();
    test_string_index();
    test_arrays();
    test_records();
    test_deep_copy();
    test_math();
    test_type_checking();
    test_conversion();
    test_guard_throw();
    test_split();
    test_array_set();
    test_record_set();
    test_edge_cases();
    test_higher_order();

    printf("\n%d/%d tests passed\n", tests_passed, tests_run);
    return tests_passed == tests_run ? 0 : 1;
}
