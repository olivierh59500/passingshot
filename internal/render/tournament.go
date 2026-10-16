package render

import (
	"encoding/binary"
	"image"

	"github.com/olivierh59500/passingshot/internal/disk"
	"github.com/olivierh59500/passingshot/internal/game"
)

func (r *Renderer) TournamentResultIndices(state game.TournamentResult, mode int) []byte {
	im := image.NewPaletted(image.Rect(0, 0, Width, Height), r.Resources.MenuPalette)
	labels := r.Resources.Raw["tournament_labels"]
	offset := int(binary.BigEndian.Uint32(labels[state.Level*4:])) - 0x40ba
	r.textStream(im, labels[offset:], 0)
	messages := append([]byte(nil), r.Resources.Raw["victory_messages"]...)
	r.textStream(im, messages[0xb3:], 0)
	if mode != 4 {
		r.menuPortrait(im, 0, 32)
	}
	if mode != 8 {
		r.menuPortrait(im, 1, 88)
	}
	composite(im, r.sprite("portraits", 0x2c18, 32, 64, 1), 128, 135, false)
	composite(im, r.sprite("portraits", 0x3018, 32, 64, 1), 160, 135, false)
	if state.Champion {
		offset := 0
		if mode == 0 {
			offset = 0x29
		}
		r.textStream(im, r.Resources.Raw["champion_messages"][offset:], 0)
	} else {
		praise := 0x76
		if mode == 0 {
			praise = 0x95
		}
		r.textStream(im, messages[praise:], 0)
		messages[0x2f], messages[0x5a] = byte('2'+state.Level), byte('2'+state.Level)
		messages[0x74] = byte('3' + state.Level)
		advance := 0
		if state.Bonus {
			advance = 0x31
		}
		r.textStream(im, messages[advance:], 0)
	}
	return im.Pix
}

func (r *Renderer) TournamentResultRGBA(state game.TournamentResult, mode int) *image.RGBA {
	pixels := r.TournamentResultIndices(state, mode)
	palette := disk.Palette(r.Resources.Raw["portrait_palettes"][:32])
	im := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for i, index := range pixels {
		im.Set(i%Width, i/Width, palette[index])
	}
	return im
}
