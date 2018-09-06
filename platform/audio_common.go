package platform

import (
	"time"
)

// AudioBuffer lets you play sound.
type AudioBuffer struct {
	SamplesPerSecond uint32
	BitsPerSample    uint32
	ChannelCount     uint32
	BlockSize        uint32
	BlockCount       uint32

	currentBlockIndex int

	blocks       []soundBlock
	
	output audioOutput

	closer chan bool
	writer chan int
}

type soundBlock struct {
	bytes []byte
	used  int
	busy bool
}

// OpenAudioBuffer creates and returns a new playing buffer
func OpenAudioBuffer(blockCount, blockSize, samplesPerSecond, bitsPerSample, channelCount uint32) (*AudioBuffer, error) {
	ab := AudioBuffer{
		SamplesPerSecond: samplesPerSecond,
		BitsPerSample:    bitsPerSample,
		ChannelCount:     channelCount,
		BlockCount: blockCount,
		BlockSize: blockSize,
		writer: make(chan int, blockCount+1),
		closer: make(chan bool),
	}
	ab.blocks = make([]soundBlock, blockCount)
	for i := range ab.blocks {
		ab.blocks[i].bytes = make([]byte, blockSize)
	}

	err := ab.output.init(&ab)
	if err != nil {
		return nil, err
	}

	go ab.writerLoop()

	return &ab, nil
}

func (ab *AudioBuffer) updateFreeBlockInfo() {
	for i := range ab.blocks {
		if ab.output.noLongerBusy(i) {
			ab.blocks[i].busy = false
		}
	}
}

func (ab *AudioBuffer) Write(data []byte) {
	ab.updateFreeBlockInfo()
	for len(data) > 0 {

		if ab.blocks[ab.currentBlockIndex].busy {
			ab.currentBlockIndex = ab.waitOnFreeBlock()
			ab.blocks[ab.currentBlockIndex].used = 0
		}

		block := &ab.blocks[ab.currentBlockIndex]

		spaceLeft := len(block.bytes) - block.used

		if len(data) < spaceLeft {
			copy(block.bytes[block.used:], data)
			block.used += len(data)
			break
		}
		copy(block.bytes[block.used:], data[:spaceLeft])
		data = data[spaceLeft:]

		block.busy = true

		// the api calls sometimes takes a few ms, so let's not wait on them
		ab.writer <- ab.currentBlockIndex
	}
}

// Close closes the buffer and releases all resourses.
// It waits for all queued buffer writes to finish playing first.
func (ab *AudioBuffer) Close() error {
	for ab.BufferAvailable() / int(ab.BlockSize) < len(ab.blocks) {
		ab.updateFreeBlockInfo()
		time.Sleep(5)
	}
	err := ab.output.close()
	if err != nil {
		return err
	}
	ab.closer <- true
	return nil
}

// TODO: timeout w/ err
func (ab *AudioBuffer) waitOnFreeBlock() int {
	for {
		ab.updateFreeBlockInfo()
		for i := range ab.blocks {
			if !ab.blocks[i].busy {
				return i
			}
		}
		time.Sleep(1)
	}
}

// BufferAvailable returns the number of bytes available
// to be filled in all the blocks not currently queued.
func (ab *AudioBuffer) BufferAvailable() int {
	available := 0
	ab.updateFreeBlockInfo()
	for i := range ab.blocks {
		if !ab.blocks[i].busy {
			block := &ab.blocks[i]
			if i == ab.currentBlockIndex {
				available += int(ab.BlockSize) - block.used
			} else {
				available += int(ab.BlockSize)
			}
		}
	}
	return available
}

func (ab *AudioBuffer) BufferSize() int {
	return int(ab.BlockCount * ab.BlockSize)
}

func (ab *AudioBuffer) writerLoop() {
	for {
		select {
		case i := <-ab.writer:
			block := &ab.blocks[i]
			ab.output.write(block.bytes, i)
		case <-ab.closer:
			return
		}
	}
}
