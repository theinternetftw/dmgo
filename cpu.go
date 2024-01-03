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
			cs.runCycles(8)
			cs.pushOp16(cs.PC)
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

	zero := flags & 0xf000
	sub := flags & 0xf00
	halfCarry := flags & 0xf0
	carry := flags & 0xf

	if zero == 0x1000 {
		cs.F |= 0x80
	} else if zero == 0x0000 {
		cs.F &^= 0x80
	}
	if sub == 0x100 {
		cs.F |= 0x40
	} else if sub == 0x000 {
		cs.F &^= 0x40
	}
	if halfCarry == 0x10 {
		cs.F |= 0x20
	} else if halfCarry == 0x00 {
		cs.F &^= 0x20
	}
	if carry == 1 {
		cs.F |= 0x10
	} else if carry == 0 {
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
