package main

import "fmt"

const maxDepth = 14

func treeSize(depth int) int64 {
	size := int64(1)
	for i := 0; i <= depth; i++ {
		size *= 2
	}
	return size - 1
}

func buildAndCheck(depth int) int64 {
	numNodes := treeSize(depth)
	pool := make([]int64, numNodes*3)
	half := numNodes / 2

	for i := numNodes - 1; i >= half; i-- {
		base := i * 3
		pool[base] = 1
		pool[base+1] = -1
		pool[base+2] = -1
	}

	for i := half - 1; i >= 0; i-- {
		base := i * 3
		pool[base] = 0
		pool[base+1] = 2*i + 1
		pool[base+2] = 2*i + 2
	}

	stack := make([]int64, 0, numNodes)
	stack = append(stack, 0)
	check := int64(0)

	for len(stack) > 0 {
		nodeIdx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		base := nodeIdx * 3
		check += pool[base]
		if pool[base+1] >= 0 {
			stack = append(stack, pool[base+1])
			stack = append(stack, pool[base+2])
		}
	}
	return check
}

func main() {
	total := buildAndCheck(maxDepth + 1)
	longLived := buildAndCheck(maxDepth)

	for depth := 4; depth <= maxDepth; depth += 2 {
		iterations := int64(1)
		for d := 0; d < maxDepth-depth+4; d++ {
			iterations *= 2
		}
		for i := int64(0); i < iterations; i++ {
			total += buildAndCheck(depth)
		}
	}
	total += longLived
	fmt.Println(total)
}
