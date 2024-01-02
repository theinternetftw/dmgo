package dmgo

import "fmt"

func (cs *cpuState) setOp8(cycles uint, reg *uint8, val uint8, flags uint16) {
	cs.runCycles(cycles)
	*reg = val
	cs.setFlags(flags)
}

func setOpA(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.A, val, 0x2222)
}
func setOpB(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.B, val, 0x2222)
}
func setOpC(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.C, val, 0x2222)
}
func setOpD(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.D, val, 0x2222)
}
func setOpE(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.E, val, 0x2222)
}
func setOpL(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.L, val, 0x2222)
}
func setOpH(cs *cpuState, cycles uint, val uint8) {
	cs.setOp8(cycles, &cs.H, val, 0x2222)
}

func (cs *cpuState) setOp16(cycles uint, setFn func(uint16), val uint16, flags uint16) {
	cs.runCycles(cycles)
	setFn(val)
	cs.setFlags(flags)
}

func (cs *cpuState) setOpHL(cycles uint, val uint16) {
	cs.setOp16(cycles, cs.setHL, val, 0x2222)
}
func (cs *cpuState) setOpBC(cycles uint, val uint16) {
	cs.setOp16(cycles, cs.setBC, val, 0x2222)
}
func (cs *cpuState) setOpDE(cycles uint, val uint16) {
	cs.setOp16(cycles, cs.setDE, val, 0x2222)
}
func (cs *cpuState) setOpSP(cycles uint, val uint16) {
	cs.setOp16(cycles, cs.setSP, val, 0x2222)
}
func (cs *cpuState) setOpPC(cycles uint, val uint16) {
	cs.setOp16(cycles, cs.setPC, val, 0x2222)
}

func (cs *cpuState) setOpMem8(cycles uint, addr uint16, val uint8, flags uint16) {
	cs.runCycles(2)
	cs.write(addr, val)
	cs.runCycles(cycles - 2)
	cs.setFlags(flags)
}
func (cs *cpuState) setOpMem16(cycles uint, addr uint16, val uint16, flags uint16) {
	cs.runCycles(2)
	cs.write16(addr, val)
	cs.runCycles(cycles - 2)
	cs.setFlags(flags)
}

func (cs *cpuState) jmpRel8(cyclesTaken uint, cyclesNotTaken uint, test bool, relAddr int8) {
	if test {
		cs.runCycles(cyclesTaken)
		cs.PC = uint16(int(cs.PC) + int(relAddr))
	} else {
		cs.runCycles(cyclesNotTaken)
	}
}
func (cs *cpuState) jmpAbs16(cyclesTaken uint, cyclesNotTaken uint, test bool, addr uint16) {
	if test {
		cs.runCycles(cyclesTaken)
		cs.PC = addr
	} else {
		cs.runCycles(cyclesNotTaken)
	}
}

func (cs *cpuState) jmpCall(cyclesTaken uint, cyclesNotTaken uint, test bool, addr uint16) {
	if test {
		cs.pushOp16(cyclesTaken, cs.PC)
		cs.PC = addr
	} else {
		cs.setOpFn(cyclesNotTaken, func() {}, 0x2222)
	}
}
func (cs *cpuState) jmpRet(cyclesTaken uint, cyclesNotTaken uint, test bool) {
	if test {
		cs.popOp16(cyclesTaken, cs.setPC)
	} else {
		cs.setOpFn(cyclesNotTaken, func() {}, 0x2222)
	}
}

func (cs *cpuState) followBC() byte { return cs.read(cs.getBC()) }
func (cs *cpuState) followDE() byte { return cs.read(cs.getDE()) }
func (cs *cpuState) followHL() byte { return cs.read(cs.getHL()) }
func (cs *cpuState) followSP() byte { return cs.read(cs.SP) }
func (cs *cpuState) followPC() byte { return cs.read(cs.PC) }

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

func (cs *cpuState) setOpFn(cycles uint, fn func(), flags uint16) {
	cs.runCycles(cycles)
	fn()
	cs.setFlags(flags)
}

func (cs *cpuState) pushOp16(cycles uint, val uint16) {
	cs.setOpMem16(cycles, cs.SP-2, val, 0x2222)
	cs.SP -= 2
}
func (cs *cpuState) popOp16(cycles uint, setFn func(val uint16)) {
	cs.setOpFn(cycles, func() { setFn(cs.read16(cs.SP)) }, 0x2222)
	cs.SP += 2
}

func (cs *cpuState) incOpReg(reg *byte) {
	val := *reg
	cs.setOp8(4, reg, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
}
func (cs *cpuState) incOpHL() {
	val := cs.followHL()
	cs.setOpMem8(12, cs.getHL(), val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
}

func (cs *cpuState) decOpReg(reg *byte) {
	val := *reg
	cs.setOp8(4, reg, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
}
func (cs *cpuState) decOpHL() {
	val := cs.followHL()
	cs.setOpMem8(12, cs.getHL(), val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
}

func (cs *cpuState) daaOp() {

	newCarryFlag := uint16(0)
	if cs.getSubFlag() {
		diff := byte(0)
		if cs.getHalfCarryFlag() {
			diff += 0x06
		}
		if cs.getCarryFlag() {
			newCarryFlag = 0x0001
			diff += 0x60
		}
		cs.A -= diff
	} else {
		diff := byte(0)
		if cs.A&0x0f > 0x09 || cs.getHalfCarryFlag() {
			diff += 0x06
		}
		if cs.A > 0x99 || cs.getCarryFlag() {
			newCarryFlag = 0x0001
			diff += 0x60
		}
		cs.A += diff
	}

	cs.setFlags(zFlag(cs.A) | 0x0200 | newCarryFlag)
	cs.runCycles(4)
}

func (cs *cpuState) ifToString() string {
	out := []byte("     ")
	if cs.VBlankIRQ {
		out[0] = 'V'
	}
	if cs.LCDStatIRQ {
		out[1] = 'L'
	}
	if cs.TimerIRQ {
		out[2] = 'T'
	}
	if cs.SerialIRQ {
		out[3] = 'S'
	}
	if cs.JoypadIRQ {
		out[4] = 'J'
	}
	return string(out)
}
func (cs *cpuState) ieToString() string {
	out := []byte("     ")
	if cs.VBlankInterruptEnabled {
		out[0] = 'V'
	}
	if cs.LCDStatInterruptEnabled {
		out[1] = 'L'
	}
	if cs.TimerInterruptEnabled {
		out[2] = 'T'
	}
	if cs.SerialInterruptEnabled {
		out[3] = 'S'
	}
	if cs.JoypadInterruptEnabled {
		out[4] = 'J'
	}
	return string(out)
}
func (cs *cpuState) imeToString() string {
	if cs.InterruptMasterEnable {
		return "1"
	}
	return "0"
}
func (cs *cpuState) DebugStatusLine() string {

	return fmt.Sprintf("Step:%08d, ", cs.Steps) +
		fmt.Sprintf("Cycles:%08d, ", cs.Cycles) +
		fmt.Sprintf("(*PC)[0:2]:%02x%02x%02x, ", cs.read(cs.PC), cs.read(cs.PC+1), cs.read(cs.PC+2)) +
		fmt.Sprintf("(*SP):%04x, ", cs.read16(cs.SP)) +
		fmt.Sprintf("[PC:%04x ", cs.PC) +
		fmt.Sprintf("SP:%04x ", cs.SP) +
		fmt.Sprintf("AF:%04x ", cs.getAF()) +
		fmt.Sprintf("BC:%04x ", cs.getBC()) +
		fmt.Sprintf("DE:%04x ", cs.getDE()) +
		fmt.Sprintf("HL:%04x ", cs.getHL()) +
		fmt.Sprintf("IME:%v ", cs.imeToString()) +
		fmt.Sprintf("IE:%v ", cs.ieToString()) +
		fmt.Sprintf("IF:%v ", cs.ifToString()) +
		fmt.Sprintf("LY:%02x ", cs.LCD.LYReg) +
		fmt.Sprintf("LYC:%02x ", cs.LCD.LYCReg) +
		fmt.Sprintf("LC:%02x ", cs.LCD.readControlReg()) +
		fmt.Sprintf("LS:%02x ", cs.LCD.readStatusReg()) +
		fmt.Sprintf("ROM:%d]", cs.Mem.mbc.GetROMBankNumber())
}

func addOpA(cs *cpuState, cycles uint, val byte) {
	cs.setOp8(cycles, &cs.A, cs.A+val, (zFlag(cs.A+val) | hFlagAdd(cs.A, val) | cFlagAdd(cs.A, val)))
}
func adcOpA(cs *cpuState, cycles uint, val byte) {
	carry := (cs.F >> 4) & 0x01
	cs.setOp8(cycles, &cs.A, cs.A+val+carry, (zFlag(cs.A+val+carry) | hFlagAdc(cs.A, val, cs.F) | cFlagAdc(cs.A, val, cs.F)))
}
func subOpA(cs *cpuState, cycles uint, val byte) {
	cs.setOp8(cycles, &cs.A, cs.A-val, (zFlag(cs.A-val) | 0x100 | hFlagSub(cs.A, val) | cFlagSub(cs.A, val)))
}
func sbcOpA(cs *cpuState, cycles uint, val byte) {
	carry := (cs.F >> 4) & 0x01
	cs.setOp8(cycles, &cs.A, cs.A-val-carry, (zFlag(cs.A-val-carry) | 0x100 | hFlagSbc(cs.A, val, cs.F) | cFlagSbc(cs.A, val, cs.F)))
}
func andOpA(cs *cpuState, cycles uint, val byte) {
	cs.setOp8(cycles, &cs.A, cs.A&val, (zFlag(cs.A&val) | 0x010))
}
func xorOpA(cs *cpuState, cycles uint, val byte) {
	cs.setOp8(cycles, &cs.A, cs.A^val, zFlag(cs.A^val))
}
func orOpA(cs *cpuState, cycles uint, val byte) {
	cs.setOp8(cycles, &cs.A, cs.A|val, zFlag(cs.A|val))
}
func cpOp(cs *cpuState, cycles uint, val byte) {
	cs.setOpFn(cycles, func() {}, (zFlag(cs.A-val) | hFlagSub(cs.A, val) | cFlagSub(cs.A, val) | 0x0100))
}

func (cs *cpuState) callOp(cycles uint, callAddr uint16) {
	cs.pushOp16(cycles, cs.PC)
	cs.PC = callAddr
}

// NOTE: should be the relevant bits only
func (cs *cpuState) getRegFromOpBits(opBits byte) *byte {
	switch opBits {
	case 0:
		return &cs.B
	case 1:
		return &cs.C
	case 2:
		return &cs.D
	case 3:
		return &cs.E
	case 4:
		return &cs.H
	case 5:
		return &cs.L
	case 6:
		return nil // (hl)
	case 7:
		return &cs.A
	}
	panic("getRegFromOpBits: unknown bits passed")
}

func (cs *cpuState) getCyclesAndValFromOpBits(cyclesReg uint, cyclesHL uint, opcode byte) (uint, byte) {
	if reg := cs.getRegFromOpBits(opcode & 0x07); reg != nil {
		return cyclesReg, *reg
	}
	return cyclesHL, cs.followHL()
}

// opcode >> 3
var isSimpleOp = []bool{
	false, false, false, false, false, false, false, false,
	true, true, true, true, true, true, false, true,
	true, true, true, true, true, true, true, true,
	false, false, false, false, false, false, false, false,
}

// opcode >> 3
var simpleOpFnTable = []func(*cpuState, uint, byte){
	nil, nil, nil, nil, nil, nil, nil, nil,
	setOpB, setOpC, setOpD, setOpE, setOpH, setOpL, nil, setOpA,
	addOpA, adcOpA, subOpA, sbcOpA, andOpA, xorOpA, orOpA, cpOp,
}

func (cs *cpuState) cpuReadAndIncPC() byte {
	val := cs.read(cs.PC)
	cs.PC++
	return val
}

func (cs *cpuState) cpuReadAndIncPC16() uint16 {
	lsb := cs.read(cs.PC)
	cs.PC++
	msb := cs.read(cs.PC)
	cs.PC++
	return (uint16(msb) << 8) | uint16(lsb)
}

func (cs *cpuState) stepOpcode() {

	opcode := cs.cpuReadAndIncPC()

	// simple cases [ ld R, R_OR_(HL) or ALU_OP R_OR_(HL) ]
	sel := opcode >> 3
	if isSimpleOp[sel] {
		cycles, val := cs.getCyclesAndValFromOpBits(4, 8, opcode)
		simpleOpFnTable[sel](cs, cycles, val)
		return
	}

	// complex cases
	switch opcode {

	case 0x00: // nop
		cs.setOpFn(4, func() {}, 0x2222)
	case 0x01: // ld bc, n16
		cs.setOpBC(12, cs.cpuReadAndIncPC16())
	case 0x02: // ld (bc), a
		cs.setOpMem8(8, cs.getBC(), cs.A, 0x2222)
	case 0x03: // inc bc
		cs.setOpBC(8, cs.getBC()+1)
	case 0x04: // inc b
		cs.incOpReg(&cs.B)
	case 0x05: // dec b
		cs.decOpReg(&cs.B)
	case 0x06: // ld b, n8
		setOpB(cs, 8, cs.cpuReadAndIncPC())
	case 0x07: // rlca
		cs.rlcaOp()

	case 0x08: // ld (a16), sp
		cs.setOpMem16(20, cs.cpuReadAndIncPC16(), cs.SP, 0x2222)
	case 0x09: // add hl, bc
		v1, v2 := cs.getHL(), cs.getBC()
		cs.setOp16(8, cs.setHL, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x0a: // ld a, (bc)
		setOpA(cs, 8, cs.followBC())
	case 0x0b: // dec bc
		cs.setOpBC(8, cs.getBC()-1)
	case 0x0c: // inc c
		cs.incOpReg(&cs.C)
	case 0x0d: // dec c
		cs.decOpReg(&cs.C)
	case 0x0e: // ld c, n8
		setOpC(cs, 8, cs.cpuReadAndIncPC())
	case 0x0f: // rrca
		cs.rrcaOp()

	case 0x10: // stop
		cs.setOpFn(4, func() { cs.InStopMode = true }, 0x2222)
	case 0x11: // ld de, n16
		cs.setOpDE(12, cs.cpuReadAndIncPC16())
	case 0x12: // ld (de), a
		cs.setOpMem8(8, cs.getDE(), cs.A, 0x2222)
	case 0x13: // inc de
		cs.setOpDE(8, cs.getDE()+1)
	case 0x14: // inc d
		cs.incOpReg(&cs.D)
	case 0x15: // dec d
		cs.decOpReg(&cs.D)
	case 0x16: // ld d, n8
		setOpD(cs, 8, cs.cpuReadAndIncPC())
	case 0x17: // rla
		cs.rlaOp()

	case 0x18: // jr r8
		cs.jmpRel8(12, 12, true, int8(cs.cpuReadAndIncPC()))
	case 0x19: // add hl, de
		v1, v2 := cs.getHL(), cs.getDE()
		cs.setOp16(8, cs.setHL, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x1a: // ld a, (de)
		setOpA(cs, 8, cs.followDE())
	case 0x1b: // dec de
		cs.setOpDE(8, cs.getDE()-1)
	case 0x1c: // inc e
		cs.incOpReg(&cs.E)
	case 0x1d: // dec e
		cs.decOpReg(&cs.E)
	case 0x1e: // ld e, n8
		setOpE(cs, 8, cs.cpuReadAndIncPC())
	case 0x1f: // rra
		cs.rraOp()

	case 0x20: // jr nz, r8
		cs.jmpRel8(12, 8, !cs.getZeroFlag(), int8(cs.cpuReadAndIncPC()))
	case 0x21: // ld hl, n16
		cs.setOpHL(12, cs.cpuReadAndIncPC16())
	case 0x22: // ld (hl++), a
		cs.setOpMem8(8, cs.getHL(), cs.A, 0x2222)
		cs.setHL(cs.getHL() + 1)
	case 0x23: // inc hl
		cs.setOpHL(8, cs.getHL()+1)
	case 0x24: // inc h
		cs.incOpReg(&cs.H)
	case 0x25: // dec h
		cs.decOpReg(&cs.H)
	case 0x26: // ld h, n8
		setOpH(cs, 8, cs.cpuReadAndIncPC())
	case 0x27: // daa
		cs.daaOp()

	case 0x28: // jr z, r8
		cs.jmpRel8(12, 8, cs.getZeroFlag(), int8(cs.cpuReadAndIncPC()))
	case 0x29: // add hl, hl
		v1, v2 := cs.getHL(), cs.getHL()
		cs.setOp16(8, cs.setHL, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x2a: // ld a, (hl++)
		setOpA(cs, 8, cs.followHL())
		cs.setHL(cs.getHL() + 1)
	case 0x2b: // dec hl
		cs.setOpHL(8, cs.getHL()-1)
	case 0x2c: // inc l
		cs.incOpReg(&cs.L)
	case 0x2d: // dec l
		cs.decOpReg(&cs.L)
	case 0x2e: // ld l, n8
		setOpL(cs, 8, cs.cpuReadAndIncPC())
	case 0x2f: // cpl
		cs.setOp8(4, &cs.A, ^cs.A, 0x2112)

	case 0x30: // jr nc, r8
		cs.jmpRel8(12, 8, !cs.getCarryFlag(), int8(cs.cpuReadAndIncPC()))
	case 0x31: // ld sp, n16
		cs.setOpSP(12, cs.cpuReadAndIncPC16())
	case 0x32: // ld (hl--) a
		cs.setOpMem8(8, cs.getHL(), cs.A, 0x2222)
		cs.setHL(cs.getHL() - 1)
	case 0x33: // inc sp
		cs.setOpSP(8, cs.SP+1)
	case 0x34: // inc (hl)
		cs.incOpHL()
	case 0x35: // dec (hl)
		cs.decOpHL()
	case 0x36: // ld (hl) n8
		cs.setOpMem8(12, cs.getHL(), cs.cpuReadAndIncPC(), 0x2222)
	case 0x37: // scf
		cs.setOpFn(4, func() {}, 0x2001)

	case 0x38: // jr c, r8
		cs.jmpRel8(12, 8, cs.getCarryFlag(), int8(cs.cpuReadAndIncPC()))
	case 0x39: // add hl, sp
		v1, v2 := cs.getHL(), cs.SP
		cs.setOp16(8, cs.setHL, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x3a: // ld a, (hl--)
		setOpA(cs, 8, cs.followHL())
		cs.setHL(cs.getHL() - 1)
	case 0x3b: // dec sp
		cs.setOpSP(8, cs.SP-1)
	case 0x3c: // inc a
		cs.incOpReg(&cs.A)
	case 0x3d: // dec a
		cs.decOpReg(&cs.A)
	case 0x3e: // ld a, n8
		setOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0x3f: // ccf
		carry := uint16((cs.F>>4)&0x01) ^ 0x01
		cs.setOpFn(4, func() {}, 0x2000|carry)

	case 0x70: // ld (hl), b
		cs.setOpMem8(8, cs.getHL(), cs.B, 0x2222)
	case 0x71: // ld (hl), c
		cs.setOpMem8(8, cs.getHL(), cs.C, 0x2222)
	case 0x72: // ld (hl), d
		cs.setOpMem8(8, cs.getHL(), cs.D, 0x2222)
	case 0x73: // ld (hl), e
		cs.setOpMem8(8, cs.getHL(), cs.E, 0x2222)
	case 0x74: // ld (hl), h
		cs.setOpMem8(8, cs.getHL(), cs.H, 0x2222)
	case 0x75: // ld (hl), l
		cs.setOpMem8(8, cs.getHL(), cs.L, 0x2222)
	case 0x76: // halt
		cs.setOpFn(4, func() { cs.InHaltMode = true }, 0x2222)
	case 0x77: // ld (hl), a
		cs.setOpMem8(8, cs.getHL(), cs.A, 0x2222)

	case 0xc0: // ret nz
		cs.jmpRet(20, 8, !cs.getZeroFlag())
	case 0xc1: // pop bc
		cs.popOp16(12, cs.setBC)
	case 0xc2: // jp nz, a16
		cs.jmpAbs16(16, 12, !cs.getZeroFlag(), cs.cpuReadAndIncPC16())
	case 0xc3: // jp a16
		cs.setOpPC(16, cs.cpuReadAndIncPC16())
	case 0xc4: // call nz, a16
		cs.jmpCall(24, 12, !cs.getZeroFlag(), cs.cpuReadAndIncPC16())
	case 0xc5: // push bc
		cs.pushOp16(16, cs.getBC())
	case 0xc6: // add a, n8
		addOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xc7: // rst 00h
		cs.callOp(16, 0x0000)

	case 0xc8: // ret z
		cs.jmpRet(20, 8, cs.getZeroFlag())
	case 0xc9: // ret
		cs.popOp16(16, cs.setPC)
	case 0xca: // jp z, a16
		cs.jmpAbs16(16, 12, cs.getZeroFlag(), cs.cpuReadAndIncPC16())
	case 0xcb: // extended opcode prefix
		cs.stepExtendedOpcode()
	case 0xcc: // call z, a16
		cs.jmpCall(24, 12, cs.getZeroFlag(), cs.cpuReadAndIncPC16())
	case 0xcd: // call a16
		cs.callOp(24, cs.cpuReadAndIncPC16())
	case 0xce: // adc a, n8
		adcOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xcf: // rst 08h
		cs.callOp(16, 0x0008)

	case 0xd0: // ret nc
		cs.jmpRet(20, 8, !cs.getCarryFlag())
	case 0xd1: // pop de
		cs.popOp16(12, cs.setDE)
	case 0xd2: // jp nc, a16
		cs.jmpAbs16(16, 12, !cs.getCarryFlag(), cs.cpuReadAndIncPC16())
	case 0xd3:
		cs.illegalOpcode(opcode)
	case 0xd4: // call nc, a16
		cs.jmpCall(24, 12, !cs.getCarryFlag(), cs.cpuReadAndIncPC16())
	case 0xd5: // push de
		cs.pushOp16(16, cs.getDE())
	case 0xd6: // sub n8
		subOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xd7: // rst 10h
		cs.callOp(16, 0x0010)

	case 0xd8: // ret c
		cs.jmpRet(20, 8, cs.getCarryFlag())
	case 0xd9: // reti
		cs.popOp16(16, cs.setPC)
		cs.MasterEnableRequested = true
	case 0xda: // jp c, a16
		cs.jmpAbs16(16, 12, cs.getCarryFlag(), cs.cpuReadAndIncPC16())
	case 0xdb:
		cs.illegalOpcode(opcode)
	case 0xdc: // call c, a16
		cs.jmpCall(24, 12, cs.getCarryFlag(), cs.cpuReadAndIncPC16())
	case 0xdd:
		cs.illegalOpcode(opcode)
	case 0xde: // sbc n8
		sbcOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xdf: // rst 18h
		cs.callOp(16, 0x0018)

	case 0xe0: // ld (0xFF00 + n8), a
		val := cs.cpuReadAndIncPC()
		cs.setOpMem8(12, 0xff00+uint16(val), cs.A, 0x2222)
	case 0xe1: // pop hl
		cs.popOp16(12, cs.setHL)
	case 0xe2: // ld (0xFF00 + c), a
		val := cs.C
		cs.setOpMem8(8, 0xff00+uint16(val), cs.A, 0x2222)
	case 0xe3:
		cs.illegalOpcode(opcode)
	case 0xe4:
		cs.illegalOpcode(opcode)
	case 0xe5: // push hl
		cs.pushOp16(16, cs.getHL())
	case 0xe6: // and n8
		andOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xe7: // rst 20h
		cs.callOp(16, 0x0020)

	case 0xe8: // add sp, r8
		v1, v2 := cs.SP, uint16(int8(cs.cpuReadAndIncPC()))
		cs.setOp16(16, cs.setSP, v1+v2, (hFlagAdd(byte(v1), byte(v2)) | cFlagAdd(byte(v1), byte(v2))))
	case 0xe9: // jp hl (also written jp (hl))
		cs.setOpPC(4, cs.getHL())
	case 0xea: // ld (a16), a
		cs.setOpMem8(16, cs.cpuReadAndIncPC16(), cs.A, 0x2222)
	case 0xeb:
		cs.illegalOpcode(opcode)
	case 0xec:
		cs.illegalOpcode(opcode)
	case 0xed:
		cs.illegalOpcode(opcode)
	case 0xee: // xor n8
		xorOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xef: // rst 28h
		cs.callOp(16, 0x0028)

	case 0xf0: // ld a, (0xFF00 + n8)
		val := cs.cpuReadAndIncPC()
		setOpA(cs, 12, cs.read(0xff00+uint16(val)))
	case 0xf1: // pop af
		cs.popOp16(12, cs.setAF)
	case 0xf2: // ld a, (0xFF00 + c)
		val := cs.C
		setOpA(cs, 8, cs.read(0xff00+uint16(val)))
	case 0xf3: // di
		cs.setOpFn(4, func() { cs.InterruptMasterEnable = false }, 0x2222)
	case 0xf4:
		cs.illegalOpcode(opcode)
	case 0xf5: // push af
		cs.pushOp16(16, cs.getAF())
	case 0xf6: // or n8
		orOpA(cs, 8, cs.cpuReadAndIncPC())
	case 0xf7: // rst 30h
		cs.callOp(16, 0x0030)

	case 0xf8: // ld hl, sp+r8
		v1, v2 := cs.SP, uint16(int8(cs.cpuReadAndIncPC()))
		cs.setOp16(12, cs.setHL, v1+v2, (hFlagAdd(byte(v1), byte(v2)) | cFlagAdd(byte(v1), byte(v2))))
	case 0xf9: // ld sp, hl
		cs.setOpSP(8, cs.getHL())
	case 0xfa: // ld a, (a16)
		setOpA(cs, 16, cs.read(cs.cpuReadAndIncPC16()))
	case 0xfb: // ei
		cs.setOpFn(4, func() { cs.MasterEnableRequested = true }, 0x2222)
	case 0xfc:
		cs.illegalOpcode(opcode)
	case 0xfd:
		cs.illegalOpcode(opcode)
	case 0xfe: // cp a, n8
		cpOp(cs, 8, cs.cpuReadAndIncPC())
	case 0xff: // rst 38h
		cs.callOp(16, 0x0038)

	default:
		cs.stepErr(fmt.Sprintf("Unknown Opcode: 0x%02x\r\n", opcode))
	}
}

func (cs *cpuState) illegalOpcode(opcode uint8) {
	cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
}

func (cs *cpuState) stepExtendedOpcode() {

	extOpcode := cs.cpuReadAndIncPC()

	switch extOpcode & 0xf8 {

	case 0x00: // rlc R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.rlcOp)
	case 0x08: // rrc R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.rrcOp)
	case 0x10: // rl R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.rlOp)
	case 0x18: // rr R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.rrOp)
	case 0x20: // sla R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.slaOp)
	case 0x28: // sra R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.sraOp)
	case 0x30: // swap R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.swapOp)
	case 0x38: // srl R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.srlOp)

	case 0x40: // bit 0, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 0)
	case 0x48: // bit 1, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 1)
	case 0x50: // bit 2, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 2)
	case 0x58: // bit 3, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 3)
	case 0x60: // bit 4, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 4)
	case 0x68: // bit 5, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 5)
	case 0x70: // bit 6, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 6)
	case 0x78: // bit 7, R_OR_(HL)
		cs.bitOp(8, 12, extOpcode, 7)

	case 0x80: // res 0, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(0))
	case 0x88: // res 1, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(1))
	case 0x90: // res 2, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(2))
	case 0x98: // res 3, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(3))
	case 0xa0: // res 4, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(4))
	case 0xa8: // res 5, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(5))
	case 0xb0: // res 6, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(6))
	case 0xb8: // res 6, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getResOp(7))

	case 0xc0: // set 0, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(0))
	case 0xc8: // set 1, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(1))
	case 0xd0: // set 2, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(2))
	case 0xd8: // set 3, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(3))
	case 0xe0: // set 4, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(4))
	case 0xe8: // set 5, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(5))
	case 0xf0: // set 6, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(6))
	case 0xf8: // set 7, R_OR_(HL)
		cs.extSetOp(8, 16, extOpcode, cs.getBitSetOp(7))
	}
}

func (cs *cpuState) extSetOp(cyclesReg uint, cyclesHL uint, opcode byte,
	opFn func(val byte) (result byte, flags uint16)) {

	if reg := cs.getRegFromOpBits(opcode & 0x07); reg != nil {
		result, flags := opFn(*reg)
		cs.setOp8(cyclesReg, reg, result, flags)
	} else {
		result, flags := opFn(cs.followHL())
		cs.setOpMem8(cyclesHL, cs.getHL(), result, flags)
	}
}

func (cs *cpuState) swapOp(val byte) (byte, uint16) {
	result := val>>4 | (val&0x0f)<<4
	return result, zFlag(result)
}

func (cs *cpuState) rlaOp() {
	result, flags := cs.rlOp(cs.A)
	cs.setOp8(4, &cs.A, result, flags&^0x1000) // rla is 000c, unlike other rl's
}
func (cs *cpuState) rlOp(val byte) (byte, uint16) {
	result, carry := (val<<1)|((cs.F>>4)&0x01), (val >> 7)
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) rraOp() {
	result, flags := cs.rrOp(cs.A)
	cs.setOp8(4, &cs.A, result, flags&^0x1000) // rra is 000c, unlike other rr's
}
func (cs *cpuState) rrOp(val byte) (byte, uint16) {
	result, carry := ((cs.F<<3)&0x80)|(val>>1), (val & 0x01)
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) rlcaOp() {
	result, flags := cs.rlcOp(cs.A)
	cs.setOp8(4, &cs.A, result, flags&^0x1000) // rlca is 000c, unlike other rlc's
}
func (cs *cpuState) rlcOp(val byte) (byte, uint16) {
	result, carry := (val<<1)|(val>>7), val>>7
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) rrcaOp() {
	result, flags := cs.rrcOp(cs.A)
	cs.setOp8(4, &cs.A, result, flags&^0x1000) // rrca is 000c, unlike other rrc's
}
func (cs *cpuState) rrcOp(val byte) (byte, uint16) {
	result, carry := (val<<7)|(val>>1), (val & 0x01)
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) srlOp(val byte) (byte, uint16) {
	result, carry := val>>1, val&0x01
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) slaOp(val byte) (byte, uint16) {
	result, carry := val<<1, val>>7
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) sraOp(val byte) (byte, uint16) {
	result, carry := (val&0x80)|(val>>1), val&0x01
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) bitOp(cyclesReg uint, cyclesHL uint, opcode byte, bitNum uint8) {
	cycles, val := cs.getCyclesAndValFromOpBits(cyclesReg, cyclesHL, opcode)
	cs.setOpFn(cycles, func() {}, zFlag(val&(1<<bitNum))|0x012)
}

func (cs *cpuState) getResOp(bitNum uint) func(byte) (byte, uint16) {
	return func(val byte) (byte, uint16) {
		result := val &^ (1 << bitNum)
		return result, 0x2222
	}
}

func (cs *cpuState) getBitSetOp(bitNum uint8) func(byte) (byte, uint16) {
	return func(val byte) (byte, uint16) {
		result := val | (1 << bitNum)
		return result, 0x2222
	}
}

func (cs *cpuState) stepErr(msg string) {
	fmt.Println(msg)
	fmt.Println(cs.DebugStatusLine())
	panic("stepErr()")
}
