package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

var worker = flag.String("Worker", "13.220.202.132:8030", "IP:Connect to worker.")
//Generic int pointer that is a blank response (could be used for debugging).
var intLog=0
var orchestrate *int=&intLog





// Creates the distributor channel struct.
type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}


// Stubs and Structs
// For single-threaded implementations.
type CompactType struct {
	            //Shows the turn of the machine.
	Turn int
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

//This collects the state of the board at a particular point.
type BoardState struct {
	Turn int
	Board [][]byte
}

type PauseState struct {
	State bool
	Turn int
}

//State Coordinator.
var loop sync.WaitGroup
    //This is to communicate with all running instances.

//This is the quit global variable, blocks normal quit procedure from happening.

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels,keyPresses <-chan rune) {
	//Mutex to hold race conditions.
	


	flag.Parse()
	//Creates two worlds for double buffering and calls them current and next world respectively.
	worldA := make([][]byte, p.ImageHeight)
	worldB := make([][]byte, p.ImageHeight)

	

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

	//Modifies the provided saveImage function towards saving the board response into the IO.
	
	savePacketImage := func(b BoardState) {
		c.ioCommand <- ioOutput
		c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, b.Turn)
		for r := 0; r < p.ImageHeight; r++ {
			for col := 0; col < p.ImageWidth; col++ {
				c.ioOutput <- b.Board[r][col]
			}
		}
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- ImageOutputComplete{CompletedTurns: b.Turn, Filename: fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, b.Turn)}
	}
    




	

	client, err := rpc.Dial("tcp", *worker)

	if err != nil {
		panic(err)
	}
	defer client.Close()

	response := new(CompactType)
	shuttle:=new(BoardState)
	init := CompactType{Turn:0,P: p, Height: p.ImageHeight, Width: p.ImageWidth, Src: currentWorld, Dst: nextWorld}

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

	//There will be a bunch of functions running at the same time in the distributor, mutliple 'jobs sent

	
    
	go func(){
	roughTurn:=0
	
	hasQuit:=false
	for hasQuit==false{
		
		key:=<-keyPresses
		switch key{
			case 'p','P':
				pauser:=new(PauseState)
				fmt.Println("Before",pauser)
				pauser.Turn=roughTurn
				client.Call("CellCalc.GamePause",true,pauser)
				roughTurn=pauser.Turn
								

				if (pauser.State==false){
					fmt.Println("Continuing....")
					c.events <- StateChange{roughTurn, Executing}
				}else{
					fmt.Println("Current turn:",roughTurn)
					c.events<-StateChange{roughTurn,Paused}
				}


					
			case 's','S':
					fmt.Println("Save pressed.")
					err=client.Call("CellCalc.S",1,shuttle)
					savePacketImage(*shuttle)

			case 'q','Q':
				
				fmt.Println("Quit pressed.")
				err=client.Call("CellCalc.GameQuit",1,orchestrate)

			case 'k','K':
				fmt.Println("Stopping Distributed Machine.")
				err=client.Call("CellCalc.SystemQuit",1,orchestrate)
				
					

				
		}
		


	}
	}()
	

	//This function just lets the turns be independent from keyPresses.
    //This is the function which calls the client for the game logic state.
	gameLogic:=func(){
			err = client.Call("CellCalc.SingleCalc", init, response)
			if err != nil {
				panic(err)
			}
			loop.Done()
			
	}

	//This lets the gameLogic run in a seperate window.
	
	loop.Add(1)
	go gameLogic()
	loop.Wait()


	
	close(done)
	<-stopped


	currentWorld = response.Src
	nextWorld = response.Dst



	//Output saving.
	saveImage(response.Turn)
	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{CompletedTurns: response.Turn, Alive: calculateAliveCells(p, currentWorld)}

	// Make sure that the Io has finished any output before exiting.

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{response.Turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

	


}
