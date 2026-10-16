package game

// ServicePreparation briefly presents the court and scoreboard before actors
// begin their service animation. Assets are already decoded before gameplay.
type ServicePreparation struct{ Remaining int }

func (g *Model) BeginServicePreparation() {
	g.prepareServiceScene(false)
	g.Preparation = &ServicePreparation{Remaining: 6}
}

func (g *Model) stepServicePreparation() {
	g.Preparation.Remaining--
	if g.Preparation.Remaining > 0 {
		return
	}
	g.Preparation = nil
	g.PhaseTick = 0
	// The service cue suspends the stage composition without resetting it.
	g.song(-2)
	g.effect(10)
	g.StepService()
}
