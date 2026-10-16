package render

import (
	"encoding/binary"
	"image"
	"image/color"

	"github.com/olivierh59500/passingshot/internal/disk"
)

func (r *Renderer) IntroductionIndices(level, mode int) []byte {
	if r.introductionBase[level][mode/4] == nil {
		im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.MenuPalette)
		table := r.Resources.Raw["introduction_table"]
		draw := func(opponent, person, shift int) {
			offset := int(binary.BigEndian.Uint32(table[level*8+opponent*4:])) - 0x434c
			r.portraitLayout(im, r.Resources.Raw["opponent_layout"][offset:], shift)
			r.menuPortrait(im, person, shift)
		}
		switch mode {
		case 0:
			draw(0, 0, 32)
			draw(1, 1, 88)
		case 4:
			draw(0, 1, 64)
		case 8:
			draw(1, 0, 64)
		}
		labels := r.Resources.Raw["tournament_labels"]
		offset := int(binary.BigEndian.Uint32(labels[level*4:])) - 0x40ba
		r.textStream(im, labels[offset:], 0)
		banner := r.Resources.Raw["versus_banner"]

		for y := 0; y < 8; y++ {
			for column := 0; column < 20; column++ {
				introductionWord(im, column*16, 90+y, banner[y*6:])
			}
		}
		for y := 0; y < 19; y++ {
			for column := 0; column < 4; column++ {
				introductionWord(im, 128+column*16, 90+y, banner[48+(y*4+column)*6:])
			}
		}
		r.introductionBase[level][mode/4] = im
	}
	return append([]byte(nil), r.introductionBase[level][mode/4].Pix...)
}

func introductionWord(im *image.Paletted, x, y int, data []byte) {
	planes := [3]uint16{binary.BigEndian.Uint16(data), binary.BigEndian.Uint16(data[2:]), binary.BigEndian.Uint16(data[4:])}
	for bit := 0; bit < 16; bit++ {
		index := byte(8)
		for plane, word := range planes {
			index |= byte((word>>uint(15-bit))&1) << uint(plane)
		}
		im.Pix[y*Width+x+bit] = index
	}
}

func (r *Renderer) IntroductionRGBA(level, mode int) *image.RGBA {
	pixels := r.IntroductionIndices(level, mode)
	table := r.Resources.Raw["introduction_palette"]
	palettes := r.Resources.Raw["portrait_palettes"]
	offset := int(binary.BigEndian.Uint32(table[level*4:])) - 0x452c - 2
	top := disk.Palette(palettes[offset : offset+32])
	var zero [2]byte
	copy(zero[:], table[24+level*2:26+level*2])
	background := disk.Palette(zero[:])[0]
	top[0] = background
	middle := append(color.Palette(nil), top...)
	middle[8] = background
	middle[10] = color.RGBA{238, 68, 68, 255}
	middle[11] = color.RGBA{136, 136, 136, 255}
	middle[12] = color.RGBA{170, 170, 170, 255}
	middle[13] = color.RGBA{238, 238, 238, 255}
	bottom := disk.Palette(palettes[:32])
	bottom[0] = background
	im := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for i, index := range pixels {
		palette := top
		if i >= 111*Width {
			palette = bottom
		} else if i >= 90*Width {
			palette = middle
		}
		im.Set(i%Width, i/Width, palette[index])
	}
	return im
}

// IntroductionPaletteWords supplies original words to native display work.
func (r *Renderer) IntroductionPaletteWords(level int) [3][16]uint16 {
	return r.Resources.IntroductionPaletteWords(level)
}
