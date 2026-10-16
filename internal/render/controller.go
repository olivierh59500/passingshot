package render

import (
	"image"
	"image/color"
	"math"

	"github.com/olivierh59500/passingshot/internal/game"
	"github.com/olivierh59500/passingshot/internal/input"
)

// ControllerFrame draws application controls in the side gutters. Geometry is
// native UI; text uses only the original reconstructed game glyphs.
func (r *Renderer) ControllerFrame(layout input.ControllerLayout, state input.ControllerState, g *game.Model) *image.RGBA {
	phase, paused := g.Phase, g.Paused
	if r.controllerFrame == nil || r.controllerFrame.Rect.Dx() != layout.Width || r.controllerFrame.Rect.Dy() != layout.Height {
		r.controllerFrame = image.NewRGBA(image.Rect(0, 0, layout.Width, layout.Height))
	}
	im := r.controllerFrame
	clear(im.Pix)
	if phase < game.Introduction {
		return im
	}
	base, edge, active := color.RGBA{28, 36, 52, 255}, color.RGBA{210, 220, 235, 255}, color.RGBA{245, 190, 60, 255}
	circle := func(cx, cy, radius float64, fill color.RGBA) {
		for y := int(cy-radius) - 1; y <= int(cy+radius)+1; y++ {
			for x := int(cx-radius) - 1; x <= int(cx+radius)+1; x++ {
				d := math.Hypot(float64(x)+.5-cx, float64(y)+.5-cy)
				if d <= radius {
					c := fill
					if d > radius-2 {
						c = edge
					}
					im.SetRGBA(x, y, c)
				}
			}
		}
	}
	circle(layout.PadX, layout.PadY, layout.PadRadius, base)
	triangle := func(cx, cy, dx, dy float64, on bool) {
		p := [3][2]float64{{cx + dx*.80, cy + dy*.80}, {cx + dx*.35 - dy*.21, cy + dy*.35 + dx*.21}, {cx + dx*.35 + dy*.21, cy + dy*.35 - dx*.21}}
		e := func(a, b [2]float64, x, y float64) float64 { return (x-a[0])*(b[1]-a[1]) - (y-a[1])*(b[0]-a[0]) }
		c := edge
		if on && !paused {
			c = active
		}
		for y := int(cy - layout.PadRadius); y <= int(cy+layout.PadRadius); y++ {
			for x := int(cx - layout.PadRadius); x <= int(cx+layout.PadRadius); x++ {
				a, b, d := e(p[0], p[1], float64(x)+.5, float64(y)+.5), e(p[1], p[2], float64(x)+.5, float64(y)+.5), e(p[2], p[0], float64(x)+.5, float64(y)+.5)
				if a >= 0 && b >= 0 && d >= 0 || a <= 0 && b <= 0 && d <= 0 {
					im.SetRGBA(x, y, c)
				}
			}
		}
	}
	for _, dir := range []struct {
		dx, dy  float64
		control game.Control
	}{{0, -1, game.Up}, {0, 1, game.Down}, {-1, 0, game.Left}, {1, 0, game.Right}} {
		triangle(layout.PadX, layout.PadY, dir.dx*layout.PadRadius, dir.dy*layout.PadRadius, state.Direction&dir.control != 0)
	}
	hit := base
	if state.HitDown || state.ServePending {
		hit = active
	}
	circle(layout.HitX, layout.HitY, layout.HitRadius, hit)
	pause := base
	if state.PauseDown {
		pause = active
	}
	circle(layout.PauseX, layout.PauseY, layout.PauseRadius, pause)
	text := func(s string, x, y int) {
		for _, ch := range s {
			glyph := r.Resources.Fonts[0][int(ch-'0')]
			for gy := 0; gy < 8; gy++ {
				for gx := 0; gx < 8; gx++ {
					index := glyph.ColorIndexAt(gx, gy)
					if index != 0 {
						im.Set(x+gx, y+gy, r.Resources.CourtPalette[index])
					}
				}
			}
			x += 8
		}
	}
	label := "HIT"
	if phase == game.Serve {
		label = "SERVE"
		if g.Service.Started || state.ServePending {
			label = "SERVING"
		}
		if g.Service.Config.Far {
			label = "WAIT"
		}
	}
	if phase == game.Introduction || phase >= game.SetResult {
		label = "START"
	}
	if g.AutomaticPresentation() {
		label = "WAIT"
	}
	if paused {
		label = "PLAY"
	}
	text(label, int(layout.HitX)-len(label)*4, int(layout.HitY)-4)
	text("P", int(layout.PauseX)-4, int(layout.PauseY)-4)
	text("MOVE", int(layout.PadX)-16, int(layout.PadY+layout.PadRadius)+6)
	return im
}
