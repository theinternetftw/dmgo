package platform

import (
	"fmt"

	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	whdrDone      = 0x01
	mmSysErrNoErr = 0x00

	waveFormatPCM = 0x0001
	waveMapper = 0xffffffff
	callbackNull = 0x00000
)

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

var (
	waveOutPrepareHeader   *windows.Proc
	waveOutUnprepareHeader *windows.Proc
	waveOutWrite           *windows.Proc
	waveOutOpen            *windows.Proc
	waveOutClose           *windows.Proc
)

func init() {
	winmm := windows.MustLoadDLL("Winmm.dll")
	waveOutPrepareHeader = winmm.MustFindProc("waveOutPrepareHeader")
	waveOutWrite = winmm.MustFindProc("waveOutWrite")
	waveOutOpen = winmm.MustFindProc("waveOutOpen")
	waveOutClose = winmm.MustFindProc("waveOutClose")
	waveOutUnprepareHeader = winmm.MustFindProc("waveOutUnprepareHeader")
}

// NOTE: Right now this is pretty much designed around
// the "submit blocks" style of audio api.
//
// Transition to circular buffer?
//
// For "submit blocks" underlying implementations
// you'd just move the cursor forward however many
// blocks are ready, copy that data and submit it.

type audioOutput struct {
	ab *AudioBuffer

	hdrs []wavehdr
	hWaveOut     uintptr
}

func (ao *audioOutput) init(ab *AudioBuffer) error {
	ao.ab = ab

	ao.hdrs = make([]wavehdr, len(ab.blocks))
	for i := range ao.hdrs {
		ao.hdrs[i].dwFlags = whdrDone
	}

	wfx := waveFormatEx{
		wFormatTag:     waveFormatPCM,
		nSamplesPerSec: ab.SamplesPerSecond,
		wBitsPerSample: uint16(ab.BitsPerSample),
		nChannels:      uint16(ab.ChannelCount),
		nBlockAlign:    uint16(ab.BitsPerSample * ab.ChannelCount / 8),
		cbSize:         0,
	}
	wfx.nAvgBytesPerSec = uint32(wfx.nBlockAlign) * wfx.nSamplesPerSec

	r1, r2, lastErr := waveOutOpen.Call(
		uintptr(unsafe.Pointer(&ao.hWaveOut)),
		waveMapper, uintptr(unsafe.Pointer(&wfx)),
		uintptr(0), uintptr(0), callbackNull)
	if r1 != mmSysErrNoErr {
		return fmt.Errorf("waveOutOpen error: %v, %v, %v", r1, r2, lastErr)
	}

	return nil
}

func (ao *audioOutput) close() error {
	for i := range ao.hdrs {
		hdr := &ao.hdrs[i]
		r1, r2, lastErr := waveOutUnprepareHeader.Call(
			ao.hWaveOut, uintptr(unsafe.Pointer(hdr)), unsafe.Sizeof(*hdr))
		if r1 != 0 {
			// NOTE: try to keep going instead?
			return fmt.Errorf("waveOutUnprepareHeader error: %v, %v, %v", r1, r2, lastErr)
		}
	}
	r1, r2, lastErr := waveOutClose.Call(ao.hWaveOut)
	if r1 != 0 {
		return fmt.Errorf("waveOutClose error: %v, %v, %v", r1, r2, lastErr)
	}
	return nil
}

func (ao *audioOutput) noLongerBusy(blockIndex int) bool {
	hdr := &ao.hdrs[blockIndex]
	if hdr.dwFlags&whdrDone != 0 {
		hdr.dwFlags &^= whdrDone
		return true
	}
	return false
}

func (ao *audioOutput) write(bytes []byte, i int) {
	hdr := &ao.hdrs[i]

	hdr.lpData = uintptr(unsafe.Pointer(&bytes[0]))
	hdr.dwBufferLength = uint32(len(bytes))

	r1, r2, lastErr := waveOutPrepareHeader.Call(
		ao.hWaveOut, uintptr(unsafe.Pointer(hdr)), unsafe.Sizeof(*hdr))
	if r1 != 0 {
		fmt.Printf("waveOutPrepareHeader error: %v, %v, %v", r1, r2, lastErr)
	}

	r1, r2, lastErr = waveOutWrite.Call(
		ao.hWaveOut, uintptr(unsafe.Pointer(hdr)), unsafe.Sizeof(*hdr))
	if r1 != 0 {
		fmt.Printf("waveOutWrite error: %v, %v, %v", r1, r2, lastErr)
	}
}
