package tests

import (
	"fmt"
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkGol(b *testing.B) {
	os.Stdout = nil
	for threads := 1; threads <= 16; threads *= 2 {
		b.Run(fmt.Sprintf("%d_workers", threads), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				p := gol.Params{
					Turns:       5,
					Threads:     threads,
					ImageWidth:  256,
					ImageHeight: 256,
				}

				events := make(chan gol.Event, 5000)
				keyPresses := make(chan rune)
				go gol.Run(p, events, keyPresses)
				for range events {
				}
			}
		})
	}
}
