package main

import (
	"theinternetftw.com/dmgo"
	"theinternetftw.com/dmgo/profiling"
	"theinternetftw.com/dmgo/windowing"

	"fmt"
	"io/ioutil"
	"os"
)

func main() {

	defer profiling.Start().Stop()

	assert(len(os.Args) == 2, "usage: ./dmgo ROM_FILENAME")
	cartFilename := os.Args[1]

	cartBytes, err := ioutil.ReadFile(cartFilename)
	dieIf(err)

	cartInfo := dmgo.ParseCartInfo(cartBytes)
	fmt.Printf("%q\n", cartInfo.Title)

	windowing.InitDisplayLoop(640, 576, func(sharedState *windowing.SharedState) {
		startEmu(sharedState, cartBytes)
	})
}

func startEmu(sharedState *windowing.SharedState, cartBytes []byte) {
	emu := dmgo.NewEmulator(cartBytes)
	for {
		emu.Step()
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
