package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// All art is either hand-drawn as strings ('0'-'3' = darkest..lightest color,
// '.' = transparent) or generated per pixel from a pattern. Everything uses
// the sprite's own four-color palette from theme.go.

type mood int

const (
	moodIdle mood = iota
	moodHappy
	moodOops
)

// ---- Five-Way Frank ----

var chefBase = []string{
	".....000000.....",
	"....03333330....",
	"...0333333330...",
	"...0333333330...",
	"....03333330....",
	"....00000000....", // hat band
	"....02222220....",
	"....02022020....", // eyes
	"....02222220....",
	"...0000000000...", // the mustache
	"....02222220....", // chin / mouth row
	"...0111111110...",
	"..011333333110..", // apron
	"..021333333120..", // hands
	"...0133333310...",
	"....00000000....",
}

var chefFaces = map[mood]map[int]string{
	moodHappy: {
		7:  "....00022000....", // squinting
		10: "....02200220....", // open grin
	},
	moodOops: {
		6:  "....02022020.1..", // tall eyes + sweat drop
		7:  "....02022020....",
		10: "....02000020....", // yikes
	},
}

func chefRGBA(m mood) *image.RGBA {
	rows := append([]string(nil), chefBase...)
	for r, s := range chefFaces[m] {
		rows[r] = s
	}
	return rowsRGBA(rows, spritePal("chef"))
}

// ---- the health inspector: fedora, shades, clipboard ----

var inspectorRows = []string{
	".....000000.....",
	"....01111110....",
	"..000000000000..", // hat brim
	"....02222220....",
	"....00000000....", // shades
	"....02222220....",
	"....02000020....", // flat, unimpressed mouth
	".....022220.....",
	"...0111111110...",
	"..011103330110..", // clipboard
	"..021103030120..",
	"...0110333011...",
	"...0111000111...",
	"....01100110....",
	"....01100110....",
	"...0000..0000...",
}

// ---- the chili pot ----

var potRows = []string{
	"..000000000000..",
	".00111111111100.",
	".01222222222210.", // chili inside
	"..011111111110..",
	"..011111111110..",
	"..011131111110..",
	"..011111111110..",
	"...0111111110...",
	"....00000000....",
}

func rowsRGBA(rows []string, pal [4]color.RGBA) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, len(rows[0]), len(rows)))
	for y, row := range rows {
		for x, ch := range row {
			if ch >= '0' && ch <= '3' {
				im.Set(x, y, pal[ch-'0'])
			}
		}
	}
	return im
}

// ---- food layers: 50x7 pixels, drawn at 2x onto the plate ----

const layerW, layerH = 50, 7

func layerRGBA(ing Ingredient) *image.RGBA {
	pal := spritePal(ingredientNames[ing])
	im := image.NewRGBA(image.Rect(0, 0, layerW, layerH))
	for y := 0; y < layerH; y++ {
		for x := 0; x < layerW; x++ {
			if i, ok := layerPixel(ing, x, y); ok {
				im.Set(x, y, pal[i])
			}
		}
	}
	outline(im, pal[0])
	return im
}

func layerPixel(ing Ingredient, x, y int) (int, bool) {
	n := hash(x, y, int(ing)) % 100
	corner := (x < 2 || x > layerW-3) && (y == 0 || y == layerH-1)
	switch ing {
	case Spaghetti: // wavy noodle strands
		if corner {
			return 0, false
		}
		wave := int(math.Round(1.2 * math.Sin(float64(x)/2.3+float64(y)*1.7)))
		if ((y+wave)%3+3)%3 == 0 {
			return 1, true
		}
		return 2, true
	case Chili: // dark, lumpy, with bits of meat
		if y == 0 && n < 45 {
			return 0, false
		}
		if n < 22 {
			return 1, true
		}
		return 0, true
	case Cheese: // a fluffy mound of shreds
		if y < 2 && (x < 5 || x > layerW-6 || n < 30) {
			return 0, false
		}
		if (x+y*2)%4 == 0 || n < 20 {
			return 3, true
		}
		return 2, true
	case Onion: // diced white chunks
		if corner || (y == 0 && n < 35) {
			return 0, false
		}
		if hash(x/2, y/2, 99)%100 < 45 {
			return 3, true
		}
		return 1, true
	case Beans: // rows of little ovals
		if corner {
			return 0, false
		}
		bx := (x + (y/3)*3) % 6
		if bx < 4 && y%3 < 2 {
			if bx == 1 && y%3 == 0 {
				return 2, true // shine
			}
			return 1, true
		}
		return 0, true
	case Bun: // rounded loaf
		if (x < 3 || x > layerW-4) && (y < 2 || y == layerH-1) {
			return 0, false
		}
		switch {
		case y == 1:
			return 3, true
		case y == layerH-1:
			return 1, true
		}
		return 2, true
	case Dog: // a shorter sausage with rounded ends
		if x < 3 || x > layerW-4 || y == 0 || y == layerH-1 {
			return 0, false
		}
		if (x < 5 || x > layerW-6) && (y == 1 || y == layerH-2) {
			return 0, false
		}
		if y == 2 {
			return 2, true
		}
		return 1, true
	case Mustard: // a squiggle
		yy := 3 + int(math.Round(2*math.Sin(float64(x)/1.6)))
		if y == yy || y == yy+1 {
			return 2, true
		}
		return 0, false
	}
	return 0, false
}

// outline darkens any opaque pixel that touches transparency or the edge,
// so light foods stay visible against the light background.
func outline(im *image.RGBA, c color.RGBA) {
	b := im.Bounds()
	opaque := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < b.Dx() && y < b.Dy() && im.RGBAAt(x, y).A > 0
	}
	var edge [][2]int
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if !opaque(x, y) {
				continue
			}
			if !opaque(x-1, y) || !opaque(x+1, y) || !opaque(x, y-1) || !opaque(x, y+1) {
				edge = append(edge, [2]int{x, y})
			}
		}
	}
	for _, p := range edge {
		im.Set(p[0], p[1], c)
	}
}

func hash(x, y, s int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(s)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

// ---- ebiten side ----

var spriteCache = map[string]*ebiten.Image{}

func sprite(key string, build func() *image.RGBA) *ebiten.Image {
	if img, ok := spriteCache[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(build())
	spriteCache[key] = img
	return img
}

func drawSprite(dst, img *ebiten.Image, x, y, scale float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	dst.DrawImage(img, op)
}

func drawChef(dst *ebiten.Image, m mood, x, y, scale float64) {
	keys := [...]string{"chef-idle", "chef-happy", "chef-oops"}
	drawSprite(dst, sprite(keys[m], func() *image.RGBA { return chefRGBA(m) }), x, y, scale)
}

func drawInspector(dst *ebiten.Image, x, y, scale float64) {
	drawSprite(dst, sprite("inspector", func() *image.RGBA { return rowsRGBA(inspectorRows, spritePal("inspector")) }), x, y, scale)
}

func drawPot(dst *ebiten.Image, x, y, scale float64) {
	drawSprite(dst, sprite("pot", func() *image.RGBA { return rowsRGBA(potRows, spritePal("pot")) }), x, y, scale)
}

func drawLayer(dst *ebiten.Image, ing Ingredient, x, y, scale float64) {
	drawSprite(dst, sprite("layer-"+ingredientNames[ing], func() *image.RGBA { return layerRGBA(ing) }), x, y, scale)
}

// chefIcon scales Frank up for the window and taskbar icon.
func chefIcon(size int) image.Image {
	src := chefRGBA(moodHappy)
	k := size / 16
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.Set(x, y, src.At(x/k, y/k))
		}
	}
	return dst
}
