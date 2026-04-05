/* Benchmark: count primes up to N by trial division. Same algorithm as .monk. */
#include <stdio.h>
#include <stdbool.h>

int main(void) {
    long N = 200000;
    long count = 0;
    for (long n = 2; n <= N; n++) {
        bool is_prime = true;
        for (long d = 2; d * d <= n; d++) {
            if (n % d == 0) {
                is_prime = false;
                break;
            }
        }
        if (is_prime) count++;
    }
    printf("%ld\n", count);
    return 0;
}
