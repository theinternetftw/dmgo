package platform

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	whdrDone      = 0x01
	whdrPrepared  = 0x02
	whdrBeginloop = 0x04
	whdrEndloop   = 0x08
	whdrInqueue   = 0x10

	mmSysErrNoErr = 0x00

	waveFormatPCM = 0x0001

	waveMapper = 0xffffffff

	womOpen  = 0x3bb
	womClose = 0x3bc
	womDone  = 0x3bd
	wimOpen  = 0x3be
	wimClose = 0x3bf
	wimDone  = 0x3c0

	callbackNull = 0x00000
)

// AudioBuffer represents all you need to play sound.
type AudioBuffer struct {
	SamplesPerSecond uint32
	BitsPerSample    uint32
	ChannelCount     uint32
	BlockSize        uint32
	BlockCount       uint32

	currentBlock *soundBlock
	blocks       []soundBlock
	hWaveOut     uintptr
	closer chan bool
	writer chan *soundBlock
}

type soundBlock struct {
	wavehdr
	bytes []byte
	used  int
}

type wavehdr struct {
	lpData          uintptr
	dwBufferLength  uint32
	dwBytesRecorded uint32
	dwUser          uintptr
	dwFlags         uint32
	dwLoops         uint32
	lpNext          uintptr
	reserved        uintptr
}

type waveFormatEx struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

func init() {
	winmm := windows.MustLoadDLL("Winmm.dll")
	waveOutPrepareHeader = winmm.MustFindProc("waveOutPrepareHeader")
	waveOutWrite = winmm.MustFindProc("waveOutWrite")
	waveOutOpen = winmm.MustFindProc("waveOutOpen")
	waveOutClose = winmm.MustFindProc("waveOutClose")
	waveOutUnprepareHeader = winmm.MustFindProc("waveOutUnprepareHeader")
}

// OpenAudioBuffer creates and returns a new playing buffer
func OpenAudioBuffer(blockCount, blockSize, samplesPerSecond, bitsPerSample, channelCount uint32) (*AudioBuffer, error) {
	ab := AudioBuffer{
		SamplesPerSecond: samplesPerSecond,
		BitsPerSample:    bitsPerSample,
		ChannelCount:     channelCount,
		BlockCount: blockCount,
		BlockSize: blockSize,
		writer: make(chan *soundBlock, blockCount+1),
		closer: make(chan bool),
	}
	ab.blocks = make([]soundBlock, blockCount)
	for i := range ab.blocks {
		ab.blocks[i].bytes = make([]byte, blockSize)
		ab.blocks[i].dwFlags = whdrDone
	}
	ab.currentBlock = &ab.blocks[0]

	wfx := waveFormatEx{
		wFormatTag:     waveFormatPCM,
		nSamplesPerSec: ab.SamplesPerSecond,
		wBitsPerSample: uint16(ab.BitsPerSample),
		nChannels:      uint16(ab.ChannelCount),
		nBlockAlign:    uint16(ab.BitsPerSample * ab.ChannelCount / 8),
		cbSize:         0,
	}
	wfx.nAvgBytesPerSec = uint32(wfx.nBlockAlign) * wfx.nSamplesPerSec

	if r1, r2, lastErr := waveOutOpen.Call(
		uintptr(unsafe.Pointer(&ab.hWaveOut)),
		waveMapper, uintptr(unsafe.Pointer(&wfx)),
		uintptr(0), uintptr(0), callbackNull); r1 != mmSysErrNoErr {
		return nil, fmt.Errorf("waveOutOpen error: %v, %v, %v", r1, r2, lastErr)
	}

	go ab.writerLoop()

	return &ab, nil
}

// Close closes the buffer and releases all resourses.
// It waits for all queued buffer writes to finish playing first.
func (ab *AudioBuffer) Close() error {
	for ab.BufferAvailable() / int(ab.BlockSize) < len(ab.blocks) {
		time.Sleep(5)
	}
	for i := range ab.blocks {
		block := &ab.blocks[i]
		r1, r2, lastErr := waveOutUnprepareHeader.Call(
			ab.hWaveOut, uintptr(unsafe.Pointer(&block.wavehdr)), unsafe.Sizeof(block.wavehdr))
		if r1 != 0 {
			// NOTE: try to keep going instead?
			return fmt.Errorf("waveOutUnprepareHeader error: %v, %v, %v", r1, r2, lastErr)
		}
	}
	r1, r2, lastErr := waveOutClose.Call(ab.hWaveOut)
	if r1 != 0 {
		return fmt.Errorf("waveOutClose error: %v, %v, %v", r1, r2, lastErr)
	}
	ab.closer <- true
	return nil
}

var (
	waveOutPrepareHeader   *windows.Proc
	waveOutUnprepareHeader *windows.Proc
	waveOutWrite           *windows.Proc
	waveOutOpen            *windows.Proc
	waveOutClose           *windows.Proc
)

// TODO: timeout w/ err
func (ab *AudioBuffer) waitOnFreeBlock() *soundBlock {
	for {
		for i := range ab.blocks {
			if ab.blocks[i].dwFlags&whdrDone != 0 {
				return &ab.blocks[i]
			}
		}
		time.Sleep(1)
	}
}

// BufferAvailable returns the number of bytes available
// to be filled in all the blocks not currently queued.
func (ab *AudioBuffer) BufferAvailable() int {
	available := 0
	for i := range ab.blocks {
		if ab.blocks[i].dwFlags&whdrDone != 0 {
			block := &ab.blocks[i]
			if block == ab.currentBlock {
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
		case block := <-ab.writer:

			block.lpData = uintptr(unsafe.Pointer(&block.bytes[0]))

			r1, r2, lastErr := waveOutPrepareHeader.Call(
				ab.hWaveOut, uintptr(unsafe.Pointer(&block.wavehdr)), unsafe.Sizeof(block.wavehdr))
			if r1 != 0 {
				fmt.Printf("waveOutPrepareHeader error: %v, %v, %v", r1, r2, lastErr)
			}

			r1, r2, lastErr = waveOutWrite.Call(
				ab.hWaveOut, uintptr(unsafe.Pointer(&block.wavehdr)), unsafe.Sizeof(block.wavehdr))
			if r1 != 0 {
				fmt.Printf("waveOutWrite error: %v, %v, %v", r1, r2, lastErr)
			}
		case <-ab.closer:
			return
		}
	}
}

func (ab *AudioBuffer) Write(data []byte) {
	for len(data) > 0 {

		if ab.currentBlock.dwFlags & whdrDone == 0 {
			ab.currentBlock = ab.waitOnFreeBlock()
			ab.currentBlock.used = 0
		}

		block := ab.currentBlock

		spaceLeft := len(block.bytes) - block.used

		if len(data) < spaceLeft {
			copy(block.bytes[block.used:], data)
			block.used += len(data)
			break
		}
		copy(block.bytes[block.used:], data[:spaceLeft])
		block.dwBufferLength = uint32(len(block.bytes))
		data = data[spaceLeft:]

		block.dwFlags &^= whdrDone

		// the api calls sometimes takes a few ms, so let's not wait on them
		ab.writer <- block
	}
}
