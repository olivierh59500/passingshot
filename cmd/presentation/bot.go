package main

import (
	"github.com/olivierh59500/passingshot/internal/game"
)

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func chase(g *game.Model) game.Control {
	p := g.Players[g.SinglesPlayer]
	x, y := 320, 390
	if g.BallSide < 0 {
		b := g.Ball
		for n := 0; n < 100 && !b.Stopped; n++ {
			if b.Bounces > 0 && b.Height < 60 && b.Y > 320 {
				break
			}
			for i := 0; i < (b.SpeedFixed>>16)+1; i++ {
				b.StepMotion()
				b.StepArc(g.Resources.Raw["flight_curve"])
			}
		}
		x, y = b.X-8, min(510, max(320, b.Y+10))
	}
	var c game.Control
	if p.X < x-2 {
		c |= game.Right
	} else if p.X > x+2 {
		c |= game.Left
	}
	if p.Y < y-2 {
		c |= game.Down
	} else if p.Y > y+2 {
		c |= game.Up
	}
	return c
}

func distance(g *game.Model) int {
	if g.BallSide >= 0 {
		return 0
	}
	p := g.Players[g.SinglesPlayer]
	side := 0x280
	if p.X > g.Ball.X+8 {
		side = 0x200
	}
	f := g.Resources.PlayerFrame(side, 7)
	x, y := p.X+f.X+f.RacketX+8, p.Y+f.Y+f.RacketY+8
	dx, dy := abs(g.Ball.X-x), abs(g.Ball.Y-y)
	return dx*dx + dy*dy
}

// plan chooses controller input by predicting ordinary native game updates.
// The live model, scores, ball physics and opponent AI are never overridden.
func plan(g *game.Model) game.Control {
	choices := []game.Control{0, game.Up, game.Down, game.Left, game.Right, game.Up | game.Left, game.Up | game.Right, game.Down | game.Left, game.Down | game.Right, game.Up | game.Fire, game.Down | game.Fire, game.Left | game.Fire, game.Right | game.Fire}
	best, bestScore := chase(g), -1e20
	for _, choice := range choices {
		copy := *g
		copy.Events = nil
		score := 0.0
		closest := distance(&copy)
		contact := false
		for n := 0; n < 48 && copy.Phase == game.Rally; n++ {
			c := chase(&copy)
			if n < 2 {
				c = choice
			}
			copy.Update(game.Input{Players: [2]game.Control{c, 0}})
			copy.Events = nil
			closest = min(closest, distance(&copy))
			if copy.RallyHits > g.RallyHits && copy.LastHitter < 2 {
				contact = true
			}
			if copy.Phase == game.PointResult {
				if copy.PointWinner == 0 {
					score += 1e9
				} else {
					score -= 1e9
				}
			}
		}
		if contact {
			score += 1e6
		}
		score -= float64(closest)
		score -= float64(distance(&copy)) * .1
		if score > bestScore {
			bestScore, best = score, choice
		}
	}
	return best
}
