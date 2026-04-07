package main

import "fmt"

const N = 1000000

func main() {
	bestStart := int64(1)
	bestLen := int64(1)

	for n := int64(2); n <= N; n++ {
		current := n
		chainLen := int64(1)
		for current != 1 {
			if current%2 == 0 {
				current /= 2
			} else {
				current = current*3 + 1
			}
			chainLen++
		}
		if chainLen > bestLen {
			bestLen = chainLen
			bestStart = n
		}
	}
	fmt.Println(bestStart)
}
