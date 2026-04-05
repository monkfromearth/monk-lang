/* Benchmark: naive integer matrix multiplication, 400x400. Same algorithm as .monk. */
#include <stdio.h>
#include <stdlib.h>

#define N 400

int main(void) {
    long *A = malloc(N * N * sizeof(long));
    long *B = malloc(N * N * sizeof(long));
    long *C = malloc(N * N * sizeof(long));

    for (int i = 0; i < N; i++) {
        for (int j = 0; j < N; j++) {
            A[i * N + j] = i + j;
            B[i * N + j] = i - j;
            C[i * N + j] = 0;
        }
    }

    for (int i = 0; i < N; i++) {
        for (int k = 0; k < N; k++) {
            long aik = A[i * N + k];
            for (int j = 0; j < N; j++) {
                C[i * N + j] += aik * B[k * N + j];
            }
        }
    }

    long sum = C[0] + C[N - 1] + C[N * (N - 1)] + C[N * N - 1];
    printf("%ld\n", sum);
    free(A); free(B); free(C);
    return 0;
}
