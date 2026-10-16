package game

import "encoding/binary"

type ServiceConfig struct {
	Doubles       bool
	Slot          int
	Far, NearLeft bool
	NearReceiver  int
	FarRight      bool
	FarReceiver   int
}

type ServiceTrack struct{ Sequence, Cursor, Wait int }
type ServiceAction uint8

const (
	ServiceNone ServiceAction = iota
	ServiceIdleNear0
	ServiceIdleNear1
	ServiceIdleFar2
	ServiceIdleFar3
	ServiceHuman0
	ServiceHuman1
	ServiceCPU2
	ServiceCPU3
	ServiceBall
)

// ServiceState keeps the close-up actors, ball toss, and selected shot together.
type ServiceState struct {
	Config                                                  ServiceConfig
	Started                                                 bool
	Tracks                                                  [4]ServiceTrack
	Actions                                                 [5]ServiceAction
	BallCursor, BallWait, BallX, BallY, BallFrame, BallFlip int
	BallHidden                                              bool
	PowerStep, Power, Phase, End                            int
}

type ServiceSprite struct {
	X, Y, Width, Height int
	Offset              int
}

const serviceBallStart = 0x690

func (g *Model) ConfigureService(cfg ServiceConfig) {
	g.ServiceNotice = ServiceNoticeState{Remaining: 30}
	s := &g.Service
	s.Config = cfg
	s.Started = cfg.Far || g.Demo
	s.Phase = 0
	s.End = 0
	s.Power = 0
	s.BallFrame = 0x3c54
	s.BallWait = 0
	for i := range s.Tracks {
		s.Tracks[i].Cursor = 0
		s.Tracks[i].Wait = 5
		g.Players[i].Active = cfg.Doubles || i%2 == cfg.Slot
	}
	g.BlinkPlayer = 0
	g.Ball.X = 345
	g.Ball.DirectionX = -1
	g.Ball.Y = 420
	g.Ball.DirectionY = -1
	g.Ball.Height = 30
	left := cfg.NearLeft
	if cfg.Far {
		left = !cfg.FarRight
		g.Ball.Y = 180
		g.Ball.DirectionY = 1
	}
	if left {
		g.Ball.X = 295
		g.Ball.DirectionX = 1
	}
	g.BallSide = 1
	g.Miss = false
	if cfg.Far {
		g.BallSide = -1
		g.Miss = true
	}
	set := func(index, sequence, x, y int) {
		s.Tracks[index].Sequence = sequence
		g.Players[index].X = x
		g.Players[index].Y = y
	}
	nearServe := func() {
		if cfg.Slot == 0 {
			if cfg.NearLeft {
				set(0, 0x1e0, 285, 460)
				s.BallX = 106
			} else {
				set(0, 0x29e, 355, 460)
				s.BallX = 215
			}
		} else {
			if cfg.NearLeft {
				set(1, 0x35c, 285, 460)
				s.BallX = 106
			} else {
				set(1, 0x41a, 355, 460)
				s.BallX = 216
			}
		}
		s.BallCursor = serviceBallStart
		s.BallY = 164
	}
	if !cfg.Doubles {
		s.Actions = [5]ServiceAction{}
		if !cfg.Far {
			nearServe()
			s.Actions[3] = ServiceBall
			if cfg.Slot == 0 {
				s.Actions[2] = ServiceIdleFar2
				s.Actions[4] = ServiceHuman0
				if cfg.NearLeft {
					set(2, 0x17c, 355, 172)
				} else {
					set(2, 0x154, 285, 172)
				}
			} else {
				s.Actions[2] = ServiceIdleFar3
				s.Actions[4] = ServiceHuman1
				if cfg.NearLeft {
					set(3, 0x1cc, 355, 172)
				} else {
					set(3, 0x1a4, 285, 172)
				}
			}
		} else if cfg.Slot == 0 {
			s.Actions[2] = ServiceCPU2
			s.Actions[4] = ServiceIdleNear0
			if cfg.FarRight {
				set(0, 0x28, 285, 428)
				set(2, 0x546, 355, 140)
			} else {
				set(0, 0x78, 355, 428)
				set(2, 0x4d8, 285, 140)
			}
		} else {
			s.Actions[2] = ServiceCPU3
			s.Actions[4] = ServiceIdleNear1
			if cfg.FarRight {
				set(1, 0xc8, 285, 428)
				set(3, 0x622, 355, 140)
			} else {
				set(1, 0x118, 355, 428)
				set(3, 0x5b4, 285, 140)
			}
		}
		return
	}
	if !cfg.Far {
		s.Actions = [5]ServiceAction{ServiceIdleFar3, ServiceIdleFar2, ServiceBall, ServiceHuman0, ServiceIdleNear1}
		if cfg.Slot == 1 {
			s.Actions[3] = ServiceIdleNear0
			s.Actions[4] = ServiceHuman1
		}
		nearServe()
		if cfg.Slot == 0 {
			if cfg.NearLeft {
				set(1, 0xf0, 355, 330)
			} else {
				set(1, 0xa0, 285, 330)
			}
		} else {
			if cfg.NearLeft {
				set(0, 0x50, 355, 330)
			} else {
				set(0, 0x0, 285, 330)
			}
		}
		if !cfg.FarRight {
			if cfg.FarReceiver != 0 {
				set(2, 0x140, 285, 260)
				set(3, 0x1cc, 355, 172)
			} else {
				set(2, 0x154, 285, 172)
				set(3, 0x1b8, 355, 260)
			}
		} else {
			if cfg.FarReceiver != 0 {
				set(2, 0x168, 355, 260)
				set(3, 0x1a4, 285, 172)
			} else {
				set(2, 0x17c, 355, 172)
				set(3, 0x190, 285, 260)
			}
		}
	} else {
		s.Actions = [5]ServiceAction{ServiceCPU2, ServiceIdleFar3, ServiceNone, ServiceIdleNear0, ServiceIdleNear1}
		if cfg.Slot == 1 {
			s.Actions[0] = ServiceCPU3
			s.Actions[1] = ServiceIdleFar2
		}
		if !cfg.NearLeft {
			if cfg.NearReceiver != 0 {
				set(0, 0x50, 355, 330)
				set(1, 0xc8, 285, 428)
			} else {
				set(0, 0x78, 355, 428)
				set(1, 0xa0, 285, 330)
			}
		} else {
			if cfg.NearReceiver != 0 {
				set(0, 0x0, 285, 330)
				set(1, 0x118, 355, 428)
			} else {
				set(0, 0x28, 285, 428)
				set(1, 0xf0, 355, 330)
			}
		}
		if cfg.Slot == 0 {
			if cfg.FarRight {
				set(2, 0x546, 355, 140)
				set(3, 0x190, 285, 260)
			} else {
				set(2, 0x4d8, 285, 140)
				set(3, 0x1b8, 355, 260)
			}
		} else {
			if cfg.FarRight {
				set(2, 0x140, 285, 260)
				set(3, 0x622, 355, 140)
			} else {
				set(2, 0x168, 355, 260)
				set(3, 0x5b4, 285, 140)
			}
		}
	}
}

func (g *Model) SetServiceInput(input [2]Control) { g.joystickLatch = input }
func (g *Model) ServiceInput() [2]Control         { return g.joystickLatch }

// ServiceStrikeReady reports when the human server can choose a shot.
// Input buffering retains the action button through the toss animation.
func (g *Model) ServiceStrikeReady() bool {
	if g.Preparation != nil {
		return false
	}
	s := g.Service
	if s.Config.Far || s.End != 0 || s.Phase != 2 {
		return false
	}
	t := s.Tracks[s.Config.Slot]
	data := g.Resources.Raw["service_sequences"]
	offset := t.Sequence + t.Cursor
	return offset >= 0 && offset+2 <= len(data) && binary.BigEndian.Uint16(data[offset:]) == 3
}

func (g *Model) StepService() {
	g.StepServiceNotice()
	g.ServiceSpriteCount = 0
	for _, action := range g.Service.Actions {
		switch action {
		case ServiceIdleNear0:
			g.serviceActor(0, 0)
		case ServiceIdleNear1:
			g.serviceActor(1, 0)
		case ServiceIdleFar2:
			g.serviceActor(2, 0)
		case ServiceIdleFar3:
			g.serviceActor(3, 0)
		case ServiceHuman0:
			g.serviceActor(0, 1)
		case ServiceHuman1:
			g.serviceActor(1, 1)
		case ServiceCPU2:
			g.serviceActor(2, 2)
		case ServiceCPU3:
			g.serviceActor(3, 2)
		case ServiceBall:
			g.serviceBall()
		}
	}
}

func (g *Model) serviceActor(index, kind int) {
	s := &g.Service
	t := &s.Tracks[index]
	raw := g.Resources.Raw["service_sequences"]
	off := t.Sequence + t.Cursor
	entry := raw[off : off+10]
	command := int(binary.BigEndian.Uint16(entry))
	advance := func(wait int) { t.Wait = wait; t.Cursor += 10 }
	switch kind {
	case 0:
		t.Wait--
		if t.Wait < 0 {
			t.Wait = 4
			if index >= 2 {
				t.Wait = 8
			}
			if command == 1 {
				t.Cursor += 10
			} else {
				t.Cursor = 0
			}
		}
	case 1:
		switch command {
		case 1, 7:
			if command == 7 {
				g.joystickLatch[index] = 0
			}
			t.Wait--
			if t.Wait < 0 {
				advance(1)
			}
		case 8:
			advance(30)
		case 2:
			if s.Phase == 2 {
				advance(1)
			}
		case 3:
			input := g.joystickLatch[index] & 127
			if input != 0 {
				s.Power = (s.Power & 0xff00) | int(input)
				g.effect(index + 1)
				advance(1)
			}
		case 6:
			if s.Phase == 1 {
				s.End = 1
				s.Power = s.PowerStep
			} else {
				advance(1)
			}
		}
	case 2:
		switch command {
		case 1, 2:
			t.Wait--
			if t.Wait < 0 {
				advance(1)
			}
		case 8:
			advance(15)
		default:
			s.End = 1
			random := g.random()
			s.Power = (s.Power & 0xff00) | 1<<uint(random&3)
			s.Power = min(int(random>>16&15), 13) + 4
			g.effect(index - 1)
		}
	}
	position := int(binary.BigEndian.Uint16(entry[2:])) >> 2
	x := position%40*8 + int(binary.BigEndian.Uint16(entry[4:]))
	y := position / 40
	offset := int(binary.BigEndian.Uint16(entry[6:]))
	size := binary.BigEndian.Uint16(entry[8:])
	width := 32
	if size&0x8000 != 0 {
		width = 16
	}
	g.addServiceSprite(ServiceSprite{x, y, width, int(size&0x7fff) + 1, offset})
}

func (g *Model) serviceBall() {
	s := &g.Service
	s.BallWait--
	if s.BallWait < 0 {
		s.BallWait = 1
		raw := g.Resources.Raw["service_sequences"]
		cursor := s.BallCursor
		for operations := 0; operations < 256; operations++ {
			command := raw[cursor]
			cursor++
			if int8(command) < 32 {
				s.BallY += int(int8(command))
				s.PowerStep++
				s.BallFlip ^= 64
				s.BallCursor = cursor
				break
			}
			if command < 96 {
				s.BallX += int(command) - 64
				continue
			}
			switch command {
			case 0x60:
				s.Phase = 2
			case 0x61:
				s.Phase = -1
				s.End = -1
				g.forceDrawServiceBall()
				return
			case 0x62:
				s.Phase = 1
				s.PowerStep = 0
				s.BallFrame = 0x3c94
			case 0x63:
				s.Phase = 0
				s.BallFrame = 0x3c54
			case 0x67:
				s.BallWait = 30
			case 0x65:
				s.BallHidden = true
			case 0x66:
				s.BallHidden = false
			default:
				s.BallCursor = serviceBallStart
				s.BallX = 220
				s.BallY = 164
				s.Tracks[0].Cursor = 0
				s.Tracks[1].Cursor = 0
				g.forceDrawServiceBall()
				return
			}
		}
	}
	g.drawServiceBall()
}

func (g *Model) drawServiceBall() {
	if !g.Service.BallHidden {
		g.forceDrawServiceBall()
	}
}

// Fault and reset commands branch directly into the original ball renderer;
// their draw bypasses the ordinary hidden-byte test without clearing that DATA.
func (g *Model) forceDrawServiceBall() {
	s := &g.Service
	g.addServiceSprite(ServiceSprite{s.BallX, s.BallY, 16, 8, s.BallFrame*2 + s.BallFlip})
}

func (g *Model) addServiceSprite(sprite ServiceSprite) {
	g.ServiceSprites[g.ServiceSpriteCount] = sprite
	g.ServiceSpriteCount++
}
