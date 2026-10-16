package game

import "encoding/binary"

// NamePlayerState translates one of the original 36-byte records. Coordinates
// and Selection refer to the original selector and text-data lookup.
type NamePlayerState struct {
	Status, Wait, X, Y, OriginX, Cursor, Selection int
	Blink                                          bool
	Bytes                                          [4]byte
}

// NameView keeps the last drawing operations independently of editing state.
// The original copies its hidden editing surface only at a completed loop step.
type NameView struct {
	Prompt, Grid, Selector bool
	X, Y                   int
	Name                   [4]byte
}

type NameEntryState struct {
	Players          [2]NamePlayerState
	Views, Presented [2]NameView
	First            int
	Complete         bool
}

func NewNameEntry(layout []byte) NameEntryState {
	s := NameEntryState{First: -1}
	for i := range s.Players {
		d := layout[i*36:]
		x := int(binary.BigEndian.Uint16(d[22:]))
		s.Players[i] = NamePlayerState{X: x, Y: 134, OriginX: x, Bytes: [4]byte{'A', '\\', '\\', '\\'}}
	}
	return s
}

func (s *NameEntryState) Step(input *[2]Control, text []byte) {
	if s.Complete {
		return
	}
	for i := range s.Players {
		p, other := &s.Players[i], &s.Players[1-i]
		// Completion can bypass the presentation copy. Preserve the prior
		// visible frame when the other record has not joined or has finished.
		if p.Status >= 0 && (p.Status == 0 && other.Status > 0 || p.Status > 0 && other.Status >= 0) {
			s.Complete = true
			return
		}
		if p.Status > 0 {
			continue
		}
		v := &s.Views[i]
		if p.Status == 0 {
			p.Wait--
			if p.Wait < 0 {
				p.Wait = 20
				p.Blink = !p.Blink
				v.Prompt = p.Blink
			}
			if input[i]&Fire != 0 {
				input[i] &^= Fire
				if s.First < 0 {
					s.First = i
				}
				p.Status, p.Wait, p.Blink = -1, 0, false
				v.Prompt = false
			}
			continue
		}
		p.Wait--
		if p.Wait < 0 {
			p.Wait = 4
			p.Blink = !p.Blink
			v.Selector = p.Blink
		}
		v.Grid = true
		v.Name = [4]byte{p.Bytes[0], p.Bytes[1], p.Bytes[2], '\\'}
		if !p.Blink {
			v.Name[p.Cursor] = '\\'
		}
		// The original tests the countdown after decrementing it. Movement
		// and confirmation reset it, so held directions repeat every two steps.
		if p.Wait&1 != 0 {
			dx, dy, moved := 0, 0, false
			if input[i]&Fire != 0 {
				input[i] &^= Fire
				switch p.Selection {
				case 0x40:
					if p.Cursor > 0 {
						p.Bytes[p.Cursor], v.Name[p.Cursor] = '\\', '\\'
						p.Cursor--
						moved = true
					}
				case 0x42:
					p.Status = 10
					p.Bytes[p.Cursor], v.Name[p.Cursor] = '\\', '\\'
					v.Selector = false
				default:
					p.Cursor++
					if p.Cursor >= 3 {
						p.Status = 10
						v.Selector = false
						v.Name = [4]byte{p.Bytes[0], p.Bytes[1], p.Bytes[2], '\\'}
					} else {
						moved = true
					}
				}
			} else {
				if input[i]&Up != 0 {
					if p.Y > 134 {
						dy = -8
						p.Selection -= 14
					} else {
						dy = 32
						p.Selection += 56
					}
				} else if input[i]&Down != 0 {
					if p.Y < 166 {
						dy = 8
						p.Selection += 14
					} else {
						dy = -32
						p.Selection -= 56
					}
				}
				if input[i]&Left != 0 {
					if p.X > p.OriginX {
						dx = -16
						p.Selection -= 2
					} else {
						dx = 80
						p.Selection += 10
					}
				} else if input[i]&Right != 0 {
					if p.X <= p.OriginX+72 {
						dx = 16
						p.Selection += 2
					} else {
						dx = -80
						p.Selection -= 10
					}
				}
				moved = dx != 0 || dy != 0
			}
			if moved {
				p.Wait, p.Blink = 0, false
				p.X += dx
				p.Y += dy
				v.Selector = true
				p.Bytes[p.Cursor] = text[19+p.Selection]
				v.Name[p.Cursor] = p.Bytes[p.Cursor]
			}
		}
		v.X, v.Y = p.X, p.Y
	}
	s.Presented = s.Views
}

// Pick adapts a pointer or typed character to an original grid cell. Joystick
// navigation continues through Step without this optional input adaptation.
func (p *NamePlayerState) Pick(cell int, text []byte) {
	cell = clamp(cell, 0, 29)
	p.X, p.Y = p.OriginX+cell%6*16, 134+cell/6*8
	p.Selection = cell/6*14 + cell%6*2
	if p.Cursor < 3 {
		p.Bytes[p.Cursor] = text[19+p.Selection]
	}
}

// BeginNames starts the original simultaneous waiting/joining interface.
func (g *Model) BeginNames() {
	g.beginNames()
	g.requestSong(4)
}

// beginNames initializes and presents the first waiting/joining form.
// Timed palette-cut entry uses prepareNames and defers this first step.
func (g *Model) beginNames() {
	g.prepareNames()

	g.NameEntry.Step(&g.joystickLatch, g.Resources.Raw["name_entry_text"])
}

func (g *Model) prepareNames() {
	g.Demo = false
	g.NameEntry = NewNameEntry(g.Resources.Raw["name_layout"])
	for i := range g.namePicks {
		g.namePicks[i] = g.namePicks[i][:0]
	}
	g.transition(Names)
}

func (g *Model) nameFocus() int {
	i := max(0, g.NameEntry.First)
	if g.NameEntry.Players[i].Status > 0 && g.NameEntry.Players[1-i].Status < 0 {
		i = 1 - i
	}
	return i
}

func (g *Model) updateNames(in Input, confirm, back bool) {
	for i := range g.NameEntry.Players {
		if in.NameJoin[i] {
			g.joystickLatch[i] |= Fire
		}
		if in.NamePickSet[i] {
			if g.NameEntry.Players[i].Status == 0 {
				g.joystickLatch[i] |= Fire
			} else if g.NameEntry.Players[i].Status < 0 {
				g.namePicks[i] = append(g.namePicks[i], clamp(in.NamePick[i], 0, 29))
			}
		}
	}
	focus := g.nameFocus()
	if confirm {
		g.joystickLatch[focus] |= Fire
	}
	if in.Letter >= 'A' && in.Letter <= 'Z' {
		if g.NameEntry.Players[focus].Status == 0 {
			g.joystickLatch[focus] |= Fire
		}
		g.namePicks[focus] = append(g.namePicks[focus], int(in.Letter-'A'))
	}
	if back {
		if g.NameEntry.First < 0 {
			g.ReturnToTitle()
			return
		}
		g.namePicks[focus] = append(g.namePicks[focus], 28)
	}
	if g.PhaseTick%2 != 0 {
		return
	}
	text := g.Resources.Raw["name_entry_text"]
	for i := range g.NameEntry.Players {
		p := &g.NameEntry.Players[i]
		if len(g.namePicks[i]) != 0 && p.Status < 0 && p.Wait > 0 && p.Wait&1 == 0 {
			p.Pick(g.namePicks[i][0], text)
			g.namePicks[i] = g.namePicks[i][1:]
			g.joystickLatch[i] |= Fire
		}
	}
	g.NameEntry.Step(&g.joystickLatch, text)
	if !g.NameEntry.Complete {
		return
	}
	g.Doubles = g.NameEntry.Players[0].Status != 0 && g.NameEntry.Players[1].Status != 0
	g.SinglesPlayer = 0
	if g.NameEntry.Players[0].Status == 0 {
		g.SinglesPlayer = 1
	}
	for i, p := range g.NameEntry.Players {
		name := append([]byte(nil), p.Bytes[:3]...)
		for n, ch := range name {
			if ch == '\\' {
				name[n] = ' '
			}
		}
		g.Names[i] = string(name)
	}
	g.beginPaletteCut(LevelSelect, LevelPaletteCutTicks)
}
