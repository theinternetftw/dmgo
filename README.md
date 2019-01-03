# dmgo - a gameboy emulator in go

My other emulators:
[famigo](https://github.com/theinternetftw/famigo),
[vcsgo](https://github.com/theinternetftw/vcsgo),
[segmago](https://github.com/theinternetftw/segmago), and
[a1go](https://github.com/theinternetftw/a1go).

#### Features:
 * Audio!
 * Saved game support!
 * Quicksave/Quickload, too!
 * All major [MBCs](http://gbdev.gg8.se/wiki/articles/Memory_Bank_Controllers) suppported!
 * Glitches are relatively rare but still totally happen!
 * Graphical and auditory cross-platform support!

That last bit relies on [glimmer](https://github.com/theinternetftw/glimmer). Tested on windows 10 and ubuntu 18.10.

#### Dependencies:

 * You can compile on windows with no C dependencies.
 * Linux users should 'apt install libasound2-dev' or equivalent.
 * FreeBSD (and Mac?) users should 'pkg install openal-soft' or equivalent.

#### Compile instructions

 * If you have go version >= 1.11, `go build ./cmd/dmgo` should be enough.
 * The interested can also see my build script `b` for profiling and such.
 * Non-windows users will need the dependencies listed above.

#### Important Notes:

 * Keybindings are currently hardcoded to WSAD / JK / TY (arrowpad, ab, start/select)
 * Saved games use/expect a slightly different naming convention than usual: romfilename.gb.sav
 * Quicksave/Quickload is done by pressing m or l (make or load quicksave), followed by a number key
