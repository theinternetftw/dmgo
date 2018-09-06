package platform

import (
	"image"
	"image/color"
	"sync"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// WindowState contains what the window loop and program proper both need to touch
type WindowState struct {
	// the Width of the framebuffer
	Width int
	// the Height of the framebuffer
	Height int

	// a Mutex that must be held when reading or writing in WindowState
	Mutex sync.Mutex

	// Pix is the raw RGBA bytes of the framebuffer
	Pix []byte

	keyCodeArray [256]bool
	keyCodeMap map[key.Code]bool
	keyCharArray [256]bool
	keyCharMap map[rune]bool

	eventQueue screen.EventDeque
	drawRequested bool
}

// CopyKeyCharArray writes the current ascii keystate to dest
func (s *WindowState) CopyKeyCharArray(dest []bool) {
	copy(dest, s.keyCharArray[:])
}

// CharIsDown returns the key state for that char
func (s *WindowState) CharIsDown(c rune) bool {
	if c >= 0 && c < 256 {
		return s.keyCharArray[byte(c)]
	}
	return s.keyCharMap[c]
}
// CodeIsDown returns the key state for that keyCode
func (s *WindowState) CodeIsDown(c key.Code) bool {
	if c < 256 {
		return s.keyCodeArray[byte(c)]
	}
	return s.keyCodeMap[c]
}

func (s *WindowState) updateKeyboardState(e key.Event) {
	setVal := e.Direction == key.DirPress
	if setVal || e.Direction == key.DirRelease {
		if e.Code < 256 {
			s.keyCodeArray[byte(e.Code)] = setVal
		} else {
			s.keyCodeMap[e.Code] = setVal
		}
		if e.Rune >= 0 && e.Rune < 256 {
			s.keyCharArray[byte(e.Rune)] = setVal
		} else {
			s.keyCharMap[e.Rune] = setVal
		}
	}
}

// RequestDraw puts a draw request on the window loop queue
// It is assumed the mutex is already held when this function is called.
func (s *WindowState) RequestDraw() {
	if !s.drawRequested {
		s.eventQueue.Send(drawRequest{})
		s.drawRequested = true
	}
}

type drawRequest struct {}

// InitDisplayLoop creates a window and starts event loop
func InitDisplayLoop(title string, windowWidth, windowHeight, frameWidth, frameHeight int, updateLoop func(*WindowState)) {
	driver.Main(func (s screen.Screen) {

		w, err := s.NewWindow(&screen.NewWindowOptions{windowWidth, windowHeight, title})
		if err != nil {
			panic(err)
		}
		defer w.Release()

		buf, err := s.NewBuffer(image.Point{frameWidth, frameHeight})
		if err != nil {
			panic(err)
		}
		tex, err := s.NewTexture(image.Point{frameWidth, frameHeight})
		if err != nil {
			panic(err)
		}

		windowState := WindowState{
			Width: frameWidth,
			Height: frameHeight,
			Pix: make([]byte, 4*frameWidth*frameHeight),
			eventQueue: w,
			keyCodeMap: map[key.Code]bool{},
			keyCharMap: map[rune]bool{},
		}

		go updateLoop(&windowState)

		szRect := buf.Bounds()
		needFullRepaint := true

		for {
			publish := false

			switch e := w.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}

			case key.Event:
				windowState.Mutex.Lock()
				windowState.updateKeyboardState(e)
				windowState.Mutex.Unlock()

			case drawRequest:
				windowState.Mutex.Lock()
				copy(buf.RGBA().Pix, windowState.Pix)
				tex.Upload(image.Point{0, 0}, buf, buf.Bounds())
				windowState.drawRequested = false
				windowState.Mutex.Unlock()
				publish = true

			case size.Event:
				szRect = e.Bounds()

			case paint.Event:
				needFullRepaint = true
				publish = true
			}

			if publish {
				scaleFactX := float64(szRect.Max.X) / float64(tex.Bounds().Max.X)
				scaleFactY := float64(szRect.Max.Y) / float64(tex.Bounds().Max.Y)
				scaleFact := scaleFactX
				if scaleFactY < scaleFact {
					scaleFact = scaleFactY
				}
				// NOTE: flicker happens when scale is not an integer
				scaleFact = float64(int(scaleFact))
				newWidth := int(scaleFact * float64(tex.Bounds().Max.X))
				centerX := float64(szRect.Max.X/2 - newWidth/2)
				src2dst := f64.Aff3 {
					float64(int(scaleFact)), 0, centerX,
					0, float64(int(scaleFact)), 0,
				}
				identTrans := f64.Aff3 {
					1, 0, 0,
					0, 1, 0,
				}
				// get flicker when we do two draws all the time, so
				// only do it when we resize or get moved on/offscreen
				if needFullRepaint {
					w.DrawUniform(identTrans, color.Black, szRect, screen.Src, nil)
					needFullRepaint = false
				}
				w.Draw(src2dst, tex, tex.Bounds(), screen.Src, nil)
				w.Publish()
			}
		}
	})
}
