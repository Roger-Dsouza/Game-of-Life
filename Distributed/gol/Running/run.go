package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/util"
	"time"

)

//Control of the state. This is reset once the conditional loop is exited.
var toQuit=false



type CompactType struct {
                         // This will notify about the turn at a point.
	Turn int
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

type PauseState struct {
	State bool
	Turn int
}

// Reply type for the AliveCount RPC.
type AliveRes struct {
	Turn  int
	Count int
}

type BoardState struct {
	Turn int
	Board [][]byte
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

//These global variables store the turn and the current state of the board.
var turn int=0
var board [][]byte

//Returns with the turn and the state at that moment.
func (l *CellCalc) S(c int,r *BoardState) (err error){
	r.Turn=turn
	r.Board=board

	return

}


func (l *CellCalc) GameQuit(a int, b *int) (err error){
    toQuit = true
	hasPaused=false
    *b = 1
    return

}


var hasPaused=false
func (l *CellCalc) GamePause(a bool, b *PauseState) (err error){
	fmt.Printf("Before: %v\n", hasPaused)
	hasPaused = !hasPaused
	b.State=hasPaused
	b.Turn=turn
	fmt.Printf("After: %v\n", hasPaused)


	return
}




// Function to match client.Call.
func (l *CellCalc) SingleCalc(c CompactType, r *CompactType) (err error) {
	fmt.Println("Quit or not:",toQuit)
	board=c.Src
	l.mu.Lock()
	l.world, l.turn, l.p = c.Src, 0, c.P
	l.mu.Unlock()
	
    i:=0
	for toQuit==false && i<c.P.Turns{
		   
			board=c.Src
			turn=i

			l.mu.Lock()
			CalculateRows(c.P, c.Src, c.Dst, 0, c.P.ImageHeight)

			
			c.Src, c.Dst = c.Dst, c.Src
			

			
			
			l.world, l.turn = c.Src, i+1
			l.mu.Unlock()

			i++

			for (hasPaused && toQuit==false){
				time.Sleep(100*time.Millisecond)
			}
		
	}
	//Ignores the current turn.


   turn=i
   board=c.Src
   r.Turn=turn



    
	r.Src = c.Src
	r.P = c.P
	r.Dst = c.Dst


	if (toQuit==true){
		fmt.Println("Program quit because client said so.")
		fmt.Println("Parameters:",r.P)

	}else{
		fmt.Println("Returning from game logic.")

	}

	toQuit=false

	for (hasPaused && toQuit==false){
		fmt.Println(hasPaused)
		time.Sleep(100*time.Millisecond)

	}


	return
}

// Reports the current turn and alive cell count.
func (l *CellCalc) AliveCount(req int, res *AliveRes) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	res.Turn = l.turn
	if l.world != nil {
		res.Count = aliveCellCount(l.p, l.world)
	}
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
