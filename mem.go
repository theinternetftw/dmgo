package dmgo

import "fmt"

type mem struct {
	cart            []byte
	internalRAM     [0x8000]byte // go ahead and do CGB size
	highInternalRAM [0x7f]byte   // go ahead and do CGB size
	videoRAM        [0x4000]byte // go ahead and do CGB size
	cartRAM         []byte
}

func (cs *cpuState) read(addr uint16) byte {
	switch {
	case addr < 0x3fff:
		return cs.mem.cart[addr]
	case addr >= 0xc000 && addr < 0xfe00:
		ramAddr := (addr - 0xc000) & 0x1fff // 8kb with wraparound
		return cs.mem.internalRAM[ramAddr]
	case addr == 0xff44:
		return cs.lcd.lyReg
	default:
		panic(fmt.Sprintf("not implemented: read at %x\n", addr))
	}
}
func get16LE(slice []byte, address uint16) uint16 {
	high := uint16(slice[address+1])
	low := uint16(slice[address])
	return (high << 8) | low
}
func (cs *cpuState) read16(addr uint16) uint16 {
	if addr < 0x3ffe {
		return get16LE(cs.mem.cart, addr)
	}
	panic(fmt.Sprintf("not implemented: read16() at %x\n", addr))
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
	// case addr >= 0xfe00 && addr < 0xfea0:
	//	// TODO: OAM
	// case addr >= 0xfea0 && addr < 0xff00:
	//	// empty, nop
	case addr == 0xff01:
		cs.serialTransferData = val
	case addr == 0xff02:
		cs.serialTransferStartFlag = val&0x80 > 0
		cs.serialTransferClockIsInternal = val&0x01 > 0
	case addr == 0xff40:
		cs.lcd.writeControlReg(val)
	case addr == 0xff41:
		cs.lcd.writeStatusReg(val)
	case addr == 0xff42:
		cs.lcd.scrollY = val
	case addr == 0xff43:
		cs.lcd.scrollX = val
	case addr == 0xff0f:
		cs.interruptsFlagRegister = val
	// case addr >= 0xff00 && addr < 0xff4c:
	//	// TODO: I/O MAPPED!
	// case addr >= 0xff4c && addr < 0xff80:
	//	// empty, nop
	case addr >= 0xff80 && addr < 0xffff:
		cs.mem.highInternalRAM[addr-0xff80] = val
	case addr == 0xffff:
		cs.interruptsEnableRegister = val
	default:
		fatalErr(fmt.Sprintf("not implemented: write(0x%04x, %v)\n", addr, val))
	}
}
func (cs *cpuState) write16(addr uint16, val uint16) {
	fatalErr(fmt.Sprintf("not implemented: write16(0x%04x, %v)\n", addr, val))
}
