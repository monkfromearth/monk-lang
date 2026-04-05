/* Benchmark: Leibniz series approximation of π. Same algorithm as .monk. */
#include <stdio.h>

int main(void) {
    long N = 50000000;
    double sum = 0.0;
    for (long k = 0; k < N; k++) {
        double term = 1.0 / (double)(2 * k + 1);
        if (k % 2 == 0) {
            sum += term;
        } else {
            sum -= term;
        }
    }
    double pi = 4.0 * sum;
    long scaled = (long)(pi * 10000000.0);
    printf("%ld\n", scaled);
    return 0;
}
