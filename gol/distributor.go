package gol

import (
	"fmt"
	"sync"
	"time"
)

import "uk.ac.bris.cs/gameoflife/util"



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
func distributor(p Params, c distributorChannels,keyPresses <-chan rune) {
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

	c.ioCommand<-ioInput


	
	
	//Figured out how to read a file and this puts it into the current world.
	c.ioFilename<-fmt.Sprintf("%vx%v",p.ImageHeight,p.ImageWidth)

	for i:=0;i<p.ImageHeight;i++{
		for j:=0;j<p.ImageWidth;j++{
			select{
			case a:=<-c.ioInput:
				if (a==255){
					c.events<-CellFlipped{CompletedTurns:0,Cell:util.Cell{X:j,Y:i}}
				}

				currentWorld[i][j]=a
			}
		}
	}

	c.events <- StateChange{turn, Executing}
    //Infra needed for rowTask.
	tasks := make(chan rowTask, p.Threads)
	var poolWg sync.WaitGroup


    //Helper function to save images.
	saveImage := func(currentTurn int) {
		hold:=currentWorld
	    c.ioCommand <- ioOutput
	    c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, currentTurn)
        for r := 0; r < p.ImageHeight; r++ {
            for col := 0; col < p.ImageWidth; col++ {
                c.ioOutput <- hold[r][col]
            }
        }
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
        c.events <- ImageOutputComplete{CompletedTurns: currentTurn,Filename: fmt.Sprintf("%vx%vx%v",p.ImageWidth,p.ImageHeight,currentTurn)}
    }


	//Control for the pausing, quitting and saving.

    channel:=make (chan util.Cell)
	poolWg.Add(1)
	go func(){
	        
	        for task := range tasks {
	            calculateRows(p, task.src, task.dst, task.startRow, task.endRow,channel)
	            task.wg.Done()
	        }
			poolWg.Done()
	}()
	

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
					fmt.Println("Stopped.")
					stop=a
					close(c.events)
				case a:=<-traffic:
					c.events<-a
					
			
			}

		}


	}(control,traffic)


	//Heartbeat function.
	go func(p Params,world [][]byte,control chan int){
		for {
			time.Sleep(2*time.Second)
			traffic<-AliveCellsCount{CompletedTurns:i,CellsCount:aliveCellCount(p,currentWorld)}
			

		}


	}(p,nextWorld,control)


	paused := false
	quitRequested := false
	quit:=false

	turnUpdate:=make (chan int)
	go func(turn chan int,cell chan util.Cell,event chan<- Event){
		init:=0
		for{
			select{
				case t:=<-turn:
					init=t

				case c:=<-cell:
					event<-CellFlipped{CompletedTurns:init,Cell:c}
			}
		}



	}(turnUpdate,channel,c.events)

    
	for i < p.Turns  && !quitRequested{
		
		select{
		case key:=<-keyPresses:
			switch key{
			case 'p','P':
				paused=!paused
                if paused {
                    c.events <- StateChange{i, Paused}
					for paused{
						select{
						case k:=<-keyPresses:
							switch k{
							case 'p','P':
								paused=!paused
								c.events <- StateChange{i, Executing}

							case 's','S':
								saveImage(i)
							case 'q','Q':
								quit=true
								paused=false
							}
							

						}
					}
                } 
			case 's','S':
				saveImage(i)

			case 'q','Q':
				quit=true
			}
        default:
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
			

			if (quit){
				quitRequested=true
			}
			i++
			turnUpdate<-(i)
			c.events<-TurnComplete{CompletedTurns: i}
		}
	}
	close(tasks)
	poolWg.Wait()

	if (quitRequested){
		c.events<-FinalTurnComplete{CompletedTurns:p.Turns,Alive:calculateAliveCells(p,currentWorld)}
		saveImage(i)
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- StateChange{turn, Quitting}
		control<-1
		


	}else{
			//Output saving.
		saveImage(p.Turns)
		// TODO: Report the final state using FinalTurnCompleteEvent.
		c.events<-FinalTurnComplete{CompletedTurns:p.Turns,Alive:calculateAliveCells(p,currentWorld)}

		// Make sure that the Io has finished any output before exiting.

		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- StateChange{turn, Quitting}
    
	
		// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
		control<-1

	}




	
}
