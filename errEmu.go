package dmgo

import (
	"fmt"
	"os"
)

type errEmu struct {
	terminal      dbgTerminal
	screen        [160 * 144 * 4]byte
	flipRequested bool

	devMode bool
}

// NewErrEmu returns an emulator that only shows an error message
func NewErrEmu(msg string) Emulator {
	emu := errEmu{}
	emu.terminal = dbgTerminal{w: 160, h: 144, screen: emu.screen[:]}
	os.Stderr.Write([]byte(msg + "\n"))
	emu.terminal.newline()
	emu.terminal.writeString(msg)
	emu.flipRequested = true
	return &emu
}

func (e *errEmu) GetCartRAM() []byte { return []byte{} }
func (e *errEmu) SetCartRAM([]byte) error {
	return fmt.Errorf("save not implemented for errEmu")
}
func (e *errEmu) MakeSnapshot() []byte { return nil }
func (e *errEmu) LoadSnapshot([]byte) (Emulator, error) {
	return nil, fmt.Errorf("snapshots not implemented for errEmu")
}
func (e *errEmu) ReadSoundBuffer(toFill []byte) []byte { return nil }
func (e *errEmu) UpdateInput(input Input)              {}
func (e *errEmu) Step()                                {}

func (e *errEmu) Framebuffer() []byte { return e.screen[:] }
func (e *errEmu) FlipRequested() bool {
	result := e.flipRequested
	e.flipRequested = false
	return result
}

func (e *errEmu) SetDevMode(b bool) { e.devMode = b }
func (e *errEmu) InDevMode() bool   { return e.devMode }
