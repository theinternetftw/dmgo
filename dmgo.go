package dmgo

import (
	"fmt"
	"os"
)

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

	timerModuloReg byte

	joypad joypad

	steps  uint
	cycles uint
}

func (cs *cpuState) readSerialControlReg() byte {
	return boolBit(cs.serialTransferStartFlag, 7) | boolBit(cs.serialTransferClockIsInternal, 0)
}
func (cs *cpuState) writeSerialControlReg(val byte) {
	cs.serialTransferStartFlag = val&0x80 != 0
	cs.serialTransferClockIsInternal = val&0x01 != 0
}

type joypad struct {
	sel      bool
	start    bool
	up       bool
	down     bool
	left     bool
	right    bool
	a        bool
	b        bool
	readMask byte
}

func (jp *joypad) writeJoypadReg(val byte) {
	jp.readMask = (val >> 4) & 0x03
}
func (jp *joypad) readJoypadReg() byte {
	val := 0xc0 & (jp.readMask << 4) & 0x0f
	if jp.readMask&0x01 == 0 {
		val &^= boolBit(jp.down, 3)
		val &^= boolBit(jp.up, 2)
		val &^= boolBit(jp.left, 1)
		val &^= boolBit(jp.right, 0)
	}
	if jp.readMask&0x10 == 0 {
		val &^= boolBit(jp.start, 3)
		val &^= boolBit(jp.sel, 2)
		val &^= boolBit(jp.b, 1)
		val &^= boolBit(jp.a, 0)
	}
	return val
}
func (jp *joypad) updateJoypad(cs *cpuState, newJP joypad) {
	lastVal := jp.readJoypadReg() & 0x0f
	if jp.readMask&0x01 == 0 {
		jp.down = newJP.down
		jp.up = newJP.up
		jp.left = newJP.left
		jp.right = newJP.right
	}
	if jp.readMask&0x10 == 0 {
		jp.start = newJP.start
		jp.sel = newJP.sel
		jp.b = newJP.b
		jp.a = newJP.a
	}
	newVal := jp.readJoypadReg() & 0x0f
	// this is correct behavior. it only triggers
	// irq if it goes from no-buttons-pressed to
	// any-buttons-pressed.
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

const entryPoint = 0x100

func newState(cart []byte) *cpuState {
	cartInfo := ParseCartInfo(cart)
	if cartInfo.cgbOnly() {
		fatalErr("CGB-only not supported yet")
	}
	mem := mem{
		cart:    cart,
		cartRAM: make([]byte, cartInfo.GetRAMSize()),
	}
	state := cpuState{pc: entryPoint, mem: mem}
	state.lcd.init()
	state.apu.init()
	return &state
}

func (cs *cpuState) runCycles(ncycles uint) {
	for i := uint(0); i < ncycles; i++ {
		cs.cycles++
		// much TODO
		cs.lcd.runCycle(cs)
	}
}

func (cs *cpuState) setOp8(cycles uint, instLen uint16, reg *uint8, val uint8, flags uint16) {
	cs.runCycles(cycles)
	cs.pc += instLen
	*reg = val
	cs.setFlags(flags)
}

func (cs *cpuState) setOpA(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.a, val, flags)
}
func (cs *cpuState) setOpB(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.b, val, flags)
}
func (cs *cpuState) setOpC(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.c, val, flags)
}
func (cs *cpuState) setOpD(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.d, val, flags)
}
func (cs *cpuState) setOpE(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.e, val, flags)
}
func (cs *cpuState) setOpL(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.l, val, flags)
}
func (cs *cpuState) setOpH(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.h, val, flags)
}

func (cs *cpuState) setOp16(cycles uint, instLen uint16, setFn func(uint16), val uint16, flags uint16) {
	cs.runCycles(cycles)
	cs.pc += instLen
	setFn(val)
	cs.setFlags(flags)
}

func (cs *cpuState) setOpHL(cycles uint, instLen uint16, val uint16, flags uint16) {
	cs.setOp16(cycles, instLen, cs.setHL, val, flags)
}

func (cs *cpuState) setOpBC(cycles uint, instLen uint16, val uint16, flags uint16) {
	cs.setOp16(cycles, instLen, cs.setBC, val, flags)
}

func (cs *cpuState) setOpDE(cycles uint, instLen uint16, val uint16, flags uint16) {
	cs.setOp16(cycles, instLen, cs.setDE, val, flags)
}

func (cs *cpuState) setOpSP(cycles uint, instLen uint16, val uint16, flags uint16) {
	cs.setOp16(cycles, instLen, cs.setSP, val, flags)
}

func (cs *cpuState) setOpPC(cycles uint, instLen uint16, val uint16, flags uint16) {
	cs.setOp16(cycles, instLen, cs.setPC, val, flags)
}

func (cs *cpuState) setOpMem8(cycles uint, instLen uint16, addr uint16, val uint8, flags uint16) {
	cs.runCycles(cycles)
	cs.pc += instLen
	cs.write(addr, val)
	//	fmt.Printf("\twriting 0x%02x to 0x%04x\r\n", val, addr)
	cs.setFlags(flags)
}

func (cs *cpuState) setOpMem16(cycles uint, instLen uint16, addr uint16, val uint16, flags uint16) {
	cs.runCycles(cycles)
	cs.pc += instLen
	cs.write16(addr, val)
	//	fmt.Printf("\twriting 0x%04x to 0x%04x\r\n", val, addr)
	cs.setFlags(flags)
}

func (cs *cpuState) jmpRel8(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, relAddr int8) {
	cs.pc += instLen
	if test {
		cs.runCycles(cyclesTaken)
		cs.pc = uint16(int(cs.pc) + int(relAddr))
		//		fmt.Printf("\tjump succeeded to 0x%04x\r\n", cs.pc)
	} else {
		cs.runCycles(cyclesNotTaken)
		//		fmt.Printf("\tjump test failed\r\n")
	}
}

func (cs *cpuState) jmpAbs16(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, addr uint16) {
	cs.pc += instLen
	if test {
		cs.runCycles(cyclesTaken)
		cs.pc = addr
		//		fmt.Printf("\tjump succeeded to 0x%04x\r\n", cs.pc)
	} else {
		cs.runCycles(cyclesNotTaken)
		//		fmt.Printf("\tjump test failed\r\n")
	}
}

func (cs *cpuState) jmpCall(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, addr uint16) {
	if test {
		cs.pushOp16(cyclesTaken, instLen, cs.pc+instLen)
		cs.pc = addr
	} else {
		cs.setOpFn(cyclesNotTaken, instLen, func() {}, 0x2222)
	}
}

func (cs *cpuState) jmpRet(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool) {
	if test {
		cs.popOp16(cyclesTaken, instLen, cs.setPC)
	} else {
		cs.setOpFn(cyclesNotTaken, instLen, func() {}, 0x2222)
	}
}

// reminder: flags == zero, addsub, halfcarry, carry
// set all: 0x1111
// clear all: 0x0000
// ignore all: 0x2222

func zFlag(val uint8) uint16 {
	if val == 0 {
		return 0x1000
	}
	return 0x0000
}

// half carry
func hFlagAdd(val, addend uint8) uint16 {
	// 4th to 5th bit carry
	if int(val&0x0f)+int(addend&0x0f) >= 0x10 {
		return 0x10
	}
	return 0x00
}

// half carry
func hFlagAdc(val, addend, fReg uint8) uint16 {
	carry := (fReg >> 4) & 0x01
	// 4th to 5th bit carry
	if int(carry)+int(val&0x0f)+int(addend&0x0f) >= 0x10 {
		return 0x10
	}
	return 0x00
}

// half carry 16
func hFlagAdd16(val, addend uint16) uint16 {
	// 12th to 13th bit carry
	if int(val&0x0fff)+int(addend&0x0fff) >= 0x1000 {
		return 0x10
	}
	return 0x00
}

// half carry
func hFlagSub(val, subtrahend uint8) uint16 {
	if int(val&0xf)-int(subtrahend&0xf) < 0 {
		return 0x10
	}
	return 0x00
}

// half carry
func hFlagSbc(val, subtrahend, fReg uint8) uint16 {
	carry := (fReg >> 4) & 0x01
	if int(val&0xf)-int(subtrahend&0xf)-int(carry) < 0 {
		return 0x10
	}
	return 0x00
}

// carry
func cFlagAdd(val, addend uint8) uint16 {
	if int(val)+int(addend) > 0xff {
		return 0x1
	}
	return 0x0
}

// carry
func cFlagAdc(val, addend, fReg uint8) uint16 {
	carry := (fReg >> 4) & 0x01
	if int(carry)+int(val)+int(addend) > 0xff {
		return 0x1
	}
	return 0x0
}

// carry 16
func cFlagAdd16(val, addend uint16) uint16 {
	if int(val)+int(addend) > 0xffff {
		return 0x1
	}
	return 0x0
}

// carry
func cFlagSub(val, subtrahend uint8) uint16 {
	if int(val)-int(subtrahend) < 0 {
		return 0x1
	}
	return 0x0
}

func cFlagSbc(val, subtrahend, fReg uint8) uint16 {
	carry := (fReg >> 4) & 0x01
	if int(val)-int(subtrahend)-int(carry) < 0 {
		return 0x1
	}
	return 0x0
}

func (cs *cpuState) setOpFn(cycles uint, instLen uint16, fn func(), flags uint16) {
	cs.runCycles(cycles)
	cs.pc += instLen
	fn()
	cs.setFlags(flags)
}

// Emulator exposes the publi1 facing fns for an emulation session
type Emulator interface {
	Framebuffer() []byte
	FlipRequested() bool
	Step()
}

// NewEmulator creates an emulation session
func NewEmulator(cart []byte) Emulator {
	return newState(cart)
}

// Framebuffer returns the current state of the lcd screen
func (cs *cpuState) Framebuffer() []byte {
	return cs.lcd.framebuffer
}

// FlipRequested indicates if a draw request is pending
// and clears it before returning
func (cs *cpuState) FlipRequested() bool {
	if cs.lcd.flipRequested {
		cs.lcd.flipRequested = false
		return true
	}
	return false
}

// Step steps the emulator one instruction
func (cs *cpuState) Step() {

	ieAndIfFlagMatch := cs.handleInterrupts()
	if ieAndIfFlagMatch && cs.inHaltMode {
		cs.runCycles(4)
		cs.inHaltMode = false
	}
	if cs.inHaltMode {
		cs.runCycles(4)
		return
	}

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

	opcode := cs.read(cs.pc)
	cs.steps++

	if opcode != 0xcb {
		//fmt.Printf("steps: %08d, opcode:%02x, pc:%04x, sp:%04x, a:%02x, b:%02x, c:%02x, d:%02x, e:%02x, h:%02x, l:%02x\r\n", cs.steps, opcode, cs.pc, cs.sp, cs.a, cs.b, cs.c, cs.d, cs.e, cs.h, cs.l)
	}

	switch opcode {

	case 0x00: // nop
		cs.setOpFn(4, 1, func() {}, 0x2222)
	case 0x01: // ld bc, n16
		cs.setOpBC(12, 3, cs.read16(cs.pc+1), 0x2222)
	case 0x02: // ld (bc), a
		cs.setOpMem8(8, 1, cs.getBC(), cs.a, 0x2222)
	case 0x03: // inc bc
		cs.setOpBC(8, 1, cs.getBC()+1, 0x2222)
	case 0x04: // inc b
		val := cs.b
		cs.setOpB(4, 1, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x05: // dec b
		val := cs.b
		cs.setOpB(4, 1, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x06: // ld b, n8
		cs.setOpB(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x07: // rlca
		val := (cs.a << 1) | (cs.a >> 7)
		cs.setOpA(4, 1, val, uint16(cs.a>>7))

	case 0x08: // ld (a16), sp
		cs.setOpMem16(20, 3, cs.read16(cs.pc+1), cs.sp, 0x2222)
	case 0x09: // add hl, bc
		v1, v2 := cs.getHL(), cs.getBC()
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x0a: // ld a, (bc)
		cs.setOpA(8, 1, cs.followBC(), 0x2222)
	case 0x0b: // dec bc
		cs.setOpBC(8, 1, cs.getBC()-1, 0x2222)
	case 0x0c: // inc c
		val := cs.c
		cs.setOpC(4, 1, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x0d: // dec c
		val := cs.c
		cs.setOpC(4, 1, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x0e: // ld c, n8
		cs.setOpC(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x0f: // rrca
		val := (cs.a >> 1) | (cs.a << 7)
		cs.setOpA(4, 1, val, uint16(cs.a&0x01))

	case 0x10: // stop
		cs.setOpFn(4, 2, func() { cs.inStopMode = true }, 0x2222)
	case 0x11: // ld de, n16
		cs.setOpDE(12, 3, cs.read16(cs.pc+1), 0x2222)
	case 0x12: // ld (de), a
		cs.setOpMem8(8, 1, cs.getDE(), cs.a, 0x2222)
	case 0x13: // inc de
		cs.setOpDE(8, 1, cs.getDE()+1, 0x2222)
	case 0x14: // inc d
		val := cs.d
		cs.setOpD(4, 1, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x15: // dec d
		val := cs.d
		cs.setOpD(4, 1, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x16: // ld d, n8
		cs.setOpD(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x17: // rla
		val := (cs.a << 1) | ((cs.f >> 4) & 0x01)
		cs.setOpA(4, 1, val, uint16(cs.a>>7))

	case 0x18: // jr r8
		cs.jmpRel8(12, 12, 2, true, int8(cs.read(cs.pc+1)))
	case 0x19: // add hl, de
		v1, v2 := cs.getHL(), cs.getDE()
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x1a: // ld a, (de)
		cs.setOpA(8, 1, cs.followDE(), 0x2222)
	case 0x1b: // dec de
		cs.setOpDE(8, 1, cs.getDE()-1, 0x2222)
	case 0x1c: // inc e
		val := cs.e
		cs.setOpE(4, 1, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x1d: // dec e
		val := cs.e
		cs.setOpE(4, 1, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x1e: // ld e, n8
		cs.setOpE(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x1f: // rra
		val := (cs.a >> 1) | (cs.f << 3)
		cs.setOpA(4, 1, val, uint16(cs.a&0x01))

	case 0x20: // jr nz, r8
		cs.jmpRel8(12, 8, 2, !cs.getZeroFlag(), int8(cs.read(cs.pc+1)))
	case 0x21: // ld hl, n16
		cs.setOpHL(12, 3, cs.read16(cs.pc+1), 0x2222)
	case 0x22: // ld (hl++), a
		cs.setOpMem8(8, 1, cs.getHL(), cs.a, 0x2222)
		cs.setHL(cs.getHL() + 1)
	case 0x23: // inc hl
		cs.setOpHL(8, 1, cs.getHL()+1, 0x2222)
	case 0x24: // inc h
		cs.setOpH(4, 1, cs.h+1, (zFlag(cs.h+1) | hFlagAdd(cs.h, 1) | 0x0002))
	case 0x26: // ld h, d8
		cs.setOpH(8, 2, cs.read(cs.pc+1), 0x2222)

	case 0x28: // jr z, r8
		cs.jmpRel8(12, 8, 2, cs.getZeroFlag(), int8(cs.read(cs.pc+1)))
	case 0x29: // add hl, hl
		v1, v2 := cs.getHL(), cs.getHL()
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x2a: // ld a, (hl++)
		cs.setOpA(8, 1, cs.followHL(), 0x2222)
		cs.setHL(cs.getHL() + 1)
	case 0x2b: // dec hl
		cs.setOpHL(8, 1, cs.getHL()-1, 0x2222)
	case 0x2c: // inc l
		val := cs.l
		cs.setOpL(4, 1, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x2d: // dec l
		val := cs.l
		cs.setOpL(4, 1, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x2e: // ld l, d8
		cs.setOpL(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x2f: // cpl
		cs.setOpA(4, 1, ^cs.a, 0x2222)

	case 0x30: // jr z, r8
		cs.jmpRel8(12, 8, 2, !cs.getCarryFlag(), int8(cs.read(cs.pc+1)))
	case 0x31: // ld sp, n16
		cs.setOpSP(12, 3, cs.read16(cs.pc+1), 0x2222)
	case 0x32: // ld (hl--) a
		cs.setOpMem8(8, 1, cs.getHL(), cs.a, 0x2222)
		cs.setHL(cs.getHL() - 1)
	case 0x33: // inc sp
		cs.setOpSP(8, 1, cs.sp+1, 0x2222)
	case 0x34: // inc (hl)
		val := cs.followHL()
		cs.setOpMem8(12, 1, cs.getHL(), val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x35: // dec (hl)
		val := cs.followHL()
		cs.setOpMem8(12, 1, cs.getHL(), val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x36: // ld (hl) n8
		cs.setOpMem8(12, 2, cs.getHL(), cs.read(cs.pc+1), 0x2222)
	case 0x37: // scf
		cs.setOpFn(4, 1, func() {}, 0x2001)

	case 0x38: // jr c, r8
		cs.jmpRel8(12, 8, 2, cs.getCarryFlag(), int8(cs.read(cs.pc+1)))
	case 0x39: // add hl, sp
		v1, v2 := cs.getHL(), cs.sp
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x3a: // ld a, (hl--)
		cs.setOpA(8, 1, cs.followHL(), 0x2222)
		cs.setHL(cs.getHL() - 1)
	case 0x3b: // dec sp
		cs.setOpSP(8, 1, cs.sp-1, 0x2222)
	case 0x3c: // inc a
		val := cs.a
		cs.setOpA(4, 1, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
	case 0x3d: // dec a
		val := cs.a
		cs.setOpA(4, 1, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
	case 0x3e: // ld a, n8
		cs.setOpA(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x3f: // ccf
		carry := uint16((cs.f>>4)&0x01) ^ 0x01
		cs.setOpFn(4, 1, func() {}, 0x2000|carry)

	case 0x40: // ld b, b
		cs.setOpB(4, 1, cs.b, 0x2222)
	case 0x41: // ld b, c
		cs.setOpB(4, 1, cs.c, 0x2222)
	case 0x42: // ld b, d
		cs.setOpB(4, 1, cs.d, 0x2222)
	case 0x43: // ld b, e
		cs.setOpB(4, 1, cs.e, 0x2222)
	case 0x44: // ld b, h
		cs.setOpB(4, 1, cs.h, 0x2222)
	case 0x45: // ld b, l
		cs.setOpB(4, 1, cs.l, 0x2222)
	case 0x46: // ld b, (hl)
		cs.setOpB(8, 1, cs.followHL(), 0x2222)
	case 0x47: // ld b, a
		cs.setOpB(4, 1, cs.a, 0x2222)

	case 0x48: // ld c, b
		cs.setOpC(4, 1, cs.b, 0x2222)
	case 0x49: // ld c, c
		cs.setOpC(4, 1, cs.c, 0x2222)
	case 0x4a: // ld c, d
		cs.setOpC(4, 1, cs.d, 0x2222)
	case 0x4b: // ld c, e
		cs.setOpC(4, 1, cs.e, 0x2222)
	case 0x4c: // ld c, h
		cs.setOpC(4, 1, cs.h, 0x2222)
	case 0x4d: // ld c, l
		cs.setOpC(4, 1, cs.l, 0x2222)
	case 0x4e: // ld c, (hl)
		cs.setOpC(8, 1, cs.followHL(), 0x2222)
	case 0x4f: // ld c, a
		cs.setOpC(4, 1, cs.a, 0x2222)

	case 0x50: // ld d, b
		cs.setOpD(4, 1, cs.b, 0x2222)
	case 0x51: // ld d, c
		cs.setOpD(4, 1, cs.c, 0x2222)
	case 0x52: // ld d, d
		cs.setOpD(4, 1, cs.d, 0x2222)
	case 0x53: // ld d, e
		cs.setOpD(4, 1, cs.e, 0x2222)
	case 0x54: // ld d, h
		cs.setOpD(4, 1, cs.h, 0x2222)
	case 0x55: // ld d, l
		cs.setOpD(4, 1, cs.l, 0x2222)
	case 0x56: // ld d, (hl)
		cs.setOpD(8, 1, cs.followHL(), 0x2222)
	case 0x57: // ld d, a
		cs.setOpD(4, 1, cs.a, 0x2222)

	case 0x58: // ld e, b
		cs.setOpE(4, 1, cs.b, 0x2222)
	case 0x59: // ld e, c
		cs.setOpE(4, 1, cs.c, 0x2222)
	case 0x5a: // ld e, d
		cs.setOpE(4, 1, cs.d, 0x2222)
	case 0x5b: // ld e, e
		cs.setOpE(4, 1, cs.e, 0x2222)
	case 0x5c: // ld e, h
		cs.setOpE(4, 1, cs.h, 0x2222)
	case 0x5d: // ld e, l
		cs.setOpE(4, 1, cs.l, 0x2222)
	case 0x5e: // ld e, (hl)
		cs.setOpE(8, 1, cs.followHL(), 0x2222)
	case 0x5f: // ld e, a
		cs.setOpE(4, 1, cs.a, 0x2222)

	case 0x60: // ld h, b
		cs.setOpH(4, 1, cs.b, 0x2222)
	case 0x61: // ld h, c
		cs.setOpH(4, 1, cs.c, 0x2222)
	case 0x62: // ld h, d
		cs.setOpH(4, 1, cs.d, 0x2222)
	case 0x63: // ld h, e
		cs.setOpH(4, 1, cs.e, 0x2222)
	case 0x64: // ld h, h
		cs.setOpH(4, 1, cs.h, 0x2222)
	case 0x65: // ld h, l
		cs.setOpH(4, 1, cs.l, 0x2222)
	case 0x66: // ld h, (hl)
		cs.setOpH(8, 1, cs.followHL(), 0x2222)
	case 0x67: // ld h, a
		cs.setOpH(4, 1, cs.a, 0x2222)

	case 0x68: // ld l, b
		cs.setOpL(4, 1, cs.b, 0x2222)
	case 0x69: // ld l, c
		cs.setOpL(4, 1, cs.c, 0x2222)
	case 0x6a: // ld l, d
		cs.setOpL(4, 1, cs.d, 0x2222)
	case 0x6b: // ld l, e
		cs.setOpL(4, 1, cs.e, 0x2222)
	case 0x6c: // ld l, h
		cs.setOpL(4, 1, cs.h, 0x2222)
	case 0x6d: // ld l, l
		cs.setOpL(4, 1, cs.l, 0x2222)
	case 0x6e: // ld l, (hl)
		cs.setOpL(8, 1, cs.followHL(), 0x2222)
	case 0x6f: // ld l, a
		cs.setOpL(4, 1, cs.a, 0x2222)

	case 0x70: // ld (hl), b
		cs.setOpMem8(8, 1, cs.getHL(), cs.b, 0x2222)
	case 0x71: // ld (hl), c
		cs.setOpMem8(8, 1, cs.getHL(), cs.c, 0x2222)
	case 0x72: // ld (hl), d
		cs.setOpMem8(8, 1, cs.getHL(), cs.d, 0x2222)
	case 0x73: // ld (hl), e
		cs.setOpMem8(8, 1, cs.getHL(), cs.e, 0x2222)
	case 0x74: // ld (hl), h
		cs.setOpMem8(8, 1, cs.getHL(), cs.h, 0x2222)
	case 0x75: // ld (hl), l
		cs.setOpMem8(8, 1, cs.getHL(), cs.l, 0x2222)
	case 0x76: // halt
		cs.inHaltMode = true
	case 0x77: // ld (hl), a
		cs.setOpMem8(8, 1, cs.getHL(), cs.a, 0x2222)

	case 0x78: // ld a, b
		cs.setOpA(4, 1, cs.b, 0x2222)
	case 0x79: // ld a, c
		cs.setOpA(4, 1, cs.c, 0x2222)
	case 0x7a: // ld a, d
		cs.setOpA(4, 1, cs.d, 0x2222)
	case 0x7b: // ld a, e
		cs.setOpA(4, 1, cs.e, 0x2222)
	case 0x7c: // ld a, h
		cs.setOpA(4, 1, cs.h, 0x2222)
	case 0x7d: // ld a, l
		cs.setOpA(4, 1, cs.l, 0x2222)
	case 0x7e: // ld a, (hl)
		cs.setOpA(8, 1, cs.followHL(), 0x2222)
	case 0x7f: // ld a, a
		cs.setOpA(4, 1, cs.a, 0x2222)

	case 0x80: // add a, b
		val := cs.b
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x81: // add a, c
		val := cs.c
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x82: // add a, d
		val := cs.d
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x83: // add a, e
		val := cs.e
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x84: // add a, h
		val := cs.h
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x85: // add a, l
		val := cs.l
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x86: // add a, (hl)
		val := cs.read(cs.getHL())
		cs.setOpA(8, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0x87: // add a, a
		val := cs.a
		cs.setOpA(4, 1, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))

	case 0x88: // adc a, b
		val := cs.b
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x89: // adc a, c
		val := cs.c
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x8a: // adc a, d
		val := cs.d
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x8b: // adc a, e
		val := cs.e
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x8c: // adc a, h
		val := cs.h
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x8d: // adc a, l
		val := cs.l
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x8e: // adc a, (hl)
		val := cs.followHL()
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(8, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0x8f: // adc a, a
		val := cs.a
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))

	case 0x90: // sub b
		val := cs.b
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x91: // sub c
		val := cs.c
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x92: // sub d
		val := cs.d
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x93: // sub e
		val := cs.e
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x94: // sub h
		val := cs.h
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x95: // sub l
		val := cs.l
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x96: // sub (hl)
		val := cs.followHL()
		cs.setOpA(8, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0x97: // sub a
		val := cs.a
		cs.setOpA(4, 1, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))

	case 0x98: // sbc b
		val := cs.b
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x99: // sbc c
		val := cs.c
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x9a: // sbc d
		val := cs.d
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x9b: // sbc e
		val := cs.e
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x9c: // sbc h
		val := cs.h
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x9d: // sbc l
		val := cs.l
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x9e: // sbc (hl)
		val := cs.followHL()
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(8, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0x9f: // sbc a
		val := cs.a
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(4, 1, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))

	case 0xa0: // and b
		val := cs.b
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa1: // and c
		val := cs.c
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa2: // and d
		val := cs.d
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa3: // and e
		val := cs.e
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa4: // and h
		val := cs.h
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa5: // and l
		val := cs.l
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa6: // and (hl)
		val := cs.followHL()
		cs.setOpA(8, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xa7: // and a
		val := cs.a
		cs.setOpA(4, 1, cs.a&val, (zFlag(cs.a&val) | 0x010))

	case 0xa8: // xor b
		val := cs.b
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))
	case 0xa9: // xor c
		val := cs.c
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))
	case 0xaa: // xor d
		val := cs.d
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))
	case 0xab: // xor e
		val := cs.e
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))
	case 0xac: // xor h
		val := cs.h
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))
	case 0xad: // xor l
		val := cs.l
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))
	case 0xae: // xor (hl)
		val := cs.followHL()
		cs.setOpA(8, 1, cs.a^val, zFlag(cs.a^val))
	case 0xaf: // xor a
		val := cs.a
		cs.setOpA(4, 1, cs.a^val, zFlag(cs.a^val))

	case 0xb0: // or b
		val := cs.b
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb1: // or c
		val := cs.c
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb2: // or d
		val := cs.d
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb3: // or e
		val := cs.e
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb4: // or h
		val := cs.h
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb5: // or l
		val := cs.l
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb6: // or (hl)
		val := cs.followHL()
		cs.setOpA(8, 1, cs.a|val, zFlag(cs.a|val))
	case 0xb7: // or a
		val := cs.a
		cs.setOpA(4, 1, cs.a|val, zFlag(cs.a|val))

	case 0xb8: // cp b
		val := cs.b
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xb9: // cp c
		val := cs.c
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xba: // cp d
		val := cs.d
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xbb: // cp e
		val := cs.e
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xbc: // cp h
		val := cs.h
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xbd: // cp l
		val := cs.l
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xbe: // cp (hl)
		val := cs.followHL()
		cs.setOpFn(8, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xbf: // cp a
		val := cs.a
		cs.setOpFn(4, 1, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))

	case 0xc0: // ret nz
		cs.jmpRet(20, 8, 1, !cs.getZeroFlag())
	case 0xc1: // pop bc
		cs.popOp16(12, 1, cs.setBC)
	case 0xc2: // jp nz, a16
		cs.jmpAbs16(16, 12, 3, !cs.getZeroFlag(), cs.read16(cs.pc+1))
	case 0xc3: // jp a16
		cs.setOpPC(16, 3, cs.read16(cs.pc+1), 0x2222)
	case 0xc4: // call nz, a16
		cs.jmpCall(24, 12, 3, !cs.getZeroFlag(), cs.read16(cs.pc+1))
	case 0xc5: // push bc
		cs.pushOp16(16, 1, cs.getBC())
	case 0xc6: // add a, n8
		val := cs.read(cs.pc + 1)
		cs.setOpA(8, 2, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
	case 0xc7: // rst 00h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0000

	case 0xc8: // ret z
		cs.jmpRet(20, 8, 1, cs.getZeroFlag())
	case 0xc9: // ret
		cs.popOp16(16, 1, cs.setPC)
	case 0xca: // jp z, a16
		cs.jmpAbs16(16, 12, 3, cs.getZeroFlag(), cs.read16(cs.pc+1))
	case 0xcb: // extended opcode prefix
		cs.stepExtendedOpcode()
	case 0xcc: // call z, a16
		cs.jmpCall(24, 12, 3, cs.getZeroFlag(), cs.read16(cs.pc+1))
	case 0xcd: // call a16
		cs.pushOp16(24, 3, cs.pc+3)
		cs.pc = cs.read16(cs.pc - 2) // pc is sub'd to undo the move past inst
	case 0xce: // adc a, n8
		val := cs.read(cs.pc + 1)
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(8, 2, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
	case 0xcf: // rst 08h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0008

	case 0xd0: // ret nc
		cs.jmpRet(20, 8, 1, !cs.getCarryFlag())
	case 0xd1: // pop de
		cs.popOp16(12, 1, cs.setDE)
	case 0xd2: // jp nc, a16
		cs.jmpAbs16(16, 12, 3, !cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xd3: // illegal
		panic("illegal opcode")
	case 0xd4: // call nc, a16
		cs.jmpCall(24, 12, 3, !cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xd5: // push de
		cs.pushOp16(16, 1, cs.getDE())
	case 0xd6: // sub n8
		val := cs.read(cs.pc + 1)
		cs.setOpA(8, 2, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
	case 0xd7: // rst 10h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0010

	case 0xd8: // ret c
		cs.jmpRet(20, 8, 1, cs.getCarryFlag())
	case 0xd9: // reti
		cs.popOp16(16, 1, cs.setPC)
		cs.interruptMasterEnable = true
	case 0xda: // jp c, a16
		cs.jmpAbs16(16, 12, 3, cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xdb: // illegal
		panic("illegal opcode")
	case 0xdc: // call c, a16
		cs.jmpCall(24, 12, 3, cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xdd: // illegal
		panic("illegal opcode")
	case 0xde: // sbc n8
		val := cs.read(cs.pc + 1)
		carry := (cs.f >> 4) & 0x01
		cs.setOpA(8, 2, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
	case 0xdf: // rst 18h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0018

	case 0xe0: // ld (0xFF00 + n8), a
		val := cs.read(cs.pc + 1)
		cs.setOpMem8(12, 2, 0xff00+uint16(val), cs.a, 0x2222)
	case 0xe1: // pop hl
		cs.popOp16(12, 1, cs.setHL)
	case 0xe2: // ld (0xFF00 + c), a
		val := cs.c
		cs.setOpMem8(8, 1, 0xff00+uint16(val), cs.a, 0x2222)
	case 0xe3: // illegal
		panic("illegal opcode")
	case 0xe4: // illegal
		panic("illegal opcode")
	case 0xe5: // push hl
		cs.pushOp16(16, 1, cs.getHL())
	case 0xe6: // and n8
		val := cs.read(cs.pc + 1)
		cs.setOpA(8, 2, cs.a&val, (zFlag(cs.a&val) | 0x010))
	case 0xe7: // rst 20h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0020

	case 0xe9: // jp hl (also written jp (hl))
		cs.setOpPC(4, 1, cs.getHL(), 0x2222)
	case 0xea: // ld (a16), a
		cs.setOpMem8(16, 3, cs.read16(cs.pc+1), cs.a, 0x2222)
	case 0xeb: // illegal
		panic("illegal opcode")
	case 0xec: // illegal
		panic("illegal opcode")
	case 0xed: // illegal
		panic("illegal opcode")
	case 0xee: // xor n8
		val := cs.read(cs.pc + 1)
		cs.setOpA(8, 2, cs.a^val, zFlag(cs.a^val))
	case 0xef: // rst 28h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0028

	case 0xf0: // ld a, (0xFF00 + n8)
		cs.setOpA(12, 2, cs.read(0xff00+uint16(cs.read(cs.pc+1))), 0x2222)
	case 0xf1: // pop af
		cs.popOp16(12, 1, cs.setAF)
	case 0xf2: // ld a, (0xFF00 + c)
		cs.setOpA(8, 1, cs.read(0xff00+uint16(cs.c)), 0x2222)
	case 0xf3: // di
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnable = false }, 0x2222)
	case 0xf4: // illegal
		panic("illegal opcode")
	case 0xf5: // push af
		cs.pushOp16(16, 1, cs.getAF())
	case 0xf6: // or n8
		val := cs.read(cs.pc + 1)
		cs.setOpA(8, 2, cs.a|val, zFlag(cs.a|val))
	case 0xf7: // rst 30h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0030

	case 0xf9: // ld sp, hl
		cs.setOpSP(8, 1, cs.getHL(), 0x2222)
	case 0xfa: // ld a, (a16)
		cs.setOpA(16, 3, cs.read(cs.read16(cs.pc+1)), 0x2222)
	case 0xfb: // ei
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnable = true }, 0x2222)
	case 0xfc: // illegal
		panic("illegal opcode")
	case 0xfd: // illegal
		panic("illegal opcode")
	case 0xfe: // cp a, n8
		val := cs.read(cs.pc + 1)
		cs.setOpFn(8, 2, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
	case 0xff: // rst 38h
		cs.pushOp16(16, 1, cs.pc+1)
		cs.pc = 0x0038

	default:
		fatalErr(fmt.Sprintf("Unknown Opcode: 0x%02x\r\n", opcode))
	}
}

func (cs *cpuState) popOp16(cycles uint, instLen uint16, setFn func(val uint16)) {
	cs.setOpFn(cycles, instLen, func() { setFn(cs.read16(cs.sp)) }, 0x2222)
	cs.sp += 2
}

func (cs *cpuState) pushOp16(cycles uint, instLen uint16, val uint16) {
	cs.setOpMem16(cycles, instLen, cs.sp-2, val, 0x2222)
	cs.sp -= 2
}

func (cs *cpuState) followBC() byte { return cs.read(cs.getBC()) }
func (cs *cpuState) followDE() byte { return cs.read(cs.getDE()) }
func (cs *cpuState) followHL() byte { return cs.read(cs.getHL()) }
func (cs *cpuState) followSP() byte { return cs.read(cs.sp) }
func (cs *cpuState) followPC() byte { return cs.read(cs.pc) }

func (cs *cpuState) stepExtendedOpcode() {

	extOpcode := cs.read(cs.pc + 1)

	//fmt.Printf("steps: %08d, ext.op:%02x, pc:%04x, sp:%04x, a:%02x, b:%02x, c:%02x, d:%02x, e:%02x, h:%02x, l:%02x\r\n", cs.steps, extOpcode, cs.pc, cs.sp, cs.a, cs.b, cs.c, cs.d, cs.e, cs.h, cs.l)

	switch extOpcode {
	case 0x37: // swap a
		result := (cs.a >> 4) | ((cs.a & 0x0f) << 4)
		cs.setOpA(8, 2, result, zFlag(result))
	case 0x3f: // srl a
		result := cs.a >> 1
		cs.setOpA(8, 2, result, zFlag(result)|uint16(cs.a&1))

	case 0x50: // bit 2, b
		cs.bitOp(8, 2, 2, cs.b)
	case 0x57: // bit 2, a
		cs.bitOp(8, 2, 2, cs.a)

	case 0x5f: // bit 3, a
		cs.bitOp(8, 2, 3, cs.a)

	case 0x70: // bit 6, b
		cs.bitOp(8, 2, 6, cs.b)

	case 0x78: // bit 7, b
		cs.bitOp(8, 2, 7, cs.b)
	case 0x7f: // bit 7, a
		cs.bitOp(8, 2, 7, cs.a)

	case 0x87: // res 0, a
		val := cs.a
		cs.setOpA(8, 2, val&^0x01, 0x2222)

	case 0xbe: // res 7, (hl)
		val := cs.followHL()
		cs.setOpMem8(16, 2, cs.getHL(), val&^0x80, 0x2222)
	case 0xbf: // res 7, a
		cs.setOpA(8, 2, (cs.a &^ 0x80), 0x2222)

	case 0xc7: // set 0, a
		cs.setOpA(8, 2, (cs.a | 0x01), 0x2222)

	case 0xff: // set 7, a
		cs.setOpA(8, 2, (cs.a | 0x80), 0x2222)

	default:
		fatalErr(fmt.Sprintf("Unknown Extended Opcode: 0x%02x\r\n", extOpcode))
	}
}

func (cs *cpuState) bitOp(cycles uint, instLen uint16, bitNum uint8, val uint8) {
	cs.setOpFn(cycles, instLen, func() {}, zFlag(val&(1<<bitNum))|0x012)
}

func fatalErr(v ...interface{}) {
	fmt.Println(v...)
	os.Exit(1)
}
