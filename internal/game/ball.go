// Package game implements deterministic native tennis simulation.
package game

type Ball struct {
	X, Y, Height                                            int
	Range, Horizontal, XAccumulator                         int
	DirectionX, DirectionY                                  int
	CurveIndex, CurveAccumulator, CurveRange, HeightDivisor int
	SpeedFixed                                              int
	Bounces                                                 int
	Stopped                                                 bool
}

func (b *Ball) StepMotion() {
	b.XAccumulator += b.Horizontal
	if b.XAccumulator >= b.Range {
		b.XAccumulator -= b.Range
		b.X += b.DirectionX
	}
	b.Y += b.DirectionY
}

func (b *Ball) StepArc(curve []byte) bool {
	if b.CurveRange == 0 {
		b.SpeedFixed &= 0xffff
		b.Stopped = true
		return false
	}
	for {
		b.CurveIndex++
		b.CurveAccumulator += b.CurveRange
		if b.CurveAccumulator >= 256 {
			break
		}
	}
	b.CurveAccumulator -= 256
	if b.CurveIndex >= len(curve) {
		b.CurveAccumulator = 0
		b.CurveRange >>= 1
		b.HeightDivisor *= 2
		b.SpeedFixed -= 0xa000
		if b.SpeedFixed < 0 {
			b.SpeedFixed = 0
		}
		if b.SpeedFixed>>16 == 0 {
			b.Stopped = true
		}
		b.CurveIndex = 0
		b.Height = 0
		b.Bounces++
		return true
	}
	b.Height = int(curve[b.CurveIndex]) * 4 / b.HeightDivisor
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func sign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
