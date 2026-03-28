package chirashi

import (
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

const (
	defaultDeltaTime        = float32(1.0 / 60.0)
	flowSeedRange           = float32(32)
	flowSeedHalfRange       = flowSeedRange / 2
	curlNoiseEpsilon        = float32(0.05)
	flowTimeBaseFactor      = float32(0.75)
	flowTimeFrequencyGain   = float32(0.25)
	flowPrimaryTimeScale    = float32(0.9)
	flowSecondaryAmplitude  = float32(0.7)
	flowSecondarySpaceScale = float32(1.3)
	flowSecondaryTimeScale  = float32(1.1)
	flowTertiaryAmplitude   = float32(0.5)
	flowTertiaryXScale      = float32(0.8)
	flowTertiaryYScale      = float32(1.1)
	flowTertiaryTimeScale   = float32(0.6)
	flowQuaternaryAmplitude = float32(0.35)
	flowQuaternaryXScale    = float32(1.7)
	flowQuaternaryYScale    = float32(0.6)
	flowQuaternaryTimeScale = float32(0.4)
)

// System manages GPU-based particle systems with batch rendering
type System struct {
	query *donburi.Query
	cnt   int
}

// NewSystem creates a particle ECS system that updates and draws particle entities.
func NewSystem() *System {
	return &System{
		query: donburi.NewQuery(filter.Contains(Component)),
		cnt:   0,
	}
}

// Update advances particle simulation for all entities with the particle component.
func (sys *System) Update(ecs *ecs.ECS) {
	sys.cnt++
	tps := ebiten.TPS()
	deltaTime := defaultDeltaTime
	if tps > 0 {
		deltaTime = float32(1.0 / float64(tps))
	}

	for entry := range sys.query.Iter(ecs.World) {
		data := Component.Get(entry)

		startTime := time.Now()

		// Update current time
		data.CurrentTime += deltaTime

		// Spawn new particles
		sys.spawn(data)

		// Deactivate expired particles
		sys.updateParticles(data, deltaTime)
		updateTrail(data)

		// Update metrics
		data.Metrics.UpdateTimeUs = time.Since(startTime).Microseconds()
		data.Metrics.FrameCount++

		// Handle lifetime
		if !data.IsLoop {
			if data.LifeTime > 0 {
				data.LifeTime--
			}
			if data.LifeTime <= 0 && data.ActiveCount == 0 && !trailHasVisiblePoints(data) {
				ecs.World.Remove(entry.Entity())
			}
		}
	}
}

func (sys *System) spawn(data *SystemData) {
	if !data.IsLoop && data.LifeTime <= 0 {
		return
	}
	if data.SpawnInterval <= 0 || sys.cnt%data.SpawnInterval != 0 {
		return
	}

	dur := &data.AnimParams.Duration
	pos := &data.AnimParams.Position
	app := &data.AnimParams.Appearance
	clr := &data.AnimParams.Color
	currentTime := data.CurrentTime

	for i := 0; i < data.ParticlesPerSpawn && data.ActiveCount < data.MaxParticles; i++ {
		// O(1) free index retrieval
		if len(data.FreeIndices) == 0 {
			break
		}

		// Pop from free indices stack
		freeIdx := data.FreeIndices[len(data.FreeIndices)-1]
		data.FreeIndices = data.FreeIndices[:len(data.FreeIndices)-1]

		particle := &data.ParticlePool[freeIdx]
		particle.TrailPoints = particle.TrailPoints[:0]

		spawnX, spawnY := sampleEmitterPosition(data.EmitterX, data.EmitterY, data.EmitterShape, data.EmitterVector, i, data.ParticlesPerSpawn)

		// Initialize particle with randomized values
		particle.SpawnTime = currentTime
		particle.Duration = dur.Base
		if dur.Range > 0 {
			particle.Duration += (rand.Float32()*2 - 1) * dur.Range
		}

		// Position
		switch {
		case pos.UseAttractor:
			// Attractor mode: quadratic bezier P0=emitter, P1=random control, P2=AttractorX/Y
			// EndX/Y are unused; attractor coords are read from SystemData each frame.
			particle.StartX = spawnX
			particle.StartY = spawnY
			particle.ControlX = spawnX + rangeFloat32(pos.ControlXMin, pos.ControlXMax)
			particle.ControlY = spawnY + rangeFloat32(pos.ControlYMin, pos.ControlYMax)
			particle.HasAttractor = true
		case pos.UsePolar:
			angle := rangeFloat32(pos.AngleMin, pos.AngleMax)
			cosA, sinA := float32(math.Cos(float64(angle))), float32(math.Sin(float64(angle)))
			particle.StartX = spawnX
			particle.StartY = spawnY
			particle.HasAttractor = false
			if pos.UsePolarVelocity {
				// Velocity mode: duration = lifetime only, position driven by speed/angular_speed
				particle.DirX = cosA
				particle.DirY = sinA
				particle.StartAngle = angle
				particle.SpawnDist = rangeFloat32(pos.DistMin, pos.DistMax)
				particle.Speed = rangeFloat32(pos.SpeedMin, pos.SpeedMax)
				particle.AngularSpeed = rangeFloat32(pos.AngularSpeedMin, pos.AngularSpeedMax)
				particle.HasPolarVelocity = true
			} else {
				// Legacy lerp mode: convert to cartesian at spawn time
				dist := rangeFloat32(pos.DistMin, pos.DistMax)
				particle.EndX = spawnX + dist*cosA
				particle.EndY = spawnY + dist*sinA
				particle.HasPolarVelocity = false
			}
		default:
			// Cartesian mode
			particle.StartX = spawnX + rangeFloat32(pos.StartXMin, pos.StartXMax)
			particle.EndX = spawnX + rangeFloat32(pos.EndXMin, pos.EndXMax)
			particle.StartY = spawnY + rangeFloat32(pos.StartYMin, pos.StartYMax)
			particle.EndY = spawnY + rangeFloat32(pos.EndYMin, pos.EndYMax)
			particle.HasAttractor = false
		}
		particle.CurrentX = particle.StartX
		particle.CurrentY = particle.StartY
		particle.CurrentPosValid = true
		particle.CurrentPosTime = currentTime
		particle.PositionEasing = pos.Easing
		particle.HasFlow = pos.HasFlow
		if pos.HasFlow {
			particle.FlowGain = rangeFloat32(pos.FlowStrengthMin, pos.FlowStrengthMax)
			resetParticleFlowState(particle, true)
		} else {
			resetParticleFlowState(particle, false)
		}

		// Appearance
		particle.StartAlpha = app.StartAlpha
		particle.EndAlpha = app.EndAlpha
		particle.AlphaEasing = app.AlphaEasing
		particle.StartScale = app.StartScale
		particle.EndScale = app.EndScale
		particle.ScaleEasing = app.ScaleEasing
		particle.StartRotation = app.StartRotation
		particle.EndRotation = app.EndRotation
		particle.RotationEasing = app.RotationEasing

		// Color
		particle.StartR = clr.StartR
		particle.StartG = clr.StartG
		particle.StartB = clr.StartB
		particle.EndR = clr.EndR
		particle.EndG = clr.EndG
		particle.EndB = clr.EndB
		particle.ColorEasing = clr.Easing

		particle.Active = true

		// Initialize per-property sequence snapshots
		particle.HasPosXSeq = data.PosXSeq != nil
		if particle.HasPosXSeq {
			particle.PosXSnap = GenerateSnapshot(data.PosXSeq, spawnX)
		}
		particle.HasPosYSeq = data.PosYSeq != nil
		if particle.HasPosYSeq {
			particle.PosYSnap = GenerateSnapshot(data.PosYSeq, spawnY)
		}
		particle.HasScaleSeq = data.ScaleSeq != nil
		if particle.HasScaleSeq {
			particle.ScaleSnap = GenerateSnapshot(data.ScaleSeq, 0)
		}
		particle.HasRotSeq = data.RotSeq != nil
		if particle.HasRotSeq {
			particle.RotSnap = GenerateSnapshot(data.RotSeq, 0)
		}
		particle.HasAlphaSeq = data.AlphaSeq != nil
		if particle.HasAlphaSeq {
			particle.AlphaSnap = GenerateSnapshot(data.AlphaSeq, 0)
		}

		// Add to active indices
		data.ActiveIndices = append(data.ActiveIndices, freeIdx)
		data.ActiveCount++
		data.Metrics.SpawnCount++
	}
}

func sampleEmitterPosition(emitterX, emitterY float32, shape EmitterShapeParams, vector EmitterVectorParams, spawnIndex, spawnTotal int) (float32, float32) {
	if vector.Enabled {
		return sampleEmitterVectorPosition(emitterX, emitterY, vector, spawnIndex, spawnTotal)
	}
	switch shape.Type {
	case EmitterShapeCircle:
		angle := sampleCircleAngle(shape.StartAngle, shape.EndAngle)
		radius := rangeFloat32(shape.RadiusMin, shape.RadiusMax)
		if !shape.FromEdge {
			minRadiusSq := shape.RadiusMin * shape.RadiusMin
			maxRadiusSq := shape.RadiusMax * shape.RadiusMax
			radius = float32(math.Sqrt(float64(minRadiusSq + rand.Float32()*(maxRadiusSq-minRadiusSq))))
		}
		return emitterX + radius*float32(math.Cos(float64(angle))), emitterY + radius*float32(math.Sin(float64(angle)))
	case EmitterShapeBox:
		halfW := shape.Width / 2
		halfH := shape.Height / 2
		if shape.FromEdge {
			perimeter := 2 * (shape.Width + shape.Height)
			if perimeter <= 0 {
				return emitterX, emitterY
			}
			d := rand.Float32() * perimeter
			switch {
			case d < shape.Width:
				return rotateOffset(emitterX, emitterY, d-halfW, -halfH, shape.Rotation)
			case d < shape.Width+shape.Height:
				return rotateOffset(emitterX, emitterY, halfW, d-shape.Width-halfH, shape.Rotation)
			case d < 2*shape.Width+shape.Height:
				return rotateOffset(emitterX, emitterY, halfW-(d-shape.Width-shape.Height), halfH, shape.Rotation)
			default:
				return rotateOffset(emitterX, emitterY, -halfW, halfH-(d-2*shape.Width-shape.Height), shape.Rotation)
			}
		}
		return rotateOffset(
			emitterX,
			emitterY,
			rangeFloat32(-halfW, halfW),
			rangeFloat32(-halfH, halfH),
			shape.Rotation,
		)
	case EmitterShapeLine:
		halfLen := shape.Length / 2
		return rotateOffset(
			emitterX,
			emitterY,
			rangeFloat32(-halfLen, halfLen),
			0,
			shape.Rotation,
		)
	default:
		return emitterX, emitterY
	}
}

func sampleEmitterVectorPosition(emitterX, emitterY float32, vector EmitterVectorParams, spawnIndex, spawnTotal int) (float32, float32) {
	switch vector.Type {
	case EmitterVectorRect:
		return sampleRectVectorPosition(emitterX, emitterY, vector.Rect, vector.Placement, spawnIndex, spawnTotal)
	case EmitterVectorPolyline:
		return samplePolylineVectorPosition(emitterX, emitterY, vector.Polyline, spawnIndex, spawnTotal)
	default:
		return emitterX, emitterY
	}
}

func sampleRectVectorPosition(emitterX, emitterY float32, rect EmitterVectorRectParams, placement EmitterVectorPlacement, spawnIndex, spawnTotal int) (float32, float32) {
	halfW := rect.Width / 2
	halfH := rect.Height / 2
	if halfW <= 0 || halfH <= 0 {
		return emitterX, emitterY
	}

	switch placement {
	case EmitterVectorSurface:
		perimeter := 2 * (rect.Width + rect.Height)
		if perimeter <= 0 {
			return emitterX, emitterY
		}
		t := stratifiedSampleRatio(spawnIndex, spawnTotal)
		d := t * perimeter
		switch {
		case d < rect.Width:
			return rotateOffset(emitterX, emitterY, d-halfW, -halfH, rect.Rotation)
		case d < rect.Width+rect.Height:
			return rotateOffset(emitterX, emitterY, halfW, d-rect.Width-halfH, rect.Rotation)
		case d < 2*rect.Width+rect.Height:
			return rotateOffset(emitterX, emitterY, halfW-(d-rect.Width-rect.Height), halfH, rect.Rotation)
		default:
			return rotateOffset(emitterX, emitterY, -halfW, halfH-(d-2*rect.Width-rect.Height), rect.Rotation)
		}
	default:
		cols := int(math.Ceil(math.Sqrt(float64(float32(maxInt(spawnTotal, 1)) * rect.Width / rect.Height))))
		rows := (maxInt(spawnTotal, 1) + cols - 1) / cols
		col := spawnIndex % cols
		row := spawnIndex / cols
		if row >= rows {
			row = rows - 1
		}
		x := ((float32(col)+0.5)/float32(cols))*rect.Width - halfW
		y := ((float32(row)+0.5)/float32(rows))*rect.Height - halfH
		return rotateOffset(emitterX, emitterY, x, y, rect.Rotation)
	}
}

func samplePolylineVectorPosition(emitterX, emitterY float32, polyline EmitterVectorPolylineParams, spawnIndex, spawnTotal int) (float32, float32) {
	if len(polyline.Points) < 2 || polyline.TotalLength <= 0 {
		return emitterX, emitterY
	}

	target := stratifiedSampleRatio(spawnIndex, spawnTotal) * polyline.TotalLength
	accumulated := float32(0)
	for i, segmentLength := range polyline.SegmentLengths {
		if segmentLength <= 0 {
			continue
		}
		next := accumulated + segmentLength
		if target <= next || i == len(polyline.SegmentLengths)-1 {
			start, end := polylineSegmentEndpoints(polyline, i)
			localT := (target - accumulated) / segmentLength
			if localT < 0 {
				localT = 0
			} else if localT > 1 {
				localT = 1
			}
			x := start.X + (end.X-start.X)*localT
			y := start.Y + (end.Y-start.Y)*localT
			return emitterX + x, emitterY + y
		}
		accumulated = next
	}

	last := polyline.Points[len(polyline.Points)-1]
	return emitterX + last.X, emitterY + last.Y
}

func polylineSegmentEndpoints(polyline EmitterVectorPolylineParams, index int) (EmitterVectorPointParams, EmitterVectorPointParams) {
	start := polyline.Points[index]
	if index == len(polyline.Points)-1 {
		return start, polyline.Points[0]
	}
	return start, polyline.Points[index+1]
}

func stratifiedSampleRatio(index, total int) float32 {
	if total <= 1 {
		return 0.5
	}
	return (float32(index) + 0.5) / float32(total)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sampleCircleAngle(startAngle, endAngle float32) float32 {
	tau := float32(2 * math.Pi)

	rawSpan := endAngle - startAngle
	if rawSpan >= tau-fullCircleEpsilon || rawSpan <= -tau+fullCircleEpsilon {
		return rand.Float32() * tau
	}

	start := normalizeAngle(startAngle)
	end := normalizeAngle(endAngle)
	span := end - start
	if span < 0 {
		span += tau
	}

	if span <= fullCircleEpsilon {
		if math.Abs(float64(rawSpan)) > float64(fullCircleEpsilon) {
			return rand.Float32() * tau
		}
		return start
	}

	return normalizeAngle(start + rand.Float32()*span)
}

func normalizeAngle(angle float32) float32 {
	tau := float32(2 * math.Pi)
	normalized := float32(math.Mod(float64(angle), float64(tau)))
	if normalized < 0 {
		normalized += tau
	}
	return normalized
}

func rotateOffset(originX, originY, offsetX, offsetY, rotation float32) (float32, float32) {
	if rotation == 0 {
		return originX + offsetX, originY + offsetY
	}
	cos := float32(math.Cos(float64(rotation)))
	sin := float32(math.Sin(float64(rotation)))
	return originX + offsetX*cos - offsetY*sin, originY + offsetX*sin + offsetY*cos
}

func (sys *System) updateParticles(data *SystemData, deltaTime float32) {
	currentTime := data.CurrentTime
	indicesToRemove := []int{}

	// Check for expired particles
	for i := 0; i < len(data.ActiveIndices); i++ {
		particleIdx := data.ActiveIndices[i]
		particle := &data.ParticlePool[particleIdx]

		elapsed := currentTime - particle.SpawnTime
		if particle.HasFlow && data.AnimParams.Position.HasFlow {
			normalizedT := elapsed / particle.Duration
			if normalizedT < 0 {
				normalizedT = 0
			}
			if normalizedT > 1 {
				normalizedT = 1
			}
			updateParticleFlow(data, particle, elapsed, normalizedT, deltaTime)
		}
		cacheParticleCurrentPosition(data, particle, elapsed)
		if elapsed >= particle.Duration {
			particle.Active = false
			if data.Trail.Params.Mode == "particle" {
				detachParticleTrail(&data.Trail, particle.TrailPoints)
			}
			particle.TrailPoints = particle.TrailPoints[:0]
			indicesToRemove = append(indicesToRemove, i)
			data.ActiveCount--
			data.Metrics.DeactivateCount++
			// Return to free indices pool
			data.FreeIndices = append(data.FreeIndices, particleIdx)
		}
	}

	// Remove finished particles from active indices (iterate backwards)
	for i := len(indicesToRemove) - 1; i >= 0; i-- {
		removeIdx := indicesToRemove[i]
		lastIdx := len(data.ActiveIndices) - 1
		data.ActiveIndices[removeIdx] = data.ActiveIndices[lastIdx]
		data.ActiveIndices = data.ActiveIndices[:lastIdx]
	}
}

// Draw renders all particles using GPU batch rendering
func (sys *System) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	for entry := range sys.query.Iter(ecs.World) {
		data := Component.Get(entry)

		if data.Trail.Params.Enabled {
			drawTrail(screen, data)
		}

		if data.SourceImage == nil || data.Shader == nil {
			continue
		}

		if len(data.ActiveIndices) == 0 {
			continue
		}

		startTime := time.Now()

		// Build vertex/index buffers
		data.Vertices = data.Vertices[:0]
		data.Indices = data.Indices[:0]

		currentTime := data.CurrentTime
		imgW := data.ImageWidth
		imgH := data.ImageHeight
		halfW := imgW / 2
		halfH := imgH / 2

		for _, particleIdx := range data.ActiveIndices {
			p := &data.ParticlePool[particleIdx]

			// Calculate normalized time
			elapsed := currentTime - p.SpawnTime
			normalizedT := elapsed / p.Duration
			if normalizedT > 1 {
				normalizedT = 1
			}

			// Position is cached during update for draw/trail reuse.
			x, y := currentParticlePosition(data, p, elapsed)

			var scale float32
			if p.HasScaleSeq {
				scale = EvaluateSequence(data.ScaleSeq, &p.ScaleSnap, elapsed)
			} else {
				scale = lerp(p.StartScale, p.EndScale, ApplyEasing(normalizedT, p.ScaleEasing))
			}

			var rotation float32
			if p.HasRotSeq {
				rotation = EvaluateSequence(data.RotSeq, &p.RotSnap, elapsed)
			} else {
				rotation = lerp(p.StartRotation, p.EndRotation, ApplyEasing(normalizedT, p.RotationEasing))
			}

			// Calculate scaled dimensions
			scaledHalfW := halfW * scale
			scaledHalfH := halfH * scale

			// Calculate rotated corner positions
			cos := float32(1.0)
			sin := float32(0.0)
			if rotation != 0 {
				cos = float32(math.Cos(float64(rotation)))
				sin = float32(math.Sin(float64(rotation)))
			}

			// 4 corners relative to center, then rotated and translated
			// Top-left, Top-right, Bottom-left, Bottom-right
			corners := [4][2]float32{
				{-scaledHalfW, -scaledHalfH},
				{scaledHalfW, -scaledHalfH},
				{-scaledHalfW, scaledHalfH},
				{scaledHalfW, scaledHalfH},
			}

			// Vertex custom data layout:
			// color.r = startAlpha, color.g = endAlpha, color.b = alphaEasing (normalized)
			// custom.x = spawnTime, custom.y = duration
			// Color and colorEasing are passed as shader uniforms (same for all particles in system)
			var startAlpha, endAlpha, alphaEasingNorm float32
			if p.HasAlphaSeq {
				// CPU-evaluated alpha: pass same value as both start and end
				cpuAlpha := EvaluateSequence(data.AlphaSeq, &p.AlphaSnap, elapsed)
				startAlpha = cpuAlpha
				endAlpha = cpuAlpha
				alphaEasingNorm = 0 // Linear (no-op since start==end)
			} else {
				startAlpha = p.StartAlpha
				endAlpha = p.EndAlpha
				alphaEasingNorm = float32(p.AlphaEasing) / float32(easingTypeCount)
			}

			vertexBase := uint16(len(data.Vertices))

			for i, corner := range corners {
				// Rotate
				rx := corner[0]*cos - corner[1]*sin
				ry := corner[0]*sin + corner[1]*cos

				// Translate
				vx := x + rx
				vy := y + ry

				// UV coordinates
				var u, v float32
				switch i {
				case 0: // Top-left
					u, v = 0, 0
				case 1: // Top-right
					u, v = imgW, 0
				case 2: // Bottom-left
					u, v = 0, imgH
				case 3: // Bottom-right
					u, v = imgW, imgH
				}

				data.Vertices = append(data.Vertices, ebiten.Vertex{
					DstX:    vx,
					DstY:    vy,
					SrcX:    u,
					SrcY:    v,
					ColorR:  startAlpha,
					ColorG:  endAlpha,
					ColorB:  alphaEasingNorm,
					Custom0: p.SpawnTime,
					Custom1: p.Duration,
				})
			}

			// Add indices for two triangles (0,1,2), (1,3,2)
			data.Indices = append(data.Indices,
				vertexBase+0, vertexBase+1, vertexBase+2,
				vertexBase+1, vertexBase+3, vertexBase+2,
			)
		}

		clr := &data.AnimParams.Color
		opts := &ebiten.DrawTrianglesShaderOptions{
			Uniforms: map[string]interface{}{
				"Time":        currentTime,
				"StartColor":  [3]float32{clr.StartR, clr.StartG, clr.StartB},
				"EndColor":    [3]float32{clr.EndR, clr.EndG, clr.EndB},
				"ColorEasing": float32(clr.Easing),
			},
			Images: [4]*ebiten.Image{data.SourceImage},
		}

		screen.DrawTrianglesShader(data.Vertices, data.Indices, data.Shader, opts)

		data.Metrics.DrawTimeUs = time.Since(startTime).Microseconds()
	}
}

// Helper functions
func rangeFloat32(min, max float32) float32 {
	if min == max {
		return min
	}
	return min + rand.Float32()*(max-min)
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func evaluateParticleBasePosition(data *SystemData, p *Instance, elapsed, posT float32) (float32, float32) {
	switch {
	case p.HasPolarVelocity:
		dist := p.SpawnDist + p.Speed*elapsed
		if p.AngularSpeed != 0 {
			// Spiral mode: angle rotates over time
			a := p.StartAngle + p.AngularSpeed*elapsed
			return p.StartX + float32(math.Cos(float64(a)))*dist,
				p.StartY + float32(math.Sin(float64(a)))*dist
		}
		// Straight radial
		return p.StartX + p.DirX*dist, p.StartY + p.DirY*dist
	case p.HasAttractor:
		// Quadratic bezier: B(t) = (1-t)^2*P0 + 2(1-t)t*P1 + t^2*P2
		u := 1 - posT
		return u*u*p.StartX + 2*u*posT*p.ControlX + posT*posT*data.AttractorX,
			u*u*p.StartY + 2*u*posT*p.ControlY + posT*posT*data.AttractorY
	case p.HasPosXSeq:
		x := EvaluateSequence(data.PosXSeq, &p.PosXSnap, elapsed)
		if p.HasPosYSeq {
			return x, EvaluateSequence(data.PosYSeq, &p.PosYSnap, elapsed)
		}
		return x, lerp(p.StartY, p.EndY, posT)
	default:
		x := lerp(p.StartX, p.EndX, posT)
		if p.HasPosYSeq {
			return x, EvaluateSequence(data.PosYSeq, &p.PosYSnap, elapsed)
		}
		return x, lerp(p.StartY, p.EndY, posT)
	}
}

func updateParticleFlow(data *SystemData, p *Instance, elapsed, normalizedT, deltaTime float32) {
	pos := data.AnimParams.Position
	if !pos.HasFlow || p.FlowGain == 0 {
		return
	}

	baseT := ApplyEasing(normalizedT, p.PositionEasing)
	baseX, baseY := evaluateParticleBasePosition(data, p, elapsed, baseT)
	sampleX := baseX + p.FlowOffsetX
	sampleY := baseY + p.FlowOffsetY
	if pos.FlowLocalSpace {
		sampleX -= data.EmitterX
		sampleY -= data.EmitterY
	}
	sampleX = sampleX/pos.FlowScale + p.FlowSeedX
	sampleY = sampleY/pos.FlowScale + p.FlowSeedY
	t := elapsed * pos.FlowTimeScale
	fieldX, fieldY := sampleCurlNoiseField(sampleX, sampleY, t, pos.FlowOctaves, pos.FlowPersistence)
	p.FlowVelX = p.FlowVelX*pos.FlowDrag + fieldX*p.FlowGain*deltaTime
	p.FlowVelY = p.FlowVelY*pos.FlowDrag + fieldY*p.FlowGain*deltaTime
	p.FlowOffsetX += p.FlowVelX * deltaTime
	p.FlowOffsetY += p.FlowVelY * deltaTime

	if pos.FlowRespawnOnEscape && pos.FlowBoundRadius > 0 {
		dx := baseX + p.FlowOffsetX - data.EmitterX
		dy := baseY + p.FlowOffsetY - data.EmitterY
		if dx*dx+dy*dy > pos.FlowBoundRadius*pos.FlowBoundRadius {
			resetParticleFlowState(p, true)
		}
	}
}

func cacheParticleCurrentPosition(data *SystemData, p *Instance, elapsed float32) {
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
	p.CurrentX = x
	p.CurrentY = y
	p.CurrentPosValid = true
	p.CurrentPosTime = data.CurrentTime
}

func currentParticlePosition(data *SystemData, p *Instance, elapsed float32) (float32, float32) {
	if p.CurrentPosValid && p.CurrentPosTime == data.CurrentTime {
		return p.CurrentX, p.CurrentY
	}
	cacheParticleCurrentPosition(data, p, elapsed)
	return p.CurrentX, p.CurrentY
}

func resetParticleFlowState(p *Instance, randomizeSeed bool) {
	p.FlowOffsetX = 0
	p.FlowOffsetY = 0
	p.FlowVelX = 0
	p.FlowVelY = 0
	if randomizeSeed {
		p.FlowSeedX = rand.Float32()*flowSeedRange - flowSeedHalfRange
		p.FlowSeedY = rand.Float32()*flowSeedRange - flowSeedHalfRange
		return
	}
	p.FlowSeedX = 0
	p.FlowSeedY = 0
}

func sampleCurlNoiseField(x, y, t float32, octaves int, persistence float32) (float32, float32) {
	nx1 := sampleFlowScalar(x+curlNoiseEpsilon, y, t, octaves, persistence)
	nx2 := sampleFlowScalar(x-curlNoiseEpsilon, y, t, octaves, persistence)
	ny1 := sampleFlowScalar(x, y+curlNoiseEpsilon, t, octaves, persistence)
	ny2 := sampleFlowScalar(x, y-curlNoiseEpsilon, t, octaves, persistence)
	ddx := (nx1 - nx2) / (2 * curlNoiseEpsilon)
	ddy := (ny1 - ny2) / (2 * curlNoiseEpsilon)
	return ddy, -ddx
}

func sampleFlowScalar(x, y, t float32, octaves int, persistence float32) float32 {
	amp := float32(1)
	freq := float32(1)
	sumAmp := float32(0)
	value := float32(0)
	for i := 0; i < octaves; i++ {
		px := x * freq
		py := y * freq
		pt := t * (flowTimeBaseFactor + flowTimeFrequencyGain*freq)
		value += float32(math.Sin(float64(px+pt*flowPrimaryTimeScale))) * amp
		value += flowSecondaryAmplitude * float32(math.Cos(float64(py*flowSecondarySpaceScale-pt*flowSecondaryTimeScale))) * amp
		value += flowTertiaryAmplitude * float32(math.Sin(float64((px*flowTertiaryXScale+py*flowTertiaryYScale)+pt*flowTertiaryTimeScale))) * amp
		value += flowQuaternaryAmplitude * float32(math.Cos(float64((px*flowQuaternaryXScale-py*flowQuaternaryYScale)-pt*flowQuaternaryTimeScale))) * amp
		sumAmp += amp
		amp *= persistence
		freq *= 2
	}
	if sumAmp == 0 {
		return 0
	}
	return value / sumAmp
}
