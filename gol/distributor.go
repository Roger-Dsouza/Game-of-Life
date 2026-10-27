package gol

import (
	"fmt"
	"sync"
)

//Creates the distributor channel struct.
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
    endRow int
    src [][]byte
    dst [][]byte
    wg *sync.WaitGroup
}


// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
    worldA := make([][]byte, p.ImageHeight)
    for i:= range worldA {
        worldA[i] = make([]byte,p.ImageWidth)
    }
    for i:= range worldB {
        worldB[i] = make([]byte,p.ImageWidth)
    }
// Allocate two worlds for double buffering
    currentWorld := worldA
    nextWorld := worldB

	turn := 0
	c.events <- StateChange{turn, Executing}

	c.ioCommand<-ioInput
	
	//Figured out how to read a file.
	c.ioFilename<-fmt.Sprintf("%vx%v",p.ImageHeight,p.ImageWidth)

	for i:=0;i<p.ImageHeight;i++{
		for j:=0;j<p.ImageWidth;j++{
			select{
			case a:=<-c.ioInput:
				currentWorld[i][j]=a
			}
		}
	}

	tasks := make(chan rowTask, p.Threads)
	var poolWg sync.WaitGroup

	for i := 0; i < p.Threads; i++ {
	    poolWg.Add(1)
	    go func() {
	        defer poolWg.Done()
	        for task := range tasks {
	            calculateRows(p, task.src, task,dst, task.startRow, task.endRow)
	            task.wg.Done()
	        }
	    }
	}

	rowsPerThread := p.ImageHeight / p.Threads

	for i := turn; i < p.Turns; i++ {
	    var turnWg sync.WaitGroup
	    turnWg.Add(p.turns)

	    for t := 0; t < p.Threads; t++ {
	        start := t * rowsPerThread
	        end := start + rowsPerThread
	        if t == p.Threads - 1 {
	            end = p.ImageHeight
	        }

	        tasks <- rowTask{
	            startRow: start,
	            endRow: end,
	            src: currentWorld,
	            dst: nextWorld,
	            wg: &turnWg,
	        }
	    }

	    turnWg.Wait()

	    currentWorld, nextWorld = nextWorld, currentWorld

	    c.events <- StateChange{i, Executing}
	}

	close(tasks)
	poolWg.Wait()
	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events<-FinalTurnComplete{CompletedTurns:p.Turns,Alive:calculateAliveCells(p,nworld)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
    
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
