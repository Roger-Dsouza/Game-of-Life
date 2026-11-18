package tests

import (
	"fmt"
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkGol(b *testing.B) {
	os.Stdout = nil
	for _, size := range []int{256, 512, 5120} {
		for threads := 1; threads <= 16; threads *= 2 {
			b.Run(fmt.Sprintf("%dx%d/%d_workers", size, size, threads), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					p := gol.Params{Turns: 100, Threads: threads, ImageWidth: size, ImageHeight: size}
					events := make(chan gol.Event, 5000)
					go gol.Run(p, events, make(chan rune))
					for range events {
					}
				}
			})
		}
	}
}
