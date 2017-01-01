package dmgo

import "fmt"

func (cs *cpuState) setOp8(cycles uint, instLen uint16, reg *uint8, val uint8, flags uint16) {
	cs.runCycles(cycles)
	cs.PC += instLen
	*reg = val
	cs.setFlags(flags)
}

func (cs *cpuState) setOpA(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.A, val, flags)
}
func (cs *cpuState) setOpB(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.B, val, flags)
}
func (cs *cpuState) setOpC(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.C, val, flags)
}
func (cs *cpuState) setOpD(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.D, val, flags)
}
func (cs *cpuState) setOpE(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.E, val, flags)
}
func (cs *cpuState) setOpL(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.L, val, flags)
}
func (cs *cpuState) setOpH(cycles uint, instLen uint16, val uint8, flags uint16) {
	cs.setOp8(cycles, instLen, &cs.H, val, flags)
}

func (cs *cpuState) setOp16(cycles uint, instLen uint16, setFn func(uint16), val uint16, flags uint16) {
	cs.runCycles(cycles)
	cs.PC += instLen
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
	cs.PC += instLen
	cs.write(addr, val)
	cs.setFlags(flags)
}
func (cs *cpuState) setOpMem16(cycles uint, instLen uint16, addr uint16, val uint16, flags uint16) {
	cs.runCycles(cycles)
	cs.PC += instLen
	cs.write16(addr, val)
	cs.setFlags(flags)
}

func (cs *cpuState) jmpRel8(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, relAddr int8) {
	cs.PC += instLen
	if test {
		cs.runCycles(cyclesTaken)
		cs.PC = uint16(int(cs.PC) + int(relAddr))
	} else {
		cs.runCycles(cyclesNotTaken)
	}
}
func (cs *cpuState) jmpAbs16(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, addr uint16) {
	cs.PC += instLen
	if test {
		cs.runCycles(cyclesTaken)
		cs.PC = addr
	} else {
		cs.runCycles(cyclesNotTaken)
	}
}

func (cs *cpuState) jmpCall(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, addr uint16) {
	if test {
		cs.pushOp16(cyclesTaken, instLen, cs.PC+instLen)
		cs.PC = addr
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

func (cs *cpuState) setOpFn(cycles uint, instLen uint16, fn func(), flags uint16) {
	cs.runCycles(cycles)
	cs.PC += instLen
	fn()
	cs.setFlags(flags)
}

func (cs *cpuState) pushOp16(cycles uint, instLen uint16, val uint16) {
	cs.setOpMem16(cycles, instLen, cs.SP-2, val, 0x2222)
	cs.SP -= 2
}
func (cs *cpuState) popOp16(cycles uint, instLen uint16, setFn func(val uint16)) {
	cs.setOpFn(cycles, instLen, func() { setFn(cs.read16(cs.SP)) }, 0x2222)
	cs.SP += 2
}

func (cs *cpuState) incOpReg(reg *byte) {
	val := *reg
	cs.setOp8(4, 1, reg, val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
}
func (cs *cpuState) incOpHL() {
	val := cs.followHL()
	cs.setOpMem8(12, 1, cs.getHL(), val+1, (zFlag(val+1) | hFlagAdd(val, 1) | 0x0002))
}

func (cs *cpuState) decOpReg(reg *byte) {
	val := *reg
	cs.setOp8(4, 1, reg, val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
}
func (cs *cpuState) decOpHL() {
	val := cs.followHL()
	cs.setOpMem8(12, 1, cs.getHL(), val-1, (zFlag(val-1) | hFlagSub(val, 1) | 0x0102))
}

func (cs *cpuState) daaOp() {

	diff := byte(0)
	newCarryFlag := uint16(0)
	if cs.getSubFlag() {
		if cs.getHalfCarryFlag() {
			diff += 0x06
		}
		if cs.getCarryFlag() {
			newCarryFlag = 0x0001
			diff += 0x60
		}
	} else {
		if cs.A&0x0f > 0x09 || cs.getHalfCarryFlag() {
			diff += 0x06
		}
		if cs.A > 0x99 || cs.getCarryFlag() {
			newCarryFlag = 0x0001
			diff += 0x60
		}
	}

	if cs.getSubFlag() {
		cs.A -= diff
	} else {
		cs.A += diff
	}

	cs.setFlags(zFlag(cs.A) | 0x0200 | newCarryFlag)
	cs.runCycles(4)
	cs.PC++
}

func (cs *cpuState) ifToString() string {
	out := []byte("    ")
	if cs.vBlankIRQ {
		out[0] = 'V'
	}
	if cs.lcdStatIRQ {
		out[1] = 'L'
	}
	if cs.serialIRQ {
		out[2] = 'S'
	}
	if cs.joypadIRQ {
		out[3] = 'J'
	}
	return string(out)
}
func (cs *cpuState) ieToString() string {
	out := []byte("    ")
	if cs.vBlankInterruptEnabled {
		out[0] = 'V'
	}
	if cs.lcdStatInterruptEnabled {
		out[1] = 'L'
	}
	if cs.serialInterruptEnabled {
		out[2] = 'S'
	}
	if cs.joypadInterruptEnabled {
		out[3] = 'J'
	}
	return string(out)
}
func (cs *cpuState) imeToString() string {
	if cs.interruptMasterEnable {
		return "1"
	}
	return "0"
}
func (cs *cpuState) lcdStatInterruptsToString() string {
	out := []byte("    ")
	if cs.vBlankInterruptEnabled {
		out[0] = 'Y'
	}
	if cs.lcdStatInterruptEnabled {
		out[1] = 'O'
	}
	if cs.serialInterruptEnabled {
		out[2] = 'V'
	}
	if cs.joypadInterruptEnabled {
		out[3] = 'H'
	}
	return string(out)
}
func (cs *cpuState) debugStatusLine() string {

	return fmt.Sprintf("step:%08d, ", cs.steps) +
		fmt.Sprintf("(*PC)[0:2]:%02x%02x%02x, ", cs.read(cs.PC), cs.read(cs.PC+1), cs.read(cs.PC+2)) +
		fmt.Sprintf("(*SP):%04x, ", cs.read16(cs.SP)) +
		fmt.Sprintf("[PC:%04x ", cs.PC) +
		fmt.Sprintf("SP:%04x ", cs.SP) +
		fmt.Sprintf("af:%04x ", cs.getAF()) +
		fmt.Sprintf("bc:%04x ", cs.getBC()) +
		fmt.Sprintf("de:%04x ", cs.getDE()) +
		fmt.Sprintf("hl:%04x ", cs.getHL()) +
		fmt.Sprintf("ime:%v ", cs.imeToString()) +
		fmt.Sprintf("ie:%v ", cs.ieToString()) +
		fmt.Sprintf("if:%v ", cs.ifToString()) +
		fmt.Sprintf("Ly:%02x ", cs.lcd.lyReg) +
		fmt.Sprintf("Lyc:%02x ", cs.lcd.lycReg) +
		fmt.Sprintf("Lc:%02x ", cs.lcd.readControlReg()) +
		fmt.Sprintf("Ls:%02x ", cs.lcd.readStatusReg()) +
		fmt.Sprintf("ROM:%d]", cs.mem.mbc.GetROMBankNumber())
}

func (cs *cpuState) addOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.A+val, (zFlag(cs.A+val) | hFlagAdd(cs.A, val) | cFlagAdd(cs.A, val)))
}
func (cs *cpuState) adcOpA(cycles uint, instLen uint16, val byte) {
	carry := (cs.F >> 4) & 0x01
	cs.setOpA(cycles, instLen, cs.A+val+carry, (zFlag(cs.A+val+carry) | hFlagAdc(cs.A, val, cs.F) | cFlagAdc(cs.A, val, cs.F)))
}
func (cs *cpuState) subOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.A-val, (zFlag(cs.A-val) | 0x100 | hFlagSub(cs.A, val) | cFlagSub(cs.A, val)))
}
func (cs *cpuState) sbcOpA(cycles uint, instLen uint16, val byte) {
	carry := (cs.F >> 4) & 0x01
	cs.setOpA(cycles, instLen, cs.A-val-carry, (zFlag(cs.A-val-carry) | 0x100 | hFlagSbc(cs.A, val, cs.F) | cFlagSbc(cs.A, val, cs.F)))
}
func (cs *cpuState) andOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.A&val, (zFlag(cs.A&val) | 0x010))
}
func (cs *cpuState) xorOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.A^val, zFlag(cs.A^val))
}
func (cs *cpuState) orOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.A|val, zFlag(cs.A|val))
}
func (cs *cpuState) cpOp(cycles uint, instLen uint16, val byte) {
	cs.setOpFn(cycles, instLen, func() {}, (zFlag(cs.A-val) | hFlagSub(cs.A, val) | cFlagSub(cs.A, val) | 0x0100))
}

func (cs *cpuState) callOp(cycles uint, instLen, callAddr uint16) {
	cs.pushOp16(cycles, instLen, cs.PC+instLen)
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

func (cs *cpuState) loadOp(cyclesReg uint, cyclesHL uint, instLen uint16,
	opcode byte, fnPtr func(uint, uint16, byte, uint16)) {

	cycles, val := cs.getCyclesAndValFromOpBits(cyclesReg, cyclesHL, opcode)
	fnPtr(cycles, instLen, val, 0x2222)
}

func (cs *cpuState) aluOp(cyclesReg uint, cyclesHL uint, instLen uint16,
	opcode byte, fnPtr func(uint, uint16, byte)) {

	cycles, val := cs.getCyclesAndValFromOpBits(cyclesReg, cyclesHL, opcode)
	fnPtr(cycles, instLen, val)
}

func (cs *cpuState) stepSimpleOp(opcode byte) bool {
	switch opcode & 0xf8 {
	case 0x40: // ld b, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpB)
	case 0x48: // ld c, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpC)
	case 0x50: // ld d, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpD)
	case 0x58: // ld e, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpE)
	case 0x60: // ld h, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpH)
	case 0x68: // ld l, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpL)

	case 0x78: // ld a, R_OR_(HL)
		cs.loadOp(4, 8, 1, opcode, cs.setOpA)

	case 0x80: // add R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.addOpA)
	case 0x88: // adc R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.adcOpA)
	case 0x90: // sub R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.subOpA)
	case 0x98: // sbc R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.sbcOpA)
	case 0xa0: // and R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.andOpA)
	case 0xa8: // xor R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.xorOpA)
	case 0xb0: // or R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.orOpA)
	case 0xb8: // cp R_OR_(HL)
		cs.aluOp(4, 8, 1, opcode, cs.cpOp)
	default:
		return false
	}
	return true
}

func (cs *cpuState) stepOpcode() {

	cs.steps++
	opcode := cs.read(cs.PC)

	// simple cases
	if cs.stepSimpleOp(opcode) {
		return
	}

	// complex cases
	switch opcode {

	case 0x00: // nop
		cs.setOpFn(4, 1, func() {}, 0x2222)
	case 0x01: // ld bc, n16
		cs.setOpBC(12, 3, cs.read16(cs.PC+1), 0x2222)
	case 0x02: // ld (bc), a
		cs.setOpMem8(8, 1, cs.getBC(), cs.A, 0x2222)
	case 0x03: // inc bc
		cs.setOpBC(8, 1, cs.getBC()+1, 0x2222)
	case 0x04: // inc b
		cs.incOpReg(&cs.B)
	case 0x05: // dec b
		cs.decOpReg(&cs.B)
	case 0x06: // ld b, n8
		cs.setOpB(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x07: // rlca
		cs.rlcaOp()

	case 0x08: // ld (a16), sp
		cs.setOpMem16(20, 3, cs.read16(cs.PC+1), cs.SP, 0x2222)
	case 0x09: // add hl, bc
		v1, v2 := cs.getHL(), cs.getBC()
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x0a: // ld a, (bc)
		cs.setOpA(8, 1, cs.followBC(), 0x2222)
	case 0x0b: // dec bc
		cs.setOpBC(8, 1, cs.getBC()-1, 0x2222)
	case 0x0c: // inc c
		cs.incOpReg(&cs.C)
	case 0x0d: // dec c
		cs.decOpReg(&cs.C)
	case 0x0e: // ld c, n8
		cs.setOpC(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x0f: // rrca
		cs.rrcaOp()

	case 0x10: // stop
		cs.setOpFn(4, 2, func() { cs.inStopMode = true }, 0x2222)
	case 0x11: // ld de, n16
		cs.setOpDE(12, 3, cs.read16(cs.PC+1), 0x2222)
	case 0x12: // ld (de), a
		cs.setOpMem8(8, 1, cs.getDE(), cs.A, 0x2222)
	case 0x13: // inc de
		cs.setOpDE(8, 1, cs.getDE()+1, 0x2222)
	case 0x14: // inc d
		cs.incOpReg(&cs.D)
	case 0x15: // dec d
		cs.decOpReg(&cs.D)
	case 0x16: // ld d, n8
		cs.setOpD(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x17: // rla
		cs.rlaOp()

	case 0x18: // jr r8
		cs.jmpRel8(12, 12, 2, true, int8(cs.read(cs.PC+1)))
	case 0x19: // add hl, de
		v1, v2 := cs.getHL(), cs.getDE()
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x1a: // ld a, (de)
		cs.setOpA(8, 1, cs.followDE(), 0x2222)
	case 0x1b: // dec de
		cs.setOpDE(8, 1, cs.getDE()-1, 0x2222)
	case 0x1c: // inc e
		cs.incOpReg(&cs.E)
	case 0x1d: // dec e
		cs.decOpReg(&cs.E)
	case 0x1e: // ld e, n8
		cs.setOpE(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x1f: // rra
		cs.rraOp()

	case 0x20: // jr nz, r8
		cs.jmpRel8(12, 8, 2, !cs.getZeroFlag(), int8(cs.read(cs.PC+1)))
	case 0x21: // ld hl, n16
		cs.setOpHL(12, 3, cs.read16(cs.PC+1), 0x2222)
	case 0x22: // ld (hl++), a
		cs.setOpMem8(8, 1, cs.getHL(), cs.A, 0x2222)
		cs.setHL(cs.getHL() + 1)
	case 0x23: // inc hl
		cs.setOpHL(8, 1, cs.getHL()+1, 0x2222)
	case 0x24: // inc h
		cs.incOpReg(&cs.H)
	case 0x25: // dec h
		cs.decOpReg(&cs.H)
	case 0x26: // ld h, d8
		cs.setOpH(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x27: // daa
		cs.daaOp()

	case 0x28: // jr z, r8
		cs.jmpRel8(12, 8, 2, cs.getZeroFlag(), int8(cs.read(cs.PC+1)))
	case 0x29: // add hl, hl
		v1, v2 := cs.getHL(), cs.getHL()
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x2a: // ld a, (hl++)
		cs.setOpA(8, 1, cs.followHL(), 0x2222)
		cs.setHL(cs.getHL() + 1)
	case 0x2b: // dec hl
		cs.setOpHL(8, 1, cs.getHL()-1, 0x2222)
	case 0x2c: // inc l
		cs.incOpReg(&cs.L)
	case 0x2d: // dec l
		cs.decOpReg(&cs.L)
	case 0x2e: // ld l, d8
		cs.setOpL(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x2f: // cpl
		cs.setOpA(4, 1, ^cs.A, 0x2112)

	case 0x30: // jr z, r8
		cs.jmpRel8(12, 8, 2, !cs.getCarryFlag(), int8(cs.read(cs.PC+1)))
	case 0x31: // ld sp, n16
		cs.setOpSP(12, 3, cs.read16(cs.PC+1), 0x2222)
	case 0x32: // ld (hl--) a
		cs.setOpMem8(8, 1, cs.getHL(), cs.A, 0x2222)
		cs.setHL(cs.getHL() - 1)
	case 0x33: // inc sp
		cs.setOpSP(8, 1, cs.SP+1, 0x2222)
	case 0x34: // inc (hl)
		cs.incOpHL()
	case 0x35: // dec (hl)
		cs.decOpHL()
	case 0x36: // ld (hl) n8
		cs.setOpMem8(12, 2, cs.getHL(), cs.read(cs.PC+1), 0x2222)
	case 0x37: // scf
		cs.setOpFn(4, 1, func() {}, 0x2001)

	case 0x38: // jr c, r8
		cs.jmpRel8(12, 8, 2, cs.getCarryFlag(), int8(cs.read(cs.PC+1)))
	case 0x39: // add hl, sp
		v1, v2 := cs.getHL(), cs.SP
		cs.setOpHL(8, 1, v1+v2, (0x2000 | hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0x3a: // ld a, (hl--)
		cs.setOpA(8, 1, cs.followHL(), 0x2222)
		cs.setHL(cs.getHL() - 1)
	case 0x3b: // dec sp
		cs.setOpSP(8, 1, cs.SP-1, 0x2222)
	case 0x3c: // inc a
		cs.incOpReg(&cs.A)
	case 0x3d: // dec a
		cs.decOpReg(&cs.A)
	case 0x3e: // ld a, n8
		cs.setOpA(8, 2, cs.read(cs.PC+1), 0x2222)
	case 0x3f: // ccf
		carry := uint16((cs.F>>4)&0x01) ^ 0x01
		cs.setOpFn(4, 1, func() {}, 0x2000|carry)

	case 0x70: // ld (hl), b
		cs.setOpMem8(8, 1, cs.getHL(), cs.B, 0x2222)
	case 0x71: // ld (hl), c
		cs.setOpMem8(8, 1, cs.getHL(), cs.C, 0x2222)
	case 0x72: // ld (hl), d
		cs.setOpMem8(8, 1, cs.getHL(), cs.D, 0x2222)
	case 0x73: // ld (hl), e
		cs.setOpMem8(8, 1, cs.getHL(), cs.E, 0x2222)
	case 0x74: // ld (hl), h
		cs.setOpMem8(8, 1, cs.getHL(), cs.H, 0x2222)
	case 0x75: // ld (hl), l
		cs.setOpMem8(8, 1, cs.getHL(), cs.L, 0x2222)
	case 0x76: // halt
		cs.setOpFn(4, 1, func() { cs.inHaltMode = true }, 0x2222)
	case 0x77: // ld (hl), a
		cs.setOpMem8(8, 1, cs.getHL(), cs.A, 0x2222)

	case 0xc0: // ret nz
		cs.jmpRet(20, 8, 1, !cs.getZeroFlag())
	case 0xc1: // pop bc
		cs.popOp16(12, 1, cs.setBC)
	case 0xc2: // jp nz, a16
		cs.jmpAbs16(16, 12, 3, !cs.getZeroFlag(), cs.read16(cs.PC+1))
	case 0xc3: // jp a16
		cs.setOpPC(16, 3, cs.read16(cs.PC+1), 0x2222)
	case 0xc4: // call nz, a16
		cs.jmpCall(24, 12, 3, !cs.getZeroFlag(), cs.read16(cs.PC+1))
	case 0xc5: // push bc
		cs.pushOp16(16, 1, cs.getBC())
	case 0xc6: // add a, n8
		cs.addOpA(8, 2, cs.read(cs.PC+1))
	case 0xc7: // rst 00h
		cs.callOp(16, 1, 0x0000)

	case 0xc8: // ret z
		cs.jmpRet(20, 8, 1, cs.getZeroFlag())
	case 0xc9: // ret
		cs.popOp16(16, 1, cs.setPC)
	case 0xca: // jp z, a16
		cs.jmpAbs16(16, 12, 3, cs.getZeroFlag(), cs.read16(cs.PC+1))
	case 0xcb: // extended opcode prefix
		cs.stepExtendedOpcode()
	case 0xcc: // call z, a16
		cs.jmpCall(24, 12, 3, cs.getZeroFlag(), cs.read16(cs.PC+1))
	case 0xcd: // call a16
		cs.callOp(24, 3, cs.read16(cs.PC+1))
	case 0xce: // adc a, n8
		cs.adcOpA(8, 2, cs.read(cs.PC+1))
	case 0xcf: // rst 08h
		cs.callOp(16, 1, 0x0008)

	case 0xd0: // ret nc
		cs.jmpRet(20, 8, 1, !cs.getCarryFlag())
	case 0xd1: // pop de
		cs.popOp16(12, 1, cs.setDE)
	case 0xd2: // jp nc, a16
		cs.jmpAbs16(16, 12, 3, !cs.getCarryFlag(), cs.read16(cs.PC+1))
	case 0xd3: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xd4: // call nc, a16
		cs.jmpCall(24, 12, 3, !cs.getCarryFlag(), cs.read16(cs.PC+1))
	case 0xd5: // push de
		cs.pushOp16(16, 1, cs.getDE())
	case 0xd6: // sub n8
		cs.subOpA(8, 2, cs.read(cs.PC+1))
	case 0xd7: // rst 10h
		cs.callOp(16, 1, 0x0010)

	case 0xd8: // ret c
		cs.jmpRet(20, 8, 1, cs.getCarryFlag())
	case 0xd9: // reti
		cs.popOp16(16, 1, cs.setPC)
		cs.interruptMasterEnable = true
	case 0xda: // jp c, a16
		cs.jmpAbs16(16, 12, 3, cs.getCarryFlag(), cs.read16(cs.PC+1))
	case 0xdb: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xdc: // call c, a16
		cs.jmpCall(24, 12, 3, cs.getCarryFlag(), cs.read16(cs.PC+1))
	case 0xdd: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xde: // sbc n8
		cs.sbcOpA(8, 2, cs.read(cs.PC+1))
	case 0xdf: // rst 18h
		cs.callOp(16, 1, 0x0018)

	case 0xe0: // ld (0xFF00 + n8), a
		val := cs.read(cs.PC + 1)
		cs.setOpMem8(12, 2, 0xff00+uint16(val), cs.A, 0x2222)
	case 0xe1: // pop hl
		cs.popOp16(12, 1, cs.setHL)
	case 0xe2: // ld (0xFF00 + c), a
		val := cs.C
		cs.setOpMem8(8, 1, 0xff00+uint16(val), cs.A, 0x2222)
	case 0xe3: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xe4: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xe5: // push hl
		cs.pushOp16(16, 1, cs.getHL())
	case 0xe6: // and n8
		cs.andOpA(8, 2, cs.read(cs.PC+1))
	case 0xe7: // rst 20h
		cs.callOp(16, 1, 0x0020)

	case 0xe8: // add sp, r8
		v1, v2 := cs.SP, uint16(int8(cs.read(cs.PC+1)))
		cs.setOpSP(16, 2, v1+v2, (hFlagAdd(byte(v1), byte(v2)) | cFlagAdd(byte(v1), byte(v2))))
	case 0xe9: // jp hl (also written jp (hl))
		cs.setOpPC(4, 1, cs.getHL(), 0x2222)
	case 0xea: // ld (a16), a
		cs.setOpMem8(16, 3, cs.read16(cs.PC+1), cs.A, 0x2222)
	case 0xeb: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xec: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xed: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xee: // xor n8
		cs.xorOpA(8, 2, cs.read(cs.PC+1))
	case 0xef: // rst 28h
		cs.callOp(16, 1, 0x0028)

	case 0xf0: // ld a, (0xFF00 + n8)
		val := cs.read(cs.PC + 1)
		cs.setOpA(12, 2, cs.read(0xff00+uint16(val)), 0x2222)
	case 0xf1: // pop af
		cs.popOp16(12, 1, cs.setAF)
	case 0xf2: // ld a, (0xFF00 + c)
		val := cs.C
		cs.setOpA(8, 1, cs.read(0xff00+uint16(val)), 0x2222)
	case 0xf3: // di
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnable = false }, 0x2222)
	case 0xf4: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xf5: // push af
		cs.pushOp16(16, 1, cs.getAF())
	case 0xf6: // or n8
		cs.orOpA(8, 2, cs.read(cs.PC+1))
	case 0xf7: // rst 30h
		cs.callOp(16, 1, 0x0030)

	case 0xf8: // ld hl, sp+r8
		v1, v2 := cs.SP, uint16(int8(cs.read(cs.PC+1)))
		cs.setOpHL(12, 2, v1+v2, (hFlagAdd(byte(v1), byte(v2)) | cFlagAdd(byte(v1), byte(v2))))
	case 0xf9: // ld sp, hl
		cs.setOpSP(8, 1, cs.getHL(), 0x2222)
	case 0xfa: // ld a, (a16)
		cs.setOpA(16, 3, cs.read(cs.read16(cs.PC+1)), 0x2222)
	case 0xfb: // ei
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnable = true }, 0x2222)
	case 0xfc: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xfd: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xfe: // cp a, n8
		cs.cpOp(8, 2, cs.read(cs.PC+1))
	case 0xff: // rst 38h
		cs.callOp(16, 1, 0x0038)

	default:
		cs.stepErr(fmt.Sprintf("Unknown Opcode: 0x%02x\r\n", opcode))
	}
}

func (cs *cpuState) stepExtendedOpcode() {

	extOpcode := cs.read(cs.PC + 1)

	switch extOpcode & 0xf8 {

	case 0x00: // rlc R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.rlcOp)
	case 0x08: // rrc R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.rrcOp)
	case 0x10: // rl R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.rlOp)
	case 0x18: // rr R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.rrOp)
	case 0x20: // sla R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.slaOp)
	case 0x28: // sra R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.sraOp)
	case 0x30: // swap R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.swapOp)
	case 0x38: // srl R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.srlOp)

	case 0x40: // bit 0, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 0)
	case 0x48: // bit 1, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 1)
	case 0x50: // bit 2, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 2)
	case 0x58: // bit 3, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 3)
	case 0x60: // bit 4, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 4)
	case 0x68: // bit 5, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 5)
	case 0x70: // bit 6, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 6)
	case 0x78: // bit 7, R_OR_(HL)
		cs.bitOp(8, 16, 2, extOpcode, 7)

	case 0x80: // res 0, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(0))
	case 0x88: // res 1, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(1))
	case 0x90: // res 2, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(2))
	case 0x98: // res 3, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(3))
	case 0xa0: // res 4, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(4))
	case 0xa8: // res 5, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(5))
	case 0xb0: // res 6, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(6))
	case 0xb8: // res 6, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getResOp(7))

	case 0xc0: // set 0, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(0))
	case 0xc8: // set 1, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(1))
	case 0xd0: // set 2, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(2))
	case 0xd8: // set 3, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(3))
	case 0xe0: // set 4, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(4))
	case 0xe8: // set 5, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(5))
	case 0xf0: // set 6, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(6))
	case 0xf8: // set 7, R_OR_(HL)
		cs.extSetOp(8, 16, 2, extOpcode, cs.getBitSetOp(7))
	}
}

func (cs *cpuState) extSetOp(cyclesReg uint, cyclesHL uint, instLen uint16, opcode byte,
	opFn func(val byte) (result byte, flags uint16)) {

	if reg := cs.getRegFromOpBits(opcode & 0x07); reg != nil {
		result, flags := opFn(*reg)
		cs.setOp8(cyclesReg, instLen, reg, result, flags)
	} else {
		result, flags := opFn(cs.followHL())
		cs.setOpMem8(cyclesHL, instLen, cs.getHL(), result, flags)
	}
}

func (cs *cpuState) swapOp(val byte) (byte, uint16) {
	result := val>>4 | (val&0x0f)<<4
	return result, zFlag(result)
}

func (cs *cpuState) rlaOp() {
	result, flags := cs.rlOp(cs.A)
	cs.setOp8(4, 1, &cs.A, result, flags&^0x1000) // rla is 000c, unlike other rl's
}
func (cs *cpuState) rlOp(val byte) (byte, uint16) {
	result, carry := (val<<1)|((cs.F>>4)&0x01), (val >> 7)
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) rraOp() {
	result, flags := cs.rrOp(cs.A)
	cs.setOp8(4, 1, &cs.A, result, flags&^0x1000) // rra is 000c, unlike other rr's
}
func (cs *cpuState) rrOp(val byte) (byte, uint16) {
	result, carry := ((cs.F<<3)&0x80)|(val>>1), (val & 0x01)
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) rlcaOp() {
	result, flags := cs.rlcOp(cs.A)
	cs.setOp8(4, 1, &cs.A, result, flags&^0x1000) // rlca is 000c, unlike other rlc's
}
func (cs *cpuState) rlcOp(val byte) (byte, uint16) {
	result, carry := (val<<1)|(val>>7), val>>7
	return result, (zFlag(result) | uint16(carry))
}

func (cs *cpuState) rrcaOp() {
	result, flags := cs.rrcOp(cs.A)
	cs.setOp8(4, 1, &cs.A, result, flags&^0x1000) // rrca is 000c, unlike other rrc's
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

func (cs *cpuState) bitOp(cyclesReg uint, cyclesHL uint, instLen uint16, opcode byte, bitNum uint8) {
	cycles, val := cs.getCyclesAndValFromOpBits(cyclesReg, cyclesHL, opcode)
	cs.setOpFn(cycles, instLen, func() {}, zFlag(val&(1<<bitNum))|0x012)
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
	fmt.Println(cs.debugStatusLine())
	panic("stepErr()")
}
