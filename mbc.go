package dmgo

import (
	"fmt"
	"time"
)

// TODO: differentiate between those with
// batteries (can save) vs those without
// so we don't leak .sav files for those
// who don't need em. What about flash
// mem that doesn't need battery? That
// used ever?
// NOTE: bgb warns when carts have RAM
// but list a cartType that ostensibly
// doesn't. Real gameboys don't care,
// do we?
func makeMBC(cartInfo *CartInfo) mbc {
	switch cartInfo.CartridgeType {
	case 0:
		return &nullMBC{}
	case 1, 2, 3:
		return &mbc1{}
	case 5, 6:
		return &mbc2{}
	case 8, 9:
		return &nullMBC{} // but this time with RAM
	case 11, 12, 13:
		panic("MMM01 mapper requested. Not implemented!")
	case 15, 16, 17, 18, 19:
		return &mbc3{}
	case 25, 26, 27, 28, 29, 30:
		return &mbc5{}
	default:
		panic(fmt.Sprintf("makeMBC: unknown cart type %v", cartInfo.CartridgeType))
	}
}

type mbc interface {
	Init(mem *mem)
	// Read reads via the MBC
	Read(mem *mem, addr uint16) byte
	// Write writes via the MBC
	Write(mem *mem, addr uint16, val byte)

	// Gets the ROM map number (for debug)
	GetROMBankNumber() int
}

type nullMBC struct{}

func (mbc *nullMBC) GetROMBankNumber() int {
	return 1
}
func (mbc *nullMBC) Init(mem *mem) {}
func (mbc *nullMBC) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x8000:
		return mem.cart[addr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		if int(localAddr) < len(mem.cartRAM) {
			return mem.cartRAM[localAddr]
		}
		return 0xff
	default:
		panic(fmt.Sprintf("nullMBC: not implemented: read at %x\n", addr))
	}
}
func (mbc *nullMBC) Write(mem *mem, addr uint16, val byte) {
	localAddr := uint(addr - 0xa000)
	if int(localAddr) < len(mem.cartRAM) {
		mem.cartRAM[localAddr] = val
	}
}

const (
	bankingModeRAM = iota
	bankingModeROM
)

type mbc1 struct {
	romBankNumber byte
	ramBankNumber byte
	ramEnabled    bool
	bankingMode   int
}

func (mbc *mbc1) Init(mem *mem) {
	mbc.romBankNumber = 1 // can't go lower
}

func (mbc *mbc1) GetROMBankNumber() int {
	return int(mbc.romBankNumber)
}
func (mbc *mbc1) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.romBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc1: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.romBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.ramBankOffset()
		if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
			return mem.cartRAM[localAddr]
		}
		return 0xff
	default:
		panic(fmt.Sprintf("mbc1: not implemented: read at %x\n", addr))
	}
}

func (mbc *mbc1) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		mbc.ramEnabled = val&0x0f == 0x0a
	case addr >= 0x2000 && addr < 0x4000:
		bankNum := val & 0x1f
		if bankNum == 0 {
			// no bank 0 selection. note that this also (when
			// using many banks and the 2nd reg) disallows any
			// bank with 0 for the bottom 5 bits, i.e. no 0x20,
			// 0x40, or 0x60 banks, trying to do so will select
			// 0x21, 0x41, or 0x61. Thus a max of 125 banks (128-3),
			// for MBC1
			bankNum = 1
		}
		mbc.romBankNumber = (mbc.romBankNumber &^ 0x1f) | bankNum
	case addr >= 0x4000 && addr < 0x6000:
		valBits := val & 0x03
		if mbc.bankingMode == bankingModeRAM {
			mbc.ramBankNumber = valBits
		} else { // ROM mode
			mbc.romBankNumber = (mbc.romBankNumber & 0x1f) | (valBits << 5)
		}
	case addr >= 0x6000 && addr < 0x8000:
		// NOTE: do those two bits from the RAM number need to
		// be passed over to ROM after banking mode changes?
		// (and vice versa?)
		if (val&0x01) > 0 && mbc.bankingMode != bankingModeRAM {
			mbc.bankingMode = bankingModeRAM
			//mbc.ramBankNumber = (mbc.romBankNumber >> 5) & 0x03
			mbc.romBankNumber = mbc.romBankNumber & 0x1f
		} else {
			mbc.bankingMode = bankingModeROM
			//mbc.romBankNumber = (mbc.romBankNumber & 0x1f) | (mbc.ramBankNumber << 5)
			mbc.ramBankNumber = 0
		}
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.ramBankOffset()
		if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
			mem.cartRAM[localAddr] = val
		}
	default:
		panic(fmt.Sprintf("mbc1: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc1) romBankOffset() uint {
	return uint(mbc.romBankNumber) * 0x4000
}
func (mbc *mbc1) ramBankOffset() uint {
	return uint(mbc.ramBankNumber) * 0x2000
}

type mbc2 struct {
	romBankNumber byte
	ramEnabled    bool
}

func (mbc *mbc2) Init(mem *mem) {
	mbc.romBankNumber = 1 // can't go lower
}

func (mbc *mbc2) GetROMBankNumber() int {
	return int(mbc.romBankNumber)
}
func (mbc *mbc2) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.romBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc2: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.romBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
			// 4-bit ram (NOTE: pull high nibble down or up?)
			return mem.cartRAM[localAddr] & 0x0f
		}
		return 0xff
	default:
		panic(fmt.Sprintf("mbc2: not implemented: read at %x\n", addr))
	}
}

func (mbc *mbc2) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		if addr&0x0100 > 0 {
			// nop, this bit must be zero
		} else {
			mbc.ramEnabled = val&0x0f == 0x0a
		}
	case addr >= 0x2000 && addr < 0x4000:
		if addr&0x0100 == 0 {
			// nop, this bit must be one
		} else {
			// 16 rom banks
			bankNum := val & 0x0f
			if bankNum == 0 {
				// no bank 0 selection.
				bankNum = 1
			}
			mbc.romBankNumber = bankNum
		}
	case addr >= 0x4000 && addr < 0x8000:
		// nop
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
			// 4-bit RAM
			mem.cartRAM[localAddr] = val & 0x0f
		}
	default:
		panic(fmt.Sprintf("mbc2: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc2) romBankOffset() uint {
	return uint(mbc.romBankNumber) * 0x4000
}

type mbc3 struct {
	romBankNumber byte
	ramBankNumber byte
	ramEnabled    bool

	seconds byte
	minutes byte
	hours   byte
	days    uint16

	latchedSeconds byte
	latchedMinutes byte
	latchedHours   byte
	latchedDays    uint16
	dayCarry       bool

	timerStopped bool
	timerLatched bool

	timeAtLastSet time.Time
}

func (mbc *mbc3) Init(mem *mem) {
	mbc.romBankNumber = 1 // can't go lower

	// NOTE: could load last time, vals, and carry bit from save or something here, if we want
	mbc.timeAtLastSet = time.Now()
}

func (mbc *mbc3) updateTimer() {
	if mbc.timerStopped {
		return
	}
	oldUnix := time.Unix(
		int64(mbc.seconds)+
			int64(mbc.minutes)*60+
			int64(mbc.hours)*60*60+
			int64(mbc.days)*60*60*24, 0)

	ticked := time.Now().Sub(mbc.timeAtLastSet)
	mbc.timeAtLastSet = time.Now()

	newTotalSeconds := oldUnix.Add(ticked).Unix()
	mbc.seconds = byte(newTotalSeconds % 60)

	newTotalMinutes := (newTotalSeconds - int64(mbc.seconds)/60)
	mbc.minutes = byte(newTotalMinutes % 60)

	newTotalHours := (newTotalMinutes - int64(mbc.minutes)/60)
	mbc.hours = byte(newTotalHours % 24)

	newTotalDays := (newTotalHours - int64(mbc.hours)/24)
	mbc.days = uint16(newTotalDays)

	if newTotalDays > 511 {
		mbc.dayCarry = true
	}
}

func (mbc *mbc3) updateLatch() {
	mbc.updateTimer()
	mbc.latchedSeconds = mbc.seconds
	mbc.latchedMinutes = mbc.minutes
	mbc.latchedHours = mbc.hours
	mbc.latchedDays = mbc.days
}

func (mbc *mbc3) GetROMBankNumber() int {
	return int(mbc.romBankNumber)
}
func (mbc *mbc3) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.romBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc3: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.romBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		switch mbc.ramBankNumber {
		case 0, 1, 2, 3:
			localAddr := uint(addr-0xa000) + mbc.ramBankOffset()
			if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
				return mem.cartRAM[localAddr]
			}
			return 0xff
		case 8:
			return mbc.latchedSeconds
		case 9:
			return mbc.latchedMinutes
		case 10:
			return mbc.latchedHours
		case 11:
			return byte(mbc.latchedDays)
		case 12:
			return boolBit(mbc.dayCarry, 7) | boolBit(mbc.timerStopped, 6) | (byte(mbc.latchedDays>>7) & 0x01)
		}
		// might need a default of return 0xff here
	}
	panic(fmt.Sprintf("mbc3: not implemented: read at %x\n", addr))
}

func (mbc *mbc3) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		mbc.ramEnabled = val&0x0f == 0x0a
	case addr >= 0x2000 && addr < 0x4000:
		bankNum := val &^ 0x80 // 7bit selector
		if bankNum == 0 {
			// no bank 0 selection.
			bankNum = 1
		}
		mbc.romBankNumber = bankNum
	case addr >= 0x4000 && addr < 0x6000:
		switch val {
		case 0, 1, 2, 3:
			mbc.ramBankNumber = val
		case 8, 9, 10, 11, 12:
			mbc.ramBankNumber = val
		default:
			// all others nop
			// NOTE: or should they e.g. select a bank that returns all 0xff's?
		}
	case addr >= 0x6000 && addr < 0x8000:
		if !mbc.timerLatched && val&0x01 > 0 {
			mbc.timerLatched = true
			mbc.updateTimer()
		}
	case addr >= 0xa000 && addr < 0xc000:
		switch mbc.ramBankNumber {
		case 0, 1, 2, 3:
			localAddr := uint(addr-0xa000) + mbc.ramBankOffset()
			if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
				mem.cartRAM[localAddr] = val
			}
		case 8:
			mbc.updateTimer()
			mbc.seconds = val
		case 9:
			mbc.updateTimer()
			mbc.minutes = val
		case 10:
			mbc.updateTimer()
			mbc.hours = val
		case 11:
			mbc.updateTimer()
			mbc.days = uint16(val)
		case 12:
			mbc.updateTimer()
			mbc.days &^= 0x0100
			mbc.days |= uint16(val&0x01) << 8
			mbc.timerStopped = val&(1<<6) > 0
			mbc.dayCarry = val&(1<<7) > 0
		default:
			// nop
		}
	default:
		panic(fmt.Sprintf("mbc3: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc3) romBankOffset() uint {
	return uint(mbc.romBankNumber) * 0x4000
}
func (mbc *mbc3) ramBankOffset() uint {
	return uint(mbc.ramBankNumber) * 0x2000
}

type mbc5 struct {
	romBankNumber uint
	ramBankNumber uint
	ramEnabled    bool
}

func (mbc *mbc5) Init(mem *mem) {
	// NOTE: can do bank 0 now, do we still start with 1?
	mbc.romBankNumber = 1
}

func (mbc *mbc5) GetROMBankNumber() int {
	return int(mbc.romBankNumber)
}
func (mbc *mbc5) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.romBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc5: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.romBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.ramBankOffset()
		if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
			return mem.cartRAM[localAddr]
		}
		return 0xff
	default:
		panic(fmt.Sprintf("mbc5: not implemented: read at %x\n", addr))
	}
}

func (mbc *mbc5) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		mbc.ramEnabled = val&0x0f == 0x0a
	case addr >= 0x2000 && addr < 0x3000:
		mbc.romBankNumber = (mbc.romBankNumber &^ 0xff) | uint(val)
	case addr >= 0x3000 && addr < 0x4000:
		// NOTE: TCAGBD says that games that don't use the 9th bit
		// can use this to set the lower eight! I'll wait until I
		// see a game try to do that before impl'ing
		mbc.romBankNumber = (mbc.romBankNumber &^ 0x100) | (uint(val&0x01) << 8)
	case addr >= 0x4000 && addr < 0x6000:
		mbc.ramBankNumber = uint(val & 0x0f)
	case addr >= 0x6000 && addr < 0x8000:
		// nop?
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.ramBankOffset()
		if mbc.ramEnabled && int(localAddr) < len(mem.cartRAM) {
			mem.cartRAM[localAddr] = val
		}
	default:
		panic(fmt.Sprintf("mbc5: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc5) romBankOffset() uint {
	return mbc.romBankNumber * 0x4000
}
func (mbc *mbc5) ramBankOffset() uint {
	return mbc.ramBankNumber * 0x2000
}
