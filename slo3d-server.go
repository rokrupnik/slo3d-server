/*
Serve is a very simple static file server in go
Usage:
	-p="8888": port to serve on
	-d=".":    the directory of static files to host
Navigating to http://localhost:8888 will display the index.html or directory
listing file.
*/
package main

import (
    "fmt"
	"flag"
	"log"
	"net/http"
	"strconv"
	"os"
	"errors"
	"image"
	"image/png"
	"image/draw"
)
// "github.com/nfnt/resize"

const rootFolder = "/home/rokr/slo3d/"
var tileDimensions = map[int]int{
    10: 1000,
    9: 500,
    8: 250,
    7: 125,
    6: 63,
    5: 32,
    4: 16,
    3: 8,
    2: 4,
    1: 2,
}

var levelId2TileLevel = map[int]int64{
	2: 2,
	3: 2,
	4: 2,
	5: 2,
	6: 2,
	7: 3,
	8: 4,
	9: 5,
	10: 7,
	11: 10,
}

func NearestHigherPow2 (n uint) uint {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++

	return n
}

func LoadTile (x, y float64, levelId int64) (img image.Image, err error) {
	// Check if coordinates are in range
	if (x < 374000 ||
		623000 < x ||
		y < 31000 ||
		194000 < y)  {
		errRange := errors.New(fmt.Sprintf("Parameters out of range: %d, %f, %f", levelId, x, y))
		// fmt.Println(errRange)

		return nil, errRange
	}


	imgFileName := fmt.Sprintf("%sdata/tiles/%d/%d_%d.png", rootFolder, levelId, int(x/1000), int(y/1000))

	// fmt.Println("Opening image: " + imgFileName)

	imgFile, err := os.Open(imgFileName)
	if err != nil {
		// fmt.Println(err)
        return nil, err
	}
	img, errDecode := png.Decode(imgFile)
	if errDecode != nil {
		fmt.Println(err)
        return img, errDecode
	}

	return
}

func GenerateHeightMap (x0, y0 float64, dim, levelId int64) image.Image {

	// Prepare a wide enough height map by combining neighbouring tiles and then cropping them
	// startInit := time.Now()

	var x1 = x0 + float64(dim)
	var y1 = y0 + float64(dim)

	// Find how much tiles do you need in x and y direction
    var xTile0 = float64(int(x0/1000)*1000)
    var yTile0 = float64(int(y0/1000)*1000)

	var xTile1 = x0
	var yTile1 = y0

	var nTilesX = 1
	var nTilesY = 1

	for yTile1 < y1 {
		for xTile1 < x1 {
			xTile1 += 1000
			nTilesX += 1
		}
		yTile1 += 1000
		nTilesY += 1
	}

	xTile1 = xTile1
	yTile1 = yTile1
    // fmt.Println(fmt.Sprintf("nTilesX: %d, nTilesY: %d", nTilesX, nTilesY))
	// fmt.Println(fmt.Sprintf("Init tiles took: %s", time.Since(startInit)))

	// Prepare image for the whole area
	// startTiles := time.Now()
    var tileDim  = tileDimensions[int(levelId)]
	var areaDimX = nTilesX * tileDim
	var areaDimY = nTilesY * tileDim
	r := image.Rectangle{image.Point{0, 0}, image.Point{areaDimX, areaDimY}}
	area := image.NewNRGBA(r)
    // fmt.Println(fmt.Sprintf("areaDim: %d, tileDim: %d, xy: %f, %f", areaDimY, tileDim, x0, y0))

	// Load tiles
	var xDraw, yDraw int
    iy, ix := 0, 0
	for yTile := yTile0; yTile <= yTile1; yTile += 1000 {
        yDraw = areaDimY - ((iy + 1) * tileDim) // Y coordinate in images increases from top to bottom of image
        //fmt.Println("---------------------------------")

        ix = 0
		for xTile := xTile0; xTile <= xTile1; xTile += 1000 {
			xDraw = ix * tileDim

			img, errTile := LoadTile(xTile, yTile, levelId)

            // fmt.Println(fmt.Sprintf("xyTile: %f, %f", xTile, yTile))
            // fmt.Println(fmt.Sprintf("xyDraw: %d, %d", xDraw, yDraw))
            if errTile == nil {
                draw.Draw(area, image.Rectangle{image.Point{xDraw, yDraw}, image.Point{xDraw + tileDim, yDraw + tileDim}}, img, image.Point{0, 0}, draw.Src)
            }
            ix += 1
		}
        iy += 1
	}
	// fmt.Println(fmt.Sprintf("Loading tiles took: %s", time.Since(startTiles)))

	// Crop the prepared map to correct dimensions
	// startCrop := time.Now()
    var dimScale = float64(tileDim)/1000
	//var numTilesToCrop = int(dim) / 1024
    var scaledDim = int(float64(dim) * dimScale)//tileDim * numTilesToCrop//
	r = image.Rectangle{image.Point{0, 0}, image.Point{scaledDim, scaledDim}}
	croppedArea := image.NewNRGBA(r)

    xCrop := int((x0 - xTile0) * dimScale)
    yCrop := areaDimY - int((y0 - yTile0) * dimScale) - scaledDim
	draw.Draw(croppedArea, r, area, image.Point{xCrop, yCrop}, draw.Src)
	// fmt.Println(fmt.Sprintf("Cropping took: %s", time.Since(startCrop)))

    // Downscale image for faster Loading
    // startThumb1 := time.Now()
	// resizeDim := NearestHigherPow2(uint(scaledDim))
	// fmt.Println(fmt.Sprintf("dimScale per level: %d, %d, %d", dim, tileDim, scaledDim))
    // croppedAreaResized := resize.Resize(resizeDim, resizeDim, croppedArea, resize.Bicubic)
	// fmt.Println(fmt.Sprintf("Downscaling took: %s", time.Since(startThumb1)))

	return croppedArea
}

func main() {
	port := flag.String("p", "8888", "port to serve on")
	directory := flag.String("d", rootFolder, "the directory of static file to host")
	flag.Parse()

	h := http.NewServeMux()

	h.HandleFunc("/heightmaps", func(w http.ResponseWriter, r *http.Request) {

		// startParse := time.Now()
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintln(w, "Bad request - form data could not be parsed.")

			return
		}
		
		q := r.Form

		x, err0 := strconv.ParseFloat(q.Get("x"), 64)
		y, err1 := strconv.ParseFloat(q.Get("y"), 64)
		dim, err2 := strconv.ParseInt(q.Get("dim"), 10, 0)
		levelId, err3 := strconv.ParseInt(q.Get("levelId"), 10, 0)

		levelId = levelId2TileLevel[int(levelId)]

		if err0 != nil || err1 != nil || err2 != nil || err3 != nil {
			w.WriteHeader(400)
			fmt.Fprintln(w, "Bad request - form data is of wrong type.")

			return
		}
		// fmt.Println(fmt.Sprintf("Parsing request took: %s", time.Since(startParse)))

		// start := time.Now()
		img := GenerateHeightMap(x, y, dim, levelId)
		// fmt.Println(fmt.Sprintf("GenerateHeightMap took: %s", time.Since(start)))


		// startResponse := time.Now()
		err4 := png.Encode(w, img)
		if err4 != nil {
			log.Fatal(err)
		}
		// fmt.Println(fmt.Sprintf("Writing response took: %s", time.Since(startResponse)))

	})

	h.Handle("/", http.FileServer(http.Dir(*directory)))

	err := http.ListenAndServe(":"+*port, h)
	log.Fatal(err)
}
