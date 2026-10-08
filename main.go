package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// 320x288 is exactly double the Game Boy's 160x144, so text stays readable.
const (
	screenW  = 320
	screenH  = 288
	railMax  = 4
	maxPlate = 6
	rushAt   = 0.6 // fraction of the shift when the dinner rush kicks in
)

// Classic four-shade DMG palette, darkest to lightest.
var (
	shade0 = color.RGBA{0x0f, 0x38, 0x0f, 0xff}
	shade1 = color.RGBA{0x30, 0x62, 0x30, 0xff}
	shade2 = color.RGBA{0x8b, 0xac, 0x0f, 0xff}
	shade3 = color.RGBA{0x9b, 0xbc, 0x0f, 0xff}
	face   = text.NewGoXFace(basicfont.Face7x13)
)

type state int

const (
	stateMenu state = iota
	stateSeed
	stateHowTo
	stateOptions
	statePlaying
	stateOver
)

type Game struct {
	seed     uint64
	state    state
	tick     int // gameplay clock; only advances during a shift
	anim     int // always advances; drives menu animation
	snd      *Audio
	best     map[string]int
	settings Settings
	booted   bool
	quit     bool

	queue []Ticket // tickets not yet on the rail
	rail  []Ticket // rail[0] is the order you're building
	plate []Ingredient

	score, combo, served, walked, wrong int

	paused, rush, newBest bool
	lastSec               int

	Mess  // ladle, spills, puddles, inspector (mess.go)
	Menus // cursors and screen state for the menus (menu.go)

	msg       string
	msgTicks  int
	shake     int
	chefMood  mood
	chefTicks int
}

func (g *Game) start() {
	g.state = statePlaying
	g.tick = 0
	g.queue = buildRush(g.seed)
	g.rail = nil
	g.plate = nil
	g.score, g.combo, g.served, g.walked, g.wrong = 0, 1, 0, 0, 0
	g.paused, g.rush, g.newBest = false, false, false
	g.lastSec = 0
	g.msg, g.msgTicks, g.shake = "", 0, 0
	g.chefMood, g.chefTicks = moodIdle, 0
	g.Mess = Mess{}
	g.snd.startMusic(false)
}

func (g *Game) flash(s string) {
	g.msg = s
	g.msgTicks = tps
}

func (g *Game) setChef(m mood) {
	g.chefMood = m
	g.chefTicks = tps / 2
}

func (g *Game) Update() error {
	pollInput()
	g.anim++
	if !g.booted {
		g.booted = true
		g.toMenu()
	}
	if justPressed(ebiten.KeyF11) {
		g.settings.Fullscreen = !g.settings.Fullscreen
		g.applySettings()
	}
	if justPressed(ebiten.KeyM) {
		g.snd.toggleMute()
	}

	switch g.state {
	case stateMenu:
		g.updateMenu()
	case stateSeed:
		g.updateSeed()
	case stateHowTo:
		g.updateHowTo()
	case stateOptions:
		g.updateOptions()
	case statePlaying:
		g.updatePlaying()
	case stateOver:
		g.updateOver()
	}
	if g.quit {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) updatePlaying() {
	if g.paused {
		g.updatePause()
		return
	}
	if pausePressed() {
		g.paused, g.pauseSel = true, 0
		g.snd.pauseMusic(true)
		g.snd.play("select")
		return
	}
	if justPressed(ebiten.KeyH) {
		g.settings.Hints = !g.settings.Hints
		saveJSON("settings.json", g.settings)
	}

	g.tick++
	if g.msgTicks > 0 {
		g.msgTicks--
	}
	if g.shake > 0 {
		g.shake--
	}
	if g.chefTicks > 0 {
		if g.chefTicks--; g.chefTicks == 0 {
			g.chefMood = moodIdle
		}
	}

	if !g.rush && float64(g.tick)/shiftTicks >= rushAt {
		g.rush = true
		g.flash("DINNER RUSH!")
		g.snd.startMusic(true)
	}

	// Due tickets slide in from the right when there's room; otherwise they wait.
	for len(g.queue) > 0 && len(g.rail) < railMax && g.queue[0].SpawnTick <= g.tick {
		t := g.queue[0]
		t.DrawX = screenW
		g.rail = append(g.rail, t)
		g.queue = g.queue[1:]
		g.snd.play("ding")
	}

	// Patience drains; anyone who runs out walks.
	kept := g.rail[:0]
	for _, t := range g.rail {
		t.Patience--
		if t.Patience <= 0 {
			g.walked++
			g.score -= 50
			g.combo = 1
			g.flash("WALKOUT! -50")
			g.setChef(moodOops)
			g.snd.play("walkout")
			continue
		}
		kept = append(kept, t)
	}
	g.rail = kept
	if g.updateMess() {
		return
	}
	for i := range g.rail {
		target := float32(4 + i*79)
		g.rail[i].DrawX += (target - g.rail[i].DrawX) * 0.2
	}

	for i := range ingredientKeys {
		if Ingredient(i) == Chili {
			continue // chili is ladled; see mess.go
		}
		if ingredientPressed(i) && len(g.plate) < maxPlate {
			g.plate = append(g.plate, Ingredient(i))
			g.snd.play(fmt.Sprintf("add%d", i))
		}
	}
	if tossPressed() && len(g.plate) > 0 {
		g.plate = nil
		g.snd.play("toss")
	}
	if servePressed() {
		g.serve()
	}

	// Countdown ticks for the last ten seconds.
	left := (shiftTicks - g.tick + tps - 1) / tps
	if left <= 10 && left != g.lastSec {
		g.lastSec = left
		g.snd.play("tick")
	}

	if g.tick >= shiftTicks || (len(g.queue) == 0 && len(g.rail) == 0) {
		g.finish()
	}
}

func (g *Game) serve() {
	if len(g.plate) == 0 || len(g.rail) == 0 {
		return
	}
	front := g.rail[0]
	if plateMatches(g.plate, front.Recipe) {
		speedBonus := 50 * front.Patience / front.MaxPatience
		pts := 100*g.combo + speedBonus
		g.score += pts
		g.served++
		g.flash(fmt.Sprintf("ORDER UP! +%d", pts))
		g.snd.play(fmt.Sprintf("serve%d", g.combo))
		g.setChef(moodHappy)
		if g.combo < 8 {
			g.combo++
		}
		g.rail = g.rail[1:]
	} else {
		g.score -= 50
		g.wrong++
		g.combo = 1
		g.flash("WRONG PLATE -50")
		g.snd.play("wrong")
		g.setChef(moodOops)
		g.shake = 12
		g.splatter()
	}
	g.plate = nil
}

func (g *Game) finish() {
	g.state = stateOver
	g.overSel, g.overDelay = 0, tps // ignore input briefly so mashed keys don't skip it
	g.snd.stopMusic()
	if g.closed {
		g.snd.play("closed")
	} else {
		g.snd.play("over")
	}
	key := fmt.Sprint(g.seed)
	if g.score > g.best[key] {
		g.best[key] = g.score
		g.newBest = true
		saveJSON("scores.json", g.best)
	}
}

func (g *Game) applySettings() {
	g.snd.setVolumes(float64(g.settings.MusicVol)/10, float64(g.settings.SfxVol)/10)
	ebiten.SetFullscreen(g.settings.Fullscreen)
	applyTheme(g.settings.Palette)
	saveJSON("settings.json", g.settings)
}

// ---- drawing ----

func drawText(dst *ebiten.Image, s string, x, y float64, c color.Color) {
	drawTextScaled(dst, s, x, y, 1, c)
}

func drawTextScaled(dst *ebiten.Image, s string, x, y, scale float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

func drawCentered(dst *ebiten.Image, s string, y float64, c color.Color) {
	drawText(dst, s, float64(screenW-len(s)*7)/2, y, c)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(shade3)
	switch g.state {
	case stateMenu:
		g.drawMenu(screen)
	case stateSeed:
		g.drawSeed(screen)
	case stateHowTo:
		g.drawHowTo(screen)
	case stateOptions:
		g.drawOptions(screen)
	case statePlaying:
		g.drawPlaying(screen)
		if g.paused {
			g.drawPause(screen)
		}
	case stateOver:
		g.drawOver(screen)
	}
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	// Header
	vector.DrawFilledRect(screen, 0, 0, screenW, 18, shade0, false)
	left := (shiftTicks - g.tick + tps - 1) / tps
	timeColor := shade3
	if left <= 10 && (g.tick/15)%2 == 0 {
		timeColor = shade2
	}
	drawText(screen, fmt.Sprintf("SCORE %d", g.score), 4, 3, shade3)
	drawCentered(screen, fmt.Sprintf("x%d", g.combo), 3, shade2)
	drawText(screen, fmt.Sprintf("%d:%02d", left/60, left%60), screenW-36, 3, timeColor)

	// Ticket rail
	vector.DrawFilledRect(screen, 0, 21, screenW, 2, shade1, false)
	for i := len(g.rail); i < railMax; i++ {
		vector.StrokeRect(screen, float32(4+i*79), 26, 75, 44, 1, shade2, false)
	}
	for i, t := range g.rail {
		x, y := t.DrawX, float32(26)
		bg, fg, bar := shade2, shade0, shade1
		if i == 0 {
			bg, fg, bar = shade0, shade3, shade2
		}
		vector.DrawFilledRect(screen, x, y, 75, 44, bg, false)
		drawText(screen, t.Recipe.Name, float64(x)+6, float64(y)+6, fg)
		frac := float32(t.Patience) / float32(t.MaxPatience)
		vector.StrokeRect(screen, x+6, y+28, 63, 8, 1, fg, false)
		// Patience bar blinks when the customer is about to walk.
		if frac > 0.25 || (g.tick/8)%2 == 0 {
			vector.DrawFilledRect(screen, x+7, y+29, 61*frac, 6, bar, false)
		}
	}

	if g.settings.Hints && len(g.rail) > 0 {
		names := make([]string, len(g.rail[0].Recipe.Layers))
		for i, ing := range g.rail[0].Recipe.Layers {
			names[i] = ingredientNames[ing]
		}
		drawCentered(screen, "NEXT: "+strings.Join(names, " > "), 78, shade0)
	}
	if g.msgTicks > 0 {
		drawCentered(screen, g.msg, 98, shade1)
	}

	// Counter, Frank, and the plate (which shakes on a wrong order).
	vector.DrawFilledRect(screen, 0, 238, screenW, 3, shade1, false)
	period := 30
	if g.rush {
		period = 12 // Frank gets twitchy during the rush
	}
	chefY := 184.0 + float64((g.tick/period)%2)*3
	if g.chefMood == moodHappy {
		chefY = 178
	}
	drawChef(screen, g.chefMood, 26, chefY, 3)

	dx := float32(0)
	if g.shake > 0 {
		dx = float32(math.Sin(float64(g.shake)*2) * 4)
	}
	vector.DrawFilledRect(screen, 100+dx, 232, 120, 6, shade0, false)
	for i, ing := range g.plate {
		y := float64(232 - (i+1)*16)
		drawLayer(screen, ing, 110+float64(dx), y+1, 2)
		drawText(screen, ingredientNames[ing], 216, y, shade1)
	}
	g.drawCounterMess(screen)

	// Controls legend follows whichever device you touched last.
	if usingPad {
		drawCentered(screen, "^ SPG  RT LADLE  > CHS  v ONI", 244, shade0)
		drawCentered(screen, "< BEN  X BUN  Y DOG  B MUS", 258, shade0)
		drawCentered(screen, "A serve LB toss RB mop START pause", 272, shade1)
	} else {
		drawCentered(screen, "1 SPG  2 LADLE  3 CHS  4 ONI", 244, shade0)
		drawCentered(screen, "5 BEN  6 BUN  7 DOG  8 MUS", 258, shade0)
		drawCentered(screen, "ENTER serve BKSP toss C mop P pause", 272, shade1)
	}
	g.drawSplatter(screen)
}

func (g *Game) Layout(_, _ int) (int, int) { return screenW, screenH }

func newSeed() uint64 { return uint64(time.Now().UnixNano()%99999) + 1 }

func main() {
	seed := flag.Uint64("seed", 0, "ticket seed (share it to race the same rush)")
	flag.Parse()
	g := &Game{seed: *seed % 100000, snd: newAudio(), best: loadBest(), settings: loadSettings()}
	if g.seed == 0 {
		g.seed = newSeed()
	}
	g.applySettings()

	ebiten.SetWindowSize(screenW*2, screenH*2)
	ebiten.SetWindowTitle("Chili Parlor")
	ebiten.SetWindowIcon([]image.Image{chefIcon(16), chefIcon(32), chefIcon(48)})
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
