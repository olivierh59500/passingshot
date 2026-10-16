package game

import (
	"encoding/binary"
	"fmt"

	"github.com/olivierh59500/passingshot/internal/resources"
)

type Phase uint8

const (
	Title Phase = iota
	Names
	LevelSelect
	Introduction
	Serve
	Rally
	PointResult
	SetResult
	Champion
	GameOver
)

type Input struct {
	Players                             [2]Control
	Confirm, Back, Pause, Menu, Restart bool
	Letter                              byte
	Selection                           int
	SelectionSet                        bool
	NamePick                            [2]int
	NamePickSet, NameJoin               [2]bool
}

type Event struct {
	Song, Effect int
	RestartSong  bool
	RequestSong  bool
}

type Model struct {
	PlayerFirstWord                              uint16
	Preparation                                  *ServicePreparation
	Point                                        PointState
	Celebration                                  Celebration
	FaultFrames                                  []int
	BallSpin                                     uint16
	BounceMark                                   BounceMark
	Rotation                                     ServiceRotation
	Service                                      ServiceState
	ServiceNotice                                ServiceNoticeState
	ServiceSprites                               [5]ServiceSprite
	ServiceSpriteCount                           int
	joystickRaw, joystickLatch                   [2]Control
	Resources                                    *resources.Set
	Phase                                        Phase
	PaletteCutTicks                              int
	PaletteSetupTicks                            int
	paletteCutTarget                             Phase
	Tick, PhaseTick, Level, InitialLevel         int
	Players                                      [4]Player
	Ball                                         Ball
	Score                                        Score
	ServiceBoard                                 [4]byte
	Names                                        [2]string
	Opponents                                    [2]string
	NameEntry                                    NameEntryState
	namePicks                                    [2][]int
	LevelMenu                                    LevelMenuState
	Tournament                                   TournamentResult
	GameOver                                     GameOverState
	levelPick                                    int
	levelPickPending                             bool
	Doubles, Paused, Demo                        bool
	Server, LastHitter                           int
	ServeAttempt, PointWinner                    int
	CameraX, CameraY                             int
	Message                                      string
	Events                                       []Event
	RallyHits, TotalPoints                       int
	BallSide, Outcome, BlinkPlayer, NearReceiver int
	MatchFrame                                   uint16
	SinglesPlayer                                int
	Miss, Bounced                                bool
	ballFlags                                    uint16
	previous                                     Input
	rng                                          uint32
	contact                                      bool
	pointDelay                                   int
}

var LevelNames = [6]string{"FRENCH FINAL", "AUSTRALIA FINAL", "U.S.A. FINAL", "ENGLAND QT.FINAL", "ENGLAND SEMI.FINAL", "ENGLAND FINAL"}

func New(r *resources.Set, seed uint32) *Model {
	if seed == 0 {
		seed = DefaultSeed
	}
	g := &Model{Resources: r, Names: [2]string{"AAA", "AAA"}, Score: NewScore(), PointWinner: -1, rng: seed}
	g.PlayerFirstWord = binary.BigEndian.Uint16(r.Raw["players"])
	board := r.Raw["service_score_text"]
	g.Opponents = [2]string{string(board[51:54]), string(board[57:60])}
	g.LevelMenu.Wait = 20
	g.Players = [4]Player{
		{X: 230, Y: 340, Active: true, Bank: 0, FrameTable: 0x100},
		{X: 260, Y: 380, Active: true, Bank: 1, FrameTable: 0x100},
		{X: 250, Y: 250, Active: true, Far: true, Bank: 2, FrameTable: 0x100},
		{X: 300, Y: 250, Active: true, Far: true, Bank: 3, FrameTable: 0x100, FrameIndex: 1},
	}
	return g
}

func (g *Model) random() uint32 {
	// Keep random choices deterministic for replays and gameplay tests.
	g.rng = NextRandom(g.rng)
	return g.rng
}

func NextRandom(state uint32) uint32 {
	next := state << 1
	if state&0x80000000 == 0 {
		next ^= 0x1d872b45
	}
	return next
}

func (g *Model) transition(p Phase) {
	g.Phase, g.PhaseTick, g.PaletteCutTicks, g.previous = p, 0, 0, Input{}
	g.PaletteSetupTicks = 0
	g.Preparation = nil
}
func (g *Model) effect(id int) { g.Events = append(g.Events, Event{Song: -1, Effect: id}) }
func (g *Model) song(id int)   { g.Events = append(g.Events, Event{Song: id, Effect: -1}) }
func (g *Model) restartSong(id int) {
	g.Events = append(g.Events, Event{Song: id, Effect: -1, RestartSong: true})
}

func (g *Model) requestSong(id int) {
	g.Events = append(g.Events, Event{Song: id, Effect: -1, RequestSong: true})
}

func (g *Model) Update(in Input) {
	for i, control := range in.Players {
		if control != g.joystickRaw[i] {
			g.joystickRaw[i] = control
			g.joystickLatch[i] = control
		}
	}
	if in.Pause && !g.previous.Pause && g.Phase >= Introduction {
		g.Paused = !g.Paused
	}
	if g.Paused {
		resume := in.Confirm && !g.previous.Confirm
		for i, control := range in.Players {
			resume = resume || control&Fire != 0 && g.previous.Players[i]&Fire == 0
		}
		if resume {
			g.Paused = false
			g.previous = in
			return
		}
		if in.Menu {
			g.ReturnToTitle()
		}
		if in.Restart {
			g.Paused = false
			g.startSet()
		}
		g.previous = in
		return
	}
	if in.Menu {
		g.ReturnToTitle()
		return
	}
	controller := 0
	if g.Phase == LevelSelect && g.NameEntry.First >= 0 {
		controller = g.NameEntry.First
	}
	confirm := in.Confirm && !g.previous.Confirm
	fire := confirm || (in.Players[controller]&Fire != 0 && g.previous.Players[controller]&Fire == 0)
	back := in.Back && !g.previous.Back
	g.Tick++
	if g.Preparation != nil {
		g.stepServicePreparation()
		g.previous = in
		return
	}
	if g.PaletteCutTicks > 0 {
		g.stepPaletteCut()
		g.previous = in
		return
	}
	if g.PaletteSetupTicks > 0 {
		g.stepPaletteSetup()
		g.previous = in
		return
	}
	g.PhaseTick++
	switch g.Phase {
	case Title:
		if fire || g.PhaseTick >= TitleTicks {
			g.beginPaletteCut(Names, NamePaletteCutTicks)
		}
	case Names:
		g.updateNames(in, confirm, back)
	case LevelSelect:
		g.updateLevelSelection(in, confirm, back)
	case Introduction:
		if fire || g.PhaseTick >= IntroductionTicks {
			g.BeginServicePreparation()
		}
	case Serve:
		g.updateServe(in, fire)
	case Rally:
		if g.PhaseTick%2 == 0 {
			g.simulate(in)
		}
	case PointResult:
		if g.Point.Fault != 0 {
			if g.Point.GameEnded {
				if g.PhaseTick%2 == 0 {
					if g.Celebration.Ended {
						g.finishPoint()
					} else {
						g.StepServiceFault()
					}
				}
			} else if g.PhaseTick >= g.pointDelay {
				g.finishPoint()
			}
		} else if g.PhaseTick%2 == 0 {
			if g.Point.Remaining > 0 {
				g.StepPoint()
			} else {
				g.finishPoint()
			}
		}
	case SetResult:
		if !g.Demo {
			if g.PhaseTick >= TournamentAdvanceTicks {
				g.startSet()
			}
		} else if fire || g.PhaseTick > 200 {
			g.ReturnToTitle()
		}
	case Champion:
		if g.PhaseTick >= TournamentChampionTicks {
			g.BeginNames()
		}
	case GameOver:
		if g.PhaseTick%2 == 0 {
			g.stepGameOver()
		}
	}
	g.previous = in
}

func (g *Model) ReturnToTitle() {
	g.Paused = false
	g.Demo = false
	g.Doubles = false
	g.Level = 0
	g.InitialLevel = 0
	g.Message = ""
	g.transition(Title)
	g.song(-2)
}

// StartDemo starts an explicitly requested deterministic demonstration.
func (g *Model) StartDemo(level int) {
	g.Demo = true
	g.Level = clamp(level, 0, 5)
	g.Doubles = false
	g.startSet()
}

func (g *Model) startSet() {
	g.Score = NewScore()
	g.Opponents = IntroductionOpponents(g.Level, g.Resources.Raw["introduction_table"])
	g.ServiceBoard = NewServiceBoard(g.random())
	g.Server = 0
	g.Rotation.NewGame(g.Doubles, g.SinglesPlayer, false, true)
	g.ServeAttempt = 0
	g.TotalPoints = 0
	g.CameraX = 160
	g.CameraY = 300
	g.Message = LevelNames[g.Level]
	g.transition(Introduction)
	g.restartSong(TournamentSong(g.Level, g.Resources.Raw["tournament_labels"]))
}

func (g *Model) prepareServe() {
	g.prepareServiceScene(true)
}

func (g *Model) prepareServiceScene(stopMusic bool) {
	// The first player sprite has a transparent leading word during matches.
	g.PlayerFirstWord = 0
	cfg := g.Rotation.Config
	g.ConfigureService(cfg)
	g.NearReceiver = cfg.NearReceiver
	g.LastHitter = cfg.Slot
	g.Server = 0
	if cfg.Far {
		g.LastHitter += 2
		g.Server = 1
	}
	g.Outcome = 0
	g.Point = PointState{}
	g.ServiceSpriteCount = 0
	g.Message = ""
	g.transition(Serve)
	if stopMusic {
		g.song(-2)
	}
}

func (g *Model) updateServe(in Input, fire bool) {
	if g.Demo && fire {
		g.ReturnToTitle()
		return
	}
	if !g.Service.Started {
		// A human starts the toss with the action button. Waiting at the
		// baseline must not turn into a fault before the player has served.
		if in.Players[g.Service.Config.Slot]&Fire == 0 {
			if g.PhaseTick%2 == 0 {
				if g.ServiceSpriteCount == 0 {
					g.StepService()
				} else {
					g.StepServiceNotice()
				}
			}
			return
		}
		g.Service.Started = true
		g.PhaseTick = 0
	}
	if g.PhaseTick%2 != 0 {
		return
	}
	if g.Demo && !g.Service.Config.Far {
		slot := g.Service.Config.Slot
		track := g.Service.Tracks[slot]
		data := g.Resources.Raw["service_sequences"]
		if data[track.Sequence+track.Cursor] == 0 && data[track.Sequence+track.Cursor+1] == 3 {
			g.joystickLatch[slot] = Up
		}
	}
	g.StepService()
	if g.Service.End == 0 {
		return
	}
	if g.Service.End < 0 && !g.Service.Config.Far {
		g.BeginServiceFault()
		return
	}
	g.effect(4)
	g.StartServiceRally()
	g.transition(Rally)
}

func (g *Model) simulate(in Input) {
	if g.Demo && (in.Confirm || in.Players[0]&Fire != 0) {
		g.ReturnToTitle()
		return
	}
	g.advancePlayers(true)
	g.ResolveRacket()
	g.AdvanceFlight()
	g.AdvanceDrawing()
	if g.Outcome != 0 {
		g.BeginPoint(g.Outcome)
	}
}

func (g *Model) finishPoint() {
	if g.PointWinner < 0 {
		g.prepareServe()
		return
	}
	if !g.Point.Scored {
		g.scorePoint(g.PointWinner)
	} else if g.Point.Fault == 2 && (!g.Point.GameEnded || g.Score.Winner < 0) {
		g.rotatePoint()
	}
	if g.Point.Fault == 2 && g.Point.GameEnded {
		g.ServiceSpriteCount = 0
		g.ServiceNotice.Kind = NoServiceNotice
	}
	g.ServeAttempt = 0
	if g.Score.Winner >= 0 {
		if g.Score.Winner == 1 {
			g.BeginGameOver()
			return
		}
		if g.Score.Winner == 0 && !g.Demo {
			g.BeginTournamentResult()
			return
		}
		g.Message = "CONGRATULATIONS"
		g.transition(SetResult)
		g.song(3)
		return
	}
	g.prepareServe()
}

func (g *Model) followCamera() {
	b := g.Ball
	if b.Y <= 280 {
		g.CameraY = max(0, b.Y-80)
	} else if b.Y >= 320 {
		g.CameraY = min(400, b.Y-120)
	}
	x := b.X - g.CameraX
	if x < 32 {
		g.CameraX = max(0, b.X-32)
	} else if x > 288 {
		g.CameraX = min(315, b.X-288)
	}
}

func (g *Model) String() string {
	return fmt.Sprintf("phase=%d level=%d games=%v points=%v ball=(%d,%d,%d) hits=%d", g.Phase, g.Level, g.Score.Games, g.Score.Points, g.Ball.X, g.Ball.Y, g.Ball.Height, g.RallyHits)
}
