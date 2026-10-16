// Package input maps portable touch coordinates to original joystick controls.
package input

import "github.com/olivierh59500/passingshot/internal/game"

type Point struct {
	ID   int
	X, Y float64
}
type finger struct {
	x, y float64
	role int
}
type Touch struct{ fingers map[int]finger }

// Poll receives coordinates in the original 320 by 200 logical frame.
// A finger keeps its initial role until release, even if it crosses the screen.
func (t *Touch) Poll(points []Point, phase game.Phase, paused bool) game.Input {
	if t.fingers == nil {
		t.fingers = map[int]finger{}
	}
	live := make(map[int]finger, len(points))
	var out game.Input
	for _, point := range points {
		f, held := t.fingers[point.ID]
		if !held {
			f = finger{point.X, point.Y, 0}
			switch {
			case point.Y < 25 && point.X > 292:
				out.Pause = true
				f.role = 3
			case paused:
				if point.Y > 106 && point.Y < 134 {
					if point.X < 160 {
						out.Restart = true
					} else {
						out.Menu = true
					}
				} else {
					out.Confirm = true
				}
				f.role = 3
			case phase == game.Introduction || phase >= game.SetResult:
				out.Confirm = true
				f.role = 3
			case phase == game.Names:
				player, origin := 0, float64(174)
				if point.X < 160 {
					player, origin = 1, 54
				}
				if point.X >= origin && point.X < origin+96 && point.Y >= 134 && point.Y < 174 {
					out.NamePick[player] = int((point.Y-134)/8)*6 + int((point.X-origin)/16)
					out.NamePickSet[player] = true
				} else {
					out.NameJoin[player] = true
				}
				f.role = 3
			case phase == game.LevelSelect && point.X >= 96 && point.X < 224 && point.Y >= 30 && point.Y < 114:
				// The original list has four rows spaced 21 stored pixels apart.
				out.Selection = int((point.Y - 30) / 21)
				out.SelectionSet = true
				out.Confirm = true
				f.role = 3
			case point.X < 160:
				f.role = 1
			default:
				f.role = 2
			}
		}
		live[point.ID] = f
		if f.role == 1 {
			dx, dy := point.X-f.x, point.Y-f.y
			if dx < -5 {
				out.Players[0] |= game.Left
			}
			if dx > 5 {
				out.Players[0] |= game.Right
			}
			if dy < -5 {
				out.Players[0] |= game.Up
			}
			if dy > 5 {
				out.Players[0] |= game.Down
			}
		}
		if f.role == 2 {
			out.Players[0] |= game.Fire
		}
	}
	t.fingers = live
	return out
}

func (t *Touch) Reset() { t.fingers = nil }
