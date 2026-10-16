package game

// DefaultSeed gives normal games a repeatable starting random sequence.
const DefaultSeed uint32 = 0x0ff4c95b

// TitleTicks is the title timeout in 50 Hz game updates. Fire skips it.
const TitleTicks = 414

// MenuInput exposes a copy of the original mutable joystick latches for
// development state comparisons, without changing their consumption.
func (g *Model) MenuInput() [2]Control { return g.joystickLatch }
