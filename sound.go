package main

import (
	"bytes"
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Every sound is synthesized at startup from square waves and LFSR noise,
// the same building blocks as the Game Boy's sound chip. No audio files.

const sampleRate = 44100

type Audio struct {
	ctx       *audio.Context
	sfx       map[string][]byte
	active    []*audio.Player
	music     *audio.Player
	rushMusic *audio.Player
	current   *audio.Player
	muted     bool
	musicVol  float64 // 0-1
	sfxVol    float64 // 0-1
}

func newAudio() *Audio {
	a := &Audio{ctx: audio.NewContext(sampleRate), sfx: map[string][]byte{}, musicVol: 0.7, sfxVol: 0.8}

	// One blip per ingredient key, climbing a pentatonic scale.
	for i, n := range []int{72, 74, 76, 79, 81, 84, 86, 88} {
		a.sfx[fmt.Sprintf("add%d", i)] = pcm(square(midiHz(n), 0.04, 0.5, 0.25, 0.9))
	}
	// Order-up arpeggio, a semitone higher for each combo level.
	for c := 1; c <= 8; c++ {
		var s []float64
		for _, n := range []int{72, 76, 79, 84} {
			s = append(s, square(midiHz(n+c-1), 0.055, 0.25, 0.25, 0.4)...)
		}
		a.sfx[fmt.Sprintf("serve%d", c)] = pcm(s)
	}
	a.sfx["wrong"] = pcm(mix(seq(square(110, 0.12, 0.5, 0.25, 0.2), square(92.5, 0.2, 0.5, 0.25, 0.8)), noise(0.1, 0.12)))
	a.sfx["walkout"] = pcm(seq(note(67, 0.09, 0.3), note(64, 0.09, 0.3), note(60, 0.18, 0.9)))
	a.sfx["ding"] = pcm(seq(square(midiHz(91), 0.05, 0.125, 0.18, 0.5), square(midiHz(96), 0.14, 0.125, 0.18, 1)))
	a.sfx["toss"] = pcm(noise(0.12, 0.2))
	a.sfx["tick"] = pcm(noise(0.025, 0.2))
	a.sfx["over"] = pcm(seq(note(72, 0.12, 0.3), note(67, 0.12, 0.3), note(64, 0.12, 0.3), note(60, 0.4, 1)))

	// Chili and mess sounds.
	for i := 0; i < 10; i++ { // ladle filling, rising in pitch
		a.sfx[fmt.Sprintf("fill%d", i)] = pcm(square(midiHz(55+i*2), 0.03, 0.25, 0.1, 0.5))
	}
	a.sfx["plop"] = pcm(mix(seq(note(48, 0.04, 0.2), note(43, 0.08, 1)), noise(0.05, 0.08)))
	a.sfx["slop"] = pcm(mix(note(41, 0.12, 0.8), noise(0.12, 0.12)))
	a.sfx["skimpy"] = pcm(seq(square(midiHz(84), 0.05, 0.125, 0.15, 0.3), square(midiHz(79), 0.08, 0.125, 0.15, 1)))
	a.sfx["splat"] = pcm(mix(noise(0.3, 0.3), square(70, 0.25, 0.5, 0.2, 1)))
	a.sfx["mop"] = pcm(noise(0.08, 0.1))
	var siren []float64
	for i := 0; i < 4; i++ {
		siren = append(siren, square(880, 0.12, 0.25, 0.15, 0.2)...)
		siren = append(siren, square(660, 0.12, 0.25, 0.15, 0.2)...)
	}
	a.sfx["siren"] = pcm(siren)
	a.sfx["pass"] = pcm(seq(note(72, 0.07, 0.3), note(76, 0.07, 0.3), note(79, 0.14, 1)))
	// Menu sounds.
	a.sfx["move"] = pcm(square(midiHz(84), 0.025, 0.25, 0.12, 0.5))
	a.sfx["select"] = pcm(seq(square(midiHz(79), 0.04, 0.25, 0.15, 0.3), square(midiHz(91), 0.07, 0.25, 0.15, 1)))
	a.sfx["back"] = pcm(seq(square(midiHz(79), 0.04, 0.25, 0.15, 0.3), square(midiHz(72), 0.07, 0.25, 0.15, 1)))
	a.sfx["closed"] = pcm(seq(note(64, 0.2, 0.2), note(63, 0.2, 0.2), note(62, 0.2, 0.2), note(61, 0.6, 1)))

	a.music = a.loop(buildMusic(140))
	a.rushMusic = a.loop(buildMusic(175))
	return a
}

func (a *Audio) loop(b []byte) *audio.Player {
	p, err := a.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b))))
	if err != nil {
		log.Fatal(err)
	}
	return p
}

func (a *Audio) play(name string) {
	if a == nil {
		return // no audio device
	}
	if a.muted {
		return
	}
	keep := a.active[:0]
	for _, p := range a.active {
		if p.IsPlaying() {
			keep = append(keep, p)
		} else {
			p.Close()
		}
	}
	p := a.ctx.NewPlayerFromBytes(a.sfx[name])
	p.SetVolume(a.sfxVol)
	p.Play()
	a.active = append(keep, p)
}

func (a *Audio) startMusic(rush bool) {
	if a == nil {
		return // no audio device
	}
	a.stopMusic()
	a.current = a.music
	if rush {
		a.current = a.rushMusic
	}
	a.current.SetPosition(0)
	a.current.Play()
}

func (a *Audio) stopMusic() {
	if a == nil {
		return // no audio device
	}
	a.music.Pause()
	a.rushMusic.Pause()
	a.current = nil
}

func (a *Audio) pauseMusic(paused bool) {
	if a == nil {
		return // no audio device
	}
	if a.current == nil {
		return
	}
	if paused {
		a.current.Pause()
	} else {
		a.current.Play()
	}
}

func (a *Audio) toggleMute() {
	if a == nil {
		return
	}
	a.muted = !a.muted
	a.applyVolumes()
}

func (a *Audio) setVolumes(music, sfx float64) {
	if a == nil {
		return
	}
	a.musicVol, a.sfxVol = music, sfx
	a.applyVolumes()
}

func (a *Audio) applyVolumes() {
	vol := a.musicVol * 0.6
	if a.muted {
		vol = 0
	}
	a.music.SetVolume(vol)
	a.rushMusic.SetVolume(vol)
}

func (a *Audio) musicPlaying() bool { return a != nil && a.current != nil }

// buildMusic renders an 8-bar diner loop: pulse melody, pulse bass, noise hi-hats.
func buildMusic(bpm float64) []byte {
	eighth := 60 / bpm / 2
	melody := [][]int{ // 0 = rest
		{72, 76, 79, 76, 72, 76, 79, 81},
		{77, 81, 84, 81, 77, 76, 74, 72},
		{74, 79, 83, 79, 74, 77, 76, 74},
		{76, 74, 72, 67, 72, 0, 72, 0},
	}
	bass := []int{48, 41, 43, 48} // C F G C
	var mel, bs, hats []float64
	for pass := 0; pass < 2; pass++ {
		for bar := range melody {
			for i, n := range melody[bar] {
				if n == 0 {
					mel = append(mel, silence(eighth)...)
				} else {
					mel = append(mel, square(midiHz(n), eighth, 0.25, 0.09, 0.5)...)
				}
				root := bass[bar]
				if i%2 == 1 {
					root += 12
				}
				bs = append(bs, square(midiHz(root), eighth, 0.5, 0.07, 0.6)...)
				if i%2 == 1 {
					hats = append(hats, seq(noise(0.03, 0.05), silence(eighth-0.03))...)
				} else {
					hats = append(hats, silence(eighth)...)
				}
			}
		}
	}
	return pcm(mix(mix(mel, bs), hats))
}

// ---- synth helpers ----

func midiHz(n int) float64 { return 440 * math.Pow(2, float64(n-69)/12) }

func note(n int, secs, fade float64) []float64 { return square(midiHz(n), secs, 0.5, 0.22, fade) }

// square renders a pulse wave. duty is the high fraction (0.125, 0.25, 0.5 like the GB),
// fade is how far it decays by the end (0 = flat, 1 = to silence).
func square(hz, secs, duty, vol, fade float64) []float64 {
	n := int(secs * sampleRate)
	out := make([]float64, n)
	for i := range out {
		v := vol
		if math.Mod(float64(i)*hz/sampleRate, 1) >= duty {
			v = -vol
		}
		out[i] = v * (1 - fade*float64(i)/float64(n))
	}
	return out
}

// noise uses a 15-bit LFSR, the same trick as the GB's noise channel.
func noise(secs, vol float64) []float64 {
	n := int(secs * sampleRate)
	out := make([]float64, n)
	lfsr := uint16(0x7fff)
	for i := range out {
		if i%4 == 0 {
			bit := (lfsr ^ (lfsr >> 1)) & 1
			lfsr = (lfsr >> 1) | (bit << 14)
		}
		v := vol
		if lfsr&1 == 1 {
			v = -vol
		}
		out[i] = v * (1 - float64(i)/float64(n))
	}
	return out
}

func silence(secs float64) []float64 {
	if secs <= 0 {
		return nil
	}
	return make([]float64, int(secs*sampleRate))
}

func seq(parts ...[]float64) []float64 {
	var out []float64
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func mix(a, b []float64) []float64 {
	if len(b) > len(a) {
		a, b = b, a
	}
	out := append([]float64(nil), a...)
	for i := range b {
		out[i] += b[i]
	}
	return out
}

// pcm converts mono floats to the 16-bit little-endian stereo Ebiten expects.
func pcm(s []float64) []byte {
	out := make([]byte, len(s)*4)
	for i, v := range s {
		v = math.Max(-1, math.Min(1, v))
		x := int16(v * 32767)
		out[i*4], out[i*4+1] = byte(x), byte(x>>8)
		out[i*4+2], out[i*4+3] = byte(x), byte(x>>8)
	}
	return out
}
