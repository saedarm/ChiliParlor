package main

import (
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Chili is ladled, not tapped. Hold 2 (or RT) and the ladle meter rises:
//   below ladleGoodLo  -> SKIMPY, nothing added
//   up to ladleGoodHi  -> clean scoop
//   up to the top      -> SLOPPY, scoop added but -10 and some drips
//   hit the top        -> SPILL: -25 and a puddle on the counter
// Every puddle makes the ladle fill faster, so messes snowball.
// Three puddles at once brings the health inspector. Mop (C) before he
// reaches the counter or he shuts the parlor down.

const (
	ladleSecs      = 0.9 // time to fill the ladle with a clean counter
	ladleGoodLo    = 0.6
	ladleGoodHi    = 0.9
	puddleSpeedup  = 0.4 // each puddle adds 40% ladle speed
	puddleWet      = 4   // mop presses per puddle
	maxPuddles     = 5
	inspectorLimit = 3
	inspectorSecs  = 4.0 // time from the door to the counter
	inspectorStopX = 40.0
)

type puddle struct {
	X, Grow float32
	Wet     int
}

type drop struct{ X, Y, VX, VY float32 }

type blob struct{ X, Y, R float32 }

type Mess struct {
	ladling, ladleLocked bool
	ladle                float64

	puddles []puddle
	drops   []drop

	splats     []blob
	splatTicks int

	inspecting bool
	inspX      float32
	inspDir    float32
	closed     bool

	spills int
}

// updateMess runs the ladle, puddles, drips, splatter and inspector.
// It returns true if the inspector just shut the shift down.
func (g *Game) updateMess() bool {
	if mopPressed() {
		g.mop()
	}
	g.updateLadle()

	for i := range g.puddles {
		if g.puddles[i].Grow < 1 {
			g.puddles[i].Grow += 0.08
		}
	}
	kept := g.drops[:0]
	for _, d := range g.drops {
		d.VY += 0.25
		d.X += d.VX
		d.Y += d.VY
		if d.Y < 236 {
			kept = append(kept, d)
		}
	}
	g.drops = kept

	if g.splatTicks > 0 {
		g.splatTicks--
	}

	if g.inspecting {
		g.inspX += g.inspDir * (inspectorStopX + 48) / (inspectorSecs * tps)
		if g.inspDir > 0 && g.inspX >= inspectorStopX {
			g.closed = true
			g.finish()
			return true
		}
		if g.inspDir < 0 && g.inspX <= -48 {
			g.inspecting = false
		}
	}
	return false
}

func (g *Game) updateLadle() {
	if !chiliHeld() {
		if g.ladling {
			g.releaseLadle()
		}
		g.ladling, g.ladleLocked = false, false
		return
	}
	if g.ladleLocked {
		return // still holding after a spill; let go first
	}
	if !g.ladling {
		if len(g.plate) >= maxPlate {
			g.ladleLocked = true
			return
		}
		g.ladling, g.ladle = true, 0
	}
	g.ladle += (1 + puddleSpeedup*float64(len(g.puddles))) / (ladleSecs * tps)
	if g.tick%6 == 0 {
		g.snd.play(fmt.Sprintf("fill%d", min(9, int(g.ladle*10))))
	}
	if g.ladle >= 1 {
		g.spill()
		g.ladling, g.ladleLocked = false, true
	}
}

func (g *Game) releaseLadle() {
	switch {
	case g.ladle < ladleGoodLo:
		g.flash("SKIMPY! LADLE MORE")
		g.snd.play("skimpy")
	case g.ladle <= ladleGoodHi:
		g.plate = append(g.plate, Chili)
		g.snd.play("plop")
	default:
		g.plate = append(g.plate, Chili)
		g.score -= 10
		g.flash("SLOPPY -10")
		g.snd.play("slop")
		g.addDrops(4)
	}
}

func (g *Game) spill() {
	g.spills++
	g.score -= 25
	g.flash("SPILL! -25  C TO MOP")
	g.snd.play("splat")
	g.setChef(moodOops)
	g.addDrops(14)
	if len(g.puddles) < maxPuddles {
		// Land beside the plate, not under it, so the puddle is easy to see.
		x := 72 + rand.Float32()*20
		if rand.IntN(2) == 0 {
			x = 238 + rand.Float32()*30
		}
		g.puddles = append(g.puddles, puddle{X: x, Wet: puddleWet})
	}
	if len(g.puddles) >= inspectorLimit {
		g.callInspector()
	}
}

func (g *Game) addDrops(n int) {
	top := float32(232 - len(g.plate)*16)
	for i := 0; i < n; i++ {
		g.drops = append(g.drops, drop{
			X: 140 + rand.Float32()*40, Y: top,
			VX: rand.Float32()*3 - 1.5, VY: -rand.Float32() * 3,
		})
	}
}

func (g *Game) mop() {
	if len(g.puddles) == 0 {
		return
	}
	p := &g.puddles[len(g.puddles)-1]
	p.Wet--
	g.snd.play("mop")
	if p.Wet <= 0 {
		g.puddles = g.puddles[:len(g.puddles)-1]
		g.flash("CLEAN!")
	}
	if g.inspecting && g.inspDir > 0 && len(g.puddles) < inspectorLimit {
		g.inspDir = -1
		g.flash("INSPECTOR: FINE. FOR NOW.")
		g.snd.play("pass")
	}
}

func (g *Game) callInspector() {
	if !g.inspecting {
		g.inspecting = true
		g.inspX = -48
	}
	g.inspDir = 1
	g.flash("HEALTH INSPECTOR! MOP!")
	g.snd.play("siren")
}

// splatter covers part of the key legend after a wrong plate.
func (g *Game) splatter() {
	g.splats = g.splats[:0]
	for i := 0; i < 7; i++ {
		g.splats = append(g.splats, blob{
			X: 20 + rand.Float32()*280, Y: 248 + rand.Float32()*36, R: 5 + rand.Float32()*9,
		})
	}
	g.splatTicks = 3 * tps
}

// ---- drawing ----

func (g *Game) drawCounterMess(dst *ebiten.Image) {
	for _, p := range g.puddles {
		w := 40 * p.Grow * float32(p.Wet) / puddleWet
		vector.DrawFilledRect(dst, p.X-w/2, 236, w, 5, spillDark, false)
		vector.DrawFilledRect(dst, p.X-w/2+3, 233, max(0, w-6), 3, spillDark, false)
		vector.DrawFilledRect(dst, p.X-w/2+6, 231, max(0, w-12), 2, spillDark, false)
		if w > 12 {
			vector.DrawFilledRect(dst, p.X-w/4, 233, 4, 1, spillLight, false) // shine
		}
	}
	for _, d := range g.drops {
		vector.DrawFilledRect(dst, d.X, d.Y, 2, 2, spillDark, false)
	}

	// Ladle meter over the pot.
	const x, y, w, h = 292, 120, 14, 84
	vector.DrawFilledRect(dst, x, y, w, h, shade3, false)
	vector.DrawFilledRect(dst, x, y+h*(1-ladleGoodHi), w, h*(ladleGoodHi-ladleGoodLo), shade2, false)
	vector.DrawFilledRect(dst, x, y, w, h*(1-ladleGoodHi), shade1, false)
	vector.StrokeRect(dst, x, y, w, h, 1, shade0, false)
	if g.ladling {
		fh := float32(g.ladle) * h
		vector.DrawFilledRect(dst, x+4, y+h-fh, w-8, fh, spillDark, false)
	}
	drawText(dst, "LADLE", 280, 104, shade0)
	drawPot(dst, 283, 210, 2)
	steam := spritePal("pot")[1]
	if (g.tick/10)%2 == 0 { // steam
		vector.DrawFilledRect(dst, 292, 204, 2, 2, steam, false)
		vector.DrawFilledRect(dst, 304, 200, 2, 2, steam, false)
	} else {
		vector.DrawFilledRect(dst, 294, 200, 2, 2, steam, false)
		vector.DrawFilledRect(dst, 302, 204, 2, 2, steam, false)
	}

	if g.inspecting {
		bob := float64((g.tick / 8) % 2)
		drawInspector(dst, float64(g.inspX), 124+bob*2, 3)
	}
	if len(g.puddles) > 0 && (g.tick/20)%2 == 0 {
		drawCentered(dst, fmt.Sprintf("MESS x%d: C TO MOP", len(g.puddles)), 114, shade0)
	}
}

func (g *Game) drawSplatter(dst *ebiten.Image) {
	if g.splatTicks == 0 {
		return
	}
	s := min(1, float32(g.splatTicks)/tps) // shrink during the last second
	for _, b := range g.splats {
		vector.DrawFilledCircle(dst, b.X, b.Y, b.R*s, spillDark, false)
		vector.DrawFilledCircle(dst, b.X+b.R*s, b.Y-b.R*s*0.5, b.R*s*0.4, spillDark, false)
	}
}
