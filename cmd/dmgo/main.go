package main

import (
	"theinternetftw.com/dmgo"
	"theinternetftw.com/dmgo/profiling"
	"theinternetftw.com/dmgo/platform"

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

	cartInfo := dmgo.ParseCartInfo(cartBytes)
	fmt.Printf("%q\n", cartInfo.Title)
	fmt.Printf("Cart type: %d\n", cartInfo.CartridgeType)
	fmt.Printf("Cart RAM size: %d\n", cartInfo.GetRAMSize())
	fmt.Printf("Cart ROM size: %d\n", cartInfo.GetROMSize())

	platform.InitDisplayLoop(160*4, 144*4, 160, 144, func(sharedState *platform.WindowState) {
		startEmu(cartFilename, sharedState, cartBytes)
	})
}

// NOTE: assumes you have the mutex when you call
func makeInput(window *platform.WindowState) dmgo.Input {
	return dmgo.Input {
		Joypad: dmgo.Joypad {
			Sel: window.CharIsDown('t'),
			Start: window.CharIsDown('y'),
			Up: window.CharIsDown('w'),
			Down: window.CharIsDown('s'),
			Left: window.CharIsDown('a'),
			Right: window.CharIsDown('d'),
			A: window.CharIsDown('j'),
			B: window.CharIsDown('k'),
		},
	}
}

func startHeadlessEmu(cartBytes []byte) {
	emu := dmgo.NewEmulator(cartBytes)
	// FIXME: settings are for debug right now
	ticker := time.NewTicker(17*time.Millisecond)

	for {
		emu.Step()
		if emu.FlipRequested() {
			<-ticker.C
		}
	}
}

func startEmu(filename string, window *platform.WindowState, cartBytes []byte) {
	emu := dmgo.NewEmulator(cartBytes)

	// FIXME: settings are for debug right now
	lastVBlankTime := time.Now()
	lastSaveTime := time.Now()

	saveFilename := filename + ".sav"
	if saveFile, err := ioutil.ReadFile(saveFilename); err == nil {
		err = emu.SetCartRAM(saveFile)
		if err != nil {
			fmt.Println("error loading savefile,", err)
		}
		fmt.Println("loaded save!")
	}

	audio, err := platform.OpenAudioBuffer(4, 8192, 44100, 16, 2)
	dieIf(err)

	for {
		window.Mutex.Lock()
		newInput := makeInput(window)
		window.Mutex.Unlock()

		emu.UpdateInput(newInput)
		emu.Step()

		//emuSound := emu.GetSoundBuffer(audio.BufferAvailable())
		emuSound := emu.GetSoundBuffer(8192*2*2*64)
		if len(emuSound) > 0 {
			audio.Write(emuSound)
		}

		if emu.FlipRequested() {
			window.Mutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.Mutex.Unlock()
		}
		if emu.FrameWaitRequested() {
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
