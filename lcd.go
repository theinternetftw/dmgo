package dmgo

type lcd struct {
	framebuffer [160 * 144 * 4]byte

	scrollY byte
	scrollX byte

	hBlankInterrupt      bool
	vBlankInterrupt      bool
	oamInterrupt         bool
	lcyEqualsLyInterrupt bool

	lyReg  byte
	lycReg byte

	inVBlank         bool
	inHBlank         bool
	searchingOAM     bool
	transferringData bool

	displayOn                   bool
	useUpperWindowTileMap       bool
	displayWindow               bool
	useLowerBGAndWindowTileData bool
	useUpperBGAndWindowTileMap  bool
	bigSprites                  bool
	displaySprites              bool
	displayBG                   bool
}

func (lcd *lcd) writeControlReg(val byte) {
	boolsFromByte(val,
		&lcd.displayOn,
		&lcd.useUpperWindowTileMap,
		&lcd.displayWindow,
		&lcd.useLowerBGAndWindowTileData,
		&lcd.useUpperBGAndWindowTileMap,
		&lcd.bigSprites,
		&lcd.displaySprites,
		&lcd.displayBG,
	)
}
func (lcd *lcd) readControlReg() byte {
	return byteFromBools(
		lcd.displayOn,
		lcd.useUpperWindowTileMap,
		lcd.displayWindow,
		lcd.useLowerBGAndWindowTileData,
		lcd.useUpperBGAndWindowTileMap,
		lcd.bigSprites,
		lcd.displaySprites,
		lcd.displayBG,
	)
}

func (lcd *lcd) writeStatusReg(val byte) {
	boolsFromByte(val,
		nil,
		&lcd.lcyEqualsLyInterrupt,
		&lcd.oamInterrupt,
		&lcd.vBlankInterrupt,
		&lcd.hBlankInterrupt,
		nil,
		nil,
		nil,
	)
}
func (lcd *lcd) readStatusReg() byte {
	return byteFromBools(
		true, // bit 7 always set
		lcd.lcyEqualsLyInterrupt,
		lcd.oamInterrupt,
		lcd.vBlankInterrupt,
		lcd.hBlankInterrupt,
		lcd.lyReg == lcd.lycReg,
		lcd.inVBlank,
		lcd.transferringData,
	)
}

func ifBool(b bool, fn func()) {
	if b {
		fn()
	}
}
func byteFromBools(b7, b6, b5, b4, b3, b2, b1, b0 bool) byte {
	var result byte
	ifBool(b7, func() { result |= 0x80 })
	ifBool(b6, func() { result |= 0x40 })
	ifBool(b5, func() { result |= 0x20 })
	ifBool(b4, func() { result |= 0x10 })
	ifBool(b3, func() { result |= 0x08 })
	ifBool(b2, func() { result |= 0x04 })
	ifBool(b1, func() { result |= 0x02 })
	ifBool(b0, func() { result |= 0x01 })
	return result
}

func ifBoolPtrNotNil(bptr *bool, fn func()) {
	if bptr != nil {
		fn()
	}
}
func boolsFromByte(val byte, b7, b6, b5, b4, b3, b2, b1, b0 *bool) {
	ifBoolPtrNotNil(b7, func() { *b7 = val&0x80 > 0 })
	ifBoolPtrNotNil(b6, func() { *b6 = val&0x40 > 0 })
	ifBoolPtrNotNil(b5, func() { *b5 = val&0x20 > 0 })
	ifBoolPtrNotNil(b4, func() { *b4 = val&0x10 > 0 })
	ifBoolPtrNotNil(b3, func() { *b3 = val&0x08 > 0 })
	ifBoolPtrNotNil(b2, func() { *b2 = val&0x04 > 0 })
	ifBoolPtrNotNil(b1, func() { *b1 = val&0x02 > 0 })
	ifBoolPtrNotNil(b0, func() { *b0 = val&0x01 > 0 })
}
