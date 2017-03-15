package zeroui

import (
	"fmt"
	"github.com/banthar/Go-SDL/sdl"
	"log"
	"os"
	"time"
)

var (
	ErrBadColorFormat = fmt.Errorf("color value must be in form \"rrggbb\" (hex)")
)

type Screen struct {
	Widgets  []Widget
	OnRender func(*Screen)
}

func (this *Screen) Init() (err error) {
	for w := range this.Widgets {
		err = this.Widgets[w].Init()
		if err != nil {
			break
		}
	}
	return err
}

type UI struct {
	Width        int
	Height       int
	BitsPerPixel int

	screen  *sdl.Surface
	screens chan func()

	done []chan bool
}

func (this *UI) Init() {
	log.Print("sdl.Init")

	if sdl.Init(sdl.INIT_VIDEO) < 0 {
		panic(sdl.GetError())
	}

	log.Print("sdl.SetVideoMode")
	this.screen = sdl.SetVideoMode(this.Width, this.Height, this.BitsPerPixel, sdl.ANYFORMAT)
	if this.screen == nil {
		panic(sdl.GetError())
	}

	log.Print("sdl.ShowCursor")
	if sdl.ShowCursor(sdl.DISABLE) < 0 {
		panic(sdl.GetError())
	}

	this.screens = make(chan func())
}

func (this *UI) Run() {
	defer func() {
		sdl.Quit()
		os.Exit(1)
	}()
	for screen := range this.screens {
		screen()
		this.screen.Flip()
	}
}

func (this *UI) Destroy() {
	s := this.screens
	if s != nil {
		this.screens = nil
		close(s)
	}
}

func (this *UI) runLater(f func()) {
	if this.screens != nil {
		this.screens <- f
	}
}

func (this *UI) renderAll(widgets []Widget) {
	for _, w := range widgets {
		switch ww := w.(type) {
		case *Animation:
			go func() {
				ticker := time.NewTicker(ww.Delay())
				done := make(chan bool)
				this.done = append(this.done, done)
				for {
					select {
					case <-ticker.C:
						this.runLater(func() {
							ww.Render(this.screen)
						})

					case <-done:
						//log.Print("Wait.done: ", str)
						ticker.Stop()
						done <- false
						return
					}
				}
			}()

		default:
			ww.Render(this.screen)
		}
	}
}

/*
func (this *UI) PromptScreen() {
	this.ScreenDone()
	this.showScreen(func() {
		this.renderAll(this.Prompt)
	})
}
*/

func (this *UI) ScreenDone() {
	for _, done := range this.done {
		if done != nil {
			for {
				done <- true
				if (<-done) == false {
					break
				}
			}
			close(done)
		}
	}
	this.done = make([]chan bool, 1)
}

func (this *UI) ShowScreen(s *Screen) {
	this.ScreenDone()
	this.runLater(func() {
		if s.OnRender != nil {
			s.OnRender(s)
		}
		this.renderAll(s.Widgets)
	})
}
