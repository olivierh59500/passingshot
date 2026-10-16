package game

type Score struct {
	Points, Games [2]int
	TieBreak      bool
	Winner        int
}

func NewScore() Score { return Score{Winner: -1} }

// Point returns true when the point also finishes a game.
func (s *Score) Point(team int) bool {
	if s.Winner >= 0 || team < 0 || team > 1 {
		return false
	}
	other := 1 - team
	s.Points[team]++
	if s.TieBreak {
		if (s.Points[team] >= 7 && s.Points[team]-s.Points[other] >= 2) || s.Points[team] >= 14 {
			s.Games[team]++
			s.Points = [2]int{}
			s.Winner = team
			return true
		}
		return false
	}
	if s.Points[team] < 4 || s.Points[team]-s.Points[other] < 2 {
		// The original normalizes repeated deuce instead of growing counters.
		if s.Points[0] >= 4 && s.Points[1] >= 4 {
			s.Points[0]--
			s.Points[1]--
		}
		return false
	}
	s.Games[team]++
	s.Points = [2]int{}
	if s.Games[team] >= 6 && s.Games[team]-s.Games[other] >= 2 {
		s.Winner = team
	}
	if s.Games == [2]int{6, 6} {
		s.TieBreak = true
	}
	return true
}

func (s Score) PointText(team int) string {
	if s.TieBreak {
		return integer(s.Points[team])
	}
	if s.Points[team] >= 3 && s.Points[1-team] >= 3 {
		if s.Points[team] > s.Points[1-team] {
			return "AD"
		}
		return "40"
	}
	return [4]string{"0", "15", "30", "40"}[clamp(s.Points[team], 0, 3)]
}

func integer(v int) string {
	if v < 10 {
		return string(rune('0' + v))
	}
	return string([]rune{rune('0' + v/10%10), rune('0' + v%10)})
}
