package dmgo

type lcd struct {
	framebuffer   []byte
	flipRequested bool // for whateve really draws the fb

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
	useUpperBGTileMap           bool
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
	lcd.framebuffer = make([]byte, 160*144*4)
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

		// FIXME: flip on vBlank instead,
		// this is just for debug
		lcd.flipRequested = true
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

func (cs *cpuState) getTilePixel(tmapAddr, tdataAddr uint16, x, y byte) byte {
	mapByteY, mapByteX := uint16(y>>3), uint16(x>>3)
	mapByte := cs.read(tmapAddr + mapByteY*32 + mapByteX)
	if tdataAddr == 0x8800 {
		mapByte = byte(int(int8(mapByte)) + 128)
	}
	mapBitY, mapBitX := y&0x07, x&0x07
	dataByteL := cs.read(tdataAddr + uint16(mapByte)*16 + uint16(mapBitY)*2)
	dataByteH := cs.read(tdataAddr + uint16(mapByte)*16 + uint16(mapBitY)*2 + 1)
	dataBitL := (dataByteL >> (7 - mapBitX)) & 0x1
	dataBitH := (dataByteH >> (7 - mapBitX)) & 0x1
	pixel := (dataBitH << 1) | dataBitL
	if pixel == 0 {
		return 0
	} else if pixel == 1 {
		return 0x3f
	} else if pixel == 2 {
		return 0x7f
	}
	return 0xff
}

func (cs *cpuState) getBGPixel(x, y byte) byte {
	mapAddr := cs.lcd.getBGTileMapAddr()
	dataAddr := cs.lcd.getBGAndWindowTileDataAddr()
	return cs.getTilePixel(mapAddr, dataAddr, x, y)
}
func (cs *cpuState) getWindowPixel(x, y byte) byte {
	mapAddr := cs.lcd.getWindowTileMapAddr()
	dataAddr := cs.lcd.getBGAndWindowTileDataAddr()
	return cs.getTilePixel(mapAddr, dataAddr, x, y)
}

func (lcd *lcd) getBGTileMapAddr() uint16 {
	if lcd.useUpperBGTileMap {
		return 0x9c00
	}
	return 0x9800
}
func (lcd *lcd) getWindowTileMapAddr() uint16 {
	if lcd.useUpperWindowTileMap {
		return 0x9c00
	}
	return 0x9800
}
func (lcd *lcd) getBGAndWindowTileDataAddr() uint16 {
	if lcd.useLowerBGAndWindowTileData {
		return 0x8000
	}
	return 0x8800
}

func (lcd *lcd) renderScanline(cs *cpuState) {
	if lcd.lyReg >= 144 {
		return
	}
	lcd.fillScanline(0)

	y := lcd.lyReg

	if lcd.displayBG || true {
		bgY := y - lcd.scrollY
		for x := byte(0); x < 160; x++ {
			bgX := x - lcd.scrollX
			pix := cs.getBGPixel(bgX, bgY)
			lcd.setFramebufferPixel(x, y, pix, pix, pix)
		}
	}
	if lcd.displayWindow && y >= lcd.windowY {
		winY := y - lcd.windowY
		winStartX := lcd.windowX - 7
		for x := winStartX; x < 160; x++ {
			pix := cs.getWindowPixel(x-winStartX, winY)
			lcd.setFramebufferPixel(x, y, pix, pix, pix)
		}
	}

	// TODO: OAM work goes here
}
func (lcd *lcd) setFramebufferPixel(xByte, yByte, r, g, b byte) {
	x, y := int(xByte), int(yByte)
	lcd.framebuffer[y*160*4+x*4+0] = r
	lcd.framebuffer[y*160*4+x*4+1] = g
	lcd.framebuffer[y*160*4+x*4+2] = b
	lcd.framebuffer[y*160*4+x*4+3] = 0xff
}
func (lcd *lcd) fillScanline(val byte) {
	y := int(lcd.lyReg)
	for x := 0; x < 160; x++ {
		lcd.framebuffer[y*160*4+x*4+0] = val
		lcd.framebuffer[y*160*4+x*4+1] = val
		lcd.framebuffer[y*160*4+x*4+2] = val
		lcd.framebuffer[y*160*4+x*4+3] = 0xff
	}
}

func (lcd *lcd) writeControlReg(val byte) {
	boolsFromByte(val,
		&lcd.displayOn,
		&lcd.useUpperWindowTileMap,
		&lcd.displayWindow,
		&lcd.useLowerBGAndWindowTileData,
		&lcd.useUpperBGTileMap,
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
		lcd.useUpperBGTileMap,
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
