#include <stdio.h>
#include <stdint.h>

#define N 1000000

int main(void) {
    int64_t best_start = 1;
    int64_t best_len = 1;

    for (int64_t n = 2; n <= N; n++) {
        int64_t current = n;
        int64_t chain_len = 1;
        while (current != 1) {
            if (current % 2 == 0) {
                current /= 2;
            } else {
                current = current * 3 + 1;
            }
            chain_len++;
        }
        if (chain_len > best_len) {
            best_len = chain_len;
            best_start = n;
        }
    }
    printf("%lld\n", best_start);
    return 0;
}
