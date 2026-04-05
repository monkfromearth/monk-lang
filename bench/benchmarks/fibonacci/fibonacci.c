/* Benchmark: recursive fibonacci. Same algorithm as fib.monk. */
#include <stdio.h>

static long fib(long n) {
    if (n < 2) return n;
    return fib(n - 1) + fib(n - 2);
}

int main(void) {
    printf("%ld\n", fib(35));
    return 0;
}
