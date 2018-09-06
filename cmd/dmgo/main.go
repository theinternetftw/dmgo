package main

import (
	"github.com/theinternetftw/dmgo"
	"github.com/theinternetftw/dmgo/profiling"
	"github.com/theinternetftw/dmgo/platform"

	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func main() {

	defer profiling.Start().Stop()

	assert(len(os.Args) == 2, "usage: ./dmgo ROM_FILENAME")
	cartFilename := os.Args[1]

	cartBytes, err := ioutil.ReadFile(cartFilename)
	dieIf(err)

	assert(len(cartBytes) > 3, "cannot parse, file is too small")

	var emu dmgo.Emulator
	windowTitle := "dmgo"

	fileMagic := string(cartBytes[:3])
	if fileMagic == "GBS" {
		// nsf(e) file
		emu = dmgo.NewGbsPlayer(cartBytes)
	} else {
		// rom file

		cartInfo := dmgo.ParseCartInfo(cartBytes)
		fmt.Printf("%q\n", cartInfo.Title)
		fmt.Printf("Cart type: %d\n", cartInfo.CartridgeType)
		fmt.Printf("Cart RAM size: %d\n", cartInfo.GetRAMSize())
		fmt.Printf("Cart ROM size: %d\n", cartInfo.GetROMSize())

		emu = dmgo.NewEmulator(cartBytes)
		windowTitle = cartInfo.Title + " - dmgo"
	}

	platform.InitDisplayLoop(windowTitle, 160*4, 144*4, 160, 144, func(sharedState *platform.WindowState) {
		startEmu(cartFilename, sharedState, emu)
	})
}

func startHeadlessEmu(emu dmgo.Emulator) {
	// FIXME: settings are for debug right now
	ticker := time.NewTicker(17*time.Millisecond)

	for {
		emu.Step()
		if emu.FlipRequested() {
			<-ticker.C
		}
	}
}

func startEmu(filename string, window *platform.WindowState, emu dmgo.Emulator) {

	// FIXME: settings are for debug right now
	lastFlipTime := time.Now()
	lastSaveTime := time.Now()
	lastInputPollTime := time.Now()

	timer := time.NewTimer(0)
	<-timer.C

	snapshotPrefix := filename + ".snapshot"

	saveFilename := filename + ".sav"
	if saveFile, err := ioutil.ReadFile(saveFilename); err == nil {
		err = emu.SetCartRAM(saveFile)
		if err != nil {
			fmt.Println("error loading savefile,", err)
		}
		fmt.Println("loaded save!")
	}

	audio, err := platform.OpenAudioBuffer(4, 4096, 44100, 16, 2)
	workingAudioBuffer := make([]byte, audio.BufferSize())
	dieIf(err)

	maxRDiff := time.Duration(0)
	maxFDiff := time.Duration(0)
	frameCount := 0

	frametimeGoal := 1.0/60.0

	snapshotMode := 'x'

	newInput := dmgo.Input{}

	for {

		now := time.Now()

		inputDiff := now.Sub(lastInputPollTime)

		if inputDiff > 8*time.Millisecond {
			window.Mutex.Lock()
			newInput = dmgo.Input {
				Joypad: dmgo.Joypad {
					Sel:  window.CharIsDown('t'), Start: window.CharIsDown('y'),
					Up:   window.CharIsDown('w'), Down:  window.CharIsDown('s'),
					Left: window.CharIsDown('a'), Right: window.CharIsDown('d'),
					A:    window.CharIsDown('k'), B:     window.CharIsDown('j'),
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
			window.Mutex.Unlock()

			if numDown > '0' && numDown <= '9' {
				snapFilename := snapshotPrefix+string(numDown)
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
			window.Mutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.Mutex.Unlock()

			rDiff := time.Now().Sub(lastFlipTime)

			// hack to get better accuracy, could do
			// a two stage wait with spin at the end,
			// but that really adds to the cycles
			fudge := 2000000*time.Nanosecond

			toWait := 16600000*time.Nanosecond - rDiff - fudge
			if toWait > time.Duration(0) {
				<-time.NewTimer(toWait).C
			}

			frameStart := lastFlipTime
			lastFlipTime = time.Now()

			fDiff := time.Now().Sub(frameStart)
			if rDiff > maxRDiff {
				maxRDiff = rDiff
			}
			if fDiff > maxFDiff {
				maxFDiff = fDiff
			}

			frameCount++
			if frameCount & 0xff == 0 {
				fmt.Printf("maxRTime %.4f, maxFTime %.4f\n", maxRDiff.Seconds(), maxFDiff.Seconds())
				maxRDiff = 0
				maxFDiff = 0
				if frametimeGoal == 0 {
					frametimeGoal = 1
				}
			}

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
