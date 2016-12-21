package dmgo

type apu struct {
	allSoundsOn bool

	sounds [4]sound

	wavePatternRAM [16]byte

	// cart chip sounds. never used by any game?
	vInToLeftSpeaker  bool
	vInToRightSpeaker bool

	rightSpeakerVolume byte // right=S01 in docs
	leftSpeakerVolume  byte // left=S02 in docs
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

type sound struct {
	on             bool
	rightSpeakerOn bool // S01 in docs
	leftSpeakerOn  bool // S02 in docs

	envelopeDirection envDir
	envelopeStartVal  byte
	envelopeSweepVal  byte

	sweepDirection sweepDir
	sweepTime      byte
	sweepShift     byte

	playsContinuously bool
	restartRequested  bool

	freqReg uint16
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
