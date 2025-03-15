package main

import (
	"embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp/v2"
)

type Game struct {
	Pos        [500]cp.Vector
	Inputless  uint64
	LastAuto   uint64
	UserGen    line
	Drawing    bool
	HasWrapped bool
}

type line struct {
	X0    float32
	Y0    float32
	X1    float32
	Y1    float32
	Width float32
}

var (
	space      *cp.Space
	ballArray  [500]*cp.Body
	shapeArray []*cp.Shape
	mass       float64
	moment     float64
	counter    uint64 // Should be used when drawing
	writer     uint64 // Should be used when simulating
	obst       []*cp.Shape
	radius     float64
	lines      []line
	actor      *ebiten.Image
	imgMode    bool
	autonomous *bool
	debugging  *bool
	icon       []image.Image
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
	var err error
	actor, _, err = ebitenutil.NewImageFromFileSystem(fs, "data/actor.png")
	if err != nil {
		log.Fatalln("Failed to load actor")
	}
	_, iconImage, err := ebitenutil.NewImageFromFileSystem(fs, "data/icon.png")
	if err != nil {
		log.Fatalln("Failed to load logo")
	}
	icon = append(icon, iconImage)
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
	if *autonomous {
		auto = true
	}
	var releasedTouches = inpututil.AppendJustReleasedTouchIDs([]ebiten.TouchID{})
	var touch bool = false
	for range releasedTouches {
		touch = true
		break
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		imgMode = !imgMode
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
		switch g.Drawing {
		case false:
			mouseX, mouseY := ebiten.CursorPosition()
			g.UserGen.X0, g.UserGen.Y0 = float32(mouseX), float32(mouseY)
			g.Drawing = true
		case true:
			mouseX, mouseY := ebiten.CursorPosition()
			g.UserGen.X1, g.UserGen.Y1 = float32(mouseX), float32(mouseY)
			g.UserGen.Width = 4
			lines = append(lines, g.UserGen)
			usergen := cp.NewSegment(space.StaticBody, cp.Vector{X: float64(g.UserGen.X0), Y: float64(g.UserGen.Y0)}, cp.Vector{X: float64(g.UserGen.X1), Y: float64(g.UserGen.Y1)}, 2)
			usergen.SetFriction(1)
			usergen.SetElasticity(0.25)
			space.AddShape(usergen)
			g.Drawing = false
		}
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeySpace) || touch || inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0)) && !*autonomous {
		if writer > 499 {
			writer = 0
			g.HasWrapped = true
		}
		if g.HasWrapped {
			space.RemoveBody(ballArray[writer])
			space.RemoveShape(shapeArray[len(shapeArray)-499])
		}
		ballArray[writer] = space.AddBody(cp.NewBody(mass, moment))
		ballArray[writer].SetPosition(cp.Vector{X: 280 + float64(rand.Intn(80)), Y: -5})
		var circle = space.AddShape(cp.NewCircle(ballArray[writer], radius, cp.Vector{X: 0, Y: 0}))
		shapeArray = append(shapeArray, circle)
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
			g.HasWrapped = true
		}
		if g.HasWrapped {
			space.RemoveBody(ballArray[writer])
			space.RemoveShape(shapeArray[len(shapeArray)-499])
		}
		ballArray[writer] = space.AddBody(cp.NewBody(mass, moment))
		ballArray[writer].SetPosition(cp.Vector{X: 280 + float64(rand.Intn(80)), Y: -5})
		var circle = space.AddShape(cp.NewCircle(ballArray[writer], radius, cp.Vector{X: 0, Y: 0}))
		shapeArray = append(shapeArray, circle)
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
		if imgMode {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(16.0/350.0, 16.0/350.0)
			opts.GeoM.Translate(g.Pos[x].X-8, g.Pos[x].Y-8)
			screen.DrawImage(actor, opts)
		} else {
			vector.DrawFilledCircle(screen, float32(g.Pos[x].X), float32(g.Pos[x].Y), float32(radius), color.RGBA{0xef, 0x60, 0x6b, 0xff}, true)
		}
		x++
	}
	for _, item := range lines {
		vector.StrokeLine(screen, item.X0, item.Y0, item.X1, item.Y1, item.Width, color.RGBA{0xef, 0x60, 0x6b, 0xff}, true)
	}
	if *debugging {
		msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\n", ebiten.ActualTPS(), ebiten.ActualFPS())
		ebitenutil.DebugPrint(screen, msg)
	}
	if g.Drawing {
		vector.DrawFilledCircle(screen, g.UserGen.X0, g.UserGen.Y0, 5, color.RGBA{0xff, 0xc0, 0xcb, 0xff}, true)
	}
}

func (g *Game) Layout(ow, oh int) (w, h int) {
	return 0x280, 0x2ba
}
func main() {
	autonomous = flag.Bool("a", false, "Run autonomously only and ignore user input")
	debugging = flag.Bool("d", false, "Show TPS and FPS in window corner")
	var resizable = flag.Bool("r", false, "Makes the window resizable")
	var imgFlag = flag.Bool("i", false, "Show actor image instead of circle")
	flag.Parse()
	imgMode = *imgFlag == true
	ebiten.SetWindowTitle("vth")
	ebiten.SetWindowSize(0x280, 0x2ba)
	ebiten.SetWindowIcon(icon)
	if *resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatalln(err)
	}
}
