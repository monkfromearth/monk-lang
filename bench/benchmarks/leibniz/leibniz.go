//go:build ignore

package main

import "fmt"

func main() {
	N := int64(50_000_000)
	sum := 0.0
	for k := int64(0); k < N; k++ {
		term := 1.0 / float64(2*k+1)
		if k%2 == 0 {
			sum += term
		} else {
			sum -= term
		}
	}
	pi := 4.0 * sum
	scaled := int64(pi * 10000000.0)
	fmt.Println(scaled)
}
