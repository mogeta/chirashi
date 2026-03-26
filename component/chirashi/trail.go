package chirashi

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	defaultTrailMaxPoints        = 12
	defaultTrailMinPointDistance = float32(6)
	defaultTrailMaxPointAge      = float32(0.35)
	maxTrailBatchVertices        = 65535
)

var (
	trailWhiteImage     *ebiten.Image
	trailWhiteImageOnce sync.Once
)

type particleTrailBatchBuilder struct {
	screen *ebiten.Image
	trail  *TrailRuntime
	opts   ebiten.DrawTrianglesOptions
}

func buildTrailData(config *TrailConfig) TrailData {
	if config == nil || !config.Enabled {
		return TrailData{}
	}

	maxPoints := config.MaxPoints
	if maxPoints < 2 {
		maxPoints = defaultTrailMaxPoints
	}
	minPointDistance := config.MinPointDistance
	if minPointDistance <= 0 {
		minPointDistance = defaultTrailMinPointDistance
	}
	maxPointAge := config.MaxPointAge
	if maxPointAge <= 0 {
		maxPointAge = defaultTrailMaxPointAge
	}

	trail := TrailData{
		Params: TrailParams{
			Enabled:          true,
			Mode:             config.Mode,
			LocalSpace:       config.Space == "local",
			MaxPoints:        maxPoints,
			MinPointDistance: minPointDistance,
			MaxPointAge:      maxPointAge,
			WidthStart:       config.Width.Start,
			WidthEnd:         config.Width.End,
			WidthEasing:      ParseEasing(config.Width.Easing),
			AlphaStart:       config.Alpha.Start,
			AlphaEnd:         config.Alpha.End,
			AlphaEasing:      ParseEasing(config.Alpha.Easing),
			ColorStartR:      1,
			ColorStartG:      1,
			ColorStartB:      1,
			ColorEndR:        1,
			ColorEndG:        1,
			ColorEndB:        1,
			ColorEasing:      EasingLinear,
		},
		Runtime: TrailRuntime{
			Points:   make([]TrailPoint, 0, maxPoints),
			Ghosts:   make([]TrailGhost, 0),
			Vertices: make([]ebiten.Vertex, 0, maxPoints*2),
			Indices:  make([]uint16, 0, (maxPoints-1)*6),
		},
	}
	if config.Color != nil {
		trail.Params.ColorStartR = config.Color.StartR
		trail.Params.ColorStartG = config.Color.StartG
		trail.Params.ColorStartB = config.Color.StartB
		trail.Params.ColorEndR = config.Color.EndR
		trail.Params.ColorEndG = config.Color.EndG
		trail.Params.ColorEndB = config.Color.EndB
		trail.Params.ColorEasing = ParseEasing(config.Color.Easing)
	}
	return trail
}

func isParticleTrail(trail *TrailData) bool {
	return trail.Params.Enabled && trail.Params.Mode == "particle"
}

func trimTrailPoints(points []TrailPoint, maxPoints int) []TrailPoint {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	copy(points, points[len(points)-maxPoints:])
	return points[:maxPoints]
}

func shiftTrailPoints(points []TrailPoint, dx, dy float32) {
	if dx == 0 && dy == 0 {
		return
	}
	for i := range points {
		points[i].X += dx
		points[i].Y += dy
	}
}

func updateTrail(data *SystemData) {
	trail := &data.Trail
	if !trail.Params.Enabled || trail.Params.MaxPoints < 2 {
		return
	}
	if isParticleTrail(trail) {
		updateParticleTrails(data)
		return
	}
	updateEmitterTrail(data)
}

func updateEmitterTrail(data *SystemData) {
	trail := &data.Trail
	currentTime := data.CurrentTime
	pruneBefore := currentTime - trail.Params.MaxPointAge
	keepFrom := 0
	for keepFrom < len(trail.Runtime.Points) && trail.Runtime.Points[keepFrom].CapturedAt < pruneBefore {
		keepFrom++
	}
	if keepFrom > 0 {
		copy(trail.Runtime.Points, trail.Runtime.Points[keepFrom:])
		trail.Runtime.Points = trail.Runtime.Points[:len(trail.Runtime.Points)-keepFrom]
	}

	head := TrailPoint{X: data.EmitterX, Y: data.EmitterY, CapturedAt: currentTime}
	if len(trail.Runtime.Points) == 0 {
		trail.Runtime.Points = append(trail.Runtime.Points, head)
		return
	}

	last := &trail.Runtime.Points[len(trail.Runtime.Points)-1]
	dx := head.X - last.X
	dy := head.Y - last.Y
	if dx*dx+dy*dy >= trail.Params.MinPointDistance*trail.Params.MinPointDistance {
		if len(trail.Runtime.Points) == trail.Params.MaxPoints {
			copy(trail.Runtime.Points, trail.Runtime.Points[1:])
			trail.Runtime.Points[len(trail.Runtime.Points)-1] = head
			return
		}
		trail.Runtime.Points = append(trail.Runtime.Points, head)
		return
	}

	// Keep the trail head pinned to the current emitter position while preserving older history.
	*last = head
}

func trailHasVisiblePoints(data *SystemData) bool {
	if !data.Trail.Params.Enabled {
		return false
	}
	if isParticleTrail(&data.Trail) {
		for _, idx := range data.ActiveIndices {
			if len(data.ParticlePool[idx].TrailPoints) >= 2 {
				return true
			}
		}
		for _, ghost := range data.Trail.Runtime.Ghosts {
			if len(ghost.Points) >= 2 {
				return true
			}
		}
		return false
	}
	return len(data.Trail.Runtime.Points) >= 2
}

func drawTrail(screen *ebiten.Image, data *SystemData) {
	trail := &data.Trail
	if !trailHasVisiblePoints(data) {
		return
	}
	if isParticleTrail(trail) {
		drawParticleTrails(screen, data)
		return
	}

	buildEmitterTrailMesh(data)
	if len(trail.Runtime.Indices) == 0 {
		return
	}

	op := &ebiten.DrawTrianglesOptions{}
	drawTrailBatch(screen, trail.Runtime.Vertices, trail.Runtime.Indices, op)
}

func buildEmitterTrailMesh(data *SystemData) {
	trail := &data.Trail
	trail.Runtime.Vertices = trail.Runtime.Vertices[:0]
	trail.Runtime.Indices = trail.Runtime.Indices[:0]

	if trail.Params.MaxPointAge <= 0 {
		return
	}
	if len(trail.Runtime.Points) < 2 {
		return
	}

	appendTrailMeshForPoints(data, trail.Runtime.Points)
}

func updateParticleTrails(data *SystemData) {
	currentTime := data.CurrentTime
	pruneTrailGhosts(&data.Trail.Runtime, data.Trail.Params.MaxPointAge, currentTime)
	for _, idx := range data.ActiveIndices {
		p := &data.ParticlePool[idx]
		updateSingleParticleTrail(data, p, currentTime)
	}
}

func updateSingleParticleTrail(data *SystemData, p *Instance, currentTime float32) {
	points := p.TrailPoints
	pruneBefore := currentTime - data.Trail.Params.MaxPointAge
	keepFrom := 0
	for keepFrom < len(points) && points[keepFrom].CapturedAt < pruneBefore {
		keepFrom++
	}
	if keepFrom > 0 {
		copy(points, points[keepFrom:])
		points = points[:len(points)-keepFrom]
	}

	x, y := currentParticlePosition(data, p, currentTime-p.SpawnTime)
	head := TrailPoint{X: x, Y: y, CapturedAt: currentTime}
	if len(points) == 0 {
		p.TrailPoints = append(points, head)
		return
	}

	last := &points[len(points)-1]
	dx := head.X - last.X
	dy := head.Y - last.Y
	if dx*dx+dy*dy >= data.Trail.Params.MinPointDistance*data.Trail.Params.MinPointDistance {
		if len(points) == data.Trail.Params.MaxPoints {
			copy(points, points[1:])
			points[len(points)-1] = head
			p.TrailPoints = points
			return
		}
		p.TrailPoints = append(points, head)
		return
	}

	*last = head
	p.TrailPoints = points
}

func appendTrailMeshForPoints(data *SystemData, points []TrailPoint) {
	trail := &data.Trail
	if len(points) < 2 {
		return
	}
	vertexStart := len(trail.Runtime.Vertices)
	var fallbackNX, fallbackNY float32 = 0, -1
	lastIndex := len(points) - 1
	for i := range points {
		p := points[i]
		nx, ny := trailNormal(points, i, fallbackNX, fallbackNY)
		fallbackNX, fallbackNY = nx, ny

		ageNorm := clamp01((data.CurrentTime - p.CapturedAt) / trail.Params.MaxPointAge)
		width := lerp(trail.Params.WidthStart, trail.Params.WidthEnd, ApplyEasing(ageNorm, trail.Params.WidthEasing))
		alpha := lerp(trail.Params.AlphaStart, trail.Params.AlphaEnd, ApplyEasing(ageNorm, trail.Params.AlphaEasing))
		colorT := ApplyEasing(ageNorm, trail.Params.ColorEasing)
		r := lerp(trail.Params.ColorStartR, trail.Params.ColorEndR, colorT)
		g := lerp(trail.Params.ColorStartG, trail.Params.ColorEndG, colorT)
		b := lerp(trail.Params.ColorStartB, trail.Params.ColorEndB, colorT)

		halfWidth := width * 0.5
		ox := nx * halfWidth
		oy := ny * halfWidth
		v := float32(i) / float32(lastIndex)

		trail.Runtime.Vertices = append(trail.Runtime.Vertices,
			ebiten.Vertex{DstX: p.X - ox, DstY: p.Y - oy, SrcX: 0, SrcY: v, ColorR: r, ColorG: g, ColorB: b, ColorA: alpha},
			ebiten.Vertex{DstX: p.X + ox, DstY: p.Y + oy, SrcX: 1, SrcY: v, ColorR: r, ColorG: g, ColorB: b, ColorA: alpha},
		)
	}

	for i := 0; i < lastIndex; i++ {
		base := uint16(vertexStart + i*2)
		trail.Runtime.Indices = append(trail.Runtime.Indices,
			base, base+1, base+2,
			base+1, base+3, base+2,
		)
	}
}

func pruneTrailGhosts(trail *TrailRuntime, maxPointAge, currentTime float32) {
	if len(trail.Ghosts) == 0 {
		return
	}
	pruneBefore := currentTime - maxPointAge

	// Ghosts are appended in detach order, so fully expired ghosts accumulate at the front.
	firstAlive := 0
	for firstAlive < len(trail.Ghosts) {
		points := trail.Ghosts[firstAlive].Points
		if len(points) >= 2 && points[len(points)-1].CapturedAt >= pruneBefore {
			break
		}
		firstAlive++
	}
	if firstAlive > 0 {
		copy(trail.Ghosts, trail.Ghosts[firstAlive:])
		trail.Ghosts = trail.Ghosts[:len(trail.Ghosts)-firstAlive]
	}
	if len(trail.Ghosts) == 0 {
		return
	}

	writeIdx := 0
	for i := range trail.Ghosts {
		points := trail.Ghosts[i].Points
		if len(points) < 2 {
			continue
		}
		if points[0].CapturedAt >= pruneBefore {
			if writeIdx != i {
				copy(trail.Ghosts[writeIdx:], trail.Ghosts[i:])
			}
			writeIdx += len(trail.Ghosts) - i
			break
		}

		keepFrom := 0
		for keepFrom < len(points) && points[keepFrom].CapturedAt < pruneBefore {
			keepFrom++
		}
		if keepFrom > 0 {
			copy(points, points[keepFrom:])
			points = points[:len(points)-keepFrom]
		}
		if len(points) < 2 {
			continue
		}
		trail.Ghosts[writeIdx].Points = points
		writeIdx++
	}
	trail.Ghosts = trail.Ghosts[:writeIdx]
}

func detachParticleTrail(trail *TrailData, points []TrailPoint) {
	if len(points) < 2 || trail.Params.MaxPointAge <= 0 {
		return
	}
	copied := make([]TrailPoint, len(points))
	copy(copied, points)
	trail.Runtime.Ghosts = append(trail.Runtime.Ghosts, TrailGhost{Points: copied})
}

func drawTrailBatch(screen *ebiten.Image, vertices []ebiten.Vertex, indices []uint16, op *ebiten.DrawTrianglesOptions) {
	if len(indices) == 0 {
		return
	}
	screen.DrawTriangles(vertices, indices, getTrailWhiteImage(), op)
}

func drawParticleTrails(screen *ebiten.Image, data *SystemData) {
	trail := &data.Trail
	builder := newParticleTrailBatchBuilder(screen, &trail.Runtime)

	for _, idx := range data.ActiveIndices {
		builder.Append(data, data.ParticlePool[idx].TrailPoints)
	}
	for _, ghost := range trail.Runtime.Ghosts {
		builder.Append(data, ghost.Points)
	}
	builder.Flush()
}

func newParticleTrailBatchBuilder(screen *ebiten.Image, runtime *TrailRuntime) particleTrailBatchBuilder {
	runtime.Vertices = runtime.Vertices[:0]
	runtime.Indices = runtime.Indices[:0]
	return particleTrailBatchBuilder{
		screen: screen,
		trail:  runtime,
	}
}

func (b *particleTrailBatchBuilder) Append(data *SystemData, points []TrailPoint) {
	if len(points) < 2 {
		return
	}
	neededVertices := len(points) * 2
	if len(b.trail.Vertices) > 0 && len(b.trail.Vertices)+neededVertices > maxTrailBatchVertices {
		b.Flush()
	}
	appendTrailMeshForPoints(data, points)
}

func (b *particleTrailBatchBuilder) Flush() {
	drawTrailBatch(b.screen, b.trail.Vertices, b.trail.Indices, &b.opts)
	b.trail.Vertices = b.trail.Vertices[:0]
	b.trail.Indices = b.trail.Indices[:0]
}

func trailNormal(points []TrailPoint, idx int, fallbackX, fallbackY float32) (float32, float32) {
	var dx, dy float32
	switch {
	case idx == 0:
		dx = points[1].X - points[0].X
		dy = points[1].Y - points[0].Y
	case idx == len(points)-1:
		dx = points[idx].X - points[idx-1].X
		dy = points[idx].Y - points[idx-1].Y
	default:
		dx = points[idx+1].X - points[idx-1].X
		dy = points[idx+1].Y - points[idx-1].Y
	}

	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length <= 0.0001 {
		return fallbackX, fallbackY
	}
	return -dy / length, dx / length
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func getTrailWhiteImage() *ebiten.Image {
	trailWhiteImageOnce.Do(func() {
		img := ebiten.NewImage(1, 1)
		img.Fill(color.White)
		trailWhiteImage = img
	})
	return trailWhiteImage
}
