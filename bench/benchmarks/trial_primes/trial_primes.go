//go:build ignore

package main

import "fmt"

func main() {
	N := int64(200000)
	count := int64(0)
	for n := int64(2); n <= N; n++ {
		isPrime := true
		for d := int64(2); d*d <= n; d++ {
			if n%d == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			count++
		}
	}
	fmt.Println(count)
}
