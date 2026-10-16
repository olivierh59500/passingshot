package game

type PointState struct {
	Remaining, Frame, MessageIndex, Fault int
	GameEnded, Scored                     bool
}

const ServiceFaultTicks = 126

func (g *Model) BeginServiceFault() {
	g.FaultFrames = g.FaultFrames[:0]
	if g.ServeAttempt == 0 {
		g.ServeAttempt = 1
		g.PointWinner = -1
		g.Point = PointState{Fault: 1}
		g.Message = "FAULT"
		g.effect(8)
	} else {
		g.Point = PointState{Fault: 2}
		g.awardPoint(1)
		if g.Point.GameEnded {
			g.StartCelebration(1)
		}
		g.Message = "DOUBLE FAULT"
		g.effect(9)
	}
	g.pointDelay = ServiceFaultTicks
	g.transition(PointResult)
}

func (g *Model) StepServiceFault() {
	if g.Point.Fault == 2 && g.Point.GameEnded && !g.Celebration.Ended {
		g.StepCelebration()
		g.Point.Frame++
		g.FaultFrames = append(g.FaultFrames, g.Celebration.Frame)
	}
}

func (g *Model) BeginPoint(outcome int) {
	winner := 0
	if outcome&0x8000 != 0 {
		winner = 1
	}
	g.Point = PointState{Remaining: 71, MessageIndex: (outcome & 0x7fff) / 4}
	g.scorePoint(winner)
	for i := 2; i < 4; i++ {
		g.Players[i].Control = 0
	}
	if g.Point.GameEnded {
		g.StartCelebration(winner)
	}
	g.Message = [9]string{"OUT", "OUT", "BODY TOUCH", "SERVICE ACE", "RETURN ACE", "NET", "DIRECT", "ACE", "ERROR"}[g.Point.MessageIndex]
	g.transition(PointResult)
}

func (g *Model) scorePoint(winner int) {
	g.awardPoint(winner)
	g.rotatePoint()
}

func (g *Model) awardPoint(winner int) {
	g.PointWinner = winner
	g.Point.Scored = true
	g.Point.GameEnded = g.Score.Point(winner)
	g.TotalPoints++
}

func (g *Model) rotatePoint() {
	if g.Point.GameEnded {
		if g.Score.TieBreak && g.Score.Winner >= 0 {

			g.Rotation.NewGame(g.Doubles, g.SinglesPlayer, false, true)
		} else {
			g.Rotation.NewGame(g.Doubles, g.SinglesPlayer, g.Score.TieBreak, false)
		}
	} else {
		g.Rotation.Point(g.Doubles, g.SinglesPlayer, g.Score.TieBreak)
	}
	g.NearReceiver = g.Rotation.Config.NearReceiver
}

// StepPoint advances the original settling order without CPU joystick updates.
// Human movement, existing swings, racket contact, flight, and drawing continue.
func (g *Model) StepPoint() {
	if g.Point.Remaining <= 0 || g.Point.Fault != 0 {
		return
	}
	g.advancePlayers(false)
	g.ResolveRacket()
	g.AdvanceFlight()
	g.AdvanceDrawing()
	if g.Point.GameEnded {
		g.StepCelebration()
	}
	g.Point.Frame++
	g.Point.Remaining--
}

func (g *Model) advancePlayers(computer bool) {
	for i := 0; i < 2; i++ {
		g.Players[i].Control = g.joystickLatch[i]
		if g.Demo {
			g.Players[i].Control = g.demoControl(i)
		}
	}
	if computer {
		for i := 2; i < 4; i++ {
			g.Players[i].Control = g.CPUControl(i)
		}
	}
	speed := 2
	if g.Level >= 3 {
		speed = 4
	}
	g.MatchFrame++
	for i := range g.Players {
		g.Players[i].Advance(speed, g.Ball.X, g.Players[:])
	}
}
