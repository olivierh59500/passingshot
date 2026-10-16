// Package disk extracts Passing Shot artwork, audio and gameplay data.
package disk

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

const SHA256 = "b6ba6887d25bd9fa305e03720701ae891fa5f03a5724f0b674a8735fc8bdd32c"
const programOffset = 0x2c1c

// Region describes a byte range used by the original resource loader.
type Region struct {
	Name           string
	Offset, Length int
}

// fileRegions locates the original artwork and audio within the supported ADF.
var fileRegions = []Region{
	{"title", 0xdc * 512, 0x7d00},
	{"players", 0x134 * 512, 0x148c0},
	{"objects", 0x1ef * 512, 0x79a8},
	{"samples", 0x596 * 512, 0x7e60},
	{"menu", 0x23c * 512, 0x3800},
	{"portraits", 0x25d * 512, 0x3418},
	{"menu_objects", 0x27e * 512, 0x13a8},
	{"level_art", 0x294 * 512, 0x2ec0},
	{"stands", 0x2ec * 512, 0x4600},
	{"france_tiles", 0x2b5 * 512, 0x1d00},
	{"france_rows", 0x2cb * 512, 0x36c},
	{"france_net", 0x2d6 * 512, 0x1770},
	{"france_back", 0x318 * 512, 0x1a80},
	{"australia_tiles", 0x3de * 512, 0x21c0},
	{"australia_rows", 0x3f4 * 512, 0x3e4},
	{"australia_net", 0x3ff * 512, 0x1770},
	{"australia_back", 0x415 * 512, 0x1a80},
	{"usa_tiles", 0x44c * 512, 0x1d80},
	{"usa_rows", 0x462 * 512, 0x388},
	{"usa_net", 0x46d * 512, 0x1770},
	{"usa_back", 0x483 * 512, 0x1a80},
	{"england_tiles", 0x4ba * 512, 0x1cc0},
	{"england_rows", 0x4d0 * 512, 0x36c},
	{"england_net", 0x4db * 512, 0x1770},
	{"england_back", 0x4f1 * 512, 0x1a85},
}

// Regions contains the asset ranges needed by the remake.
var Regions = append(append([]Region(nil), fileRegions...), []Region{
	{"font", programOffset + 0x8f0a, 0x660},
	{"score_font", programOffset + 0x95aa, 0x180},
	{"title_palette", programOffset + 0x490e, 32},
	{"menu_palette", programOffset + 0x4edc, 32},
	{"court_palette", programOffset + 0x3f0c, 32},
	{"portrait_palettes", programOffset + 0x452c, 0xd4},
	{"portraits_layout", programOffset + 0x42fc, 0x50},
	{"name_layout", programOffset + 0x502e, 0x48},
	{"name_prompts", programOffset + 0x5076, 0x32},
	{"name_entry_text", programOffset + 0x54e6, 0x62},
	{"name_selector", programOffset + 0x5548, 0x60},
	{"level_menu_palette", programOffset + 0x4efc, 32},
	{"level_menu_text", programOffset + 0x5602, 0x1c},
	{"level_menu_colors", programOffset + 0x5742, 0x10},
	{"level_menu_markers", programOffset + 0x57a4, 0x10},
	{"level_menu_layout", programOffset + 0x591e, 0x148},
	{"tournament_labels", programOffset + 0x40ba, 0xda},
	{"introduction_table", programOffset + 0x41dc, 0x60},
	{"introduction_palette", programOffset + 0x42d2, 0x24},
	{"opponent_layout", programOffset + 0x434c, 0x1e0},
	{"versus_banner", programOffset + 0x4cc6, 0x1f8},
	{"victory_messages", programOffset + 0x2304, 0xc8},
	{"champion_messages", programOffset + 0x221a, 0x54},
	{"gameover_palette", programOffset + 0x1b04, 32},
	{"gameover_letters", programOffset + 0x1e3c, 0x380},
	{"flight_curve", programOffset + 0x3352, 256},
	{"ai_accuracy", programOffset + 0x29a4, 12},
	{"ai_shots", programOffset + 0x29b0, 32},
	{"ai_depth", programOffset + 0x29d0, 48},
	{"ai_reaction", programOffset + 0x2d6c, 12},
	{"ai_jitter", programOffset + 0x2db6, 12},
	{"random_spin", programOffset + 0x2afe, 16},
	{"shot_depth", programOffset + 0x2e00, 96},
	// Long shots can index beyond the nominal 128-byte direction table. The
	// adjacent bytes are observable lookup data in the original, never executed
	// by the native game.
	{"shot_direction", programOffset + 0x2e60, 170},
	{"service_sequences", programOffset + 0x67c6, 0x6e6},
	{"service_power", programOffset + 0x46f8, 46},
	{"point_messages", programOffset + 0x19b2, 84},
	{"point_score_text", programOffset + 0x74fa, 0x364},
	{"service_score_text", programOffset + 0x7a58, 0x3d},
	{"service_notices", programOffset + 0x7056, 0x2b},
	{"normal_point_text", programOffset + 0x749c, 12},
	{"tie_point_text", programOffset + 0x74c2, 30},
	{"fault_messages", programOffset + 0x6ef6, 25},
	{"celebration_sequences", programOffset + 0x7d06, 0x74},
	{"celebration_choices", programOffset + 0x7dd6, 0x38},
	{"celebration_decorations", programOffset + 0x7f1a, 0x30},
	{"music_pitch", programOffset + 0xa466, 0x1e0},
	{"music_patterns", programOffset + 0xa984, 0x49c},
	{"music_tracks", programOffset + 0xae1e, 0x96},
	{"music_instruments", programOffset + 0xb5ec, 0x5f72},
	{"player_frames", programOffset + 0x37a8, 0x480},
	{"ball_frames", programOffset + 0x3480, 0x140},
	{"ball_art", programOffset + 0x35c8, 0x80},
}...)

// Court rebuilds an unshifted overview of the 40 by 75 tile map. Scrolled
// rendering must retain both members of each original tile pair: some right-hand
// neighbors differ from the next map cell, particularly at map boundaries.
func Court(resources map[string][]byte, name string) (*image.Paletted, error) {
	tiles, rows, mapping := resources[name+"_tiles"], resources[name+"_rows"], resources[name+"_net"]
	p := Palette(resources["court_palette"])
	if len(mapping) != 40*75*2 {
		return nil, fmt.Errorf("invalid court map %s", name)
	}
	im := image.NewPaletted(image.Rect(0, 0, 640, 600), p)
	for ty := 0; ty < 75; ty++ {
		for tx := 0; tx < 40; tx++ {
			index := int(binary.BigEndian.Uint16(mapping[(ty*40+tx)*2:])) / 64
			if index*4+2 > len(rows) {
				return nil, fmt.Errorf("court tile %d outside row table %s", index, name)
			}
			offset := int(binary.BigEndian.Uint16(rows[index*4:]))
			if offset+64 > len(tiles) {
				return nil, fmt.Errorf("court source %d outside tiles %s", offset, name)
			}
			tile, err := Interleaved(tiles[offset:offset+64], 16, 8, 2, p)
			if err != nil {
				return nil, err
			}
			for y := 0; y < 8; y++ {
				copy(im.Pix[(ty*8+y)*640+tx*16:], tile.Pix[y*16:y*16+16])
			}
		}
	}
	return im, nil
}

// CourtPairs decodes adjacent tiles as 32 indexed pixels for smooth scrolling.
func CourtPairs(resources map[string][]byte, name string) (*image.Paletted, error) {
	tiles, rows := resources[name+"_tiles"], resources[name+"_rows"]
	if len(rows)%4 != 0 {
		return nil, fmt.Errorf("invalid court pair table %s", name)
	}
	p := Palette(resources["court_palette"])
	im := image.NewPaletted(image.Rect(0, 0, 32, len(rows)/4*8), p)
	for row := 0; row < len(rows)/4; row++ {
		for side := 0; side < 2; side++ {
			offset := int(binary.BigEndian.Uint16(rows[row*4+side*2:]))
			if offset+64 > len(tiles) {
				return nil, fmt.Errorf("court pair %d outside tiles %s", row, name)
			}
			tile, err := Interleaved(tiles[offset:offset+64], 16, 8, 2, p)
			if err != nil {
				return nil, err
			}
			for y := 0; y < 8; y++ {
				copy(im.Pix[(row*8+y)*im.Stride+side*16:], tile.Pix[y*16:y*16+16])
			}
		}
	}
	return im, nil
}

// Extract rejects unknown or damaged disks before accessing any resource range.
func Extract(adf []byte) (map[string][]byte, error) {
	if len(adf) != 901120 {
		return nil, fmt.Errorf("expected a 901120-byte disk; got %d bytes", len(adf))
	}
	if digest := fmt.Sprintf("%x", sha256.Sum256(adf)); digest != SHA256 {
		return nil, fmt.Errorf("unsupported disk SHA-256: %s (expected %s)", digest, SHA256)
	}
	result := make(map[string][]byte, len(Regions))
	for _, r := range Regions {
		if r.Offset < 0 || r.Length < 0 || r.Offset > len(adf)-r.Length {
			return nil, fmt.Errorf("invalid resource range: %s", r.Name)
		}
		result[r.Name] = append([]byte(nil), adf[r.Offset:r.Offset+r.Length]...)
	}
	return result, nil
}

// Palette expands the original 12-bit RGB colors without substitution.
func Palette(raw []byte) color.Palette {
	p := make(color.Palette, len(raw)/2)
	for i := range p {
		v := binary.BigEndian.Uint16(raw[i*2:])
		p[i] = color.NRGBA{uint8(v>>8&15) * 17, uint8(v>>4&15) * 17, uint8(v&15) * 17, 255}
	}
	return p
}

// Interleaved decodes four bitplanes stored in units of one or two bytes.
// Fonts use byte units, and screens and sprites use word or longword units.
func Interleaved(raw []byte, width, height, unit int, p color.Palette) (*image.Paletted, error) {
	if width <= 0 || width%8 != 0 || height <= 0 || (unit != 1 && unit != 2 && unit != 4) || width/8%unit != 0 || len(raw) < width/8*height*4 {
		return nil, fmt.Errorf("invalid interleaved image %dx%d, unit %d, length %d", width, height, unit, len(raw))
	}
	im := image.NewPaletted(image.Rect(0, 0, width, height), p)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			col := x / 8
			pos := y*(width/8*4) + (col/unit)*(unit*4) + col%unit
			var index uint8
			for plane := 0; plane < 4; plane++ {
				index |= ((raw[pos+plane*unit] >> uint(7-x%8)) & 1) << uint(plane)
			}
			im.SetColorIndex(x, y, index)
		}
	}
	return im, nil
}

// Planar decodes the display buffers used by the original four-plane screen.
func Planar(raw []byte, width, height int, p color.Palette) (*image.Paletted, error) {
	size := width / 8 * height
	if width <= 0 || width%8 != 0 || height <= 0 || len(raw) < size*4 {
		return nil, fmt.Errorf("invalid planar image")
	}
	im := image.NewPaletted(image.Rect(0, 0, width, height), p)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var v uint8
			for plane := 0; plane < 4; plane++ {
				v |= ((raw[plane*size+y*(width/8)+x/8] >> uint(7-x%8)) & 1) << uint(plane)
			}
			im.SetColorIndex(x, y, v)
		}
	}
	return im, nil
}
