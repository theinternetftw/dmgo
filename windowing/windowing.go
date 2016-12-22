package windowing

import (
	"image"
	"image/color"
	"sync"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/lifecycle"
//	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// SharedState contains what the window loop and program proper both need to touch
type SharedState struct {
	// the Width of the framebuffer
	Width int
	// the Height of the framebuffer
	Height int

	// a Mutex that must be held when reading or writing in SharedState
	Mutex sync.Mutex

	// Pix is the raw RGBA bytes of the framebuffer
	Pix []byte

	eventQueue screen.EventDeque
	drawRequested bool
	// e.g. keyboard/mouse goes here
}

// RequestDraw puts a draw request on the window loop queue
// It is assumed the mutex is already held when this function is called.
func (s *SharedState) RequestDraw() {
	if !s.drawRequested {
		s.eventQueue.Send(drawRequest{})
		s.drawRequested = true
	}
}

type drawRequest struct {}

// InitDisplayLoop creates a window and starts event loop
func InitDisplayLoop(windowWidth, windowHeight, frameWidth, frameHeight int, updateLoop func(*SharedState)) {
	driver.Main(func (s screen.Screen) {

		w, err := s.NewWindow(&screen.NewWindowOptions{windowWidth, windowHeight})
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

		sharedState := SharedState{
			Width: frameWidth,
			Height: frameHeight,
			Pix: make([]byte, 4*frameWidth*frameHeight),
			eventQueue: w,
		}

		go updateLoop(&sharedState)

		szRect := buf.Bounds()
		justResized := true

		for {
			publish := false

			switch e := w.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case drawRequest:
				sharedState.Mutex.Lock()
				copy(buf.RGBA().Pix, sharedState.Pix)
				tex.Upload(image.Point{0, 0}, buf, buf.Bounds())
				sharedState.drawRequested = false
				sharedState.Mutex.Unlock()

				publish = true
			case size.Event:
				szRect = e.Bounds()
				justResized = true
			case paint.Event:
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
				// get flicker when we do two draws, so
				// only do it when we resize
				if justResized {
					w.DrawUniform(identTrans, color.Black, szRect, screen.Src, nil)
					justResized = false
				}
				w.Draw(src2dst, tex, tex.Bounds(), screen.Src, nil)
				w.Publish()
			}
		}
	})
}
