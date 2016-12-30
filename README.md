# dmgo - a gameboy emulator in go

I put together dmgo in about a week and half over the holidays. Much is left to be done, but things have progessed enough to throw it up on here.

#### Features:
 * Audio (on windows)!
 * Many games playable!
 * Glitches are relatively rare but still totally happen!
 * Graphical cross-platform support in native golang, with no hooks into C libraries needed.
 
That last bit relies on [exp/shiny](https://github.com/golang/exp/tree/master/shiny), which is still a work in progress. Let me know if it fails on your platform.

#### Build instructions:

dmgo uses [glide](https://github.com/Masterminds/glide) for dependencies, so run `glide update` first (or just `go get` the packages mentioned in `glide.yaml` file).

After that, `go build ./cmd/dmgo` should be enough. The interested can also see my build script `b` for more options (profiling and cross-compiling and such).
