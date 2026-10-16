package resources

import (
	"encoding/binary"
	"fmt"
)

const (
	CourtPairCount    = 250
	CourtTileSize     = 64
	CourtShiftSize    = CourtPairCount * CourtTileSize
	CourtShifts       = 8
	courtBackdropSize = 32000
)

// courtTiles joins each pair of original tiles at the eight scrolling offsets.
func courtTiles(tiles, rows []byte) ([]byte, error) {
	var rowTable [CourtPairCount * 4]byte
	var source [0x8000]byte
	if len(tiles) == 0 || len(tiles) > len(source) || len(rows) > len(rowTable) || len(rows)%4 != 0 {
		return nil, fmt.Errorf("invalid court tiles")
	}
	copy(rowTable[:], rows)
	copy(source[:], tiles)
	table := make([]byte, CourtShiftSize*CourtShifts)
	for pair := 0; pair < CourtPairCount; pair++ {
		left := int(binary.BigEndian.Uint16(rowTable[pair*4:]))
		right := int(binary.BigEndian.Uint16(rowTable[pair*4+2:]))
		if left+CourtTileSize > len(source) || right+CourtTileSize > len(source) {
			return nil, fmt.Errorf("invalid court tile pair %d", pair)
		}
		for word := 0; word < CourtTileSize/2; word++ {
			l := binary.BigEndian.Uint16(source[left+word*2:])
			r := binary.BigEndian.Uint16(source[right+word*2:])
			for shift := 0; shift < CourtShifts; shift++ {
				value := uint16((uint32(l)<<16 | uint32(r)) << uint(shift*2) >> 16)
				binary.BigEndian.PutUint16(table[shift*CourtShiftSize+pair*CourtTileSize+word*2:], value)
			}
		}
	}
	return table, nil
}

// courtBackdrop decodes the stands, court color and lower court artwork into
// a close-up background. Its layout preserves the original border pixels.
func courtBackdrop(stands, back []byte, level int) ([]byte, error) {
	if len(stands) < 0x4600 || len(back) < 0x1a80 {
		return nil, fmt.Errorf("incomplete court backdrop")
	}
	data := make([]byte, courtBackdropSize+2)
	put := func(offset int, value uint16) { binary.BigEndian.PutUint16(data[offset:], value) }
	for group := 0; group < 2240; group++ {
		for plane := 0; plane < 4; plane++ {
			put(group*2+plane*8000, binary.BigEndian.Uint16(stands[group*8+plane*2:]))
		}
	}
	color := [4]int{13, 1, 2, 3}[min(level, 3)]
	for group := 0; group < 1600; group++ {
		for plane := 0; plane < 4; plane++ {
			value := uint16(0)
			if color&(1<<plane) != 0 {
				value = 0xffff
			}
			put(4480+group*2+plane*8000, value)
		}
	}
	for group := 0; group < 80; group++ {
		for plane := 0; plane < 4; plane++ {
			extra := 0
			if plane != 0 {
				extra = 2
			}
			offset := 7680 + group*4 + plane*8000 + extra
			put(offset, 0)
			put(offset+2, 0)
		}
	}
	for row := 0; row < 53; row++ {
		for group := 0; group < 16; group++ {
			for plane := 0; plane < 4; plane++ {
				put(5124+row*40+group*2+plane*8000, binary.BigEndian.Uint16(back[row*128+group*8+plane*2:]))
			}
		}
	}
	return data[:courtBackdropSize], nil
}
