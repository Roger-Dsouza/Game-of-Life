package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/util"
)

type CompactType struct {
	P      Params
	Height int
	Width  int
	Src    [][]byte
	Dst    [][]byte
}

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

// Reply type for the AliveCount RPC.
type AliveRes struct {
	Turn  int
	Count int
}

// Flag 0 for width, 1 for height.
func isBound(lo int, hi int, val int, maxVal int) int {
	if val >= lo && val <= hi {
		return val
	} else if val < lo {
		return (maxVal - 1)

	} else {
		return maxVal - val
	}

}

// Struct for the Game of Life Method.
type CellCalc struct {
	mu    sync.Mutex
	world [][]byte
	turn  int
	p     Params
}

func checkCell(world [][]byte, h int, w int) int {
	if h != -1 && w != -1 {
		if world[h][w] == 255 {

			return 1
		} else {
			return 0
		}
	} else {
		return 0
	}

}

// Function to match client.Call.
func (l *CellCalc) SingleCalc(c CompactType, r *CompactType) (err error) {
	l.mu.Lock()
	l.world, l.turn, l.p = c.Src, 0, c.P
	l.mu.Unlock()

	for i := 0; i < c.P.Turns; i++ {
		l.mu.Lock()
		CalculateRows(c.P, c.Src, c.Dst, 0, c.P.ImageHeight)
		c.Src, c.Dst = c.Dst, c.Src
		l.world, l.turn = c.Src, i+1
		l.mu.Unlock()
	}
	r.Src = c.Src
	r.P = c.P
	r.Dst = c.Dst

	return
}

// Reports the current turn and alive cell count.
func (l *CellCalc) AliveCount(req int, res *AliveRes) error {
	fmt.Println("AliveCount called")
	l.mu.Lock()
	defer l.mu.Unlock()
	res.Turn = l.turn
	if l.world != nil {
		res.Count = aliveCellCount(l.p, l.world)
	}
	return nil
}

// Diagnostic function.
func (l *CellCalc) Ping(c CompactType, r *CompactType) (err error) {
	fmt.Println("Ping pong.")
	return
}

func CalculateRows(p Params, world [][]byte, dst [][]byte, startRow int, endRow int) (err error) {
	var width int = p.ImageWidth
	var height int = p.ImageHeight

	for h := startRow; h < endRow; h++ {
		for w := 0; w < width; w++ {
			var activeCells int = 0
			var val byte = world[h][w]

			var low int = isBound(0, width-1, w-1, width)
			var loh int = isBound(0, height-1, h-1, height)

			var hiw int = isBound(0, width-1, w+1, width)
			var hih int = isBound(0, height-1, h+1, height)

			activeCells += (checkCell(world, loh, low) + checkCell(world, hih, hiw))
			activeCells += (checkCell(world, loh, w) + checkCell(world, hih, w))
			activeCells += (checkCell(world, h, low) + checkCell(world, h, hiw))
			activeCells += (checkCell(world, loh, hiw) + checkCell(world, hih, low))

			switch val {
			case 0:
				if activeCells == 3 {
					dst[h][w] = 255

				} else {
					dst[h][w] = 0
				}

			case 255:
				if activeCells < 2 || activeCells > 3 {
					dst[h][w] = 0

				} else {
					dst[h][w] = 255

				}
			}

		}

	}
	return
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var active []util.Cell
	var width int = p.ImageWidth
	var height int = p.ImageHeight

	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			if world[h][w] == 255 {
				c := util.Cell{Y: h, X: w}
				active = append(active, c)
			}

		}

	}
	return active
}

func aliveCellCount(p Params, world [][]byte) int {
	total := 0
	var width int = p.ImageWidth
	var height int = p.ImageHeight

	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			if world[h][w] == 255 {
				total++

			}

		}

	}
	return total
}

func main() {
	start := flag.String("port", "localhost:8030", "Listening port.")
	flag.Parse()
	rpc.Register(&CellCalc{})
	listener, err := net.Listen("tcp", *start)

	if err != nil {
		panic(err)
	}

	defer listener.Close()
	rpc.Accept(listener)

}
