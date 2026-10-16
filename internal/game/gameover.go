package game

// GameOverFrames is the original sequence's 119 presentations at 25 Hz.
// The first image is prepared immediately; each image remains for two ticks.
const GameOverFrames = 119

type GameOverLetter struct {
	X, Remainder int
}

// GameOverView retains the state actually drawn before the original buffer
// swap, coordinate decrement and loop-counter updates.
type GameOverView struct {
	Frame, X, Height, Countdown int
	Spin                        [2]uint16
	Letters                     [4]GameOverLetter
	Opening                     bool
}

type GameOverState struct {
	Ball      Ball
	Spin      uint16
	X, Frame  int
	Letters   [4]GameOverLetter
	Presented GameOverView
	Bounced   bool
	Complete  bool
	part      int
	remaining int
}

func NewGameOver(ball Ball, spin uint16) GameOverState {
	ball.Y, ball.Height = 84, 0
	ball.CurveIndex, ball.CurveAccumulator = 0, 0
	ball.CurveRange, ball.HeightDivisor = 15, 4
	ball.SpeedFixed = 8<<16 | ball.SpeedFixed&0xffff
	ball.Stopped = false
	s := GameOverState{Ball: ball, Spin: spin, X: 160}
	for i := range s.Letters {
		s.Letters[i] = GameOverLetter{149, 28}
	}
	return s
}

// Step prepares one original 25 Hz presentation. A false result means the
// final image's wait has ended and the original branches back to name entry.
func (s *GameOverState) Step(curve []byte) bool {
	if s.part == 4 {
		s.Complete = true
		return false
	}
	opening := s.part < 2
	if opening {
		if s.part == 1 {
			s.Ball.HeightDivisor = 4
		}
		s.Bounced = s.Ball.StepArc(curve)
	} else if s.part == 2 {
		for i, amount := range [4]int{56, 40, 24, 8} {
			for repeat := 0; repeat < 3; repeat++ {
				p := &s.Letters[i]
				p.Remainder -= amount
				if p.Remainder < 0 {
					p.Remainder += 56
					p.X--
				}
			}
		}
	}
	s.Frame++
	view := GameOverView{Frame: s.Frame, X: s.X, Height: s.Ball.Height, Opening: opening}
	if !opening {
		view.Letters, view.Countdown = s.Letters, s.remaining-1
	}
	for i := range view.Spin {
		s.Spin += 2
		if s.Spin > 16 {
			s.Spin = 0
		}
		view.Spin[i] = s.Spin
	}
	s.Ball.X = 320 - s.X
	s.Presented = view
	switch s.part {
	case 0:
		s.X -= 6
		if s.Bounced {
			s.part = 1
		}
	case 1:
		if s.Ball.CurveIndex >= 128 {
			s.part, s.remaining = 2, 19
		}
	case 2:
		s.remaining--
		if s.remaining == 0 {
			s.part, s.remaining = 3, 81
		}
	case 3:
		s.remaining--
		if s.remaining == 0 {
			s.part = 4
		}
	}
	return true
}

// BeginGameOver starts the original loss animation and retains sound state.
func (g *Model) BeginGameOver() {
	g.GameOver = NewGameOver(g.Ball, g.BallSpin)
	g.CameraX, g.CameraY = 0, 0
	g.ballFlags = 4
	g.Message = ""
	g.transition(GameOver)
	g.stepGameOver()
}

func (g *Model) stepGameOver() {
	if !g.GameOver.Step(g.Resources.Raw["flight_curve"]) {
		if g.Demo {
			g.ReturnToTitle()
		} else {
			g.BeginNames()
		}
		return
	}
	g.Ball, g.BallSpin, g.Bounced = g.GameOver.Ball, g.GameOver.Spin, g.GameOver.Bounced
	g.BounceMark.Remaining, g.BounceMark.Draw = 0, false
}
