package gol

import (
	"fmt"
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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
    nworld := make([][]byte, p.ImageHeight)
	for i := range nworld {
		nworld[i] = make([]byte, p.ImageWidth)
	}


	turn := 0
	c.events <- StateChange{turn, Executing}

	c.ioCommand<-ioInput
	
	//Figured out how to read a file.
	c.ioFilename<-fmt.Sprintf("%vx%v",p.ImageHeight,p.ImageWidth)

	for i:=0;i<p.ImageHeight;i++{
		for j:=0;j<p.ImageWidth;j++{
			select{
			case a:=<-c.ioInput:
				nworld[i][j]=a
			}
		}
	}
    


	// TODO: Execute all turns of the Game of Life.
	for  i:=turn;i<p.Turns;i++{
		nworld=calculateNextState(p,nworld)
		c.events <- StateChange{i, Executing}

	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events<-FinalTurnComplete{CompletedTurns:p.Turns,Alive:calculateAliveCells(p,nworld)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
    
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
