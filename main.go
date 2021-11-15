package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

const (
	screenWidth          = 1280
	screenHeight         = 720
	pageSize             = 8
	menuItems            = 8
	dropdownArrowOffsetX = 192
	dropdownBarOffsetX   = 32
	dropdownOffsetY      = 64
	selectedTileOffsetY  = 128
	saveOffsetX          = 32
	saveOffsetY          = 448
)

type Game struct {
	CursorDrag *CursorDrag
	Library    map[string]Tile // The set of available tiles
	TileNames  []string        // The names of all tiles for sorting
	Tiles      []*Tile         // The tiles places on the grid
	Menu       Menu            // The menu with all options
	Grid       Grid            // The grid where tiles are placed
	Font       font.Face       // The font the menu is rendered with
	Save       bool
}

type Menu struct {
	X             int
	Y             int
	Page          int
	Image         *ebiten.Image
	Header        *ebiten.Image
	DropdownArrow *ebiten.Image
	DropdownBar   *ebiten.Image
	SaveButton    *ebiten.Image
	SelectedTile  string
	IsOpen        bool
}

func (m *Menu) GetX() int {
	return m.X
}

func (m *Menu) GetY() int {
	return m.Y
}

func (m *Menu) SetX(x int) {
	m.X = x
}

func (m *Menu) SetY(y int) {
	m.Y = y
}

func (m *Menu) GetImage() *ebiten.Image {
	return m.Image
}

type Grid struct {
	X     int
	Y     int
	Image *ebiten.Image
}

type Tile struct {
	X     int
	Y     int
	Image *ebiten.Image
}

func (t *Tile) GetX() int {
	return t.X
}

func (t *Tile) GetY() int {
	return t.Y
}

func (t *Tile) SetX(x int) {
	t.X = x
}

func (t *Tile) SetY(y int) {
	t.Y = y
}

func (t *Tile) GetImage() *ebiten.Image {
	return t.Image
}

type CursorDrag struct {
	Origin    image.Point
	Target    CursorTarget
	IsNewTile bool
}

type CursorTarget interface {
	GetX() int
	GetY() int
	SetX(int)
	SetY(int)
	GetImage() *ebiten.Image
}

func IsPointInRect(x, y int, r image.Rectangle) bool {
	return r.Min.X < x && x < r.Max.X && r.Min.Y < y && y < r.Max.Y
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustReleased(ebiten.KeyPageUp) {
		if g.Menu.Page > 0 {
			g.Menu.Page--
		}
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyPageDown) {
		g.Menu.Page++
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		w, h := g.Menu.SaveButton.Size()
		r := image.Rect(g.Menu.X+saveOffsetX, g.Menu.Y+saveOffsetY, g.Menu.X+w+saveOffsetX, g.Menu.Y+h+saveOffsetY)
		if IsPointInRect(x, y, r) {
			g.Save = true
			return nil
		}

		w, h = g.Menu.DropdownBar.Size()
		r = image.Rect(g.Menu.X+dropdownBarOffsetX, g.Menu.Y+dropdownOffsetY, g.Menu.X+w+dropdownBarOffsetX, g.Menu.Y+h+dropdownOffsetY)
		if IsPointInRect(x, y, r) {
			// If the drowdown is clicked, open or close it
			g.Menu.IsOpen = !g.Menu.IsOpen
			return nil
		}
		// If menu is open, check for a click on a dropdown option
		if g.Menu.IsOpen {
			offset := 0
			for i := g.Menu.Page * pageSize; i < g.Menu.Page*pageSize+pageSize && i < len(g.TileNames); i++ {
				offset += 32
				r = image.Rect(g.Menu.X+dropdownBarOffsetX, g.Menu.Y+dropdownOffsetY+offset, g.Menu.X+w+dropdownBarOffsetX, g.Menu.Y+h+dropdownOffsetY+offset)
				if IsPointInRect(x, y, r) {
					g.Menu.SelectedTile = g.TileNames[i]
					break
				}
			}
		}

		// Whether an option was clicked or not, close the dropdown if the user clicked anywhere
		g.Menu.IsOpen = false

		// If the user clicked on the menu header, set up a cursor drag
		w, h = g.Menu.Header.Size()
		r = image.Rect(g.Menu.X, g.Menu.Y, g.Menu.X+w, g.Menu.Y+h)
		if IsPointInRect(x, y, r) {
			drag := &CursorDrag{
				Origin: image.Point{X: x, Y: y},
			}
			drag.Target = &g.Menu
			g.CursorDrag = drag
			return nil
		}

		// If the user is not dragging the menu, check if they are dragging a new tile
		w, h = g.Library[g.Menu.SelectedTile].Image.Size()
		r = image.Rect(g.Menu.X+dropdownBarOffsetX, g.Menu.Y+selectedTileOffsetY, g.Menu.X+dropdownBarOffsetX+w, g.Menu.Y+selectedTileOffsetY+h)
		if IsPointInRect(x, y, r) {
			drag := &CursorDrag{
				Origin:    image.Point{X: x, Y: y},
				IsNewTile: true,
			}

			snapX := x % 32
			snapY := y % 32

			tile := Tile{
				X:     x - snapX,
				Y:     y - snapY,
				Image: g.Library[g.Menu.SelectedTile].Image,
			}

			drag.Target = &tile
			g.CursorDrag = drag
			return nil
		}

		// If the user is not dragging the menu, check if they are dragging an existing tile
		for i := 0; i < len(g.Tiles); i++ {
			tile := g.Tiles[i]
			w, h = tile.Image.Size()
			r = image.Rect(tile.X, tile.Y, tile.X+w, tile.Y+h)
			if IsPointInRect(x, y, r) {
				drag := &CursorDrag{
					Origin: image.Point{X: x, Y: y},
				}
				drag.Target = tile
				g.CursorDrag = drag
				return nil
			}
		}
	} else if g.CursorDrag != nil && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		xDiff := x - g.CursorDrag.Origin.X
		if xDiff < 0 {
			xDiff += 32
		}

		yDiff := y - g.CursorDrag.Origin.Y
		if yDiff < 0 {
			yDiff += 32
		}

		snapX := xDiff % 32
		snapY := yDiff % 32

		g.CursorDrag.Target.SetX(g.CursorDrag.Target.GetX() + x - g.CursorDrag.Origin.X - snapX)
		g.CursorDrag.Target.SetY(g.CursorDrag.Target.GetY() + y - g.CursorDrag.Origin.Y - snapY)

		if g.CursorDrag.IsNewTile {
			w, h := g.Menu.Image.Size()
			r := image.Rect(g.Menu.X, g.Menu.Y, g.Menu.X+w, g.Menu.Y+h)
			// If a new tile is in the menu area, discard it on click release
			if !IsPointInRect(x, y, r) {
				g.Tiles = append(g.Tiles, &Tile{
					X:     g.CursorDrag.Target.GetX(),
					Y:     g.CursorDrag.Target.GetY(),
					Image: g.CursorDrag.Target.GetImage(),
				})
			}
		}
		g.CursorDrag = nil
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	for x := 0; x < g.Grid.X*32; x += 32 {
		for y := 0; y < g.Grid.Y*32; y += 32 {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(g.Grid.Image, op)
		}
	}

	for _, tile := range g.Tiles {
		if g.CursorDrag == nil || tile != g.CursorDrag.Target {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(tile.X), float64(tile.Y))
			screen.DrawImage(tile.Image, op)
		}
	}

	if g.Save {
		g.Save = false
		buf := new(bytes.Buffer)
		png.Encode(buf, screen)

		f, err := os.Create("./screen.png")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		_, err = io.Copy(f, buf)
		if err != nil {
			log.Fatal(err)
		}
	}

	if g.CursorDrag != nil {
		x, y := ebiten.CursorPosition()
		dx := x - g.CursorDrag.Origin.X
		dy := y - g.CursorDrag.Origin.Y
		if g.CursorDrag.Target != nil {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(g.CursorDrag.Target.GetX()+dx), float64(g.CursorDrag.Target.GetY()+dy))
			op.ColorM.Scale(1, 1, 1, 0.5)
			screen.DrawImage(g.CursorDrag.Target.GetImage(), op)
		}
	}

	op.GeoM.Reset()
	op.GeoM.Translate(float64(g.Menu.X), float64(g.Menu.Y))
	screen.DrawImage(g.Menu.Image, op)

	op.GeoM.Reset()
	op.GeoM.Translate(float64(g.Menu.X), float64(g.Menu.Y))
	screen.DrawImage(g.Menu.Header, op)

	op.GeoM.Reset()
	op.GeoM.Translate(float64(g.Menu.X+dropdownBarOffsetX), float64(g.Menu.Y+selectedTileOffsetY))
	screen.DrawImage(g.Library[g.Menu.SelectedTile].Image, op)

	op.GeoM.Reset()
	op.GeoM.Translate(float64(g.Menu.X)+saveOffsetX, float64(g.Menu.Y+saveOffsetY))
	screen.DrawImage(g.Menu.SaveButton, op)

	op.GeoM.Reset()
	op.GeoM.Translate(float64(g.Menu.X)+dropdownBarOffsetX, float64(g.Menu.Y+dropdownOffsetY))
	screen.DrawImage(g.Menu.DropdownBar, op)

	op.GeoM.Reset()
	op.GeoM.Translate(float64(g.Menu.X)+dropdownArrowOffsetX, float64(g.Menu.Y+dropdownOffsetY))
	screen.DrawImage(g.Menu.DropdownArrow, op)

	if g.Menu.IsOpen {
		y := 0
		for i := g.Menu.Page * pageSize; i < g.Menu.Page*pageSize+pageSize && i < len(g.TileNames); i++ {
			y += 32
			op.GeoM.Reset()
			op.GeoM.Translate(float64(g.Menu.X)+32, float64(g.Menu.Y+y+dropdownOffsetY))
			screen.DrawImage(g.Menu.DropdownBar, op)
			text.Draw(screen, g.TileNames[i], g.Font, g.Menu.X+32, g.Menu.Y+y+dropdownOffsetY+32, color.Black)
		}
	}

	text.Draw(screen, "Menu", g.Font, g.Menu.X+8, g.Menu.Y+23, color.Black)
	text.Draw(screen, g.Menu.SelectedTile, g.Font, g.Menu.X+32, g.Menu.Y+dropdownOffsetY+32, color.Black)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Dungeon Tiles Mapper")

	dir, err := os.Open("./tiles")
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()

	files, _ := dir.Readdir(0)
	library := make(map[string]Tile, 848)
	for _, f := range files {
		image, _, err := ebitenutil.NewImageFromFile("./tiles/" + f.Name())
		if err != nil {
			log.Fatal(err)
		}

		library[f.Name()] = Tile{
			Image: image,
		}
	}

	grid, _, err := ebitenutil.NewImageFromFile("./grid.png")
	if err != nil {
		log.Fatal(err)
	}

	menu, _, err := ebitenutil.NewImageFromFile("./menu.png")
	if err != nil {
		log.Fatal(err)
	}

	header, _, err := ebitenutil.NewImageFromFile("./menu_header.png")
	if err != nil {
		log.Fatal(err)
	}

	dropdownArrow, _, err := ebitenutil.NewImageFromFile("./dropdown_arrow.png")
	if err != nil {
		log.Fatal(err)
	}

	dropdownBar, _, err := ebitenutil.NewImageFromFile("./dropdown_bar.png")
	if err != nil {
		log.Fatal(err)
	}

	save, _, err := ebitenutil.NewImageFromFile("./save_icon.png")
	if err != nil {
		log.Fatal(err)
	}

	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    20,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	tileNames := make([]string, 0, len(library))
	for k := range library {
		tileNames = append(tileNames, k)
	}
	sort.Strings(tileNames)

	g := &Game{
		Library:   library,
		Tiles:     []*Tile{},
		TileNames: tileNames,
		Grid: Grid{
			X:     39,
			Y:     22,
			Image: grid,
		},
		Menu: Menu{
			X:             512,
			Y:             0,
			Image:         menu,
			Header:        header,
			DropdownArrow: dropdownArrow,
			DropdownBar:   dropdownBar,
			SaveButton:    save,
			SelectedTile:  "DT1_4x8_Ruins.b.0.jpg",
		},
		Font: face,
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
