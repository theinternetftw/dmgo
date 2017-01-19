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

	assert(len(cartBytes) > 3, "cannot parse file, illegal header")

	var emu dmgo.Emulator

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
	}

	platform.InitDisplayLoop(160*4, 144*4, 160, 144, func(sharedState *platform.WindowState) {
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
	lastVBlankTime := time.Now()
	lastSaveTime := time.Now()

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

	snapshotMode := 'x'

	for {
		window.Mutex.Lock()
		newInput := dmgo.Input {
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

		emu.UpdateInput(newInput)
		emu.Step()

		bufferAvailable := audio.BufferAvailable()
		if bufferAvailable == audio.BufferSize() {
			fmt.Println("Platform AudioBuffer empty!")
		}
		workingAudioBuffer = workingAudioBuffer[:bufferAvailable]
		audio.Write(emu.ReadSoundBuffer(workingAudioBuffer))

		if emu.FlipRequested() {
			window.Mutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.Mutex.Unlock()

			spent := time.Now().Sub(lastVBlankTime)
			toWait := 17*time.Millisecond - spent
			if toWait > time.Duration(0) {
				<-time.NewTimer(toWait).C
			}
			lastVBlankTime = time.Now()
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
