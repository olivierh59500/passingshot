package input

import (
	"math"

	"github.com/olivierh59500/passingshot/internal/game"
)

// Both axes use the same scale so the 320 by 200 artwork keeps square pixels.
const (
	GameScaleX = 1.2
	GameScaleY = 1.2
	GameWidth  = 384
	GameHeight = 240
)

// ControllerLayout reserves side gutters around the game picture.
// Game pixels remain separate from the native touch controls.
type ControllerLayout struct {
	Width, Height, GameX, GameY int
	PadX, PadY, PadRadius       float64
	HitX, HitY, HitRadius       float64
	PauseX, PauseY, PauseRadius float64
}

func LayoutController(width, height int) ControllerLayout {
	width, height = max(1, width), max(1, height)
	const minWidth = GameWidth + 200
	w, h := max(minWidth, int(float64(width)*GameHeight/float64(height))/2*2), GameHeight
	if w == minWidth {
		h = max(GameHeight, int(float64(height)*minWidth/float64(width))/2*2)
	}
	margin := float64(w-GameWidth) / 2
	radius := min(43.0, margin/2-6)
	return ControllerLayout{Width: w, Height: h, GameX: (w - GameWidth) / 2, GameY: (h - GameHeight) / 2,
		PadX: margin / 2, PadY: float64(h) * 0.74, PadRadius: radius,
		HitX: float64(w) - margin/2, HitY: float64(h) * 0.74, HitRadius: min(34.0, radius),
		PauseX: float64(w) - margin/2, PauseY: 25, PauseRadius: 17}
}

// GamePoint maps a displayed contact back to original artwork coordinates.
func (l ControllerLayout) GamePoint(x, y float64) (float64, float64) {
	return (x - float64(l.GameX)) / GameScaleX, (y - float64(l.GameY)) / GameScaleY
}

func (l ControllerLayout) GamePosition(x, y float64) (float64, float64) {
	return float64(l.GameX) + x*GameScaleX, float64(l.GameY) + y*GameScaleY
}

type ControllerState struct {
	Direction                   game.Control
	PadDown, HitDown, PauseDown bool
	ServePending                bool
}

type ControllerFrame struct {
	ControllerState
	HitPressed, PausePressed bool
}

type Controller struct {
	fingers                 map[int]int
	State                   ControllerState
	servePending, serveSent bool
	serveDirection          game.Control
}

func inside(x, y, cx, cy, radius float64) bool { return math.Hypot(x-cx, y-cy) <= radius }

// Poll assigns each contact to the fixed pad or button on touch-down. A tap
// on a pad direction acts immediately and supports a second firing contact.
func (c *Controller) Poll(points []Point, layout ControllerLayout) ControllerFrame {
	if c.fingers == nil {
		c.fingers = map[int]int{}
	}
	live := make(map[int]int, len(points))
	var frame ControllerFrame
	for _, p := range points {
		role, held := c.fingers[p.ID]
		if !held {
			switch {
			case inside(p.X, p.Y, layout.PadX, layout.PadY, layout.PadRadius+5):
				role = 1
			case inside(p.X, p.Y, layout.HitX, layout.HitY, layout.HitRadius+5):
				role = 2
				frame.HitPressed = true
			case inside(p.X, p.Y, layout.PauseX, layout.PauseY, layout.PauseRadius+5):
				role = 3
				frame.PausePressed = true
			}
		}
		live[p.ID] = role
		switch role {
		case 1:
			frame.PadDown = true
			dx, dy := p.X-layout.PadX, p.Y-layout.PadY
			dead := layout.PadRadius * 0.22
			if dx < -dead {
				frame.Direction |= game.Left
			}
			if dx > dead {
				frame.Direction |= game.Right
			}
			if dy < -dead {
				frame.Direction |= game.Up
			}
			if dy > dead {
				frame.Direction |= game.Down
			}
		case 2:
			frame.HitDown = true
		case 3:
			frame.PauseDown = true
		}
	}
	c.fingers = live
	c.State = frame.ControllerState
	return frame
}

func (c *Controller) Reset() {
	c.fingers = nil
	c.State = ControllerState{}
	c.servePending = false
	c.serveSent = false
	c.serveDirection = 0
}

// GameInput retains a short serve tap through the toss until the player strikes.
// Releasing the button must not cancel the serve.
func (c *Controller) GameInput(f ControllerFrame, phase game.Phase, paused, serveReady bool) game.Input {
	if phase == game.Serve && !paused && !f.PausePressed {
		if f.HitPressed {
			c.servePending = true
			c.serveSent = false
			c.serveDirection = f.Direction
		}
		if c.serveSent && !serveReady {
			c.servePending = false
		}
		if c.servePending {
			f.HitDown = true
			if f.Direction == 0 {
				f.Direction = c.serveDirection
			}
			if serveReady {
				c.serveSent = true
			}
		}
	} else {
		c.servePending = false
		c.serveSent = false
		c.serveDirection = 0
	}
	c.State.ServePending = c.servePending
	return f.GameInput(phase, paused, serveReady)
}

// GameInput adapts the action button to touch input. A plain hit selects a flat
// drive. An early serve starts the toss; its shot direction waits for the strike.
func (f ControllerFrame) GameInput(phase game.Phase, paused, serveReady bool) game.Input {
	in := game.Input{Pause: f.PausePressed}
	if f.PausePressed {
		return in
	}
	if paused || phase == game.Introduction || phase >= game.SetResult {
		in.Confirm = f.HitPressed
		return in
	}
	if phase != game.Serve && phase != game.Rally && phase != game.PointResult {
		return game.Input{}
	}
	if phase == game.Serve {
		if !f.HitDown {
			return in
		}
		if !serveReady {
			// Start the toss immediately; retain the chosen shot until the
			// animation is ready to strike, including after touch release.
			in.Players[0] = game.Fire
			return in
		}
	}
	control := f.Direction
	if f.HitDown {
		if control == 0 {
			control = game.Up
		}
		control |= game.Fire
	}
	in.Players[0] = control
	return in
}
