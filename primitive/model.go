package primitive

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"time"

	"github.com/fogleman/gg"
)

const (
	SaveFrames    = false
	OutlineShapes = false
)

type Model struct {
	W, H       int
	Background color.Color
	Target     *image.RGBA
	Current    *image.RGBA
	Buffer     *image.RGBA
	Context    *gg.Context
	Score      float64
	Alpha      int
	Scale      int
	Mode       Mode
	Shapes     []Shape
	Scores     []float64
}

func NewModel(target image.Image, alpha, scale int, mode Mode) *Model {
	c := averageImageColor(target)
	// c := color.White
	// c := color.Black
	size := target.Bounds().Size()
	model := &Model{}
	model.W = size.X
	model.H = size.Y
	model.Background = c
	model.Alpha = alpha
	model.Scale = scale
	model.Mode = mode
	model.Target = imageToRGBA(target)
	model.Current = uniformRGBA(target.Bounds(), c)
	model.Buffer = uniformRGBA(target.Bounds(), c)
	model.Score = differenceFull(model.Target, model.Current)
	model.Context = model.newContext()
	return model
}

func (model *Model) newContext() *gg.Context {
	s := model.Scale
	dc := gg.NewContext(model.W*s, model.H*s)
	dc.Scale(float64(s), float64(s))
	dc.Translate(0.5, 0.5)
	dc.SetColor(model.Background)
	dc.Clear()
	return dc
}

func (model *Model) Frames(scoreDelta float64) []image.Image {
	var result []image.Image
	dc := model.newContext()
	result = append(result, imageToRGBA(dc.Image()))
	previous := 10.0
	for i, shape := range model.Shapes {
		c := model.computeColor(shape.Rasterize(), model.Alpha)
		dc.SetRGBA255(c.R, c.G, c.B, c.A)
		shape.Draw(dc)
		dc.Fill()
		score := model.Scores[i]
		delta := previous - score
		if delta >= scoreDelta {
			previous = score
			result = append(result, imageToRGBA(dc.Image()))
		}
	}
	return result
}

func (model *Model) Add(shape Shape) {
	lines := shape.Rasterize()
	c := model.computeColor(lines, model.Alpha)
	s := model.computeScore(lines, c, model.Buffer)
	Draw(model.Current, c, lines)

	model.Score = s
	model.Shapes = append(model.Shapes, shape)
	model.Scores = append(model.Scores, s)

	model.Context.SetRGBA255(c.R, c.G, c.B, c.A)
	shape.Draw(model.Context)
	model.Context.Fill()
}

func (model *Model) Run(n int) image.Image {
	start := time.Now()
	for i := 1; i <= n; i++ {
		model.Step()
		elapsed := time.Since(start).Seconds()
		v("iteration %d, time %.3f, score %.6f\n", i, elapsed, model.Score)
		if SaveFrames {
			SavePNG("out.png", model.Current)
			model.Context.SavePNG(fmt.Sprintf("out%03d.png", i))
		}
	}
	if OutlineShapes {
		for _, shape := range model.Shapes {
			model.Context.NewSubPath()
			shape.Draw(model.Context)
		}
		c := averageImageColor(model.Target)
		model.Context.SetRGBA255(int(c.R), int(c.G), int(c.B), 64)
		model.Context.Stroke()
	}
	return model.Context.Image()
}

func (model *Model) Step() {
	state := model.BestHillClimbState(model.Buffer, model.Mode, 100, 100, 20)
	// state := model.BestRandomState(model.Buffer, model.Mode, 3000)
	// state = Anneal(state, 0.1, 0.00001, 25000).(*State)
	state = HillClimb(state, 1000).(*State)
	model.Add(state.Shape)
}

func (model *Model) BestHillClimbState(buffer *image.RGBA, t Mode, n, age, m int) *State {
	var bestEnergy float64
	var bestState *State
	for i := 0; i < m; i++ {
		state := model.BestRandomState(buffer, t, n)
		state = HillClimb(state, age).(*State)
		energy := state.Energy()
		if i == 0 || energy < bestEnergy {
			bestEnergy = energy
			bestState = state
		}
	}
	return bestState
}

func (model *Model) BestRandomState(buffer *image.RGBA, t Mode, n int) *State {
	var bestEnergy float64
	var bestState *State
	for i := 0; i < n; i++ {
		state := model.RandomState(buffer, t)
		energy := state.Energy()
		if i == 0 || energy < bestEnergy {
			bestEnergy = energy
			bestState = state
		}
	}
	return bestState
}

func (model *Model) RandomState(buffer *image.RGBA, t Mode) *State {
	switch t {
	default:
		return model.RandomState(buffer, Mode(rand.Intn(5)+1))
	case ModeTriangle:
		return NewState(model, buffer, NewRandomTriangle(model.W, model.H))
	case ModeRectangle:
		return NewState(model, buffer, NewRandomRectangle(model.W, model.H))
	case ModeEllipse:
		return NewState(model, buffer, NewRandomEllipse(model.W, model.H))
	case ModeCircle:
		return NewState(model, buffer, NewRandomCircle(model.W, model.H))
	case ModeRotatedRectangle:
		return NewState(model, buffer, NewRandomRotatedRectangle(model.W, model.H))
	}
}

func (model *Model) computeColor(lines []Scanline, alpha int) Color {
	var count int
	var rsum, gsum, bsum float64
	a := float64(alpha) / 255
	for _, line := range lines {
		i := model.Target.PixOffset(line.X1, line.Y)
		for x := line.X1; x <= line.X2; x++ {
			count++
			tr := float64(model.Target.Pix[i])
			tg := float64(model.Target.Pix[i+1])
			tb := float64(model.Target.Pix[i+2])
			cr := float64(model.Current.Pix[i])
			cg := float64(model.Current.Pix[i+1])
			cb := float64(model.Current.Pix[i+2])
			i += 4
			rsum += (a*cr - cr + tr) / a
			gsum += (a*cg - cg + tg) / a
			bsum += (a*cb - cb + tb) / a
		}
	}
	if count == 0 {
		return Color{}
	}
	r := clampInt(int(rsum/float64(count)), 0, 255)
	g := clampInt(int(gsum/float64(count)), 0, 255)
	b := clampInt(int(bsum/float64(count)), 0, 255)
	return Color{r, g, b, alpha}
}

func (model *Model) computeScore(lines []Scanline, c Color, buffer *image.RGBA) float64 {
	copy(buffer.Pix, model.Current.Pix)
	Draw(buffer, c, lines)
	return differencePartial(model.Target, model.Current, buffer, model.Score, lines)
}

func (model *Model) Energy(shape Shape, buffer *image.RGBA) float64 {
	lines := shape.Rasterize()
	c := model.computeColor(lines, model.Alpha)
	s := model.computeScore(lines, c, buffer)
	return s
}
