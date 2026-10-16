package game

type BounceMark struct {
	Remaining, X, Y   int
	ReverseRows, Draw bool
}

// AdvanceDrawing advances animation once per simulation frame, independently
// of Ebitengine presentation calls or the number of display refreshes.
func (g *Model) AdvanceDrawing() {
	g.BounceMark.Draw = false
	if g.ballFlags&2 != 0 {
		return
	}
	if g.BounceMark.Remaining != 0 {
		g.BounceMark.Draw = true
		g.BounceMark.Remaining -= 32
	}
	g.BallSpin += 2
	if g.BallSpin > 16 {
		g.BallSpin = 0
	}
}
