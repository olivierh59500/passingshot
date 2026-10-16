# Preparing the original game data

The Passing Shot ADF and reconstructed resources are not distributed in this
repository. Generate the runtime asset pack locally from your own compatible
disk before building the game.

## Supported disk

The extractor supports the 1989 Image Works revision marked `[cr Europe]`. Its
uncompressed ADF SHA-256 is:

```text
b6ba6887d25bd9fa305e03720701ae891fa5f03a5724f0b674a8735fc8bdd32c
```

Use the [Planet Emulation Amiga ADF catalogue, letter P](https://www.planetemu.net/roms/commodore-amiga-games-adf?page=P)
to identify Passing Shot revisions. Extract the `.adf` from its archive first.

## Generate and run

```sh
export GOCACHE="$PWD/.cache/go-build"
go run ./cmd/extract -adf "/path/to/Passing Shot.adf"
go run ./cmd/passingshot
```

The extractor verifies the complete disk fingerprint and reconstructs the 78
resource regions used by the native game. The application embeds the generated
pack and checks it against its checksum manifest.

Generated resources, original disk images and build products remain local. Use
`.git/info/exclude` for local paths when preparing assets in a clone; no project
`.gitignore` is required.
