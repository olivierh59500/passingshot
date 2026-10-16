// Package render reconstructs the original indexed artwork into a native frame.
// The software surface also supports captures and checks without a display.
package render

import (
	"image"
	"image/color"
	"strconv"

	"github.com/olivierh59500/passingshot/internal/disk"
	"github.com/olivierh59500/passingshot/internal/game"
	"github.com/olivierh59500/passingshot/internal/resources"
)

const Width, Height = 320, 200

type spriteKey struct {
	asset                  string
	offset                 int
	width, height, palette int
	playerWord             uint32
}

type Renderer struct {
	nameSelector     *image.Paletted
	levelMenuBase    [3]*image.Paletted
	introductionBase [6][3]*image.Paletted
	Resources        *resources.Set
	Frame            *image.RGBA
	controllerFrame  *image.RGBA
	menu             *image.RGBA
	cache            map[spriteKey]*image.Paletted
	portraitPalette  color.Palette
}

func New(r *resources.Set) *Renderer {
	v := &Renderer{Resources: r, Frame: image.NewRGBA(image.Rect(0, 0, Width, Height)), cache: map[spriteKey]*image.Paletted{}, portraitPalette: disk.Palette(r.Raw["portrait_palettes"][:32])}

	v.portraitPalette[0] = color.RGBA{0, 0, 0, 255}
	v.buildMenu()
	return v
}

func (r *Renderer) Draw(g *game.Model) *image.RGBA {
	r.fill(image.Rect(0, 0, Width, Height), color.NRGBA{0, 0, 0, 255})
	if g.PaletteCutTicks > 0 {
		return r.Frame
	}
	switch g.Phase {
	case game.Title:
		r.blit(r.Resources.Title, 0, 0, r.Resources.TitlePalette, false, false)
	case game.Names:
		copy(r.Frame.Pix, r.menuRGBA(r.NameIndices(g.NameEntry)).Pix)
	case game.LevelSelect:
		copy(r.Frame.Pix, r.LevelMenuRGBA(g.LevelMenu, g.PlayerMode()).Pix)
	case game.SetResult:
		if g.Score.Winner == 0 && !g.Demo {
			copy(r.Frame.Pix, r.TournamentResultRGBA(g.Tournament, g.PlayerMode()).Pix)
		} else {
			r.drawCourt(g)
			r.panel(20, 48, 280, 103, r.Resources.MenuPalette)
			r.center(g.Message, 65, r.Resources.MenuPalette)
			r.center(strconv.Itoa(g.Score.Games[0])+"  TO  "+strconv.Itoa(g.Score.Games[1]), 88, r.Resources.MenuPalette)
			r.center("PRESS START BUTTON", 129, r.Resources.MenuPalette)
		}
	case game.Champion:
		copy(r.Frame.Pix, r.TournamentResultRGBA(g.Tournament, g.PlayerMode()).Pix)
	case game.Introduction:
		copy(r.Frame.Pix, r.IntroductionRGBA(g.Level, g.PlayerMode()).Pix)
	case game.GameOver:
		copy(r.Frame.Pix, r.GameOverRGBA(g.GameOver.Presented).Pix)
	case game.Serve:
		r.presentCourtPixels(r.ServiceIndices(g, true))
	case game.Rally:
		// The original rally loop draws the scrolling court and actors. Score
		// presentation belongs to the close-up service and point sequences.
		r.presentCourtPixels(r.RallyIndices(g, true))
	case game.PointResult:
		r.presentCourtPixels(r.PointIndices(g, true))
	default:
		r.presentCourtPixels(r.RallyIndices(g, true))
		r.score(g)
		if g.Message != "" {
			r.panel(32, 84, 256, 24, r.Resources.CourtPalette)
			r.center(g.Message, 92, r.Resources.CourtPalette)
		}
	}
	if g.Demo {
		r.text("DEMO", 144, 185, r.Resources.CourtPalette)
	}
	if g.Paused {
		r.dim(2)
		r.panel(48, 60, 224, 80, r.Resources.MenuPalette)
		r.center("PAUSED", 70, r.Resources.MenuPalette)
		r.center("FIRE OR P TO RESUME", 91, r.Resources.MenuPalette)
		r.center("R RESTART  Q MENU", 115, r.Resources.MenuPalette)
	}
	return r.Frame
}

func (r *Renderer) drawCourt(g *game.Model) {
	index := min(g.Level, 3)
	court := r.Resources.Courts[index]
	p := r.Resources.CourtPalette
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			cx, cy := x+g.CameraX, y+g.CameraY
			if image.Pt(cx, cy).In(court.Rect) {
				r.Frame.Set(x, y, p[court.ColorIndexAt(cx, cy)])
			}
		}
	}
}

func (r *Renderer) presentCourtPixels(pixels []byte) {
	var colors [16][4]byte
	for i, c := range r.Resources.CourtPalette {
		red, green, blue, alpha := c.RGBA()
		colors[i] = [4]byte{byte(red >> 8), byte(green >> 8), byte(blue >> 8), byte(alpha >> 8)}
	}
	for i, index := range pixels {
		c := colors[index]
		copy(r.Frame.Pix[i*4:i*4+4], c[:])
	}
}

func (r *Renderer) score(g *game.Model) {
	p := r.Resources.CourtPalette
	r.panel(2, 2, 128, 21, p)
	r.text(g.Names[0]+" "+strconv.Itoa(g.Score.Games[0])+" "+g.Score.PointText(0), 6, 5, p)
	r.text("HIM "+strconv.Itoa(g.Score.Games[1])+" "+g.Score.PointText(1), 6, 14, p)
	if g.Score.TieBreak {
		r.text("TIE BREAK", 226, 5, p)
	} else {
		r.text("LEVEL "+strconv.Itoa(g.Level+1), 250, 5, p)
	}
}

func (r *Renderer) buildMenu() {
	r.menu = r.NameBackdropRGBA(3)
	copy(r.Frame.Pix, r.menu.Pix)
}

func (r *Renderer) sprite(asset string, offset, width, height, palette int) *image.Paletted {
	return r.spriteWithWord(asset, offset, width, height, palette, nil)
}

func (r *Renderer) playerSprite(offset, width, height, palette int, word uint16) *image.Paletted {
	return r.spriteWithWord("players", offset, width, height, palette, &word)
}

func (r *Renderer) spriteWithWord(asset string, offset, width, height, palette int, word *uint16) *image.Paletted {
	key := spriteKey{asset: asset, offset: offset, width: width, height: height, palette: palette, playerWord: 1 << 16}
	if word != nil {
		key.playerWord = uint32(*word)
	}
	if im, ok := r.cache[key]; ok {
		return im
	}
	p := r.Resources.CourtPalette
	if palette == 1 {
		p = r.portraitPalette
	}
	var im *image.Paletted
	if word == nil {
		im = r.Resources.Sprite(asset, offset, width, height, p)
	} else {
		im = r.Resources.SpriteWithPlayerWord(offset, width, height, p, *word)
	}
	r.cache[key] = im
	return im
}

func (r *Renderer) blit(im *image.Paletted, x, y int, p color.Palette, transparent, mirror bool) {
	if im == nil {
		return
	}
	for sy := 0; sy < im.Rect.Dy(); sy++ {
		if y+sy < 0 || y+sy >= Height {
			continue
		}
		for sx := 0; sx < im.Rect.Dx(); sx++ {
			tx := x + sx
			if tx < 0 || tx >= Width {
				continue
			}
			sourceX := sx
			if mirror {
				sourceX = im.Rect.Dx() - 1 - sx
			}
			c := im.ColorIndexAt(sourceX, sy)
			if transparent && c == 0 {
				continue
			}
			if int(c) < len(p) {
				r.Frame.Set(tx, y+sy, p[c])
			}
		}
	}
}

func (r *Renderer) text(s string, x, y int, p color.Palette) {
	for _, ch := range s {
		switch ch {
		case ' ':
			ch = '\\'
		case '.':
			ch = '['
		case '-':
			ch = '='
		case '/':
			ch = ']'
		}
		index := int(ch - '0')
		if index >= 0 && index < len(r.Resources.Fonts[0]) {
			r.blit(r.Resources.Fonts[0][index], x, y, p, true, false)
		}
		x += 8
	}
}
func (r *Renderer) center(s string, y int, p color.Palette) { r.text(s, (Width-len(s)*8)/2, y, p) }
func (r *Renderer) fill(rect image.Rectangle, c color.Color) {
	rect = rect.Intersect(r.Frame.Rect)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r.Frame.Set(x, y, c)
		}
	}
}
func (r *Renderer) pixel(x, y int, c color.Color) {
	if image.Pt(x, y).In(r.Frame.Rect) {
		r.Frame.Set(x, y, c)
	}
}
func (r *Renderer) rect(rect image.Rectangle, c color.Color) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		r.pixel(x, rect.Min.Y, c)
		r.pixel(x, rect.Max.Y-1, c)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		r.pixel(rect.Min.X, y, c)
		r.pixel(rect.Max.X-1, y, c)
	}
}
func (r *Renderer) panel(x, y, w, h int, p color.Palette) {
	r.fill(image.Rect(x, y, x+w, y+h), p[0])
	r.rect(image.Rect(x, y, x+w, y+h), p[15])
}
func (r *Renderer) dim(divisor uint8) {
	for i := 0; i < len(r.Frame.Pix); i += 4 {
		r.Frame.Pix[i] /= divisor
		r.Frame.Pix[i+1] /= divisor
		r.Frame.Pix[i+2] /= divisor
	}
}
func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }
