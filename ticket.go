package main

import "math/rand/v2"

type Ingredient int

const (
	Spaghetti Ingredient = iota
	Chili
	Cheese
	Onion
	Beans
	Bun
	Dog
	Mustard
	numIngredients
)

var ingredientNames = [numIngredients]string{"SPG", "CHL", "CHS", "ONI", "BEN", "BUN", "DOG", "MUS"}

// Recipe layers are listed bottom to top: the order you build them.
type Recipe struct {
	Name   string
	Layers []Ingredient
}

var recipes = []Recipe{
	{"3-WAY", []Ingredient{Spaghetti, Chili, Cheese}},
	{"4-ONI", []Ingredient{Spaghetti, Chili, Onion, Cheese}},
	{"4-BEN", []Ingredient{Spaghetti, Chili, Beans, Cheese}},
	{"5-WAY", []Ingredient{Spaghetti, Chili, Beans, Onion, Cheese}},
	{"CONEY", []Ingredient{Bun, Dog, Mustard, Chili, Onion, Cheese}},
	{"DBL-D", []Ingredient{Spaghetti, Chili, Spaghetti, Chili, Cheese}}, // double-decker
}

type Ticket struct {
	Recipe      *Recipe
	SpawnTick   int
	Patience    int // ticks left before the customer walks out
	MaxPatience int
	DrawX       float32 // on-screen x, eased toward its rail slot
}

const (
	tps        = 60 // Ebiten's default ticks per second
	shiftTicks = 90 * tps
)

// buildRush generates the whole shift's ticket queue up front from the seed,
// so two players with the same seed get the exact same orders at the same times.
func buildRush(seed uint64) []Ticket {
	rng := rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))
	var q []Ticket
	t := 2 * tps
	for t < shiftTicks-5*tps {
		progress := float64(t) / float64(shiftTicks)

		// Early in the shift, only the simple orders come in.
		r := &recipes[rng.IntN(len(recipes))]
		if progress < 0.3 {
			r = &recipes[rng.IntN(2)]
		}

		// Customers get less patient and tickets come faster as the rush builds.
		patience := int((22 - 10*progress) * tps)
		q = append(q, Ticket{Recipe: r, SpawnTick: t, Patience: patience, MaxPatience: patience})

		gap := 7.0 - 5.0*progress + rng.Float64()*1.5
		t += int(gap * tps)
	}
	return q
}

func plateMatches(plate []Ingredient, r *Recipe) bool {
	if len(plate) != len(r.Layers) {
		return false
	}
	for i := range plate {
		if plate[i] != r.Layers[i] {
			return false
		}
	}
	return true
}
