package dmgo

import "fmt"

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
	cs.setFlags(flags)
}
func (cs *cpuState) setOpMem16(cycles uint, instLen uint16, addr uint16, val uint16, flags uint16) {
	cs.runCycles(cycles)
	cs.pc += instLen
	cs.write16(addr, val)
	cs.setFlags(flags)
}

func (cs *cpuState) jmpRel8(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, relAddr int8) {
	cs.pc += instLen
	if test {
		cs.runCycles(cyclesTaken)
		cs.pc = uint16(int(cs.pc) + int(relAddr))
	} else {
		cs.runCycles(cyclesNotTaken)
	}
}
func (cs *cpuState) jmpAbs16(cyclesTaken uint, cyclesNotTaken uint, instLen uint16, test bool, addr uint16) {
	cs.pc += instLen
	if test {
		cs.runCycles(cyclesTaken)
		cs.pc = addr
	} else {
		cs.runCycles(cyclesNotTaken)
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

func (cs *cpuState) followBC() byte { return cs.read(cs.getBC()) }
func (cs *cpuState) followDE() byte { return cs.read(cs.getDE()) }
func (cs *cpuState) followHL() byte { return cs.read(cs.getHL()) }
func (cs *cpuState) followSP() byte { return cs.read(cs.sp) }
func (cs *cpuState) followPC() byte { return cs.read(cs.pc) }

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

func (cs *cpuState) pushOp16(cycles uint, instLen uint16, val uint16) {
	cs.setOpMem16(cycles, instLen, cs.sp-2, val, 0x2222)
	cs.sp -= 2
}
func (cs *cpuState) popOp16(cycles uint, instLen uint16, setFn func(val uint16)) {
	cs.setOpFn(cycles, instLen, func() { setFn(cs.read16(cs.sp)) }, 0x2222)
	cs.sp += 2
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
	if cs.a&0x0f > 0x09 || cs.getHalfCarryFlag() {
		diff += 0x06
		cs.f |= 0x20
	}
	if cs.a&0xf0 > 0x90 || cs.getCarryFlag() {
		diff += 0x60
		cs.f |= 0x10
	}
	if cs.getAddSubFlag() {
		cs.a -= diff
	} else {
		cs.a += diff
	}
	cs.runCycles(4)
	cs.pc++
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
		fmt.Sprintf("(*pc)[0:2]:%02x%02x%02x, ", cs.read(cs.pc), cs.read(cs.pc+1), cs.read(cs.pc+2)) +
		fmt.Sprintf("(*sp):%04x, ", cs.read16(cs.sp)) +
		fmt.Sprintf("[pc:%04x ", cs.pc) +
		fmt.Sprintf("sp:%04x ", cs.sp) +
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
	cs.setOpA(cycles, instLen, cs.a+val, (zFlag(cs.a+val) | hFlagAdd(cs.a, val) | cFlagAdd(cs.a, val)))
}
func (cs *cpuState) adcOpA(cycles uint, instLen uint16, val byte) {
	carry := (cs.f >> 4) & 0x01
	cs.setOpA(cycles, instLen, cs.a+val+carry, (zFlag(cs.a+val+carry) | hFlagAdc(cs.a, val, cs.f) | cFlagAdc(cs.a, val, cs.f)))
}
func (cs *cpuState) subOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.a-val, (zFlag(cs.a-val) | 0x100 | hFlagSub(cs.a, val) | cFlagSub(cs.a, val)))
}
func (cs *cpuState) sbcOpA(cycles uint, instLen uint16, val byte) {
	carry := (cs.f >> 4) & 0x01
	cs.setOpA(cycles, instLen, cs.a-val-carry, (zFlag(cs.a-val-carry) | 0x100 | hFlagSbc(cs.a, val, cs.f) | cFlagSbc(cs.a, val, cs.f)))
}
func (cs *cpuState) andOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.a&val, (zFlag(cs.a&val) | 0x010))
}
func (cs *cpuState) xorOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.a^val, zFlag(cs.a^val))
}
func (cs *cpuState) orOpA(cycles uint, instLen uint16, val byte) {
	cs.setOpA(cycles, instLen, cs.a|val, zFlag(cs.a|val))
}
func (cs *cpuState) cpOp(cycles uint, instLen uint16, val byte) {
	cs.setOpFn(cycles, instLen, func() {}, (zFlag(cs.a-val) | hFlagSub(cs.a, val) | cFlagSub(cs.a, val) | 0x0100))
}

func (cs *cpuState) callOp(cycles uint, instLen, callAddr uint16) {
	cs.pushOp16(cycles, instLen, cs.pc+instLen)
	cs.pc = callAddr
}

// NOTE: should be the relevant bits only
func (cs *cpuState) getSrcFromOpBits(opBits byte) *byte {
	switch opBits {
	case 0:
		return &cs.b
	case 1:
		return &cs.c
	case 2:
		return &cs.d
	case 3:
		return &cs.e
	case 4:
		return &cs.h
	case 5:
		return &cs.l
	case 6:
		return nil // (hl)
	case 7:
		return &cs.a
	}
	panic("getSrcFromOpBits: unknown bits passed")
}

func (cs *cpuState) loadOp(cyclesReg uint, cyclesHL uint, instLen uint16,
	opcode byte, fnPtr func(uint, uint16, byte, uint16)) {
	if reg := cs.getSrcFromOpBits(opcode & 0x07); reg != nil {
		fnPtr(cyclesReg, instLen, *reg, 0x2222)
	} else {
		fnPtr(cyclesHL, instLen, cs.followHL(), 0x2222)
	}
}

func (cs *cpuState) aluOp(cyclesReg uint, cyclesHL uint, instLen uint16,
	opcode byte, fnPtr func(uint, uint16, byte)) {
	if reg := cs.getSrcFromOpBits(opcode & 0x07); reg != nil {
		fnPtr(cyclesReg, instLen, *reg)
	} else {
		fnPtr(cyclesHL, instLen, cs.followHL())
	}
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
	opcode := cs.read(cs.pc)

	// simple cases
	if cs.stepSimpleOp(opcode) {
		return
	}

	// complex cases
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
		cs.incOpReg(&cs.b)
	case 0x05: // dec b
		cs.decOpReg(&cs.b)
	case 0x06: // ld b, n8
		cs.setOpB(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x07: // rlca
		cs.rlcaOp()

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
		cs.incOpReg(&cs.c)
	case 0x0d: // dec c
		cs.decOpReg(&cs.c)
	case 0x0e: // ld c, n8
		cs.setOpC(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x0f: // rrca
		cs.rrcaOp()

	case 0x10: // stop
		cs.setOpFn(4, 2, func() { cs.inStopMode = true }, 0x2222)
	case 0x11: // ld de, n16
		cs.setOpDE(12, 3, cs.read16(cs.pc+1), 0x2222)
	case 0x12: // ld (de), a
		cs.setOpMem8(8, 1, cs.getDE(), cs.a, 0x2222)
	case 0x13: // inc de
		cs.setOpDE(8, 1, cs.getDE()+1, 0x2222)
	case 0x14: // inc d
		cs.incOpReg(&cs.d)
	case 0x15: // dec d
		cs.decOpReg(&cs.d)
	case 0x16: // ld d, n8
		cs.setOpD(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x17: // rla
		cs.rlaOp()

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
		cs.incOpReg(&cs.e)
	case 0x1d: // dec e
		cs.decOpReg(&cs.e)
	case 0x1e: // ld e, n8
		cs.setOpE(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x1f: // rra
		cs.rraOp()

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
		cs.incOpReg(&cs.h)
	case 0x25: // dec h
		cs.decOpReg(&cs.h)
	case 0x26: // ld h, d8
		cs.setOpH(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x27: // daa
		cs.daaOp()

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
		cs.incOpReg(&cs.l)
	case 0x2d: // dec l
		cs.decOpReg(&cs.l)
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
		cs.incOpHL()
	case 0x35: // dec (hl)
		cs.decOpHL()
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
		cs.incOpReg(&cs.a)
	case 0x3d: // dec a
		cs.decOpReg(&cs.a)
	case 0x3e: // ld a, n8
		cs.setOpA(8, 2, cs.read(cs.pc+1), 0x2222)
	case 0x3f: // ccf
		carry := uint16((cs.f>>4)&0x01) ^ 0x01
		cs.setOpFn(4, 1, func() {}, 0x2000|carry)

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
		cs.setOpFn(4, 1, func() { cs.inHaltMode = true }, 0x2222)
	case 0x77: // ld (hl), a
		cs.setOpMem8(8, 1, cs.getHL(), cs.a, 0x2222)

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
		cs.addOpA(8, 2, cs.read(cs.pc+1))
	case 0xc7: // rst 00h
		cs.callOp(16, 1, 0x0000)

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
		cs.callOp(24, 3, cs.read16(cs.pc+1))
	case 0xce: // adc a, n8
		cs.adcOpA(8, 2, cs.read(cs.pc+1))
	case 0xcf: // rst 08h
		cs.callOp(16, 1, 0x0008)

	case 0xd0: // ret nc
		cs.jmpRet(20, 8, 1, !cs.getCarryFlag())
	case 0xd1: // pop de
		cs.popOp16(12, 1, cs.setDE)
	case 0xd2: // jp nc, a16
		cs.jmpAbs16(16, 12, 3, !cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xd3: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xd4: // call nc, a16
		cs.jmpCall(24, 12, 3, !cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xd5: // push de
		cs.pushOp16(16, 1, cs.getDE())
	case 0xd6: // sub n8
		cs.subOpA(8, 2, cs.read(cs.pc+1))
	case 0xd7: // rst 10h
		cs.callOp(16, 1, 0x0010)

	case 0xd8: // ret c
		cs.jmpRet(20, 8, 1, cs.getCarryFlag())
	case 0xd9: // reti
		cs.popOp16(16, 1, cs.setPC)
		cs.interruptMasterEnable = true
	case 0xda: // jp c, a16
		cs.jmpAbs16(16, 12, 3, cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xdb: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xdc: // call c, a16
		cs.jmpCall(24, 12, 3, cs.getCarryFlag(), cs.read16(cs.pc+1))
	case 0xdd: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xde: // sbc n8
		cs.sbcOpA(8, 2, cs.read(cs.pc+1))
	case 0xdf: // rst 18h
		cs.callOp(16, 1, 0x0018)

	case 0xe0: // ld (0xFF00 + n8), a
		val := cs.read(cs.pc + 1)
		cs.setOpMem8(12, 2, 0xff00+uint16(val), cs.a, 0x2222)
	case 0xe1: // pop hl
		cs.popOp16(12, 1, cs.setHL)
	case 0xe2: // ld (0xFF00 + c), a
		val := cs.c
		cs.setOpMem8(8, 1, 0xff00+uint16(val), cs.a, 0x2222)
	case 0xe3: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xe4: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xe5: // push hl
		cs.pushOp16(16, 1, cs.getHL())
	case 0xe6: // and n8
		cs.andOpA(8, 2, cs.read(cs.pc+1))
	case 0xe7: // rst 20h
		cs.callOp(16, 1, 0x0020)

	case 0xe8: // add sp, r8
		v1, v2 := cs.sp, uint16(int(cs.read(cs.pc+1)))
		cs.setOpSP(16, 2, v1+v2, (hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0xe9: // jp hl (also written jp (hl))
		cs.setOpPC(4, 1, cs.getHL(), 0x2222)
	case 0xea: // ld (a16), a
		cs.setOpMem8(16, 3, cs.read16(cs.pc+1), cs.a, 0x2222)
	case 0xeb: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xec: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xed: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xee: // xor n8
		cs.xorOpA(8, 2, cs.read(cs.pc+1))
	case 0xef: // rst 28h
		cs.callOp(16, 1, 0x0028)

	case 0xf0: // ld a, (0xFF00 + n8)
		val := cs.read(cs.pc + 1)
		cs.setOpA(12, 2, cs.read(0xff00+uint16(val)), 0x2222)
	case 0xf1: // pop af
		cs.popOp16(12, 1, cs.setAF)
	case 0xf2: // ld a, (0xFF00 + c)
		val := cs.c
		cs.setOpA(8, 1, cs.read(0xff00+uint16(val)), 0x2222)
	case 0xf3: // di
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnable = false }, 0x2222)
	case 0xf4: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xf5: // push af
		cs.pushOp16(16, 1, cs.getAF())
	case 0xf6: // or n8
		cs.orOpA(8, 2, cs.read(cs.pc+1))
	case 0xf7: // rst 30h
		cs.callOp(16, 1, 0x0030)

	case 0xf8: // ld hl, sp+r8
		v1, v2 := cs.sp, uint16(int(cs.read(cs.pc+1)))
		cs.setOpHL(12, 2, v1+v2, (hFlagAdd16(v1, v2) | cFlagAdd16(v1, v2)))
	case 0xf9: // ld sp, hl
		cs.setOpSP(8, 1, cs.getHL(), 0x2222)
	case 0xfa: // ld a, (a16)
		cs.setOpA(16, 3, cs.read(cs.read16(cs.pc+1)), 0x2222)
	case 0xfb: // ei
		cs.setOpFn(4, 1, func() { cs.interruptMasterEnable = true }, 0x2222)
	case 0xfc: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xfd: // illegal
		cs.stepErr(fmt.Sprintf("illegal opcode %02x", opcode))
	case 0xfe: // cp a, n8
		cs.cpOp(8, 2, cs.read(cs.pc+1))
	case 0xff: // rst 38h
		cs.callOp(16, 1, 0x0038)

	default:
		cs.stepErr(fmt.Sprintf("Unknown Opcode: 0x%02x\r\n", opcode))
	}
}

func (cs *cpuState) stepExtendedOpcode() {

	extOpcode := cs.read(cs.pc + 1)

	switch extOpcode {

	case 0x00: // rlc b
		cs.rlcOpReg(&cs.b)
	case 0x01: // rlc c
		cs.rlcOpReg(&cs.c)
	case 0x02: // rlc d
		cs.rlcOpReg(&cs.d)
	case 0x03: // rlc e
		cs.rlcOpReg(&cs.e)
	case 0x04: // rlc h
		cs.rlcOpReg(&cs.h)
	case 0x05: // rlc l
		cs.rlcOpReg(&cs.l)
	case 0x06: // rlc (hl)
		cs.rlcOpHL()
	case 0x07: // rlc a
		cs.rlcOpReg(&cs.a)

	case 0x08: // rrc b
		cs.rrcOpReg(&cs.b)
	case 0x09: // rrc c
		cs.rrcOpReg(&cs.c)
	case 0x0a: // rrc d
		cs.rrcOpReg(&cs.d)
	case 0x0b: // rrc e
		cs.rrcOpReg(&cs.e)
	case 0x0c: // rrc h
		cs.rrcOpReg(&cs.h)
	case 0x0d: // rrc l
		cs.rrcOpReg(&cs.l)
	case 0x0e: // rrc (hl)
		cs.rrcOpHL()
	case 0x0f: // rrc a
		cs.rrcOpReg(&cs.a)

	case 0x10: // rl b
		cs.rlOpReg(&cs.b)
	case 0x11: // rl c
		cs.rlOpReg(&cs.c)
	case 0x12: // rl d
		cs.rlOpReg(&cs.d)
	case 0x13: // rl e
		cs.rlOpReg(&cs.e)
	case 0x14: // rl h
		cs.rlOpReg(&cs.h)
	case 0x15: // rl l
		cs.rlOpReg(&cs.l)
	case 0x16: // rl (hl)
		cs.rlOpHL()
	case 0x17: // rl a
		cs.rlOpReg(&cs.a)

	case 0x18: // rr b
		cs.rrOpReg(&cs.b)
	case 0x19: // rr c
		cs.rrOpReg(&cs.c)
	case 0x1a: // rr d
		cs.rrOpReg(&cs.d)
	case 0x1b: // rr e
		cs.rrOpReg(&cs.e)
	case 0x1c: // rr h
		cs.rrOpReg(&cs.h)
	case 0x1d: // rr l
		cs.rrOpReg(&cs.l)
	case 0x1e: // rr (hl)
		cs.rrOpHL()
	case 0x1f: // rr a
		cs.rrOpReg(&cs.a)

	case 0x20: // sla b
		cs.slaOpReg(&cs.b)
	case 0x21: // sla c
		cs.slaOpReg(&cs.c)
	case 0x22: // sla d
		cs.slaOpReg(&cs.d)
	case 0x23: // sla e
		cs.slaOpReg(&cs.e)
	case 0x24: // sla h
		cs.slaOpReg(&cs.h)
	case 0x25: // sla l
		cs.slaOpReg(&cs.l)
	case 0x26: // sla (hl)
		cs.slaOpHL()
	case 0x27: // sla a
		cs.slaOpReg(&cs.a)

	case 0x28: // sra b
		cs.sraOpReg(&cs.b)
	case 0x29: // sra c
		cs.sraOpReg(&cs.c)
	case 0x2a: // sra d
		cs.sraOpReg(&cs.d)
	case 0x2b: // sra e
		cs.sraOpReg(&cs.e)
	case 0x2c: // sra h
		cs.sraOpReg(&cs.h)
	case 0x2d: // sra l
		cs.sraOpReg(&cs.l)
	case 0x2e: // sra (hl)
		cs.sraOpHL()
	case 0x2f: // sra a
		cs.sraOpReg(&cs.a)

	case 0x30: // swap b
		cs.swapOpReg(&cs.b)
	case 0x31: // swap c
		cs.swapOpReg(&cs.c)
	case 0x32: // swap d
		cs.swapOpReg(&cs.d)
	case 0x33: // swap e
		cs.swapOpReg(&cs.e)
	case 0x34: // swap h
		cs.swapOpReg(&cs.h)
	case 0x35: // swap l
		cs.swapOpReg(&cs.l)
	case 0x36: // swap (hl)
		cs.swapOpHL()
	case 0x37: // swap a
		cs.swapOpReg(&cs.a)

	case 0x38: // srl b
		cs.srlOpReg(&cs.b)
	case 0x39: // srl c
		cs.srlOpReg(&cs.c)
	case 0x3a: // srl d
		cs.srlOpReg(&cs.d)
	case 0x3b: // srl e
		cs.srlOpReg(&cs.e)
	case 0x3c: // srl h
		cs.srlOpReg(&cs.h)
	case 0x3d: // srl l
		cs.srlOpReg(&cs.l)
	case 0x3e: // srl (hl)
		cs.srlOpHL()
	case 0x3f: // srl a
		cs.srlOpReg(&cs.a)

	case 0x40: // bit 0, b
		cs.bitOpReg(0, cs.b)
	case 0x41: // bit 0, c
		cs.bitOpReg(0, cs.c)
	case 0x42: // bit 0, d
		cs.bitOpReg(0, cs.d)
	case 0x43: // bit 0, e
		cs.bitOpReg(0, cs.e)
	case 0x44: // bit 0, h
		cs.bitOpReg(0, cs.h)
	case 0x45: // bit 0, l
		cs.bitOpReg(0, cs.l)
	case 0x46: // bit 0, (hl)
		cs.bitOpHL(0)
	case 0x47: // bit 0, a
		cs.bitOpReg(0, cs.a)

	case 0x48: // bit 1, b
		cs.bitOpReg(1, cs.b)
	case 0x49: // bit 1, c
		cs.bitOpReg(1, cs.c)
	case 0x4a: // bit 1, d
		cs.bitOpReg(1, cs.d)
	case 0x4b: // bit 1, e
		cs.bitOpReg(1, cs.e)
	case 0x4c: // bit 1, h
		cs.bitOpReg(1, cs.h)
	case 0x4d: // bit 1, l
		cs.bitOpReg(1, cs.l)
	case 0x4e: // bit 1, (hl)
		cs.bitOpHL(1)
	case 0x4f: // bit 1, a
		cs.bitOpReg(1, cs.a)

	case 0x50: // bit 2, b
		cs.bitOpReg(2, cs.b)
	case 0x51: // bit 2, c
		cs.bitOpReg(2, cs.c)
	case 0x52: // bit 2, d
		cs.bitOpReg(2, cs.d)
	case 0x53: // bit 2, e
		cs.bitOpReg(2, cs.e)
	case 0x54: // bit 2, h
		cs.bitOpReg(2, cs.h)
	case 0x55: // bit 2, l
		cs.bitOpReg(2, cs.l)
	case 0x56: // bit 2, (hl)
		cs.bitOpHL(2)
	case 0x57: // bit 2, a
		cs.bitOpReg(2, cs.a)

	case 0x58: // bit 3, b
		cs.bitOpReg(3, cs.b)
	case 0x59: // bit 3, c
		cs.bitOpReg(3, cs.c)
	case 0x5a: // bit 3, d
		cs.bitOpReg(3, cs.d)
	case 0x5b: // bit 3, e
		cs.bitOpReg(3, cs.e)
	case 0x5c: // bit 3, h
		cs.bitOpReg(3, cs.h)
	case 0x5d: // bit 3, l
		cs.bitOpReg(3, cs.l)
	case 0x5e: // bit 3, (hl)
		cs.bitOpHL(3)
	case 0x5f: // bit 3, a
		cs.bitOpReg(3, cs.a)

	case 0x60: // bit 4, b
		cs.bitOpReg(4, cs.b)
	case 0x61: // bit 4, c
		cs.bitOpReg(4, cs.c)
	case 0x62: // bit 4, d
		cs.bitOpReg(4, cs.d)
	case 0x63: // bit 4, e
		cs.bitOpReg(4, cs.e)
	case 0x64: // bit 4, h
		cs.bitOpReg(4, cs.h)
	case 0x65: // bit 4, l
		cs.bitOpReg(4, cs.l)
	case 0x66: // bit 4, (hl)
		cs.bitOpHL(4)
	case 0x67: // bit 4, a
		cs.bitOpReg(4, cs.a)

	case 0x68: // bit 5, b
		cs.bitOpReg(5, cs.b)
	case 0x69: // bit 5, c
		cs.bitOpReg(5, cs.c)
	case 0x6a: // bit 5, d
		cs.bitOpReg(5, cs.d)
	case 0x6b: // bit 5, e
		cs.bitOpReg(5, cs.e)
	case 0x6c: // bit 5, h
		cs.bitOpReg(5, cs.h)
	case 0x6d: // bit 5, l
		cs.bitOpReg(5, cs.l)
	case 0x6e: // bit 5, (hl)
		cs.bitOpHL(5)
	case 0x6f: // bit 5, a
		cs.bitOpReg(5, cs.a)

	case 0x70: // bit 6, b
		cs.bitOpReg(6, cs.b)
	case 0x71: // bit 6, c
		cs.bitOpReg(6, cs.c)
	case 0x72: // bit 6, d
		cs.bitOpReg(6, cs.d)
	case 0x73: // bit 6, e
		cs.bitOpReg(6, cs.e)
	case 0x74: // bit 6, h
		cs.bitOpReg(6, cs.h)
	case 0x75: // bit 6, l
		cs.bitOpReg(6, cs.l)
	case 0x76: // bit 6, (hl)
		cs.bitOpHL(6)
	case 0x77: // bit 6, a
		cs.bitOpReg(6, cs.a)

	case 0x78: // bit 7, b
		cs.bitOpReg(7, cs.b)
	case 0x79: // bit 7, c
		cs.bitOpReg(7, cs.c)
	case 0x7a: // bit 7, d
		cs.bitOpReg(7, cs.d)
	case 0x7b: // bit 7, e
		cs.bitOpReg(7, cs.e)
	case 0x7c: // bit 7, h
		cs.bitOpReg(7, cs.h)
	case 0x7d: // bit 7, l
		cs.bitOpReg(7, cs.l)
	case 0x7e: // bit 7, (hl)
		cs.bitOpHL(7)
	case 0x7f: // bit 7, a
		cs.bitOpReg(7, cs.a)

	case 0x80: // res 0, b
		cs.resOpReg(0, &cs.b)
	case 0x81: // res 0, c
		cs.resOpReg(0, &cs.c)
	case 0x82: // res 0, d
		cs.resOpReg(0, &cs.d)
	case 0x83: // res 0, e
		cs.resOpReg(0, &cs.e)
	case 0x84: // res 0, h
		cs.resOpReg(0, &cs.h)
	case 0x85: // res 0, l
		cs.resOpReg(0, &cs.l)
	case 0x86: // res 0, (hl)
		cs.resOpHL(0)
	case 0x87: // res 0, a
		cs.resOpReg(0, &cs.a)

	case 0x88: // res 1, b
		cs.resOpReg(1, &cs.b)
	case 0x89: // res 1, c
		cs.resOpReg(1, &cs.c)
	case 0x8a: // res 1, d
		cs.resOpReg(1, &cs.d)
	case 0x8b: // res 1, e
		cs.resOpReg(1, &cs.e)
	case 0x8c: // res 1, h
		cs.resOpReg(1, &cs.h)
	case 0x8d: // res 1, l
		cs.resOpReg(1, &cs.l)
	case 0x8e: // res 1, (hl)
		cs.resOpHL(1)
	case 0x8f: // res 1, a
		cs.resOpReg(1, &cs.a)

	case 0x90: // res 2, b
		cs.resOpReg(2, &cs.b)
	case 0x91: // res 2, c
		cs.resOpReg(2, &cs.c)
	case 0x92: // res 2, d
		cs.resOpReg(2, &cs.d)
	case 0x93: // res 2, e
		cs.resOpReg(2, &cs.e)
	case 0x94: // res 2, h
		cs.resOpReg(2, &cs.h)
	case 0x95: // res 2, l
		cs.resOpReg(2, &cs.l)
	case 0x96: // res 2, (hl)
		cs.resOpHL(2)
	case 0x97: // res 2, a
		cs.resOpReg(2, &cs.a)

	case 0x98: // res 3, b
		cs.resOpReg(3, &cs.b)
	case 0x99: // res 3, c
		cs.resOpReg(3, &cs.c)
	case 0x9a: // res 3, d
		cs.resOpReg(3, &cs.d)
	case 0x9b: // res 3, e
		cs.resOpReg(3, &cs.e)
	case 0x9c: // res 3, h
		cs.resOpReg(3, &cs.h)
	case 0x9d: // res 3, l
		cs.resOpReg(3, &cs.l)
	case 0x9e: // res 3, (hl)
		cs.resOpHL(3)
	case 0x9f: // res 3, a
		cs.resOpReg(3, &cs.a)

	case 0xa0: // res 4, b
		cs.resOpReg(4, &cs.b)
	case 0xa1: // res 4, c
		cs.resOpReg(4, &cs.c)
	case 0xa2: // res 4, d
		cs.resOpReg(4, &cs.d)
	case 0xa3: // res 4, e
		cs.resOpReg(4, &cs.e)
	case 0xa4: // res 4, h
		cs.resOpReg(4, &cs.h)
	case 0xa5: // res 4, l
		cs.resOpReg(4, &cs.l)
	case 0xa6: // res 4, (hl)
		cs.resOpHL(4)
	case 0xa7: // res 4, a
		cs.resOpReg(4, &cs.a)

	case 0xa8: // res 5, b
		cs.resOpReg(5, &cs.b)
	case 0xa9: // res 5, c
		cs.resOpReg(5, &cs.c)
	case 0xaa: // res 5, d
		cs.resOpReg(5, &cs.d)
	case 0xab: // res 5, e
		cs.resOpReg(5, &cs.e)
	case 0xac: // res 5, h
		cs.resOpReg(5, &cs.h)
	case 0xad: // res 5, l
		cs.resOpReg(5, &cs.l)
	case 0xae: // res 5, (hl)
		cs.resOpHL(5)
	case 0xaf: // res 5, a
		cs.resOpReg(5, &cs.a)

	case 0xb0: // res 6, b
		cs.resOpReg(6, &cs.b)
	case 0xb1: // res 6, c
		cs.resOpReg(6, &cs.c)
	case 0xb2: // res 6, d
		cs.resOpReg(6, &cs.d)
	case 0xb3: // res 6, e
		cs.resOpReg(6, &cs.e)
	case 0xb4: // res 6, h
		cs.resOpReg(6, &cs.h)
	case 0xb5: // res 6, l
		cs.resOpReg(6, &cs.l)
	case 0xb6: // res 6, (hl)
		cs.resOpHL(6)
	case 0xb7: // res 6, a
		cs.resOpReg(6, &cs.a)

	case 0xb8: // res 7, b
		cs.resOpReg(7, &cs.b)
	case 0xb9: // res 7, c
		cs.resOpReg(7, &cs.c)
	case 0xba: // res 7, d
		cs.resOpReg(7, &cs.d)
	case 0xbb: // res 7, e
		cs.resOpReg(7, &cs.e)
	case 0xbc: // res 7, h
		cs.resOpReg(7, &cs.h)
	case 0xbd: // res 7, l
		cs.resOpReg(7, &cs.l)
	case 0xbe: // res 7, (hl)
		cs.resOpHL(7)
	case 0xbf: // res 7, a
		cs.resOpReg(7, &cs.a)

	case 0xc0: // set 0, b
		cs.bSetOpReg(0, &cs.b)
	case 0xc1: // set 0, c
		cs.bSetOpReg(0, &cs.c)
	case 0xc2: // set 0, d
		cs.bSetOpReg(0, &cs.d)
	case 0xc3: // set 0, e
		cs.bSetOpReg(0, &cs.e)
	case 0xc4: // set 0, h
		cs.bSetOpReg(0, &cs.h)
	case 0xc5: // set 0, l
		cs.bSetOpReg(0, &cs.l)
	case 0xc6: // set 0, (hl)
		cs.bSetOpHL(0)
	case 0xc7: // set 0, a
		cs.bSetOpReg(0, &cs.a)

	case 0xc8: // set 1, b
		cs.bSetOpReg(1, &cs.b)
	case 0xc9: // set 1, c
		cs.bSetOpReg(1, &cs.c)
	case 0xca: // set 1, d
		cs.bSetOpReg(1, &cs.d)
	case 0xcb: // set 1, e
		cs.bSetOpReg(1, &cs.e)
	case 0xcc: // set 1, h
		cs.bSetOpReg(1, &cs.h)
	case 0xcd: // set 1, l
		cs.bSetOpReg(1, &cs.l)
	case 0xce: // set 1, (hl)
		cs.bSetOpHL(1)
	case 0xcf: // set 1, a
		cs.bSetOpReg(1, &cs.a)

	case 0xd0: // set 2, b
		cs.bSetOpReg(2, &cs.b)
	case 0xd1: // set 2, c
		cs.bSetOpReg(2, &cs.c)
	case 0xd2: // set 2, d
		cs.bSetOpReg(2, &cs.d)
	case 0xd3: // set 2, e
		cs.bSetOpReg(2, &cs.e)
	case 0xd4: // set 2, h
		cs.bSetOpReg(2, &cs.h)
	case 0xd5: // set 2, l
		cs.bSetOpReg(2, &cs.l)
	case 0xd6: // set 2, (hl)
		cs.bSetOpHL(2)
	case 0xd7: // set 2, a
		cs.bSetOpReg(2, &cs.a)

	case 0xd8: // set 3, b
		cs.bSetOpReg(3, &cs.b)
	case 0xd9: // set 3, c
		cs.bSetOpReg(3, &cs.c)
	case 0xda: // set 3, d
		cs.bSetOpReg(3, &cs.d)
	case 0xdb: // set 3, e
		cs.bSetOpReg(3, &cs.e)
	case 0xdc: // set 3, h
		cs.bSetOpReg(3, &cs.h)
	case 0xdd: // set 3, l
		cs.bSetOpReg(3, &cs.l)
	case 0xde: // set 3, (hl)
		cs.bSetOpHL(3)
	case 0xdf: // set 3, a
		cs.bSetOpReg(3, &cs.a)

	case 0xe0: // set 4, b
		cs.bSetOpReg(4, &cs.b)
	case 0xe1: // set 4, c
		cs.bSetOpReg(4, &cs.c)
	case 0xe2: // set 4, d
		cs.bSetOpReg(4, &cs.d)
	case 0xe3: // set 4, e
		cs.bSetOpReg(4, &cs.e)
	case 0xe4: // set 4, h
		cs.bSetOpReg(4, &cs.h)
	case 0xe5: // set 4, l
		cs.bSetOpReg(4, &cs.l)
	case 0xe6: // set 4, (hl)
		cs.bSetOpHL(4)
	case 0xe7: // set 4, a
		cs.bSetOpReg(4, &cs.a)

	case 0xe8: // set 5, b
		cs.bSetOpReg(5, &cs.b)
	case 0xe9: // set 5, c
		cs.bSetOpReg(5, &cs.c)
	case 0xea: // set 5, d
		cs.bSetOpReg(5, &cs.d)
	case 0xeb: // set 5, e
		cs.bSetOpReg(5, &cs.e)
	case 0xec: // set 5, h
		cs.bSetOpReg(5, &cs.h)
	case 0xed: // set 5, l
		cs.bSetOpReg(5, &cs.l)
	case 0xee: // set 5, (hl)
		cs.bSetOpHL(5)
	case 0xef: // set 5, a
		cs.bSetOpReg(5, &cs.a)

	case 0xf0: // set 6, b
		cs.bSetOpReg(6, &cs.b)
	case 0xf1: // set 6, c
		cs.bSetOpReg(6, &cs.c)
	case 0xf2: // set 6, d
		cs.bSetOpReg(6, &cs.d)
	case 0xf3: // set 6, e
		cs.bSetOpReg(6, &cs.e)
	case 0xf4: // set 6, h
		cs.bSetOpReg(6, &cs.h)
	case 0xf5: // set 6, l
		cs.bSetOpReg(6, &cs.l)
	case 0xf6: // set 6, (hl)
		cs.bSetOpHL(6)
	case 0xf7: // set 6, a
		cs.bSetOpReg(6, &cs.a)

	case 0xf8: // set 7, b
		cs.bSetOpReg(7, &cs.b)
	case 0xf9: // set 7, c
		cs.bSetOpReg(7, &cs.c)
	case 0xfa: // set 7, d
		cs.bSetOpReg(7, &cs.d)
	case 0xfb: // set 7, e
		cs.bSetOpReg(7, &cs.e)
	case 0xfc: // set 7, h
		cs.bSetOpReg(7, &cs.h)
	case 0xfd: // set 7, l
		cs.bSetOpReg(7, &cs.l)
	case 0xfe: // set 7, (hl)
		cs.bSetOpHL(7)
	case 0xff: // set 7, a
		cs.bSetOpReg(7, &cs.a)

	default:
		cs.stepErr(fmt.Sprintf("Unknown Extended Opcode: 0x%02x\r\n", extOpcode))
	}
}

func (cs *cpuState) swapOpReg(reg *byte) {
	val := *reg
	result := val>>4 | (val&0x0f)<<4
	cs.setOp8(8, 2, reg, result, zFlag(result))
}
func (cs *cpuState) swapOpHL() {
	val := cs.followHL()
	result := val>>4 | (val&0x0f)<<4
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result))
}

func (cs *cpuState) rlaOp() {
	val := cs.a
	result := (val << 1) | ((cs.f >> 4) & 0x01)
	carry := val >> 7
	cs.setOpA(4, 1, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rlOpReg(reg *byte) {
	val := *reg
	result := (val << 1) | ((cs.f >> 4) & 0x01)
	carry := val >> 7
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rlOpHL() {
	val := cs.followHL()
	result := (val << 1) | ((cs.f >> 4) & 0x01)
	carry := val >> 7
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) rraOp() {
	val := cs.a
	result := ((cs.f << 3) & 0x80) | (val >> 1)
	carry := val & 0x01
	cs.setOpA(4, 1, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rrOpReg(reg *byte) {
	val := *reg
	result := ((cs.f << 3) & 0x80) | (val >> 1)
	carry := val & 0x01
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rrOpHL() {
	val := cs.followHL()
	result := ((cs.f << 3) & 0x80) | (val >> 1)
	carry := val & 0x01
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) rlcaOp() {
	val := cs.a
	result := (val << 1) | (val >> 7)
	carry := val >> 7
	cs.setOpA(4, 1, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rlcOpReg(reg *byte) {
	val := *reg
	result := (val << 1) | (val >> 7)
	carry := val >> 7
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rlcOpHL() {
	val := cs.followHL()
	result := (val << 1) | (val >> 7)
	carry := val >> 7
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) rrcaOp() {
	val := cs.a
	result := (val << 7) | (val >> 1)
	carry := val & 0x01
	cs.setOpA(4, 1, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rrcOpReg(reg *byte) {
	val := *reg
	result := (val << 7) | (val >> 1)
	carry := val & 0x01
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) rrcOpHL() {
	val := cs.followHL()
	result := (val << 7) | (val >> 1)
	carry := val & 0x01
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) srlOpReg(reg *byte) {
	val := *reg
	result, carry := val>>1, val&0x01
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) srlOpHL() {
	val := cs.followHL()
	result, carry := val>>1, val&0x01
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) slaOpReg(reg *byte) {
	val := *reg
	result, carry := val<<1, val>>7
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) slaOpHL() {
	val := cs.followHL()
	result, carry := val<<1, val>>7
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) sraOpReg(reg *byte) {
	val := *reg
	result, carry := (val&0x80)|(val>>1), val&0x01
	cs.setOp8(8, 2, reg, result, zFlag(result)|uint16(carry))
}
func (cs *cpuState) sraOpHL() {
	val := cs.followHL()
	result, carry := (val&0x80)|(val>>1), val&0x01
	cs.setOpMem8(16, 2, cs.getHL(), result, zFlag(result)|uint16(carry))
}

func (cs *cpuState) bitOp(cycles uint, instLen uint16, bitNum uint8, val uint8) {
	cs.setOpFn(cycles, instLen, func() {}, zFlag(val&(1<<bitNum))|0x012)
}
func (cs *cpuState) bitOpReg(bitNum, val uint8) {
	cs.bitOp(8, 2, bitNum, val)
}
func (cs *cpuState) bitOpHL(bitNum uint8) {
	cs.bitOp(16, 2, bitNum, cs.followHL())
}

func (cs *cpuState) resOpReg(bitNum uint8, reg *byte) {
	val := *reg
	result := val &^ (1 << bitNum)
	cs.setOp8(8, 2, reg, result, 0x2222)
}
func (cs *cpuState) resOpHL(bitNum uint8) {
	val := cs.followHL()
	result := val &^ (1 << bitNum)
	cs.setOpMem8(16, 2, cs.getHL(), result, 0x2222)
}

func (cs *cpuState) bSetOpReg(bitNum uint8, reg *byte) {
	val := *reg
	result := val | (1 << bitNum)
	cs.setOp8(8, 2, reg, result, 0x2222)
}
func (cs *cpuState) bSetOpHL(bitNum uint8) {
	val := cs.followHL()
	result := val | (1 << bitNum)
	cs.setOpMem8(16, 2, cs.getHL(), result, 0x2222)
}

func (cs *cpuState) stepErr(msg string) {
	fmt.Println(msg)
	fmt.Println(cs.debugStatusLine())
	panic("stepErr()")
}
