package render

import (
	"encoding/binary"
	"image"

	"github.com/olivierh59500/passingshot/internal/game"
)

func (r *Renderer) RallyIndices(g *game.Model, background bool) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	if background {
		r.courtPixels(im, g)
	}
	r.rallySprites(im, g)
	return im.Pix
}

func (r *Renderer) courtPixels(dst *image.Paletted, g *game.Model) {
	level := min(g.Level, 3)
	name := [4]string{"france", "australia", "usa", "england"}[level]
	mapping := r.Resources.Raw[name+"_net"]
	// The original has eight preshifted tile atlases and ignores the low
	// horizontal camera bit. Sprites still subtract the exact camera word.
	shift, column := g.CameraX&14, g.CameraX>>4
	tiles := r.Resources.CourtTiles[level][shift>>1]
	for y := 0; y < Height; y++ {
		worldY := y + g.CameraY
		if worldY < 0 || worldY >= 600 {
			continue
		}
		for x := 0; x < Width; x += 16 {
			mapX := column + x/16
			if mapX < 0 || mapX >= 40 {
				continue
			}
			offset := (worldY/8*40 + mapX) * 2
			row := int(binary.BigEndian.Uint16(mapping[offset:])) / 64
			src := (row*8 + worldY%8) * tiles.Stride
			copy(dst.Pix[y*dst.Stride+x:y*dst.Stride+x+16], tiles.Pix[src:src+16])
		}
	}
}

// RallyIndicesOn exposes constant backgrounds for external transparency probes.
func (r *Renderer) RallyIndicesOn(g *game.Model, fill byte) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.CourtPalette)
	for i := range im.Pix {
		im.Pix[i] = fill
	}
	r.rallySprites(im, g)
	return im.Pix
}

func (r *Renderer) rallySprites(im *image.Paletted, g *game.Model) {
	players := func() {
		for i := 3; i >= 0; i-- {
			p := g.Players[i]
			if !p.Active || g.BlinkPlayer == i+1 && g.MatchFrame&1 == 0 {
				continue
			}
			f := r.Resources.PlayerFrame(p.FrameTable, p.FrameIndex)
			shadow := r.sprite("players", 0x12480+f.Shadow, 16, 10, 0)
			composite(im, shadow, p.X+f.X+f.RacketX-g.CameraX, p.Y+f.Y+f.RacketY-g.CameraY, false)
		}
		for i := 3; i >= 0; i-- {
			p := g.Players[i]
			if !p.Active || g.BlinkPlayer == i+1 && g.MatchFrame&1 == 0 {
				continue
			}
			f := r.Resources.PlayerFrame(p.FrameTable, p.FrameIndex)
			sprite := r.playerSprite(p.Bank*0x4920+int(f.Address), f.Width, f.Height, 0, g.PlayerFirstWord)
			// Bit 30 selects bottom-up source rows, not horizontal mirroring.
			composite(im, sprite, p.X+f.X-g.CameraX, p.Y+f.Y-g.CameraY, f.ReverseRows)
		}
	}
	if g.Ball.Height <= 20 {
		r.rallyBall(im, g)
		players()
	} else {
		players()
		r.rallyBall(im, g)
	}
}

func (r *Renderer) rallyBall(dst *image.Paletted, g *game.Model) {
	if g.RallyState().Flags&2 != 0 {
		return
	}
	mark := g.BounceMark
	if mark.Draw {
		sprite := r.sprite("ball_art", mark.Remaining, 16, 4, 0)

		composite(dst, sprite, mark.X-8-g.CameraX, mark.Y-g.CameraY, false)
	}
	b := g.Ball
	index := clamp(b.Height>>3&^1, 0, 30) / 2
	heights := [16]int{2, 2, 4, 4, 6, 8, 10, 12, 14, 16, 20, 24, 24, 28, 28, 28}
	radii := [16]int{1, 1, 2, 2, 3, 4, 5, 6, 7, 8, 10, 12, 12, 14, 14, 14}
	off := index*20 + int(g.BallSpin&0x1c)
	frame := r.Resources.BallSprite(off)
	sprite := r.sprite(frame.Asset, frame.Offset, frame.Width, heights[index], 0)
	if sprite == nil {
		panic("invalid original ball sprite")
	}
	x, y := b.X-g.CameraX, b.Y-g.CameraY
	composite(dst, sprite, x-radii[index], y, false)
	// The original places this alternating mask after the ball itself, even
	// when the two overlap, and includes the smallest low-ball shadow.
	for sy := 0; sy < sprite.Rect.Dy(); sy++ {
		for sx := 0; sx < sprite.Rect.Dx(); sx++ {
			dx, dy := x-radii[index]*2+sx, y+radii[index]*2+sy
			if dx >= 0 && dx < Width && dy >= 0 && dy < Height && (sx+sy)&1 == 0 && sprite.ColorIndexAt(sx, sy) != 0 {
				dst.Pix[dy*Width+dx] = 14
			}
		}
	}
}

func composite(dst, src *image.Paletted, x, y int, reverseRows bool) {
	if src == nil {
		panic("original sprite is unavailable")
	}
	for dy := 0; dy < src.Rect.Dy(); dy++ {
		targetY := y + dy
		if targetY < 0 || targetY >= dst.Rect.Dy() {
			continue
		}
		sy := dy
		if reverseRows {
			sy = src.Rect.Dy() - 1 - dy
		}
		for dx := 0; dx < src.Rect.Dx(); dx++ {
			targetX := x + dx
			if targetX < 0 || targetX >= dst.Rect.Dx() {
				continue
			}
			index := src.ColorIndexAt(dx, sy)
			if index != 0 {
				dst.Pix[targetY*dst.Stride+targetX] = index
			}
		}
	}
}
