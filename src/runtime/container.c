/*
 * Array + record operations.
 *
 * Arrays: indexed read/write, structural mutators (append, prepend, pop,
 * drop, take, slice), range builder.
 * Records: property read/write with fixed-shape discipline.
 */

#include "internal.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* --- Array --- */

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

MonkValue monk_append(MonkValue arr, MonkValue elem) {
    if (arr.kind != MONK_ARRAY) monk_panic("append: expected array");
    int64_t new_len = arr.array_val->length + 1;
    MonkValue *new_data = monk_malloc_internal(sizeof(MonkValue) * new_len);
    for (int64_t i = 0; i < arr.array_val->length; i++)
        new_data[i] = monk_deep_copy(arr.array_val->data[i]);
    new_data[new_len - 1] = monk_deep_copy(elem);
    MonkArray *new_arr = monk_malloc_internal(sizeof(MonkArray));
    new_arr->data = new_data;
    new_arr->length = new_len;
    return (MonkValue){.kind = MONK_ARRAY, .array_val = new_arr};
}

MonkValue monk_prepend(MonkValue arr, MonkValue elem) {
    if (arr.kind != MONK_ARRAY) monk_panic("prepend: expected array");
    int64_t new_len = arr.array_val->length + 1;
    MonkValue *new_data = monk_malloc_internal(sizeof(MonkValue) * new_len);
    new_data[0] = monk_deep_copy(elem);
    for (int64_t i = 0; i < arr.array_val->length; i++)
        new_data[i + 1] = monk_deep_copy(arr.array_val->data[i]);
    MonkArray *new_arr = monk_malloc_internal(sizeof(MonkArray));
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
    MonkValue *data = monk_malloc_internal(sizeof(MonkValue) * n);
    for (int64_t i = 0; i < n; i++) data[i] = monk_int(i);
    MonkArray *arr = monk_malloc_internal(sizeof(MonkArray));
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
