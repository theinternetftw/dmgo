// +build !windows

package platform

// AudioBuffer represents a fake audio implementation
type AudioBuffer struct {
	BlockCount uint32
	BlockSize uint32
}

// OpenAudioBuffer creates and returns a fake playing buffer.
func OpenAudioBuffer(blockCount, blockSize, samplesPerSecond, bitsPerSample, channelCount uint32) (*AudioBuffer, error) {
	ab := AudioBuffer{
		BlockCount: blockCount,
		BlockSize: blockSize,
	}
	return &ab, nil
}

// Close closes the fake buffer
func (ab *AudioBuffer) Close() error {
	return nil
}

// BufferAvailable returns a reasonable value for an available (fake) buffer
func (ab *AudioBuffer) BufferAvailable() int {
	return ab.BufferSize() / 2
}

// BufferSize returns a reasonable size for the fake buffer
func (ab *AudioBuffer) BufferSize() int {
	return int(ab.BlockSize*ab.BlockCount)
}

// Write pretends to write to a fake buffer
func (ab *AudioBuffer) Write(data []byte) error {
	return nil
}
