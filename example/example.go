package main

import (
	"fmt"
	"github.com/banthar/Go-SDL/sdl"
	ui "github.com/lexszero/zeroui"
	"log"
	"time"
)

var (
	counter = 0

	ColorBlack = ui.Color{0, 0, 0}
	ColorWhite = ui.Color{255, 255, 255}

	u = ui.UI{
		Width:        320,
		Height:       240,
		BitsPerPixel: 16,
	}

	screen = ui.Screen{
		Widgets: []ui.Widget{
			&ui.TextBox{
				Rect: sdl.Rect{
					X: 10,
					Y: 60,
					W: 300,
					H: 40,
				},
				Color:    ColorBlack,
				BgColor:  ColorWhite,
				FontName: "DejaVuSans.ttf",
				FontSize: 20,
				Interval: 10,
				Center:   true,
				Text:     "Hello",
			},
			&ui.Animation{
				File: "spinner.gif",
				Rect: sdl.Rect{
					X: 116,
					Y: 120,
				},
			},
		},
		OnRender: func(s *ui.Screen) {
			log.Println("OnRender called")
			s.Widgets[0].(*ui.TextBox).Text = fmt.Sprintf("Hello %v", counter)
		},
	}
)

func main() {
	u.Init()
	go u.Run()

	err := screen.Init()
	if err != nil {
		log.Fatal("screen.Init failed: ", err)
	}

	for range time.Tick(time.Second) {
		counter++
		u.ShowScreen(&screen)
	}
}
