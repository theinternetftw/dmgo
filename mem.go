package dmgo

import "fmt"

type mem struct {
	cart            []byte
	internalRAM     [0x8000]byte // go ahead and do CGB size
	highInternalRAM [0x7f]byte   // go ahead and do CGB size
	videoRAM        [0x4000]byte // go ahead and do CGB size
	cartRAM         []byte
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

	case addr < 0x4000:
		val = cs.mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		// TODO: bank switching
		val = cs.mem.cart[addr]
	case addr >= 0x8000 && addr < 0xa000:
		val = cs.mem.videoRAM[addr-0x8000]
	case addr >= 0xc000 && addr < 0xfe00:
		ramAddr := (addr - 0xc000) & 0x1fff // 8kb with wraparound
		val = cs.mem.internalRAM[ramAddr]

	case addr == 0xff00:
		val = cs.joypad.readJoypadReg()
	case addr == 0xff02:
		val = cs.readSerialControlReg()

	case addr == 0xff04:
		val = byte(cs.timerDivCycles >> 8)
	case addr == 0xff05:
		val = cs.timerCounterReg
	case addr == 0xff06:
		val = cs.timerModuloReg
	case addr == 0xff07:
		val = cs.readTimerControlReg()

	case addr == 0xff0f:
		val = cs.readInterruptFlagReg()

	case addr >= 0xff30 && addr < 0xff40:
		val = cs.apu.sounds[2].wavePatternRAM[addr-0xff30]

	case addr == 0xff40:
		val = cs.lcd.readControlReg()
	case addr == 0xff41:
		val = cs.lcd.readStatusReg()
	case addr == 0xff44:
		val = cs.lcd.lyReg
	case addr == 0xff45:
		val = cs.lcd.lycReg

	case addr >= 0xff80 && addr < 0xffff:
		val = cs.mem.highInternalRAM[addr-0xff80]
	case addr == 0xffff:
		val = cs.readInterruptEnableReg()

	default:
		panic(fmt.Sprintf("not implemented: read at %x\n", addr))
	}
	//	fmt.Printf("\treading 0x%02x from 0x%04x\n", val, addr)
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
		// cart ROM, looks like writing to read-only is a nop?
	case addr >= 0x8000 && addr < 0xa000:
		cs.mem.videoRAM[addr-0x8000] = val
	case addr >= 0xa000 && addr < 0xc000:
		if len(cs.mem.cartRAM) == 0 {
			break // nop
		}
		fatalErr(fmt.Sprintf("real cartRAM not yet implemented: write(0x%04x, %v)\n", addr, val))
	case addr >= 0xc000 && addr < 0xfe00:
		cs.mem.internalRAM[((addr - 0xc000) & 0x1fff)] = val // 8kb with wraparound
	case addr >= 0xfe00 && addr < 0xfea0:
		cs.lcd.writeOAM(addr-0xfe00, val)
	case addr >= 0xfea0 && addr < 0xff00:
		// empty, nop (can be more complicated, see TCAGBD)

	case addr == 0xff00:
		cs.joypad.writeJoypadReg(val)
	case addr == 0xff01:
		cs.serialTransferData = val
	case addr == 0xff02:
		cs.writeSerialControlReg(val)

	case addr == 0xff04:
		cs.timerDivCycles = 0
	case addr == 0xff05:
		cs.timerCounterReg = val
	case addr == 0xff06:
		cs.timerModuloReg = val
	case addr == 0xff07:
		cs.writeTimerControlReg(val)

	case addr == 0xff10:
		cs.apu.sounds[0].writeSweepReg(val)
	case addr == 0xff11:
		cs.apu.sounds[0].writeLenDutyReg(val)
	case addr == 0xff12:
		cs.apu.sounds[0].writeSoundEnvReg(val)
	case addr == 0xff13:
		cs.apu.sounds[0].writeFreqLowReg(val)
	case addr == 0xff14:
		cs.apu.sounds[0].writeFreqHighReg(val)

	case addr == 0xff16:
		cs.apu.sounds[1].writeLenDutyReg(val)
	case addr == 0xff17:
		cs.apu.sounds[1].writeSoundEnvReg(val)
	case addr == 0xff18:
		cs.apu.sounds[1].writeFreqLowReg(val)
	case addr == 0xff19:
		cs.apu.sounds[1].writeFreqHighReg(val)

	case addr == 0xff1a:
		cs.apu.sounds[2].on = val&0x80 != 0
	case addr == 0xff1b:
		cs.apu.sounds[2].lengthData = val
	case addr == 0xff1c:
		cs.apu.sounds[2].writeWaveOutLvlReg(val)
	case addr == 0xff1d:
		cs.apu.sounds[2].writeFreqLowReg(val)
	case addr == 0xff1e:
		cs.apu.sounds[2].writeFreqHighReg(val)

	case addr == 0xff20:
		cs.apu.sounds[3].lengthData = val & 0x1f
	case addr == 0xff21:
		cs.apu.sounds[3].writeSoundEnvReg(val)
	case addr == 0xff22:
		cs.apu.sounds[3].writePolyCounterReg(val)
	case addr == 0xff23:
		cs.apu.sounds[3].writeFreqHighReg(val) // noise channel uses control bits, freq ignored

	case addr == 0xff24:
		cs.apu.writeVolumeReg(val)
	case addr == 0xff25:
		cs.apu.writeSpeakerSelectReg(val)
	case addr == 0xff26:
		cs.apu.writeSoundOnOffReg(val)
	case addr >= 0xff30 && addr < 0xff40:
		cs.apu.sounds[2].wavePatternRAM[addr-0xff30] = val

	case addr == 0xff40:
		cs.lcd.writeControlReg(val)
	case addr == 0xff41:
		cs.lcd.writeStatusReg(val)
	case addr == 0xff42:
		cs.lcd.scrollY = val
	case addr == 0xff43:
		cs.lcd.scrollX = val
	case addr == 0xff45:
		cs.lcd.lycReg = val
	case addr == 0xff46:
		cs.oamDMA(uint16(val) << 8)
	case addr == 0xff47:
		cs.lcd.backgroundPaletteReg = val
	case addr == 0xff48:
		cs.lcd.objectPalette0Reg = val
	case addr == 0xff49:
		cs.lcd.objectPalette1Reg = val
	case addr == 0xff4a:
		cs.lcd.windowY = val
	case addr == 0xff4b:
		cs.lcd.windowX = val

	case addr == 0xff0f:
		cs.writeInterruptFlagReg(val)
	case addr >= 0xff4c && addr < 0xff80:
		// empty, nop (can be more complicated, see TCAGBD)
	case addr >= 0xff80 && addr < 0xffff:
		cs.mem.highInternalRAM[addr-0xff80] = val
	case addr == 0xffff:
		cs.writeInterruptEnableReg(val)
	default:
		fatalErr(fmt.Sprintf("not implemented: write(0x%04x, %v)\n", addr, val))
	}
	//	fmt.Printf("\twriting 0x%02x to 0x%04x\n", val, addr)
}

func (cs *cpuState) write16(addr uint16, val uint16) {
	cs.write(addr, byte(val))
	cs.write(addr+1, byte(val>>8))
}
