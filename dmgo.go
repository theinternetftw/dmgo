package dmgo

import "fmt"

type cpuState struct {
	pc                     uint16
	sp                     uint16
	a, f, b, c, d, e, h, l byte
	mem                    mem

	lcd lcd
	apu apu

	inHaltMode bool
	inStopMode bool

	interruptMasterEnable bool

	vBlankInterruptEnabled  bool
	lcdStatInterruptEnabled bool
	timerInterruptEnabled   bool
	serialInterruptEnabled  bool
	joypadInterruptEnabled  bool
	dummyEnableBits         [3]bool

	vBlankIRQ  bool
	lcdStatIRQ bool
	timerIRQ   bool
	serialIRQ  bool
	joypadIRQ  bool

	serialTransferData            byte
	serialTransferStartFlag       bool
	serialTransferClockIsInternal bool

	timerOn           bool
	timerModuloReg    byte
	timerCounterReg   byte
	timerFreqSelector byte
	timerDivCycles    uint16 // div reg is top 8 bits of this

	joypad Joypad

	steps  uint
	cycles uint
}

// NOTE: timer is more complicated than this.
// See TCAGBD
func (cs *cpuState) runTimerCycle() {

	cs.timerDivCycles++

	if !cs.timerOn {
		return
	}

	cycleCount := map[byte]uint{
		0: 1024,
		1: 16,
		2: 64,
		3: 256,
	}[cs.timerFreqSelector]
	if cs.cycles&(cycleCount-1) == 0 {
		cs.timerCounterReg++
		if cs.timerCounterReg == 0 {
			cs.timerCounterReg = cs.timerModuloReg
			cs.timerIRQ = true
		}
	}
}

func (cs *cpuState) readTimerControlReg() byte {
	return boolBit(cs.timerOn, 2) | cs.timerFreqSelector
}
func (cs *cpuState) writeTimerControlReg(val byte) {
	cs.timerOn = val&0x04 != 0
	cs.timerFreqSelector = val & 0x03
}

func (cs *cpuState) readSerialControlReg() byte {
	return boolBit(cs.serialTransferStartFlag, 7) | boolBit(cs.serialTransferClockIsInternal, 0)
}
func (cs *cpuState) writeSerialControlReg(val byte) {
	cs.serialTransferStartFlag = val&0x80 != 0
	cs.serialTransferClockIsInternal = val&0x01 != 0
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
	lastVal := cs.joypad.readJoypadReg() & 0x0f
	if cs.joypad.readMask&0x01 == 0 {
		cs.joypad.Down = newJP.Down
		cs.joypad.Up = newJP.Up
		cs.joypad.Left = newJP.Left
		cs.joypad.Right = newJP.Right
	}
	if cs.joypad.readMask&0x10 == 0 {
		cs.joypad.Start = newJP.Start
		cs.joypad.Sel = newJP.Sel
		cs.joypad.B = newJP.B
		cs.joypad.A = newJP.A
	}
	newVal := cs.joypad.readJoypadReg() & 0x0f
	// this is correct behavior. it only triggers irq
	// if it goes from no-buttons-pressed to any-pressed.
	if lastVal == 0x0f && newVal < lastVal {
		cs.joypadIRQ = true
	}
}

// TODO: handle HALT hardware bug (see TCAGBD)
func (cs *cpuState) handleInterrupts() bool {

	var intFlag *bool
	var intAddr uint16
	if cs.vBlankInterruptEnabled && cs.vBlankIRQ {
		intFlag, intAddr = &cs.vBlankIRQ, 0x0040
	} else if cs.lcdStatInterruptEnabled && cs.lcdStatIRQ {
		intFlag, intAddr = &cs.lcdStatIRQ, 0x0048
	} else if cs.timerInterruptEnabled && cs.timerIRQ {
		intFlag, intAddr = &cs.timerIRQ, 0x0050
	} else if cs.serialInterruptEnabled && cs.serialIRQ {
		intFlag, intAddr = &cs.serialIRQ, 0x0058
	} else if cs.joypadInterruptEnabled && cs.joypadIRQ {
		intFlag, intAddr = &cs.joypadIRQ, 0x0060
	}

	if intFlag != nil {
		if cs.interruptMasterEnable {
			cs.interruptMasterEnable = false
			*intFlag = false
			cs.pushOp16(20, 0, cs.pc)
			cs.pc = intAddr
		}
		return true
	}
	return false
}

func (cs *cpuState) writeInterruptEnableReg(val byte) {
	boolsFromByte(val,
		&cs.dummyEnableBits[2],
		&cs.dummyEnableBits[1],
		&cs.dummyEnableBits[0],
		&cs.joypadInterruptEnabled,
		&cs.serialInterruptEnabled,
		&cs.timerInterruptEnabled,
		&cs.lcdStatInterruptEnabled,
		&cs.vBlankInterruptEnabled,
	)
}
func (cs *cpuState) readInterruptEnableReg() byte {
	return byteFromBools(
		cs.dummyEnableBits[2],
		cs.dummyEnableBits[1],
		cs.dummyEnableBits[0],
		cs.joypadInterruptEnabled,
		cs.serialInterruptEnabled,
		cs.timerInterruptEnabled,
		cs.lcdStatInterruptEnabled,
		cs.vBlankInterruptEnabled,
	)
}

func (cs *cpuState) writeInterruptFlagReg(val byte) {
	boolsFromByte(val,
		nil, nil, nil,
		&cs.joypadIRQ,
		&cs.serialIRQ,
		&cs.timerIRQ,
		&cs.lcdStatIRQ,
		&cs.vBlankIRQ,
	)
}
func (cs *cpuState) readInterruptFlagReg() byte {
	return byteFromBools(
		true, true, true,
		cs.joypadIRQ,
		cs.serialIRQ,
		cs.timerIRQ,
		cs.lcdStatIRQ,
		cs.vBlankIRQ,
	)
}

func (cs *cpuState) getZeroFlag() bool      { return cs.f&0x80 > 0 }
func (cs *cpuState) getAddSubFlag() bool    { return cs.f&0x40 > 0 }
func (cs *cpuState) getHalfCarryFlag() bool { return cs.f&0x20 > 0 }
func (cs *cpuState) getCarryFlag() bool     { return cs.f&0x10 > 0 }

func (cs *cpuState) setFlags(flags uint16) {

	setZero, clearZero := flags&0x1000 != 0, flags&0xf000 == 0
	setAddSub, clearAddSub := flags&0x100 != 0, flags&0xf00 == 0
	setHalfCarry, clearHalfCarry := flags&0x10 != 0, flags&0xf0 == 0
	setCarry, clearCarry := flags&0x1 != 0, flags&0xf == 0

	if setZero {
		cs.f |= 0x80
	} else if clearZero {
		cs.f &^= 0x80
	}
	if setAddSub {
		cs.f |= 0x40
	} else if clearAddSub {
		cs.f &^= 0x40
	}
	if setHalfCarry {
		cs.f |= 0x20
	} else if clearHalfCarry {
		cs.f &^= 0x20
	}
	if setCarry {
		cs.f |= 0x10
	} else if clearCarry {
		cs.f &^= 0x10
	}
}

func (cs *cpuState) getAF() uint16 { return (uint16(cs.a) << 8) | uint16(cs.f) }
func (cs *cpuState) getBC() uint16 { return (uint16(cs.b) << 8) | uint16(cs.c) }
func (cs *cpuState) getDE() uint16 { return (uint16(cs.d) << 8) | uint16(cs.e) }
func (cs *cpuState) getHL() uint16 { return (uint16(cs.h) << 8) | uint16(cs.l) }

func (cs *cpuState) setAF(val uint16) { cs.a, cs.f = byte(val>>8), byte(val) }
func (cs *cpuState) setBC(val uint16) { cs.b, cs.c = byte(val>>8), byte(val) }
func (cs *cpuState) setDE(val uint16) { cs.d, cs.e = byte(val>>8), byte(val) }
func (cs *cpuState) setHL(val uint16) { cs.h, cs.l = byte(val>>8), byte(val) }

func (cs *cpuState) setSP(val uint16) { cs.sp = val }
func (cs *cpuState) setPC(val uint16) { cs.pc = val }

func newState(cart []byte) *cpuState {
	cartInfo := ParseCartInfo(cart)
	if cartInfo.cgbOnly() {
		fatalErr("CGB-only not supported yet")
	}
	mem := mem{
		cart:    cart,
		cartRAM: make([]byte, cartInfo.GetRAMSize()),
		mbc:     makeMBC(cartInfo),
	}
	state := cpuState{mem: mem}
	state.mem.mbc.Init(&state.mem)
	state.initRegisters()
	state.lcd.init()
	state.apu.init()
	return &state
}

func (cs *cpuState) initRegisters() {
	// NOTE: these are DMG values,
	// others are different, see
	// TCAGBD
	cs.setAF(0x01b0)
	cs.setBC(0x0013)
	cs.setDE(0x00d8)
	cs.setHL(0x014d)
	cs.setSP(0xfffe)
	cs.setPC(0x0100)
}

// much TODO
func (cs *cpuState) runCycles(ncycles uint) {
	for i := uint(0); i < ncycles; i++ {
		cs.cycles++
		cs.runTimerCycle()
	}
	cs.lcd.runCycles(cs, ncycles)
}

// Emulator exposes the publi1 facing fns for an emulation session
type Emulator interface {
	Framebuffer() []byte
	FlipRequested() bool
	FrameWaitRequested() bool
	GetCartRAM() []byte
	SetCartRAM([]byte) error
	UpdateInput(input Input)
	Step()
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

// GetCartRAM returns the current state of external RAM
func (cs *cpuState) GetCartRAM() []byte {
	return cs.mem.cartRAM
}

// SetCartRAM attempts to set the RAM, returning error if size not correct
func (cs *cpuState) SetCartRAM(ram []byte) error {
	if len(cs.mem.cartRAM) == len(ram) {
		copy(cs.mem.cartRAM, ram)
		return nil
	}
	// TODO: better checks (e.g. real format, cart title/checksum, etc.)
	return fmt.Errorf("ram size mismatch")
}

func (cs *cpuState) UpdateInput(input Input) {
	cs.updateJoypad(input.Joypad)
}

// Framebuffer returns the current state of the lcd screen
func (cs *cpuState) Framebuffer() []byte {
	return cs.lcd.framebuffer
}

// FlipRequested indicates if a draw request is pending
// and clears it before returning
func (cs *cpuState) FlipRequested() bool {
	val := cs.lcd.flipRequested
	cs.lcd.flipRequested = false
	return val
}

// FrameWaitRequested indicates, separatate from an actual
// draw event, whether or not there should be a wait until
// when the frame would have been drawn
func (cs *cpuState) FrameWaitRequested() bool {
	val := cs.lcd.frameWaitRequested
	cs.lcd.frameWaitRequested = false
	return val
}

// Step steps the emulator one instruction
func (cs *cpuState) Step() {

	ieAndIfFlagMatch := cs.handleInterrupts()
	if cs.inHaltMode {
		if ieAndIfFlagMatch {
			cs.runCycles(4)
			cs.inHaltMode = false
		} else {
			cs.runCycles(4)
			return
		}
	}

	// if !cs.inHaltMode && cs.steps&0x2ffff == 0 {
	// if true {
	// 	fmt.Println(cs.debugStatusLine())
	// }

	// TODO: correct behavior, e.g. check for
	// button press only. but for now lets
	// treat it like halt
	if ieAndIfFlagMatch && cs.inStopMode {
		cs.runCycles(4)
		cs.inHaltMode = false
	}
	if cs.inStopMode {
		cs.runCycles(4)
	}

	cs.stepOpcode()
}

func fatalErr(v ...interface{}) {
	fmt.Println(v...)
	panic("fatalErr()")
}
