package gol

import (
	"fmt"
	"sync"
	"time"
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
	//Creates two worlds for double buffering and calls them current and next world respectively.
    worldA := make([][]byte, p.ImageHeight)
	worldB := make([][]byte, p.ImageHeight)

    for i:= range worldA {
        worldA[i] = make([]byte,p.ImageWidth)
    }
    for i:= range worldB {
        worldB[i] = make([]byte,p.ImageWidth)
    }

    currentWorld := worldA
    nextWorld := worldB

    //Initialising.
	turn := 0
	c.events <- StateChange{turn, Executing}
	c.ioCommand<-ioInput


	
	
	//Figured out how to read a file and this puts it into the current world.
	c.ioFilename<-fmt.Sprintf("%vx%v",p.ImageHeight,p.ImageWidth)

	for i:=0;i<p.ImageHeight;i++{
		for j:=0;j<p.ImageWidth;j++{
			select{
			case a:=<-c.ioInput:
				currentWorld[i][j]=a
			}
		}
	}

    //Infra needed for rowTask.
	tasks := make(chan rowTask, p.Threads)
	var poolWg sync.WaitGroup

	for i := 0; i < p.Threads; i++ {
	    poolWg.Add(1)
	    go func(){
	        
	        for task := range tasks {
	            calculateRows(p, task.src, task.dst, task.startRow, task.endRow)
	            task.wg.Done()
	        }
			poolWg.Done()
	    }()
	}

	rowsPerThread := p.ImageHeight / p.Threads
	i:=0

    control:=make (chan int)
    traffic:=make (chan Event)

	//Router function
	go func(control chan int,traffic chan Event){
		stop:=0

		for stop==0{
			select{
				case a:=<-control:
					stop=a
				case a:=<-traffic:
					c.events<-a
					
			
			}

		}


	}(control,traffic)


	go func(p Params,world [][]byte,control chan int){
		time.Sleep(2*time.Second)
		traffic<-AliveCellsCount{CompletedTurns:i,CellsCount:aliveCellCount(p,currentWorld)}

	}(p,nextWorld,control)


	for i < p.Turns {
	    var turnWg sync.WaitGroup
	    turnWg.Add((p.Threads))

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
		i++


	}

	close(tasks)
	poolWg.Wait()
	//Output saving.
	c.ioCommand<-ioOutput
	c.ioFilename<-fmt.Sprintf("%vx%vx%v",p.ImageHeight,p.ImageWidth,p.Turns)

	for i:=0;i<p.ImageHeight;i++{
		for j:=0;j<p.ImageWidth;j++{
			c.ioOutput<-currentWorld[i][j]

		}
	}
    

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events<-FinalTurnComplete{CompletedTurns:p.Turns,Alive:calculateAliveCells(p,currentWorld)}

	// Make sure that the Io has finished any output before exiting.

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
    
	
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	control<-1
	close(c.events)
}
