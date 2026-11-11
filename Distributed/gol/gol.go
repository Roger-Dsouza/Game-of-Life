package gol

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {

	//	TODO: Put the missing channels in here.

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	iFile:=make(chan string)
	iIn:=make(chan uint8)
	iOut:=make(chan uint8)

	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: iFile,
		output:   iOut,
		input:    iIn,
	}
	go startIo(p, ioChannels)

    //Same channel so that the communication from ditributor goes to IO.

	distributorChannels := distributorChannels{
		events:     events,
		ioCommand:  ioCommand,
		ioIdle:     ioIdle,
		ioFilename: iFile,
		ioOutput:   iOut,
		ioInput:    iIn,
	}
	distributor(p, distributorChannels)
}
