package render

import (
	"encoding/binary"
	"image"
	"image/color"

	"github.com/olivierh59500/passingshot/internal/disk"
	"github.com/olivierh59500/passingshot/internal/game"
)

func (r *Renderer) LevelMenuBaseIndices(mode int) []byte {
	index := mode / 4
	if r.levelMenuBase[index] == nil {
		im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.MenuPalette)
		for row := 0; row < 84; row++ {
			offset := 0x4bc + row*40
			r.levelMenuSprite(im, 0x5b8, offset, 8, 1)
			for col := 1; col <= 5; col++ {
				r.levelMenuSprite(im, 0x5c0, offset+col*2, 8, 1)
			}
			r.levelMenuSprite(im, 0x5c8, offset+12, 8, 1)
		}
		data := r.Resources.Raw["level_menu_layout"]
		for off := 0; off < len(data); off += 8 {
			d := data[off:]
			r.levelMenuSprite(im, int(binary.BigEndian.Uint16(d)),
				int(binary.BigEndian.Uint16(d[2:])), int(binary.BigEndian.Uint16(d[4:])),
				int(binary.BigEndian.Uint16(d[6:]))+1)
		}
		switch mode {
		case 0:
			r.menuPortrait(im, 0, 32)
			r.menuPortrait(im, 1, 88)
		case 4:
			r.menuPortrait(im, 1, 64)
		case 8:
			r.menuPortrait(im, 0, 64)
		}
		r.levelMenuBase[index] = im
	}
	return append([]byte(nil), r.levelMenuBase[index].Pix...)
}

func (r *Renderer) levelMenuSprite(im *image.Paletted, spriteOffset, offset, shift, height int) {
	composite(im, r.sprite("menu_objects", spriteOffset, 16, height, 0), offset%40*8+shift, offset/40, false)
}

// LevelMenuIndices retains each row's transparent animation history and draws
// the original blinking prompt over the static restoration surface.
func (r *Renderer) LevelMenuIndices(state game.LevelMenuState, mode int) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.MenuPalette)
	copy(im.Pix, r.LevelMenuBaseIndices(mode))
	positions := r.Resources.Raw["level_menu_markers"]
	for row, count := range state.MarkerCount {
		offset := int(binary.BigEndian.Uint32(positions[row*4:])) - 0x78000
		for _, frame := range state.MarkerOrder[row][:count] {
			for piece := 0; piece < 3; piece++ {
				address := 0xce8 + int(frame)*0x1b0 + piece*0x90
				r.levelMenuSprite(im, address, offset+piece*2, 0, 18)
			}
		}
	}
	if state.Prompt {
		text := append([]byte(nil), r.Resources.Raw["level_menu_text"]...)
		text[1] = byte(state.Shift)
		r.textStream(im, text, 0)
	}
	return im.Pix
}

func (r *Renderer) LevelMenuRGBA(state game.LevelMenuState, mode int) *image.RGBA {
	pixels := r.LevelMenuIndices(state, mode)
	var words [32]byte
	for i, v := range state.Palette {
		binary.BigEndian.PutUint16(words[i*2:], v)
	}
	top := disk.Palette(words[:])
	bottom := disk.Palette(r.Resources.Raw["portrait_palettes"][:32])
	bottom[0] = color.RGBA{0, 0, 102, 255}
	im := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for i, index := range pixels {
		palette := top
		if i >= 119*Width {
			palette = bottom
		}
		im.Set(i%Width, i/Width, palette[index])
	}
	return im
}
