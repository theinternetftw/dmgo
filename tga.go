package dmgo

import (
	"fmt"
	"os"
)

func die(s string) {
	fmt.Fprintln(os.Stderr, s)
	os.Exit(1)
}

func writeTgaRGB(fname string, w int, h int, rgbData []byte) {
	if w*h*3 != len(rgbData) {
		fmt.Fprintf(os.Stderr, "writeTgaRGB(): bad sizes, %v*%v*3 != %v\n", w, h, len(rgbData))
	}

	outb := []byte{
		0, 0, 2, 0, 0, 0, 0, 0, // main hdr + color map info (unused)
		0, 0, 0, 0, // img origin
		byte(w), byte(w >> 8),
		byte(h), byte(h >> 8),
		24,   // 24 bbp
		0x20, // top to bottom, left to right ordering, no alpha
	}
	// switch to BGR, that it is mandated is the only bad thing about tga
	for i := 0; i < len(rgbData); i += 3 {
		outb = append(outb, rgbData[i+2])
		outb = append(outb, rgbData[i+1])
		outb = append(outb, rgbData[i])
	}

	os.WriteFile(fname, outb, 0777)
}

func writeTgaRGBA(fname string, w int, h int, rgbaData []byte) {
	if w*h*4 != len(rgbaData) {
		fmt.Fprintf(os.Stderr, "writeTgaRGBA(): bad sizes, %v*%v*4 != %v\n", w, h, len(rgbaData))
	}

	outb := []byte{
		0, 0, 2, 0, 0, 0, 0, 0, // main hdr + color map info (unused)
		0, 0, 0, 0, // img origin
		byte(w), byte(w >> 8),
		byte(h), byte(h >> 8),
		32,   // 24 bbp
		0x28, // top to bottom, left to right ordering, 8-bit alpha
	}
	// switch to BGRA, that it is mandated is the only bad thing about tga
	for i := 0; i < len(rgbaData); i += 4 {
		outb = append(outb, rgbaData[i+2])
		outb = append(outb, rgbaData[i+1])
		outb = append(outb, rgbaData[i])
		outb = append(outb, rgbaData[i+3])
	}

	os.WriteFile(fname, outb, 0777)
}
