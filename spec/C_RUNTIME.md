# Monk C Runtime Reference

> API reference for the C runtime library linked into every compiled Monk program.
> Source: `src/runtime/` — see `src/runtime/INDEX.md` for the file map.

---

## Overview

The runtime is ~1,000 lines of C11, split across 8 library `.c` files plus the standalone `runtime_test.c` harness and 2 headers (`runtime.h` public API, `internal.h` shared helpers). Every Monk program links all library files. The codegen emits calls to these functions — Monk `+` becomes `monk_add()`, Monk `show` becomes `monk_show()`, etc.

**Scalar unboxed fast path (Phase 6).** When the type checker proves a variable is `int`/`float`/`bool`, codegen stores it as a raw `int64_t`/`double`/`bool` and emits raw C arithmetic — skipping the runtime entirely. The runtime is only called at boxing boundaries (`show`, `to_string`, etc.) and for heap types.

**Typed array fast path (Phase 6.5).** `int[]`, `float[]`, `bool[]` variables use `int64_t*`/`double*`/`bool*` backing stores (`MONK_INT_ARRAY`, `MONK_FLOAT_ARRAY`, `MONK_BOOL_ARRAY`). Element access emits direct pointer arithmetic. OOB panics. Typed arrays benchmark at ~2× C; generic arrays at ~12× C.

**Design rules enforced here:**
- Value semantics via `monk_deep_copy()` on every assignment; arrays use copy-on-write internally
- Truthiness: `false`, `none`, `0` are falsy
- Graceful on reads: out-of-bounds returns `none`
- Strict on writes: out-of-bounds write is a runtime error
- Fixed-shape records: no adding new fields
- Error handling: `setjmp`/`longjmp` for `guard`/`against`/`throw`

---

## MonkValue — The Tagged Union

Every value in Monk is a `MonkValue`. Primitives are inline. Heap types are pointers.

```c
typedef enum {
    MONK_INT,        // int64_t
    MONK_FLOAT,      // double
    MONK_STRING,     // char* (heap-allocated, null-terminated)
    MONK_BOOL,       // bool
    MONK_NONE,       // no payload
    MONK_ARRAY,      // MonkArray* — generic tagged-union elements
    MONK_RECORD,     // MonkRecord*
    MONK_FUNCTION,   // MonkFunction*
    MONK_INT_ARRAY,  // MonkIntArray*  — int64_t* backing store (Phase 6.5)
    MONK_FLOAT_ARRAY,// MonkFloatArray* — double* backing store
    MONK_BOOL_ARRAY  // MonkBoolArray*  — bool* backing store
} MonkValueKind;

struct MonkValue {
    MonkValueKind kind;
    union {
        int64_t          int_val;
        double           float_val;
        char            *str_val;
        bool             bool_val;
        MonkArray       *array_val;
        MonkRecord      *record_val;
        MonkFunction    *func_val;
        MonkIntArray    *int_array_val;   /* heap int64_t* elements */
        MonkFloatArray  *float_array_val; /* heap double* elements */
        MonkBoolArray   *bool_array_val;  /* heap bool* elements */
    };
};
```

### Heap Structs

```c
// Generic array — elements are MonkValue (tagged union, any type)
struct MonkArray {
    MonkValue *data;       // heap array of MonkValues
    int64_t    length;
    int32_t    refcount;   // copy-on-write sharing count
};

// Typed array backing stores — raw element types, no union overhead
// Created by monk_int_array_from() / monk_float_array_from() / monk_bool_array_from()
// when codegen assigns to an int[] / float[] / bool[] variable.
struct MonkIntArray {
    int64_t *data;
    int64_t  length;
    int32_t  refcount;
};
struct MonkFloatArray {
    double  *data;
    int64_t  length;
    int32_t  refcount;
};
struct MonkBoolArray {
    bool    *data;
    int64_t  length;
    int32_t  refcount;
};

struct MonkRecordField {
    const char *key;       // field name (string literal, not freed)
    MonkValue   value;
};

struct MonkRecord {
    MonkRecordField *fields;
    int64_t          length;   // number of fields
};

struct MonkFunction {
    MonkFuncPtr  fn;            // C function pointer
    MonkValue   *captures;      // captured variables (deep copied)
    int64_t      capture_count;
};

typedef MonkValue (*MonkFuncPtr)(MonkValue *args, int64_t argc);
```

---

## Value Constructors

All constructors **copy** their inputs. The caller retains ownership of the originals.

| Function | Signature | Notes |
|----------|-----------|-------|
| `monk_int` | `(int64_t n) -> MonkValue` | Inline, no allocation |
| `monk_float` | `(double f) -> MonkValue` | Inline, no allocation |
| `monk_string` | `(const char *s) -> MonkValue` | `strdup`s the string |
| `monk_bool` | `(bool b) -> MonkValue` | Inline |
| `monk_none` | `(void) -> MonkValue` | Inline |
| `monk_array` | `(MonkValue *elems, int64_t len) -> MonkValue` | Deep copies all elements |
| `monk_record` | `(MonkRecordField *fields, int64_t len) -> MonkValue` | Deep copies all field values |

---

## Memory Management

### `monk_deep_copy(MonkValue v) -> MonkValue`

Semantically copies a value. Primitives return as-is. Strings are `strdup`'d. Records and functions allocate new memory and deep-copy all contents. Arrays increment their backing-store refcount and detach on first mutation.

**Called by codegen on:** every boxed `let`/`const` declaration, every boxed reassignment, every boxed loop variable, every closure capture. Scalar-unboxed variables use plain C assignment and skip this function entirely.

**Fast path:** `monk_deep_copy` is a `static inline` function in `runtime.h` that short-circuits primitive kinds (int/float/bool/none) and only calls `monk_deep_copy_heap` (in `runtime.c`) for heap types. This is a significant perf win — a 20M-copy inner loop over int arrays used to make real function calls; now it's a kind-check and return.

### `monk_free(MonkValue v)`

Recursively frees a value. Primitives are no-ops. Frees strings, array data + elements, record fields + values, function captures.

**Called by codegen on:** scope exit for boxed variables, before reassignment (after computing the new value).

**Fast path:** Same inline/heap split as `monk_deep_copy`.

### `monk_panic(const char *msg)` — NEW

Exported so codegen can raise runtime errors on the unboxed fast path (e.g., `int / 0` or `int % 0` produces the same "division by zero" / "modulo by zero" error as the boxed path). Prints to stderr and calls `exit(1)`.

Prior to Phase 6 unboxing, `monk_panic` was `static` in the runtime; now it's declared in `runtime.h` and linked into the generated object file.

### Memory Pattern in Generated C

```c
// Assignment: let x = expr
MonkValue mk_x = monk_deep_copy(expr);

// Reassignment: x = new_expr
{ MonkValue _tmp = monk_deep_copy(new_expr);  // compute FIRST
  monk_free(mk_x);                             // THEN free old
  mk_x = _tmp; }

// Scope exit
monk_free(mk_x);
```

**Critical:** always compute new value before freeing old. Prevents use-after-free when the expression references the variable being assigned.

---

## Core Operations

### `monk_is_truthy(MonkValue v) -> bool`

Returns `false` for: `MONK_BOOL` with `false`, `MONK_NONE`, `MONK_INT` with `0`.
Returns `true` for everything else (including empty string and empty array).

### `monk_type_name(MonkValue v) -> const char*`

Returns a static string: `"int"`, `"float"`, `"string"`, `"boolean"`, `"none"`, `"array"`, `"record"`, `"function"`.

---

## Arithmetic

All functions take `MonkValue` and return `MonkValue`. Type promotion: `int op float = float`.

| Function | Monk operator | Notes |
|----------|--------------|-------|
| `monk_add(a, b)` | `+` | Also handles `string + string` → `monk_string_concat` |
| `monk_sub(a, b)` | `-` | |
| `monk_mul(a, b)` | `*` | |
| `monk_div(a, b)` | `/` | `int/int = int` (truncates). Throws on divide by zero. |
| `monk_mod(a, b)` | `%` | Throws on mod by zero. |
| `monk_neg(v)` | unary `-` | |

---

## Comparison

All return `MonkValue` of kind `MONK_BOOL`.

| Function | Monk operator |
|----------|--------------|
| `monk_equal(a, b)` | `==` |
| `monk_not_equal(a, b)` | `!=` |
| `monk_less(a, b)` | `<` |
| `monk_greater(a, b)` | `>` |
| `monk_less_equal(a, b)` | `<=` |
| `monk_greater_equal(a, b)` | `>=` |

Strings compare lexicographically. Cross-type comparison (e.g. `int == string`) returns `false` for `==`, `true` for `!=`.

---

## String Operations

| Function | Monk builtin | Notes |
|----------|-------------|-------|
| `monk_string_concat(a, b)` | `string + string` | Allocates new string |
| `monk_string_append_in_place(&target, suffix)` | generated for `s = s + rhs` / `s += rhs` | Reallocates target string directly; preserves user-visible value semantics |
| `monk_length(v)` | `length(v)` | Works on strings (Unicode codepoints), arrays, and records |
| `monk_substring(s, start, end)` | `substring(s, start, end)` | Indices clamped to bounds |
| `monk_index_of(s, search)` | `index_of(s, search)` | Returns -1 if not found |
| `monk_split(s, delim)` | `split(s, delim)` | Returns array of strings |
| `monk_trim(s)` | `trim(s)` | Strips leading/trailing whitespace |
| `monk_to_upper_case(s)` | `to_upper_case(s)` | |
| `monk_to_lower_case(s)` | `to_lower_case(s)` | |
| `monk_string_index(s, idx)` | `s[i]` | Returns single Unicode character as string. Out-of-bounds returns `none`. |

---

## Array Operations

All array functions return **new arrays** (value semantics). The original is never modified.

| Function | Monk builtin | Notes |
|----------|-------------|-------|
| `monk_array_get(arr, idx)` | `arr[i]` | Out-of-bounds returns `none` (generic array) |
| `monk_array_set(*arr, idx, val)` | `arr[i] = val` | Out-of-bounds is a **runtime error** |
| `monk_append(arr, elem)` | `append(arr, elem)` | New array with elem at end |
| `monk_prepend(arr, elem)` | `prepend(arr, elem)` | New array with elem at start |
| `monk_pop(arr)` | `pop(arr)` | New array without last element. `pop([])` returns `[]` |
| `monk_drop(arr, n)` | `drop(arr, n)` | Remove first n elements. Clamps. |
| `monk_take(arr, n)` | `take(arr, n)` | Keep first n elements. Clamps. |
| `monk_slice(arr, start, end)` | `slice(arr, start, end)` | Subarray. Indices clamped. |
| `monk_range(n)` | `range(n)` | `[0, 1, ..., n-1]`. `range(0)` returns `[]`. |
| `monk_fill(n, value)` | `fill(n, value)` | Array of n deep copies. `fill(0, x)` returns `[]`. |
| `monk_map(arr, fn)` | `map(arr, fn)` | New array: `fn(elem)` for each element |
| `monk_filter(arr, fn)` | `filter(arr, fn)` | New array: elements where `fn(elem)` is truthy |
| `monk_reduce(arr, fn, initial)` | `reduce(arr, fn, initial)` | Fold left. Returns `initial` on empty array. |

### Typed Array Converters

Convert a generic `MONK_ARRAY` (or same-kind typed array) into a typed
backing-store array. Generic arrays are converted into a new typed backing
store; same-kind typed arrays share backing storage copy-on-write.

| Function | Purpose |
|----------|---------|
| `monk_int_array_from(v)` | → `MONK_INT_ARRAY` with `int64_t*` backing |
| `monk_float_array_from(v)` | → `MONK_FLOAT_ARRAY` with `double*` backing |
| `monk_bool_array_from(v)` | → `MONK_BOOL_ARRAY` with `bool*` backing |
| `monk_int_array_ensure_unique(&v)` | Detach shared int[] backing before direct write |
| `monk_float_array_ensure_unique(&v)` | Detach shared float[] backing before direct write |
| `monk_bool_array_ensure_unique(&v)` | Detach shared bool[] backing before direct write |

Codegen calls these at assignment boundaries when the declared type is
`int[]`, `float[]`, or `bool[]`. Generated element access emits direct
pointer arithmetic (`arr.int_array_val->data[i]`) — no tagged-union
dispatch. Out-of-bounds panics (strict write model applies to typed arrays).

**Why it matters:** typed arrays hit ~2× C on `matmul` vs ~12× C for
the generic `MONK_ARRAY`. The entire array index hot path avoids the
union overhead.

---

## Record Operations

| Function | Monk syntax | Notes |
|----------|------------|-------|
| `monk_record_get(rec, key)` | `rec.field` | Returns `none` if field doesn't exist |
| `monk_record_set(*rec, key, val)` | `rec.field = val` | **Error** if field doesn't exist (fixed shape) |

---

## Math

All take and return `MonkValue`. Numeric types only — runtime error on non-numeric input.

| Function | Monk builtin |
|----------|-------------|
| `monk_abs(v)` | `abs(v)` |
| `monk_floor(v)` | `floor(v)` |
| `monk_ceil(v)` | `ceil(v)` |
| `monk_round(v)` | `round(v)` |
| `monk_sqrt(v)` | `sqrt(v)` — throws if v < 0 |
| `monk_pow(base, exp)` | `pow(base, exp)` |
| `monk_log(v)` | `log(v)` — throws if v <= 0 |
| `monk_log10(v)` | `log10(v)` — throws if v <= 0 |
| `monk_exp(v)` | `exp(v)` |
| `monk_min(a, b)` | `min(a, b)` |
| `monk_max(a, b)` | `max(a, b)` |
| `monk_sin(v)` | `sin(v)` |
| `monk_cos(v)` | `cos(v)` |
| `monk_tan(v)` | `tan(v)` |
| `monk_asin(v)` | `asin(v)` |
| `monk_acos(v)` | `acos(v)` |
| `monk_atan(v)` | `atan(v)` |

---

## Type Checking

All return `MonkValue` of kind `MONK_BOOL`, except `monk_typeof` which returns `MONK_STRING`.
Codegen may inline these for pure arguments with non-optional static types; effectful or unknown arguments still call the runtime helpers.

| Function | Monk builtin |
|----------|-------------|
| `monk_typeof(v)` | `typeof(v)` — returns `MONK_STRING` |
| `monk_is_number(v)` | `is_number(v)` — true for int or float |
| `monk_is_string(v)` | `is_string(v)` |
| `monk_is_boolean(v)` | `is_boolean(v)` |
| `monk_is_array(v)` | `is_array(v)` |
| `monk_is_record(v)` | `is_record(v)` |
| `monk_is_function(v)` | `is_function(v)` |
| `monk_is_none(v)` | `is_none(v)` |

---

## Output & Conversion

| Function | Monk builtin | Notes |
|----------|-------------|-------|
| `monk_show(v)` | `show(v)` | Prints to stdout + newline. Strings print without quotes. |
| `monk_to_string(v)` | `to_string(v)` | Returns string representation |
| `monk_to_int(v)` | `to_int(v)` | String to int (strict parse, throws on failure) |
| `monk_to_float(v)` | `to_float(v)` | String to float (throws on failure) |

---

## File System & Environment

| Function | Monk builtin | Notes |
|----------|-------------|-------|
| `monk_file_read(path)` | `file_read(path)` | Reads entire file. Throws on failure. |
| `monk_file_write(path, content)` | `file_write(path, content)` | Creates/overwrites. Throws on failure. |
| `monk_file_exists(path)` | `file_exists(path)` | Returns bool. |
| `monk_env_get(name)` | `env_get(name)` | Returns string (empty if unset). |
| `monk_exit(code)` | `exit(code)` | Calls C `exit()`. |
| `monk_args()` | `args()` | Returns string array of CLI args. |

---

## Error Handling

Monk's `guard`/`against`/`throw` compiles to `setjmp`/`longjmp`.

### Structs

```c
typedef struct {
    jmp_buf buf;
    void *prev;     // previous guard context (linked list stack)
} MonkGuardContext;
```

### Functions

| Function | Purpose |
|----------|---------|
| `monk_guard_begin_ctx(ctx)` | Push ctx onto the guard stack |
| `monk_guard_end(ctx)` | Pop ctx from the guard stack (normal exit) |
| `monk_throw(error)` | Store error value, `longjmp` to nearest guard |
| `monk_current_error()` | Retrieve the thrown error value |

### Macro

```c
#define monk_guard_begin(ctx) (monk_guard_begin_ctx(ctx), setjmp((ctx)->buf))
```

### Generated Pattern

```c
// Monk: guard result = risky_call() against error { result = fallback }

MonkGuardContext _guard_ctx;
MonkValue mk_result;
if (monk_guard_begin(&_guard_ctx) == 0) {
    mk_result = monk_deep_copy(risky_call());
    monk_guard_end(&_guard_ctx);
} else {
    MonkValue mk_error = monk_current_error();
    mk_result = monk_deep_copy(fallback);
    monk_free(mk_error);
}
```

Guard contexts form a **linked-list stack**. Nested guards work correctly. `monk_throw` jumps to the nearest active guard. If no guard is active, the program crashes with an error message.

---

## Compilation

The runtime is compiled alongside the generated C:

```bash
cc -std=c11 -O3 -flto -I<runtime_dir> program.c \
   value.c arith.c string.c container.c math.c builtins.c error.c \
   -lm -o program
```

The `-lm` flag links the math library (required for `sin`, `cos`, `sqrt`, etc.).

When using the `monk` CLI, all runtime files are embedded in the Go binary via `go:embed` and extracted to `~/.cache/monk/runtime/` on first use.
