package dmgo

type apu struct {
	// not marshalled in snapshot
	buffer apuCircleBuf

	// everything else marshalled

	AllSoundsOn bool

	SampleTimeCounter float64
	LastSweepCycle    float64
	LastEnvCycle      float64
	LastLengthCycle   float64

	Sounds [4]sound

	// cart chip sounds. never used by any game?
	VInToLeftSpeaker  bool
	VInToRightSpeaker bool

	RightSpeakerVolume byte // right=S01 in docs
	LeftSpeakerVolume  byte // left=S02 in docs
}

func (apu *apu) init() {
	apu.Sounds[0].SoundType = squareSoundType
	apu.Sounds[1].SoundType = squareSoundType
	apu.Sounds[2].SoundType = waveSoundType
	apu.Sounds[3].SoundType = noiseSoundType

	apu.Sounds[3].PolyFeedbackReg = 0x01
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

var timePerCycle = 1.0 / (4 * 1024 * 1024)

func (apu *apu) runCycle(cs *cpuState) {

	if apu.SampleTimeCounter-apu.LastLengthCycle >= 1.0/256.0 {
		apu.runLengthCycle()
		apu.LastLengthCycle = apu.SampleTimeCounter
	}
	if apu.SampleTimeCounter-apu.LastEnvCycle >= 1.0/64.0 {
		apu.runEnvCycle()
		apu.LastEnvCycle = apu.SampleTimeCounter
	}

	if !apu.buffer.full() {

		left, right := 0.0, 0.0
		if apu.AllSoundsOn {
			apu.runFreqCycle()

			left0, right0 := apu.Sounds[0].getSample()
			left1, right1 := apu.Sounds[1].getSample()
			left2, right2 := apu.Sounds[2].getSample()
			left3, right3 := apu.Sounds[3].getSample()
			left = (left0 + left1 + left2 + left3) * 0.25
			right = (right0 + right1 + right2 + right3) * 0.25
			left = left * 0.125 * float64(apu.LeftSpeakerVolume+1)
			right = right * 0.125 * float64(apu.RightSpeakerVolume+1)
		}

		sampleL, sampleR := int16(left*32767.0), int16(right*32767.0)
		apu.buffer.write([]byte{
			byte(sampleL & 0xff),
			byte(sampleL >> 8),
			byte(sampleR & 0xff),
			byte(sampleR >> 8),
		})
	}

	if apu.SampleTimeCounter-apu.LastSweepCycle >= 1.0/128.0 {
		apu.Sounds[0].runSweepCycle()
		apu.LastSweepCycle = apu.SampleTimeCounter
	}

	apu.SampleTimeCounter += timePerCycle
}

func (apu *apu) runFreqCycle() {
	apu.Sounds[0].runFreqCycle()
	apu.Sounds[1].runFreqCycle()
	apu.Sounds[2].runFreqCycle()
	apu.Sounds[3].runFreqCycle()
}
func (apu *apu) runLengthCycle() {
	apu.Sounds[0].runLengthCycle()
	apu.Sounds[1].runLengthCycle()
	apu.Sounds[2].runLengthCycle()
	apu.Sounds[3].runLengthCycle()
}
func (apu *apu) runEnvCycle() {
	apu.Sounds[0].runEnvCycle()
	apu.Sounds[1].runEnvCycle()
	apu.Sounds[2].runEnvCycle()
	apu.Sounds[3].runEnvCycle()
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
	SoundType uint8

	On             bool
	RightSpeakerOn bool // S01 in docs
	LeftSpeakerOn  bool // S02 in docs

	EnvelopeDirection envDir
	EnvelopeStartVal  byte
	EnvelopeSweepVal  byte
	CurrentEnvelope   byte
	EnvelopeCounter   byte

	T       float64
	Freq    float64
	FreqReg uint16

	SweepCounter   byte
	SweepDirection sweepDir
	SweepTime      byte
	SweepShift     byte

	LengthData    uint16
	CurrentLength uint16
	WaveDuty      byte

	WaveOutLvl        byte // sound[2] only
	WavePatternRAM    [16]byte
	WavePatternCursor byte
	WavePatternBias   float64

	PolyFeedbackReg  uint16 // sound[3] only
	PolyDivisorShift byte
	PolyDivisorBase  byte
	Poly7BitMode     bool
	PolySample       float64

	PlaysContinuously bool
	RestartRequested  bool
}

func (sound *sound) runFreqCycle() {

	sound.T += sound.Freq * timePerSample

	for sound.T > 1.0 {
		sound.T -= 1.0
		if sound.SoundType == waveSoundType {
			sound.WavePatternCursor = (sound.WavePatternCursor + 1) & 31
		}
		if sound.SoundType == noiseSoundType {
			sound.updatePolyCounter()
		}
	}
}

func (sound *sound) updatePolyCounter() {
	newHigh := (sound.PolyFeedbackReg & 0x01) ^ ((sound.PolyFeedbackReg >> 1) & 0x01)
	sound.PolyFeedbackReg >>= 1
	sound.PolyFeedbackReg &^= 1 << 14
	sound.PolyFeedbackReg |= newHigh << 14
	if sound.Poly7BitMode {
		sound.PolyFeedbackReg &^= 1 << 6
		sound.PolyFeedbackReg |= newHigh << 6
	}
	var newSample float64
	if sound.PolyFeedbackReg&0x01 == 0 {
		newSample = 1
	} else {
		newSample = -1
	}
	if sound.Poly7BitMode && sound.Freq > 22050 {
		// HACK: high freq 7bit doesn't sound right without one hell
		// of a low-pass filter. Even this is a bit wrong (freq sounds
		// low). Doing this for now/until I do something more drastic, e.g.
		// switching from per-sample sound generation to per-clock
		// generation with some final interpolation step for everything.
		newSample = sound.PolySample + 0.2*(newSample-sound.PolySample)
	}
	sound.PolySample = newSample
}

func (sound *sound) runLengthCycle() {
	if sound.CurrentLength > 0 && !sound.PlaysContinuously {
		sound.CurrentLength--
		if sound.CurrentLength == 0 {
			sound.On = false
		}
	}
	if sound.RestartRequested {
		sound.On = true
		sound.RestartRequested = false
		if sound.LengthData == 0 {
			if sound.SoundType == waveSoundType {
				sound.LengthData = 256
			} else {
				sound.LengthData = 64
			}
		}
		sound.CurrentLength = sound.LengthData
		sound.CurrentEnvelope = sound.EnvelopeStartVal
		sound.SweepCounter = 0
		sound.WavePatternCursor = 0
		sound.PolyFeedbackReg = 0xffff
	}
}

func (sound *sound) runSweepCycle() {
	if sound.SweepTime != 0 {
		if sound.SweepCounter < sound.SweepTime {
			sound.SweepCounter++
		} else {
			sound.SweepCounter = 0
			var nextFreq uint16
			if sound.SweepDirection == sweepUp {
				nextFreq = sound.FreqReg + (sound.FreqReg >> uint16(sound.SweepShift))
			} else {
				nextFreq = sound.FreqReg - (sound.FreqReg >> uint16(sound.SweepShift))
			}
			if nextFreq > 2047 {
				sound.On = false
			} else {
				sound.FreqReg = nextFreq
				sound.updateFreq()
			}
		}
	}
}

func (sound *sound) runEnvCycle() {
	// more complicated, see GBSOUND
	if sound.EnvelopeSweepVal != 0 {
		if sound.EnvelopeCounter < sound.EnvelopeSweepVal {
			sound.EnvelopeCounter++
		} else {
			sound.EnvelopeCounter = 0
			if sound.EnvelopeDirection == envUp {
				if sound.CurrentEnvelope < 0x0f {
					sound.CurrentEnvelope++
				}
			} else {
				if sound.CurrentEnvelope > 0x00 {
					sound.CurrentEnvelope--
				}
			}
		}
	}
}

func (sound *sound) inDutyCycle() bool {
	switch sound.WaveDuty {
	case 0:
		return sound.T > 0.875
	case 1:
		return sound.T < 0.125 || sound.T > 0.875
	case 2:
		return sound.T < 0.125 || sound.T > 0.625
	case 3:
		return sound.T > 0.125 && sound.T < 0.875
	default:
		panic("unknown wave duty")
	}
}

func (sound *sound) getSample() (float64, float64) {
	sample := 0.0
	if sound.On {
		switch sound.SoundType {
		case squareSoundType:
			vol := float64(sound.CurrentEnvelope) / 15.0
			if sound.inDutyCycle() {
				sample = vol
			} else {
				sample = -vol
			}
		case waveSoundType:
			if sound.WaveOutLvl > 0 {
				sampleByte := sound.WavePatternRAM[sound.WavePatternCursor/2]
				var sampleBits byte
				if sound.WavePatternCursor&1 == 0 {
					sampleBits = sampleByte >> 4
				} else {
					sampleBits = sampleByte & 0x0f
				}
				unbiasedSample := float64(sampleBits) - sound.WavePatternBias
				sample = (2.0 * unbiasedSample / 15.0) - 1.0
				if sound.WaveOutLvl > 1 {
					sample /= float64(2 * (sound.WaveOutLvl - 1))
				}
			}
		case noiseSoundType:
			vol := float64(sound.CurrentEnvelope) / 15.0
			sample = vol * sound.PolySample
		}
	}

	left, right := 0.0, 0.0
	if sound.LeftSpeakerOn {
		left = sample
	}
	if sound.RightSpeakerOn {
		right = sample
	}
	return left, right
}

func (sound *sound) updateFreq() {
	switch sound.SoundType {
	case waveSoundType:
		sound.Freq = 32 * 65536.0 / float64(2048-sound.FreqReg)
	case noiseSoundType:
		divisor := 8.0
		if sound.PolyDivisorBase > 0 {
			if sound.PolyDivisorShift >= 14 {
				sound.Freq = 0.0 // clock stops on invalid shift value
			}
			divisor = float64(uint(sound.PolyDivisorBase) << uint(sound.PolyDivisorShift+4))
		}
		sound.Freq = 4194304.0 / divisor
	case squareSoundType:
		sound.Freq = 131072.0 / float64(2048-sound.FreqReg)
	default:
		panic("unexpected sound type")
	}
}

func (sound *sound) writeWaveOnOffReg(val byte) {
	sound.On = val&0x80 != 0
	if sound.On {
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
	for _, b := range sound.WavePatternRAM {
		update(b >> 4)
		update(b & 0x0f)
	}
	sound.WavePatternBias = float64(max-min)/2.0 - 7.5
}

func (sound *sound) writeWavePatternValue(addr uint16, val byte) {
	sound.WavePatternRAM[addr] = val
}

func (sound *sound) writePolyCounterReg(val byte) {
	sound.Poly7BitMode = val&0x08 != 0
	sound.PolyDivisorShift = val >> 4
	sound.PolyDivisorBase = val & 0x07
}
func (sound *sound) readPolyCounterReg() byte {
	val := byte(0)
	if sound.Poly7BitMode {
		val |= 0x08
	}
	val |= sound.PolyDivisorShift << 4
	val |= sound.PolyDivisorBase
	return val
}

func (sound *sound) writeWaveOutLvlReg(val byte) {
	sound.WaveOutLvl = (val >> 5) & 0x03
}
func (sound *sound) readWaveOutLvlReg() byte {
	return (sound.WaveOutLvl << 5) | 0x9f
}

func (sound *sound) writeLengthDataReg(val byte) {
	switch sound.SoundType {
	case waveSoundType:
		sound.LengthData = 256 - uint16(val)
	case noiseSoundType:
		sound.LengthData = 64 - uint16(val&0x3f)
	default:
		panic("writeLengthData: unexpected sound type")
	}
}
func (sound *sound) readLengthDataReg() byte {
	switch sound.SoundType {
	case waveSoundType:
		return byte(256 - sound.LengthData)
	case noiseSoundType:
		return byte(64 - sound.LengthData)
	default:
		panic("writeLengthData: unexpected sound type")
	}
}
func (sound *sound) writeLenDutyReg(val byte) {
	sound.LengthData = 64 - uint16(val&0x3f)
	sound.WaveDuty = val >> 6
}
func (sound *sound) readLenDutyReg() byte {
	return (sound.WaveDuty << 6) | 0x3f
}

func (sound *sound) writeSweepReg(val byte) {
	sound.SweepTime = (val >> 4) & 0x07
	sound.SweepShift = val & 0x07
	if val&0x08 != 0 {
		sound.SweepDirection = sweepDown
	} else {
		sound.SweepDirection = sweepUp
	}
}
func (sound *sound) readSweepReg() byte {
	val := sound.SweepTime << 4
	val |= sound.SweepShift
	if sound.SweepDirection == sweepDown {
		val |= 0x08
	}
	return val | 0x80
}

func (sound *sound) writeSoundEnvReg(val byte) {
	sound.EnvelopeStartVal = val >> 4
	if sound.EnvelopeStartVal == 0 {
		sound.On = false
	}
	if val&0x08 != 0 {
		sound.EnvelopeDirection = envUp
	} else {
		sound.EnvelopeDirection = envDown
	}
	sound.EnvelopeSweepVal = val & 0x07
}
func (sound *sound) readSoundEnvReg() byte {
	val := sound.EnvelopeStartVal<<4 | sound.EnvelopeSweepVal
	if sound.EnvelopeDirection == envUp {
		val |= 0x08
	}
	return val
}

func (sound *sound) writeFreqLowReg(val byte) {
	sound.FreqReg &^= 0x00ff
	sound.FreqReg |= uint16(val)
	sound.updateFreq()
}
func (sound *sound) readFreqLowReg() byte {
	return 0xff
}

func (sound *sound) writeFreqHighReg(val byte) {
	if val&0x80 != 0 {
		sound.RestartRequested = true
	}
	sound.PlaysContinuously = val&0x40 == 0
	sound.FreqReg &^= 0xff00
	sound.FreqReg |= uint16(val&0x07) << 8
	sound.updateFreq()
}
func (sound *sound) readFreqHighReg() byte {
	val := byte(0xff)
	if sound.PlaysContinuously {
		val &^= 0x40 // continuous == 0, uses length == 1
	}
	return val
}

func (apu *apu) writeVolumeReg(val byte) {
	apu.VInToLeftSpeaker = val&0x80 != 0
	apu.VInToRightSpeaker = val&0x08 != 0
	apu.RightSpeakerVolume = (val >> 4) & 0x07
	apu.LeftSpeakerVolume = val & 0x07
}
func (apu *apu) readVolumeReg() byte {
	val := apu.RightSpeakerVolume<<4 | apu.LeftSpeakerVolume
	if apu.VInToLeftSpeaker {
		val |= 0x80
	}
	if apu.VInToRightSpeaker {
		val |= 0x08
	}
	return val
}

func (apu *apu) writeSpeakerSelectReg(val byte) {
	boolsFromByte(val,
		&apu.Sounds[3].LeftSpeakerOn,
		&apu.Sounds[2].LeftSpeakerOn,
		&apu.Sounds[1].LeftSpeakerOn,
		&apu.Sounds[0].LeftSpeakerOn,
		&apu.Sounds[3].RightSpeakerOn,
		&apu.Sounds[2].RightSpeakerOn,
		&apu.Sounds[1].RightSpeakerOn,
		&apu.Sounds[0].RightSpeakerOn,
	)
}
func (apu *apu) readSpeakerSelectReg() byte {
	return byteFromBools(
		apu.Sounds[3].LeftSpeakerOn,
		apu.Sounds[2].LeftSpeakerOn,
		apu.Sounds[1].LeftSpeakerOn,
		apu.Sounds[0].LeftSpeakerOn,
		apu.Sounds[3].RightSpeakerOn,
		apu.Sounds[2].RightSpeakerOn,
		apu.Sounds[1].RightSpeakerOn,
		apu.Sounds[0].RightSpeakerOn,
	)
}

func (apu *apu) writeSoundOnOffReg(val byte) {
	// sound on off shows sounds 1-4 status in
	// lower bits, but writing does not
	// change them.
	boolsFromByte(val,
		&apu.AllSoundsOn,
		nil, nil, nil, nil, nil, nil, nil,
	)
}
func (apu *apu) readSoundOnOffReg() byte {
	return byteFromBools(
		apu.AllSoundsOn,
		true, true, true,
		apu.Sounds[3].On,
		apu.Sounds[2].On,
		apu.Sounds[1].On,
		apu.Sounds[0].On,
	)
}
