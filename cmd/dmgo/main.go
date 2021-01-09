package main

import (
	"github.com/theinternetftw/dmgo"
	"github.com/theinternetftw/dmgo/profiling"
	"github.com/theinternetftw/glimmer"

	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func main() {

	defer profiling.Start().Stop()

	assert(len(os.Args) == 2, "usage: ./dmgo ROM_FILENAME")
	cartFilename := os.Args[1]

	var cartBytes []byte
	var err error
	if strings.HasSuffix(cartFilename, ".zip") {
		cartBytes = readZipFileOrDie(cartFilename)
	} else {
		cartBytes, err = ioutil.ReadFile(cartFilename)
		dieIf(err)
	}


	assert(len(cartBytes) > 3, "cannot parse, file is too small")

	// TODO: config file instead
	devMode := fileExists("devmode")

	var emu dmgo.Emulator
	windowTitle := "dmgo"

	fileMagic := string(cartBytes[:3])
	if fileMagic == "GBS" {
		// nsf(e) file
		emu = dmgo.NewGbsPlayer(cartBytes, devMode)
	} else {
		// rom file

		cartInfo := dmgo.ParseCartInfo(cartBytes)
		if devMode {
			fmt.Printf("Game title: %q\n", cartInfo.Title)
			fmt.Printf("Cart type: %d\n", cartInfo.CartridgeType)
			fmt.Printf("Cart RAM size: %d\n", cartInfo.GetRAMSize())
			fmt.Printf("Cart ROM size: %d\n", cartInfo.GetROMSize())
		}

		emu = dmgo.NewEmulator(cartBytes, devMode)
		windowTitle = fmt.Sprintf("dmgo - %q", cartInfo.Title)
	}

	glimmer.InitDisplayLoop(windowTitle, 160*4, 144*4, 160, 144, func(sharedState *glimmer.WindowState) {
		startEmu(cartFilename, sharedState, emu)
	})
}

func readZipFileOrDie(filename string) []byte {
	zipReader, err := zip.OpenReader(filename)
	dieIf(err)

	// TODO: make list of filenames, sort abc, grab first alpha-sorted file with .gb or .gbc in name
	f := zipReader.File[0]
	fmt.Printf("unzipping first file found: %q\n", f.FileHeader.Name)
	cartReader, err := f.Open()
	dieIf(err)
	cartBytes, err := ioutil.ReadAll(cartReader)
	dieIf(err)

	cartReader.Close()
	zipReader.Close()
	return cartBytes
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func startHeadlessEmu(emu dmgo.Emulator) {
	// FIXME: settings are for debug right now
	ticker := time.NewTicker(17 * time.Millisecond)

	for {
		emu.Step()
		if emu.FlipRequested() {
			<-ticker.C
		}
	}
}

func startEmu(filename string, window *glimmer.WindowState, emu dmgo.Emulator) {

	snapshotPrefix := filename + ".snapshot"

	saveFilename := filename + ".sav"
	if saveFile, err := ioutil.ReadFile(saveFilename); err == nil {
		err = emu.SetCartRAM(saveFile)
		if err != nil {
			fmt.Println("error loading savefile,", err)
		} else {
			fmt.Println("loaded save!")
		}
	}

	audio, err := glimmer.OpenAudioBuffer(2, 8192, 44100, 16, 2)
	workingAudioBuffer := make([]byte, audio.BufferSize())
	dieIf(err)

	snapshotMode := 'x'

	newInput := dmgo.Input{}

	frameTimer := glimmer.MakeFrameTimer(1.0 / 60.0)

	lastSaveTime := time.Now()
	lastInputPollTime := time.Now()

	count := 0
	for {

		count++
		if count == 100 {
			count = 0
			now := time.Now()

			inputDiff := now.Sub(lastInputPollTime)

			if inputDiff > 8*time.Millisecond {
				window.InputMutex.Lock()
				bDown := window.CharIsDown('b')
				newInput = dmgo.Input{
					Joypad: dmgo.Joypad{
						Sel: bDown || window.CharIsDown('t'),
						Start: bDown || window.CharIsDown('y'),
						Up: window.CharIsDown('w'),
						Down: window.CharIsDown('s'),
						Left: window.CharIsDown('a'),
						Right: window.CharIsDown('d'),
						A: bDown || window.CharIsDown('k'),
						B: bDown || window.CharIsDown('j'),
					},
				}

				numDown := 'x'
				for r := '0'; r <= '9'; r++ {
					if window.CharIsDown(r) {
						numDown = r
						break
					}
				}
				if window.CharIsDown('m') {
					snapshotMode = 'm'
				} else if window.CharIsDown('l') {
					snapshotMode = 'l'
				}
				window.InputMutex.Unlock()

				if numDown > '0' && numDown <= '9' {
					snapFilename := snapshotPrefix + string(numDown)
					if snapshotMode == 'm' {
						snapshotMode = 'x'
						snapshot := emu.MakeSnapshot()
						ioutil.WriteFile(snapFilename, snapshot, os.FileMode(0644))
					} else if snapshotMode == 'l' {
						snapshotMode = 'x'
						snapBytes, err := ioutil.ReadFile(snapFilename)
						if err != nil {
							fmt.Println("failed to load snapshot:", err)
							continue
						}
						newEmu, err := emu.LoadSnapshot(snapBytes)
						if err != nil {
							fmt.Println("failed to load snapshot:", err)
							continue
						}
						emu = newEmu
					}
				}
				lastInputPollTime = time.Now()
			}
		}

		emu.UpdateInput(newInput)
		emu.Step()

		bufferAvailable := audio.BufferAvailable()

		// TODO: set this up so it's useful, but doesn't spam
		// if bufferAvailable == audio.BufferSize() {
		// fmt.Println("Platform AudioBuffer empty!")
		// }

		audioBufSlice := workingAudioBuffer[:bufferAvailable]
		audio.Write(emu.ReadSoundBuffer(audioBufSlice))

		if emu.FlipRequested() {
			window.RenderMutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.RenderMutex.Unlock()

			frameTimer.WaitForFrametime()
			if emu.InDevMode() {
				frameTimer.PrintStatsEveryXFrames(60*5)
			}

			if time.Now().Sub(lastSaveTime) > 5*time.Second {
				ram := emu.GetCartRAM()
				if len(ram) > 0 {
					ioutil.WriteFile(saveFilename, ram, os.FileMode(0644))
					lastSaveTime = time.Now()
				}
			}
		}
	}
}

func assert(test bool, msg string) {
	if !test {
		fmt.Println(msg)
		os.Exit(1)
	}
}

func dieIf(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
