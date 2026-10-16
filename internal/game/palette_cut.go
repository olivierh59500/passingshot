package game

const NamePaletteCutTicks = 4

const NamePaletteSetupTicks = 1
const LevelPaletteCutTicks = 6
const LevelPaletteSetupTicks = 2

func (g *Model) beginPaletteCut(target Phase, ticks int) {
	g.paletteCutTarget, g.PaletteCutTicks = target, ticks
	if target == Names {

		g.requestSong(4)
	}
}

func (g *Model) stepPaletteCut() {
	g.PaletteCutTicks--
	if g.paletteCutTarget == LevelSelect && g.PaletteCutTicks == LevelPaletteCutTicks-1 {

		g.joystickLatch = [2]Control{}
	}
	if g.PaletteCutTicks != 0 {
		return
	}
	switch g.paletteCutTarget {
	case Names:
		g.prepareNames()
		g.PaletteSetupTicks = NamePaletteSetupTicks
	case LevelSelect:
		g.prepareLevelSelection()
		g.PaletteSetupTicks = LevelPaletteSetupTicks
	}
}

func (g *Model) stepPaletteSetup() {
	g.PaletteSetupTicks--
	if g.PaletteSetupTicks != 0 {
		return
	}
	switch g.Phase {
	case Names:
		g.NameEntry.Step(&g.joystickLatch, g.Resources.Raw["name_entry_text"])
	case LevelSelect:
		g.stepLevelSelection()
	}
}
