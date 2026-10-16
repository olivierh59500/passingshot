package game

type Control uint8

const (
	Up Control = 1 << iota
	Down
	Left
	Right
	Fire Control = 128
)

type Player struct {
	X, Y                                                                                          int
	Active, Far                                                                                   bool
	Bank                                                                                          int
	Control                                                                                       Control
	Animation, AnimationWait, AnimationFrame, Swing, SwingKind, SwingSide, FrameTable, FrameIndex int
	TargetX, TargetY, Reaction                                                                    int
}

func (p *Player) Advance(speed, ballX int, others []Player) {
	if !p.Active {
		return
	}
	if p.Swing > 0 {
		p.Swing--
		p.selectSwing()
		return
	}
	c := p.Control
	if c&Fire != 0 && c&(Up|Down|Left|Right) != 0 {
		p.SwingKind = 0
		if c&Up != 0 {
			p.SwingKind = 0
		} else if c&Down != 0 {
			p.SwingKind = 1
		} else if c&Left != 0 {
			p.SwingKind = 2
		} else if c&Right != 0 {
			p.SwingKind = 3
		}
		p.Swing = 7
		p.Animation = 0
		p.AnimationWait = 0
		p.AnimationFrame = 0
		p.SwingSide = 0
		if p.X > ballX+8 {
			p.SwingSide = 1
		}
		p.selectSwing()
		return
	}
	p.FrameTable = 0x400
	p.Animation = 0
	if p.Far {
		p.FrameTable = 0x440
	}
	if c&(Up|Down|Left|Right) != 0 {
		lo, hi := 318, 520
		if p.Far {
			lo, hi = 80, 282
		}
		y := p.Y
		if c&Up != 0 && y > lo {
			y -= speed
		} else if c&Up == 0 && c&Down != 0 && y <= hi {
			y += speed
		}
		if p.canMove(p.X, y, others) {
			p.Y = y
		}
		x := p.X
		if c&Left != 0 && x >= 88 {
			x -= speed
		} else if c&Left == 0 && c&Right != 0 && x < 552 {
			x += speed
		}
		if p.canMove(x, y, others) {
			p.X = x
		}
		switch {
		case c&Up != 0 && c&Left != 0:
			p.Animation = 8
		case c&Up != 0 && c&Right != 0:
			p.Animation = 2
		case c&Down != 0 && c&Left != 0:
			p.Animation = 6
		case c&Down != 0 && c&Right != 0:
			p.Animation = 4
		case c&Up != 0:
			p.Animation = 1
		case c&Down != 0:
			p.Animation = 5
		case c&Left != 0:
			p.Animation = 7
		case c&Right != 0:
			p.Animation = 3
		}
		tables := [9]int{0x400, 0, 0x40, 0x80, 0xc0, 0x100, 0x140, 0x180, 0x1c0}
		p.FrameTable = tables[p.Animation]
	}
	p.AnimationWait--
	if p.AnimationWait < 0 {
		p.AnimationWait = 2
		p.AnimationFrame = (p.AnimationFrame + 1) & 3
	}
	p.FrameIndex = p.AnimationFrame
}

func (p *Player) canMove(x, y int, others []Player) bool {
	for _, other := range others {
		dx, dy := x-other.X, y-other.Y
		if other.Active && other.Bank != p.Bank && other.Bank/2 == p.Bank/2 && dx >= -15 && dx < 15 && dy >= -15 && dy < 15 {
			return false
		}
	}
	return true
}

func (p *Player) selectSwing() {
	if p.Far {
		if p.SwingSide != 0 {
			p.FrameTable = 0x380
		} else {
			p.FrameTable = 0x300
		}
	} else {
		if p.SwingSide != 0 {
			p.FrameTable = 0x200
		} else {
			p.FrameTable = 0x280
		}
	}
	p.FrameIndex = p.Swing
}
