package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"time"

	"github.com/as/frame"
	"github.com/as/memdraw"
	"github.com/as/shiny/event/key"
	"github.com/as/shiny/event/lifecycle"
	"github.com/as/shiny/event/mouse"
	"github.com/as/shiny/screen"
	"github.com/as/ui"
)

var (
	size   = image.Pt(2560, 1440)
	w      = dev.Window()
	dev, _ = ui.Init(&ui.Config{
		Width: size.X, Height: size.Y,
		Title: "harness",
	})
	img, _ = dev.NewBuffer(size)
	fb     = img.RGBA()
	down   uint

)

var (
	xvga  = image.Rect(0, 0, 640, 480)
	x720  = image.Rect(0, 0, 1280, 720)
	x1080 = image.Rect(0, 0, 1920, 1080)
)

var (
	black  = image.Black
	red    = image.NewUniform(color.RGBA{255, 0, 0, 255})
	green  = image.NewUniform(color.RGBA{0, 255, 0, 255})
	blue   = image.NewUniform(color.RGBA{0, 0, 255, 255})
	yellow = image.NewUniform(color.RGBA{255, 255, 0, 255})
	white  = image.White
)

var fr = [...]*frame.Frame{
	frame.New(fb, image.Rect(5, 5,             1024, 5+50), &frame.Config{Color: frame.Color{Palette: frame.Palette{Text: red, Back: black}}}),
	frame.New(fb, image.Rect(5, 5+50,       1024, 5+50+50), &frame.Config{Color: frame.Color{Palette: frame.Palette{Text: yellow, Back: black}}}),
	frame.New(fb, image.Rect(5, 5+50+50, 1024, 5+50+50+50), &frame.Config{Color: frame.Color{Palette: frame.Palette{Text: green, Back: black}}}),
}

var hz = time.NewTicker(time.Second / 128).C

var names = [3]string{
	"source",
	"ffmpeg",
	"zcrop",
}
var world = [3]image.Rectangle{
	x1080, 
	x720,  
	x720,
}

// zcrop takes the source s and crop parameter c
// and outputs a weighted crop paramter according
// to the original source's aspect ratio
func zcrop(s, c image.Rectangle) image.Rectangle {
	c = c.Intersect(s) // fix bounds on bad crops

	// obtain the center of the crop and save it
	// then translate the crop to (0,0)
	cc := center(c)
	c = c.Sub(c.Min)

	// compute the source's aspect ratio
	// in integral units and compute the
	// number of units cropped off, rounding
	// up to the nearest whole
	ar := aspect(s)
	d := delta(ar, s, c)

	// pick the largest cut on either the x or y axis
	u := max(d.X, d.Y)

	// set the new crop rectangle to the number of pixels
	// those units represent in the source
	c.Max.X = s.Max.X - u*ar.X
	c.Max.Y = s.Max.Y - u*ar.Y

	// translate the new crop to the center of where the
	// old one used to be
	return c.Add(cc.Sub(center(c)))
}

func blank() {
	if fb == nil {
		panic("graphics: fb == nil")
	}
	draw.Draw(fb, fb.Bounds(), image.Black, image.ZP, draw.Src)
}

func redraw() {
	world[2] = zcrop(world[0], world[1])
	for i, fr := range fr{
		fr.Delete(0, fr.Len())
		fmt.Fprintf(fr, "%s=%s (%s)", names[i], world[i], aspect(world[i]))
	}
	memdraw.Border(fb, world[0], 5, image.ZP, red)
	memdraw.Border(fb, world[1], 3, image.ZP, blue)
	memdraw.Border(fb, world[0].Intersect(world[1]), 1, image.ZP, yellow)
	memdraw.Border(fb, world[2], 3, image.ZP, green)
	w.Upload(image.ZP, img, img.Bounds())
	w.Publish()
}

// aspect returns r's aspect ratio as a vector
// for every ar.X pixels subtracted from r, ar.Y pixels
// must be subtracted from r to maintain the aspect ratio
func aspect(r image.Rectangle) (ar image.Point) {
	s := r.Size()
	return s.Div(gcd(s.X, s.Y))
}

// delta returns the difference between src and r
// in units of the aspect ratio ar, rounded up to the
// nearest whole unit
func delta(ar image.Point, src, r image.Rectangle) (units image.Point) {
	return image.Pt(
		((src.Dx() - r.Dx() + ar.X - 1) / ar.X),
		((src.Dy() - r.Dy() + ar.Y - 1) / ar.Y),
	)
}

func gcd(a, b int) int {
	for a != 0 {
		a, b = b%a, a
	}
	return b
}


func center(r image.Rectangle) image.Point {
	return r.Min.Add(r.Size().Div(2))
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// crop=1280:640:0:40 or crop=dx,dy,x,y
type ffcrop [4]int

func (a ffcrop) Rect() image.Rectangle {
	return image.Rect(
		a[2],
		a[3],
		a[0]+a[2],
		a[1]+a[3],
	)
}

func print(src, c image.Rectangle) {
	fmt.Printf("ffmpeg.431 -i 183.mp4 -vf crop=%d:%d:%d:%d,scale=%d,%d -c:a copy -vframes 1 out.183.jpg\n",
		c.Size().X,
		c.Size().Y,
		c.Min.X,
		c.Min.Y,
		src.Size().X,
		src.Size().Y,
	)
}

func main() {
	d := screen.Dev
	blank()
	redraw()

	for {
		select {
		case e := <-d.Mouse:
			e = readmouse(e)
			current := -1
			switch {
			case HasButton(2, down):
				current = 0
				fallthrough
			case HasButton(1, down):
				if current < 0{ current=1}
				for down != 0 {
					select {
					case e = <-d.Mouse:
						sp := pt(readmouse(e))
						sz := world[current].Size()
						world[current].Min = sp
						world[current].Max = sp.Add(sz)
					case <-d.Key:
						continue
					}
					blank()
					redraw()
				}
			}
		case e := <-d.Key:
			if e.Direction == key.DirRelease {
				continue
			}
			blank()
			switch e.Code {
			case key.CodeRightArrow:
				world[1].Max.X++
			case key.CodeLeftArrow:
				world[1].Min.X--
			case key.CodeUpArrow:
				world[1].Max.Y--
			case key.CodeDownArrow:
				world[1].Max.Y++
			}
			redraw()
		case e := <-d.Size:
			size = image.Pt(e.WidthPx, e.HeightPx)
		case e := <-d.Paint:
			e = e
			redraw()
		case e := <-d.Lifecycle:
			if e.To == lifecycle.StageDead {
				os.Exit(0)
			}
		}
	}
}

/*
 * Curve Helpers
 */

func drawEllipses(dst draw.Image, src image.Image, c, b, a int, p ...image.Point) {
	for _, p := range p {
		memdraw.Ellipse(dst, p, c, b, a, src, dst.Bounds().Min, 1, 1)
	}
}

/*
 * Mouse Stuff
 */

func pt(e mouse.Event) image.Point {
	return image.Pt(int(e.X), int(e.Y))
}

const (
	KShift = 1 << iota
	KCtrl
	KAlt
	KMeta
)

func Button(n uint) uint {
	return 1 << n
}
func HasButton(n, mask uint) bool {
	return Button(n)&mask != 0
}

func readmouse(e mouse.Event) mouse.Event {
	if e.Button == 1 {
		if km := e.Modifiers; km&KCtrl != 0 {
			e.Button = 3
		} else if km&KAlt != 0 {
			e.Button = 2
		}
	}
	if dir := e.Direction; dir == 1 {
		down |= 1 << uint(e.Button)
	} else if dir == 2 {
		down &^= 1 << uint(e.Button)
	}
	return e
}
