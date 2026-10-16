package render

import (
	"image"

	"github.com/olivierh59500/passingshot/internal/game"
)

// ServiceIndices composes original service sprites without display scaling.
// The blank option isolates sprite composition for external raster probes.
func (r *Renderer) ServiceIndices(g *game.Model, background bool) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	if background {
		copy(im.Pix, r.serviceBackdrop(g.Level).Pix)
		r.serviceScore(im, g)
		r.serviceNotice(im, g.ServiceNotice.Kind)
	}
	for i := 0; i < g.ServiceSpriteCount; i++ {
		s := g.ServiceSprites[i]
		sprite := r.sprite("objects", s.Offset, s.Width, s.Height, 0)
		if sprite == nil {
			panic("original service sprite outside reconstructed resources")
		}
		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				dx, dy := s.X+x, s.Y+y
				if dx < 0 || dx >= Width || dy < 0 || dy >= Height {
					continue
				}
				index := sprite.ColorIndexAt(x, y)
				if index != 0 {
					im.Pix[dy*Width+dx] = index
				}
			}
		}
	}
	return im.Pix
}

func (r *Renderer) serviceBackdrop(level int) *image.Paletted {
	return r.Resources.CourtBackdrops[min(level, 3)]
}

func (r *Renderer) ServiceNoticeIndices(g *game.Model) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	for i := range im.Pix {
		im.Pix[i] = 15
	}
	r.serviceNotice(im, g.ServiceNotice.Kind)
	return im.Pix
}

func (r *Renderer) serviceNotice(dst *image.Paletted, kind game.ServiceNoticeKind) {
	if kind != game.NoServiceNotice {
		offset := [4]int{0, 0, 15, 30}[kind]
		r.textStream(dst, r.Resources.Raw["service_notices"][offset:], 0)
	}
}

// serviceScore draws the current score using the extracted text layout.
func (r *Renderer) serviceScore(dst *image.Paletted, g *game.Model) {
	r.textStream(dst, r.serviceScoreData(g), 0)
}

func (r *Renderer) serviceScoreData(g *game.Model) []byte { return g.ServiceScoreText() }
