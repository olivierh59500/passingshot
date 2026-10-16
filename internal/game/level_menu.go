package game

import "encoding/binary"

type LevelMenuState struct {
	Selection, Animation, Wait, Shift int
	Cycle                             uint16
	Blink, Prompt, Complete           bool
	Palette                           [16]uint16
	MarkerOrder                       [4][4]uint8
	MarkerCount                       [4]uint8
}

func NewLevelMenu(palette []byte, mode, first, wait int, blink bool) LevelMenuState {
	s := LevelMenuState{Wait: wait, Blink: blink, Shift: 14}
	if mode == 0 {
		s.Shift = 20
		if first == 1 {
			s.Shift = 6
		}
	}
	for i := range s.Palette {
		s.Palette[i] = binary.BigEndian.Uint16(palette[i*2:])
	}
	return s
}

// Step consumes the selected player's mutable joystick latch in the original
// palette, prompt, fire, direction, and marker-drawing order.
func (s *LevelMenuState) Step(input *Control, colors []byte) {
	if s.Complete {
		return
	}
	s.Animation += 0x1b0
	if s.Animation >= 0x6c0 {
		s.Animation = 0
	}
	s.Cycle++
	s.Palette[2], s.Palette[3] = 0xeee, 0xeee
	if s.Cycle&2 != 0 {
		s.Palette[2] = 0xe00
	}
	if s.Cycle&4 != 0 {
		s.Palette[3] = 0x00e
	}
	if s.Cycle&1 == 0 {
		s.Palette[4], s.Palette[5], s.Palette[6] = s.Palette[5], s.Palette[6], s.Palette[4]
		s.Palette[7], s.Palette[8], s.Palette[9] = s.Palette[9], s.Palette[7], s.Palette[8]
	}
	s.Wait--
	if s.Wait < 0 {
		s.Wait = 20
		s.Blink = !s.Blink
		s.Prompt = s.Blink
	}
	if *input&Fire != 0 {
		*input &^= Fire
		s.Complete = true
		return
	}
	// Direction handling leaves the old selection in the drawing register.
	// A move restores that row to frame zero before the new row animates.
	drawRow := s.Selection
	moved := false
	if *input&Up != 0 {
		*input &^= Up
		if s.Selection > 0 {
			s.Selection--
			moved = true
		}
	} else if *input&Down != 0 {
		*input &^= Down
		if s.Selection < 3 {
			s.Selection++
			moved = true
		}
	}
	if moved {
		s.Animation = 0
	} else {
		s.Palette[1] = binary.BigEndian.Uint16(colors[s.Selection*4:])
	}
	order, count := &s.MarkerOrder[drawRow], &s.MarkerCount[drawRow]
	frame := uint8(s.Animation / 0x1b0)
	for i := uint8(0); i < *count; i++ {
		if order[i] == frame {
			copy(order[i:], order[i+1:*count])
			*count--
			break
		}
	}
	order[*count] = frame
	*count++
}

// PlayerMode is the original portrait and gameplay mode selected by the names.
func (g *Model) PlayerMode() int {
	if g.Doubles {
		return 0
	}
	if g.SinglesPlayer == 1 {
		return 8
	}
	return 4
}

// BeginLevelSelection enters the original list and presents its first loop
// immediately. The first joined player's joystick owns this menu.
func (g *Model) BeginLevelSelection() {
	g.joystickLatch = [2]Control{}
	g.prepareLevelSelection()
	g.stepLevelSelection()
}

func (g *Model) prepareLevelSelection() {
	g.LevelMenu = NewLevelMenu(g.Resources.Raw["level_menu_palette"], g.PlayerMode(),
		max(0, g.NameEntry.First), g.LevelMenu.Wait, g.LevelMenu.Blink)
	g.levelPickPending = false
	g.Level = 0
	g.transition(LevelSelect)
}

func (g *Model) stepLevelSelection() {
	input := &g.joystickLatch[max(0, g.NameEntry.First)]
	g.LevelMenu.Step(input, g.Resources.Raw["level_menu_colors"])
	g.Level = g.LevelMenu.Selection
	if g.LevelMenu.Complete {
		g.InitialLevel = g.Level
		g.startSet()
	}
}

func (g *Model) updateLevelSelection(in Input, confirm, back bool) {
	if back {
		g.BeginNames()
		return
	}
	controller := max(0, g.NameEntry.First)
	if in.SelectionSet {
		g.levelPick, g.levelPickPending = clamp(in.Selection, 0, 3), true
	} else if confirm {
		g.joystickLatch[controller] |= Fire
	}
	if g.PhaseTick%2 != 0 {
		return
	}
	input := &g.joystickLatch[controller]
	if g.levelPickPending {
		// A pointer pick walks the same original transitions, then confirms.
		// It cannot confirm the previous row before the requested row is reached.
		*input &^= Up | Down | Fire
		switch {
		case g.levelPick < g.LevelMenu.Selection:
			*input |= Up
		case g.levelPick > g.LevelMenu.Selection:
			*input |= Down
		default:
			*input |= Fire
			g.levelPickPending = false
		}
	}
	g.stepLevelSelection()
}
