package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Menus struct {
	menuSel, overSel, pauseSel, optSel, howPage int
	overDelay                                   int
	seedDigits                                  [5]int
	seedCursor                                  int
}

var (
	mainItems  = []string{"START SHIFT", "ENTER SEED", "HOW TO PLAY", "OPTIONS", "QUIT"}
	pauseItems = []string{"RESUME", "RESTART SHIFT", "QUIT TO MENU"}
	overItems  = []string{"REPLAY THIS SEED", "NEW SEED", "MAIN MENU"}
	optItems   = []string{"MUSIC", "SOUND FX", "HINTS", "FULLSCREEN", "COLORS", "BACK"}
)

// menuNav moves a cursor up and down a list and reports a confirm.
func (g *Game) menuNav(sel *int, n int) bool {
	if navUp() {
		*sel = (*sel + n - 1) % n
		g.snd.play("move")
	}
	if navDown() {
		*sel = (*sel + 1) % n
		g.snd.play("move")
	}
	if confirmPressed() {
		g.snd.play("select")
		return true
	}
	return false
}

func (g *Game) toMenu() {
	g.state, g.menuSel, g.paused = stateMenu, 0, false
	g.snd.startMusic(false)
}

func (g *Game) backToMenu() {
	g.snd.play("back")
	g.state = stateMenu
}

// ---- main menu ----

func (g *Game) updateMenu() {
	if !g.menuNav(&g.menuSel, len(mainItems)) {
		return
	}
	switch g.menuSel {
	case 0:
		g.start()
	case 1:
		g.openSeed()
	case 2:
		g.state, g.howPage = stateHowTo, 0
	case 3:
		g.state, g.optSel = stateOptions, 0
	case 4:
		g.quit = true
	}
}

func (g *Game) drawMenu(dst *ebiten.Image) {
	title := "CHILI PARLOR"
	drawTextScaled(dst, title, float64(screenW-len(title)*14)/2, 12, 2, shade0)
	drawCentered(dst, "a Cincinnati short-order rush", 44, shade1)

	bob := float64((g.anim / 30) % 2)
	drawChef(dst, moodIdle, 108, 64+bob*3, 3)
	// A 3-way on the counter next to him, steaming.
	vector.DrawFilledRect(dst, 168, 110, 56, 3, shade0, false)
	for i, ing := range recipes[0].Layers {
		drawLayer(dst, ing, 171, float64(103-i*7), 1)
	}
	sx := float32(186 + (g.anim/12)%2*4)
	vector.DrawFilledRect(dst, sx, 80, 2, 2, shade1, false)
	vector.DrawFilledRect(dst, sx+12, 76, 2, 2, shade1, false)

	drawCentered(dst, "with FIVE-WAY FRANK", 120, shade0)
	drawCentered(dst, fmt.Sprintf("SEED %05d   BEST %d", g.seed, g.best[fmt.Sprint(g.seed)]), 138, shade1)
	drawItems(dst, mainItems, g.menuSel, 162)
	if usingPad {
		drawCentered(dst, "D-PAD MOVE   A SELECT", 268, shade1)
	} else {
		drawCentered(dst, "ARROWS MOVE   ENTER SELECT", 268, shade1)
	}
}

func drawItems(dst *ebiten.Image, items []string, sel int, y0 float64) {
	for i, it := range items {
		y := y0 + float64(i)*18
		if i == sel {
			vector.DrawFilledRect(dst, 80, float32(y)-1, 160, 15, shade0, false)
			drawCentered(dst, it, y, shade3)
		} else {
			drawCentered(dst, it, y, shade0)
		}
	}
}

// ---- seed entry ----

var digitKeys = [10][2]ebiten.Key{
	{ebiten.KeyDigit0, ebiten.KeyNumpad0}, {ebiten.KeyDigit1, ebiten.KeyNumpad1},
	{ebiten.KeyDigit2, ebiten.KeyNumpad2}, {ebiten.KeyDigit3, ebiten.KeyNumpad3},
	{ebiten.KeyDigit4, ebiten.KeyNumpad4}, {ebiten.KeyDigit5, ebiten.KeyNumpad5},
	{ebiten.KeyDigit6, ebiten.KeyNumpad6}, {ebiten.KeyDigit7, ebiten.KeyNumpad7},
	{ebiten.KeyDigit8, ebiten.KeyNumpad8}, {ebiten.KeyDigit9, ebiten.KeyNumpad9},
}

func (g *Game) openSeed() {
	g.state, g.seedCursor = stateSeed, 4
	g.setSeedDigits(g.seed)
}

func (g *Game) setSeedDigits(n uint64) {
	for i := 4; i >= 0; i-- {
		g.seedDigits[i] = int(n % 10)
		n /= 10
	}
}

func (g *Game) seedValue() uint64 {
	var n uint64
	for _, d := range g.seedDigits {
		n = n*10 + uint64(d)
	}
	return n
}

func (g *Game) updateSeed() {
	d := &g.seedDigits
	for k, keys := range digitKeys {
		if justPressed(keys[0], keys[1]) { // typed digits shift in from the right
			copy(d[:4], d[1:])
			d[4], g.seedCursor = k, 4
			g.snd.play("move")
		}
	}
	if justPressed(ebiten.KeyBackspace) {
		copy(d[1:], d[:4])
		d[0] = 0
		g.snd.play("move")
	}
	if navLeft() && g.seedCursor > 0 {
		g.seedCursor--
		g.snd.play("move")
	}
	if navRight() && g.seedCursor < 4 {
		g.seedCursor++
		g.snd.play("move")
	}
	if navUp() {
		d[g.seedCursor] = (d[g.seedCursor] + 1) % 10
		g.snd.play("move")
	}
	if navDown() {
		d[g.seedCursor] = (d[g.seedCursor] + 9) % 10
		g.snd.play("move")
	}
	if justPressed(ebiten.KeyR) || padJust(ebiten.StandardGamepadButtonRightTop) {
		g.setSeedDigits(newSeed())
		g.snd.play("move")
	}
	if confirmPressed() {
		n := g.seedValue()
		if n == 0 {
			n = newSeed()
		}
		g.seed = n
		g.snd.play("select")
		g.state = stateMenu
	}
	if justPressed(ebiten.KeyEscape) || padJust(ebiten.StandardGamepadButtonRightRight) {
		g.backToMenu()
	}
}

func (g *Game) drawSeed(dst *ebiten.Image) {
	drawCentered(dst, "ENTER A SEED", 28, shade0)
	drawCentered(dst, "Same seed = same tickets, same timing.", 48, shade1)
	drawCentered(dst, "Share it to race a friend.", 62, shade1)
	for i, digit := range g.seedDigits {
		x, y := float32(77+i*34), float32(84)
		fg := color.Color(shade0)
		if i == g.seedCursor {
			vector.DrawFilledRect(dst, x, y, 30, 44, shade0, false)
			fg = shade3
		} else {
			vector.StrokeRect(dst, x, y, 30, 44, 2, shade0, false)
		}
		drawTextScaled(dst, fmt.Sprint(digit), float64(x)+4.5, float64(y)+3, 3, fg)
	}
	if usingPad {
		drawCentered(dst, "D-PAD CHANGES DIGITS   Y RANDOM", 146, shade0)
		drawCentered(dst, "A OK   B CANCEL", 162, shade0)
	} else {
		drawCentered(dst, "TYPE DIGITS, OR USE THE ARROWS", 146, shade0)
		drawCentered(dst, "R RANDOM   BKSP DELETE", 162, shade0)
		drawCentered(dst, "ENTER OK   ESC CANCEL", 178, shade0)
	}
	drawCentered(dst, "00000 = RANDOM SEED", 204, shade1)
	drawCentered(dst, fmt.Sprintf("BEST ON THIS SEED %d", g.best[fmt.Sprint(g.seedValue())]), 224, shade1)
}

// ---- how to play ----

const howPages = 3

func (g *Game) updateHowTo() {
	conf := confirmPressed()
	if navRight() || conf {
		if g.howPage < howPages-1 {
			g.howPage++
			g.snd.play("move")
		} else if conf {
			g.backToMenu()
			return
		}
	}
	if navLeft() && g.howPage > 0 {
		g.howPage--
		g.snd.play("move")
	}
	if backPressed() {
		g.backToMenu()
	}
}

var controlRows = [][3]string{
	{"", "KEYBOARD", "CONTROLLER"},
	{"SPAGHETTI", "1", "D-PAD UP"},
	{"LADLE CHILI", "hold 2", "hold RT"},
	{"CHEESE", "3", "D-PAD RIGHT"},
	{"ONION", "4", "D-PAD DOWN"},
	{"BEANS", "5", "D-PAD LEFT"},
	{"BUN", "6", "X"},
	{"HOT DOG", "7", "Y"},
	{"MUSTARD", "8", "B"},
	{"SERVE", "ENTER/SPACE", "A"},
	{"TOSS PLATE", "BACKSPACE/X", "LB"},
	{"MOP", "C", "RB"},
	{"PAUSE", "P/ESC", "START"},
	{"HINTS / MUTE", "H / M", ""},
	{"FULLSCREEN", "F11", ""},
}

func (g *Game) drawHowTo(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, 0, 0, screenW, 18, shade0, false)
	drawText(dst, "HOW TO PLAY", 4, 3, shade3)
	drawText(dst, fmt.Sprintf("%d/%d", g.howPage+1, howPages), screenW-28, 3, shade3)
	lines := func(y float64, ls ...string) {
		for i, l := range ls {
			drawText(dst, l, 12, y+float64(i)*14, shade0)
		}
	}

	switch g.howPage {
	case 0:
		drawCentered(dst, "THE ORDERS", 26, shade1)
		ls := []string{
			"Tickets slide onto the rail up top.",
			"Build the dark ticket on the left,",
			"bottom layer first, then SERVE it.",
			"",
		}
		for _, r := range recipes {
			names := make([]string, len(r.Layers))
			for i, ing := range r.Layers {
				names[i] = ingredientNames[ing]
			}
			ls = append(ls, fmt.Sprintf("%-6s %s", r.Name, strings.Join(names, " ")))
		}
		ls = append(ls, "",
			"Clean plates raise your combo, up to x8.",
			"Wrong plate or walkout: -50, combo resets.")
		lines(46, ls...)
		vector.DrawFilledRect(dst, 248, 147, 56, 3, shade0, false)
		for i, ing := range recipes[0].Layers {
			drawLayer(dst, ing, 251, float64(140-i*7), 1)
		}
	case 1:
		drawCentered(dst, "THE CHILI", 26, shade1)
		lines(46,
			"Chili gets LADLED, not tapped.",
			"Hold the ladle and watch the meter.",
			"",
			"Let go too soon .. SKIMPY, no chili",
			"Light band ....... clean scoop",
			"Dark band ........ SLOPPY, -10",
			"Top of the meter . SPILL, -25",
			"",
			"Spills leave puddles on the counter.",
			"Each puddle makes the ladle fill faster.",
			"MOP to clean them up.",
			"",
			"3 puddles brings the HEALTH INSPECTOR.",
			"Mop before he reaches you or he",
			"shuts the parlor down.")
		// A little copy of the ladle meter next to the zone list.
		const x, y, w, h = 292, 88, 12, 56
		vector.DrawFilledRect(dst, x, y+h*(1-ladleGoodHi), w, h*(ladleGoodHi-ladleGoodLo), shade2, false)
		vector.DrawFilledRect(dst, x, y, w, h*(1-ladleGoodHi), shade1, false)
		vector.StrokeRect(dst, x, y, w, h, 1, shade0, false)
	case 2:
		drawCentered(dst, "CONTROLS", 26, shade1)
		for i, r := range controlRows {
			y := 44 + float64(i)*14
			c := color.Color(shade0)
			if i == 0 {
				c = shade1
			}
			drawText(dst, r[0], 12, y, c)
			drawText(dst, r[1], 106, y, c)
			drawText(dst, r[2], 204, y, c)
		}
	}

	if usingPad {
		drawCentered(dst, "D-PAD PAGE   A NEXT   B BACK", 268, shade1)
	} else {
		drawCentered(dst, "ARROWS PAGE   ENTER NEXT   ESC BACK", 268, shade1)
	}
}

// ---- options ----

func (g *Game) updateOptions() {
	n := len(optItems)
	if navUp() {
		g.optSel = (g.optSel + n - 1) % n
		g.snd.play("move")
	}
	if navDown() {
		g.optSel = (g.optSel + 1) % n
		g.snd.play("move")
	}
	dir := 0
	if navLeft() {
		dir = -1
	}
	if navRight() {
		dir = 1
	}
	conf := confirmPressed()
	s := &g.settings
	changed := false
	switch g.optSel {
	case 0:
		if dir != 0 {
			s.MusicVol = max(0, min(10, s.MusicVol+dir))
			changed = true
		}
	case 1:
		if dir != 0 {
			s.SfxVol = max(0, min(10, s.SfxVol+dir))
			changed = true
		}
	case 2:
		if dir != 0 || conf {
			s.Hints = !s.Hints
			changed = true
		}
	case 3:
		if dir != 0 || conf {
			s.Fullscreen = !s.Fullscreen
			changed = true
		}
	case 4:
		if dir != 0 || conf {
			if dir == 0 {
				dir = 1
			}
			s.Palette = ((s.Palette+dir)%len(themes) + len(themes)) % len(themes)
			changed = true
		}
	case 5:
		if conf {
			g.backToMenu()
			return
		}
	}
	if changed {
		g.applySettings() // also saves
		g.snd.play("add3")
	}
	if backPressed() {
		g.backToMenu()
	}
}

func (g *Game) drawOptions(dst *ebiten.Image) {
	drawCentered(dst, "OPTIONS", 30, shade0)
	onOff := func(b bool) string {
		if b {
			return "< ON >"
		}
		return "< OFF >"
	}
	for i, it := range optItems {
		y := float32(70 + i*26)
		fg := color.Color(shade0)
		if i == g.optSel {
			vector.DrawFilledRect(dst, 40, y-2, 240, 17, shade0, false)
			fg = shade3
		}
		if i == len(optItems)-1 {
			drawCentered(dst, it, float64(y), fg)
			continue
		}
		drawText(dst, it, 52, float64(y), fg)
		switch i {
		case 0, 1:
			vol := g.settings.MusicVol
			if i == 1 {
				vol = g.settings.SfxVol
			}
			for j := 0; j < 10; j++ {
				bx := float32(160 + j*11)
				if j < vol {
					vector.DrawFilledRect(dst, bx, y+1, 8, 11, fg, false)
				} else {
					vector.StrokeRect(dst, bx, y+1, 8, 11, 1, fg, false)
				}
			}
		case 2:
			drawText(dst, onOff(g.settings.Hints), 160, float64(y), fg)
		case 3:
			drawText(dst, onOff(g.settings.Fullscreen), 160, float64(y), fg)
		case 4:
			drawText(dst, "< "+theme.Name+" >", 160, float64(y), fg)
		}
	}
	if usingPad {
		drawCentered(dst, "D-PAD CHANGES   B BACK", 240, shade0)
	} else {
		drawCentered(dst, "ARROWS CHANGE   ESC BACK", 240, shade0)
	}
	drawCentered(dst, "Saved automatically.", 256, shade1)
}

// ---- pause ----

func (g *Game) updatePause() {
	if pausePressed() || backPressed() {
		g.resume()
		return
	}
	if !g.menuNav(&g.pauseSel, len(pauseItems)) {
		return
	}
	switch g.pauseSel {
	case 0:
		g.resume()
	case 1:
		g.start()
	case 2:
		g.toMenu()
	}
}

func (g *Game) resume() {
	g.paused = false
	g.snd.pauseMusic(false)
}

func (g *Game) drawPause(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, 60, 96, 200, 96, shade0, false)
	vector.StrokeRect(dst, 63, 99, 194, 90, 1, shade2, false)
	drawCentered(dst, "PAUSED", 106, shade3)
	for i, it := range pauseItems {
		y := 130 + float64(i)*18
		if i == g.pauseSel {
			vector.DrawFilledRect(dst, 80, float32(y)-1, 160, 15, shade3, false)
			drawCentered(dst, it, y, shade0)
		} else {
			drawCentered(dst, it, y, shade2)
		}
	}
}

// ---- game over ----

func (g *Game) updateOver() {
	if g.overDelay > 0 {
		g.overDelay--
		return
	}
	if justPressed(ebiten.KeyR) {
		g.start()
		return
	}
	if justPressed(ebiten.KeyN) {
		g.seed = newSeed()
		g.start()
		return
	}
	if !g.menuNav(&g.overSel, len(overItems)) {
		return
	}
	switch g.overSel {
	case 0:
		g.start()
	case 1:
		g.seed = newSeed()
		g.start()
	case 2:
		g.toMenu()
	}
}

func (g *Game) drawOver(dst *ebiten.Image) {
	m := moodOops
	if g.newBest {
		m = moodHappy
	}
	if g.closed {
		drawCentered(dst, "SHUT DOWN!", 20, shade0)
		drawChef(dst, m, 104, 38, 3)
		drawInspector(dst, 168, 38, 3)
		drawCentered(dst, "CLOSED BY THE HEALTH INSPECTOR", 90, shade1)
	} else {
		drawCentered(dst, "SHIFT OVER", 20, shade0)
		drawChef(dst, m, float64(screenW-48)/2, 38, 3)
	}
	drawCentered(dst, fmt.Sprintf("SCORE %d", g.score), 108, shade0)
	if g.newBest && (g.anim/20)%2 == 0 {
		drawCentered(dst, "NEW BEST!", 124, shade1)
	} else if !g.newBest {
		drawCentered(dst, fmt.Sprintf("BEST %d", g.best[fmt.Sprint(g.seed)]), 124, shade1)
	}
	drawCentered(dst, fmt.Sprintf("SERVED %d   WALKOUTS %d", g.served, g.walked), 146, shade1)
	drawCentered(dst, fmt.Sprintf("WRONG %d   SPILLS %d", g.wrong, g.spills), 162, shade1)
	drawCentered(dst, fmt.Sprintf("SEED %05d", g.seed), 186, shade0)
	drawItems(dst, overItems, g.overSel, 214)
}
