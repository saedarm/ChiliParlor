# Chili Parlor

A short-order arcade game set in a Cincinnati chili parlor. Tickets slide in across the top: 3-ways, 4-ways, 5-ways, cheese coneys and double-deckers. You build each order in the right stacked order before the customer walks out. Ladle the chili carefully, because spills make a mess, and three puddles brings the health inspector.

Written in Go with [Ebiten](https://ebitengine.org). Every sprite and sound is generated in code, so there are no asset files.

![Main menu, a cheese coney in progress, and the health inspector walking in](screenshot.png)

## Play

Download `ChiliParlor.exe` from [Releases](../../releases) and run it. Windows 10 or 11, 64-bit, nothing else to install.

The .exe isn't code-signed, so Windows may show "Windows protected your PC" the first time. Click **More info → Run anyway**.

## How it works

- Build the dark ticket on the left, bottom layer first, then serve it.
- Clean plates raise your combo, up to ×8. A wrong plate or a walkout costs 50 points and resets it.
- Chili is ladled, not tapped. Hold the ladle key and let go in the middle band of the meter for a clean scoop. Let go in the top band and it's sloppy (−10). Hold it to the top and it spills (−25) and leaves a puddle.
- Each puddle makes the ladle fill faster. Mop them up. At three puddles the health inspector walks in, and if you don't mop below three before he reaches the counter, he shuts you down.
- The shift lasts 90 seconds, and the dinner rush starts 54 seconds in.

**Race a friend:** pick **Enter Seed** on the main menu and both type the same five digits. Same seed means the same tickets at the same times. Best scores are saved per seed.

## Controls

| Action | Keyboard | Controller |
| --- | --- | --- |
| Spaghetti | 1 | D-pad up |
| Ladle chili | hold 2 | hold RT |
| Cheese | 3 | D-pad right |
| Onion | 4 | D-pad down |
| Beans | 5 | D-pad left |
| Bun | 6 | X |
| Hot dog | 7 | Y |
| Mustard | 8 | B |
| Serve | Enter / Space | A |
| Toss plate | Backspace / X | LB |
| Mop | C | RB |
| Pause | P / Esc | Start |
| Hints, mute | H, M | |
| Fullscreen | F11 | |

## Menu

| Recipe | Layers, bottom to top |
| --- | --- |
| 3-way | spaghetti, chili, cheese |
| 4-way onion | spaghetti, chili, onion, cheese |
| 4-way bean | spaghetti, chili, beans, cheese |
| 5-way | spaghetti, chili, beans, onion, cheese |
| Cheese coney | bun, hot dog, mustard, chili, onion, cheese |
| Double-decker | spaghetti, chili, spaghetti, chili, cheese |

## Build from source

Needs Go 1.26 or newer. No C compiler is needed on Windows.

```powershell
go run .                      # play
go run . -seed 4242           # play a specific seed
go build -ldflags "-H windowsgui -s -w" -o ChiliParlor.exe .   # release build
```

`-H windowsgui` stops a console window opening behind the game, and `-s -w` strips debug info. The Frank icon comes from `rsrc_windows_amd64.syso`, which Go links into Windows builds automatically.

## Files

| File | What's in it |
| --- | --- |
| `main.go` | Game state, the shift loop, serving, scoring, the play screen |
| `ticket.go` | Ingredients, recipes, the seeded ticket queue |
| `mess.go` | Ladle, spills, puddles, mopping, the health inspector |
| `menu.go` | Main menu, seed entry, How to Play, Options, pause and game-over |
| `input.go` | Keyboard and controller mappings |
| `sound.go` | Sound effects and music, synthesized from square waves and noise |
| `sprites.go` | Frank, the inspector, the pot, the food layers |
| `theme.go` | Color palettes: Diner, Game Boy, Pocket |
| `save.go` | Settings and best scores |

Difficulty is set by the constants at the top of `ticket.go`, `main.go` and `mess.go`. Settings and scores are saved in `%AppData%\ChiliParlor`.
