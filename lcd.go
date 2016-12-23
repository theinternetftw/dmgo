package dmgo

import (
	"sort"
)

type lcd struct {
	framebuffer        []byte
	flipRequested      bool // for whatever really draws the fb
	frameWaitRequested bool // for timing when we skip redraws

	videoRAM [0x4000]byte // go ahead and do CGB size

	oam            [160]byte
	oamForScanline []oamEntry

	stateChangeSinceLastVblank bool

	// for oam sprite priority
	bgMask     [160]bool
	spriteMask [160]bool

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
	lcd.checkStateChangeAndAssignByte(&lcd.oam[addr], val)
}

// lcd is on at startup
func (lcd *lcd) init() {
	lcd.displayOn = true
	lcd.accessingOAM = true // at start of line
	lcd.framebuffer = make([]byte, 160*144*4)
}

func (lcd *lcd) writeVideoRAM(addr uint16, val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.videoRAM[addr], val)
}

// FIXME: timings will have to change for double-speed mode
// (maybe instead of counting cycles I'll count actual instruction time?)
// (or maybe it'll always be dmg cycles and gbc will just produce half as many of them?
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

		if lcd.stateChangeSinceLastVblank {
			lcd.renderScanline()
		}

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

		if lcd.stateChangeSinceLastVblank {
			lcd.flipRequested = true
		}
		lcd.frameWaitRequested = true

		lcd.stateChangeSinceLastVblank = false

		cs.vBlankIRQ = true
		if lcd.vBlankInterrupt {
			cs.lcdStatIRQ = true
		}
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
	tileX := byte(int16(x) - e.x)
	tileY := byte(int16(y) - e.y)
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

var standardPalette = [][]byte{
	{0x00, 0x00, 0x00},
	{0x55, 0x55, 0x55},
	{0xaa, 0xaa, 0xaa},
	{0xff, 0xff, 0xff},
}

func (lcd *lcd) applyCustomPalette(val byte) (byte, byte, byte) {
	// TODO: actual custom palette choices stored in lcd
	outVal := standardPalette[3-val]
	return outVal[0], outVal[1], outVal[2]
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

func (e *oamEntry) inX(xByte byte) bool {
	x := int16(xByte)
	return x >= e.x && x < e.x+8
}
func yInSprite(y byte, spriteY int16, height int) bool {
	return int16(y) >= spriteY && int16(y) < spriteY+int16(height)
}
func (lcd *lcd) parseOAMForScanline(scanline byte) {
	height := 8
	if lcd.bigSprites {
		height = 16
	}
	// use re-slice so we keep backing arry and don't realloc
	lcd.oamForScanline = lcd.oamForScanline[:0]
	for i := 0; i < 40; i++ {
		addr := i * 4
		spriteY := int16(lcd.oam[addr]) - 16
		if yInSprite(scanline, spriteY, height) {
			lcd.oamForScanline = append(lcd.oamForScanline, oamEntry{
				y:         spriteY,
				x:         int16(lcd.oam[addr+1]) - 8,
				height:    byte(height),
				tileNum:   lcd.oam[addr+2],
				flagsByte: lcd.oam[addr+3],
			})
		}
	}
	// lower xs have higher priority
	sort.Stable(sortableOAM(lcd.oamForScanline))
	// limit of 10 sprites per line
	if len(lcd.oamForScanline) > 10 {
		lcd.oamForScanline = lcd.oamForScanline[:10]
	}
}

type sortableOAM []oamEntry

func (s sortableOAM) Less(i, j int) bool { return s[i].x < s[j].x }
func (s sortableOAM) Len() int           { return len(s) }
func (s sortableOAM) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (lcd *lcd) renderScanline() {
	if lcd.lyReg >= 144 {
		return
	}
	lcd.fillScanline(0)

	y := lcd.lyReg

	for i := 0; i < 160; i++ {
		lcd.bgMask[i] = false
		lcd.spriteMask[i] = false
	}
	maskR, maskG, maskB := lcd.applyCustomPalette(0)

	if lcd.displayBG || true {
		bgY := y + lcd.scrollY
		for x := byte(0); x < 160; x++ {
			bgX := x + lcd.scrollX
			r, g, b := lcd.getBGPixel(bgX, bgY)
			lcd.setFramebufferPixel(x, y, r, g, b)
			if r == maskR && g == maskG && b == maskB {
				lcd.bgMask[x] = true
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
				lcd.bgMask[x] = true
			}
		}
	}

	if lcd.displaySprites {
		lcd.parseOAMForScanline(y)
		for i := range lcd.oamForScanline {
			e := &lcd.oamForScanline[i]
			lcd.renderSpriteAtScanline(e, y)
		}
	}
}

func (lcd *lcd) renderSpriteAtScanline(e *oamEntry, y byte) {
	startX := byte(0)
	if e.x > 0 {
		startX = byte(e.x)
	}
	endX := byte(e.x + 8)
	for x := startX; x < endX && x < 160; x++ {
		if (!e.behindBG() || lcd.bgMask[x]) && !lcd.spriteMask[x] {
			if r, g, b, a := lcd.getSpritePixel(e, x, y); a {
				lcd.setFramebufferPixel(x, y, r, g, b)
				lcd.spriteMask[x] = true
			}
		}
	}
}

func (lcd *lcd) getFramebufferPixel(xByte, yByte byte) (byte, byte, byte) {
	x, y := int(xByte), int(yByte)
	yIdx := y * 160 * 4
	r := lcd.framebuffer[yIdx+x*4+0]
	g := lcd.framebuffer[yIdx+x*4+1]
	b := lcd.framebuffer[yIdx+x*4+2]
	return r, g, b
}
func (lcd *lcd) setFramebufferPixel(xByte, yByte, r, g, b byte) {
	x, y := int(xByte), int(yByte)
	yIdx := y * 160 * 4
	lcd.framebuffer[yIdx+x*4+0] = r
	lcd.framebuffer[yIdx+x*4+1] = g
	lcd.framebuffer[yIdx+x*4+2] = b
	lcd.framebuffer[yIdx+x*4+3] = 0xff
}
func (lcd *lcd) fillScanline(val byte) {
	yIdx := int(lcd.lyReg) * 160 * 4
	for x := 0; x < 160; x++ {
		lcd.framebuffer[yIdx+x*4+0] = val
		lcd.framebuffer[yIdx+x*4+1] = val
		lcd.framebuffer[yIdx+x*4+2] = val
		lcd.framebuffer[yIdx+x*4+3] = 0xff
	}
}

func (lcd *lcd) checkStateChangeAndAssignByte(dest *byte, val byte) {
	if *dest != val {
		lcd.stateChangeSinceLastVblank = true
		*dest = val
	}
}

func (lcd *lcd) writeScrollY(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.scrollY, val)
}
func (lcd *lcd) writeScrollX(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.scrollX, val)
}
func (lcd *lcd) writeLycReg(val byte) {
	lcd.lycReg = val
}
func (lcd *lcd) writeBackgroundPaletteReg(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.backgroundPaletteReg, val)
}
func (lcd *lcd) writeObjectPalette0Reg(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.objectPalette0Reg, val)
}
func (lcd *lcd) writeObjectPalette1Reg(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.objectPalette1Reg, val)
}
func (lcd *lcd) writeWindowY(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.windowY, val)
}
func (lcd *lcd) writeWindowX(val byte) {
	lcd.checkStateChangeAndAssignByte(&lcd.windowX, val)
}

func (lcd *lcd) writeControlReg(val byte) {
	if lcd.readControlReg() != val {
		lcd.stateChangeSinceLastVblank = true

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
