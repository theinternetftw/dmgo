package main

import (
	"theinternetftw.com/dmgo"
	"theinternetftw.com/dmgo/profiling"
	"theinternetftw.com/dmgo/windowing"

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

	windowing.InitDisplayLoop(160*4, 144*4, 160, 144, func(sharedState *windowing.SharedState) {
		startEmu(sharedState, cartBytes)
	})
}

// NOTE: assumes you have the mutex when you call
func makeInput(window *windowing.SharedState) dmgo.Input {
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

func startEmu(window *windowing.SharedState, cartBytes []byte) {
	emu := dmgo.NewEmulator(cartBytes)

	// FIXME: settings are for debug right now
	ticker := time.NewTicker(33*time.Millisecond)

	for {
		window.Mutex.Lock()
		newInput := makeInput(window)
		window.Mutex.Unlock()

		emu.UpdateInput(newInput)
		emu.Step()

		if emu.FlipRequested() {
			window.Mutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.Mutex.Unlock()
			<-ticker.C
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
