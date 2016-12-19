package dmgo

import "fmt"

// CartInfo represents a dmg cart header
type CartInfo struct {
	// Title is the game title (11 or 16 chars)
	Title string
	// ManufacturerCode is a mysterious optional 4-char code
	ManufacturerCode string
	// CGBFlag describes if it's CGB, DMG, or both-supported
	CGBFlag byte
	// NewLicenseeCode is used to indicate the publisher
	NewLicenseeCode string
	// SGBFlag indicates SGB support
	SGBFlag byte
	// CartridgeType indicates MBC-type, accessories, etc
	CartridgeType byte
	// ROMSizeCode indicates the size of the ROM
	ROMSizeCode byte
	// RAMSizeCode indicates the size of the RAM
	RAMSizeCode byte
	// DestinationCode shows if the game is meant for Japan or not
	DestinationCode byte
	// OldLicenseeCode is the pre-SGB way to indicate the publisher.
	// 0x33 indicates the NewLicenseeCode is used instead.
	// SGB will not function if the old code is not 0x33.
	OldLicenseeCode byte
	// MaskRomVersion is the version of the game cart. Usually 0x00.
	MaskRomVersion byte
}

// GetRAMSize decodes the ram size code into an actual size
func (ci *CartInfo) GetRAMSize() uint {
	codeSizeMap := map[byte]uint{
		0x00: 0,
		0x01: 2 * 1024,
		0x02: 8 * 1024,
		0x03: 32 * 1024,
		0x04: 128 * 1024,
		0x05: 64 * 1024,
	}
	if size, ok := codeSizeMap[ci.RAMSizeCode]; ok {
		return size
	}
	panic(fmt.Sprintf("unknown RAM size code 0x%02x", ci.RAMSizeCode))
}

func (ci *CartInfo) cgbOnly() bool { return ci.CGBFlag == 0xc0 }

// ParseCartInfo parses a dmg cart header
func ParseCartInfo(cartBytes []byte) *CartInfo {
	cart := CartInfo{}

	cart.CGBFlag = cartBytes[0x143]
	if cart.CGBFlag >= 0x80 {
		cart.Title = string(cartBytes[0x134:0x13f])
		cart.ManufacturerCode = string(cartBytes[0x13f:0x143])
	} else {
		cart.Title = string(cartBytes[0x134:0x144])
	}
	cart.Title = stripZeroes(cart.Title)
	cart.SGBFlag = cartBytes[0x146]
	cart.CartridgeType = cartBytes[0x147]
	cart.ROMSizeCode = cartBytes[0x148]
	cart.RAMSizeCode = cartBytes[0x149]
	cart.DestinationCode = cartBytes[0x14a]
	cart.OldLicenseeCode = cartBytes[0x14b]
	if cart.OldLicenseeCode == 0x33 {
		cart.NewLicenseeCode = string(cartBytes[0x144:0x146])
	}
	cart.MaskRomVersion = cartBytes[0x14c]

	return &cart
}

func stripZeroes(s string) string {
	cursor := len(s)
	for cursor > 0 && s[cursor-1] == '\x00' {
		cursor--
	}
	return s[:cursor]
}
