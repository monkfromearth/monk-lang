//go:build ignore

package main

import "fmt"

const N = 400

func main() {
	A := make([]int64, N*N)
	B := make([]int64, N*N)
	C := make([]int64, N*N)

	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			A[i*N+j] = int64(i + j)
			B[i*N+j] = int64(i - j)
			C[i*N+j] = 0
		}
	}

	for i := 0; i < N; i++ {
		for k := 0; k < N; k++ {
			aik := A[i*N+k]
			for j := 0; j < N; j++ {
				C[i*N+j] += aik * B[k*N+j]
			}
		}
	}

	sum := C[0] + C[N-1] + C[N*(N-1)] + C[N*N-1]
	fmt.Println(sum)
}
