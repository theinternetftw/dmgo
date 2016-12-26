package dmgo

type apu struct {
	allSoundsOn bool

	lastMaxRequested int

	buffer []byte

	sounds [4]sound

	// cart chip sounds. never used by any game?
	vInToLeftSpeaker  bool
	vInToRightSpeaker bool

	rightSpeakerVolume byte // right=S01 in docs
	leftSpeakerVolume  byte // left=S02 in docs
}

func (apu *apu) init() {
	apu.lastMaxRequested = 8192 * 2 * 2

	apu.sounds[0].soundType = squareSoundType
	apu.sounds[1].soundType = squareSoundType
	apu.sounds[2].soundType = waveSoundType
	apu.sounds[3].soundType = noiseSoundType
}

const timePerSample = 1.0 / 44100.0

func (apu *apu) runCycle(cs *cpuState) {
	for len(apu.buffer) < 2*apu.lastMaxRequested {
		apu.runFreqCycle()
		if cs.timerDivCycles&0x3f == 0x3f {
			apu.runLengthCycle()
		}

		left, right := int(0), int(0)
		if apu.allSoundsOn {
			left0, right0 := apu.sounds[0].getSample()
			left1, right1 := apu.sounds[1].getSample()
			left = (left0 + left1) / 2
			right = (right0 + right1) / 2
			//left2, right2 := apu.sounds[2].getSample()
			//left = (left0 + left1 + left2) / 3
			//right = (right0 + right1 + right2) / 3
			left = left / 7 * int(apu.leftSpeakerVolume)
			right = right / 7 * int(apu.rightSpeakerVolume)
		}
		apu.buffer = append(apu.buffer,
			byte(left&0xff),
			byte(left>>8),
			byte(right&0xff),
			byte(right>>8))
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

	sweepDirection sweepDir
	sweepTime      byte
	sweepShift     byte

	lengthData    uint16
	currentLength uint16
	waveDuty      byte

	waveOutLvl     byte // sound[2] only
	wavePatternRAM [16]byte

	polyShiftFreq byte // sound[3] only
	polyStep      byte
	polyDivRatio  byte

	playsContinuously bool
	restartRequested  bool

	freqReg uint16

	t float64
}

func (sound *sound) runFreqCycle() {
	sound.t += sound.getFreq() * timePerSample
	if sound.t > 1.0 {
		sound.t -= 1.0
	}
}
func (sound *sound) runLengthCycle() {
	if sound.playsContinuously {
		return
	}
	if sound.currentLength > 0 {
		sound.currentLength--
		if sound.currentLength == 0 {
			sound.on = false
		}
	}
	if sound.restartRequested {
		sound.on = true
		sound.restartRequested = false
		sound.currentLength = sound.lengthData
	}
}
func (sound *sound) getDutyCycle() float64 {
	switch sound.waveDuty {
	case 0:
		return 0.125
	case 1:
		return 0.25
	case 2:
		return 0.50
	default:
		return 0.75
	}
}
func (sound *sound) getSample() (int, int) {
	if sound.currentLength == 0 && !sound.playsContinuously {
		return 0, 0
	}
	var left, right int
	if sound.t > sound.getDutyCycle() {
		left, right = 32767, 32767
	} else {
		left, right = -32767, -32767
	}
	if !sound.leftSpeakerOn {
		left = 0
	}
	if !sound.rightSpeakerOn {
		right = 0
	}
	return left, right
}
func (sound *sound) getFreq() float64 {
	switch sound.soundType {
	case waveSoundType:
		return 65536.0 / float64(sound.freqReg+1)
	case noiseSoundType:
		r := float64(sound.polyDivRatio)
		if r == 0 {
			r = 0.5
		}
		// NOTE: where does polystep fit into this?
		twoShiftS := float64(uint(2) << uint(sound.polyShiftFreq))
		if twoShiftS == 0 {
			twoShiftS = 0.5
		}
		return 524288.0 / r / twoShiftS
	default:
		return 131072.0 / float64(sound.freqReg+1)
	}
}

func (sound *sound) writePolyCounterReg(val byte) {
	if val&0x08 != 0 {
		sound.polyStep = 7
	} else {
		sound.polyStep = 15
	}
	sound.polyShiftFreq = val >> 4
	sound.polyDivRatio = val & 0x07
}
func (sound *sound) readPolyCounterReg() byte {
	val := byte(0)
	if sound.polyStep == 7 {
		val |= 8
	}
	val |= sound.polyShiftFreq << 4
	val |= sound.polyDivRatio
	return val
}

func (sound *sound) writeWaveOutLvlReg(val byte) {
	sound.waveOutLvl = (val >> 5) & 0x03
}
func (sound *sound) readWaveOutLvlReg() byte {
	return (sound.waveOutLvl << 5) | 0x9f
}

func (sound *sound) writeLengthData(val byte) {
	switch sound.soundType {
	case waveSoundType:
		sound.lengthData = 256 - uint16(val)
	default:
		sound.lengthData = 64 - uint16(val)
	}
}
func (sound *sound) writeLenDutyReg(val byte) {
	sound.lengthData = uint16(val & 0x3f)
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
	if val&0x08 != 0 {
		sound.envelopeDirection = envUp
	} else {
		sound.envelopeDirection = envDown
	}
	sound.envelopeStartVal = val >> 4
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
}
func (sound *sound) readFreqLowReg() byte {
	return 0xff
}

func (sound *sound) writeFreqHighReg(val byte) {
	sound.restartRequested = val&0x80 != 0
	sound.playsContinuously = val&0x40 == 0
	sound.freqReg &^= 0xff00
	sound.freqReg |= uint16(val&0x03) << 8
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
