package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

var worker = flag.String("Worker", "localhost:8030", "IP:Connect to worker.")

// Creates the distributor channel struct.
type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// New type for carrying useful references for worker
type rowTask struct {
	startRow int
	endRow   int
	src      [][]byte
	dst      [][]byte
	wg       *sync.WaitGroup
}

// Stubs and Structs
// For single-threaded implementations.
type CompactType struct {
	P      Params
	Height int
	Width  int
	Src    [][]byte
	Dst    [][]byte
}

// Reply type for the AliveCount RPC.
type AliveRes struct {
	Turn  int
	Count int
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	flag.Parse()
	//Creates two worlds for double buffering and calls them current and next world respectively.
	worldA := make([][]byte, p.ImageHeight)
	worldB := make([][]byte, p.ImageHeight)

	fmt.Println("Before turns.")

	for i := range worldA {
		worldA[i] = make([]byte, p.ImageWidth)
	}
	for i := range worldB {
		worldB[i] = make([]byte, p.ImageWidth)
	}

	currentWorld := worldA
	nextWorld := worldB

	//Initialising.
	turn := 0

	c.ioCommand <- ioInput

	//Figured out how to read a file and this puts it into the current world.
	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			select {
			case a := <-c.ioInput:
				if a == 255 {
					c.events <- CellFlipped{CompletedTurns: 0, Cell: util.Cell{X: j, Y: i}}
				}

				currentWorld[i][j] = a
			}
		}
	}

	c.events <- StateChange{turn, Executing}

	//Helper function to save images.
	saveImage := func(currentTurn int) {
		hold := currentWorld

		c.ioCommand <- ioOutput
		c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, currentTurn)
		for r := 0; r < p.ImageHeight; r++ {
			for col := 0; col < p.ImageWidth; col++ {
				c.ioOutput <- hold[r][col]
			}
		}
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- ImageOutputComplete{CompletedTurns: currentTurn, Filename: fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, currentTurn)}
	}

	i := 0
	client, err := rpc.Dial("tcp", *worker)

	if err != nil {
		panic(err)
	}
	defer client.Close()

	response := new(CompactType)
	init := CompactType{P: p, Height: p.ImageHeight, Width: p.ImageWidth, Src: currentWorld, Dst: nextWorld}

	// Ticker: every 2 seconds ask the engine for its alive count and report it.
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var ar AliveRes
				e := client.Call("CellCalc.AliveCount", 0, &ar)
				if e != nil {
					fmt.Println("AliveCount error:", e)
				} else {
					c.events <- AliveCellsCount{CompletedTurns: ar.Turn, CellsCount: ar.Count}
				}
			case <-done:
				return
			}
		}
	}()
	err = client.Call("CellCalc.SingleCalc", init, response)
	close(done)
	<-stopped
	if err != nil {
		panic(err)
	}

	currentWorld = response.Src
	nextWorld = response.Dst
	i++

	//Output saving.
	saveImage(p.Turns)
	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{CompletedTurns: p.Turns, Alive: calculateAliveCells(p, currentWorld)}
	fmt.Println("\nOutput state of the progam.")

	// Make sure that the Io has finished any output before exiting.

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}
