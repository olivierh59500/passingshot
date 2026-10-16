// Command passingshot runs the native desktop game.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/passingshot/internal/app"
)

func main() {
	var options app.Options
	flag.BoolVar(&options.Touch, "touch", false, "enable the virtual pad and contextual hit button")
	flag.BoolVar(&options.Silent, "silent", false, "disable audio output")
	flag.BoolVar(&options.Demo, "demo", false, "start an automatic demonstration")
	flag.IntVar(&options.Level, "level", 1, "initial demo tournament level (1-6)")
	flag.IntVar(&options.Frames, "frames", 0, "exit after this many updates (0 runs normally)")
	flag.StringVar(&options.Capture, "capture", "", "write an original-resolution PNG on exit")
	flag.StringVar(&options.Preview, "preview", "", "development scene with automatic handoff: advance, champion or gameover")
	seed := flag.Uint("seed", 0, "game random seed (0 retains the original title-entry state)")
	flag.Parse()
	options.Seed = uint32(*seed)
	a, err := app.New(options)
	if err != nil {
		fail(err)
	}
	defer a.Close()
	ebiten.SetTPS(50)
	ebiten.SetWindowSize(960, 720)
	ebiten.SetWindowTitle("Passing Shot")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(a); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
