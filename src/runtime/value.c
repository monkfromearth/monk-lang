/*
 * Value model: constructors, deep copy, free, truthiness, string rep.
 *
 * Also hosts monk_panic and the panic-on-failure allocator wrappers because
 * they are value-layer primitives every other module depends on.
 */

#include "internal.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* --- Panic + allocators --- */

/* Exported so codegen-emitted unboxed paths can raise runtime errors
 * (e.g. int division by zero) with the same message format as the
 * classic runtime path. */
void monk_panic(const char *msg) {
    fprintf(stderr, "monk: runtime error: %s\n", msg);
    exit(1);
}

char *monk_strdup_internal(const char *s) {
    char *result = strdup(s ? s : "");
    if (!result) monk_panic("out of memory");
    return result;
}

void *monk_malloc_internal(size_t size) {
    void *ptr = malloc(size > 0 ? size : 1);
    if (!ptr) monk_panic("out of memory");
    return ptr;
}

void *monk_realloc_internal(void *old, size_t size) {
    /* realloc(p, 0) is implementation-defined — some libcs free and return
     * NULL, others return a 1-byte allocation. Mirror monk_malloc_internal
     * and demand at least 1 byte so the NULL-check below is unambiguous. */
    void *ptr = realloc(old, size > 0 ? size : 1);
    if (!ptr) monk_panic("out of memory");
    return ptr;
}

/* --- UTF-8 helpers --- */

/* Count UTF-8 codepoints (Unicode scalar values, not bytes) */
int64_t monk_utf8_strlen(const char *s) {
    int64_t count = 0;
    while (*s) {
        if ((*s & 0xC0) != 0x80) count++;
        s++;
    }
    return count;
}

/* Advance pointer by n UTF-8 codepoints, return byte offset */
int64_t monk_utf8_offset(const char *s, int64_t n) {
    int64_t offset = 0;
    while (n > 0 && s[offset]) {
        if ((s[offset] & 0xC0) != 0x80) n--;
        if (n > 0 || (s[offset] & 0xC0) == 0x80) offset++;
        else { offset++; break; }
    }
    /* Continue past continuation bytes */
    while (s[offset] && (s[offset] & 0xC0) == 0x80) offset++;
    return offset;
}

double monk_as_c_double(MonkValue v) {
    if (v.kind == MONK_INT) return (double)v.int_val;
    if (v.kind == MONK_FLOAT) return v.float_val;
    monk_panic("expected number");
    return 0;
}

/* --- Value constructors --- */

MonkValue monk_int(int64_t n) {
    return (MonkValue){.kind = MONK_INT, .int_val = n};
}

MonkValue monk_float(double f) {
    return (MonkValue){.kind = MONK_FLOAT, .float_val = f};
}

MonkValue monk_string(const char *s) {
    return (MonkValue){.kind = MONK_STRING, .str_val = monk_strdup_internal(s)};
}

MonkValue monk_bool(bool b) {
    return (MonkValue){.kind = MONK_BOOL, .bool_val = b};
}

MonkValue monk_none(void) {
    return (MonkValue){.kind = MONK_NONE};
}

MonkValue monk_array(MonkValue *elements, int64_t length) {
    MonkArray *arr = monk_malloc_internal(sizeof(MonkArray));
    arr->length = length;
    arr->refcount = 1;
    arr->data = monk_malloc_internal(sizeof(MonkValue) * (length > 0 ? length : 1));
    for (int64_t i = 0; i < length; i++) {
        arr->data[i] = monk_deep_copy(elements[i]);
    }
    return (MonkValue){.kind = MONK_ARRAY, .array_val = arr};
}

MonkValue monk_record(MonkRecordField *fields, int64_t length) {
    MonkRecord *rec = monk_malloc_internal(sizeof(MonkRecord));
    rec->length = length;
    rec->fields = monk_malloc_internal(sizeof(MonkRecordField) * (length > 0 ? length : 1));
    for (int64_t i = 0; i < length; i++) {
        rec->fields[i].key = monk_strdup_internal(fields[i].key);
        rec->fields[i].value = monk_deep_copy(fields[i].value);
    }
    return (MonkValue){.kind = MONK_RECORD, .record_val = rec};
}

MonkValue monk_make_function(MonkFuncPtr fn, MonkValue *captures, int64_t capture_count) {
    MonkFunction *f = monk_malloc_internal(sizeof(MonkFunction));
    f->fn = fn;
    f->capture_count = capture_count;
    if (capture_count > 0 && captures) {
        f->captures = monk_malloc_internal(sizeof(MonkValue) * capture_count);
        for (int64_t i = 0; i < capture_count; i++) {
            f->captures[i] = monk_deep_copy(captures[i]);
        }
    } else {
        f->captures = NULL;
    }
    return (MonkValue){.kind = MONK_FUNCTION, .func_val = f};
}

MonkValue monk_call(MonkValue fn, MonkValue *args, int64_t argc) {
    if (fn.kind != MONK_FUNCTION || !fn.func_val || !fn.func_val->fn) {
        monk_panic("cannot call non-function value");
    }
    return fn.func_val->fn(fn.func_val, args, argc);
}

/* --- Typed array converters --- */

static MonkValue monk_array_share(MonkValue v) {
    /* COW copy: share now, detach on first write.
     * Pass: `let b = a` increments refcount and stays O(1).
     * Fail: mutating b without ensure_unique would also mutate a. */
    if (v.array_val) v.array_val->refcount++;
    return v;
}

static MonkValue monk_int_array_share(MonkValue v) {
    if (v.int_array_val) v.int_array_val->refcount++;
    return v;
}

static MonkValue monk_float_array_share(MonkValue v) {
    if (v.float_array_val) v.float_array_val->refcount++;
    return v;
}

static MonkValue monk_bool_array_share(MonkValue v) {
    if (v.bool_array_val) v.bool_array_val->refcount++;
    return v;
}

/* monk_int_array_from: produce a MONK_INT_ARRAY from either:
 *   - MONK_ARRAY:     extract int_val from each element, free input.
 *   - MONK_INT_ARRAY: share the int64_t* backing store copy-on-write.
 * Codegen emits this at every int[] variable declaration. */
MonkValue monk_int_array_from(MonkValue v) {
    if (v.kind == MONK_INT_ARRAY) {
        return monk_int_array_share(v);
    }
    if (v.kind == MONK_ARRAY) {
        int64_t len = v.array_val->length;
        int64_t *data = monk_malloc_internal(sizeof(int64_t) * (len > 0 ? len : 1));
        for (int64_t i = 0; i < len; i++) data[i] = v.array_val->data[i].int_val;
        MonkIntArray *arr = monk_malloc_internal(sizeof(MonkIntArray));
        arr->data = data; arr->length = len; arr->refcount = 1;
        monk_free_heap(v);
        return (MonkValue){.kind = MONK_INT_ARRAY, .int_array_val = arr};
    }
    monk_panic("int_array_from: expected int[] or array");
    return monk_none();
}

MonkValue monk_float_array_from(MonkValue v) {
    if (v.kind == MONK_FLOAT_ARRAY) {
        return monk_float_array_share(v);
    }
    if (v.kind == MONK_ARRAY) {
        int64_t len = v.array_val->length;
        double *data = monk_malloc_internal(sizeof(double) * (len > 0 ? len : 1));
        for (int64_t i = 0; i < len; i++) {
            MonkValue e = v.array_val->data[i];
            data[i] = (e.kind == MONK_INT) ? (double)e.int_val : e.float_val;
        }
        MonkFloatArray *arr = monk_malloc_internal(sizeof(MonkFloatArray));
        arr->data = data; arr->length = len; arr->refcount = 1;
        monk_free_heap(v);
        return (MonkValue){.kind = MONK_FLOAT_ARRAY, .float_array_val = arr};
    }
    monk_panic("float_array_from: expected float[] or array");
    return monk_none();
}

MonkValue monk_bool_array_from(MonkValue v) {
    if (v.kind == MONK_BOOL_ARRAY) {
        return monk_bool_array_share(v);
    }
    if (v.kind == MONK_ARRAY) {
        int64_t len = v.array_val->length;
        bool *data = monk_malloc_internal(sizeof(bool) * (len > 0 ? len : 1));
        for (int64_t i = 0; i < len; i++) data[i] = v.array_val->data[i].bool_val;
        MonkBoolArray *arr = monk_malloc_internal(sizeof(MonkBoolArray));
        arr->data = data; arr->length = len; arr->refcount = 1;
        monk_free_heap(v);
        return (MonkValue){.kind = MONK_BOOL_ARRAY, .bool_array_val = arr};
    }
    monk_panic("bool_array_from: expected bool[] or array");
    return monk_none();
}

/* --- Deep copy (value semantics) --- */

MonkValue monk_deep_copy_heap(MonkValue v) {
    switch (v.kind) {
    case MONK_STRING:
        return monk_string(v.str_val);
    case MONK_ARRAY: {
        return monk_array_share(v);
    }
    case MONK_INT_ARRAY: {
        return monk_int_array_share(v);
    }
    case MONK_FLOAT_ARRAY: {
        return monk_float_array_share(v);
    }
    case MONK_BOOL_ARRAY: {
        return monk_bool_array_share(v);
    }
    case MONK_RECORD: {
        MonkRecord *src = v.record_val;
        return monk_record(src->fields, src->length);
    }
    case MONK_FUNCTION: {
        if (!v.func_val) return v;
        MonkFunction *src = v.func_val;
        MonkFunction *fn = monk_malloc_internal(sizeof(MonkFunction));
        fn->fn = src->fn;
        fn->capture_count = src->capture_count;
        fn->captures = monk_malloc_internal(sizeof(MonkValue) * (src->capture_count > 0 ? src->capture_count : 1));
        for (int64_t i = 0; i < src->capture_count; i++) {
            fn->captures[i] = monk_deep_copy(src->captures[i]);
        }
        return (MonkValue){.kind = MONK_FUNCTION, .func_val = fn};
    }
    default:
        return v; /* primitives are value types already */
    }
}

void monk_free_heap(MonkValue v) {
    switch (v.kind) {
    case MONK_STRING:
        free(v.str_val);
        break;
    case MONK_ARRAY:
        v.array_val->refcount--;
        if (v.array_val->refcount > 0) break;
        for (int64_t i = 0; i < v.array_val->length; i++)
            monk_free(v.array_val->data[i]);
        free(v.array_val->data);
        free(v.array_val);
        break;
    case MONK_INT_ARRAY:
        v.int_array_val->refcount--;
        if (v.int_array_val->refcount > 0) break;
        free(v.int_array_val->data);
        free(v.int_array_val);
        break;
    case MONK_FLOAT_ARRAY:
        v.float_array_val->refcount--;
        if (v.float_array_val->refcount > 0) break;
        free(v.float_array_val->data);
        free(v.float_array_val);
        break;
    case MONK_BOOL_ARRAY:
        v.bool_array_val->refcount--;
        if (v.bool_array_val->refcount > 0) break;
        free(v.bool_array_val->data);
        free(v.bool_array_val);
        break;
    case MONK_RECORD:
        for (int64_t i = 0; i < v.record_val->length; i++) {
            free((char *)v.record_val->fields[i].key);
            monk_free(v.record_val->fields[i].value);
        }
        free(v.record_val->fields);
        free(v.record_val);
        break;
    case MONK_FUNCTION:
        if (v.func_val) {
            for (int64_t i = 0; i < v.func_val->capture_count; i++)
                monk_free(v.func_val->captures[i]);
            free(v.func_val->captures);
            free(v.func_val);
        }
        break;
    default:
        break;
    }
}

/* --- Truthiness --- */

bool monk_is_truthy(MonkValue v) {
    /* Design decision: only false, none, and 0 are falsy.
       "" and [] are truthy. See spec "Design Philosophy." */
    switch (v.kind) {
    case MONK_BOOL:  return v.bool_val;
    case MONK_NONE:  return false;
    case MONK_INT:   return v.int_val != 0;
    default:         return true;
    }
}

const char *monk_type_name(MonkValue v) {
    switch (v.kind) {
    case MONK_INT:        return "int";
    case MONK_FLOAT:      return "float";
    case MONK_STRING:     return "string";
    case MONK_BOOL:       return "boolean";
    case MONK_NONE:       return "none";
    case MONK_ARRAY:      return "array";
    case MONK_INT_ARRAY:  return "array";
    case MONK_FLOAT_ARRAY:return "array";
    case MONK_BOOL_ARRAY: return "array";
    case MONK_RECORD:     return "record";
    case MONK_FUNCTION:   return "function";
    default:              return "unknown";
    }
}

/* --- String representation (malloc'd, caller frees) --- */

char *monk_value_to_cstr(MonkValue v) {
    char buf[64];
    switch (v.kind) {
    case MONK_INT:
        snprintf(buf, sizeof(buf), "%lld", (long long)v.int_val);
        return strdup(buf);
    case MONK_FLOAT:
        snprintf(buf, sizeof(buf), "%g", v.float_val);
        return strdup(buf);
    case MONK_BOOL:
        return strdup(v.bool_val ? "true" : "false");
    case MONK_NONE:
        return strdup("none");
    case MONK_STRING:
        return monk_strdup_internal(v.str_val);
    case MONK_FUNCTION:
        return strdup("<function>");
    case MONK_ARRAY: {
        /* Build "[elem, elem, ...]" with strings quoted */
        size_t cap = 64;
        char *result = monk_malloc_internal(cap);
        strcpy(result, "[");
        for (int64_t i = 0; i < v.array_val->length; i++) {
            if (i > 0) { strcat(result, ", "); }
            char *elem;
            if (v.array_val->data[i].kind == MONK_STRING) {
                elem = monk_malloc_internal(strlen(v.array_val->data[i].str_val) + 3);
                sprintf(elem, "\"%s\"", v.array_val->data[i].str_val);
            } else {
                elem = monk_value_to_cstr(v.array_val->data[i]);
            }
            while (strlen(result) + strlen(elem) + 4 > cap) {
                cap *= 2;
                result = monk_realloc_internal(result, cap);
            }
            strcat(result, elem);
            free(elem);
        }
        strcat(result, "]");
        return result;
    }
    case MONK_INT_ARRAY: {
        size_t cap = 64;
        char *result = monk_malloc_internal(cap);
        strcpy(result, "[");
        for (int64_t i = 0; i < v.int_array_val->length; i++) {
            char buf[32];
            if (i > 0) strcat(result, ", ");
            snprintf(buf, sizeof(buf), "%lld", (long long)v.int_array_val->data[i]);
            while (strlen(result) + strlen(buf) + 4 > cap) {
                cap *= 2; result = monk_realloc_internal(result, cap);
            }
            strcat(result, buf);
        }
        strcat(result, "]");
        return result;
    }
    case MONK_FLOAT_ARRAY: {
        size_t cap = 64;
        char *result = monk_malloc_internal(cap);
        strcpy(result, "[");
        for (int64_t i = 0; i < v.float_array_val->length; i++) {
            char buf[32];
            if (i > 0) strcat(result, ", ");
            snprintf(buf, sizeof(buf), "%g", v.float_array_val->data[i]);
            while (strlen(result) + strlen(buf) + 4 > cap) {
                cap *= 2; result = monk_realloc_internal(result, cap);
            }
            strcat(result, buf);
        }
        strcat(result, "]");
        return result;
    }
    case MONK_BOOL_ARRAY: {
        size_t cap = 64;
        char *result = monk_malloc_internal(cap);
        strcpy(result, "[");
        for (int64_t i = 0; i < v.bool_array_val->length; i++) {
            if (i > 0) strcat(result, ", ");
            const char *s = v.bool_array_val->data[i] ? "true" : "false";
            while (strlen(result) + strlen(s) + 4 > cap) {
                cap *= 2; result = monk_realloc_internal(result, cap);
            }
            strcat(result, s);
        }
        strcat(result, "]");
        return result;
    }
    case MONK_RECORD: {
        size_t cap = 64;
        char *result = monk_malloc_internal(cap);
        strcpy(result, "{");
        for (int64_t i = 0; i < v.record_val->length; i++) {
            if (i > 0) { strcat(result, ", "); }
            char *val;
            if (v.record_val->fields[i].value.kind == MONK_STRING) {
                val = monk_malloc_internal(strlen(v.record_val->fields[i].value.str_val) + 3);
                sprintf(val, "\"%s\"", v.record_val->fields[i].value.str_val);
            } else {
                val = monk_value_to_cstr(v.record_val->fields[i].value);
            }
            while (strlen(result) + strlen(v.record_val->fields[i].key) + strlen(val) + 6 > cap) {
                cap *= 2;
                result = monk_realloc_internal(result, cap);
            }
            strcat(result, v.record_val->fields[i].key);
            strcat(result, ": ");
            strcat(result, val);
            free(val);
        }
        strcat(result, "}");
        return result;
    }
    }
    return strdup("<unknown>");
}

/* --- Output & conversion --- */

void monk_show(MonkValue v) {
    char *s = monk_value_to_cstr(v);
    printf("%s\n", s);
    free(s);
}

MonkValue monk_to_string(MonkValue v) {
    char *s = monk_value_to_cstr(v);
    MonkValue result = {.kind = MONK_STRING, .str_val = s};
    return result;
}

MonkValue monk_to_int(MonkValue v) {
    /* Accept int (identity), float (truncate), string (strict parse).
     * String parsing stays strict: "3.14" is rejected — use to_int(to_float(s))
     * or floor(to_float(s)) if that's what you want. */
    switch (v.kind) {
    case MONK_INT:
        return v;
    case MONK_FLOAT:
        /* Truncate toward zero, matching C's (int64_t) cast. */
        return monk_int((int64_t)v.float_val);
    case MONK_STRING: {
        char *end;
        long long n = strtoll(v.str_val, &end, 0);
        if (*end != '\0') {
            char msg[128];
            snprintf(msg, sizeof(msg), "to_int: cannot parse \"%s\" as integer", v.str_val);
            monk_panic(msg);
        }
        return monk_int((int64_t)n);
    }
    default:
        monk_panic("to_int: expected int, float, or string");
        return monk_none(); /* unreachable — monk_panic longjmps */
    }
}

MonkValue monk_to_float(MonkValue v) {
    /* Accept int (widen), float (identity), string (parse). */
    switch (v.kind) {
    case MONK_INT:
        return monk_float((double)v.int_val);
    case MONK_FLOAT:
        return v;
    case MONK_STRING: {
        char *end;
        double f = strtod(v.str_val, &end);
        if (*end != '\0') {
            char msg[128];
            snprintf(msg, sizeof(msg), "to_float: cannot parse \"%s\" as float", v.str_val);
            monk_panic(msg);
        }
        return monk_float(f);
    }
    default:
        monk_panic("to_float: expected int, float, or string");
        return monk_none(); /* unreachable */
    }
}
