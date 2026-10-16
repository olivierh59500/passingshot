package game

func (g *Model) demoControl(index int) Control {
	p := &g.Players[index]
	if !p.Active {
		return 0
	}
	targetX, targetY := 320, 460
	if g.BallSide < 0 {
		remaining := (256 - g.Ball.CurveIndex) * g.Ball.CurveRange / 256
		targetX = g.Ball.X + g.Ball.DirectionX*g.Ball.Horizontal*remaining/max(1, g.Ball.Range)
		targetY = clamp(g.Ball.Y+remaining+10, 318, 520)
	}
	var c Control
	if p.X < targetX-2 {
		c |= Right
	} else if p.X > targetX+2 {
		c |= Left
	}
	if p.Y < targetY-2 {
		c |= Down
	} else if p.Y > targetY+2 {
		c |= Up
	}
	if g.BallSide < 0 && !g.contact && g.Ball.Height <= 128 && abs(g.Ball.X-p.X) < 25 && g.Ball.Y >= p.Y-18 && g.Ball.Y <= p.Y+15 {
		c = Fire | Up
	}
	return c
}
