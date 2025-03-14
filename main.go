package main

import (
	"image/color"
	"log"
	"math/rand"
	"embed"
	_ "image/png"
	"flag"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp/v2"
)

type Game struct {
	Pos       [500]cp.Vector
	Inputless uint64
	LastAuto  uint64
}

type line struct {
	X0    float32
	Y0    float32
	X1    float32
	Y1    float32
	Width float32
}

var (
	space     *cp.Space
	ballArray [500]*cp.Body
	mass      float64
	moment    float64
	counter   uint64 // Should be used when drawing
	writer    uint64 // Should be used when simulating
	obst      []*cp.Shape
	radius    float64
	lines     []line
	actor *ebiten.Image
	imgMode *bool
)

//go:embed data
var fs embed.FS

func obstGen(x0, y0, x1, y1 float64, visible bool) {
	if visible {
		var z line
		z.X0, z.Y0, z.X1, z.Y1, z.Width = float32(x0), float32(y0), float32(x1), float32(y1), 4.0
		lines = append(lines, z)
	}
	obst = append(obst, cp.NewSegment(space.StaticBody, cp.Vector{X: x0, Y: y0}, cp.Vector{X: x1, Y: y1}, 2))
}

func init() {
	imgMode = flag.Bool("i",false,"Show actor image instead of circle")
	flag.Parse()
	if *imgMode {
		var err error
		actor, _, err = ebitenutil.NewImageFromFileSystem(fs,"data/actor.png")
		if err != nil {
			log.Fatalln("Failed to load actor")
		}
	}
	space = cp.NewSpace()
	space.SetGravity(cp.Vector{X: 0.0, Y: 300.0})
	obstGen(160, 100, 320, 60, true)
	obstGen(320, 60, 480, 100, true)
	obstGen(0, 140, 160, 180, true)
	obstGen(640, 140, 480, 180, true)
	obstGen(0, 0, 0, 0x300, false)
	obstGen(0x280, 0, 0x280, 0x300, false)
	obstGen(0, -5, 0x280, -5, false)
	obstGen(0, 240, 500, 300, true)
	obstGen(640, 400, 140, 460, true)
	obstGen(0, 520, 300, 600, true)
	obstGen(640, 520, 340, 600, true)

	for _, x := range obst {
		x.SetFriction(1)
		x.SetElasticity(0.25)
		space.AddShape(x)
	}
	radius = 8.0
	mass = 1.0
	moment = cp.MomentForCircle(mass, 0, radius, cp.Vector{X: 0, Y: 0})
	ballArray[0] = space.AddBody(cp.NewBody(mass, moment))
	ballArray[0].SetPosition(cp.Vector{X: 280 + float64(rand.Intn(80)), Y: -5})
	var circle = space.AddShape(cp.NewCircle(ballArray[0], radius, cp.Vector{X: 0, Y: 0}))
	circle.SetElasticity(1)
	circle.SetCollisionType(cp.CollisionHandlerDefault.TypeB)
	counter = 1
	writer = 1
}

func (g *Game) Update() error {
	var auto bool = g.Inputless >= 900
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if writer > 499 {
			writer = 0
		}
		ballArray[writer] = space.AddBody(cp.NewBody(mass, moment))
		ballArray[writer].SetPosition(cp.Vector{X: 280 + float64(rand.Intn(80)), Y: -5})
		var circle = space.AddShape(cp.NewCircle(ballArray[writer], radius, cp.Vector{X: 0, Y: 0}))
		circle.SetElasticity(1)
		circle.SetCollisionType(cp.CollisionHandlerDefault.TypeB)
		if counter < 499 {
			counter++
		}
		writer++
		g.Inputless = 0
	} else if auto && g.LastAuto == 15 {
		if writer > 499 {
			writer = 0
		}
		ballArray[writer] = space.AddBody(cp.NewBody(mass, moment))
		ballArray[writer].SetPosition(cp.Vector{X: 280 + float64(rand.Intn(80)), Y: -5})
		var circle = space.AddShape(cp.NewCircle(ballArray[writer], radius, cp.Vector{X: 0, Y: 0}))
		circle.SetElasticity(1)
		circle.SetCollisionType(cp.CollisionHandlerDefault.TypeB)
		if counter < 499 {
			counter++
		}
		writer++
		g.LastAuto = 0
	} else if auto {
		g.LastAuto++
	} else {
		g.Inputless++
	}
	var x int = 0
	for x < int(counter) {
		g.Pos[x] = ballArray[x].Position()
		x++
	}
	space.Step(1.0 / 60.0)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xff, 0xe0, 0xeb, 0xff})
	var x int = 0
	for x < int(counter) {
		if *imgMode {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(16.0/350.0,16.0/350.0)
			opts.GeoM.Translate(g.Pos[x].X-8, g.Pos[x].Y-8)
			screen.DrawImage(actor,opts)
		} else {
			vector.DrawFilledCircle(screen, float32(g.Pos[x].X), float32(g.Pos[x].Y), float32(radius), color.RGBA{0xef, 0x60, 0x6b, 0xff}, true)
		}
		x++
	}
	for _, item := range lines {
		vector.StrokeLine(screen, item.X0, item.Y0, item.X1, item.Y1, item.Width, color.RGBA{0xef, 0x60, 0x6b, 0xff}, true)
	}
}

func (g *Game) Layout(ow, oh int) (w, h int) {
	return 0x280, 0x2ba
}
func main() {
	ebiten.SetWindowTitle("vth")
	ebiten.SetWindowSize(0x280, 0x2ba)
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatalln(err)
	}
}
