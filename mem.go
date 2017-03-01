package dmgo

import "fmt"

type mem struct {
	// not marshalled in snapshot
	cart []byte

	// everything else marshalled

	CartRAM               []byte
	InternalRAM           []byte
	InternalRAMBankNumber uint16
	HighInternalRAM       [0x7f]byte
	mbc                   mbc
}

func (mem *mem) mbcRead(addr uint16) byte {
	return mem.mbc.Read(mem, addr)
}
func (mem *mem) mbcWrite(addr uint16, val byte) {
	mem.mbc.Write(mem, addr, val)
}

// TODO / FIXME: DMA timing enforcement!
// also, dma cancel / restart (if you
// run a 2nd dma, it cancels the running one
// and starts the new one immediately)
func (cs *cpuState) oamDMA(addr uint16) {
	for i := uint16(0); i < 0xa0; i++ {
		cs.write(0xfe00+i, cs.read(addr+i))
	}
}

func (cs *cpuState) read(addr uint16) byte {
	var val byte
	switch {

	case addr < 0x8000:
		val = cs.Mem.mbcRead(addr)

	case addr >= 0x8000 && addr < 0xa000:
		val = cs.LCD.readVideoRAM(addr - 0x8000)

	case addr >= 0xa000 && addr < 0xc000:
		val = cs.Mem.mbcRead(addr)

	case addr >= 0xc000 && addr < 0xfe00:
		ramAddr := (addr - 0xc000) & 0x1fff // 8kb with wraparound
		if ramAddr >= 0x1000 {
			ramAddr = (ramAddr - 0x1000) + 0x1000*cs.Mem.InternalRAMBankNumber
		}
		val = cs.Mem.InternalRAM[ramAddr]

	case addr >= 0xfe00 && addr < 0xfea0:
		val = cs.LCD.readOAM(addr - 0xfe00)

	case addr >= 0xfea0 && addr < 0xff00:
		val = 0xff // (empty mem, but can be more complicated, see TCAGBD)

	case addr == 0xff00:
		val = cs.Joypad.readJoypadReg()
	case addr == 0xff01:
		val = cs.SerialTransferData
	case addr == 0xff02:
		val = cs.readSerialControlReg()

	case addr == 0xff03:
		val = 0xff // unmapped bytes

	case addr == 0xff04:
		val = byte(cs.TimerDivCycles >> 8)
	case addr == 0xff05:
		val = cs.TimerCounterReg
	case addr == 0xff06:
		val = cs.TimerModuloReg
	case addr == 0xff07:
		val = cs.readTimerControlReg()

	case addr >= 0xff08 && addr < 0xff0f:
		val = 0xff // unmapped bytes

	case addr == 0xff0f:
		val = cs.readInterruptFlagReg()

	case addr == 0xff10:
		val = cs.APU.Sounds[0].readSweepReg()
	case addr == 0xff11:
		val = cs.APU.Sounds[0].readLenDutyReg()
	case addr == 0xff12:
		val = cs.APU.Sounds[0].readSoundEnvReg()
	case addr == 0xff13:
		val = cs.APU.Sounds[0].readFreqLowReg()
	case addr == 0xff14:
		val = cs.APU.Sounds[0].readFreqHighReg()

	case addr == 0xff15:
		val = 0xff // unmapped bytes
	case addr == 0xff16:
		val = cs.APU.Sounds[1].readLenDutyReg()
	case addr == 0xff17:
		val = cs.APU.Sounds[1].readSoundEnvReg()
	case addr == 0xff18:
		val = cs.APU.Sounds[1].readFreqLowReg()
	case addr == 0xff19:
		val = cs.APU.Sounds[1].readFreqHighReg()

	case addr == 0xff1a:
		val = boolBit(cs.APU.Sounds[2].On, 7) | 0x7f
	case addr == 0xff1b:
		val = cs.APU.Sounds[2].readLengthDataReg()
	case addr == 0xff1c:
		val = cs.APU.Sounds[2].readWaveOutLvlReg()
	case addr == 0xff1d:
		val = cs.APU.Sounds[2].readFreqLowReg()
	case addr == 0xff1e:
		val = cs.APU.Sounds[2].readFreqHighReg()

	case addr == 0xff1f:
		val = 0xff // unmapped bytes
	case addr == 0xff20:
		val = cs.APU.Sounds[3].readLengthDataReg()
	case addr == 0xff21:
		val = cs.APU.Sounds[3].readSoundEnvReg()
	case addr == 0xff22:
		val = cs.APU.Sounds[3].readPolyCounterReg()
	case addr == 0xff23:
		val = cs.APU.Sounds[3].readFreqHighReg()

	case addr == 0xff24:
		val = cs.APU.readVolumeReg()
	case addr == 0xff25:
		val = cs.APU.readSpeakerSelectReg()
	case addr == 0xff26:
		val = cs.APU.readSoundOnOffReg()

	case addr >= 0xff27 && addr < 0xff30:
		val = 0xff // unmapped bytes

	case addr >= 0xff30 && addr < 0xff40:
		val = cs.APU.Sounds[2].WavePatternRAM[addr-0xff30]

	case addr == 0xff40:
		val = cs.LCD.readControlReg()
	case addr == 0xff41:
		val = cs.LCD.readStatusReg()
	case addr == 0xff42:
		val = cs.LCD.ScrollY
	case addr == 0xff43:
		val = cs.LCD.ScrollX
	case addr == 0xff44:
		val = cs.LCD.LYReg
	case addr == 0xff45:
		val = cs.LCD.LYCReg

	case addr == 0xff46:
		val = 0xff // oam DMA reg, write-only

	case addr == 0xff47:
		val = cs.LCD.BackgroundPaletteReg
	case addr == 0xff48:
		val = cs.LCD.ObjectPalette0Reg
	case addr == 0xff49:
		val = cs.LCD.ObjectPalette1Reg
	case addr == 0xff4a:
		val = cs.LCD.WindowY
	case addr == 0xff4b:
		val = cs.LCD.WindowX

	case addr == 0xff4c:
		val = 0xff // unmapped bytes

	case addr == 0xff4d:
		if cs.CGBMode {
			val = cs.readSpeedSwitchReg()
		}

	case addr >= 0xff4e && addr < 0xff51:
		val = 0xff // unmapped bytes

	case addr == 0xff51:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff52:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff53:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff54:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff55:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff56:
		if cs.CGBMode {
			val = cs.readIRPortReg()
		}

	case addr >= 0xff57 && addr < 0xff68:
		val = 0xff // unmapped bytes

	case addr == 0xff68:
		if cs.CGBMode {
			cs.stepErr("CGB bg pal idx: not implemented")
		}
	case addr == 0xff69:
		if cs.CGBMode {
			cs.stepErr("CGB bg pal data: not implemented")
		}
	case addr == 0xff6a:
		if cs.CGBMode {
			cs.stepErr("CGB sprite pal idx: not implemented")
		}
	case addr == 0xff6b:
		if cs.CGBMode {
			cs.stepErr("CGB sprite pal data: not implemented")
		}

	case addr >= 0xff6c && addr < 0xff70:
		val = 0xff // unmapped bytes

	case addr == 0xff70:
		if cs.CGBMode {
			val = byte(cs.Mem.InternalRAMBankNumber)
		}

	case addr >= 0xff71 && addr < 0xff80:
		val = 0xff // unmapped bytes

	case addr >= 0xff80 && addr < 0xffff:
		val = cs.Mem.HighInternalRAM[addr-0xff80]
	case addr == 0xffff:
		val = cs.readInterruptEnableReg()

	default:
		cs.stepErr(fmt.Sprintf("not implemented: read at %x\n", addr))
	}
	return val
}

func (cs *cpuState) read16(addr uint16) uint16 {
	high := uint16(cs.read(addr + 1))
	low := uint16(cs.read(addr))
	return (high << 8) | low
}

func (cs *cpuState) write(addr uint16, val byte) {
	switch {

	case addr < 0x8000:
		cs.Mem.mbcWrite(addr, val)

	case addr >= 0x8000 && addr < 0xa000:
		cs.LCD.writeVideoRAM(addr-0x8000, val)

	case addr >= 0xa000 && addr < 0xc000:
		cs.Mem.mbcWrite(addr, val)

	case addr >= 0xc000 && addr < 0xfe00:
		ramAddr := (addr - 0xc000) & 0x1fff // 8kb with wraparound
		if ramAddr >= 0x1000 {
			ramAddr = (ramAddr - 0x1000) + 0x1000*cs.Mem.InternalRAMBankNumber
		}
		cs.Mem.InternalRAM[ramAddr] = val
	case addr >= 0xfe00 && addr < 0xfea0:
		cs.LCD.writeOAM(addr-0xfe00, val)
	case addr >= 0xfea0 && addr < 0xff00:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr == 0xff00:
		cs.Joypad.writeJoypadReg(val)
	case addr == 0xff01:
		cs.SerialTransferData = val
	case addr == 0xff02:
		cs.writeSerialControlReg(val)

	case addr == 0xff03:
		// nop (unmapped bytes)

	case addr == 0xff04:
		cs.TimerDivCycles = 0
	case addr == 0xff05:
		cs.TimerCounterReg = val
	case addr == 0xff06:
		cs.TimerModuloReg = val
	case addr == 0xff07:
		cs.writeTimerControlReg(val)

	case addr >= 0xff08 && addr < 0xff0f:
		// nop (unmapped bytes)

	case addr == 0xff0f:
		cs.writeInterruptFlagReg(val)

	case addr == 0xff10:
		cs.APU.Sounds[0].writeSweepReg(val)
	case addr == 0xff11:
		cs.APU.Sounds[0].writeLenDutyReg(val)
	case addr == 0xff12:
		cs.APU.Sounds[0].writeSoundEnvReg(val)
	case addr == 0xff13:
		cs.APU.Sounds[0].writeFreqLowReg(val)
	case addr == 0xff14:
		cs.APU.Sounds[0].writeFreqHighReg(val)

	case addr == 0xff15:
		// nop (unmapped bytes)

	case addr == 0xff16:
		cs.APU.Sounds[1].writeLenDutyReg(val)
	case addr == 0xff17:
		cs.APU.Sounds[1].writeSoundEnvReg(val)
	case addr == 0xff18:
		cs.APU.Sounds[1].writeFreqLowReg(val)
	case addr == 0xff19:
		cs.APU.Sounds[1].writeFreqHighReg(val)

	case addr == 0xff1a:
		cs.APU.Sounds[2].writeWaveOnOffReg(val)
	case addr == 0xff1b:
		cs.APU.Sounds[2].writeLengthDataReg(val)
	case addr == 0xff1c:
		cs.APU.Sounds[2].writeWaveOutLvlReg(val)
	case addr == 0xff1d:
		cs.APU.Sounds[2].writeFreqLowReg(val)
	case addr == 0xff1e:
		cs.APU.Sounds[2].writeFreqHighReg(val)

	case addr == 0xff1f:
		// nop (unmapped bytes)

	case addr == 0xff20:
		cs.APU.Sounds[3].writeLengthDataReg(val)
	case addr == 0xff21:
		cs.APU.Sounds[3].writeSoundEnvReg(val)
	case addr == 0xff22:
		cs.APU.Sounds[3].writePolyCounterReg(val)
	case addr == 0xff23:
		cs.APU.Sounds[3].writeFreqHighReg(val) // noise channel uses control bits, freq ignored

	case addr == 0xff24:
		cs.APU.writeVolumeReg(val)
	case addr == 0xff25:
		cs.APU.writeSpeakerSelectReg(val)
	case addr == 0xff26:
		cs.APU.writeSoundOnOffReg(val)

	case addr >= 0xff27 && addr < 0xff30:
		// nop (unmapped bytes)

	case addr >= 0xff30 && addr < 0xff40:
		cs.APU.Sounds[2].writeWavePatternValue(addr-0xff30, val)

	case addr == 0xff40:
		cs.LCD.writeControlReg(val)
	case addr == 0xff41:
		cs.LCD.writeStatusReg(val)
	case addr == 0xff42:
		cs.LCD.writeScrollY(val)
	case addr == 0xff43:
		cs.LCD.writeScrollX(val)
	case addr == 0xff44:
		// nop? pandocs says something
		// about "resetting the counter",
		// bgb seems to do nothing. doesn't
		// reset LYReg, doesn't change any
		// counter I see...
	case addr == 0xff45:
		cs.LCD.writeLycReg(val)
	case addr == 0xff46:
		cs.oamDMA(uint16(val) << 8)
	case addr == 0xff47:
		cs.LCD.writeBackgroundPaletteReg(val)
	case addr == 0xff48:
		cs.LCD.writeObjectPalette0Reg(val)
	case addr == 0xff49:
		cs.LCD.writeObjectPalette1Reg(val)
	case addr == 0xff4a:
		cs.LCD.writeWindowY(val)
	case addr == 0xff4b:
		cs.LCD.writeWindowX(val)

	case addr == 0xff4c:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr == 0xff4d:
		if cs.CGBMode {
			cs.writeSpeedSwitchReg(val)
		}

	case addr >= 0xff4e && addr < 0xff51:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr == 0xff51:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff52:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff53:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff54:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff55:
		if cs.CGBMode {
			cs.stepErr("CGB DMA: not implemented")
		}
	case addr == 0xff56:
		if cs.CGBMode {
			cs.writeIRPortReg(val)
		}

	case addr >= 0xff57 && addr < 0xff68:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr == 0xff68:
		if cs.CGBMode {
			cs.stepErr("CGB bg pal idx: not implemented")
		}
	case addr == 0xff69:
		if cs.CGBMode {
			cs.stepErr("CGB bg pal data: not implemented")
		}
	case addr == 0xff6a:
		if cs.CGBMode {
			cs.stepErr("CGB sprite pal idx: not implemented")
		}
	case addr == 0xff6b:
		if cs.CGBMode {
			cs.stepErr("CGB sprite pal data: not implemented")
		}

	case addr >= 0xff6c && addr < 0xff70:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr == 0xff70:
		if cs.CGBMode {
			if val&0x07 == 0 {
				val = 1
			}
			cs.Mem.InternalRAMBankNumber = uint16(val) & 0x07
		}

	case addr >= 0xff71 && addr < 0xff80:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr >= 0xff80 && addr < 0xffff:
		cs.Mem.HighInternalRAM[addr-0xff80] = val
	case addr == 0xffff:
		cs.writeInterruptEnableReg(val)
	default:
		cs.stepErr(fmt.Sprintf("not implemented: write(0x%04x, %v)\n", addr, val))
	}
}

func (cs *cpuState) write16(addr uint16, val uint16) {
	cs.write(addr, byte(val))
	cs.write(addr+1, byte(val>>8))
}
