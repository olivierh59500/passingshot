// Package resources loads only data reconstructed from the user's original disk.
package resources

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"

	"github.com/olivierh59500/passingshot/internal/disk"
)

// Generated proprietary data is deliberately excluded from version control.
// Run cmd/extract before building a desktop or Android application.
//
//go:embed generated/resources.zip checksums.json
var files embed.FS

type Set struct {
	Raw                                     map[string][]byte
	Title, Menu                             *image.Paletted
	Courts                                  [4]*image.Paletted
	CourtTiles                              [4][CourtShifts]*image.Paletted
	CourtAtlas                              [4][]byte
	CourtBackdrops                          [4]*image.Paletted
	TitlePalette, MenuPalette, CourtPalette color.Palette
	Fonts                                   [2][]*image.Paletted
}

func Load() (*Set, error) {
	data, err := files.ReadFile("generated/resources.zip")
	if err != nil {
		return nil, err
	}
	manifest, err := files.ReadFile("checksums.json")
	if err != nil {
		return nil, err
	}
	var sums map[string]string
	if err = json.Unmarshal(manifest, &sums); err != nil {
		return nil, err
	}
	if sums["disk"] != disk.SHA256 {
		return nil, fmt.Errorf("resource manifest has an unsupported source disk")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	s := &Set{Raw: map[string][]byte{}}
	for _, f := range z.File {
		if f.UncompressedSize64 > 1<<20 {
			return nil, fmt.Errorf("oversized resource %s", f.Name)
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(r, 1<<20))
		closeErr := r.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		name := strings.TrimSuffix(f.Name, ".bin")
		if sums[name] != fmt.Sprintf("%x", sha256.Sum256(b)) {
			return nil, fmt.Errorf("resource checksum failed: %s", name)
		}
		if _, exists := s.Raw[name]; exists {
			return nil, fmt.Errorf("duplicate resource: %s", name)
		}
		s.Raw[name] = b
	}
	for _, r := range disk.Regions {
		if len(s.Raw[r.Name]) != r.Length {
			return nil, fmt.Errorf("missing or invalid resource: %s", r.Name)
		}
	}
	s.TitlePalette = disk.Palette(s.Raw["title_palette"])
	s.MenuPalette = disk.Palette(s.Raw["menu_palette"])
	s.CourtPalette = disk.Palette(s.Raw["court_palette"])
	s.Title, err = disk.Interleaved(s.Raw["title"], 320, 200, 2, s.TitlePalette)
	if err != nil {
		return nil, err
	}
	s.Menu, err = disk.Interleaved(s.Raw["menu"], 256, 112, 2, s.MenuPalette)
	if err != nil {
		return nil, err
	}
	for i, name := range []string{"france", "australia", "usa", "england"} {
		s.Courts[i], err = disk.Court(s.Raw, name)
		if err != nil {
			return nil, err
		}
		table, err := courtTiles(s.Raw[name+"_tiles"], s.Raw[name+"_rows"])
		if err != nil {
			return nil, err
		}
		s.CourtAtlas[i] = table
		for shift := 0; shift < CourtShifts; shift++ {
			s.CourtTiles[i][shift], err = disk.Interleaved(table[shift*CourtShiftSize:], 16, CourtPairCount*8, 2, s.CourtPalette)
			if err != nil {
				return nil, err
			}
		}
		backing, err := courtBackdrop(s.Raw["stands"], s.Raw[name+"_back"], i)
		if err != nil {
			return nil, err
		}
		s.CourtBackdrops[i], err = disk.Planar(backing, 320, 200, s.CourtPalette)
		if err != nil {
			return nil, err
		}
	}
	for i, name := range []string{"font", "score_font"} {
		for off := 0; off+32 <= len(s.Raw[name]); off += 32 {
			glyph, err := disk.Interleaved(s.Raw[name][off:off+32], 8, 8, 1, s.MenuPalette)
			if err != nil {
				return nil, err
			}
			s.Fonts[i] = append(s.Fonts[i], glyph)
		}
	}
	return s, nil
}

type Frame struct {
	Address                                uint32
	Height, X, Y, RacketX, RacketY, Shadow int
	Width                                  int
	ReverseRows                            bool
}

func (s *Set) PlayerFrame(table, frame int) Frame {
	b := s.Raw["player_frames"]
	off := table + frame*16
	if off < 0 || off+16 > len(b) {
		return Frame{}
	}
	p := binary.BigEndian.Uint32(b[off:])
	w := 32
	if p&0x80000000 != 0 {
		w = 16
	}
	return Frame{p & 0x3fffffff, int(binary.BigEndian.Uint16(b[off+4:])) + 1, int(int16(binary.BigEndian.Uint16(b[off+6:]))), int(int16(binary.BigEndian.Uint16(b[off+8:]))), int(int16(binary.BigEndian.Uint16(b[off+10:]))), int(int16(binary.BigEndian.Uint16(b[off+12:]))), int(binary.BigEndian.Uint16(b[off+14:])), w, p&0x40000000 != 0}
}

// Sprite decodes a sprite from a named asset and a byte offset within it.
func (s *Set) Sprite(asset string, offset, width, height int, p color.Palette) *image.Paletted {
	return s.sprite(asset, offset, width, height, p, nil)
}

// SpriteWithPlayerWord applies the native loader's mutable first bank word
// while retaining the unmodified original resource pack for other callers.
func (s *Set) SpriteWithPlayerWord(offset, width, height int, p color.Palette, word uint16) *image.Paletted {
	return s.sprite("players", offset, width, height, p, &word)
}

func (s *Set) sprite(asset string, off, width, height int, p color.Palette, playerWord *uint16) *image.Paletted {
	b := s.Raw[asset]
	n := width / 8 * height * 4
	if off < 0 || n <= 0 || off > len(b)-n {
		return nil
	}
	unit := width / 8
	if unit > 4 {
		unit = 4
	}
	data := b[off : off+n]
	if asset == "players" && off == 0 && playerWord != nil {
		data = append([]byte(nil), data...)
		binary.BigEndian.PutUint16(data, *playerWord)
	}
	im, err := disk.Interleaved(data, width, height, unit, p)
	if err != nil {
		return nil
	}
	return im
}

type BallSprite struct {
	Asset         string
	Offset, Width int
}

// BallSprite resolves an encoded artwork frame descriptor.
func (s *Set) BallSprite(offset int) BallSprite {
	value := binary.BigEndian.Uint32(s.Raw["ball_frames"][offset:])
	width := 32
	if value&0x80000000 != 0 {
		width = 16
	}
	location := value & 0x3fffffff
	if location >= 0x41428 {
		return BallSprite{"objects", int(location - 0x41428), width}
	}
	return BallSprite{"players", int(location - 0x2cb68), width}
}
