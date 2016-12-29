package dmgo

type apu struct {
	allSoundsOn bool

	buffer apuCircleBuf

	sampleTimeCounter float64
	lastSweepCycle    float64
	lastEnvCycle      float64
	lastLengthCycle   float64

	sounds [4]sound

	// cart chip sounds. never used by any game?
	vInToLeftSpeaker  bool
	vInToRightSpeaker bool

	rightSpeakerVolume byte // right=S01 in docs
	leftSpeakerVolume  byte // left=S02 in docs
}

func (apu *apu) init() {
	apu.sounds[0].soundType = squareSoundType
	apu.sounds[1].soundType = squareSoundType
	apu.sounds[2].soundType = waveSoundType
	apu.sounds[3].soundType = noiseSoundType

	apu.sounds[3].polyFeedbackReg = 0x01
}

const (
	amountGenerateAhead = 64 * 4
	samplesPerSecond    = 44100
	timePerSample       = 1.0 / samplesPerSecond
)

const apuCircleBufSize = amountGenerateAhead

// NOTE: size must be power of 2
type apuCircleBuf struct {
	writeIndex uint
	readIndex  uint
	buf        [apuCircleBufSize]byte
}

func (c *apuCircleBuf) write(bytes []byte) (writeCount int) {
	for _, b := range bytes {
		if c.full() {
			return writeCount
		}
		c.buf[c.mask(c.writeIndex)] = b
		c.writeIndex++
		writeCount++
	}
	return writeCount
}
func (c *apuCircleBuf) read(preSizedBuf []byte) []byte {
	readCount := 0
	for i := range preSizedBuf {
		if c.size() == 0 {
			break
		}
		preSizedBuf[i] = c.buf[c.mask(c.readIndex)]
		c.readIndex++
		readCount++
	}
	return preSizedBuf[:readCount]
}
func (c *apuCircleBuf) mask(i uint) uint { return i & (uint(len(c.buf)) - 1) }
func (c *apuCircleBuf) size() uint       { return c.writeIndex - c.readIndex }
func (c *apuCircleBuf) full() bool       { return c.size() == uint(len(c.buf)) }

func (apu *apu) runCycle(cs *cpuState) {

	if !apu.buffer.full() {

		if apu.sampleTimeCounter-apu.lastLengthCycle >= 1.0/256.0 {
			apu.runLengthCycle()
			apu.lastLengthCycle = apu.sampleTimeCounter
		}
		if apu.sampleTimeCounter-apu.lastEnvCycle >= 1.0/64.0 {
			apu.runEnvCycle()
			apu.lastEnvCycle = apu.sampleTimeCounter
		}

		left, right := 0.0, 0.0
		if apu.allSoundsOn {
			apu.runFreqCycle()

			left0, right0 := apu.sounds[0].getSample()
			left1, right1 := apu.sounds[1].getSample()
			left2, right2 := apu.sounds[2].getSample()
			left3, right3 := apu.sounds[3].getSample()
			left = (left0 + left1 + left2 + left3) / 4.0
			right = (right0 + right1 + right2 + right3) / 4.0
			left = left / 8.0 * float64(apu.leftSpeakerVolume+1)
			right = right / 8.0 * float64(apu.rightSpeakerVolume+1)
		}
		sampleL, sampleR := int16(left*32767.0), int16(right*32767.0)
		apu.buffer.write([]byte{
			byte(sampleL & 0xff),
			byte(sampleL >> 8),
			byte(sampleR & 0xff),
			byte(sampleR >> 8),
		})

		if apu.sampleTimeCounter-apu.lastSweepCycle >= 1.0/128.0 {
			apu.sounds[0].runSweepCycle()
			apu.lastSweepCycle = apu.sampleTimeCounter
		}

		apu.sampleTimeCounter += timePerSample
	}
}

func (apu *apu) runFreqCycle() {
	apu.sounds[0].runFreqCycle()
	apu.sounds[1].runFreqCycle()
	apu.sounds[2].runFreqCycle()
	apu.sounds[3].runFreqCycle()
}
func (apu *apu) runLengthCycle() {
	apu.sounds[0].runLengthCycle()
	apu.sounds[1].runLengthCycle()
	apu.sounds[2].runLengthCycle()
	apu.sounds[3].runLengthCycle()
}
func (apu *apu) runEnvCycle() {
	apu.sounds[0].runEnvCycle()
	apu.sounds[1].runEnvCycle()
	apu.sounds[2].runEnvCycle()
	apu.sounds[3].runEnvCycle()
}

type envDir bool

var (
	envUp   = envDir(true)
	envDown = envDir(false)
)

type sweepDir bool

var (
	sweepUp   = sweepDir(false)
	sweepDown = sweepDir(true)
)

const (
	squareSoundType = 0
	waveSoundType   = 1
	noiseSoundType  = 2
)

type sound struct {
	soundType uint8

	on             bool
	rightSpeakerOn bool // S01 in docs
	leftSpeakerOn  bool // S02 in docs

	envelopeDirection envDir
	envelopeStartVal  byte
	envelopeSweepVal  byte
	currentEnvelope   byte
	envelopeCounter   byte

	t            float64
	freq         float64
	freqReg      uint16
	sweepCounter byte

	sweepDirection sweepDir
	sweepTime      byte
	sweepShift     byte

	lengthData    uint16
	currentLength uint16
	waveDuty      byte

	waveOutLvl        byte // sound[2] only
	wavePatternRAM    [16]byte
	wavePatternCursor byte
	wavePatternBias   float64

	polyFeedbackReg  uint16 // sound[3] only
	polyDivisorShift byte
	polyDivisorBase  byte
	poly7BitMode     bool
	polySample       float64

	playsContinuously bool
	restartRequested  bool
}

func (sound *sound) runFreqCycle() {

	sound.t += sound.freq * timePerSample

	for sound.t > 1.0 {
		sound.t -= 1.0
		if sound.soundType == waveSoundType {
			sound.wavePatternCursor = (sound.wavePatternCursor + 1) & 31
		}
		if sound.soundType == noiseSoundType {
			sound.updatePolyCounter()
		}
	}
}

func (sound *sound) updatePolyCounter() {
	newHigh := (sound.polyFeedbackReg & 0x01) ^ ((sound.polyFeedbackReg >> 1) & 0x01)
	sound.polyFeedbackReg >>= 1
	sound.polyFeedbackReg &^= 1 << 14
	sound.polyFeedbackReg |= newHigh << 14
	if sound.poly7BitMode {
		sound.polyFeedbackReg &^= 1 << 6
		sound.polyFeedbackReg |= newHigh << 6
	}
	var newSample float64
	if sound.polyFeedbackReg&0x01 == 0 {
		newSample = 1
	} else {
		newSample = -1
	}
	if sound.poly7BitMode && sound.freq > 22050 {
		// HACK: high freq 7bit doesn't sound right without one hell
		// of a low-pass filter. Even this is a bit wrong (freq sounds
		// low). Doing this for now/until I do something more drastic, e.g.
		// switching from per-sample sound generation to per-clock
		// generation with some final interpolation step for everything.
		newSample = sound.polySample + 0.2*(newSample-sound.polySample)
	}
	sound.polySample = newSample
}

func (sound *sound) runLengthCycle() {
	if sound.currentLength > 0 && !sound.playsContinuously {
		sound.currentLength--
		if sound.currentLength == 0 {
			sound.on = false
		}
	}
	if sound.restartRequested {
		sound.on = true
		sound.restartRequested = false
		if sound.lengthData == 0 {
			if sound.soundType == waveSoundType {
				sound.lengthData = 256
			} else {
				sound.lengthData = 64
			}
		}
		sound.currentLength = sound.lengthData
		sound.currentEnvelope = sound.envelopeStartVal
		sound.sweepCounter = 0
		sound.wavePatternCursor = 0
		sound.polyFeedbackReg = 0xffff
	}
}

func (sound *sound) runSweepCycle() {
	if sound.sweepTime != 0 {
		if sound.sweepCounter < sound.sweepTime {
			sound.sweepCounter++
		} else {
			sound.sweepCounter = 0
			var nextFreq uint16
			if sound.sweepDirection == sweepUp {
				nextFreq = sound.freqReg + (sound.freqReg >> uint16(sound.sweepShift))
			} else {
				nextFreq = sound.freqReg - (sound.freqReg >> uint16(sound.sweepShift))
			}
			if nextFreq > 2047 {
				sound.on = false
			} else {
				sound.freqReg = nextFreq
				sound.updateFreq()
			}
		}
	}
}

func (sound *sound) runEnvCycle() {
	// more complicated, see GBSOUND
	if sound.envelopeSweepVal != 0 {
		if sound.envelopeCounter < sound.envelopeSweepVal {
			sound.envelopeCounter++
		} else {
			sound.envelopeCounter = 0
			if sound.envelopeDirection == envUp {
				if sound.currentEnvelope < 0x0f {
					sound.currentEnvelope++
				}
			} else {
				if sound.currentEnvelope > 0x00 {
					sound.currentEnvelope--
				}
			}
		}
	}
}

func (sound *sound) inDutyCycle() bool {
	switch sound.waveDuty {
	case 0:
		return sound.t > 0.875
	case 1:
		return sound.t < 0.125 || sound.t > 0.875
	case 2:
		return sound.t < 0.125 || sound.t > 0.625
	case 3:
		return sound.t > 0.125 && sound.t < 0.875
	default:
		panic("unknown wave duty")
	}
}

func (sound *sound) getSample() (float64, float64) {
	sample := 0.0
	if sound.on {
		switch sound.soundType {
		case squareSoundType:
			vol := float64(sound.currentEnvelope) / 15.0
			if sound.inDutyCycle() {
				sample = vol
			} else {
				sample = -vol
			}
		case waveSoundType:
			if sound.waveOutLvl > 0 {
				sampleByte := sound.wavePatternRAM[sound.wavePatternCursor/2]
				var sampleBits byte
				if sound.wavePatternCursor&1 == 0 {
					sampleBits = sampleByte >> 4
				} else {
					sampleBits = sampleByte & 0x0f
				}
				unbiasedSample := float64(sampleBits) - sound.wavePatternBias
				sample = (2.0 * unbiasedSample / 15.0) - 1.0
				if sound.waveOutLvl > 1 {
					sample /= float64(2 * (sound.waveOutLvl - 1))
				}
			}
		case noiseSoundType:
			vol := float64(sound.currentEnvelope) / 15.0
			sample = vol * sound.polySample
		}
	}

	left, right := 0.0, 0.0
	if sound.leftSpeakerOn {
		left = sample
	}
	if sound.rightSpeakerOn {
		right = sample
	}
	return left, right
}

func (sound *sound) updateFreq() {
	switch sound.soundType {
	case waveSoundType:
		sound.freq = 32 * 65536.0 / float64(2048-sound.freqReg)
	case noiseSoundType:
		divisor := 8.0
		if sound.polyDivisorBase > 0 {
			if sound.polyDivisorShift >= 14 {
				sound.freq = 0.0 // clock stops on invalid shift value
			}
			divisor = float64(uint16(sound.polyDivisorBase) << uint16(sound.polyDivisorShift+4))
		}
		sound.freq = 4194304.0 / divisor
	case squareSoundType:
		sound.freq = 131072.0 / float64(2048-sound.freqReg)
	default:
		panic("unexpected sound type")
	}
}

func (sound *sound) writeWaveOnOffReg(val byte) {
	sound.on = val&0x80 != 0
	if sound.on {
		sound.updateWavePatternBias()
	}
}

func (sound *sound) updateWavePatternBias() {
	max, min := byte(0), byte(0)
	update := func(nib byte) {
		if nib < min {
			min = nib
		}
		if nib > max {
			max = nib
		}
	}
	for _, b := range sound.wavePatternRAM {
		update(b >> 4)
		update(b & 0x0f)
	}
	sound.wavePatternBias = float64(max-min)/2.0 - 7.5
}

func (sound *sound) writeWavePatternValue(addr uint16, val byte) {
	sound.wavePatternRAM[addr] = val
}

func (sound *sound) writePolyCounterReg(val byte) {
	sound.poly7BitMode = val&0x08 != 0
	sound.polyDivisorShift = val >> 4
	sound.polyDivisorBase = val & 0x07
}
func (sound *sound) readPolyCounterReg() byte {
	val := byte(0)
	if sound.poly7BitMode {
		val |= 0x08
	}
	val |= sound.polyDivisorShift << 4
	val |= sound.polyDivisorBase
	return val
}

func (sound *sound) writeWaveOutLvlReg(val byte) {
	sound.waveOutLvl = (val >> 5) & 0x03
}
func (sound *sound) readWaveOutLvlReg() byte {
	return (sound.waveOutLvl << 5) | 0x9f
}

func (sound *sound) writeLengthDataReg(val byte) {
	switch sound.soundType {
	case waveSoundType:
		sound.lengthData = 256 - uint16(val)
	case noiseSoundType:
		sound.lengthData = 64 - uint16(val&0x3f)
	default:
		panic("writeLengthData: unexpected sound type")
	}
}
func (sound *sound) readLengthDataReg() byte {
	switch sound.soundType {
	case waveSoundType:
		return byte(256 - sound.lengthData)
	case noiseSoundType:
		return byte(64 - sound.lengthData)
	default:
		panic("writeLengthData: unexpected sound type")
	}
}
func (sound *sound) writeLenDutyReg(val byte) {
	sound.lengthData = 64 - uint16(val&0x3f)
	sound.waveDuty = val >> 6
}
func (sound *sound) readLenDutyReg() byte {
	return (sound.waveDuty << 6) & 0x3f
}

func (sound *sound) writeSweepReg(val byte) {
	sound.sweepTime = (val >> 4) & 0x07
	sound.sweepShift = val & 0x07
	if val&0x08 != 0 {
		sound.sweepDirection = sweepDown
	} else {
		sound.sweepDirection = sweepUp
	}
}
func (sound *sound) readSweepReg() byte {
	val := sound.sweepTime << 4
	val |= sound.sweepShift
	if sound.sweepDirection == sweepDown {
		val |= 0x08
	}
	return val
}

func (sound *sound) writeSoundEnvReg(val byte) {
	sound.envelopeStartVal = val >> 4
	if val&0x08 != 0 {
		sound.envelopeDirection = envUp
	} else {
		sound.envelopeDirection = envDown
	}
	sound.envelopeSweepVal = val & 0x07
}
func (sound *sound) readSoundEnvReg() byte {
	val := sound.envelopeStartVal<<4 | sound.envelopeSweepVal
	if sound.envelopeDirection == envUp {
		val |= 0x08
	}
	return val
}

func (sound *sound) writeFreqLowReg(val byte) {
	sound.freqReg &^= 0x00ff
	sound.freqReg |= uint16(val)
	sound.updateFreq()
}
func (sound *sound) readFreqLowReg() byte {
	return 0xff
}

func (sound *sound) writeFreqHighReg(val byte) {
	if val&0x80 != 0 {
		sound.restartRequested = true
	}
	sound.playsContinuously = val&0x40 == 0
	sound.freqReg &^= 0xff00
	sound.freqReg |= uint16(val&0x07) << 8
	sound.updateFreq()
}
func (sound *sound) readFreqHighReg() byte {
	val := byte(0xff)
	if sound.playsContinuously {
		val &^= 0x40 // continuous == 0, uses length == 1
	}
	return val
}

func (apu *apu) writeVolumeReg(val byte) {
	apu.vInToLeftSpeaker = val&0x80 != 0
	apu.vInToRightSpeaker = val&0x08 != 0
	apu.rightSpeakerVolume = (val >> 4) & 0x07
	apu.leftSpeakerVolume = val & 0x07
}
func (apu *apu) readVolumeReg() byte {
	val := apu.rightSpeakerVolume<<4 | apu.leftSpeakerVolume
	if apu.vInToLeftSpeaker {
		val |= 0x80
	}
	if apu.vInToRightSpeaker {
		val |= 0x08
	}
	return val
}

func (apu *apu) writeSpeakerSelectReg(val byte) {
	boolsFromByte(val,
		&apu.sounds[3].leftSpeakerOn,
		&apu.sounds[2].leftSpeakerOn,
		&apu.sounds[1].leftSpeakerOn,
		&apu.sounds[0].leftSpeakerOn,
		&apu.sounds[3].rightSpeakerOn,
		&apu.sounds[2].rightSpeakerOn,
		&apu.sounds[1].rightSpeakerOn,
		&apu.sounds[0].rightSpeakerOn,
	)
}
func (apu *apu) readSpeakerSelectReg() byte {
	return byteFromBools(
		apu.sounds[3].leftSpeakerOn,
		apu.sounds[2].leftSpeakerOn,
		apu.sounds[1].leftSpeakerOn,
		apu.sounds[0].leftSpeakerOn,
		apu.sounds[3].rightSpeakerOn,
		apu.sounds[2].rightSpeakerOn,
		apu.sounds[1].rightSpeakerOn,
		apu.sounds[0].rightSpeakerOn,
	)
}

func (apu *apu) writeSoundOnOffReg(val byte) {
	// sound on off shows sounds 1-4 status in
	// lower bits, but writing does not
	// change them.
	boolsFromByte(val,
		&apu.allSoundsOn,
		nil, nil, nil, nil, nil, nil, nil,
	)
}
func (apu *apu) readSoundOnOffReg() byte {
	return byteFromBools(
		apu.allSoundsOn,
		true, true, true,
		apu.sounds[3].on,
		apu.sounds[2].on,
		apu.sounds[1].on,
		apu.sounds[0].on,
	)
}
