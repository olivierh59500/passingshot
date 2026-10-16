package render

// GlyphLayout describes one glyph from an extracted artwork text stream.
type GlyphLayout struct {
	Code             byte
	Font, X, Y, Rows int
}

// TextLayout decodes the original text layout data into pixel positions.
func TextLayout(data []byte, font int) []GlyphLayout {
	base := byte('0')
	if font == 1 {
		base = 'A'
	}
	x, y, shift := 0, 0, 0
	double := false
	var result []GlyphLayout
	newline := func() {
		limit := 24
		if double {
			limit = 23
		}
		if y < limit {
			y++
		}
	}
	advance := func() {
		x++
		if x > 39 {
			newline()
		}
	}
	for i := 0; i < len(data); i++ {
		ch := data[i]
		if ch == 0 {
			return result
		}
		if ch >= base {
			rows := 1
			if double {
				rows = 2
			}
			result = append(result, GlyphLayout{ch, font, (x + shift) * 8, y * 8, rows})
			advance()
			continue
		}
		switch ch {
		case 1, 8:
			double = true
		case 2:
			double = false
		case 3:
			if i+2 >= len(data) {
				return result
			}
			x, y = int(data[i+1]), int(data[i+2])
			i += 2
		case 9:
			if i+1 >= len(data) {
				return result
			}
			i++
			shift = int(data[i])
		case 4:
			x = max(0, x-1)
		case 5:
			advance()
		case 6:
			y = max(0, y-1)
		case 7:
			newline()
		}
	}
	return result
}
