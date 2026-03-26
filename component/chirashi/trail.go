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
	defaultTrailWidthStart       = float32(18)
	defaultTrailWidthEnd         = float32(0)
	defaultTrailAlphaStart       = float32(0.8)
	defaultTrailAlphaEnd         = float32(0)
	maxTrailBatchVertices        = 65535
)

var (
	trailWhiteImage     *ebiten.Image
	trailWhiteImageOnce sync.Once
)

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

	widthStart := config.Width.Start
	widthEnd := config.Width.End
	if widthStart == 0 && widthEnd == 0 {
		widthStart = defaultTrailWidthStart
		widthEnd = defaultTrailWidthEnd
	}

	alphaStart := config.Alpha.Start
	alphaEnd := config.Alpha.End
	if alphaStart == 0 && alphaEnd == 0 {
		alphaStart = defaultTrailAlphaStart
		alphaEnd = defaultTrailAlphaEnd
	}

	trail := TrailData{
		Enabled:          true,
		Mode:             config.Mode,
		LocalSpace:       config.Space == "local",
		MaxPoints:        maxPoints,
		MinPointDistance: minPointDistance,
		MaxPointAge:      maxPointAge,
		WidthStart:       widthStart,
		WidthEnd:         widthEnd,
		WidthEasing:      ParseEasing(config.Width.Easing),
		AlphaStart:       alphaStart,
		AlphaEnd:         alphaEnd,
		AlphaEasing:      ParseEasing(config.Alpha.Easing),
		ColorStartR:      1,
		ColorStartG:      1,
		ColorStartB:      1,
		ColorEndR:        1,
		ColorEndG:        1,
		ColorEndB:        1,
		ColorEasing:      EasingLinear,
		Points:           make([]TrailPoint, 0, maxPoints),
		Ghosts:           make([]TrailGhost, 0),
		Vertices:         make([]ebiten.Vertex, 0, maxPoints*2),
		Indices:          make([]uint16, 0, (maxPoints-1)*6),
	}
	if config.Color != nil {
		trail.ColorStartR = config.Color.StartR
		trail.ColorStartG = config.Color.StartG
		trail.ColorStartB = config.Color.StartB
		trail.ColorEndR = config.Color.EndR
		trail.ColorEndG = config.Color.EndG
		trail.ColorEndB = config.Color.EndB
		trail.ColorEasing = ParseEasing(config.Color.Easing)
	}
	return trail
}

func isParticleTrail(trail *TrailData) bool {
	return trail.Enabled && trail.Mode == "particle"
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
	if !trail.Enabled || trail.MaxPoints < 2 {
		return
	}
	if isParticleTrail(trail) {
		updateParticleTrails(data)
		return
	}

	currentTime := data.CurrentTime
	pruneBefore := currentTime - trail.MaxPointAge
	keepFrom := 0
	for keepFrom < len(trail.Points) && trail.Points[keepFrom].CapturedAt < pruneBefore {
		keepFrom++
	}
	if keepFrom > 0 {
		copy(trail.Points, trail.Points[keepFrom:])
		trail.Points = trail.Points[:len(trail.Points)-keepFrom]
	}

	head := TrailPoint{X: data.EmitterX, Y: data.EmitterY, CapturedAt: currentTime}
	if len(trail.Points) == 0 {
		trail.Points = append(trail.Points, head)
		return
	}

	last := &trail.Points[len(trail.Points)-1]
	dx := head.X - last.X
	dy := head.Y - last.Y
	if dx*dx+dy*dy >= trail.MinPointDistance*trail.MinPointDistance {
		if len(trail.Points) == trail.MaxPoints {
			copy(trail.Points, trail.Points[1:])
			trail.Points[len(trail.Points)-1] = head
			return
		}
		trail.Points = append(trail.Points, head)
		return
	}

	// Keep the trail head pinned to the current emitter position while preserving older history.
	*last = head
}

func trailHasVisiblePoints(data *SystemData) bool {
	if !data.Trail.Enabled {
		return false
	}
	if isParticleTrail(&data.Trail) {
		for _, idx := range data.ActiveIndices {
			if len(data.ParticlePool[idx].TrailPoints) >= 2 {
				return true
			}
		}
		for _, ghost := range data.Trail.Ghosts {
			if len(ghost.Points) >= 2 {
				return true
			}
		}
		return false
	}
	return len(data.Trail.Points) >= 2
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

	buildTrailMesh(data)
	if len(trail.Indices) == 0 {
		return
	}

	op := &ebiten.DrawTrianglesOptions{}
	drawTrailBatch(screen, trail.Vertices, trail.Indices, op)
}

func buildTrailMesh(data *SystemData) {
	trail := &data.Trail
	trail.Vertices = trail.Vertices[:0]
	trail.Indices = trail.Indices[:0]

	if trail.MaxPointAge <= 0 {
		return
	}
	if isParticleTrail(trail) {
		buildParticleTrailMesh(data)
		return
	}
	if len(trail.Points) < 2 {
		return
	}

	var fallbackNX, fallbackNY float32 = 0, -1
	lastIndex := len(trail.Points) - 1
	for i := range trail.Points {
		p := trail.Points[i]
		nx, ny := trailNormal(trail.Points, i, fallbackNX, fallbackNY)
		fallbackNX, fallbackNY = nx, ny

		ageNorm := clamp01((data.CurrentTime - p.CapturedAt) / trail.MaxPointAge)
		width := lerp(trail.WidthStart, trail.WidthEnd, ApplyEasing(ageNorm, trail.WidthEasing))
		alpha := lerp(trail.AlphaStart, trail.AlphaEnd, ApplyEasing(ageNorm, trail.AlphaEasing))
		colorT := ApplyEasing(ageNorm, trail.ColorEasing)
		r := lerp(trail.ColorStartR, trail.ColorEndR, colorT)
		g := lerp(trail.ColorStartG, trail.ColorEndG, colorT)
		b := lerp(trail.ColorStartB, trail.ColorEndB, colorT)

		halfWidth := width * 0.5
		ox := nx * halfWidth
		oy := ny * halfWidth
		v := float32(i) / float32(lastIndex)

		trail.Vertices = append(trail.Vertices,
			ebiten.Vertex{
				DstX:   p.X - ox,
				DstY:   p.Y - oy,
				SrcX:   0,
				SrcY:   v,
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: alpha,
			},
			ebiten.Vertex{
				DstX:   p.X + ox,
				DstY:   p.Y + oy,
				SrcX:   1,
				SrcY:   v,
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: alpha,
			},
		)
	}

	for i := 0; i < lastIndex; i++ {
		base := uint16(i * 2)
		trail.Indices = append(trail.Indices,
			base, base+1, base+2,
			base+1, base+3, base+2,
		)
	}
}

func updateParticleTrails(data *SystemData) {
	currentTime := data.CurrentTime
	pruneTrailGhosts(&data.Trail, currentTime)
	for _, idx := range data.ActiveIndices {
		p := &data.ParticlePool[idx]
		updateSingleParticleTrail(data, p, currentTime)
	}
}

func updateSingleParticleTrail(data *SystemData, p *Instance, currentTime float32) {
	points := p.TrailPoints
	pruneBefore := currentTime - data.Trail.MaxPointAge
	keepFrom := 0
	for keepFrom < len(points) && points[keepFrom].CapturedAt < pruneBefore {
		keepFrom++
	}
	if keepFrom > 0 {
		copy(points, points[keepFrom:])
		points = points[:len(points)-keepFrom]
	}

	x, y := evaluateTrailAnchorPosition(data, p, currentTime)
	head := TrailPoint{X: x, Y: y, CapturedAt: currentTime}
	if len(points) == 0 {
		p.TrailPoints = append(points, head)
		return
	}

	last := &points[len(points)-1]
	dx := head.X - last.X
	dy := head.Y - last.Y
	if dx*dx+dy*dy >= data.Trail.MinPointDistance*data.Trail.MinPointDistance {
		if len(points) == data.Trail.MaxPoints {
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

func buildParticleTrailMesh(data *SystemData) {
	for _, idx := range data.ActiveIndices {
		appendTrailMeshForPoints(data, data.ParticlePool[idx].TrailPoints)
	}
	for _, ghost := range data.Trail.Ghosts {
		appendTrailMeshForPoints(data, ghost.Points)
	}
}

func appendTrailMeshForPoints(data *SystemData, points []TrailPoint) {
	trail := &data.Trail
	if len(points) < 2 {
		return
	}
	vertexStart := len(trail.Vertices)
	var fallbackNX, fallbackNY float32 = 0, -1
	lastIndex := len(points) - 1
	for i := range points {
		p := points[i]
		nx, ny := trailNormal(points, i, fallbackNX, fallbackNY)
		fallbackNX, fallbackNY = nx, ny

		ageNorm := clamp01((data.CurrentTime - p.CapturedAt) / trail.MaxPointAge)
		width := lerp(trail.WidthStart, trail.WidthEnd, ApplyEasing(ageNorm, trail.WidthEasing))
		alpha := lerp(trail.AlphaStart, trail.AlphaEnd, ApplyEasing(ageNorm, trail.AlphaEasing))
		colorT := ApplyEasing(ageNorm, trail.ColorEasing)
		r := lerp(trail.ColorStartR, trail.ColorEndR, colorT)
		g := lerp(trail.ColorStartG, trail.ColorEndG, colorT)
		b := lerp(trail.ColorStartB, trail.ColorEndB, colorT)

		halfWidth := width * 0.5
		ox := nx * halfWidth
		oy := ny * halfWidth
		v := float32(i) / float32(lastIndex)

		trail.Vertices = append(trail.Vertices,
			ebiten.Vertex{DstX: p.X - ox, DstY: p.Y - oy, SrcX: 0, SrcY: v, ColorR: r, ColorG: g, ColorB: b, ColorA: alpha},
			ebiten.Vertex{DstX: p.X + ox, DstY: p.Y + oy, SrcX: 1, SrcY: v, ColorR: r, ColorG: g, ColorB: b, ColorA: alpha},
		)
	}

	for i := 0; i < lastIndex; i++ {
		base := uint16(vertexStart + i*2)
		trail.Indices = append(trail.Indices,
			base, base+1, base+2,
			base+1, base+3, base+2,
		)
	}
}

func pruneTrailGhosts(trail *TrailData, currentTime float32) {
	if len(trail.Ghosts) == 0 {
		return
	}
	writeIdx := 0
	pruneBefore := currentTime - trail.MaxPointAge
	for _, ghost := range trail.Ghosts {
		points := ghost.Points
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
	if len(points) < 2 || trail.MaxPointAge <= 0 {
		return
	}
	copied := make([]TrailPoint, len(points))
	copy(copied, points)
	trail.Ghosts = append(trail.Ghosts, TrailGhost{Points: copied})
}

func drawTrailBatch(screen *ebiten.Image, vertices []ebiten.Vertex, indices []uint16, op *ebiten.DrawTrianglesOptions) {
	if len(indices) == 0 {
		return
	}
	screen.DrawTriangles(vertices, indices, getTrailWhiteImage(), op)
}

func drawParticleTrails(screen *ebiten.Image, data *SystemData) {
	trail := &data.Trail
	trail.Vertices = trail.Vertices[:0]
	trail.Indices = trail.Indices[:0]

	op := &ebiten.DrawTrianglesOptions{}

	flush := func() {
		drawTrailBatch(screen, trail.Vertices, trail.Indices, op)
		trail.Vertices = trail.Vertices[:0]
		trail.Indices = trail.Indices[:0]
	}

	appendPoints := func(points []TrailPoint) {
		if len(points) < 2 {
			return
		}
		neededVertices := len(points) * 2
		if len(trail.Vertices) > 0 && len(trail.Vertices)+neededVertices > maxTrailBatchVertices {
			flush()
		}
		appendTrailMeshForPoints(data, points)
	}

	for _, idx := range data.ActiveIndices {
		appendPoints(data.ParticlePool[idx].TrailPoints)
	}
	for _, ghost := range trail.Ghosts {
		appendPoints(ghost.Points)
	}
	flush()
}

func evaluateTrailAnchorPosition(data *SystemData, p *Instance, currentTime float32) (float32, float32) {
	elapsed := currentTime - p.SpawnTime
	normalizedT := elapsed / p.Duration
	if normalizedT < 0 {
		normalizedT = 0
	}
	if normalizedT > 1 {
		normalizedT = 1
	}
	posT := ApplyEasing(normalizedT, p.PositionEasing)
	x, y := evaluateParticleBasePosition(data, p, elapsed, posT)
	if p.HasFlow {
		x += p.FlowOffsetX
		y += p.FlowOffsetY
	}
	return x, y
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
