package input

import "github.com/olivierh59500/passingshot/internal/game"

// GamepadPlayer keeps menu joining stable, follows the chosen player when
// only one gamepad is connected, and pairs touch player one with an external
// gamepad controlling player two in doubles.
func GamepadPlayer(index, connected int, g *game.Model, touch bool) int {
	if connected == 1 && g.Phase == game.LevelSelect {
		return max(0, g.NameEntry.First)
	}
	if g.Phase < game.Introduction {
		return index
	}
	if !g.Doubles {
		if connected == 1 {
			return g.SinglesPlayer
		}
		return index
	}
	if touch {
		return 1 - index
	}
	return index
}
