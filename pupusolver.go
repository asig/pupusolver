/*
 * Copyright (c) 2024 Andreas Signer <asigner@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/png"
	"os"
	"strings"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	zoom = 4

	tileW = 16
	tileH = 16

	playfieldW = 12
	playfieldH = 12
)

var (
	flagLevelData = flag.String("level", `............
............
.....##.....
....# C#....
...#  PC#...
..#   GPR#..
..#   RG+#..
...#  +C#...
....# R#....
.....##.....
............
............
`, "level data")

	flagScreenshot = flag.String("screenshot", "", "Load level data from screenshot")
)

// ================================================
// == TILES
// ==

type tile int

const (
	// Need to be in the same order as in tiles.png!
	tile0     tile = iota // H(eart)
	tile1                 // D(iamond)
	tile2                 // T(riangle)
	tile3                 // C(ircle)
	tile4                 // +
	tile5                 // (hour)G(lass)
	tile6                 // P(ate)
	tile7                 // R(ectangle)
	tileWall              // '#'
	tileBg                // '.'
	tileFloor             // ' '
)

func (t tile) isMobile() bool {
	return t >= tile0 && t <= tile7
}

// ================================================
// == PLAYFIELD
// ==

var (
	tileToChar = make(map[tile]rune)
	charToTile = make(map[rune]tile)
)

func addTileMapping(r rune, t tile) {
	tileToChar[t] = r
	charToTile[r] = t
}

func initTileMap() {
	addTileMapping('H', tile0)
	addTileMapping('D', tile1)
	addTileMapping('T', tile2)
	addTileMapping('C', tile3)
	addTileMapping('+', tile4)
	addTileMapping('G', tile5)
	addTileMapping('P', tile6)
	addTileMapping('R', tile7)
	addTileMapping('#', tileWall)
	addTileMapping('.', tileBg)
	addTileMapping(' ', tileFloor)
}

type move struct {
	fromY, fromX int
	toX          int
}

type tiles [playfieldH][playfieldW]tile
type playfield struct {
	tiles tiles
	path  []move
}

func (pf *playfield) clone() *playfield {
	pf2 := playfield{}
	pf2.tiles = pf.tiles
	pf2.path = append(pf2.path, pf.path...)
	return &pf2
}

func (pf *playfield) apply(m move) *playfield {
	pf2 := pf.clone()
	pf2.path = append(pf2.path, m)

	y := m.fromY

	t := pf2.tiles[y][m.fromX]
	pf2.tiles[y][m.fromX] = tileFloor
	pf2.tiles[y][m.toX] = t

	for {
		// drop all the tiles that can drop
		changed := pf2.dropTiles()

		// remove all the tiles that can be removed
		changed = changed || pf2.removeTiles()

		if !changed {
			return pf2
		}
	}
}

func (pf *playfield) removeTiles() bool {
	changed := false
	for y := 0; y < playfieldH-1; y++ {
		for x := 0; x < playfieldW-1; x++ {
			t := pf.tiles[y][x]
			if !t.isMobile() {
				continue
			}
			if t == pf.tiles[y+1][x] {
				pf.tiles[y][x] = tileFloor
				pf.tiles[y+1][x] = tileFloor
				changed = true
			}
			if t == pf.tiles[y][x+1] {
				pf.tiles[y][x] = tileFloor
				pf.tiles[y][x+1] = tileFloor
				changed = true
				if t == pf.tiles[y+1][x+1] {
					pf.tiles[y+1][x+1] = tileFloor
					changed = true
				}
			}
		}
	}
	return changed
}

func (pf *playfield) dropTiles() bool {
	changed := false
	for y := playfieldH - 2; y >= 0; y-- {
		for x := 0; x < playfieldW; x++ {
			t := pf.tiles[y][x]
			if t.isMobile() && pf.tiles[y+1][x] == tileFloor {
				// let it fall
				y2 := y
				for pf.tiles[y2+1][x] == tileFloor {
					y2++
				}
				pf.tiles[y][x] = tileFloor
				pf.tiles[y2][x] = t
				changed = true
			}
		}
	}
	return changed
}

func (pf *playfield) isSolved() bool {
	for y := playfieldH - 2; y >= 0; y-- {
		for x := 0; x < playfieldW; x++ {
			t := pf.tiles[y][x]
			if t >= tile0 && t <= tile7 {
				return false
			}
		}
	}
	return true
}

func (pf *playfield) isSolvable() bool {
	cnts := make([]int, 8)
	for y := playfieldH - 2; y >= 0; y-- {
		for x := 0; x < playfieldW; x++ {
			t := pf.tiles[y][x]
			if t >= tile0 && t <= tile7 {
				cnts[t]++
			}
		}
	}
	for _, cnt := range cnts {
		if cnt == 1 {
			return false
		}
	}
	return true

}

func (pf *playfield) possibleMoves() []move {
	var moves []move

	for y := 0; y < playfieldH; y++ {
		for x := 0; x < playfieldW; x++ {
			t := pf.tiles[y][x]
			if !t.isMobile() {
				continue
			}

			// Generate all moves
			for _, dirX := range []int{-1, 1} {
				x2 := x + dirX
				for pf.tiles[y][x2] == tileFloor {
					// We can move here!
					moves = append(moves, move{fromY: y, fromX: x, toX: x2})
					if pf.tiles[y+1][x2] == tileFloor || pf.tiles[y+1][x2] == t {
						// Floor or same tile: we're done
						break
					}
					x2 += dirX
				}
			}
		}
	}
	return moves
}

func (pf *playfield) render(r *sdl.Renderer) {
	r.SetDrawColor(0, 255, 55, 255)
	r.Clear()
	for y := 0; y < playfieldH; y++ {
		for x := 0; x < playfieldW; x++ {
			t := pf.tiles[y][x]
			srcRect := &sdl.Rect{X: int32(t * tileW), Y: 0, W: tileW, H: tileH}
			dstRect := &sdl.Rect{X: int32(x * tileW * zoom), Y: int32(y * tileH * zoom), W: int32(tileW * zoom), H: int32(tileH * zoom)}
			r.Copy(tilesTexture, srcRect, dstRect)
		}
	}
}

func (pf *playfield) dump() {
	for y := 0; y < playfieldH; y++ {
		for x := 0; x < playfieldW; x++ {
			fmt.Printf("%c", tileToChar[pf.tiles[y][x]])
		}
		fmt.Println()
	}
}

func badLevelData() {
	fmt.Fprintf(os.Stderr, `Bad level data, needs to be 12 lines of 12 chars per line.

Valid characters:
'H' -> Heart tile
'D' -> Diamond tile
'T' -> Triangle tile
'C' -> Circle tile
'+' -> Cross tile
'G' -> Hourglass tile
'P' -> Pate cross tile
'R' -> Rectangle tile
'#' -> Wall
'.' -> Background
' ' -> Floor

Example data (Level 93):

............
............
.....##.....
....# C#....
...#  PC#...
..#   GPR#..
..#   RG+#..
...#  +C#...
....# R#....
.....##.....
............
............
`)
	os.Exit(1)
}

func playfieldFromString(text string) *playfield {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if len(l) > 0 {
			lines = append(lines, l)
		}
	}

	if len(lines) != playfieldH {
		badLevelData()
	}

	var res playfield
	for y, l := range lines {
		if len(l) != playfieldW {
			badLevelData()
		}
		for x, c := range l {
			t, found := charToTile[c]
			if !found {
				fmt.Fprintf(os.Stderr, "'%c' is not a valid tile.\n", c)
				badLevelData()
			}
			res.tiles[y][x] = t
		}
	}
	return &res
}

func colToInt(c color.Color) int {
	r, g, b, _ := c.RGBA()
	if r == 0 && g == 0 && b == 0 {
		return 0
	}
	return 1
}

func playfieldFromScreenshot(screenshot string) *playfield {
	// First, load the tiles for comparison
	r := bytes.NewReader(tilesData)
	img, _, err := image.Decode(r)
	if err != nil {
		panic(err)
	}
	nofTiles := 11
	tileLineW := nofTiles * tileW
	var tilesPix = make([]int, tileLineW*tileH)
	for y := 0; y < tileH; y++ {
		for x := 0; x < 11*tileW; x++ {
			tilesPix[y*tileLineW+x] = colToInt(img.At(x, y))
		}
	}

	// Now load screenshot
	f, err := os.Open(screenshot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open screenshot: %v", err)
		os.Exit(1)
	}
	defer f.Close()
	img, _, err = image.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load screenshot: %v", err)
		os.Exit(1)
	}
	levelW := img.Bounds().Dx()
	levelH := img.Bounds().Dy()
	var levelPix = make([]int, levelW*levelH)
	for y := 0; y < levelH; y++ {
		for x := 0; x < levelW; x++ {
			levelPix[y*levelW+x] = colToInt(img.At(x, y))
		}
	}

	// Find top border
	top := 0
	for {
		sum := 0
		for x := 0; x < levelW; x++ {
			sum += levelPix[top*levelW+x]
		}
		if sum != 0 {
			break
		}
		top++
	}

	// Find left border
	left := 0
	for {
		sum := 0
		for y := 0; y < levelH; y++ {
			sum += levelPix[y*levelW+left]
		}
		if sum != 0 {
			break
		}
		left++
	}

	// Finally, we can read the tiles!
	pf := playfield{}
	for pfY := 0; pfY < playfieldH; pfY++ {
		for pfX := 0; pfX < playfieldW; pfX++ {
			tileFound := -1
			for t := 0; tileFound < 0 && t < nofTiles; t++ {
				tileMatch := true
				for y2 := 2; tileMatch && y2 < tileH-2; y2++ { // 2 pix border, we might have the cursor in
					for x2 := 2; tileMatch && x2 < tileW-2; x2++ {
						if tilesPix[y2*tileLineW+t*tileW+x2] != levelPix[(top+pfY*tileH+y2)*levelW+left+pfX*tileW+x2] {
							tileMatch = false
						}
					}
				}
				if tileMatch {
					tileFound = t
				}
			}
			if tileFound < 0 {
				tileFound = int(tileBg)
			}
			pf.tiles[pfY][pfX] = tile(tileFound)
		}
	}

	return &pf

}

// ================================================
// == DEQUE
// ==

type deque_elem struct {
	next *deque_elem
	val  *playfield
}

type deque struct {
	head *deque_elem
	tail *deque_elem
}

func (d *deque) empty() bool {
	return d.head == nil
}

func (d *deque) pop() *playfield {
	res := d.head.val
	d.head = d.head.next
	if d.head == nil {
		d.tail = nil
	}
	return res
}

func (d *deque) push(pf *playfield) {
	elem := &deque_elem{val: pf}
	if d.head == nil {
		// first elem
		d.head = elem
		d.tail = elem
	} else {
		d.tail.next = elem
		d.tail = elem
	}
}

func (d *deque) dump() {
	fmt.Print("Deque dump begin:\n")
	cur := d.head
	i := 0
	for cur != nil {
		fmt.Printf("Elem %3d: %v\n", i, cur)
		i++
		cur = cur.next
	}
	fmt.Print("Deque dump end\n")
}

// ================================================
// == GRAPHICS HELPERS
// ==

var (
	//go:embed tiles.png
	tilesData []byte

	//go:embed font.png
	fontData []byte

	fontTexture  *sdl.Texture
	tilesTexture *sdl.Texture
)

func loadTexture(r *sdl.Renderer, png []byte) *sdl.Texture {
	data, _ := sdl.RWFromMem(png)
	surfaceImg, err := img.LoadRW(data, true)
	if err != nil {
		panic(err)
	}
	textureImg, err := r.CreateTextureFromSurface(surfaceImg)
	if err != nil {
		panic(err)
	}
	surfaceImg.Free()
	return textureImg
}

func loadImages(r *sdl.Renderer) {
	tilesTexture = loadTexture(r, tilesData)
	fontTexture = loadTexture(r, fontData)
}

func renderMove(m move, r *sdl.Renderer) {
	r.SetDrawColor(0, 255, 55, 255)
	y := m.fromY*zoom*tileW + zoom*tileW/2
	x := m.fromX*zoom*tileH + zoom*tileH/2
	r.FillRect(&sdl.Rect{X: int32(x - zoom*tileH/4), Y: int32(y - zoom*tileW/4), W: int32(zoom * tileW / 2), H: int32(zoom * tileH / 2)})

	y = m.fromY*zoom*tileW + zoom*tileW/2
	x = m.toX*zoom*tileH + zoom*tileH/2
	r.FillRect(&sdl.Rect{X: int32(x - zoom*tileH/4), Y: int32(y - zoom*tileW/4), W: int32(zoom * tileW / 2), H: int32(zoom * tileH / 2)})
}

func text(x, y int, s string, r *sdl.Renderer) {
	zoom := 2
	for _, c := range s {
		cy := (c / 32) * 16
		cx := (c % 32) * 9
		srcRect := &sdl.Rect{X: int32(cx), Y: int32(cy), W: 9, H: 16}
		dstRect := &sdl.Rect{X: int32(x), Y: int32(y), W: int32(9 * zoom), H: int32(16 * zoom)}
		r.Copy(fontTexture, srcRect, dstRect)
		x += 9 * zoom
	}
}

// ================================================
// == MAIN
// ==

func main() {
	flag.Parse()

	initTileMap()

	var startPf *playfield
	if len(*flagScreenshot) > 0 {
		startPf = playfieldFromScreenshot(*flagScreenshot)
	} else {
		startPf = playfieldFromString(*flagLevelData)
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Pupu64 Solver", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(playfieldW*tileW*zoom), int32(playfieldH*tileH*zoom), sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", err)
		os.Exit(3)
	}
	defer renderer.Destroy()
	renderer.Clear()

	loadImages(renderer)

	seen := make(map[tiles]bool)
	playfields := deque{}

	playfields.push(startPf)

	var solution *playfield

	pfCnt := 0
	for solution == nil && !playfields.empty() {

		pf := playfields.pop()

		pfCnt++

		moves := pf.possibleMoves()
		for _, m := range moves {
			pf2 := pf.apply(m)
			if _, found := seen[pf2.tiles]; found {
				// already processed or in queue
				continue
			}

			seen[pf2.tiles] = true

			if !pf2.isSolvable() {
				// not solvable, ignore
				continue
			}

			if pf2.isSolved() {
				// WOOHOO!!!!!
				solution = pf2
			}

			playfields.push(pf2)
		}
	}
	fmt.Printf("%d playfields analyzed.\n", pfCnt)

	solved := solution != nil
	if solution == nil {
		fmt.Printf("No solution found. WTF???\n")
		solution = startPf
	} else {
		fmt.Printf("Solution found:\n")
		for idx, m := range solution.path {
			fmt.Printf("Step %d: (%d,%d)->(%d,%d)\n", idx+1, m.fromX, m.fromY, m.toX, m.fromY)
		}
	}

	moves := solution.path
	steps := []*playfield{startPf}
	cur := startPf
	// cur.dump()
	// fmt.Println()
	for _, m := range moves {
		cur = cur.apply(m)
		// cur.dump()
		// fmt.Println()
		steps = append(steps, cur)
	}

	idx := 0
	running := true
	window.SetTitle(fmt.Sprintf("Pupu64 Solver: Use Crsr-Left and Crsr-Right, Q to quit"))
	for running {
		// Handle all the events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch ev := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				if ev.Type == sdl.KEYDOWN {
					switch ev.Keysym.Sym {
					case 'q':
						running = false
					case sdl.K_RIGHT:
						if idx < len(moves) {
							idx++
						}
					case sdl.K_LEFT:
						if idx > 0 {
							idx--
						}
					}
				}
			}
		}

		steps[idx].render(renderer)
		if idx < len(moves) {
			m := moves[idx]
			renderMove(moves[idx], renderer)
			text(0, 0, fmt.Sprintf("Step %d of %d: Move (%d,%d) to (%d,%d)", idx+1, len(steps), m.fromX, m.fromY, m.toX, m.fromY), renderer)
		} else if solved {
			text(0, 0, fmt.Sprintf("Step %d of %d: SOLVED!", idx+1, len(steps)), renderer)
		} else {
			text(0, 0, "NO SOLUTION FOUND!", renderer)
		}
		renderer.Present()
	}
}
