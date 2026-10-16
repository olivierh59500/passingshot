package game

type ServiceNoticeKind uint8

const (
	NoServiceNotice ServiceNoticeKind = iota
	BreakPointNotice
	MatchPointNotice
	TieBreakNotice
)

// ServiceNoticeState retains the original thirty-frame announcement countdown.
// A notice is drawn on even countdown values before the five service actions.
type ServiceNoticeState struct {
	Remaining int
	Kind      ServiceNoticeKind
}

func (g *Model) StepServiceNotice() {
	n := g.ServiceNotice.Remaining
	g.ServiceNotice.Kind = NoServiceNotice
	if n == 0 {
		return
	}
	g.ServiceNotice.Remaining--
	if n&1 == 0 {
		g.ServiceNotice.Kind = ServiceNoticeFor(g.Score, g.Service.Config.Far)
	}
}

// ServiceNoticeFor preserves the original normal-game, deuce, advantage, and
// capped tie-break conditions. Match-point announcements take precedence.
func ServiceNoticeFor(score Score, farServer bool) ServiceNoticeKind {
	near, far := score.Points[0], score.Points[1]
	if score.TieBreak {
		if near == 0 && far == 0 {
			return TieBreakNotice
		}
		difference := near - far
		if difference < 0 {
			difference = -difference
		}
		if max(near, far) >= 13 || max(near, far) >= 6 && difference >= 1 {
			return MatchPointNotice
		}
		return NoServiceNotice
	}
	threat := 1
	if near > 3 || near == 3 && far < 3 {
		threat = 0
	} else if near < 3 && far < 3 || near == 3 && far == 3 {
		return NoServiceNotice
	}
	games, other := score.Games[threat], score.Games[1-threat]
	if other < 5 && games >= 5 || other >= 5 && games >= 6 {
		return MatchPointNotice
	}
	receiver := 1
	if farServer {
		receiver = 0
	}
	points, serverPoints := score.Points[receiver], score.Points[1-receiver]
	if points > 3 || points == 3 && serverPoints < 3 {
		return BreakPointNotice
	}
	return NoServiceNotice
}
