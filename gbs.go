package dmgo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type gbsPlayer struct {
	cpuState
	Hdr              gbsHeader
	PlayCallInterval float64
	LastPlayCall     time.Time
	CurrentSong      byte
	CurrentSongStart time.Time
	Paused           bool
	PauseStartTime   time.Time
	DbgTerminal      dbgTerminal
	DbgScreen        [160 * 144 * 4]byte
}

func (gp *gbsPlayer) GetCartRAM() []byte { return nil }
func (gp *gbsPlayer) SetCartRAM(ram []byte) error {
	return fmt.Errorf("saves not implemented for GBSs")
}
func (gp *gbsPlayer) MakeSnapshot() []byte { return nil }
func (gp *gbsPlayer) LoadSnapshot(snapBytes []byte) (Emulator, error) {
	return nil, fmt.Errorf("snapshots not implemented for GBSs")
}

type gbsHeader struct {
	Magic           [3]byte
	Version         byte
	NumSongs        byte
	StartSong       byte
	LoadAddr        uint16
	InitAddr        uint16
	PlayAddr        uint16
	StackPtr        uint16
	TimerModulo     byte
	TimerControl    byte
	TitleString     [32]byte
	AuthorString    [32]byte
	CopyrightString [32]byte
}

func parseGbs(gbs []byte) (gbsHeader, []byte, error) {
	hdr := gbsHeader{}
	if err := readStructLE(gbs, &hdr); err != nil {
		return gbsHeader{}, nil, fmt.Errorf("gbs player error\n%s", err.Error())
	}
	if hdr.Version != 1 {
		return gbsHeader{}, nil, fmt.Errorf("gbs player error\nunsupported gbs version: %v", hdr.Version)
	}
	data := gbs[0x70:]
	return hdr, data, nil
}

func readStructLE(structBytes []byte, iface interface{}) error {
	return binary.Read(bytes.NewReader(structBytes), binary.LittleEndian, iface)
}

// NewGbsPlayer creates an gbsPlayer session
func NewGbsPlayer(gbs []byte) Emulator {

	var hdr gbsHeader
	var data []byte
	var err error
	hdr, data, err = parseGbs(gbs)
	if err != nil {
		return NewErrEmu(fmt.Sprintf("gbs player error\n%s", err.Error()))
	}

	cart := append(make([]byte, hdr.LoadAddr), data...)
	paddingNeeded := len(cart) % 16 * 1024
	if paddingNeeded != 0 {
		cart = append(cart, make([]byte, paddingNeeded)...)
	}

	gp := gbsPlayer{
		cpuState: cpuState{
			Mem: mem{
				mbc:                   &gbsMBC{},
				cart:                  cart,
				CartRAM:               make([]byte, 8192),
				InternalRAMBankNumber: 1,
			},
		},
		Hdr: hdr,
	}
	gp.DbgTerminal = dbgTerminal{w: 160, h: 144, screen: gp.DbgScreen[:]}

	fmt.Println("uses timer:", gp.usesTimer())

	timerMod := float64(gp.Hdr.TimerModulo)

	if gp.Hdr.TimerControl&0x80 > 0 {
		// correct all timers for our dmg timer speeds
		timerMod /= 2
		fmt.Println("GBC Speed Requested")
	}

	if gp.usesTimer() {
		counterRate := []float64{4096, 262144, 65536, 16384}[gp.Hdr.TimerControl&0x03]
		gp.PlayCallInterval = 1.0 / (counterRate / (256 - timerMod))
	} else {
		gp.PlayCallInterval = 1.0 / 59.7
	}

	gp.init()

	gp.patchRsts()

	gp.initTune(gp.Hdr.StartSong - 1)

	gp.updateScreen()

	return &gp
}

func (gp *gbsPlayer) usesTimer() bool {
	return gp.Hdr.TimerControl&0x04 == 0x04
}

func (gp *gbsPlayer) patchRsts() {
	addrs := []uint16{0x00, 0x08, 0x10, 0x18, 0x20, 0x28, 0x30, 0x38}
	for _, addr := range addrs {
		newAddr := gp.Hdr.LoadAddr + addr
		patch := []byte{0xcd, byte(newAddr), byte(newAddr >> 8)}
		copy(gp.Mem.cart[addr:addr+3], patch)
	}
}

func (gp *gbsPlayer) initTune(songNum byte) {

	gp.F = 0
	gp.B, gp.C = 0, 0
	gp.D, gp.E = 0, 0
	gp.H, gp.L = 0, 0

	for i := uint16(0xa000); i < 0xfe00; i++ {
		gp.write(i, 0)
	}

	for i := uint16(0xff80); i < 0xffff; i++ {
		gp.write(i, 0)
	}

	gp.initIORegs()

	gp.APU.Sounds[0].RestartRequested = false
	gp.APU.Sounds[1].RestartRequested = false
	gp.APU.Sounds[2].RestartRequested = false
	gp.APU.Sounds[3].RestartRequested = false

	gp.A = songNum

	// force a call to INIT
	gp.SP = gp.Hdr.StackPtr
	gp.pushOp16(0, 0, 0x0130)
	gp.PC = gp.Hdr.InitAddr
	for gp.PC != 0x0130 {
		gp.step()
	}

	gp.CurrentSong = songNum
	gp.CurrentSongStart = time.Now()
}

func (gp *gbsPlayer) updateScreen() {

	gp.DbgTerminal.clearScreen()

	gp.DbgTerminal.setPos(0, 1)

	gp.DbgTerminal.writeString("GBS Player\n\n")
	gp.DbgTerminal.writeString(string(gp.Hdr.TitleString[:]) + "\n")
	gp.DbgTerminal.writeString(string(gp.Hdr.AuthorString[:]) + "\n")

	copyStr := string(gp.Hdr.CopyrightString[:])
	copyParts := strings.SplitN(copyStr, " ", 2)
	if len(copyParts) > 1 {
		// almost always improves presentation
		gp.DbgTerminal.writeString(copyParts[0] + "\n")
		gp.DbgTerminal.writeString(copyParts[1] + "\n")
	} else {
		gp.DbgTerminal.writeString(copyStr + "\n")
	}

	gp.DbgTerminal.newline()

	gp.DbgTerminal.writeString(fmt.Sprintf("Track %02d/%02d\n", gp.CurrentSong+1, gp.Hdr.NumSongs))

	nowTime := int(time.Now().Sub(gp.CurrentSongStart).Seconds())
	nowTimeStr := fmt.Sprintf("%02d:%02d", nowTime/60, nowTime%60)

	gp.DbgTerminal.writeString(fmt.Sprintf("%s", nowTimeStr))

	if gp.Paused {
		gp.DbgTerminal.writeString(" *PAUSED*\n")
	} else {
		gp.DbgTerminal.newline()
	}

	gp.LCD.FlipRequested = true
}

func (gp *gbsPlayer) prevSong() {
	if gp.CurrentSong > 0 {
		gp.CurrentSong--
		gp.initTune(gp.CurrentSong)
		gp.updateScreen()
	}
}
func (gp *gbsPlayer) nextSong() {
	if gp.CurrentSong < gp.Hdr.NumSongs-1 {
		gp.CurrentSong++
		gp.initTune(gp.CurrentSong)
		gp.updateScreen()
	}
}
func (gp *gbsPlayer) togglePause() {
	gp.Paused = !gp.Paused
	if gp.Paused {
		gp.PauseStartTime = time.Now()
	} else {
		gp.CurrentSongStart = gp.CurrentSongStart.Add(time.Now().Sub(gp.PauseStartTime))
	}
	gp.updateScreen()
}

var lastInput time.Time

func (gp *gbsPlayer) UpdateInput(input Input) {
	now := time.Now()
	if now.Sub(lastInput).Seconds() > 0.20 {
		if input.Joypad.Left {
			gp.prevSong()
			lastInput = now
		}
		if input.Joypad.Right {
			gp.nextSong()
			lastInput = now
		}
		if input.Joypad.Start {
			gp.togglePause()
			lastInput = now
		}
	}
}

var lastScreenUpdate time.Time

func (gp *gbsPlayer) Step() {
	if !gp.Paused {

		now := time.Now()
		if now.Sub(lastScreenUpdate) >= 100*time.Millisecond {
			lastScreenUpdate = now
			gp.updateScreen()
		}

		if gp.PC == 0x0130 {
			if now.Sub(gp.LastPlayCall).Seconds() > gp.PlayCallInterval {
				gp.LastPlayCall = now
				gp.SP = gp.Hdr.StackPtr
				gp.pushOp16(0, 0, 0x0130)
				gp.PC = gp.Hdr.PlayAddr
			}
		}

		if gp.PC != 0x0130 {
			gp.step()
		} else {
			gp.runCycles(4)
		}
	}
}

func (gp *gbsPlayer) ReadSoundBuffer(toFill []byte) []byte {
	buf := gp.APU.buffer.read(toFill)
	return buf
}

func (gp *gbsPlayer) Framebuffer() []byte {
	return gp.DbgScreen[:]
}
