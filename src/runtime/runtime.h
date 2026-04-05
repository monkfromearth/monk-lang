/*
 * Monk Lang Runtime Library
 *
 * This is the C runtime linked into every compiled Monk program.
 * Generated .c files #include this header and call these functions.
 *
 * Design decisions reflected here:
 * - Value semantics: all values are copied on assignment (monk_deep_copy)
 * - Deep const: enforced at codegen level, not in the runtime
 * - Truthiness: false, none, 0 are falsy (monk_is_truthy)
 * - Graceful on reads: out-of-bounds returns none (monk_array_get, monk_string_index)
 * - Strict on operations: out-of-bounds write is an error (monk_array_set)
 * - Records have fixed shape: no adding new fields (monk_record_set)
 * - Error handling: setjmp/longjmp for guard/against/throw
 *
 * See spec/REFERENCE.md for the full language specification.
 */

#ifndef MONK_RUNTIME_H
#define MONK_RUNTIME_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <setjmp.h>

/* --- Value types --- */

typedef enum {
    MONK_INT,
    MONK_FLOAT,
    MONK_STRING,
    MONK_BOOL,
    MONK_NONE,
    MONK_ARRAY,
    MONK_RECORD,
    MONK_FUNCTION
} MonkValueKind;

/* Forward declarations — pointers only until MonkValue is defined */
typedef struct MonkArray MonkArray;
typedef struct MonkRecord MonkRecord;
typedef struct MonkRecordField MonkRecordField;
typedef struct MonkFunction MonkFunction;
typedef struct MonkValue MonkValue;

/* MonkValue must be defined before structs that embed it by value */
struct MonkValue {
    MonkValueKind kind;
    union {
        int64_t int_val;
        double float_val;
        char *str_val;          /* heap-allocated, null-terminated */
        bool bool_val;
        MonkArray *array_val;   /* heap-allocated */
        MonkRecord *record_val; /* heap-allocated */
        MonkFunction *func_val; /* heap-allocated */
    };
};

struct MonkArray {
    MonkValue *data;
    int64_t length;
};

struct MonkRecordField {
    const char *key;
    MonkValue value;
};

struct MonkRecord {
    MonkRecordField *fields;
    int64_t length;
};

typedef MonkValue (*MonkFuncPtr)(MonkValue *args, int64_t argc);

struct MonkFunction {
    MonkFuncPtr fn;
    MonkValue *captures;    /* captured environment (copied values) */
    int64_t capture_count;
};

/* --- Value constructors --- */

MonkValue monk_int(int64_t n);
MonkValue monk_float(double f);
MonkValue monk_string(const char *s);     /* copies the string */
MonkValue monk_bool(bool b);
MonkValue monk_none(void);
MonkValue monk_array(MonkValue *elements, int64_t length);  /* copies elements */
MonkValue monk_record(MonkRecordField *fields, int64_t length); /* copies fields */

/* --- Value operations --- */

/* Slow-path functions that handle heap types. Do not call directly — use the
 * inline wrappers below, which short-circuit for primitive (non-heap) values. */
MonkValue monk_deep_copy_heap(MonkValue v);
void monk_free_heap(MonkValue v);

/* Fast-path wrappers. Primitives (INT, FLOAT, BOOL, NONE) need no allocation,
 * so we inline the kind check and skip the call entirely. This is a ~10-40×
 * speedup for loops that mutate arrays of primitives — every array_set call
 * does both a free and a deep_copy, which were function calls per iteration. */
static inline MonkValue monk_deep_copy(MonkValue v) {
    switch (v.kind) {
    case MONK_INT:
    case MONK_FLOAT:
    case MONK_BOOL:
    case MONK_NONE:
        return v;
    default:
        return monk_deep_copy_heap(v);
    }
}

static inline void monk_free(MonkValue v) {
    switch (v.kind) {
    case MONK_INT:
    case MONK_FLOAT:
    case MONK_BOOL:
    case MONK_NONE:
        return;
    default:
        monk_free_heap(v);
    }
}
bool monk_is_truthy(MonkValue v);
const char *monk_type_name(MonkValue v);

/* --- Output & conversion --- */

void monk_show(MonkValue v);
MonkValue monk_to_string(MonkValue v);
MonkValue monk_to_int(MonkValue v);       /* string → int, strict */
MonkValue monk_to_float(MonkValue v);     /* string → float */

/* --- Comparison --- */

MonkValue monk_equal(MonkValue a, MonkValue b);
MonkValue monk_not_equal(MonkValue a, MonkValue b);
MonkValue monk_less(MonkValue a, MonkValue b);
MonkValue monk_greater(MonkValue a, MonkValue b);
MonkValue monk_less_equal(MonkValue a, MonkValue b);
MonkValue monk_greater_equal(MonkValue a, MonkValue b);

/* --- Arithmetic --- */

MonkValue monk_add(MonkValue a, MonkValue b);
MonkValue monk_sub(MonkValue a, MonkValue b);
MonkValue monk_mul(MonkValue a, MonkValue b);
MonkValue monk_div(MonkValue a, MonkValue b);
MonkValue monk_mod(MonkValue a, MonkValue b);
MonkValue monk_neg(MonkValue v);

/* --- String --- */

MonkValue monk_length(MonkValue v);
MonkValue monk_substring(MonkValue s, MonkValue start, MonkValue end);
MonkValue monk_index_of(MonkValue s, MonkValue search);
MonkValue monk_split(MonkValue s, MonkValue delim);
MonkValue monk_trim(MonkValue s);
MonkValue monk_to_upper_case(MonkValue s);
MonkValue monk_to_lower_case(MonkValue s);
MonkValue monk_string_concat(MonkValue a, MonkValue b);

/* --- Array --- */

MonkValue monk_array_get(MonkValue arr, MonkValue index);
void monk_array_set(MonkValue *arr, MonkValue index, MonkValue value);
MonkValue monk_append(MonkValue arr, MonkValue elem);
MonkValue monk_prepend(MonkValue arr, MonkValue elem);
MonkValue monk_pop(MonkValue arr);
MonkValue monk_drop(MonkValue arr, MonkValue n);
MonkValue monk_take(MonkValue arr, MonkValue n);
MonkValue monk_slice(MonkValue arr, MonkValue start, MonkValue end);
MonkValue monk_range(MonkValue n);

/* --- String indexing --- */

MonkValue monk_string_index(MonkValue s, MonkValue index);

/* --- Record --- */

MonkValue monk_record_get(MonkValue rec, const char *key);
void monk_record_set(MonkValue *rec, const char *key, MonkValue value);

/* --- Math --- */

MonkValue monk_abs(MonkValue v);
MonkValue monk_floor(MonkValue v);
MonkValue monk_ceil(MonkValue v);
MonkValue monk_round(MonkValue v);
MonkValue monk_sqrt(MonkValue v);
MonkValue monk_pow(MonkValue base, MonkValue exp);
MonkValue monk_log(MonkValue v);
MonkValue monk_log10(MonkValue v);
MonkValue monk_exp(MonkValue v);
MonkValue monk_min(MonkValue a, MonkValue b);
MonkValue monk_max(MonkValue a, MonkValue b);
MonkValue monk_sin(MonkValue v);
MonkValue monk_cos(MonkValue v);
MonkValue monk_tan(MonkValue v);
MonkValue monk_asin(MonkValue v);
MonkValue monk_acos(MonkValue v);
MonkValue monk_atan(MonkValue v);

/* --- Type checking --- */

MonkValue monk_typeof(MonkValue v);
MonkValue monk_is_number(MonkValue v);
MonkValue monk_is_string(MonkValue v);
MonkValue monk_is_boolean(MonkValue v);
MonkValue monk_is_array(MonkValue v);
MonkValue monk_is_record(MonkValue v);
MonkValue monk_is_function(MonkValue v);
MonkValue monk_is_none(MonkValue v);

/* --- File system & environment --- */

MonkValue monk_file_read(MonkValue path);
MonkValue monk_file_write(MonkValue path, MonkValue content);
MonkValue monk_file_exists(MonkValue path);
MonkValue monk_env_get(MonkValue name);
void monk_exit(MonkValue code);
MonkValue monk_args(void);

/* --- Error handling (guard/against/throw) --- */

/*
 * Error handling uses setjmp/longjmp. A guard block calls setjmp to save
 * a checkpoint. throw calls longjmp to jump back to it. The error value
 * is stored in a thread-local.
 *
 * Usage in generated C:
 *
 *   MonkGuardContext _guard_ctx;
 *   if (monk_guard_begin(&_guard_ctx) == 0) {
 *       // try expression
 *       result = some_call();
 *       monk_guard_end(&_guard_ctx);
 *   } else {
 *       // against block — error is in monk_current_error()
 *       MonkValue error = monk_current_error();
 *       result = fallback;
 *   }
 */

typedef struct {
    jmp_buf buf;
    void *prev;  /* previous guard context (linked list stack) */
} MonkGuardContext;

void monk_guard_begin_ctx(MonkGuardContext *ctx);
void monk_guard_end(MonkGuardContext *ctx);
void monk_throw(MonkValue error);
MonkValue monk_current_error(void);

/* Macro for the setjmp pattern */
#define monk_guard_begin(ctx) (monk_guard_begin_ctx(ctx), setjmp((ctx)->buf))

#endif /* MONK_RUNTIME_H */
