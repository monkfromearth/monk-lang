/* C reference: manual map/filter/reduce loops (no function pointer overhead). */
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>

int main() {
    int64_t N = 10000;
    int64_t *data = malloc(N * sizeof(int64_t));
    for (int64_t i = 0; i < N; i++) data[i] = i;

    int64_t total = 0;
    for (int round = 0; round < 100; round++) {
        /* map: double each element */
        int64_t *doubled = malloc(N * sizeof(int64_t));
        for (int64_t i = 0; i < N; i++) doubled[i] = data[i] * 2;

        /* filter: keep elements divisible by 4 */
        int64_t *evens = malloc(N * sizeof(int64_t));
        int64_t evens_len = 0;
        for (int64_t i = 0; i < N; i++) {
            if (doubled[i] % 4 == 0) evens[evens_len++] = doubled[i];
        }

        /* reduce: sum */
        int64_t sum = 0;
        for (int64_t i = 0; i < evens_len; i++) sum += evens[i];
        total += sum;

        free(doubled);
        free(evens);
    }
    free(data);
    printf("%lld\n", total);
    return 0;
}
