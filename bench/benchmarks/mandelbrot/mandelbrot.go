//go:build ignore

package main

import "fmt"

const (
	W        = 800
	H        = 800
	MaxIter  = 50
	Limit    = 4.0
)

func main() {
	Wf := float64(W)
	Hf := float64(H)
	var count int64 = 0
	for y := 0; y < H; y++ {
		cy := float64(y)/Hf*2.0 - 1.0
		for x := 0; x < W; x++ {
			cx := float64(x)/Wf*3.0 - 2.0
			zx, zy := 0.0, 0.0
			iter := 0
			escaped := false
			for iter < MaxIter {
				zx2 := zx * zx
				zy2 := zy * zy
				if zx2+zy2 > Limit {
					escaped = true
					iter = MaxIter
				} else {
					newZx := zx2 - zy2 + cx
					zy = 2.0*zx*zy + cy
					zx = newZx
					iter++
				}
			}
			if !escaped {
				count++
			}
		}
	}
	fmt.Println(count)
}
