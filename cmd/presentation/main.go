// Command presentation exports documentation screenshots and overhead gameplay
// using the shared native game, renderer and sound player.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/olivierh59500/passingshot/internal/game"
	controls "github.com/olivierh59500/passingshot/internal/input"
	"github.com/olivierh59500/passingshot/internal/render"
	"github.com/olivierh59500/passingshot/internal/resources"
	"github.com/olivierh59500/passingshot/internal/sound"
)

const updatesPerSecond = 50

type report struct {
	DurationSeconds                                      float64
	SimulationUpdates, Points, Rallies, LongestRallyHits int
	Seed                                                 uint32
	Frames                                               int
	Video, Method                                        string
}

func main() {
	output := flag.String("out", "recordings/passingshot-presentation.mp4", "new MP4 output (never overwrites a video)")
	duration := flag.Duration("duration", 90*time.Second, "duration of overhead gameplay at normal speed")
	screenshots := flag.String("screenshots", "docs/screenshots", "documentation PNG output directory")
	seed := flag.Uint("seed", 2, "deterministic input demonstration seed")
	flag.Parse()
	if err := export(*output, *screenshots, *duration, uint32(*seed)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func export(output, screenshots string, duration time.Duration, seed uint32) error {
	if duration < 10*time.Second || duration > 10*time.Minute {
		return fmt.Errorf("-duration must be between 10s and 10m")
	}
	if _, err := os.Stat(output); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("choose a new output filename: %s", output)
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("install FFmpeg with H.264 and AAC encoders: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	work, err := os.MkdirTemp(filepath.Dir(output), ".passingshot-export-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	r, err := resources.Load()
	if err != nil {
		return err
	}
	model, view, player := game.New(r, seed), render.New(r), sound.New(r)
	silentVideo, audioPath := filepath.Join(work, "video.mp4"), filepath.Join(work, "audio.s16le")
	audioFile, err := os.Create(audioPath)
	if err != nil {
		return err
	}
	defer audioFile.Close()
	audio := bufio.NewWriter(audioFile)
	encoder := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-n",
		"-f", "rawvideo", "-pixel_format", "rgba", "-video_size", "320x200", "-framerate", "25", "-i", "pipe:0",
		"-vf", "scale=1280:800:flags=neighbor", "-c:v", "libx264", "-preset", "fast", "-crf", "18", "-pix_fmt", "yuv420p", "-threads", "2", silentVideo)
	encoder.Stderr = os.Stderr
	pipe, err := encoder.StdinPipe()
	if err != nil {
		return err
	}
	if err := encoder.Start(); err != nil {
		return err
	}
	finished := false
	defer func() {
		if !finished {
			pipe.Close()
			encoder.Process.Kill()
			encoder.Wait()
		}
	}()
	video := bufio.NewWriterSize(pipe, render.Width*render.Height*4*2)
	pcm := make([]byte, sound.SampleRate/updatesPerSecond*4)
	stats := report{Seed: seed, Video: output, Method: "Shared native game, renderer and sound; automated controller input; normal speed. Only overhead rallies are retained; service and point scenes are omitted. Native frame export, not a physical Android capture."}
	var buttons controls.Buttons
	letters, letter := "GOP", 0
	totalTicks := int(duration.Seconds()*updatesPerSecond) / 2 * 2
	outputTicks, lastPhase := 0, model.Phase
	saved := map[string]bool{}
	if err := saveScreenshots(screenshots, saved, model, view.Draw(model)); err != nil {
		return err
	}
	for tick := 0; tick < 200000 && outputTicks < totalTicks; tick++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		in := game.Input{}
		switch model.Phase {
		case game.Title, game.Introduction:
			in.Confirm = model.PhaseTick%2 == 0
		case game.Names:
			if letter < len(letters) && model.PaletteCutTicks == 0 && model.PaletteSetupTicks == 0 && model.PhaseTick%20 == 0 {
				in.Letter = letters[letter]
				letter++
			}
		case game.LevelSelect:
			if model.PhaseTick >= 40 {
				in.Selection, in.SelectionSet = 0, true
			}
		case game.Serve:
			phase := model.Phase
			if model.Preparation != nil || model.Service.Config.Far {
				phase = game.Introduction
			}
			in.Players[0] = buttons.Map(game.Fire, phase, false, model.ServiceStrikeReady())
		case game.Rally:
			buttons.Map(0, game.Rally, false, false)
			in.Players[0] = plan(model)
		}
		model.Update(in)
		dispatch(player, model)
		if _, err := io.ReadFull(player, pcm); err != nil {
			return err
		}
		if err := player.Err(); err != nil {
			return err
		}
		if model.Phase == game.Rally && lastPhase != game.Rally {
			stats.Rallies++
		}
		lastPhase = model.Phase
		stats.LongestRallyHits = max(stats.LongestRallyHits, model.RallyHits)
		im := view.Draw(model)
		if err := saveScreenshots(screenshots, saved, model, im); err != nil {
			return err
		}
		if model.Phase == game.Rally {
			if _, err := audio.Write(pcm); err != nil {
				return err
			}
			if outputTicks%2 == 0 {
				if _, err := video.Write(im.Pix); err != nil {
					return err
				}
				stats.Frames++
			}
			outputTicks++
		}
		stats.SimulationUpdates = tick + 1
		stats.Points = model.TotalPoints
	}
	if outputTicks != totalTicks || !saved["overhead"] {
		return fmt.Errorf("demonstration did not produce enough overhead gameplay (%d/%d ticks)", outputTicks, totalTicks)
	}
	if err := video.Flush(); err != nil {
		return err
	}
	if err := pipe.Close(); err != nil {
		return err
	}
	err = encoder.Wait()
	finished = true
	if err != nil {
		return err
	}
	if err := audio.Flush(); err != nil {
		return err
	}
	if err := audioFile.Close(); err != nil {
		return err
	}
	stats.DurationSeconds = float64(outputTicks) / updatesPerSecond
	finalPath := filepath.Join(work, "presentation.mp4")
	mux := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-n", "-i", silentVideo,
		"-f", "s16le", "-ar", "44100", "-ac", "2", "-i", audioPath, "-map", "0:v:0", "-map", "1:a:0",
		"-c:v", "copy", "-c:a", "aac", "-b:a", "192k", "-metadata", "title=Passing Shot - Overhead Gameplay",
		"-metadata", "comment="+stats.Method, "-movflags", "+faststart", "-threads", "2", finalPath)
	mux.Stderr = os.Stderr
	if err := mux.Run(); err != nil {
		return err
	}
	// Exclusive creation protects an existing presentation, including a late arrival.
	destination, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		destination.Close()
		if !complete {
			os.Remove(output)
		}
	}()
	source, err := os.Open(finalPath)
	if err != nil {
		return err
	}
	defer source.Close()
	if _, err := io.Copy(destination, source); err != nil {
		return err
	}
	if err := destination.Close(); err != nil {
		return err
	}
	complete = true
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(output+".json", append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Exported %s: %.0fs overhead gameplay, %d rallies, longest %d hits.\n", output, stats.DurationSeconds, stats.Rallies, stats.LongestRallyHits)
	return nil
}

func dispatch(player *sound.Player, model *game.Model) {
	for _, event := range model.Events {
		if event.Song != -1 {
			switch {
			case event.RequestSong:
				player.RequestSong(event.Song)
			case event.RestartSong:
				player.RestartSong(event.Song)
			default:
				player.Song(event.Song)
			}
		}
		if event.Effect >= 0 {
			player.Effect(event.Effect)
		}
	}
	model.Events = model.Events[:0]
}

func saveScreenshots(dir string, saved map[string]bool, model *game.Model, im *image.RGBA) error {
	if model.PaletteCutTicks != 0 || model.PaletteSetupTicks != 0 {
		return nil
	}
	name := ""
	switch {
	case model.Phase == game.Title:
		name = "title"
	case model.Phase == game.LevelSelect && model.PhaseTick > 30:
		name = "stages"
	case model.Phase == game.Serve && model.Preparation == nil && !model.Service.Config.Far && !model.Service.Started:
		name = "service"
	case model.Phase == game.Rally && model.PhaseTick > 50 && model.RallyHits >= 2:
		name = "overhead"
	}
	// Prefer a rally frame with both players fully visible in the scrolling view.
	framed := false
	if name == "overhead" {
		framed = true
		for _, p := range model.Players {
			if !p.Active {
				continue
			}
			f := model.Resources.PlayerFrame(p.FrameTable, p.FrameIndex)
			x, y := p.X+f.X-model.CameraX, p.Y+f.Y-model.CameraY
			framed = framed && x >= 4 && y >= 8 && x+f.Width < render.Width-4 && y+f.Height < render.Height-8
		}
	}
	if name == "" || saved[name] && (name != "overhead" || saved["overhead-framed"] || !framed) {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		return err
	}
	large := image.NewRGBA(image.Rect(0, 0, render.Width*2, render.Height*2))
	for y := 0; y < large.Rect.Dy(); y++ {
		for x := 0; x < large.Rect.Dx(); x++ {
			large.SetRGBA(x, y, im.RGBAAt(x/2, y/2))
		}
	}
	if err := png.Encode(f, large); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	saved[name] = true
	if name == "overhead" && framed {
		saved["overhead-framed"] = true
	}
	return nil
}
