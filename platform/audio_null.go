// +build !windows

package platform

import "time"

// AudioBuffer represents a fake audio implementation
type AudioBuffer struct {
	SamplesPerSecond uint32
	BitsPerSample    uint32
	ChannelCount     uint32
	BlockSize        uint32
	BlockCount       uint32

	fakeBufferConsumed int
	lastAvailCheck time.Time
}

// OpenAudioBuffer creates and returns a fake playing buffer.
func OpenAudioBuffer(blockCount, blockSize, samplesPerSecond, bitsPerSample, channelCount uint32) (*AudioBuffer, error) {
	ab := AudioBuffer{
		SamplesPerSecond: samplesPerSecond,
		BitsPerSample:    bitsPerSample,
		ChannelCount:     channelCount,
		BlockCount: blockCount,
		BlockSize: blockSize,
		lastAvailCheck: time.Now(),
	}
	return &ab, nil
}

// Close closes the fake buffer
func (ab *AudioBuffer) Close() error {
	return nil
}

// BufferAvailable returns a reasonable value for an available (fake) buffer
func (ab *AudioBuffer) BufferAvailable() int {
	samplesPlayed := int(time.Now().Sub(ab.lastAvailCheck).Seconds() * float64(ab.SamplesPerSecond))
	bytesWritten := samplesPlayed * int(ab.ChannelCount) * int(ab.BitsPerSample / 8)
	ab.fakeBufferConsumed -= bytesWritten
	if ab.fakeBufferConsumed < 0 {
		ab.fakeBufferConsumed = 0
	}
	ab.lastAvailCheck = time.Now()
	return ab.BufferSize() - ab.fakeBufferConsumed
}

// BufferSize returns a reasonable size for the fake buffer
func (ab *AudioBuffer) BufferSize() int {
	return int(ab.BlockSize*ab.BlockCount)
}

// Write pretends to write to a fake buffer
func (ab *AudioBuffer) Write(data []byte) error {
	ab.fakeBufferConsumed += len(data)
	if ab.fakeBufferConsumed > ab.BufferSize() {
		ab.fakeBufferConsumed = ab.BufferSize()
	}
	return nil
}
