package input

import "github.com/olivierh59500/passingshot/internal/game"

// Buttons applies short-serve buffering and a default forward shot to the
// combined keyboard, gamepad, mouse and touch actions for one player.
type Buttons struct {
	controller Controller
	previous   game.Control
}

func (b *Buttons) Map(raw game.Control, phase game.Phase, paused, serveReady bool) game.Control {
	pressed := raw&game.Fire != 0 && b.previous&game.Fire == 0
	b.previous = raw
	if paused || phase < game.Serve || phase > game.PointResult {
		b.controller.Reset()
		return raw
	}
	frame := ControllerFrame{
		ControllerState: ControllerState{Direction: raw & (game.Up | game.Down | game.Left | game.Right), HitDown: raw&game.Fire != 0},
		HitPressed:      pressed,
	}
	return b.controller.GameInput(frame, phase, false, serveReady).Players[0]
}

func (b *Buttons) Reset() { *b = Buttons{} }
