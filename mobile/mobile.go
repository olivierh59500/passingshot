// Package mobile registers the same native application with Ebitengine's view.
package mobile

import (
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/olivierh59500/passingshot/internal/app"
	"github.com/olivierh59500/passingshot/internal/input"
)

var suspended, back atomic.Bool

// probeResult is set only by the development APK builder's linker option.
// Its default value preserves normal startup in distributed builds.
var probeResult string

type bootstrap struct {
	app           *app.App
	width, height int
}

func init() { ebiten.SetTPS(50); enginemobile.SetGame(&bootstrap{}) }
func (b *bootstrap) Update() error {
	if b.app == nil {
		a, err := app.New(app.Options{Touch: true, Preview: probeResult})
		if err != nil {
			return err
		}
		b.app = a
		b.app.Layout(b.width, b.height)
	}
	if suspended.Swap(false) {
		b.app.SuspendRequested.Store(true)
	}
	if back.Swap(false) {
		b.app.BackRequested.Store(true)
	}
	return b.app.Update()
}
func (b *bootstrap) Draw(dst *ebiten.Image) {
	if b.app != nil {
		b.app.Draw(dst)
	}
}
func (b *bootstrap) Layout(width, height int) (int, int) {
	b.width, b.height = width, height
	if b.app != nil {
		return b.app.Layout(width, height)
	}
	layout := input.LayoutController(width, height)
	return layout.Width, layout.Height
}

// DrawFinalScreen forwards the shared palette-preserving display policy.
func (b *bootstrap) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	if b.app != nil {
		b.app.DrawFinalScreen(screen, offscreen, geoM)
	}
}

// Suspend releases touch input and pauses the match on the next update.
func Suspend() { suspended.Store(true) }

// Back routes Android's back action through the shared native game state.
func Back() { back.Store(true) }
