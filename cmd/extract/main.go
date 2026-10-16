// Command extract reconstructs the remake's resources from a supplied ADF.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/olivierh59500/passingshot/internal/disk"
)

func main() {
	source := flag.String("adf", "", "path to the supported original ADF")
	out := flag.String("out", "internal/resources/generated", "resource output directory")
	previews := flag.String("previews", "assets", "directory for reconstructed PNG previews")
	flag.Parse()
	if *source == "" {
		fail(fmt.Errorf("provide -adf /path/to/game.adf"))
	}
	b, err := os.ReadFile(*source)
	fail(err)
	resources, err := disk.Extract(b)
	fail(err)
	fail(os.MkdirAll(*out, 0755))
	fail(os.MkdirAll(*previews, 0755))
	f, err := os.Create(filepath.Join(*out, "resources.zip"))
	fail(err)
	z := zip.NewWriter(f)
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}
	sort.Strings(names)
	sums := map[string]string{"disk": disk.SHA256}
	for _, name := range names {
		data := resources[name]
		w, err := z.Create(name + ".bin")
		fail(err)
		_, err = w.Write(data)
		fail(err)
		sums[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	fail(z.Close())
	fail(f.Close())
	manifest, err := json.MarshalIndent(sums, "", "  ")
	fail(err)
	fail(os.WriteFile(filepath.Join(*out, "checksums.json"), append(manifest, '\n'), 0644))
	p := disk.Palette(resources["title_palette"])
	im, err := disk.Interleaved(resources["title"], 320, 200, 2, p)
	fail(err)
	f, err = os.Create(filepath.Join(*previews, "title.png"))
	fail(err)
	fail(png.Encode(f, im))
	fail(f.Close())
	menu, err := disk.Interleaved(resources["menu"], 256, 112, 2, disk.Palette(resources["menu_palette"]))
	fail(err)
	f, err = os.Create(filepath.Join(*previews, "menu.png"))
	fail(err)
	fail(png.Encode(f, menu))
	fail(f.Close())
	player, err := disk.Interleaved(resources["players"], 32, 30, 4, disk.Palette(resources["court_palette"]))
	fail(err)
	f, err = os.Create(filepath.Join(*previews, "player.png"))
	fail(err)
	fail(png.Encode(f, player))
	fail(f.Close())
	for _, name := range []string{"france", "australia", "usa", "england"} {
		court, err := disk.Court(resources, name)
		fail(err)
		f, err = os.Create(filepath.Join(*previews, name+".png"))
		fail(err)
		fail(png.Encode(f, court))
		fail(f.Close())
	}
	fmt.Printf("Verified the ADF and reconstructed %d resource regions.\n", len(disk.Regions))
}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
