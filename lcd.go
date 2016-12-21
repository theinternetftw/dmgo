package dmgo

type lcd struct {
	framebuffer [160 * 144 * 4]byte

	oam [160]byte

	scrollY byte
	scrollX byte
	windowY byte
	windowX byte

	backgroundPaletteReg byte
	objectPalette0Reg    byte
	objectPalette1Reg    byte

	hBlankInterrupt      bool
	vBlankInterrupt      bool
	oamInterrupt         bool
	lcyEqualsLyInterrupt bool

	lyReg  byte
	lycReg byte

	inVBlank     bool
	inHBlank     bool
	accessingOAM bool
	readingData  bool

	// control bits
	displayOn                   bool
	useUpperWindowTileMap       bool
	displayWindow               bool
	useLowerBGAndWindowTileData bool
	useUpperBGAndWindowTileMap  bool
	bigSprites                  bool
	displaySprites              bool
	displayBG                   bool

	cyclesSinceLYInc       uint
	cyclesSinceVBlankStart uint
}

func (lcd *lcd) writeOAM(addr uint16, val byte) {
	// TODO: display mode checks (most disallow writing)
	// TODO: OAM
	lcd.oam[addr] = val
}

// lcd is on at startup
func (lcd *lcd) init() {
	lcd.displayOn = true
	lcd.accessingOAM = true // at start of line
}

// FIXME: timings will have to change for double-speed mode
// FIXME: also need to actually *do* lcd activies here
// FIXME: will need to pass in mem here once actually drawing
// (maybe instead of counting cycles I'll count actual instruction time?)
// (or maybe it'll always be dmg cycles and gbc will run the fn e.g. twice instead of 4 times)
//
// FIXME: surely better way instead of having
// a "run every cycle" function, would be a
// step function that takes the number of
// cycles as an arg, does lcd += cycles,
// then updates flags accordingly? Would
// cut number of times this fn is run by
// prolly ~8x.
func (lcd *lcd) runCycle(cs *cpuState) {
	if !lcd.displayOn {
		return
	}

	lcd.cyclesSinceLYInc++
	if lcd.cyclesSinceLYInc == 80 {
		lcd.accessingOAM = false
		lcd.readingData = true
	} else if lcd.cyclesSinceLYInc == 252 {
		lcd.readingData = false
		lcd.inHBlank = true
	} else if lcd.cyclesSinceLYInc == 456 {
		lcd.renderScanline(cs)
		lcd.cyclesSinceLYInc = 0
		lcd.accessingOAM = true
		lcd.inHBlank = false
		lcd.lyReg++
	}

	if lcd.lyReg == 144 && !lcd.inVBlank {
		lcd.inVBlank = true
		cs.vBlankIRQ = true
	}
	if lcd.inVBlank {
		lcd.cyclesSinceVBlankStart++
		if lcd.cyclesSinceVBlankStart > 456*10 {
			lcd.lyReg = 0
			lcd.inVBlank = false
			lcd.cyclesSinceVBlankStart = 0
		}
	}
}

func (lcd *lcd) renderScanline(cs *cpuState) {
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
		lcd.accessingOAM || lcd.readingData,
		lcd.inVBlank || lcd.readingData,
	)
}
