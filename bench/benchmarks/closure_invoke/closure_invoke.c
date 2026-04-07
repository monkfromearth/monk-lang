/* C reference: direct function call (no closure overhead). */
#include <stdio.h>
#include <stdint.h>

static int64_t adder(int64_t x, int64_t captured_i) {
    return x + captured_i;
}

int main() {
    int64_t N = 1000000;
    int64_t total = 0;
    for (int64_t i = 0; i < N; i++) {
        total += adder(10, i);
    }
    printf("%lld\n", total);
    return 0;
}
