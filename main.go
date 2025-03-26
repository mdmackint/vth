package main

import (
	"embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp/v2"
)

type Game struct {
	Visible    [500]bool
	Pos        [500]cp.Vector
	Radii      [500]float64
	Inputless  uint64
	LastAuto   uint64
	UserGen    line
	Drawing    bool
	HasWrapped bool
	Paused     bool
	TempImage  imageTimeout
	Elasticity float64
	RandRad    bool
	Height     int
	Width      int
	LastDebug  uint
}

type line struct {
	X0    float32
	Y0    float32
	X1    float32
	Y1    float32
	Width float32
}

type imageTimeout struct {
	Image     *ebiten.Image
	TicksLeft uint
}

var (
	gravDisabled *bool
	ugc          *bool
	space        *cp.Space
	ballArray    [500]*cp.Body
	shapeArray   []*cp.Shape
	gravImages   [6]*ebiten.Image
	pauseImg     [2]*ebiten.Image
	mass         float64
	moment       float64
	counter      uint64 // Should be used when drawing
	writer       uint64 // Should be used when simulating
	obst         []*cp.Shape
	radius       float64
	lines        []line
	actor        *ebiten.Image
	imgMode      bool
	autonomous   *bool
	debugging    *bool
	icon         []image.Image
	speedImg     [3]*ebiten.Image
	instaclose   *bool
	miscImg      [4]*ebiten.Image
)

//go:embed data
var fs embed.FS

func obstGen(x0, y0, x1, y1, r float64, visible bool) {
	if visible {
		var z line
		z.X0, z.Y0, z.X1, z.Y1, z.Width = float32(x0), float32(y0), float32(x1), float32(y1), float32(r)
		lines = append(lines, z)
	}
	obst = append(obst, cp.NewSegment(space.StaticBody, cp.Vector{X: x0, Y: y0}, cp.Vector{X: x1, Y: y1}, 2))
}

func loadImage(path string) *ebiten.Image {
	x, _, err := ebitenutil.NewImageFromFileSystem(fs, path)
	if err == nil {
		return x
	} else {
		log.Fatalln("Failed to load image with path "+path+"; error:\n", err)
		return nil
	}
}
func loadMultiple(paths []string) []*ebiten.Image {
	var images []*ebiten.Image
	for n, i := range paths {
		x, _, err := ebitenutil.NewImageFromFileSystem(fs, i)
		if err != nil {
			log.Fatalf("Loading images failed! Image no. %d, path %s\n", n, i)
		}
		images = append(images, x)
	}
	return images
}

func (g *Game) Tick(div float64, f int) {
	for range f {
		space.Step(div)
	}
}

func init() {
	var err error
	// Load actor (mario coin)
	actor = loadImage("data/actor.png")
	// Load window icon
	_, iconImage, err := ebitenutil.NewImageFromFileSystem(fs, "data/icon.png")
	if err != nil {
		log.Fatalln("Failed to load logo")
	}

	// Load gravity control messages, copy them into array
	images := loadMultiple([]string{"data/xgravadd.png", "data/xgravsub.png", "data/ygravadd.png", "data/ygravsub.png", "data/gravreset.png", "data/gravlimit.png"})
	copy(gravImages[:], images[:])

	// Load pause messages, copy them into array
	pauseDialogues := loadMultiple([]string{"data/paused.png", "data/resumed.png"})
	copy(pauseImg[:], pauseDialogues[:])

	// Load speed control messages, copy them into array
	speedImgSlice := loadMultiple([]string{"data/speedup.png", "data/slowdown.png","data/normalspeed.png"})
	copy(speedImg[:], speedImgSlice[:])

	miscImgSlice := loadMultiple([]string{"data/elasticadd.png","data/elasticsub.png","data/fixedrad.png","data/randrad.png"})
	copy(miscImg[:],miscImgSlice[:])

	// Append icon image to slice
	icon = append(icon, iconImage)

	// Create physics simulation space and set gravity
	space = cp.NewSpace()
	space.SetGravity(cp.Vector{X: 0.0, Y: 300.0})
	// Add obstacles and set properties
	obstGen(160, 100, 320, 60, 4.0, true)
	obstGen(320, 60, 480, 100, 4.0, true)
	obstGen(0, 140, 160, 180, 4.0, true)
	obstGen(640, 140, 480, 180, 4.0, true)
	obstGen(-2, 0, -2, 0x300, 10.0, false)
	obstGen(0x282, 0, 0x282, 0x300, 10.0, false)
	obstGen(0, -5, 0x280, -5, 4.0, false)
	obstGen(0, 240, 500, 300, 4.0, true)
	obstGen(640, 400, 140, 460, 4.0, true)
	obstGen(0, 520, 300, 600, 4.0, true)
	obstGen(640, 520, 340, 600, 4.0, true)
	for _, x := range obst {
		x.SetFriction(1)
		x.SetElasticity(0.25)
		space.AddShape(x)
	}
	// Generate first ball and set some properties
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

// Runs every tick.
// Welcome to the land of the if statements.
func (g *Game) Update() error {
	g.LastDebug++
	radius = float64(rand.Intn(4)+6)
	if !g.RandRad {
		radius = 8.0
	}
	if *instaclose {
		os.Exit(0)
	}
	// Pause the game if user strikes key K
	if inpututil.IsKeyJustPressed(ebiten.KeyK) {
		switch g.Paused {
		case true:
			g.Paused = false
			g.TempImage.Image = pauseImg[1]
			g.TempImage.TicksLeft = 30
		case false:
			g.Paused = true
			g.TempImage.Image = pauseImg[0]
			g.TempImage.TicksLeft = math.MaxUint
		}
	}
	var auto bool = g.Inputless >= 900
	if *autonomous {
		auto = true
	}
	// Reduce time remaining for temporary dialogues
	if g.TempImage.TicksLeft != 0 {
		g.TempImage.TicksLeft--
	}
	var releasedTouches = inpututil.AppendJustReleasedTouchIDs([]ebiten.TouchID{})
	var touch bool = false
	for range releasedTouches {
		touch = true
		break
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		imgMode = !imgMode
		if imgMode && g.RandRad {
			imgMode = false
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.RandRad = !g.RandRad
		switch g.RandRad {
		case true:
			g.TempImage.Image, g.TempImage.TicksLeft = miscImg[3], 30
			imgMode = false
		case false: g.TempImage.Image, g.TempImage.TicksLeft = miscImg[2], 30
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		switch g.Elasticity {
		case 1.0:
			g.Elasticity = 1.15
			g.TempImage.Image, g.TempImage.TicksLeft = miscImg[0], 30
			for index, item := range shapeArray {
				if index == int(counter) {
					break
				}
				item.SetElasticity(1.15)
				
			}
		default:
			g.Elasticity = 1.0
			g.TempImage.Image, g.TempImage.TicksLeft = miscImg[1], 30
			for index, item := range shapeArray {
				if index == int(counter) {
					break
				}
				item.SetElasticity(1.0)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) && !*gravDisabled {
		space.SetGravity(space.Gravity().Sub(cp.Vector{X: 50, Y: 0}))
		g.TempImage.Image, g.TempImage.TicksLeft = gravImages[1], 30
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) && !*gravDisabled {
		space.SetGravity(space.Gravity().Sub(cp.Vector{X: 0, Y: 50}))
		g.TempImage.Image, g.TempImage.TicksLeft = gravImages[3], 30
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && !*gravDisabled {
		space.SetGravity(space.Gravity().Add(cp.Vector{X: 0, Y: 50}))
		g.TempImage.Image, g.TempImage.TicksLeft = gravImages[2], 30
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) && !*gravDisabled {
		space.SetGravity(space.Gravity().Add(cp.Vector{X: 50, Y: 0}))
		g.TempImage.Image, g.TempImage.TicksLeft = gravImages[0], 30
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		space.SetGravity(cp.Vector{X: 0, Y: 300})
		g.TempImage.Image, g.TempImage.TicksLeft = gravImages[4], 30
	}

	// Check that gravity is not outside of reasonable bounds
	// Physics can break a bit when gravity is too strong
	grav := space.Gravity()
	modified := false
	if grav.X > 500 {
		space.SetGravity(cp.Vector{X: 500, Y: grav.Y})
		modified = true
	}
	if grav.X < -500 {
		space.SetGravity(cp.Vector{X: -500, Y: grav.Y})
		modified = true
	}
	if grav.Y > 500 {
		space.SetGravity(cp.Vector{X: grav.X, Y: 500})
		modified = true
	}
	if grav.Y < -500 {
		space.SetGravity(cp.Vector{X: grav.X, Y: -500})
		modified = true
	}
	if modified {
		g.TempImage.Image = gravImages[5]
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) && *ugc {
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
	if (inpututil.IsKeyJustPressed(ebiten.KeySpace) || touch || inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0)) && !*autonomous && !g.Paused {
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
		circle.SetElasticity(g.Elasticity)
		circle.SetCollisionType(cp.CollisionHandlerDefault.TypeB)
		shapeArray = append(shapeArray, circle)
		g.Visible[writer] = true
		g.Radii[writer] = radius
		if counter < 499 {
			counter++
		}
		writer++
		g.Inputless = 0
	} else if auto && g.LastAuto == 15 && !g.Paused {
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
		circle.SetElasticity(g.Elasticity)
		circle.SetCollisionType(cp.CollisionHandlerDefault.TypeB)
		shapeArray = append(shapeArray, circle)
		g.Visible[writer] = true
		g.Radii[writer] = radius
		if counter < 499 {
			counter++
		}
		writer++
		g.LastAuto = 0
	} else if auto && !g.Paused {
		g.LastAuto++
	} else if !g.Paused {
		g.Inputless++
	}
	var x int = 0
	for x < int(counter) {
		g.Pos[x] = ballArray[x].Position()
		x++
	}
	if !g.Paused {
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			if g.TempImage.Image != speedImg[0] {
				g.TempImage.Image = speedImg[0]
				g.TempImage.TicksLeft = 30
			}

			g.Tick(1.0 / 480.0, 16)
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			if g.TempImage.Image != speedImg[1] {
				g.TempImage.Image = speedImg[1]
				g.TempImage.TicksLeft = 30
			}
			g.Tick(1.0 / 480.0, 4)
		} else {
			g.Tick(1.0 / 480.0, 8)
		}
		if inpututil.IsKeyJustReleased(ebiten.KeyArrowUp) || inpututil.IsKeyJustReleased(ebiten.KeyArrowDown) {
			g.TempImage.Image = speedImg[2]
			g.TempImage.TicksLeft = 30
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xff, 0xe0, 0xeb, 0xff})
	for x := range counter {
		if !g.Visible[x] {
			continue
		}
		if imgMode {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(16.0/350.0, 16.0/350.0)
			opts.GeoM.Translate(g.Pos[x].X-8, g.Pos[x].Y-8)
			screen.DrawImage(actor, opts)
		} else {
			vector.DrawFilledCircle(screen, float32(g.Pos[x].X), float32(g.Pos[x].Y), float32(g.Radii[x]), color.RGBA{0xef, 0x60, 0x6b, 0xff}, true)
		}
	}
	for _, item := range lines {
		vector.StrokeLine(
			screen, item.X0, item.Y0, item.X1, item.Y1,
			item.Width, color.RGBA{0xef, 0x60, 0x6b, 0xff}, true,
		)
	}
	if *debugging {
		msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\n", ebiten.ActualTPS(), ebiten.ActualFPS())
		ebitenutil.DebugPrint(screen, msg)
	}
	if g.Drawing {
		vector.DrawFilledCircle(screen, g.UserGen.X0, g.UserGen.Y0, 5, color.RGBA{0xff, 0xc0, 0xcb, 0xff}, true)
	}
	if g.TempImage.TicksLeft > 0 {
		opts := &ebiten.DrawImageOptions{}
		switch g.TempImage.TicksLeft {
		case 30, 1:
			opts.ColorScale.SetA(0.1)
		case 29, 2:
			opts.ColorScale.SetA(0.3)
		case 28, 3:
			opts.ColorScale.SetA(0.5)
		case 27, 4:
			opts.ColorScale.SetA(0.7)
		}
		screen.DrawImage(g.TempImage.Image, opts)
	} else if g.TempImage.TicksLeft == 0 && g.Paused {
		g.TempImage.Image = pauseImg[0]
		g.TempImage.TicksLeft = math.MaxUint
		screen.DrawImage(g.TempImage.Image, nil)
	}
}

func (g *Game) Layout(ow, oh int) (w, h int) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ebiten.SetWindowSize(0x280,oh)
	}
	g.Width = 0x280
	if oh > 0x2ba {
		g.Height = oh
		return 0x280, oh
	}
	g.Height = 0x2ba
	return 0x280, 0x2ba
}
func main() {
	instaclose = flag.Bool("instaclose",false,"Instantly quit on first frame - debugging only!")
	gravDisabled = flag.Bool("g", false, "Disable gravity controls")
	var undecorated = *flag.Bool("t", false, "Hide titlebar of window")
	ugc = flag.Bool("u", false, "Allow user-generated obstacles (default false)")
	autonomous = flag.Bool("a", false, "Run autonomously only and ignore user input")
	debugging = flag.Bool("d", false, "Show TPS and FPS in window corner")
	var resizable = flag.Bool("r", false, "Disables resizing of the window")
	resizableBool := !*resizable
	resizable = &resizableBool
	var imgFlag = flag.Bool("i", false, "Show actor image instead of circle")
	flag.Parse()
	imgMode = *imgFlag
	ebiten.SetWindowTitle("vth")
	ebiten.SetWindowSize(0x280, 0x2ba)
	ebiten.SetWindowIcon(icon)
	if *resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	ebiten.SetWindowDecorated(!undecorated)
	if err := ebiten.RunGame(&Game{Radii: [500]float64{8}, Elasticity: 1.0, LastDebug: 60, Visible: [500]bool{true}}); err != nil {
		log.Fatalln(err)
	}
}
