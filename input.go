package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Keyboard and controller share one set of actions. Controllers use Ebiten's
// "standard" layout, so Xbox, PlayStation and most USB pads map the same way.
//
//	D-pad up/right/down/left  spaghetti / cheese / onion / beans
//	X / Y / B                 bun / dog / mustard
//	hold RT                   ladle chili
//	A serve   LB toss   RB mop   START pause

type padBtn = ebiten.StandardGamepadButton

var ingredientPad = [numIngredients]padBtn{
	ebiten.StandardGamepadButtonLeftTop,          // spaghetti
	ebiten.StandardGamepadButtonFrontBottomRight, // chili (held, see chiliHeld)
	ebiten.StandardGamepadButtonLeftRight,        // cheese
	ebiten.StandardGamepadButtonLeftBottom,       // onion
	ebiten.StandardGamepadButtonLeftLeft,         // beans
	ebiten.StandardGamepadButtonRightLeft,        // bun (X)
	ebiten.StandardGamepadButtonRightTop,         // dog (Y)
	ebiten.StandardGamepadButtonRightRight,       // mustard (B)
}

var ingredientKeys = [numIngredients][2]ebiten.Key{
	{ebiten.KeyDigit1, ebiten.KeyNumpad1},
	{ebiten.KeyDigit2, ebiten.KeyNumpad2},
	{ebiten.KeyDigit3, ebiten.KeyNumpad3},
	{ebiten.KeyDigit4, ebiten.KeyNumpad4},
	{ebiten.KeyDigit5, ebiten.KeyNumpad5},
	{ebiten.KeyDigit6, ebiten.KeyNumpad6},
	{ebiten.KeyDigit7, ebiten.KeyNumpad7},
	{ebiten.KeyDigit8, ebiten.KeyNumpad8},
}

var (
	padIDs   []ebiten.GamepadID
	keyBuf   []ebiten.Key
	usingPad bool // which control labels to show; follows the last device used
)

// pollInput runs once per frame before anything reads input.
func pollInput() {
	padIDs = ebiten.AppendGamepadIDs(padIDs[:0])
	for _, id := range padIDs {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		for b := padBtn(0); b <= ebiten.StandardGamepadButtonMax; b++ {
			if inpututil.IsStandardGamepadButtonJustPressed(id, b) {
				usingPad = true
			}
		}
	}
	keyBuf = inpututil.AppendJustPressedKeys(keyBuf[:0])
	if len(keyBuf) > 0 {
		usingPad = false
	}
}

func justPressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func padJust(b padBtn) bool {
	for _, id := range padIDs {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && inpututil.IsStandardGamepadButtonJustPressed(id, b) {
			return true
		}
	}
	return false
}

func padHeld(b padBtn) bool {
	for _, id := range padIDs {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && ebiten.IsStandardGamepadButtonPressed(id, b) {
			return true
		}
	}
	return false
}

// ---- menu actions ----

func navUp() bool {
	return justPressed(ebiten.KeyArrowUp, ebiten.KeyW) || padJust(ebiten.StandardGamepadButtonLeftTop)
}
func navDown() bool {
	return justPressed(ebiten.KeyArrowDown, ebiten.KeyS) || padJust(ebiten.StandardGamepadButtonLeftBottom)
}
func navLeft() bool {
	return justPressed(ebiten.KeyArrowLeft, ebiten.KeyA) || padJust(ebiten.StandardGamepadButtonLeftLeft)
}
func navRight() bool {
	return justPressed(ebiten.KeyArrowRight, ebiten.KeyD) || padJust(ebiten.StandardGamepadButtonLeftRight)
}
func confirmPressed() bool {
	return justPressed(ebiten.KeyEnter, ebiten.KeySpace) || padJust(ebiten.StandardGamepadButtonRightBottom)
}
func backPressed() bool {
	return justPressed(ebiten.KeyEscape, ebiten.KeyBackspace) || padJust(ebiten.StandardGamepadButtonRightRight)
}

// ---- gameplay actions ----

func pausePressed() bool {
	return justPressed(ebiten.KeyP, ebiten.KeyEscape) || padJust(ebiten.StandardGamepadButtonCenterRight)
}
func servePressed() bool {
	return justPressed(ebiten.KeyEnter, ebiten.KeySpace) || padJust(ebiten.StandardGamepadButtonRightBottom)
}
func tossPressed() bool {
	return justPressed(ebiten.KeyBackspace, ebiten.KeyX) || padJust(ebiten.StandardGamepadButtonFrontTopLeft)
}
func mopPressed() bool {
	return justPressed(ebiten.KeyC) || padJust(ebiten.StandardGamepadButtonFrontTopRight)
}
func ingredientPressed(i int) bool {
	return justPressed(ingredientKeys[i][0], ingredientKeys[i][1]) || padJust(ingredientPad[i])
}
func chiliHeld() bool {
	return ebiten.IsKeyPressed(ebiten.KeyDigit2) || ebiten.IsKeyPressed(ebiten.KeyNumpad2) ||
		padHeld(ebiten.StandardGamepadButtonFrontBottomRight)
}
