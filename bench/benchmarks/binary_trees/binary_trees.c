#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

#define MAX_DEPTH 14

static int64_t tree_size(int depth) {
    int64_t size = 1;
    for (int i = 0; i <= depth; i++) size *= 2;
    return size - 1;
}

static int64_t build_and_check(int depth) {
    int64_t num_nodes = tree_size(depth);
    int64_t *pool = malloc((size_t)(num_nodes * 3) * sizeof(int64_t));
    int64_t half = num_nodes / 2;

    /* Leaves */
    for (int64_t i = num_nodes - 1; i >= half; i--) {
        int64_t base = i * 3;
        pool[base] = 1;
        pool[base + 1] = -1;
        pool[base + 2] = -1;
    }

    /* Internal nodes */
    for (int64_t i = half - 1; i >= 0; i--) {
        int64_t base = i * 3;
        pool[base] = 0;
        pool[base + 1] = 2 * i + 1;
        pool[base + 2] = 2 * i + 2;
    }

    /* Walk with explicit stack */
    int64_t *stack = malloc((size_t)(num_nodes) * sizeof(int64_t));
    int64_t sp = 0;
    stack[sp++] = 0;
    int64_t check = 0;

    while (sp > 0) {
        int64_t node_idx = stack[--sp];
        int64_t base = node_idx * 3;
        check += pool[base];
        if (pool[base + 1] >= 0) {
            stack[sp++] = pool[base + 1];
            stack[sp++] = pool[base + 2];
        }
    }

    free(stack);
    free(pool);
    return check;
}

int main(void) {
    int64_t total = build_and_check(MAX_DEPTH + 1);
    int64_t long_lived = build_and_check(MAX_DEPTH);

    for (int depth = 4; depth <= MAX_DEPTH; depth += 2) {
        int64_t iterations = 1;
        for (int d = 0; d < MAX_DEPTH - depth + 4; d++) {
            iterations *= 2;
        }
        for (int64_t i = 0; i < iterations; i++) {
            total += build_and_check(depth);
        }
    }
    total += long_lived;
    printf("%lld\n", total);
    return 0;
}
