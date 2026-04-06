#include <stdio.h>
#include <stdint.h>

int main() {
    int64_t N = 10000;
    int64_t arr[10000];
    int64_t seed = 42;
    for (int64_t i = 0; i < N; i++) {
        seed = (seed * 1103515245 + 12345) % 2147483648;
        arr[i] = seed % 100000;
    }

    /* Iterative quicksort with explicit stack */
    int64_t stack[20000];
    int64_t sp = 0;
    stack[sp++] = 0;
    stack[sp++] = N - 1;

    while (sp > 0) {
        int64_t hi = stack[--sp];
        int64_t lo = stack[--sp];

        if (lo < hi) {
            int64_t pivot = arr[hi];
            int64_t i = lo;
            for (int64_t j = lo; j < hi; j++) {
                if (arr[j] <= pivot) {
                    int64_t tmp = arr[i]; arr[i] = arr[j]; arr[j] = tmp;
                    i++;
                }
            }
            int64_t tmp = arr[i]; arr[i] = arr[hi]; arr[hi] = tmp;

            if (i + 1 < hi) { stack[sp++] = i + 1; stack[sp++] = hi; }
            if (i - 1 > lo) { stack[sp++] = lo; stack[sp++] = i - 1; }
        }
    }

    printf("%lld\n", arr[0] + arr[N/2] + arr[N-1]);
    return 0;
}
