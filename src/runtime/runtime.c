/*
 * Monk Lang Runtime Library — Implementation
 * See runtime.h for documentation and design decisions.
 */

#include "runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <ctype.h>

/* --- Error handling globals --- */

static MonkGuardContext *guard_stack = NULL;
static MonkValue current_error;
static bool has_error = false;

/* --- Internal helpers --- */

static void monk_panic(const char *msg) {
    fprintf(stderr, "monk: runtime error: %s\n", msg);
    exit(1);
}

static char *monk_strdup(const char *s) {
    char *result = strdup(s ? s : "");
    if (!result) monk_panic("out of memory");
    return result;
}

static void *monk_malloc(size_t size) {
    void *ptr = malloc(size > 0 ? size : 1);
    if (!ptr) monk_panic("out of memory");
    return ptr;
}

static void *monk_realloc(void *old, size_t size) {
    void *ptr = realloc(old, size);
    if (!ptr) monk_panic("out of memory");
    return ptr;
}

/* Count UTF-8 codepoints (Unicode scalar values, not bytes) */
static int64_t utf8_strlen(const char *s) {
    int64_t count = 0;
    while (*s) {
        if ((*s & 0xC0) != 0x80) count++;
        s++;
    }
    return count;
}

/* Advance pointer by n UTF-8 codepoints, return byte offset */
static int64_t utf8_offset(const char *s, int64_t n) {
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

static double monk_to_go_float(MonkValue v) {
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
    return (MonkValue){.kind = MONK_STRING, .str_val = monk_strdup(s)};
}

MonkValue monk_bool(bool b) {
    return (MonkValue){.kind = MONK_BOOL, .bool_val = b};
}

MonkValue monk_none(void) {
    return (MonkValue){.kind = MONK_NONE};
}

MonkValue monk_array(MonkValue *elements, int64_t length) {
    MonkArray *arr = monk_malloc(sizeof(MonkArray));
    arr->length = length;
    arr->data = monk_malloc(sizeof(MonkValue) * (length > 0 ? length : 1));
    for (int64_t i = 0; i < length; i++) {
        arr->data[i] = monk_deep_copy(elements[i]);
    }
    return (MonkValue){.kind = MONK_ARRAY, .array_val = arr};
}

MonkValue monk_record(MonkRecordField *fields, int64_t length) {
    MonkRecord *rec = monk_malloc(sizeof(MonkRecord));
    rec->length = length;
    rec->fields = monk_malloc(sizeof(MonkRecordField) * (length > 0 ? length : 1));
    for (int64_t i = 0; i < length; i++) {
        rec->fields[i].key = monk_strdup(fields[i].key);
        rec->fields[i].value = monk_deep_copy(fields[i].value);
    }
    return (MonkValue){.kind = MONK_RECORD, .record_val = rec};
}

/* --- Deep copy (value semantics) --- */

MonkValue monk_deep_copy_heap(MonkValue v) {
    switch (v.kind) {
    case MONK_STRING:
        return monk_string(v.str_val);
    case MONK_ARRAY: {
        MonkArray *src = v.array_val;
        return monk_array(src->data, src->length);
    }
    case MONK_RECORD: {
        MonkRecord *src = v.record_val;
        return monk_record(src->fields, src->length);
    }
    case MONK_FUNCTION: {
        if (!v.func_val) return v;
        MonkFunction *src = v.func_val;
        MonkFunction *fn = monk_malloc(sizeof(MonkFunction));
        fn->fn = src->fn;
        fn->capture_count = src->capture_count;
        fn->captures = monk_malloc(sizeof(MonkValue) * (src->capture_count > 0 ? src->capture_count : 1));
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
        for (int64_t i = 0; i < v.array_val->length; i++)
            monk_free(v.array_val->data[i]);
        free(v.array_val->data);
        free(v.array_val);
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
    case MONK_INT:      return "int";
    case MONK_FLOAT:    return "float";
    case MONK_STRING:   return "string";
    case MONK_BOOL:     return "boolean";
    case MONK_NONE:     return "none";
    case MONK_ARRAY:    return "array";
    case MONK_RECORD:   return "record";
    case MONK_FUNCTION: return "function";
    default:            return "unknown";
    }
}

/* --- to_string (internal, returns malloc'd string) --- */

static char *value_to_str(MonkValue v) {
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
        return monk_strdup(v.str_val);
    case MONK_FUNCTION:
        return strdup("<function>");
    case MONK_ARRAY: {
        /* Build "[elem, elem, ...]" with strings quoted */
        size_t cap = 64;
        char *result = monk_malloc(cap);
        strcpy(result, "[");
        for (int64_t i = 0; i < v.array_val->length; i++) {
            if (i > 0) { strcat(result, ", "); }
            char *elem;
            if (v.array_val->data[i].kind == MONK_STRING) {
                elem = monk_malloc(strlen(v.array_val->data[i].str_val) + 3);
                sprintf(elem, "\"%s\"", v.array_val->data[i].str_val);
            } else {
                elem = value_to_str(v.array_val->data[i]);
            }
            while (strlen(result) + strlen(elem) + 4 > cap) {
                cap *= 2;
                result = monk_realloc(result, cap);
            }
            strcat(result, elem);
            free(elem);
        }
        strcat(result, "]");
        return result;
    }
    case MONK_RECORD: {
        size_t cap = 64;
        char *result = monk_malloc(cap);
        strcpy(result, "{");
        for (int64_t i = 0; i < v.record_val->length; i++) {
            if (i > 0) { strcat(result, ", "); }
            char *val;
            if (v.record_val->fields[i].value.kind == MONK_STRING) {
                val = monk_malloc(strlen(v.record_val->fields[i].value.str_val) + 3);
                sprintf(val, "\"%s\"", v.record_val->fields[i].value.str_val);
            } else {
                val = value_to_str(v.record_val->fields[i].value);
            }
            while (strlen(result) + strlen(v.record_val->fields[i].key) + strlen(val) + 6 > cap) {
                cap *= 2;
                result = monk_realloc(result, cap);
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
    char *s = value_to_str(v);
    printf("%s\n", s);
    free(s);
}

MonkValue monk_to_string(MonkValue v) {
    char *s = value_to_str(v);
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

/* --- Comparison --- */

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
        return monk_float(monk_to_go_float(a) + monk_to_go_float(b));
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return monk_int(a.int_val + b.int_val);
    monk_panic("cannot add these types");
    return monk_none();
}

MonkValue monk_sub(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT)
        return monk_float(monk_to_go_float(a) - monk_to_go_float(b));
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return monk_int(a.int_val - b.int_val);
    monk_panic("cannot subtract these types");
    return monk_none();
}

MonkValue monk_mul(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT)
        return monk_float(monk_to_go_float(a) * monk_to_go_float(b));
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return monk_int(a.int_val * b.int_val);
    monk_panic("cannot multiply these types");
    return monk_none();
}

MonkValue monk_div(MonkValue a, MonkValue b) {
    if (a.kind == MONK_FLOAT || b.kind == MONK_FLOAT) {
        double denom = monk_to_go_float(b);
        if (denom == 0) monk_panic("division by zero");
        return monk_float(monk_to_go_float(a) / denom);
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
    if (v.kind == MONK_INT) return monk_int(-v.int_val);
    if (v.kind == MONK_FLOAT) return monk_float(-v.float_val);
    monk_panic("cannot negate this type");
    return monk_none();
}

MonkValue monk_string_concat(MonkValue a, MonkValue b) {
    /* Design decision: string + string only. No auto-coercion. */
    if (a.kind != MONK_STRING || b.kind != MONK_STRING)
        monk_panic("string concatenation requires two strings (use to_string())");
    size_t len = strlen(a.str_val) + strlen(b.str_val) + 1;
    char *result = monk_malloc(len);
    strcpy(result, a.str_val);
    strcat(result, b.str_val);
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

/* --- String functions --- */

MonkValue monk_length(MonkValue v) {
    switch (v.kind) {
    case MONK_STRING: return monk_int(utf8_strlen(v.str_val));
    case MONK_ARRAY:  return monk_int(v.array_val->length);
    case MONK_RECORD: return monk_int(v.record_val->length);
    default: monk_panic("length: expected string, array, or record"); return monk_none();
    }
}

MonkValue monk_substring(MonkValue s, MonkValue start_v, MonkValue end_v) {
    if (s.kind != MONK_STRING) monk_panic("substring: expected string");
    int64_t slen = utf8_strlen(s.str_val);
    int64_t start = start_v.int_val;
    int64_t end = end_v.int_val;
    /* Design decision: indices clamp (graceful on reads) */
    if (start < 0) start = 0;
    if (end > slen) end = slen;
    if (start >= end) return monk_string("");
    int64_t byte_start = utf8_offset(s.str_val, start);
    int64_t byte_end = utf8_offset(s.str_val, end);
    int64_t byte_len = byte_end - byte_start;
    char *result = monk_malloc(byte_len + 1);
    memcpy(result, s.str_val + byte_start, byte_len);
    result[byte_len] = '\0';
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

MonkValue monk_index_of(MonkValue s, MonkValue search) {
    if (s.kind != MONK_STRING || search.kind != MONK_STRING)
        monk_panic("index_of: expected two strings");
    const char *found = strstr(s.str_val, search.str_val);
    if (!found) return monk_int(-1);
    /* Return character index, not byte index */
    int64_t byte_pos = found - s.str_val;
    int64_t char_pos = 0;
    for (int64_t i = 0; i < byte_pos; i++) {
        if ((s.str_val[i] & 0xC0) != 0x80) char_pos++;
    }
    return monk_int(char_pos);
}

MonkValue monk_split(MonkValue s, MonkValue delim) {
    if (s.kind != MONK_STRING || delim.kind != MONK_STRING)
        monk_panic("split: expected two strings");
    /* Count parts first */
    int64_t count = 1;
    const char *p = s.str_val;
    size_t dlen = strlen(delim.str_val);
    if (dlen == 0) {
        /* Split by character */
        int64_t slen = utf8_strlen(s.str_val);
        MonkValue *parts = monk_malloc(sizeof(MonkValue) * slen);
        p = s.str_val;
        for (int64_t i = 0; i < slen; i++) {
            int64_t clen = 1;
            while ((p[clen] & 0xC0) == 0x80) clen++;
            char *ch = monk_malloc(clen + 1);
            memcpy(ch, p, clen);
            ch[clen] = '\0';
            parts[i] = (MonkValue){.kind = MONK_STRING, .str_val = ch};
            p += clen;
        }
        MonkValue result = monk_array(parts, slen);
        for (int64_t i = 0; i < slen; i++) free(parts[i].str_val);
        free(parts);
        return result;
    }
    while ((p = strstr(p, delim.str_val)) != NULL) { count++; p += dlen; }
    MonkValue *parts = monk_malloc(sizeof(MonkValue) * count);
    p = s.str_val;
    for (int64_t i = 0; i < count; i++) {
        const char *next = (i < count - 1) ? strstr(p, delim.str_val) : p + strlen(p);
        size_t part_len = next - p;
        char *part = monk_malloc(part_len + 1);
        memcpy(part, p, part_len);
        part[part_len] = '\0';
        parts[i] = (MonkValue){.kind = MONK_STRING, .str_val = part};
        p = next + dlen;
    }
    MonkValue result = monk_array(parts, count);
    for (int64_t i = 0; i < count; i++) free(parts[i].str_val);
    free(parts);
    return result;
}

MonkValue monk_trim(MonkValue s) {
    if (s.kind != MONK_STRING) monk_panic("trim: expected string");
    const char *start = s.str_val;
    while (*start && isspace((unsigned char)*start)) start++;
    const char *end = s.str_val + strlen(s.str_val);
    while (end > start && isspace((unsigned char)*(end - 1))) end--;
    size_t len = end - start;
    char *result = monk_malloc(len + 1);
    memcpy(result, start, len);
    result[len] = '\0';
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

MonkValue monk_to_upper_case(MonkValue s) {
    if (s.kind != MONK_STRING) monk_panic("to_upper_case: expected string");
    char *result = monk_strdup(s.str_val);
    for (char *p = result; *p; p++) *p = toupper((unsigned char)*p);
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

MonkValue monk_to_lower_case(MonkValue s) {
    if (s.kind != MONK_STRING) monk_panic("to_lower_case: expected string");
    char *result = monk_strdup(s.str_val);
    for (char *p = result; *p; p++) *p = tolower((unsigned char)*p);
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

/* --- Array functions --- */

MonkValue monk_array_get(MonkValue arr, MonkValue index) {
    /* Design decision: out-of-bounds read returns none (graceful on reads) */
    if (arr.kind != MONK_ARRAY) monk_panic("cannot index non-array");
    int64_t idx = index.int_val;
    if (idx < 0 || idx >= arr.array_val->length) return monk_none();
    return monk_deep_copy(arr.array_val->data[idx]);
}

void monk_array_set(MonkValue *arr, MonkValue index, MonkValue value) {
    /* Design decision: out-of-bounds write is an error (strict on operations) */
    if (arr->kind != MONK_ARRAY) monk_panic("cannot index-assign non-array");
    int64_t idx = index.int_val;
    if (idx < 0 || idx >= arr->array_val->length) monk_panic("array index out of bounds");
    monk_free(arr->array_val->data[idx]);
    arr->array_val->data[idx] = monk_deep_copy(value);
}

MonkValue monk_string_index(MonkValue s, MonkValue index) {
    if (s.kind != MONK_STRING) monk_panic("cannot index non-string");
    int64_t idx = index.int_val;
    int64_t slen = utf8_strlen(s.str_val);
    if (idx < 0 || idx >= slen) return monk_none();
    int64_t byte_start = utf8_offset(s.str_val, idx);
    int64_t byte_end = utf8_offset(s.str_val, idx + 1);
    int64_t clen = byte_end - byte_start;
    char *ch = monk_malloc(clen + 1);
    memcpy(ch, s.str_val + byte_start, clen);
    ch[clen] = '\0';
    return (MonkValue){.kind = MONK_STRING, .str_val = ch};
}

MonkValue monk_append(MonkValue arr, MonkValue elem) {
    if (arr.kind != MONK_ARRAY) monk_panic("append: expected array");
    int64_t new_len = arr.array_val->length + 1;
    MonkValue *new_data = monk_malloc(sizeof(MonkValue) * new_len);
    for (int64_t i = 0; i < arr.array_val->length; i++)
        new_data[i] = monk_deep_copy(arr.array_val->data[i]);
    new_data[new_len - 1] = monk_deep_copy(elem);
    MonkArray *new_arr = monk_malloc(sizeof(MonkArray));
    new_arr->data = new_data;
    new_arr->length = new_len;
    return (MonkValue){.kind = MONK_ARRAY, .array_val = new_arr};
}

MonkValue monk_prepend(MonkValue arr, MonkValue elem) {
    if (arr.kind != MONK_ARRAY) monk_panic("prepend: expected array");
    int64_t new_len = arr.array_val->length + 1;
    MonkValue *new_data = monk_malloc(sizeof(MonkValue) * new_len);
    new_data[0] = monk_deep_copy(elem);
    for (int64_t i = 0; i < arr.array_val->length; i++)
        new_data[i + 1] = monk_deep_copy(arr.array_val->data[i]);
    MonkArray *new_arr = monk_malloc(sizeof(MonkArray));
    new_arr->data = new_data;
    new_arr->length = new_len;
    return (MonkValue){.kind = MONK_ARRAY, .array_val = new_arr};
}

MonkValue monk_pop(MonkValue arr) {
    if (arr.kind != MONK_ARRAY) monk_panic("pop: expected array");
    /* Design decision: pop([]) returns [] (graceful) */
    if (arr.array_val->length == 0) return monk_array(NULL, 0);
    return monk_array(arr.array_val->data, arr.array_val->length - 1);
}

MonkValue monk_drop(MonkValue arr, MonkValue n_val) {
    if (arr.kind != MONK_ARRAY) monk_panic("drop: expected array");
    int64_t n = n_val.int_val;
    /* Design decision: clamps (graceful) */
    if (n >= arr.array_val->length) return monk_array(NULL, 0);
    if (n < 0) n = 0;
    return monk_array(arr.array_val->data + n, arr.array_val->length - n);
}

MonkValue monk_take(MonkValue arr, MonkValue n_val) {
    if (arr.kind != MONK_ARRAY) monk_panic("take: expected array");
    int64_t n = n_val.int_val;
    /* Design decision: clamps (graceful) */
    if (n >= arr.array_val->length) n = arr.array_val->length;
    if (n < 0) n = 0;
    return monk_array(arr.array_val->data, n);
}

MonkValue monk_slice(MonkValue arr, MonkValue start_v, MonkValue end_v) {
    if (arr.kind != MONK_ARRAY) monk_panic("slice: expected array");
    int64_t start = start_v.int_val;
    int64_t end = end_v.int_val;
    /* Design decision: indices clamp (graceful) */
    if (start < 0) start = 0;
    if (end > arr.array_val->length) end = arr.array_val->length;
    if (start >= end) return monk_array(NULL, 0);
    return monk_array(arr.array_val->data + start, end - start);
}

MonkValue monk_range(MonkValue n_val) {
    if (n_val.kind != MONK_INT) monk_panic("range: expected int");
    int64_t n = n_val.int_val;
    /* Design decision: range(0) and range(-5) return [] (graceful) */
    if (n <= 0) return monk_array(NULL, 0);
    MonkValue *data = monk_malloc(sizeof(MonkValue) * n);
    for (int64_t i = 0; i < n; i++) data[i] = monk_int(i);
    MonkArray *arr = monk_malloc(sizeof(MonkArray));
    arr->data = data;
    arr->length = n;
    return (MonkValue){.kind = MONK_ARRAY, .array_val = arr};
}

/* --- Record --- */

MonkValue monk_record_get(MonkValue rec, const char *key) {
    if (rec.kind != MONK_RECORD) monk_panic("cannot access property of non-record");
    for (int64_t i = 0; i < rec.record_val->length; i++) {
        if (strcmp(rec.record_val->fields[i].key, key) == 0)
            return monk_deep_copy(rec.record_val->fields[i].value);
    }
    /* Design decision: untyped record missing field = none (graceful on reads) */
    return monk_none();
}

void monk_record_set(MonkValue *rec, const char *key, MonkValue value) {
    if (rec->kind != MONK_RECORD) monk_panic("cannot set property on non-record");
    for (int64_t i = 0; i < rec->record_val->length; i++) {
        if (strcmp(rec->record_val->fields[i].key, key) == 0) {
            monk_free(rec->record_val->fields[i].value);
            rec->record_val->fields[i].value = monk_deep_copy(value);
            return;
        }
    }
    /* Design decision: records have fixed shape. Cannot add new fields. */
    char msg[128];
    snprintf(msg, sizeof(msg), "record has no field '%s'", key);
    monk_panic(msg);
}

/* --- Math --- */

MonkValue monk_abs(MonkValue v) {
    if (v.kind == MONK_INT) return monk_int(v.int_val < 0 ? -v.int_val : v.int_val);
    if (v.kind == MONK_FLOAT) return monk_float(fabs(v.float_val));
    monk_panic("abs: expected number"); return monk_none();
}

MonkValue monk_floor(MonkValue v) { return monk_int((int64_t)floor(monk_to_go_float(v))); }
MonkValue monk_ceil(MonkValue v)  { return monk_int((int64_t)ceil(monk_to_go_float(v))); }
MonkValue monk_round(MonkValue v) { return monk_int((int64_t)round(monk_to_go_float(v))); }

MonkValue monk_sqrt(MonkValue v) {
    double f = monk_to_go_float(v);
    if (f < 0) monk_panic("sqrt: cannot take square root of negative number");
    return monk_float(sqrt(f));
}

MonkValue monk_pow(MonkValue base, MonkValue exp) {
    return monk_float(pow(monk_to_go_float(base), monk_to_go_float(exp)));
}

MonkValue monk_log(MonkValue v) {
    double f = monk_to_go_float(v);
    if (f <= 0) monk_panic("log: argument must be positive");
    return monk_float(log(f));
}

MonkValue monk_log10(MonkValue v) {
    double f = monk_to_go_float(v);
    if (f <= 0) monk_panic("log10: argument must be positive");
    return monk_float(log10(f));
}

MonkValue monk_exp(MonkValue v)  { return monk_float(exp(monk_to_go_float(v))); }

MonkValue monk_min(MonkValue a, MonkValue b) {
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return a.int_val < b.int_val ? a : b;
    return monk_float(fmin(monk_to_go_float(a), monk_to_go_float(b)));
}

MonkValue monk_max(MonkValue a, MonkValue b) {
    if (a.kind == MONK_INT && b.kind == MONK_INT)
        return a.int_val > b.int_val ? a : b;
    return monk_float(fmax(monk_to_go_float(a), monk_to_go_float(b)));
}

MonkValue monk_sin(MonkValue v)  { return monk_float(sin(monk_to_go_float(v))); }
MonkValue monk_cos(MonkValue v)  { return monk_float(cos(monk_to_go_float(v))); }
MonkValue monk_tan(MonkValue v)  { return monk_float(tan(monk_to_go_float(v))); }
MonkValue monk_asin(MonkValue v) { return monk_float(asin(monk_to_go_float(v))); }
MonkValue monk_acos(MonkValue v) { return monk_float(acos(monk_to_go_float(v))); }
MonkValue monk_atan(MonkValue v) { return monk_float(atan(monk_to_go_float(v))); }

/* --- Type checking --- */

MonkValue monk_typeof(MonkValue v)      { return monk_string(monk_type_name(v)); }
MonkValue monk_is_number(MonkValue v)   { return monk_bool(v.kind == MONK_INT || v.kind == MONK_FLOAT); }
MonkValue monk_is_string(MonkValue v)   { return monk_bool(v.kind == MONK_STRING); }
MonkValue monk_is_boolean(MonkValue v)  { return monk_bool(v.kind == MONK_BOOL); }
MonkValue monk_is_array(MonkValue v)    { return monk_bool(v.kind == MONK_ARRAY); }
MonkValue monk_is_record(MonkValue v)   { return monk_bool(v.kind == MONK_RECORD); }
MonkValue monk_is_function(MonkValue v) { return monk_bool(v.kind == MONK_FUNCTION); }
MonkValue monk_is_none(MonkValue v)     { return monk_bool(v.kind == MONK_NONE); }

/* --- File system & environment --- */

MonkValue monk_file_read(MonkValue path) {
    if (path.kind != MONK_STRING) monk_panic("file_read: expected string path");
    FILE *f = fopen(path.str_val, "rb");
    if (!f) monk_panic("file_read: cannot open file");
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    if (size < 0) { fclose(f); monk_panic("file_read: cannot determine file size"); }
    fseek(f, 0, SEEK_SET);
    char *buf = monk_malloc(size + 1);
    size_t read = fread(buf, 1, size, f);
    buf[read] = '\0';
    fclose(f);
    return (MonkValue){.kind = MONK_STRING, .str_val = buf};
}

MonkValue monk_file_write(MonkValue path, MonkValue content) {
    if (path.kind != MONK_STRING) monk_panic("file_write: expected string path");
    if (content.kind != MONK_STRING) monk_panic("file_write: expected string content");
    FILE *f = fopen(path.str_val, "wb");
    if (!f) monk_panic("file_write: cannot open file");
    fwrite(content.str_val, 1, strlen(content.str_val), f);
    fclose(f);
    return monk_none();
}

MonkValue monk_file_exists(MonkValue path) {
    if (path.kind != MONK_STRING) monk_panic("file_exists: expected string path");
    FILE *f = fopen(path.str_val, "r");
    if (f) { fclose(f); return monk_bool(true); }
    return monk_bool(false);
}

MonkValue monk_env_get(MonkValue name) {
    if (name.kind != MONK_STRING) monk_panic("env_get: expected string");
    const char *val = getenv(name.str_val);
    if (!val) return monk_none();
    return monk_string(val);
}

void monk_exit(MonkValue code) {
    if (code.kind != MONK_INT) monk_panic("exit: expected int");
    exit((int)code.int_val);
}

MonkValue monk_args(void) {
    /* Will be populated by generated main() with argc/argv */
    return monk_array(NULL, 0);
}

/* --- Error handling --- */

void monk_guard_begin_ctx(MonkGuardContext *ctx) {
    ctx->prev = guard_stack;
    guard_stack = ctx;
    has_error = false;
}

void monk_guard_end(MonkGuardContext *ctx) {
    guard_stack = ctx->prev;
}

void monk_throw(MonkValue error) {
    if (!guard_stack) {
        /* Unhandled throw at top level — terminate program */
        char *s = value_to_str(error);
        fprintf(stderr, "monk: unhandled error: %s\n", s);
        free(s);
        exit(1);
    }
    current_error = monk_deep_copy(error);
    has_error = true;
    MonkGuardContext *ctx = guard_stack;
    guard_stack = ctx->prev;
    longjmp(ctx->buf, 1);
}

MonkValue monk_current_error(void) {
    return current_error;
}
