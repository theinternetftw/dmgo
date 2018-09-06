package dmgo

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
