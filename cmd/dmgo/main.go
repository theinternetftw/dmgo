package main

import (
	"github.com/theinternetftw/dmgo"
	"github.com/theinternetftw/dmgo/profiling"
	"github.com/theinternetftw/glimmer"

	"archive/zip"
    "bytes"
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

	audio, audioErr := glimmer.OpenAudioBuffer(glimmer.OpenAudioBufferOptions{
        OutputBufDuration: 25*time.Millisecond,
        SamplesPerSecond: 44100,
        BitsPerSample: 16,
        ChannelCount: 2,
    })
    dieIf(audioErr)

    snapshotPrefix := cartFilename + ".snapshot"
    saveFilename := cartFilename + ".sav"

	if saveFile, err := ioutil.ReadFile(saveFilename); err == nil {
		err = emu.SetCartRAM(saveFile)
		if err != nil {
			fmt.Println("error loading savefile,", err)
		} else {
			fmt.Println("loaded save!")
		}
	}

    session := sessionState{
        snapshotPrefix: snapshotPrefix,
        saveFilename: saveFilename,
        frameTimer: glimmer.MakeFrameTimer(),
        lastSaveTime: time.Now(),
        lastInputPollTime: time.Now(),
        audio: audio,
        emu: emu,
    }

	glimmer.InitDisplayLoop(glimmer.InitDisplayLoopOptions{
        WindowTitle: windowTitle,
        RenderWidth: 160, RenderHeight: 144,
        WindowWidth: 160*4, WindowHeight: 144*4,
        InitCallback: func(sharedState *glimmer.WindowState) {
            runEmu(&session, sharedState)
        },
    })
}

type sessionState struct {
    snapshotMode rune
    snapshotPrefix string
    saveFilename string
    audio *glimmer.AudioBuffer
    latestInput dmgo.Input
    frameTimer glimmer.FrameTimer
    lastSaveTime time.Time
    lastInputPollTime time.Time
    ticksSincePollingInput int
    lastSaveRAM []byte
    emu dmgo.Emulator
    currentNumFrames int
    audioBytesProduced int
}

var maxWaited time.Duration

func runEmu(session *sessionState, window *glimmer.WindowState) {

    var audioChunkBuf []byte
    audioBufModifier := 0
    audioPrevReadLen := <-session.audio.ReadLenNotifier
    audioToGen := audioPrevReadLen + audioBufModifier

	for {
		session.ticksSincePollingInput++
		if session.ticksSincePollingInput == 100 {
			session.ticksSincePollingInput = 0
			now := time.Now()

			inputDiff := now.Sub(session.lastInputPollTime)

			if inputDiff > 8*time.Millisecond {
				session.lastInputPollTime = now

				window.InputMutex.Lock()
				bDown := window.CharIsDown('b')
				session.latestInput = dmgo.Input{
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
					session.snapshotMode = 'm'
				} else if window.CharIsDown('l') {
					session.snapshotMode = 'l'
				}
				window.InputMutex.Unlock()

				if numDown > '0' && numDown <= '9' {
					snapFilename := session.snapshotPrefix + string(numDown)
					if session.snapshotMode == 'm' {
						session.snapshotMode = 'x'
						snapshot := session.emu.MakeSnapshot()
						ioutil.WriteFile(snapFilename, snapshot, os.FileMode(0644))
					} else if session.snapshotMode == 'l' {
						session.snapshotMode = 'x'
						snapBytes, err := ioutil.ReadFile(snapFilename)
						if err != nil {
							fmt.Println("failed to load snapshot:", err)
							continue
						}
						newEmu, err := session.emu.LoadSnapshot(snapBytes)
						if err != nil {
							fmt.Println("failed to load snapshot:", err)
							continue
						}
						session.emu = newEmu
					}
				}
                session.emu.UpdateInput(session.latestInput)
			}
		}

        session.emu.Step()
        bufInfo := session.emu.GetSoundBufferInfo()
        if bufInfo.IsValid && bufInfo.UsedSize >= audioToGen {
            if cap(audioChunkBuf) < audioToGen {
                audioChunkBuf = make([]byte, audioToGen)
            }
            session.audio.Write(session.emu.ReadSoundBuffer(audioChunkBuf[:audioToGen]))
        }

		if session.emu.FlipRequested() {
			window.RenderMutex.Lock()
			copy(window.Pix, session.emu.Framebuffer())
			window.RenderMutex.Unlock()

            session.frameTimer.MarkRenderComplete()

            session.currentNumFrames++

            start := time.Now()
            if session.audio.GetLenUnplayedData() > audioPrevReadLen+4 {
                for session.audio.GetLenUnplayedData() > audioPrevReadLen+4 {
                    audioPrevReadLen = <- session.audio.ReadLenNotifier
                }
                if audioBufModifier > -4 {
                    audioBufModifier -= 4
                }
            } else {
                if audioBufModifier < 2*audioPrevReadLen {
                    audioBufModifier += 4
                }
            }
            audioToGen = audioPrevReadLen + audioBufModifier
            audioDiff := time.Now().Sub(start)
            if audioDiff > maxWaited {
                maxWaited = audioDiff
            }
            if session.currentNumFrames & 0x3f == 0 {
                // fmt.Println("[dmgo] max waited for audio:", maxWaited, "buf modifier now:", audioBufModifier)
                maxWaited = time.Duration(0)
            }

            session.frameTimer.MarkFrameComplete()

			if session.emu.InDevMode() {
				session.frameTimer.PrintStatsEveryXFrames(60*5)
			}

			if time.Now().Sub(session.lastSaveTime) > 5*time.Second {
				ram := session.emu.GetCartRAM()
				if len(ram) > 0 && !bytes.Equal(ram, session.lastSaveRAM) {
					ioutil.WriteFile(session.saveFilename, ram, os.FileMode(0644))
					session.lastSaveTime = time.Now()
                    session.lastSaveRAM = ram
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
