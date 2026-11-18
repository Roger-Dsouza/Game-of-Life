package main

import (
	"flag"
	"net"
	"net/rpc"
	"os"
	"time"
)

type StepReq struct {
	Width int
	Rows  [][]byte
}

type StepRes struct {
	Rows [][]byte
}

type Worker struct{}

// Step computes the next state of rows 1..len-2; rows 0 and len-1 are halos.
func (w *Worker) Step(req StepReq, res *StepRes) error {
	h := len(req.Rows) - 2
	W := req.Width
	out := make([][]byte, h)
	for y := 1; y <= h; y++ {
		row := make([]byte, W)
        top := req.Rows[y-1]
		middle := req.Rows[y]
		bottom := req.Rows[y+1]

		for x := 0; x < W; x++ {
			left, right := x-1, x+1
			if left < 0 {
				left = W - 1
			}
			if right == W {
				right = 0
			}

		n := 0
		for _, cell := range []byte{
			top[left], top[x], top[right],
			middle[left], middle[right],
			bottom[left], bottom[x], bottom[right],
		} {
		if cell == 255 {
			n++
		}
	}

	alive := middle[x] == 255
	if (alive && (n == 2 || n == 3)) || (!alive && n == 3) {
		row[x] = 255
	}
}

		out[y-1] = row
	}
	res.Rows = out
	return nil
}

func (w *Worker) Shutdown(a int, b *int) error {
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
	return nil
}

func main() {
	port := flag.String("port", "localhost:8031", "Listening address.")
	flag.Parse()
	rpc.Register(&Worker{})
	ln, err := net.Listen("tcp", *port)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	rpc.Accept(ln)
}