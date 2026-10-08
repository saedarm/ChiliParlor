package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings and best scores live in %AppData%\ChiliParlor on Windows.

type Settings struct {
	MusicVol   int  `json:"music_volume"` // 0-10
	SfxVol     int  `json:"sfx_volume"`   // 0-10
	Hints      bool `json:"hints"`
	Fullscreen bool `json:"fullscreen"`
	Palette    int  `json:"palette"` // index into themes; 0 = DINER
}

func savePath(name string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "chiliparlor_" + name
	}
	return filepath.Join(dir, "ChiliParlor", name)
}

func loadJSON(name string, v any) {
	if b, err := os.ReadFile(savePath(name)); err == nil {
		json.Unmarshal(b, v)
	}
}

func saveJSON(name string, v any) {
	p := savePath(name)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if b, err := json.MarshalIndent(v, "", "  "); err == nil {
		os.WriteFile(p, b, 0o644)
	}
}

func loadSettings() Settings {
	s := Settings{MusicVol: 7, SfxVol: 8, Hints: true}
	loadJSON("settings.json", &s)
	s.MusicVol = max(0, min(10, s.MusicVol))
	s.SfxVol = max(0, min(10, s.SfxVol))
	s.Palette = max(0, min(len(themes)-1, s.Palette))
	return s
}

func loadBest() map[string]int {
	m := map[string]int{}
	loadJSON("scores.json", &m)
	return m
}
