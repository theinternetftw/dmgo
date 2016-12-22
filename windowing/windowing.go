package windowing

import (
	"image"
	"sync"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
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
			case paint.Event:
				publish = true
			}

			if publish {
				w.Scale(szRect, tex, tex.Bounds(), screen.Src, nil)
				// NOTE: due to weird opengl problems, can run w.Scale only once, or will get flicker!
				// So aspect ratio preserving scaling will have to be done like this:
				// * make a 2nd texture in the beginning, "borderedTexture"
				// * on size events update that bordered texture to match the aspect ratio of
				//   the window, but keep the smaller dimension the same size as the framebuffer
				// * fill the borderedTexture with black upon creation/resize
				// * upload the framebuffer, centered, into the texture
				// * then run Scale once, passing in the borderedTexture
				w.Publish()
			}
		}
	})
}
