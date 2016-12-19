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

	interruptMasterEnableFlag bool
	interruptsEnableRegister  byte
	interruptsFlagRegister    byte

	serialTransferData            byte
	serialTransferStartFlag       bool
	serialTransferClockIsInternal bool

	steps uint
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
	return &cpuState{pc: entryPoint, mem: mem}
}

func (cs *cpuState) runCycle() { /* TODO */ }

func (cs *cpuState) cycles(ncycles uint) {
	for i := uint(0); i < ncycles; i++ {
		cs.runCycle()
	}
}

func (cs *cpuState) setOp8(cycles uint, instLen uint16, reg *uint8, val uint8, flags uint16) {
	cs.cycles(cycles)
	*reg = val
	cs.setFlags(flags)
	cs.pc += instLen
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

func (cs *cpuState) setOp16(cycles uint, instLen uint16, setFn func(uint16), val uint16, flags uint16) {
	cs.cycles(cycles)
	setFn(val)
	cs.setFlags(flags)
	cs.pc += instLen
}

func (cs *cpuState) setOpHL(cycles uint, instLen uint16, val uint16, flags uint16) {
	cs.setOp16(cycles, instLen, cs.setHL, val, flags)
}

func (cs *cpuState) setOpMem8(cycles uint, instLen uint16, addr uint16, val uint8, flags uint16) {
	cs.cycles(cycles)
	cs.write(addr, val)
	cs.setFlags(flags)
	cs.pc += instLen
}

func (cs *cpuState) jmpRel8(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, relAddr int8) {
	cs.pc += instLen
	if test {
		cs.cycles(cyclesTaken)
		cs.pc = uint16(int(cs.pc) + int(relAddr))
	} else {
		cs.cycles(cyclesNotTaken)
	}
}

// reminder: flags == zero, addsub, halfcarry, carry
// set all: 0x1111
// clear all: 0x0000
// ignore all: 0x2222

func zeroFlag(val uint8) uint16 {
	if val == 0 {
		return 0x1000
	}
	return 0x0000
}
func halfCarryAdd(val, addend uint8) uint16 {
	if int(val&0xf)+int(addend&0xf) > 0x10 {
		return 0x10
	}
	return 0x00
}
func halfCarrySub(val, subtrahend uint8) uint16 {
	if int(val&0xf)-int(subtrahend&0xf) < 0 {
		return 0x10
	}
	return 0x00
}
func carryAdd(val, addend uint8) uint16 {
	if int(val)+int(addend) > 0xff {
		return 0x1
	}
	return 0x0
}
func carrySub(val, subtrahend uint8) uint16 {
	if int(val)-int(subtrahend) < 0 {
		return 0x1
	}
	return 0x0
}

func (cs *cpuState) setOpFn(cycles uint, instLen uint16, fn func(), flags uint16) {
	cs.cycles(cycles)
	cs.pc += instLen
	fn()
	cs.setFlags(flags)
}

// Emulator exposes the public facing fns for an emulation session
type Emulator interface {
	Framebuffer() []byte
	Step()
}

// NewEmulator creates an emulation session
func NewEmulator(cart []byte) Emulator {
	return newState(cart)
}

// Framebuffer returns the current state of the lcd screen
func (cs *cpuState) Framebuffer() []byte {
	return cs.lcd.framebuffer[:]
}

// Step steps the emulator one instruction
func (cs *cpuState) Step() {
	opcode := cs.read(cs.pc)
	cs.steps++

	fmt.Printf("steps: %08d, opcode: 0x%02x\r\n", cs.steps, opcode)

	switch opcode {
	case 0x00: // nop
		cs.setOpFn(4, 1, func() {}, 0x2222)
	case 0x05: // dec b
		cs.setOpB(4, 1, cs.b-1, (zeroFlag(cs.b-1) | halfCarrySub(cs.b, 1) | 0x0102))
	case 0x06: // ld b, n8
		cs.setOpB(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x0d: // dec c
		cs.setOpC(4, 1, cs.c-1, (zeroFlag(cs.c-1) | halfCarrySub(cs.c, 1) | 0x0102))
	case 0x0e: // ld c, n8
		cs.setOpC(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x20: // jrnz r8
		cs.jmpRel8(12, 8, 2, !cs.getZeroFlag(), int8(cs.read(cs.pc+1)))
	case 0x21: // ld hl, n16
		cs.setOpHL(12, 3, cs.read16(cs.pc+1), 0x2222)
	case 0x32: // ld (hl--) a
		cs.setOpMem8(8, 1, cs.getHL(), cs.a, 0x2222)
		cs.setHL(cs.getHL() - 1)
	case 0x3e: // ld a, n8
		cs.setOpA(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0xc3: // jp a16
		cs.cycles(16)
		cs.pc = cs.read16(cs.pc + 1)
	case 0xaf:
		cs.setOpA(4, 1, cs.a^cs.a, 0x1000)
	case 0xe0: // ld (0xFF00 + d8), a
		cs.setOpMem8(12, 2, 0xff00+uint16(cs.read(cs.pc+1)), cs.a, 0x2222)
	case 0xf0: // ld a, (0xFF00 + d8)
		cs.setOpA(12, 2, cs.read(0xff00+uint16(cs.read(cs.pc+1))), 0x2222)
	case 0xf3: // di
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnableFlag = false }, 0x2222)
	case 0xfe: // cp a, d8
		val, result := cs.read(cs.pc+1), cs.a-cs.read(cs.pc+1)
		cs.setOpFn(8, 2, func() {}, (zeroFlag(result) | halfCarrySub(cs.a, val) | carrySub(cs.a, val) | 0x0100))
	default:
		fatalErr(fmt.Sprintf("Unknown Opcode: 0x%02x\n", opcode))
	}
}

func fatalErr(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
