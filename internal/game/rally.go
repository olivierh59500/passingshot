package game

import "encoding/binary"

type RallyState struct {
	Ball                                                  Ball
	Players                                               [4]Player
	Random                                                uint32
	Level, Hits, Side, Outcome, BlinkPlayer, NearReceiver int
	Doubles, Miss, Contact, Bounced                       bool
	Flags                                                 uint16
	SinglesPlayer                                         int
}

func (g *Model) RallyState() RallyState {
	return RallyState{Ball: g.Ball, Players: g.Players, Random: g.rng, Level: g.Level, Hits: g.RallyHits, Side: g.BallSide, Outcome: g.Outcome, BlinkPlayer: g.BlinkPlayer, NearReceiver: g.NearReceiver, SinglesPlayer: g.SinglesPlayer, Doubles: g.Doubles, Miss: g.Miss, Contact: g.contact, Bounced: g.Bounced, Flags: g.ballFlags}
}

func (g *Model) RestoreRally(s RallyState) {
	g.Ball = s.Ball
	g.Players = s.Players
	g.rng = s.Random
	g.Level = s.Level
	g.RallyHits = s.Hits
	g.BallSide = s.Side
	g.Outcome = s.Outcome
	g.BlinkPlayer = s.BlinkPlayer
	g.NearReceiver = s.NearReceiver
	g.SinglesPlayer = s.SinglesPlayer
	g.Doubles = s.Doubles
	g.Miss = s.Miss
	g.contact = s.Contact
	g.Bounced = s.Bounced
	g.ballFlags = s.Flags
}

func word(data []byte, index int) int {
	if index < 0 || index*2+2 > len(data) {
		panic("original table index outside reconstructed data")
	}
	return int(binary.BigEndian.Uint16(data[index*2:]))
}

func (g *Model) ResolveRacket() bool {
	if g.Ball.Height < 4 || g.Ball.Height > 128 {
		g.contact = false
		return false
	}
	for i := 0; i < 2; i++ {
		p := &g.Players[i]
		if !p.Active || p.Swing == 0 {
			continue
		}
		f := g.Resources.PlayerFrame(p.FrameTable, p.FrameIndex)
		x, y := p.X+f.X+f.RacketX, p.Y+f.Y+f.RacketY
		if g.Ball.X >= x-4 && g.Ball.X <= x+20 && g.Ball.Y >= y-4 && g.Ball.Y <= y+20 {
			return g.ResolveShot(i)
		}
	}
	for i := 2; i < 4; i++ {
		p := &g.Players[i]
		if p.Active && p.Swing > 0 && p.Swing <= 5 {
			return g.ResolveShot(i)
		}
	}
	g.contact = false
	return false
}

func (g *Model) nearShot(index int) {
	p := &g.Players[index]
	if g.RallyHits == 0 {
		if g.Ball.Bounces == 0 {
			g.Outcome = 0x8018
		}
		if g.Doubles && index != g.NearReceiver {
			g.Outcome = 0x8020
		}
	}
	kind := p.SwingKind
	depthIndex := clamp(g.Ball.Y-300, 0, 191) >> 4
	rangeY := word(g.Resources.Raw["shot_depth"], kind*12+depthIndex)
	directionIndex := p.Swing*16 + (rangeY >> 4)
	horizontal := int(g.Resources.Raw["shot_direction"][directionIndex])
	direction := -1
	if (p.Swing-1)&4 != 0 {
		direction = 1
	}
	g.setShot(kind, rangeY, horizontal, direction, -1)
	g.BallSide = 1
	g.Miss = false
	g.LastHitter = index
	g.effect(kind + 4)
	g.PredictCPU()
	g.finishShot()
}

// ResolveShot runs the selected native handler after its racket has contacted
// the ball, preserving the original human/CPU side and contact-latch gates.
func (g *Model) ResolveShot(index int) bool {
	if g.contact {
		return false
	}
	if index < 2 {
		if g.BallSide >= 0 {
			return false
		}
		g.nearShot(index)
		return true
	}
	if g.BallSide < 0 {
		return false
	}
	if g.RallyHits != 0 && int(g.random()&255) >= word(g.Resources.Raw["ai_accuracy"], g.Level) {
		g.Miss = true
		return false
	}
	g.farShot(index)
	return true
}

func (g *Model) farShot(index int) {
	p := &g.Players[index]
	p.SwingKind = word(g.Resources.Raw["ai_shots"], int(g.random()&0x1e)/2)
	kind := p.SwingKind
	depth := word(g.Resources.Raw["ai_depth"], kind*6+g.Level)
	var random uint32
	for {
		random = g.random()
		if int(random&255) <= depth {
			break
		}
	}
	variation := depth >> 4
	if random&0x40000000 != 0 {
		variation = -variation
	}
	rangeY := int(uint16(int(random&255) + 300 - g.Ball.Y + variation))
	minimum := 40
	if kind == 1 {
		minimum = 100
	}
	rangeY = max(rangeY, minimum)
	if int(uint16(g.Ball.Y+rangeY)) < 300 {
		rangeY = 300 - g.Ball.Y
	}
	spin := int(g.Resources.Raw["random_spin"][g.random()&15])
	horizontal := int(g.Resources.Raw["shot_direction"][spin*16+(rangeY>>4&255)])
	direction := 1
	if g.Ball.X >= 320 {
		direction = -1
	}
	g.setShot(kind, rangeY, horizontal, direction, 1)
	g.BallSide = -1
	g.Miss = true
	g.LastHitter = index
	g.effect(kind + 4)
	g.PredictCPU()
	g.finishShot()
}

func (g *Model) setShot(kind, rangeY, horizontal, dx, dy int) {
	b := &g.Ball
	b.Range = rangeY
	b.Horizontal = horizontal
	b.DirectionX = dx
	b.DirectionY = dy
	b.HeightDivisor = [4]int{8, 4, 16, 8}[kind]
	b.SpeedFixed = ([4]int{4, 3, 2, 3}[kind] + g.Level/2) << 16
}

func (g *Model) finishShot() {
	b := &g.Ball
	b.XAccumulator = 0
	b.CurveAccumulator = 0
	b.SpeedFixed &= ^0xffff
	b.Bounces = 0
	b.CurveIndex = 0
	b.CurveRange = b.Range
	b.Stopped = false
	g.RallyHits++
	g.contact = true
}

func (g *Model) PredictCPU() int {
	a, b := 2, 3
	x0, x1 := 279, 360
	if !g.Doubles {
		if g.SinglesPlayer == 1 {
			a = 3
		}
		b = a
		x0, x1 = 320, 320
	}
	if g.BallSide < 0 {
		if g.Players[a].Y > g.Players[b].Y {
			a, b = b, a
			x0, x1 = x1, x0
		}
		x, y := g.jitterPair(x0, 254)
		xOther, yOther := g.jitterPair(x1, 194)
		g.Players[b].TargetX = xOther
		g.Players[b].TargetY = yOther
		g.Players[a].TargetX = x
		g.Players[a].TargetY = y
		g.Players[a].Reaction = 600
		g.Players[b].Reaction = 600
		return a
	}
	landingX := g.Ball.X + g.Ball.DirectionX*(g.Ball.Horizontal+16)
	if landingX > 320 {
		a, b = b, a
		x0, x1 = x1, x0
	}
	x, y := g.jitterPair(landingX, g.Ball.Y-g.Ball.Range-50)
	otherY := 200
	if y <= 223 {
		otherY = 260
	}
	xOther, yOther := g.jitterPair(x1, otherY)
	g.Players[b].TargetX = xOther
	g.Players[b].TargetY = yOther
	if y > 282 {
		y -= 24
	}
	g.Players[a].TargetX = x
	g.Players[a].TargetY = y
	reaction := y + (g.Ball.Range>>2)*2 + word(g.Resources.Raw["ai_reaction"], g.Level)
	random := g.random()
	reaction += int(random&31) - 15
	if reaction < y {
		reaction = y
	}
	g.Players[b].Reaction = reaction + int(random>>16&31)
	g.Players[a].Reaction = reaction
	return a
}

func (g *Model) jitterPair(x, y int) (int, int) {
	limit := word(g.Resources.Raw["ai_jitter"], g.Level)
	jitter := func(v int) int {
		for {
			random := g.random()
			delta := int(random & 7)
			if delta >= limit {
				continue
			}
			if random&0x40000000 != 0 {
				v -= delta
			} else {
				v += delta
			}
			return v &^ 3
		}
	}
	return jitter(x), jitter(y)
}

func (g *Model) CPUControl(index int) Control {
	p := &g.Players[index]
	if !p.Active {
		return 0
	}
	var c Control
	if g.Outcome == 0 && !g.Miss && g.BallSide >= 0 && !g.contact && g.Ball.Height <= 153 {
		if g.Ball.X >= p.X-20 && g.Ball.X <= p.X+20 && g.Ball.Y >= p.Y-8 && g.Ball.Y <= p.Y+20 {
			c = 255
		}
	}
	if g.Ball.Y > p.Reaction {
		return c
	}
	x, y := p.X&^3, p.Y&^3
	if x > p.TargetX {
		c |= Left
	} else if x < p.TargetX {
		c |= Right
	}
	if y > p.TargetY {
		c |= Up
	} else if y < p.TargetY {
		c |= Down
	}
	return c
}

func (g *Model) DecideBall() {
	g.courtContact()
	if g.Outcome != 0 || !g.Bounced {
		return
	}
	l, r := 239, 400
	if g.Doubles {
		l, r = 223, 416
	}
	out := g.Ball.X < l || g.Ball.X > r || g.Ball.Y < 143 || g.Ball.Y >= 456
	if !out && g.Ball.Bounces != 2 {
		return
	}
	code := 0
	if out && g.Ball.Bounces < 2 {
		code = 0x8004
		g.effect(12)
	} else {
		if g.RallyHits == 0 {
			code = 12
		} else if g.RallyHits == 1 {
			code = 16
		} else {
			code = 28
		}
	}
	if g.BallSide < 0 {
		code ^= 0x8000
	}
	g.Outcome = code
}

func (g *Model) courtContact() {
	b := &g.Ball
	if g.ballFlags != 0 || b.Height > 50 {
		return
	}
	reflect := false
	switch {
	case b.X <= 79:
		b.DirectionX = 1
		reflect = true
	case b.X >= 560:
		b.DirectionX = -1
		reflect = true
	case b.Y <= 63:
		b.DirectionY = 1
		reflect = true
	case b.Y > 535:
		b.DirectionY = -1
		reflect = true
	default:
		if b.Height > 20 {
			return
		}
		if b.Height >= 8 {
			for i := range g.Players {
				p := &g.Players[i]
				if p.Active && b.X >= p.X-7 && b.X <= p.X+7 && b.Y >= p.Y-7 && b.Y <= p.Y+7 {
					if g.Outcome == 0 {
						g.Outcome = 8
						if i < 2 {
							g.Outcome |= 0x8000
						}
						g.BlinkPlayer = i + 1
					}
					b.DirectionX = -b.DirectionX
					b.DirectionY = -b.DirectionY
					g.reflectArc()
					return
				}
			}
		}
		if b.Height <= 15 && b.Y >= 299 && b.Y <= 303 && b.X >= 210 && b.X < 428 {
			g.effect(11)
			g.Outcome = 20
			if g.BallSide >= 0 {
				g.Outcome |= 0x8000
			}
			b.DirectionX = -b.DirectionX
			b.DirectionY = -b.DirectionY
			reflect = true
		}
	}
	if reflect {
		g.reflectArc()
	}
}

func (g *Model) reflectArc() {
	if g.Ball.CurveIndex <= 128 {
		g.Ball.CurveIndex = 256 - g.Ball.CurveIndex
	}
}

func (g *Model) AdvanceFlight() {
	steps := (g.Ball.SpeedFixed >> 16) + 1
	for i := 0; i < steps && g.ballFlags&1 == 0; i++ {
		g.Ball.StepMotion()
		b := &g.Ball
		if b.Y < -32 || b.Y >= 632 || b.X < -32 || b.X >= 652 {
			g.ballFlags = 3
			b.Bounces++
			g.Bounced = true
		}
		g.Bounced = b.StepArc(g.Resources.Raw["flight_curve"])
		if g.Bounced {
			g.BounceMark = BounceMark{Remaining: 128, X: b.X, Y: b.Y, ReverseRows: b.DirectionY == 0}
			g.effect(0)
		}
		if b.Stopped {
			g.ballFlags |= 1
		}
		if g.ballFlags&4 == 0 && b.Height <= 50 && (b.Y < 47 || b.Y >= 551 || b.X < 63 || b.X > 576) {
			g.ballFlags = 3
			b.Bounces++
			g.Bounced = true
		}
		g.DecideBall()
	}
	g.Ball.Stopped = g.ballFlags&1 != 0
	g.followCamera()
}
