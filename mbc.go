package dmgo

import "fmt"

func makeMBC(cartInfo *CartInfo) mbc {
	switch cartInfo.CartridgeType {
	case 0:
		return &nullMBC{}
	case 1, 2, 3:
		return &mbc1{}
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
}

type nullMBC struct{}

func (mbc *nullMBC) Init(mem *mem) {}
func (mbc *nullMBC) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x8000:
		return mem.cart[addr]
	case addr >= 0xa000 && addr < 0xc000:
		return 0xff
	default:
		panic(fmt.Sprintf("nullMBC: not implemented: read at %x\n", addr))
	}
}
func (mbc *nullMBC) Write(mem *mem, addr uint16, val byte) {
	// if landscape triggers this often, could just access it.
	if len(mem.cartRAM) != 0 {
		panic("RAM not implemented in nullMBC!")
	}
}

type mbc1 struct {
	romBankNumber byte
	ramBankNumber byte
	ramEnabled    bool
	bankingMode   int
}

const (
	bankingModeRAM = iota
	bankingModeROM
)

func (mbc *mbc1) Init(mem *mem) {
	mbc.romBankNumber = 1 // can't go lower
}

func (mbc *mbc1) Read(mem *mem, addr uint16) byte {
	switch {
	case addr < 0x4000:
		return mem.cart[addr]
	case addr >= 0x4000 && addr < 0x8000:
		localAddr := uint(addr-0x4000) + mbc.romBankOffset()
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
