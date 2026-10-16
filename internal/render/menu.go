package render

import (
	"encoding/binary"
	"image"

	"github.com/olivierh59500/passingshot/internal/disk"
	"github.com/olivierh59500/passingshot/internal/game"
)

func (r *Renderer) NameBackdropIndices(mask int) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.MenuPalette)
	for y := 0; y < 112; y++ {
		for x := 0; x < 256; x++ {
			im.Pix[y*Width+32+x] = r.Resources.Menu.ColorIndexAt(x, y)
		}
	}
	for y := 112; y < Height; y++ {
		for x := 32; x < 288; x++ {
			im.Pix[y*Width+x] = 5
		}
	}
	for person := 0; person < 2; person++ {
		if mask&(1<<person) == 0 {
			continue
		}
		r.menuPortrait(im, person, [2]int{32, 88}[person])
	}
	return im.Pix
}

func (r *Renderer) menuPortrait(im *image.Paletted, person, shift int) {
	r.portraitLayout(im, r.Resources.Raw["portraits_layout"][person*40:], shift)
}

func (r *Renderer) portraitLayout(im *image.Paletted, data []byte, shift int) {
	for piece := 0; piece < 5; piece++ {
		d := data[piece*8:]
		position := (int(binary.BigEndian.Uint16(d)) + shift) >> 2
		x := position%40*8 + int(binary.BigEndian.Uint16(d[2:]))
		y := position / 40
		size := binary.BigEndian.Uint16(d[4:])
		width := 32
		if size&0x8000 != 0 {
			width = 16
		}
		height := int(size&0x7fff) + 1
		offset := int(binary.BigEndian.Uint16(d[6:]))
		composite(im, r.sprite("portraits", offset, width, height, 1), x, y, false)
	}
}

func (r *Renderer) NameBackdropRGBA(mask int) *image.RGBA {
	return r.menuRGBA(r.NameBackdropIndices(mask))
}

// NameIndices composes the original two-player name form from the last
// presented editing state, including its transparent selector and glyphs.
func (r *Renderer) NameIndices(state game.NameEntryState) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.MenuPalette)
	base := r.NameBackdropIndices(3)
	copy(im.Pix, base)
	for i, view := range state.Presented {
		if view.Prompt {
			r.textStream(im, r.Resources.Raw["name_prompts"][i*25:], 0)
		}
		if !view.Grid {
			continue
		}
		if view.Selector {
			if r.nameSelector == nil {
				var err error
				r.nameSelector, err = disk.Interleaved(r.Resources.Raw["name_selector"], 16, 12, 2, r.Resources.MenuPalette)
				if err != nil {
					panic(err)
				}
			}
			composite(im, r.nameSelector, view.X, view.Y, false)
		}
		data := append([]byte(nil), r.Resources.Raw["name_entry_text"]...)
		data[1] = r.Resources.Raw["name_layout"][i*36+13]
		copy(data[90:94], view.Name[:])
		r.textStream(im, data, 0)
	}

	copy(im.Pix[199*Width:], base[199*Width:])
	return im.Pix
}

func (r *Renderer) menuRGBA(pixels []byte) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for i, index := range pixels {
		palette := r.Resources.MenuPalette
		if i >= 112*Width {
			palette = r.portraitPalette
		}
		im.Set(i%Width, i/Width, palette[index])
	}
	return im
}
