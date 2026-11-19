package main

import (
	"os"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"uk.ac.bris.cs/gameoflife/util"
	"time"

)

//Control of the state. This is reset once the conditional loop is exited.
var toQuit=false
//This monitors when the main loop ends. Then it can exit the program.
var loopEnd=false

var control sync.Mutex





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

type StepReq struct {
    Width int
    Rows [][]byte
}

type StepRes struct {
    Rows [][]byte
}
var workers []*rpc.Client

func stepWorkers(p Params, src, dst [][]byte) error {
    H, W := p.ImageHeight, p.ImageWidth
    n := len(workers)
    if n > H {
        n = H
    }
    errs:= make([]error, n)
    var wg sync.WaitGroup
    for i := 0; i < n; i++ {
        start, end := i*H/n, (i+1)*H/n
        rows := make([][]byte, 0 , end-start + 2)
        rows = append(rows, src[(start-1+H)%H])
        rows = append(rows,src[start:end]...)
        rows = append(rows, src[end%H])
        wg.Add(1)
        go func(i, start int, rows [][]byte) {
            defer wg.Done()
            var res StepRes
            if e := workers[i].Call("Worker.Step", StepReq{Width: W, Rows: rows}, &res); e != nil {
                errs[i] = e
                return 
            }
            for k, row := range res.Rows {
                copy(dst[start+k], row)
            }
        }(i, start, rows)
    }
    wg.Wait()
    for _, e := range errs {
        if e != nil {
            return e
        }
    }
    return nil
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


func emptyBoard(p Params) [][]byte {
	board := make([][]byte, p.ImageHeight)
	for y := range board {
		board[y] = make([]byte, p.ImageWidth)
	}
	return board
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
func (l *CellCalc) S(_ int, r *BoardState) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	r.Turn = l.turn
	r.Board = make([][]byte, len(l.world))
	for y := range l.world {
		r.Board[y] = append([]byte(nil), l.world[y]...)
	}
	return nil
}


func (l *CellCalc) GameQuit(a int, b *int) (err error){
	control.Lock()

    toQuit = true
	hasPaused=false

	control.Unlock()
    *b = 1
	
    return

}

func (l *CellCalc) SystemQuit(a int, b *int) (err error){
	control.Lock()
    toQuit = true
	hasPaused=false
	control.Unlock()

    *b = 1

	//Concurrent program which shuts down the goroutine.
	go func(){
		for !loopEnd{
			time.Sleep(10*time.Millisecond)
		}
		time.Sleep(50*time.Millisecond)
		for _, w := range workers {
			var x int
			w.Call("Worker.Shutdown", 0, &x)
			
		}
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)

	}()
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
			control.Lock()

			board=c.Src
			turn=i


			if e := stepWorkers(c.P, c.Src, c.Dst); e != nil {
            	control.Unlock()
            	return e
            }

			if toQuit {
				control.Unlock()
				break
			}

			c.Src, c.Dst = c.Dst, c.Src
			i++
			
			l.mu.Lock()
			l.world, l.turn = c.Src, i
			l.mu.Unlock()


			control.Unlock()


			for hasPaused && !toQuit {
				time.Sleep(10 * time.Millisecond)
			}

			





		
	}
	//Ignores the current turn if it quit.


   turn=i
   board=c.Src
   r.Turn=turn


	l.mu.Lock()
	r.Turn = l.turn

	if toQuit && l.turn==0{
		r.Src=emptyBoard(c.P)
	}else{
		r.Src = make([][]byte, len(l.world))
		for y := range l.world {
			r.Src[y] = append([]byte(nil), l.world[y]...)
		}

	}


	l.mu.Unlock()

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

	loopEnd=true


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
	start := flag.String("port", ":8030", "Listening port.")
	ws := flag.String("workers", "", "Comma-separated worker addresses.")
	flag.Parse()

	for _, addr := range strings.Split(*ws, ",") {
		cl, err := rpc.Dial("tcp", strings.TrimSpace(addr))
		if err != nil {
			panic(err)
		}
		workers = append(workers, cl)
	}
	fmt.Println("Connected to", len(workers), "workers")

	rpc.Register(&CellCalc{})
	listener, err := net.Listen("tcp", *start)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	rpc.Accept(listener)
}
