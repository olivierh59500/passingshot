package render

import (
	"encoding/binary"
	"image"

	"github.com/olivierh59500/passingshot/internal/game"
)

type pointLayout struct {
	frame, text      int
	names            [4]int
	near, far, width int
}

// PointScoreIndices isolates the live score-card compositor for external
// comparisons of the stage-selected names and original score formats.
func (r *Renderer) PointScoreIndices(g *game.Model) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	r.pointScore(im, g)
	return im.Pix
}

// PointIndices composes the original score frames, decision text, and result
// animation. The decision frame itself retains the preceding rally display.
func (r *Renderer) PointIndices(g *game.Model, background bool) []byte {
	var pixels []byte
	if g.Point.Fault != 0 {
		pixels = r.ServiceIndices(g, background)
	} else {
		pixels = r.RallyIndices(g, background)
	}
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	copy(im.Pix, pixels)
	if g.Point.Fault != 0 {
		offset := 0
		if g.Point.Fault == 2 {
			offset = 9
		}
		r.textStream(im, r.Resources.Raw["fault_messages"][offset:], 0)
		if g.Point.Fault == 2 {
			r.pointScore(im, g)
			if g.Point.GameEnded && g.Point.Frame > 0 {
				// Unlike the rally loop, the fault loop does not redraw the
				// background. Transparent face pixels retain earlier frames
				// from the same alternating display buffer.
				for i := (len(g.FaultFrames) - 1) & 1; i < len(g.FaultFrames); i += 2 {
					c := g.Celebration
					c.Frame = g.FaultFrames[i]
					r.celebration(im, c)
				}
			}
		}
	} else if g.Point.Frame > 0 {
		r.pointScore(im, g)
		if g.Point.GameEnded {
			r.celebration(im, g.Celebration)
		}
		offsets := [9]int{0, 0, 7, 21, 36, 50, 57, 67, 74}
		r.textStream(im, r.Resources.Raw["point_messages"][offsets[g.Point.MessageIndex]:], 0)
	}
	return im.Pix
}

func (r *Renderer) pointScore(dst *image.Paletted, g *game.Model) {
	s := g.Score
	var layout pointLayout
	if g.Point.GameEnded {
		layout = pointLayout{0x0, 0x56, [4]int{4, -1, 13, -1}, 9, 18, 1}
		if g.Doubles {
			layout = pointLayout{0x0, 0x6a, [4]int{3, 14, 20, 31}, 9, 26, 1}
		}
	} else if s.TieBreak {
		layout = pointLayout{0x2cc, 0x329, [4]int{4, -1, 14, -1}, 9, 19, 2}
		if g.Doubles {
			layout = pointLayout{0x2cc, 0x33f, [4]int{3, 15, 21, 33}, 9, 27, 2}
		}
	} else if s.Points[0] < 3 || s.Points[0] == 3 && s.Points[1] < 3 {
		layout = pointLayout{0x8d, 0xea, [4]int{4, -1, 14, -1}, 9, 19, 2}
		if g.Doubles {
			layout = pointLayout{0x8d, 0x100, [4]int{3, 15, 21, 33}, 9, 27, 2}
		}
	} else if s.Points == [2]int{3, 3} {
		layout = pointLayout{0x125, 0x197, [4]int{4, -1, 17, -1}, 0, 0, 0}
		if g.Doubles {
			layout = pointLayout{0x125, 0x1b3, [4]int{3, 18, 24, 39}, 0, 0, 0}
		}
	} else {
		layout = pointLayout{0x1de, 0x275, [4]int{4, -1, 21, -1}, 9, 26, 9}
		if g.Doubles {
			layout = pointLayout{0x1de, 0x299, [4]int{3, 22, 28, 47}, 9, 34, 9}
		}
	}
	data := append([]byte(nil), r.pointText(layout.text)...)
	// Stage initialization selects the far-side names; the near-side names
	// are the three characters entered through the original name form.
	names := [4]string{g.Names[0], g.Names[1], g.Opponents[0], g.Opponents[1]}
	if !g.Doubles && g.SinglesPlayer == 1 {
		names[0], names[2] = names[1], names[3]
	}
	for i, offset := range layout.names {
		if offset < 0 {
			continue
		}
		for n := 0; n < 3; n++ {
			ch := byte('\\')
			if n < len(names[i]) && names[i][n] != ' ' {
				ch = names[i][n]
			}
			data[offset+n] = ch
		}
	}
	switch layout.width {
	case 1:
		data[layout.near], data[layout.far] = byte('0'+s.Games[0]), byte('0'+s.Games[1])
	case 2:
		table := r.Resources.Raw["normal_point_text"]
		if s.TieBreak {
			table = r.Resources.Raw["tie_point_text"]
		}
		copy(data[layout.near:layout.near+2], table[s.Points[0]*2:s.Points[0]*2+2])
		copy(data[layout.far:layout.far+2], table[s.Points[1]*2:s.Points[1]*2+2])
	case 9:
		loser := layout.far
		if s.Points[0] == 3 {
			loser = layout.near
		}
		for i := 0; i < 9; i++ {
			data[loser+i] = '\\'
		}
	}

	r.textStream(dst, data, 0)
	r.textStream(dst, r.pointText(layout.frame), 1)
}

func (r *Renderer) pointText(address int) []byte {
	data := r.Resources.Raw["point_score_text"][address:]
	for i, b := range data {
		if b == 0 {
			return data[:i+1]
		}
	}
	panic("unterminated original score text")
}

func (r *Renderer) textStream(dst *image.Paletted, data []byte, font int) {
	base := byte('0')
	if font == 1 {
		base = 'A'
	}
	for _, work := range TextLayout(data, font) {
		glyph := r.Resources.Fonts[font][int(work.Code-base)]
		for gy := 0; gy < 8; gy++ {
			for gx := 0; gx < 8; gx++ {
				index := glyph.ColorIndexAt(gx, gy)
				if index == 0 {
					continue
				}
				for repeat := 0; repeat < work.Rows; repeat++ {
					dx, dy := work.X+gx, work.Y+gy*work.Rows+repeat
					if dx >= 0 && dx < Width && dy >= 0 && dy < Height {
						dst.Pix[dy*dst.Stride+dx] = index
					}
				}
			}
		}
	}
}

func (r *Renderer) celebration(dst *image.Paletted, c game.Celebration) {
	data := r.Resources.Raw["celebration_decorations"]
	for i := 0; i < len(data); i += 4 {
		dest, source := int(binary.BigEndian.Uint16(data[i:])), binary.BigEndian.Uint16(data[i+2:])
		x, y := dest%40*8, dest/40
		sprite := r.sprite("level_art", int(source&0x7fff), 16, 16, 0)
		composite(dst, sprite, x, y, source&0x8000 != 0)
	}
	composite(dst, r.sprite("level_art", c.Frame, 32, 48, 0), 56, 136, false)
	composite(dst, r.sprite("level_art", 0x300+c.Frame, 16, 44, 0), 88, 138, false)
}
