package game

// ServiceScoreText rebuilds the original mutable score stream from native match state.
func (g *Model) ServiceScoreText() []byte {
	data := append([]byte(nil), g.Resources.Raw["service_score_text"]...)
	for i, offset := range [4]int{4, 8, 12, 16} {
		if g.ServiceBoard[i] != 0 {
			data[offset] = g.ServiceBoard[i]
		}
	}
	data[20], data[24] = byte('0'+g.Score.Games[0]), byte('0'+g.Score.Games[1])
	table := g.Resources.Raw["normal_point_text"]
	if g.Score.TieBreak {
		table = g.Resources.Raw["tie_point_text"]
	}
	copy(data[28:30], table[g.Score.Points[0]*2:g.Score.Points[0]*2+2])
	copy(data[33:35], table[g.Score.Points[1]*2:g.Score.Points[1]*2+2])
	names := [4]string{g.Names[0], g.Names[1], g.Opponents[0], g.Opponents[1]}
	if !g.Doubles {
		names = [4]string{"", names[g.SinglesPlayer], "", names[2+g.SinglesPlayer]}
	}
	for i, offset := range [4]int{39, 45, 51, 57} {
		for j := 0; j < 3; j++ {
			ch := byte('\\')
			if j < len(names[i]) && names[i][j] != ' ' {
				ch = names[i][j]
			}
			data[offset+j] = ch
		}
	}
	return data
}
