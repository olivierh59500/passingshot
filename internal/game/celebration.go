package game

import "encoding/binary"

type Celebration struct {
	Cursor, Wait, Repeats, Frame int
	Ended                        bool
}

// StartCelebration chooses the original sequence from the winner's game count
// and lead. The previous frame is preserved until the first animation update.
func (g *Model) StartCelebration(winner int) {
	games := g.Score.Games[winner]
	choice := 6
	if games < 6 {
		choice = max(0, games-g.Score.Games[1-winner])
	}
	index := choice
	if winner == 0 {
		index += 7
	}
	address := binary.BigEndian.Uint32(g.Resources.Raw["celebration_choices"][index*4:])
	g.Celebration.Cursor = int(address) - 0x7d06
	g.Celebration.Wait = 0
	g.Celebration.Repeats = 0
	g.Celebration.Ended = false
}

// StepCelebration preserves the original hold, repeat, and terminal behavior.
// A frame lasts three updates: its selection and two countdown updates.
func (g *Model) StepCelebration() {
	c := &g.Celebration
	if c.Wait != 0 {
		c.Wait--
		return
	}
	data := g.Resources.Raw["celebration_sequences"]
	for {
		command := binary.BigEndian.Uint16(data[c.Cursor:])
		switch command {
		case 0xffff:
			c.Ended = true
			return
		case 0xfffe:
			c.Wait = 100
			c.Cursor += 2
			return
		case 0xfffd:
			if c.Repeats == 0 {
				c.Repeats = 10
			} else {
				c.Repeats--
				if c.Repeats == 0 {
					c.Ended = true
					return
				}
			}
			back := int(binary.BigEndian.Uint16(data[c.Cursor+2:]))
			c.Cursor -= back * 2
		default:
			c.Frame = int(command)
			c.Wait = 2
			c.Cursor += 2
			return
		}
	}
}
