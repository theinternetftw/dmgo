package dmgo

type lcd struct {
	framebuffer   []byte
	flipRequested bool // for whateve really draws the fb

	videoRAM [0x4000]byte // go ahead and do CGB size

	oam              [160]byte
	parsedOAMEntries [40]oamEntry

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
	lycEqualsLyInterrupt bool

	lyReg  byte
	lycReg byte

	inVBlank     bool
	inHBlank     bool
	accessingOAM bool
	readingData  bool

	// needs to be here for buf, see runCycles
	pendingDisplayWindow bool

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

func (lcd *lcd) updateBufferedControlBits() {
	lcd.displayWindow = lcd.pendingDisplayWindow
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
// (maybe instead of counting cycles I'll count actual instruction time?)
// (or maybe it'll always be dmg cycles and gbc will run the fn e.g. twice instead of 4 times)
func (lcd *lcd) runCycles(cs *cpuState, ncycles uint) {
	if !lcd.displayOn {
		return
	}

	lcd.cyclesSinceLYInc += ncycles

	if lcd.accessingOAM && lcd.cyclesSinceLYInc >= 80 {
		lcd.accessingOAM = false
		lcd.readingData = true
	}

	if lcd.readingData && lcd.cyclesSinceLYInc >= 252 {
		lcd.readingData = false
		lcd.inHBlank = true

		if lcd.hBlankInterrupt {
			cs.lcdStatIRQ = true
		}
	}

	if lcd.cyclesSinceLYInc >= 456 {

		lcd.renderScanline()

		lcd.cyclesSinceLYInc = 0
		if !lcd.inVBlank {
			lcd.accessingOAM = true
			if lcd.oamInterrupt {
				cs.lcdStatIRQ = true
			}
		}
		lcd.inHBlank = false
		lcd.lyReg++

		// It looks like some internal control bits are only
		// updated at the beginning of each scanline.
		// Putting this here because it looks like a game
		// does something like "ok, ly=lyc for the last
		// line of my window, so lets turn the window off",
		// which would fail to draw that last line if you
		// didn't buffer up those changes until after the
		// hblank.
		lcd.updateBufferedControlBits()

		if lcd.lycEqualsLyInterrupt {
			if lcd.lyReg == lcd.lycReg {
				cs.lcdStatIRQ = true
			}
		}
	}

	if lcd.lyReg >= 144 && !lcd.inVBlank {
		lcd.inVBlank = true

		cs.vBlankIRQ = true
		if lcd.vBlankInterrupt {
			cs.lcdStatIRQ = true
		}

		lcd.flipRequested = true
	}

	if lcd.inVBlank {
		lcd.cyclesSinceVBlankStart += ncycles
		if lcd.cyclesSinceVBlankStart >= 456*10 {
			lcd.lyReg = 0
			lcd.inVBlank = false
			lcd.accessingOAM = true
			lcd.cyclesSinceVBlankStart = 0

			if lcd.lycEqualsLyInterrupt {
				if lcd.lyReg == lcd.lycReg {
					cs.lcdStatIRQ = true
				}
			}
		}
	}
}

func (lcd *lcd) getTilePixel(tdataAddr uint16, tileNum, x, y byte) byte {
	if tdataAddr == 0x0800 { // 0x8000 relative
		tileNum = byte(int(int8(tileNum)) + 128)
	}
	mapBitY, mapBitX := y&0x07, x&0x07
	dataByteL := lcd.videoRAM[tdataAddr+(uint16(tileNum)<<4)+(uint16(mapBitY)<<1)]
	dataByteH := lcd.videoRAM[tdataAddr+(uint16(tileNum)<<4)+(uint16(mapBitY)<<1)+1]
	dataBitL := (dataByteL >> (7 - mapBitX)) & 0x1
	dataBitH := (dataByteH >> (7 - mapBitX)) & 0x1
	return (dataBitH << 1) | dataBitL
}
func (lcd *lcd) getTileNum(tmapAddr uint16, x, y byte) byte {
	tileNumY, tileNumX := uint16(y>>3), uint16(x>>3)
	tileNum := lcd.videoRAM[tmapAddr+tileNumY*32+tileNumX]
	return tileNum
}

func (lcd *lcd) getBGPixel(x, y byte) (byte, byte, byte) {
	mapAddr := lcd.getBGTileMapAddr()
	dataAddr := lcd.getBGAndWindowTileDataAddr()
	tileNum := lcd.getTileNum(mapAddr, x, y)
	rawPixel := lcd.getTilePixel(dataAddr, tileNum, x, y)
	palettedPixel := (lcd.backgroundPaletteReg >> (rawPixel * 2)) & 0x03
	return lcd.applyCustomPalette(palettedPixel)
}

func (lcd *lcd) getWindowPixel(x, y byte) (byte, byte, byte) {
	mapAddr := lcd.getWindowTileMapAddr()
	dataAddr := lcd.getBGAndWindowTileDataAddr()
	tileNum := lcd.getTileNum(mapAddr, x, y)
	rawPixel := lcd.getTilePixel(dataAddr, tileNum, x, y)
	palettedPixel := (lcd.backgroundPaletteReg >> (rawPixel * 2)) & 0x03
	return lcd.applyCustomPalette(palettedPixel)
}

func (lcd *lcd) getSpritePixel(e *oamEntry, x, y byte) (byte, byte, byte, bool) {
	tileX, tileY := byte(int16(x)-e.x), byte(int16(y)-e.y)
	if e.xFlip() {
		tileX = 7 - tileX
	}
	if e.yFlip() {
		tileY = e.height - 1 - tileY
	}
	tileNum := e.tileNum
	if e.height == 16 {
		tileNum &^= 0x01
		if tileY >= 8 {
			tileNum++
			tileY -= 8
		}
	}
	rawPixel := lcd.getTilePixel(0x0000, tileNum, tileX, tileY) // addr 8000 relative
	if rawPixel == 0 {
		return 0, 0, 0, false // transparent
	}
	palReg := lcd.objectPalette0Reg
	if e.palSelector() {
		palReg = lcd.objectPalette1Reg
	}
	palettedPixel := (palReg >> (rawPixel * 2)) & 0x03
	r, g, b := lcd.applyCustomPalette(palettedPixel)
	return r, g, b, true
}

func (lcd *lcd) applyCustomPalette(val byte) (byte, byte, byte) {
	// TODO: actual custom palette choices stored in lcd
	outVal := (0xff / 3) * (3 - val)
	return outVal, outVal, outVal
}

// 0x8000 relative
func (lcd *lcd) getBGTileMapAddr() uint16 {
	if lcd.useUpperBGTileMap {
		return 0x1c00
	}
	return 0x1800
}

// 0x8000 relative
func (lcd *lcd) getWindowTileMapAddr() uint16 {
	if lcd.useUpperWindowTileMap {
		return 0x1c00
	}
	return 0x1800
}

// 0x8000 relative
func (lcd *lcd) getBGAndWindowTileDataAddr() uint16 {
	if lcd.useLowerBGAndWindowTileData {
		return 0x0000
	}
	return 0x0800
}

type oamEntry struct {
	y         int16
	x         int16
	height    byte
	tileNum   byte
	flagsByte byte
}

func (e *oamEntry) behindBG() bool    { return e.flagsByte&0x80 != 0 }
func (e *oamEntry) yFlip() bool       { return e.flagsByte&0x40 != 0 }
func (e *oamEntry) xFlip() bool       { return e.flagsByte&0x20 != 0 }
func (e *oamEntry) palSelector() bool { return e.flagsByte&0x10 != 0 }

func (e *oamEntry) inScanline(yByte byte) bool {
	y := int16(yByte)
	return y >= e.y && y < e.y+int16(e.height)
}
func (e *oamEntry) inX(xByte byte) bool {
	x := int16(xByte)
	return x >= e.x && x < e.x+8
}
func (lcd *lcd) parseOAM() {
	height := 8
	if lcd.bigSprites {
		height = 16
	}
	for i := 0; i < 40; i++ {
		addr := i * 4
		lcd.parsedOAMEntries[i] = oamEntry{
			y:         int16(lcd.oam[addr]) - 16,
			x:         int16(lcd.oam[addr+1]) - 8,
			height:    byte(height),
			tileNum:   lcd.oam[addr+2],
			flagsByte: lcd.oam[addr+3],
		}
	}
}

func (lcd *lcd) renderScanline() {
	if lcd.lyReg >= 144 {
		return
	}
	lcd.fillScanline(0)

	y := lcd.lyReg

	// for sprite priority
	bgMask := make([]bool, 160)
	maskR, maskG, maskB := lcd.applyCustomPalette(0)

	if lcd.displayBG || true {
		bgY := y + lcd.scrollY
		for x := byte(0); x < 160; x++ {
			bgX := x + lcd.scrollX
			r, g, b := lcd.getBGPixel(bgX, bgY)
			lcd.setFramebufferPixel(x, y, r, g, b)
			if r == maskR && g == maskG && b == maskB {
				bgMask[x] = true
			}
		}
	}
	if lcd.displayWindow && y >= lcd.windowY {
		winY := y - lcd.windowY
		winStartX := int(lcd.windowX) - 7
		for x := winStartX; x < 160; x++ {
			if x < 0 {
				continue
			}
			r, g, b := lcd.getWindowPixel(byte(x-winStartX), winY)
			lcd.setFramebufferPixel(byte(x), y, r, g, b)
			if r == maskR && g == maskG && b == maskB {
				bgMask[x] = true
			}
		}
	}

	if lcd.displaySprites {
		seen := 0
		lcd.parseOAM()
		for x := byte(0); x < 160 && seen < 11; x++ {
			for i := range lcd.parsedOAMEntries {
				e := &lcd.parsedOAMEntries[i]
				if e.inScanline(y) && e.inX(x) {
					if e.x == int16(x) || x == 0 {
						if seen++; seen == 11 {
							break
						}
					}
					if r, g, b, a := lcd.getSpritePixel(e, x, y); a {
						if !e.behindBG() || bgMask[x] {
							lcd.setFramebufferPixel(x, y, r, g, b)
						}
						break
					}
				}
			}
		}
	}
}

func (lcd *lcd) getFramebufferPixel(xByte, yByte byte) (byte, byte, byte) {
	x, y := int(xByte), int(yByte)
	r := lcd.framebuffer[y*160*4+x*4+0]
	g := lcd.framebuffer[y*160*4+x*4+1]
	b := lcd.framebuffer[y*160*4+x*4+2]
	return r, g, b
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
		&lcd.pendingDisplayWindow,
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
		lcd.pendingDisplayWindow,
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
		&lcd.lycEqualsLyInterrupt,
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
		lcd.lycEqualsLyInterrupt,
		lcd.oamInterrupt,
		lcd.vBlankInterrupt,
		lcd.hBlankInterrupt,
		lcd.lyReg == lcd.lycReg,
		lcd.accessingOAM || lcd.readingData,
		lcd.inVBlank || lcd.readingData,
	)
}
