package render

import (
	"image"

	"github.com/olivierh59500/passingshot/internal/disk"
	"github.com/olivierh59500/passingshot/internal/game"
)

func (r *Renderer) GameOverIndices(view game.GameOverView) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	if view.Frame == 0 {
		return im.Pix
	}
	if !view.Opening {
		for _, pair := range []struct{ group, left, right int }{{3, 3, 4}, {2, 2, 5}, {1, 1, 3}, {0, 0, 6}} {
			x := view.Letters[pair.group].X
			composite(im, r.sprite("gameover_letters", pair.left*128, 16, 16, 0), x, 92, false)
			composite(im, r.sprite("gameover_letters", pair.right*128, 16, 16, 0), 301-x, 92, false)
		}
	}
	index := (view.Height >> 3 &^ 1) / 2
	heights := [16]int{2, 2, 4, 4, 6, 8, 10, 12, 14, 16, 20, 24, 24, 28, 28, 28}
	radii := [16]int{1, 1, 2, 2, 3, 4, 5, 6, 7, 8, 10, 12, 12, 14, 14, 14}
	for i, x := range []int{view.X, 320 - view.X} {
		offset := index*20 + int(view.Spin[i]&0x1c)
		frame := r.Resources.BallSprite(offset)
		composite(im, r.sprite(frame.Asset, frame.Offset, frame.Width, heights[index], 0), x-radii[index], 84, false)
	}
	return im.Pix
}

func (r *Renderer) GameOverRGBA(view game.GameOverView) *image.RGBA {
	pixels := r.GameOverIndices(view)
	palette := disk.Palette(r.Resources.Raw["gameover_palette"])
	im := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for i, index := range pixels {
		im.Set(i%Width, i/Width, palette[index])
	}
	return im
}
