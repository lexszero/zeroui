package zeroui

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/banthar/Go-SDL/sdl"
	"github.com/banthar/Go-SDL/ttf"
	"image"
	"image/draw"
	"image/gif"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Widget interface {
	Render(*sdl.Surface)
	Init() error
}

type wAnyWidget struct {
	TextBox   *TextBox   `json:",omitempty"`
	Image     *Image     `json:",omitempty"`
	Animation *Animation `json:",omitempty"`
}

type AnyWidget struct {
	Widget
}

func (this *AnyWidget) UnmarshalJSON(buf []byte) (err error) {
	t := wAnyWidget{}
	if err = json.Unmarshal(buf, &t); err != nil {
		return
	}
	switch {
	case t.TextBox != nil:
		this.Widget = t.TextBox
	case t.Image != nil:
		this.Widget = t.Image
	case t.Animation != nil:
		this.Widget = t.Animation
	}
	return nil
}

type Color struct {
	R, G, B uint8
}

func (this Color) String() string {
	return fmt.Sprintf("#%06x", this.uint32rgb())
}

func (this *Color) uint32rgb() uint32 {
	return uint32(this.R)<<16 | uint32(this.G)<<8 | uint32(this.B)
}

func (this Color) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%06x\"", this.uint32rgb())), nil
}

func (this *Color) UnmarshalJSON(b []byte) error {
	if len(b) != 8 {
		return ErrBadColorFormat
	}
	v, err := strconv.ParseUint(string(b[1:7]), 16, 32)
	if err != nil {
		return ErrBadColorFormat
	}
	*this = Color{
		R: uint8((v >> 16) & 0xFF),
		G: uint8((v >> 8) & 0xFF),
		B: uint8(v & 0xFF),
	}
	return nil
}

func (this *Color) SDL() sdl.Color {
	return sdl.Color{
		R: this.R,
		G: this.G,
		B: this.B,
	}
}

func (this *Color) MapRGB(pixfmt *sdl.PixelFormat) uint32 {
	return sdl.MapRGB(pixfmt, this.R, this.G, this.B)
}

type wTextBox struct {
	sdl.Rect
	Color    Color
	BgColor  Color
	FontName string
	FontSize int
	Interval int16
	Center   bool
	Text     string

	y    int16
	font *ttf.Font
}

type TextBox wTextBox

func (this *TextBox) Init() error {
	log.Printf("loading font %s, size=%v", this.FontName, this.FontSize)
	if ttf.WasInit() == 0 {
		if ttf.Init() != 0 {
			return errors.New("ttf.Init")
		}
	}
	this.font = ttf.OpenFont(this.FontName, this.FontSize)
	if this.font == nil {
		return errors.New(fmt.Sprintf("ttf.OpenFont %s %v", this.FontName, this.FontSize))
	}
	return nil
}

func (this *TextBox) UnmarshalJSON(buf []byte) (err error) {
	t := wTextBox{}
	if err = json.Unmarshal(buf, &t); err != nil {
		return
	}
	*this = TextBox(t)
	return this.Init()
}

func (this *TextBox) renderString(str string) *sdl.Surface {
	text := ttf.RenderUTF8_Shaded(this.font, str, this.Color.SDL(), this.BgColor.SDL())
	if text == nil {
		panic(sdl.GetError())
	}
	return text
}

func (this *TextBox) wordWrap(text string, overflow func(string) bool) (lines []string) {
	line := ""
	for _, word := range strings.Split(text, " ") {
		for {
			n := strings.IndexRune(line, '\n')
			if n < 0 {
				break
			}
			lines = append(lines, line[:n])
			line = line[n+1:]
		}

		oldLine := line
		line += word + " "

		if overflow(line) {
			lines = append(lines, oldLine)
			line = word + " "
		}
	}
	return append(lines, line)
}

func (this *TextBox) Render(s *sdl.Surface) {
	this.y = this.Y

	for _, line := range this.wordWrap(this.Text, func(s string) bool {
		text := this.renderString(s)
		defer text.Free()
		return text.Clip_rect.W >= this.W
	}) {
		text := this.renderString(line)
		defer text.Free()
		rect := text.Clip_rect
		srect := rect
		rect.X = this.X
		if this.Center {
			w := (int16(this.W) - int16(rect.W)) / 2
			if w < 0 {
				srect.X -= w
				srect.W += uint16(-w)
				rect.W = this.W
			} else {
				rect.X += w
			}
		} else {
			if rect.W > this.W {
				rect.W = this.W
				srect.W = this.W
			}
		}
		if this.y-this.Y > int16(this.H) {
			continue
		}
		rect.Y = this.y
		if s.Blit(&rect, text, &srect) < 0 {
			panic(sdl.GetError())
		}
		this.y += int16(rect.H) + this.Interval
	}
}

type wImage struct {
	File string
	X, Y int16

	surface *sdl.Surface
}

type Image wImage

func (this *Image) Init() error {
	log.Print("loading image ", this.File)
	f, err := os.Open(this.File)
	if err != nil {
		return err
	}
	defer f.Close()

	this.surface = sdl.Load(this.File)
	if this.surface == nil {
		return fmt.Errorf(sdl.GetError())
	}

	return nil
}

func (this *Image) UnmarshalJSON(buf []byte) error {
	t := wImage{}
	if err := json.Unmarshal(buf, &t); err != nil {
		return err
	}
	*this = Image(t)
	return this.Init()
}

func (this *Image) Render(s *sdl.Surface) {
	rect := this.surface.Clip_rect
	rect.X += this.X
	rect.Y += this.Y
	if s.Blit(&rect, this.surface, nil) < 0 {
		panic(sdl.GetError())
	}
}

type wAnimation struct {
	sdl.Rect
	File string

	sprite   *sdl.Surface
	delays   []time.Duration
	frameNum int16
}

type Animation wAnimation

func (this *Animation) Init() error {
	log.Print("loading animation ", this.File)
	f, err := os.Open(this.File)
	if err != nil {
		return err
	}
	defer f.Close()

	var img *gif.GIF
	img, err = gif.DecodeAll(f)
	if err != nil {
		return err
	}

	this.delays = make([]time.Duration, len(img.Image))

	rect := img.Image[0].Bounds()
	w, h := rect.Dx(), rect.Dy()
	this.W = uint16(w)
	this.H = uint16(h)

	sprite := image.NewRGBA(image.Rect(0, 0, w, h*len(img.Image)))
	for i, frame := range img.Image {
		drect := image.Rect(0, h*i, w, h*(i+1))
		//log.Printf("put frame #%v at %#v", i, drect)
		draw.Draw(sprite, drect, frame, image.ZP, draw.Src)
		this.delays[i] = time.Duration(img.Delay[i]*10) * time.Millisecond
	}
	this.sprite = sdl.CreateSurfaceFromImage(sprite)

	return nil
}

func (this *Animation) UnmarshalJSON(buf []byte) error {
	t := wAnimation{}
	if err := json.Unmarshal(buf, &t); err != nil {
		return err
	}
	*this = Animation(t)
	return this.Init()
}

func (this *Animation) Render(s *sdl.Surface) {
	srect := sdl.Rect{0, int16(this.H) * this.frameNum, this.W, this.H}
	if s.Blit(&this.Rect, this.sprite, &srect) < 0 {
		panic(sdl.GetError())
	}
	this.frameNum++
	if int(this.frameNum) >= len(this.delays) {
		this.frameNum = 0
	}
}

func (this *Animation) Delay() time.Duration {
	return this.delays[this.frameNum]
}
