package game

import "encoding/binary"

const TournamentAdvanceTicks = 240
const TournamentChampionTicks = 366

type TournamentResult struct {
	Level, NextLevel, Song int
	Bonus, Champion        bool
}

// NewTournamentResult uses ordinary game counts. The original stores twice
// each count, so its signed difference greater than four means at least three
// games. Only the first four levels permit a two-level advance.
func NewTournamentResult(level int, games [2]int, labels []byte) TournamentResult {
	s := TournamentResult{Level: level, NextLevel: level, Song: TournamentSong(level, labels), Champion: level >= 5}
	if !s.Champion {
		s.NextLevel++
		s.Bonus = level < 4 && games[0]-games[1] > 2
		if s.Bonus {
			s.NextLevel++
		}
	}
	return s
}

func TournamentSong(level int, labels []byte) int {
	return int(binary.BigEndian.Uint16(labels[0x18+level*2:]))
}

// BeginTournamentResult enters the original congratulations or champion scene.
// The global challenge level changes before the message is presented; the
// result retains the old level for its title and restarted composition.
func (g *Model) BeginTournamentResult() {
	g.Tournament = NewTournamentResult(g.Level, g.Score.Games, g.Resources.Raw["tournament_labels"])
	g.Level = g.Tournament.NextLevel
	if g.Tournament.Champion {
		g.transition(Champion)
	} else {
		g.transition(SetResult)
	}
	g.restartSong(g.Tournament.Song)
}

// AutomaticPresentation identifies loading, win and loss scenes that advance
// without confirmation input. Android uses it to label the action button.
func (g *Model) AutomaticPresentation() bool {
	return g.Preparation != nil || g.PaletteCutTicks > 0 || g.PaletteSetupTicks > 0 || g.Phase == Title || g.Phase == Introduction || g.Phase == Champion || g.Phase == GameOver || g.Phase == SetResult && g.Score.Winner == 0 && !g.Demo
}
