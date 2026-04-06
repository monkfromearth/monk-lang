/* Higher-order array functions: map, filter, reduce. */

#include "runtime.h"
#include "internal.h"
#include <stdlib.h>

MonkValue monk_map(MonkValue arr, MonkValue fn) {
    if (arr.kind != MONK_ARRAY || !arr.array_val) {
        monk_panic("map: first argument must be an array");
    }
    if (fn.kind != MONK_FUNCTION || !fn.func_val) {
        monk_panic("map: second argument must be a function");
    }
    MonkArray *src = arr.array_val;
    MonkValue *results = monk_malloc_internal(sizeof(MonkValue) * (src->length > 0 ? src->length : 1));
    for (int64_t i = 0; i < src->length; i++) {
        MonkValue arg = src->data[i];
        results[i] = fn.func_val->fn(fn.func_val, &arg, 1);
    }
    MonkValue out = monk_array(results, src->length);
    /* monk_array deep-copies, so free our temporary results */
    for (int64_t i = 0; i < src->length; i++) {
        monk_free(results[i]);
    }
    free(results);
    return out;
}

MonkValue monk_filter(MonkValue arr, MonkValue fn) {
    if (arr.kind != MONK_ARRAY || !arr.array_val) {
        monk_panic("filter: first argument must be an array");
    }
    if (fn.kind != MONK_FUNCTION || !fn.func_val) {
        monk_panic("filter: second argument must be a function");
    }
    MonkArray *src = arr.array_val;
    /* Worst case: all elements pass */
    MonkValue *results = monk_malloc_internal(sizeof(MonkValue) * (src->length > 0 ? src->length : 1));
    int64_t count = 0;
    for (int64_t i = 0; i < src->length; i++) {
        MonkValue arg = src->data[i];
        MonkValue pred = fn.func_val->fn(fn.func_val, &arg, 1);
        if (monk_is_truthy(pred)) {
            results[count++] = src->data[i];
        }
        monk_free(pred);
    }
    MonkValue out = monk_array(results, count);
    free(results);
    return out;
}

MonkValue monk_reduce(MonkValue arr, MonkValue fn, MonkValue initial) {
    if (arr.kind != MONK_ARRAY || !arr.array_val) {
        monk_panic("reduce: first argument must be an array");
    }
    if (fn.kind != MONK_FUNCTION || !fn.func_val) {
        monk_panic("reduce: second argument must be a function");
    }
    MonkArray *src = arr.array_val;
    MonkValue acc = monk_deep_copy(initial);
    for (int64_t i = 0; i < src->length; i++) {
        MonkValue args[2] = {acc, src->data[i]};
        MonkValue new_acc = fn.func_val->fn(fn.func_val, args, 2);
        monk_free(acc);
        acc = new_acc;
    }
    return acc;
}
