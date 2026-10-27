package gol

import "uk.ac.bris.cs/gameoflife/util"

//Flag 0 for width, 1 for height.
func isBound(lo int, hi int, val int,maxVal int) int {
	if (val >= lo && val <=hi){
		return val
	} else if (val<lo) {
		return (maxVal-1)
		
	}else{
		return maxVal-val
	}

}



func checkCell(world [][]byte,h int,w int) int{
	if (h!=-1 && w!=-1){
		if (world[h][w]==255){

			return 1;
		}else{
			return 0;
		}
	}else{
		return 0
	}



}


func calculateRows(p Params, world [][]byte,dst [][]byte,startRow int,endRow int) {
	
	var width int = p.ImageWidth
	var height int = p.ImageHeight

    for h := startRow; h < endRow; h++ {
        for w:= 0; w < width; w++ {
            var activeCells int = 0
            var val byte = world[h][w]


			var low int = isBound(0, width-1, w-1,width)
			var loh int = isBound(0, height-1, h-1,height)

			var hiw int = isBound(0, width-1, w+1,width)
			var hih int = isBound(0, height-1, h+1,height)



            activeCells+=(checkCell(world,loh,low)+checkCell(world,hih,hiw))
			activeCells+=(checkCell(world,loh,w)+checkCell(world,hih,w))
			activeCells+=(checkCell(world,h,low)+checkCell(world,h,hiw))
			activeCells+=(checkCell(world,loh,hiw)+checkCell(world,hih,low))
			
			

			switch val {
            case 0:
				if activeCells == 3 {
					dst[h][w] = 255
				}else{
					dst[h][w] = 0
				}

			case 255:
				if activeCells < 2 || activeCells > 3 {
					dst[h][w] = 0

				}else{
					dst[h][w]=255
				}
			}

		}

	}
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

