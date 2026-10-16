// Package app connects the native game, original artwork, audio, and Ebitengine.
package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/passingshot/internal/game"
	controls "github.com/olivierh59500/passingshot/internal/input"
	"github.com/olivierh59500/passingshot/internal/render"
	"github.com/olivierh59500/passingshot/internal/resources"
	"github.com/olivierh59500/passingshot/internal/sound"
)

type Options struct {
	Touch, Silent, Demo bool
	Level, Frames       int
	Capture             string
	Seed                uint32
	Preview             string
}

type App struct {
	Model                           *game.Model
	Renderer                        *render.Renderer
	Sound                           *sound.Player
	audio                           *audio.Player
	texture                         *ebiten.Image
	touch                           controls.Touch
	controller                      controls.Controller
	buttons                         [2]controls.Buttons
	layout                          controls.ControllerLayout
	controlTexture                  *ebiten.Image
	touchIDs                        []ebiten.TouchID
	gamepadIDs                      []ebiten.GamepadID
	options                         Options
	frames                          int
	muted                           bool
	SuspendRequested, BackRequested atomic.Bool
}

func New(options Options) (*App, error) {
	if options.Preview != "" && options.Preview != "advance" && options.Preview != "champion" && options.Preview != "gameover" {
		return nil, fmt.Errorf("unknown development preview %q", options.Preview)
	}
	r, err := resources.Load()
	if err != nil {
		return nil, fmt.Errorf("load original resources: %w", err)
	}
	seed := options.Seed
	if seed == 0 && (options.Demo || options.Preview != "") {
		// Development scenes use a fixed seed for repeatable captures.
		seed = 0x13579bdf
	}
	player := sound.New(r)
	a := &App{Model: game.New(r, seed), Renderer: render.New(r), Sound: player, options: options, texture: ebiten.NewImage(render.Width, render.Height)}
	if options.Demo {
		a.Model.StartDemo(options.Level - 1)
	}
	if options.Preview != "" {
		// Development previews exercise the real scene and automatic handoff.
		// Normal builds and startup leave this option empty.
		a.Model.Demo = false
		a.Model.Level = 0
		if options.Preview == "champion" {
			a.Model.Level = 5
		}
		if options.Preview == "gameover" {
			a.Model.Score = game.Score{Games: [2]int{3, 6}, Winner: 1}
			a.Model.BeginGameOver()
		} else {
			a.Model.Score = game.Score{Games: [2]int{6, 3}, Winner: 0}
			a.Model.BeginTournamentResult()
		}
	}
	a.dispatchAudio()
	if !options.Silent {
		ctx := audio.CurrentContext()
		if ctx == nil {
			ctx = audio.NewContext(sound.SampleRate)
		}
		a.audio, err = ctx.NewPlayer(a.Sound)
		if err != nil {
			return nil, fmt.Errorf("initialize audio: %w", err)
		}
		// Keep generated hit effects close to their gameplay events.
		a.audio.SetBufferSize(40 * time.Millisecond)
		a.audio.Play()
	}
	return a, nil
}

func (a *App) Update() error {
	if a.SuspendRequested.Swap(false) {
		a.touch.Reset()
		a.controller.Reset()
		for i := range a.buttons {
			a.buttons[i].Reset()
		}
		if a.Model.Phase >= game.Introduction {
			a.Model.Paused = true
		}
	}
	in := a.poll()
	if a.BackRequested.Swap(false) {
		if a.Model.Phase >= game.Introduction {
			in.Pause = true
		} else {
			in.Back = true
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		a.muted = !a.muted
		a.Sound.SetMuted(a.muted)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	// Gameplay advances once per fixed Ebitengine update on every platform.
	a.Model.Update(in)
	a.dispatchAudio()
	if err := a.Sound.Err(); err != nil {
		return fmt.Errorf("play audio: %w", err)
	}
	a.frames++
	if a.options.Frames > 0 && a.frames >= a.options.Frames {
		if a.options.Capture != "" {
			if err := a.Capture(a.options.Capture); err != nil {
				return err
			}
		}
		return ebiten.Termination
	}
	return nil
}

func (a *App) poll() game.Input {
	var in game.Input
	for i, keys := range [2][5]ebiten.Key{{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeySpace}, {ebiten.KeyW, ebiten.KeyS, ebiten.KeyA, ebiten.KeyD, ebiten.KeyF}} {
		for j, key := range keys {
			if ebiten.IsKeyPressed(key) {
				in.Players[i] |= [5]game.Control{game.Up, game.Down, game.Left, game.Right, game.Fire}[j]
			}
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyControlRight) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
		in.Players[0] |= game.Fire
	}
	in.Confirm = ebiten.IsKeyPressed(ebiten.KeyEnter)
	in.Back = ebiten.IsKeyPressed(ebiten.KeyBackspace)
	in.Pause = ebiten.IsKeyPressed(ebiten.KeyP)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if a.Model.Phase >= game.Introduction {
			in.Pause = true
		} else {
			in.Back = true
		}
	}
	if a.Model.Paused {
		in.Menu = ebiten.IsKeyPressed(ebiten.KeyQ)
		in.Restart = ebiten.IsKeyPressed(ebiten.KeyR)
	}
	if a.Model.Phase == game.Names {
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			for key := ebiten.KeyA; key <= ebiten.KeyZ; key++ {
				if inpututil.IsKeyJustPressed(key) {
					in.Letter = byte('A' + key - ebiten.KeyA)
				}
			}
			in.Players[1] = 0
		}
		if ebiten.IsKeyPressed(ebiten.Key2) {
			in.Players[1] |= game.Fire
		}
	}
	a.gamepadIDs = ebiten.AppendGamepadIDs(a.gamepadIDs[:0])
	standard := 0
	for _, id := range a.gamepadIDs {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			a.gamepadIDs[standard] = id
			standard++
		}
	}
	a.gamepadIDs = a.gamepadIDs[:standard]
	for index, id := range a.gamepadIDs {
		if index >= 2 {
			break
		}
		i := controls.GamepadPlayer(index, len(a.gamepadIDs), a.Model, a.options.Touch)
		if ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal) < -0.25 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			in.Players[i] |= game.Left
		}
		if ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal) > 0.25 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			in.Players[i] |= game.Right
		}
		if ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical) < -0.25 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			in.Players[i] |= game.Up
		}
		if ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical) > 0.25 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			in.Players[i] |= game.Down
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			in.Players[i] |= game.Fire
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			in.Pause = true
		}
	}
	a.touchIDs = ebiten.AppendTouchIDs(a.touchIDs[:0])
	points := make([]controls.Point, 0, len(a.touchIDs)+1)
	for _, id := range a.touchIDs {
		x, y := ebiten.TouchPosition(id)
		points = append(points, controls.Point{ID: int(id), X: float64(x), Y: float64(y)})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		points = append(points, controls.Point{ID: -1, X: float64(x), Y: float64(y)})
	}
	var t game.Input
	if a.options.Touch {
		frame := a.controller.Poll(points, a.layout)
		phase := a.Model.Phase
		if a.Model.Preparation != nil || phase == game.Serve && a.Model.Service.Config.Far {
			// WAIT is an automatic scene; do not arm a serve from its button.
			phase = game.Introduction
		}
		t = a.controller.GameInput(frame, phase, a.Model.Paused, a.Model.ServiceStrikeReady())
	}
	if !a.options.Touch || a.Model.Phase <= game.Introduction || a.Model.Phase >= game.SetResult || a.Model.Paused {
		var menuPoints []controls.Point
		for _, p := range points {
			p.X, p.Y = a.layout.GamePoint(p.X, p.Y)
			if p.X >= 0 && p.X < 320 && p.Y >= 0 && p.Y < 200 {
				menuPoints = append(menuPoints, p)
			}
		}
		menu := a.touch.Poll(menuPoints, a.Model.Phase, a.Model.Paused)
		t.NamePick, t.NamePickSet, t.NameJoin = menu.NamePick, menu.NamePickSet, menu.NameJoin
		t.Confirm = t.Confirm || menu.Confirm
		t.Pause = t.Pause || menu.Pause
		t.Menu = menu.Menu
		t.Restart = menu.Restart
		t.Selection, t.SelectionSet = menu.Selection, menu.SelectionSet
		for i := range t.Players {
			t.Players[i] |= menu.Players[i]
		}
	}
	player := 0
	if a.Model.Phase >= game.Introduction && !a.Model.Doubles {
		player = a.Model.SinglesPlayer
	}
	in.Players[player] |= t.Players[0]
	in.Players[1] |= t.Players[1]
	in.NamePick, in.NamePickSet, in.NameJoin = t.NamePick, t.NamePickSet, t.NameJoin
	in.Confirm = in.Confirm || t.Confirm
	in.Pause = in.Pause || t.Pause
	in.Menu = in.Menu || t.Menu
	in.Restart = in.Restart || t.Restart
	if t.SelectionSet {
		in.Selection = t.Selection
		in.SelectionSet = true
	}
	buttonPhase := a.Model.Phase
	if a.Model.Preparation != nil || buttonPhase == game.Serve && a.Model.Service.Config.Far {
		buttonPhase = game.Introduction
	}
	for i := range in.Players {
		in.Players[i] = a.buttons[i].Map(in.Players[i], buttonPhase, a.Model.Paused, a.Model.ServiceStrikeReady())
	}
	return in
}

func (a *App) Draw(screen *ebiten.Image) {
	im := a.Renderer.Draw(a.Model)
	a.texture.WritePixels(im.Pix)
	var options ebiten.DrawImageOptions

	options.GeoM.Scale(controls.GameScaleX, controls.GameScaleY)
	options.GeoM.Translate(float64(a.layout.GameX), float64(a.layout.GameY))
	options.Filter = ebiten.FilterNearest
	screen.DrawImage(a.texture, &options)
	if a.options.Touch {
		im := a.Renderer.ControllerFrame(a.layout, a.controller.State, a.Model)
		if a.controlTexture == nil || a.controlTexture.Bounds().Dx() != im.Rect.Dx() || a.controlTexture.Bounds().Dy() != im.Rect.Dy() {
			if a.controlTexture != nil {
				a.controlTexture.Dispose()
			}
			a.controlTexture = ebiten.NewImage(im.Rect.Dx(), im.Rect.Dy())
		}
		a.controlTexture.WritePixels(im.Pix)
		screen.DrawImage(a.controlTexture, nil)
	}
}

func (a *App) Layout(width, height int) (int, int) {
	if a.options.Touch {
		a.layout = controls.LayoutController(width, height)
		return a.layout.Width, a.layout.Height
	}
	const desktopHeight = controls.GameWidth * 3 / 4
	a.layout = controls.ControllerLayout{Width: controls.GameWidth, Height: desktopHeight, GameY: (desktopHeight - controls.GameHeight) / 2}
	return a.layout.Width, a.layout.Height
}

// DrawFinalScreen preserves the original palette at fractional display scales.
// Ebitengine's default final filter blends colors even when the game texture
// itself was drawn with nearest-neighbor sampling.
func (a *App) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	options := ebiten.DrawImageOptions{GeoM: geoM, Filter: ebiten.FilterNearest}
	screen.DrawImage(offscreen, &options)
}

func (a *App) Capture(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, a.Renderer.Draw(a.Model)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (a *App) Close() error {
	if a.audio != nil {
		return a.audio.Close()
	}
	return nil
}

func (a *App) dispatchAudio() {
	for _, event := range a.Model.Events {
		if event.Song != -1 {
			switch {
			case event.RequestSong:
				a.Sound.RequestSong(event.Song)
			case event.RestartSong:
				a.Sound.RestartSong(event.Song)
			default:
				a.Sound.Song(event.Song)
			}
		}
		if event.Effect >= 0 {
			a.Sound.Effect(event.Effect)
		}
	}
	a.Model.Events = a.Model.Events[:0]
}
func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }
