package dmgo

import (
	"encoding/json"
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

	// for debug
	GetROMBankNumber() int
	GetRAMBankNumber() int

	ROMBankOffset() uint
	RAMBankOffset() uint

	Marshal() marshalledMBC
}

type marshalledMBC struct {
	Name string
	Data []byte
}

func unmarshalMBC(m marshalledMBC) (mbc, error) {
	switch m.Name {
	case "nullMBC":
		return &nullMBC{}, nil
	case "mbc1":
		var mbc1 mbc1
		if err := json.Unmarshal(m.Data, &mbc1); err != nil {
			return nil, err
		}
		return &mbc1, nil
	case "mbc2":
		var mbc2 mbc2
		if err := json.Unmarshal(m.Data, &mbc2); err != nil {
			return nil, err
		}
		return &mbc2, nil
	case "mbc3":
		var mbc3 mbc3
		if err := json.Unmarshal(m.Data, &mbc3); err != nil {
			return nil, err
		}
		return &mbc3, nil
	case "mbc5":
		var mbc5 mbc5
		if err := json.Unmarshal(m.Data, &mbc5); err != nil {
			return nil, err
		}
		return &mbc5, nil
	default:
		return nil, fmt.Errorf("state contained unknown mbc %q", m.Name)
	}
}

type bankNumbers struct {
	ROMBankNumber uint16
	RAMBankNumber uint16
	MaxRAMBank    uint16
	MaxROMBank    uint16
}

func (bn *bankNumbers) GetROMBankNumber() int {
	return int(bn.ROMBankNumber)
}
func (bn *bankNumbers) GetRAMBankNumber() int {
	return int(bn.RAMBankNumber)
}
func (bn *bankNumbers) setROMBankNumber(bankNum uint16) {
	// ran into this in dkland2, which will write trash
	// past the amount of rom they have. I figure they
	// don't have the lines hooked up, so let's only
	// "hook up" lines that are actually addressable
	topBit := uint16(0x8000)
	for bn.MaxROMBank < topBit {
		bankNum &^= topBit
		topBit >>= 1
	}
	bn.ROMBankNumber = bankNum
}
func (bn *bankNumbers) setRAMBankNumber(bankNum uint16) {
	topBit := uint16(0x8000)
	for bn.MaxRAMBank < topBit {
		bankNum &^= topBit
		topBit >>= 1
	}
	bn.RAMBankNumber = bankNum
}
func (bn *bankNumbers) init(mem *mem) {
	bn.MaxROMBank = uint16(len(mem.cart)/0x4000 - 1)
	bn.MaxRAMBank = uint16(len(mem.CartRAM)/0x2000 - 1)
}
func (bn *bankNumbers) ROMBankOffset() uint {
	return uint(bn.ROMBankNumber) * 0x4000
}
func (bn *bankNumbers) RAMBankOffset() uint {
	return uint(bn.RAMBankNumber) * 0x2000
}

type nullMBC struct {
	bankNumbers
}

func (mbc *nullMBC) Init(mem *mem) {
	mbc.bankNumbers.init(mem)
	mbc.ROMBankNumber = 1 // set up a flat map
}
func (mbc *nullMBC) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x8000:
		return mem.cart[addr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		if int(localAddr) < len(mem.CartRAM) {
			return mem.CartRAM[localAddr]
		}
		return 0xff
	default:
		panic(fmt.Sprintf("nullMBC: not implemented: read at %x\n", addr))
	}
}
func (mbc *nullMBC) Write(mem *mem, addr uint16, val byte) {
	localAddr := uint(addr - 0xa000)
	if int(localAddr) < len(mem.CartRAM) {
		mem.CartRAM[localAddr] = val
	}
}
func (mbc *nullMBC) Marshal() marshalledMBC {
	return marshalledMBC{Name: "nullMBC"}
}

const (
	bankingModeRAM = iota
	bankingModeROM
)

type mbc1 struct {
	bankNumbers

	RAMEnabled  bool
	BankingMode int
}

func (mbc *mbc1) Init(mem *mem) {
	mbc.bankNumbers.init(mem)
	mbc.ROMBankNumber = 1 // can't go lower
}

func (mbc *mbc1) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.ROMBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc1: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.ROMBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.RAMBankOffset()
		if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
			return mem.CartRAM[localAddr]
		}
		return 0xff
	default:
		panic(fmt.Sprintf("mbc1: not implemented: read at %x\n", addr))
	}
}

func (mbc *mbc1) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		mbc.RAMEnabled = val&0x0f == 0x0a
	case addr >= 0x2000 && addr < 0x4000:
		bankNum := uint16(val & 0x1f)
		if bankNum == 0 {
			// No bank 0 selection. This also disallows any bank
			// with 0 for the bottom 5 bits, i.e. no 0x20, 0x40,
			// or 0x60 banks. Trying to select them will select
			// 0x21, 0x41, or 0x61. Thus a max of 125 banks,
			// 128-3, for MBC1
			bankNum = 1
		}
		bankNum = (mbc.ROMBankNumber &^ 0x1f) | bankNum
		mbc.setROMBankNumber(bankNum)
	case addr >= 0x4000 && addr < 0x6000:
		valBits := uint16(val & 0x03)
		if mbc.BankingMode == bankingModeRAM {
			mbc.setRAMBankNumber(valBits)
		} else { // ROM mode
			bankNum := (valBits << 5) | (mbc.ROMBankNumber & 0x1f)
			mbc.setROMBankNumber(bankNum)
		}
	case addr >= 0x6000 && addr < 0x8000:
		// NOTE: do those two bits from the RAM number need to
		// be passed over to the ROM number after the banking
		// mode changes? (and vice versa?)
		if (val&0x01) > 0 && mbc.BankingMode != bankingModeRAM {
			mbc.BankingMode = bankingModeRAM
			mbc.setROMBankNumber(mbc.ROMBankNumber & 0x1f)
		} else {
			mbc.BankingMode = bankingModeROM
			mbc.setRAMBankNumber(0)
		}
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.RAMBankOffset()
		if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
			mem.CartRAM[localAddr] = val
		}
	default:
		panic(fmt.Sprintf("mbc1: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc1) Marshal() marshalledMBC {
	rawJSON, err := json.Marshal(mbc)
	if err != nil {
		panic(err)
	}
	return marshalledMBC{
		Name: "mbc1",
		Data: rawJSON,
	}
}

type mbc2 struct {
	bankNumbers

	RAMEnabled bool
}

func (mbc *mbc2) Init(mem *mem) {
	mbc.bankNumbers.init(mem)
	mbc.ROMBankNumber = 1 // can't go lower
}

func (mbc *mbc2) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.ROMBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc2: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.ROMBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
			// 4-bit ram (FIXME: pull high nibble down or up?)
			return mem.CartRAM[localAddr] & 0x0f
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
			mbc.RAMEnabled = val&0x0f == 0x0a
		}
	case addr >= 0x2000 && addr < 0x4000:
		if addr&0x0100 == 0 {
			// nop, this bit must be one
		} else {
			// 16 rom banks
			bankNum := uint16(val & 0x0f)
			if bankNum == 0 {
				// no bank 0 selection.
				bankNum = 1
			}
			mbc.setROMBankNumber(bankNum)
		}
	case addr >= 0x4000 && addr < 0x8000:
		// nop
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
			// 4-bit RAM
			mem.CartRAM[localAddr] = val & 0x0f
		}
	default:
		panic(fmt.Sprintf("mbc2: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc2) Marshal() marshalledMBC {
	rawJSON, err := json.Marshal(mbc)
	if err != nil {
		panic(err)
	}
	return marshalledMBC{
		Name: "mbc2",
		Data: rawJSON,
	}
}

type mbc3 struct {
	bankNumbers

	RAMEnabled bool

	Seconds byte
	Minutes byte
	Hours   byte
	Days    uint16

	LatchedSeconds byte
	LatchedMinutes byte
	LatchedHours   byte
	LatchedDays    uint16
	DayCarry       bool

	TimerStopped bool
	TimerLatched bool

	TimeAtLastSet time.Time
}

func (mbc *mbc3) Init(mem *mem) {
	mbc.bankNumbers.init(mem)
	mbc.ROMBankNumber = 1 // can't go lower

	// NOTE: could load this/vals/carry from save or something
	// here. Considering .sav is a simple format (just the RAM
	// proper), should make e.g. an additional .rtc file for this
	mbc.TimeAtLastSet = time.Now()
}

func (mbc *mbc3) updateTimer() {
	if mbc.TimerStopped {
		return
	}
	oldUnix := time.Unix(
		int64(mbc.Seconds)+
			int64(mbc.Minutes)*60+
			int64(mbc.Hours)*60*60+
			int64(mbc.Days)*60*60*24, 0)

	ticked := time.Now().Sub(mbc.TimeAtLastSet)
	mbc.TimeAtLastSet = time.Now()

	newTotalSeconds := oldUnix.Add(ticked).Unix()
	mbc.Seconds = byte(newTotalSeconds % 60)

	newTotalMinutes := (newTotalSeconds - int64(mbc.Seconds)/60)
	mbc.Minutes = byte(newTotalMinutes % 60)

	newTotalHours := (newTotalMinutes - int64(mbc.Minutes)/60)
	mbc.Hours = byte(newTotalHours % 24)

	newTotalDays := (newTotalHours - int64(mbc.Hours)/24)
	mbc.Days = uint16(newTotalDays)

	if newTotalDays > 511 {
		mbc.DayCarry = true
	}
}

func (mbc *mbc3) updateLatch() {
	mbc.updateTimer()
	mbc.LatchedSeconds = mbc.Seconds
	mbc.LatchedMinutes = mbc.Minutes
	mbc.LatchedHours = mbc.Hours
	mbc.LatchedDays = mbc.Days
}

func (mbc *mbc3) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.ROMBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc3: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.ROMBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		switch mbc.RAMBankNumber {
		case 0, 1, 2, 3:
			localAddr := uint(addr-0xa000) + mbc.RAMBankOffset()
			if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
				return mem.CartRAM[localAddr]
			}
			return 0xff
		case 8:
			return mbc.LatchedSeconds
		case 9:
			return mbc.LatchedMinutes
		case 10:
			return mbc.LatchedHours
		case 11:
			return byte(mbc.LatchedDays)
		case 12:
			return boolBit(mbc.DayCarry, 7) | boolBit(mbc.TimerStopped, 6) | (byte(mbc.LatchedDays>>7) & 0x01)
		}
		// might need a default of return 0xff here
	}
	panic(fmt.Sprintf("mbc3: not implemented: read at %x\n", addr))
}

func (mbc *mbc3) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		mbc.RAMEnabled = val&0x0f == 0x0a
	case addr >= 0x2000 && addr < 0x4000:
		bankNum := uint16(val &^ 0x80) // 7bit selector
		if bankNum == 0 {
			// no bank 0 selection.
			bankNum = 1
		}
		mbc.setROMBankNumber(bankNum)
	case addr >= 0x4000 && addr < 0x6000:
		switch val {
		case 0, 1, 2, 3:
			mbc.setRAMBankNumber(uint16(val))
		case 8, 9, 10, 11, 12:
			// sidestep the bank set semantics for the rtc regs
			mbc.RAMBankNumber = uint16(val)
		default:
			// all others nop
			// NOTE: or should they e.g. select a bank that returns all 0xff's?
		}
	case addr >= 0x6000 && addr < 0x8000:
		switch {
		case val&0x01 == 0:
			mbc.TimerLatched = false
		case val&0x01 == 1 && !mbc.TimerLatched:
			mbc.TimerLatched = true
			mbc.updateTimer()
		}
	case addr >= 0xa000 && addr < 0xc000:
		switch mbc.RAMBankNumber {
		case 0, 1, 2, 3:
			localAddr := uint(addr-0xa000) + mbc.RAMBankOffset()
			if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
				mem.CartRAM[localAddr] = val
			}
		case 8:
			mbc.updateTimer()
			mbc.Seconds = val
		case 9:
			mbc.updateTimer()
			mbc.Minutes = val
		case 10:
			mbc.updateTimer()
			mbc.Hours = val
		case 11:
			mbc.updateTimer()
			mbc.Days = uint16(val)
		case 12:
			mbc.updateTimer()
			mbc.Days &^= 0x0100
			mbc.Days |= uint16(val&0x01) << 8
			mbc.TimerStopped = val&(1<<6) > 0
			mbc.DayCarry = val&(1<<7) > 0
		default:
			// nop
		}
	default:
		panic(fmt.Sprintf("mbc3: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc3) Marshal() marshalledMBC {
	rawJSON, err := json.Marshal(mbc)
	if err != nil {
		panic(err)
	}
	return marshalledMBC{
		Name: "mbc3",
		Data: rawJSON,
	}
}

type mbc5 struct {
	bankNumbers

	RAMEnabled bool
}

func (mbc *mbc5) Init(mem *mem) {
	mbc.bankNumbers.init(mem)

	// NOTE: can do bank 0 now, but still start with 1
	mbc.ROMBankNumber = 1
}

func (mbc *mbc5) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.ROMBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("mbc5: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.ROMBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.RAMBankOffset()
		if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
			return mem.CartRAM[localAddr]
		}
		return 0xff
	default:
		panic(fmt.Sprintf("mbc5: not implemented: read at %x\n", addr))
	}
}

func (mbc *mbc5) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		mbc.RAMEnabled = val&0x0f == 0x0a
	case addr >= 0x2000 && addr < 0x3000:
		mbc.setROMBankNumber((mbc.ROMBankNumber &^ 0xff) | uint16(val))
	case addr >= 0x3000 && addr < 0x4000:
		// NOTE: TCAGBD says that games that don't use the 9th bit
		// can use this to set the lower eight! I'll wait until I
		// see a game try to do that before impl'ing
		mbc.setROMBankNumber((mbc.ROMBankNumber &^ 0x100) | uint16(val&0x01)<<8)
	case addr >= 0x4000 && addr < 0x6000:
		mbc.setRAMBankNumber(uint16(val & 0x0f))
	case addr >= 0x6000 && addr < 0x8000:
		// nop?
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr-0xa000) + mbc.RAMBankOffset()
		if mbc.RAMEnabled && int(localAddr) < len(mem.CartRAM) {
			mem.CartRAM[localAddr] = val
		}
	default:
		panic(fmt.Sprintf("mbc5: not implemented: write at %x\n", addr))
	}
}

func (mbc *mbc5) Marshal() marshalledMBC {
	rawJSON, err := json.Marshal(mbc)
	if err != nil {
		panic(err)
	}
	return marshalledMBC{
		Name: "mbc5",
		Data: rawJSON,
	}
}

type gbsMBC struct {
	bankNumbers
}

func (mbc *gbsMBC) Init(mem *mem) {
	mbc.bankNumbers.init(mem)
	mbc.ROMBankNumber = 1 // can't go lower
}

func (mbc *gbsMBC) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.ROMBankOffset()
		if localAddr >= uint(len(mem.cart)) {
			panic(fmt.Sprintf("gbsMBC: bad rom local addr: 0x%06x, bank number: %d\r\n", localAddr, mbc.ROMBankNumber))
		}
		return mem.cart[localAddr]
	case addr >= 0xa000 && addr < 0xc000:
		return mem.CartRAM[addr-0xa000]
	default:
		panic(fmt.Sprintf("gbsMBC: not implemented: read at %x\n", addr))
	}
}

func (mbc *gbsMBC) Write(mem *mem, addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		// nop
	case addr >= 0x2000 && addr < 0x4000:
		// 16 rom banks
		bankNum := uint16(val & 0x0f)
		mbc.setROMBankNumber(bankNum)
	case addr >= 0x4000 && addr < 0x8000:
		// nop
	case addr >= 0xa000 && addr < 0xc000:
		localAddr := uint(addr - 0xa000)
		mem.CartRAM[localAddr] = val
	default:
		panic(fmt.Sprintf("gbsMBC: not implemented: write at %x\n", addr))
	}
}

func (mbc *gbsMBC) Marshal() marshalledMBC {
	rawJSON, err := json.Marshal(mbc)
	if err != nil {
		panic(err)
	}
	return marshalledMBC{
		Name: "gbsMBC",
		Data: rawJSON,
	}
}
