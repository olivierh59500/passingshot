package game

type ServiceRotation struct {
	Config ServiceConfig
	Cycle  int
}

func (s *ServiceRotation) Point(doubles bool, slot int, tieBreak bool) {
	index := s.Cycle
	far, actor := false, slot
	if doubles {
		if tieBreak {
			far = [8]bool{false, true, true, false, false, true, true, false}[index]
			actor = [8]int{0, 0, 0, 1, 1, 1, 1, 0}[index]
		} else {
			far = index&1 != 0
			actor = index / 2
		}
	} else if tieBreak {
		far = [4]bool{false, true, true, false}[index]
	} else {
		far = index&1 != 0
	}
	if tieBreak {
		length := 4
		if doubles {
			length = 8
		}
		s.Cycle = (index + 1) % length
	}
	c := &s.Config
	c.Doubles = doubles
	c.Far = far
	c.Slot = actor
	if !doubles {
		c.NearLeft = !c.NearLeft
		c.FarRight = !c.FarRight
		if far {
			c.NearReceiver = slot
		} else {
			c.FarReceiver = slot
		}
	} else if far {
		c.NearReceiver ^= 1
		c.FarRight = !c.FarRight
	} else {
		c.NearLeft = !c.NearLeft
		c.FarReceiver ^= 1
	}
}

func (s *ServiceRotation) NewGame(doubles bool, slot int, tieBreak, first bool) {
	if first || tieBreak {
		s.Cycle = 0
	} else {
		s.Cycle = (s.Cycle + 1) % 4
	}
	s.Point(doubles, slot, tieBreak)
	s.Config.NearLeft = false
	s.Config.NearReceiver = 0
	s.Config.FarRight = false
	s.Config.FarReceiver = 0
}

func (g *Model) StartServiceRally() {
	b := &g.Ball
	g.Outcome = 0
	g.BounceMark = BounceMark{}
	g.CameraX = 160
	g.ballFlags = 0
	b.Stopped = false
	g.RallyHits = 0
	b.Bounces = 0
	b.CurveAccumulator = 0
	b.XAccumulator = 0
	b.CurveIndex = 7
	g.contact = true
	selection := g.Service.Power & 255
	kind := 3
	if selection&1 != 0 {
		kind = 0
	} else if selection&2 != 0 {
		kind = 1
	} else if selection&4 != 0 {
		kind = 2
	}
	b.HeightDivisor = [4]int{8, 4, 16, 8}[kind]
	b.SpeedFixed = ([4]int{4, 3, 2, 3}[kind] + g.Level/2) << 16
	b.Range = int(int16(uint16(word(g.Resources.Raw["service_power"], g.Service.Power)))) + 120
	b.Horizontal = int(g.random()&7)*8 + 30
	b.CurveRange = b.Range
	for i := range g.Players {
		p := &g.Players[i]
		p.Control = 0
		p.Swing = 0
		p.Animation = 0
	}
	actor := g.PredictCPU()
	g.Players[actor].TargetY = min(220, g.Players[actor].TargetY)
}
