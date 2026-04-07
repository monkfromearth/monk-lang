package main

import "fmt"

const N = 1000000

func main() {
	isPrime := make([]bool, N+1)
	for i := range isPrime {
		isPrime[i] = true
	}
	isPrime[0] = false
	isPrime[1] = false

	for i := 2; i*i <= N; i++ {
		if isPrime[i] {
			for j := i * i; j <= N; j += i {
				isPrime[j] = false
			}
		}
	}

	count := 0
	for i := 2; i <= N; i++ {
		if isPrime[i] {
			count++
		}
	}
	fmt.Println(count)
}
