package main

import "image/color"

// A Theme is four UI shades (darkest to lightest, like the original Game Boy)
// plus, Game Boy Color style, an optional four-color palette per sprite.
// Sprites without their own palette fall back to the UI shades.

type Theme struct {
	Name    string
	UI      [4]color.RGBA
	Spill   [2]color.RGBA // chili on the counter: dark, shine
	Sprites map[string][4]color.RGBA
}

func hex(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}

func pal4(a, b, c, d uint32) [4]color.RGBA { return [4]color.RGBA{hex(a), hex(b), hex(c), hex(d)} }

var themes = []Theme{
	{
		Name:  "DINER",
		UI:    pal4(0x2a1b14, 0xb5361f, 0xf0b43c, 0xfbf1dc), // espresso, chili red, mustard, cream
		Spill: [2]color.RGBA{hex(0x7a2416), hex(0xc0582f)},
		Sprites: map[string][4]color.RGBA{
			"chef":      pal4(0x2a1b14, 0x3d6fb6, 0xf2c39b, 0xffffff), // outline, blue shirt, skin, whites
			"inspector": pal4(0x1f1f24, 0x6b6f7a, 0xe9b996, 0xffffff), // gray suit, clipboard paper
			"pot":       pal4(0x1e1e22, 0x8a8f99, 0x8c2f1c, 0xe6e8ec), // steel pot, chili inside
			"SPG":       pal4(0x6b4a1e, 0xd9a441, 0xf6d77a, 0xfff2c0),
			"CHL":       pal4(0x7a2416, 0xc0582f, 0xd9733f, 0xe89a62),
			"CHS":       pal4(0xb5651d, 0xe08a1e, 0xf5a623, 0xffd55a), // bright orange cheddar
			"ONI":       pal4(0x7d6a8c, 0xc9b8d6, 0xe8dff0, 0xffffff),
			"BEN":       pal4(0x3b130e, 0x8e2b22, 0xd9735e, 0xf0a090),
			"BUN":       pal4(0x7a4a1f, 0xc98a3f, 0xe8b469, 0xf7dca6),
			"DOG":       pal4(0x5a1a14, 0xc0452f, 0xe8826a, 0xf4b0a0),
			"MUS":       pal4(0xa0780a, 0xd4a20c, 0xffd400, 0xfff07a),
		},
	},
	{
		Name:  "GAME BOY",
		UI:    pal4(0x0f380f, 0x306230, 0x8bac0f, 0x9bbc0f),
		Spill: [2]color.RGBA{hex(0x0f380f), hex(0x8bac0f)},
	},
	{
		Name:  "POCKET",
		UI:    pal4(0x1a1a1a, 0x5a5a5a, 0xa8a8a8, 0xe8e8e8),
		Spill: [2]color.RGBA{hex(0x1a1a1a), hex(0xa8a8a8)},
	},
}

var (
	theme      = themes[0]
	spillDark  color.RGBA
	spillLight color.RGBA
)

// applyTheme swaps every color in the game and rebuilds the sprites.
func applyTheme(i int) {
	theme = themes[((i%len(themes))+len(themes))%len(themes)]
	shade0, shade1, shade2, shade3 = theme.UI[0], theme.UI[1], theme.UI[2], theme.UI[3]
	spillDark, spillLight = theme.Spill[0], theme.Spill[1]
	for k, img := range spriteCache {
		img.Deallocate()
		delete(spriteCache, k)
	}
}

func spritePal(key string) [4]color.RGBA {
	if p, ok := theme.Sprites[key]; ok {
		return p
	}
	return theme.UI
}
