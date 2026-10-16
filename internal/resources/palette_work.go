package resources

import "encoding/binary"

// IntroductionPaletteWords retains the original twelve-bit memory values for
// native palette programming; it does not recover unused colors from pixels.
func (s *Set) IntroductionPaletteWords(level int) [3][16]uint16 {
	table := s.Raw["introduction_palette"]
	data := s.Raw["portrait_palettes"]
	offset := int(binary.BigEndian.Uint32(table[level*4:])) - 0x452c - 2
	var palettes [3][16]uint16
	for i := 0; i < 16; i++ {
		palettes[0][i] = binary.BigEndian.Uint16(data[offset+i*2:])
		palettes[2][i] = binary.BigEndian.Uint16(data[i*2:])
	}
	palettes[1] = palettes[0]
	zero := binary.BigEndian.Uint16(table[24+level*2:])
	for i := range palettes {
		palettes[i][0] = zero
	}
	palettes[1][8], palettes[1][10], palettes[1][11], palettes[1][12], palettes[1][13] = zero, 0xe44, 0x888, 0xaaa, 0xeee
	return palettes
}
