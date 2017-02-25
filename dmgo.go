package dmgo

import (
	"fmt"
)

type cpuState struct {
	// everything here marshalled for snapshot

	PC                     uint16
	SP                     uint16
	A, F, B, C, D, E, H, L byte
	Mem                    mem

	Title          string
	HeaderChecksum byte

	LCD lcd
	APU apu

	InHaltMode bool
	InStopMode bool

	CGBMode bool

	IRDataReadEnable bool
	IRSendDataEnable bool

	InterruptMasterEnable bool
	MasterEnableRequested bool

	VBlankInterruptEnabled  bool
	LCDStatInterruptEnabled bool
	TimerInterruptEnabled   bool
	SerialInterruptEnabled  bool
	JoypadInterruptEnabled  bool
	DummyEnableBits         [3]bool

	VBlankIRQ  bool
	LCDStatIRQ bool
	TimerIRQ   bool
	SerialIRQ  bool
	JoypadIRQ  bool

	SerialTransferData            byte
	SerialTransferStartFlag       bool
	SerialTransferClockIsInternal bool
	SerialClock                   uint16
	SerialBitsTransferred         byte

	TimerOn           bool
	TimerModuloReg    byte
	TimerCounterReg   byte
	TimerFreqSelector byte
	TimerDivCycles    uint16 // div reg is top 8 bits of this

	Joypad Joypad

	Steps  uint
	Cycles uint
}

func (cs *cpuState) runSerialCycle() {
	if !cs.SerialTransferStartFlag {
		cs.SerialBitsTransferred = 0
		cs.SerialClock = 0
		return
	}
	if !cs.SerialTransferClockIsInternal {
		// no real link cable, so wait forever
		// (hopefully til game times out transfer)
		return
	}
	cs.SerialClock++
	if cs.SerialClock == 512 { // 8192Hz
		cs.SerialClock = 0
		cs.SerialTransferData <<= 1
		// emulate a disconnected cable
		cs.SerialTransferData |= 0x01
		cs.SerialBitsTransferred++
		if cs.SerialBitsTransferred == 8 {
			cs.SerialBitsTransferred = 0
			cs.SerialClock = 0
			cs.SerialIRQ = true
		}
	}
}

// NOTE: timer is more complicated than this.
// See TCAGBD
func (cs *cpuState) runTimerCycle() {

	cs.TimerDivCycles++

	if !cs.TimerOn {
		return
	}

	cycleCount := [...]uint{
		1024, 16, 64, 256,
	}[cs.TimerFreqSelector]
	if cs.Cycles&(cycleCount-1) == 0 {
		cs.TimerCounterReg++
		if cs.TimerCounterReg == 0 {
			cs.TimerCounterReg = cs.TimerModuloReg
			cs.TimerIRQ = true
		}
	}
}

func (cs *cpuState) readTimerControlReg() byte {
	return 0xf8 | boolBit(cs.TimerOn, 2) | cs.TimerFreqSelector
}
func (cs *cpuState) writeTimerControlReg(val byte) {
	cs.TimerOn = val&0x04 != 0
	cs.TimerFreqSelector = val & 0x03
}

func (cs *cpuState) readSerialControlReg() byte {
	return 0x7e | boolBit(cs.SerialTransferStartFlag, 7) | boolBit(cs.SerialTransferClockIsInternal, 0)
}
func (cs *cpuState) writeSerialControlReg(val byte) {
	cs.SerialTransferStartFlag = val&0x80 != 0
	cs.SerialTransferClockIsInternal = val&0x01 != 0
}

// Joypad represents the buttons on a gameboy
type Joypad struct {
	Sel      bool
	Start    bool
	Up       bool
	Down     bool
	Left     bool
	Right    bool
	A        bool
	B        bool
	readMask byte
}

func (jp *Joypad) writeJoypadReg(val byte) {
	jp.readMask = (val >> 4) & 0x03
}
func (jp *Joypad) readJoypadReg() byte {
	val := 0xc0 | (jp.readMask << 4) | 0x0f
	if jp.readMask&0x01 == 0 {
		val &^= boolBit(jp.Down, 3)
		val &^= boolBit(jp.Up, 2)
		val &^= boolBit(jp.Left, 1)
		val &^= boolBit(jp.Right, 0)
	}
	if jp.readMask&0x02 == 0 {
		val &^= boolBit(jp.Start, 3)
		val &^= boolBit(jp.Sel, 2)
		val &^= boolBit(jp.B, 1)
		val &^= boolBit(jp.A, 0)
	}
	return val
}

func (cs *cpuState) updateJoypad(newJP Joypad) {
	lastVal := cs.Joypad.readJoypadReg() & 0x0f
	if cs.Joypad.readMask&0x01 == 0 {
		cs.Joypad.Down = newJP.Down
		cs.Joypad.Up = newJP.Up
		cs.Joypad.Left = newJP.Left
		cs.Joypad.Right = newJP.Right
	}
	if cs.Joypad.readMask&0x10 == 0 {
		cs.Joypad.Start = newJP.Start
		cs.Joypad.Sel = newJP.Sel
		cs.Joypad.B = newJP.B
		cs.Joypad.A = newJP.A
	}
	newVal := cs.Joypad.readJoypadReg() & 0x0f
	// this is correct behavior. it only triggers irq
	// if it goes from no-buttons-pressed to any-pressed.
	if lastVal == 0x0f && newVal < lastVal {
		cs.JoypadIRQ = true
	}
}

// TODO: handle HALT hardware bug (see TCAGBD)
func (cs *cpuState) handleInterrupts() bool {

	var intFlag *bool
	var intAddr uint16
	if cs.VBlankInterruptEnabled && cs.VBlankIRQ {
		intFlag, intAddr = &cs.VBlankIRQ, 0x0040
	} else if cs.LCDStatInterruptEnabled && cs.LCDStatIRQ {
		intFlag, intAddr = &cs.LCDStatIRQ, 0x0048
	} else if cs.TimerInterruptEnabled && cs.TimerIRQ {
		intFlag, intAddr = &cs.TimerIRQ, 0x0050
	} else if cs.SerialInterruptEnabled && cs.SerialIRQ {
		intFlag, intAddr = &cs.SerialIRQ, 0x0058
	} else if cs.JoypadInterruptEnabled && cs.JoypadIRQ {
		intFlag, intAddr = &cs.JoypadIRQ, 0x0060
	}

	if intFlag != nil {
		if cs.InterruptMasterEnable {
			cs.InterruptMasterEnable = false
			*intFlag = false
			cs.pushOp16(20, 0, cs.PC)
			cs.PC = intAddr
		}
		return true
	}
	return false
}

func (cs *cpuState) writeInterruptEnableReg(val byte) {
	boolsFromByte(val,
		&cs.DummyEnableBits[2],
		&cs.DummyEnableBits[1],
		&cs.DummyEnableBits[0],
		&cs.JoypadInterruptEnabled,
		&cs.SerialInterruptEnabled,
		&cs.TimerInterruptEnabled,
		&cs.LCDStatInterruptEnabled,
		&cs.VBlankInterruptEnabled,
	)
}
func (cs *cpuState) readInterruptEnableReg() byte {
	return byteFromBools(
		cs.DummyEnableBits[2],
		cs.DummyEnableBits[1],
		cs.DummyEnableBits[0],
		cs.JoypadInterruptEnabled,
		cs.SerialInterruptEnabled,
		cs.TimerInterruptEnabled,
		cs.LCDStatInterruptEnabled,
		cs.VBlankInterruptEnabled,
	)
}

func (cs *cpuState) writeInterruptFlagReg(val byte) {
	boolsFromByte(val,
		nil, nil, nil,
		&cs.JoypadIRQ,
		&cs.SerialIRQ,
		&cs.TimerIRQ,
		&cs.LCDStatIRQ,
		&cs.VBlankIRQ,
	)
}
func (cs *cpuState) readInterruptFlagReg() byte {
	return byteFromBools(
		true, true, true,
		cs.JoypadIRQ,
		cs.SerialIRQ,
		cs.TimerIRQ,
		cs.LCDStatIRQ,
		cs.VBlankIRQ,
	)
}

func (cs *cpuState) writeIRPortReg(val byte) {
	cs.IRDataReadEnable = val&0xc0 == 0xc0
	cs.IRSendDataEnable = val&0x01 == 0x01
}
func (cs *cpuState) readIRPortReg() byte {
	out := byte(0)
	if cs.IRDataReadEnable {
		out |= 0xc2 // no data received
	}
	if cs.IRSendDataEnable {
		out |= 0x01
	}
	return out
}

func (cs *cpuState) getZeroFlag() bool      { return cs.F&0x80 > 0 }
func (cs *cpuState) getSubFlag() bool       { return cs.F&0x40 > 0 }
func (cs *cpuState) getHalfCarryFlag() bool { return cs.F&0x20 > 0 }
func (cs *cpuState) getCarryFlag() bool     { return cs.F&0x10 > 0 }

func (cs *cpuState) setFlags(flags uint16) {

	setZero, clearZero := flags&0x1000 != 0, flags&0xf000 == 0
	setSub, clearSub := flags&0x100 != 0, flags&0xf00 == 0
	setHalfCarry, clearHalfCarry := flags&0x10 != 0, flags&0xf0 == 0
	setCarry, clearCarry := flags&0x1 != 0, flags&0xf == 0

	if setZero {
		cs.F |= 0x80
	} else if clearZero {
		cs.F &^= 0x80
	}
	if setSub {
		cs.F |= 0x40
	} else if clearSub {
		cs.F &^= 0x40
	}
	if setHalfCarry {
		cs.F |= 0x20
	} else if clearHalfCarry {
		cs.F &^= 0x20
	}
	if setCarry {
		cs.F |= 0x10
	} else if clearCarry {
		cs.F &^= 0x10
	}
}

func (cs *cpuState) getAF() uint16 { return (uint16(cs.A) << 8) | uint16(cs.F) }
func (cs *cpuState) getBC() uint16 { return (uint16(cs.B) << 8) | uint16(cs.C) }
func (cs *cpuState) getDE() uint16 { return (uint16(cs.D) << 8) | uint16(cs.E) }
func (cs *cpuState) getHL() uint16 { return (uint16(cs.H) << 8) | uint16(cs.L) }

func (cs *cpuState) setAF(val uint16) {
	cs.A = byte(val >> 8)
	cs.F = byte(val) &^ 0x0f
}
func (cs *cpuState) setBC(val uint16) { cs.B, cs.C = byte(val>>8), byte(val) }
func (cs *cpuState) setDE(val uint16) { cs.D, cs.E = byte(val>>8), byte(val) }
func (cs *cpuState) setHL(val uint16) { cs.H, cs.L = byte(val>>8), byte(val) }

func (cs *cpuState) setSP(val uint16) { cs.SP = val }
func (cs *cpuState) setPC(val uint16) { cs.PC = val }

func newState(cart []byte) *cpuState {
	cartInfo := ParseCartInfo(cart)
	// if cartInfo.cgbOnly() {
	// 	fatalErr("CGB-only not supported yet")
	// }
	state := cpuState{
		Title:          cartInfo.Title,
		HeaderChecksum: cartInfo.HeaderChecksum,
		Mem: mem{
			cart:                  cart,
			CartRAM:               make([]byte, cartInfo.GetRAMSize()),
			InternalRAM:           make([]byte, 0x8000),
			InternalRAMBankNumber: 1,
			mbc: makeMBC(cartInfo),
		},
		LCD: lcd{
			VideoRAM: make([]byte, 0x2000), // only DMG size for now
		},
		CGBMode: cartInfo.cgbOptional() || cartInfo.cgbOnly(),
	}
	state.init()
	return &state
}

func (cs *cpuState) init() {
	// NOTE: these are DMG values,
	// others are different, see
	// TCAGBD
	if cs.CGBMode {
		cs.setAF(0x11b0)
	} else {
		cs.setAF(0x01b0)
	}
	cs.setBC(0x0013)
	cs.setDE(0x00d8)
	cs.setHL(0x014d)
	cs.setSP(0xfffe)
	cs.setPC(0x0100)

	cs.TimerDivCycles = 0xabcc

	cs.LCD.init()
	cs.APU.init()

	cs.Mem.mbc.Init(&cs.Mem)

	cs.initIORegs()

	cs.APU.Sounds[0].RestartRequested = false
	cs.APU.Sounds[1].RestartRequested = false
	cs.APU.Sounds[2].RestartRequested = false
	cs.APU.Sounds[3].RestartRequested = false

	cs.initVRAM()
	cs.VBlankIRQ = true
}

func (cs *cpuState) initIORegs() {
	cs.write(0xff10, 0x80)
	cs.write(0xff11, 0xbf)
	cs.write(0xff12, 0xf3)
	cs.write(0xff14, 0xbf)
	cs.write(0xff17, 0x3f)
	cs.write(0xff19, 0xbf)
	cs.write(0xff1a, 0x7f)
	cs.write(0xff1b, 0xff)
	cs.write(0xff1c, 0x9f)
	cs.write(0xff1e, 0xbf)
	cs.write(0xff20, 0xff)
	cs.write(0xff23, 0xbf)
	cs.write(0xff24, 0x77)
	cs.write(0xff25, 0xf3)
	cs.write(0xff26, 0xf1)

	cs.write(0xff40, 0x91)
	cs.write(0xff47, 0xfc)
	cs.write(0xff48, 0xff)
	cs.write(0xff49, 0xff)
}

func (cs *cpuState) initVRAM() {
	nibbleLookup := []byte{
		0x00, 0x03, 0x0c, 0x0f, 0x30, 0x33, 0x3c, 0x3f,
		0xc0, 0xc3, 0xcc, 0xcf, 0xf0, 0xf3, 0xfc, 0xff,
	}

	hdrTileData := []byte{}
	for i := 0x104; i < 0x104+48; i++ {
		packed := cs.read(uint16(i))
		b1, b2 := nibbleLookup[packed>>4], nibbleLookup[packed&0x0f]
		hdrTileData = append(hdrTileData, b1, 0, b1, 0, b2, 0, b2, 0)
	}

	// append boot rom tile data
	hdrTileData = append(hdrTileData,
		0x3c, 0x00, 0x42, 0x00, 0xb9, 0x00, 0xa5, 0x00, 0xb9, 0x00, 0xa5, 0x00, 0x42, 0x00, 0x3c, 0x00,
	)

	bootTileMap := []byte{
		0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
		0x19, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	}

	for i := range hdrTileData {
		cs.write(uint16(0x8010+i), hdrTileData[i])
	}
	for i := range bootTileMap {
		cs.write(uint16(0x9900+i), bootTileMap[i])
	}
}

func (cs *cpuState) runCycles(numCycles uint) {
	for i := uint(0); i < numCycles; i++ {
		cs.Cycles++
		cs.runTimerCycle()
		cs.runSerialCycle()
		cs.APU.runCycle(cs)
		cs.LCD.runCycle(cs)
	}
}

// Emulator exposes the public facing fns for an emulation session
type Emulator interface {
	Step()

	Framebuffer() []byte
	FlipRequested() bool

	UpdateInput(input Input)
	ReadSoundBuffer([]byte) []byte

	GetCartRAM() []byte
	SetCartRAM([]byte) error

	MakeSnapshot() []byte
	LoadSnapshot([]byte) (Emulator, error)
}

func (cs *cpuState) MakeSnapshot() []byte {
	return cs.makeSnapshot()
}

func (cs *cpuState) LoadSnapshot(snapBytes []byte) (Emulator, error) {
	return cs.loadSnapshot(snapBytes)
}

// NewEmulator creates an emulation session
func NewEmulator(cart []byte) Emulator {
	return newState(cart)
}

// Input covers all outside info sent to the Emulator
// TODO: add dt?
type Input struct {
	Joypad Joypad
}

// GetSoundBuffer returns a 44100hz * 16bit * 2ch sound buffer.
// A pre-sized buffer must be provided, which is returned resized
// if the buffer was less full than the length requested.
func (cs *cpuState) ReadSoundBuffer(toFill []byte) []byte {
	return cs.APU.buffer.read(toFill)
}

// GetCartRAM returns the current state of external RAM
func (cs *cpuState) GetCartRAM() []byte {
	return cs.Mem.CartRAM
}

// SetCartRAM attempts to set the RAM, returning error if size not correct
func (cs *cpuState) SetCartRAM(ram []byte) error {
	if len(cs.Mem.CartRAM) == len(ram) {
		copy(cs.Mem.CartRAM, ram)
		return nil
	}
	// TODO: better checks if possible (e.g. real format, cart title/checksum, etc.)
	return fmt.Errorf("ram size mismatch")
}

func (cs *cpuState) UpdateInput(input Input) {
	cs.updateJoypad(input.Joypad)
}

// Framebuffer returns the current state of the lcd screen
func (cs *cpuState) Framebuffer() []byte {
	return cs.LCD.framebuffer[:]
}

// FlipRequested indicates if a draw request is pending
// and clears it before returning
func (cs *cpuState) FlipRequested() bool {
	val := cs.LCD.FlipRequested
	cs.LCD.FlipRequested = false
	return val
}

var lastSP = int(-1)

func (cs *cpuState) debugLineOnStackChange() {
	if lastSP != int(cs.SP) {
		lastSP = int(cs.SP)
		fmt.Println(cs.debugStatusLine())
	}
}

// Step steps the emulator one instruction
func (cs *cpuState) Step() {
	cs.step()
}

var hitTarget = false

func (cs *cpuState) step() {
	ieAndIfFlagMatch := cs.handleInterrupts()
	if cs.InHaltMode {
		if ieAndIfFlagMatch {
			cs.runCycles(4)
			cs.InHaltMode = false
		} else {
			cs.runCycles(4)
			return
		}
	}

	// cs.debugLineOnStackChange()
	// if cs.Steps&0x2ffff == 0 {
	// if cs.PC == 0x4d19 {
	// 	hitTarget = true
	// }
	// if hitTarget {
	// 	fmt.Println(cs.debugStatusLine())
	// }

	// TODO: correct behavior, e.g. check for
	// button press only. but for now lets
	// treat it like halt
	if ieAndIfFlagMatch && cs.InStopMode {
		cs.runCycles(4)
		cs.InHaltMode = false
	}
	if cs.InStopMode {
		cs.runCycles(4)
	}

	// this is here to lag behind the request by
	// one instruction.
	if cs.MasterEnableRequested {
		cs.MasterEnableRequested = false
		cs.InterruptMasterEnable = true
	}

	cs.Steps++

	cs.stepOpcode()
}

func fatalErr(v ...interface{}) {
	fmt.Println(v...)
	panic("fatalErr()")
}
