package chirashi

import (
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

const (
	// maxParticleBatchVertices keeps vertex counts addressable by uint16
	// indices (65535); the largest multiple of 4 below that is 65532.
	maxParticleBatchVertices = 65532

	defaultDeltaTime        = float32(1.0 / 60.0)
	flowSeedRange           = float32(32)
	flowSeedHalfRange       = flowSeedRange / 2
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

	emissionScale := clampEmissionScale(data.EmissionScale)
	if emissionScale <= 0 {
		return
	}

	maxParticles := scaledMaxParticles(data.MaxParticles, emissionScale)
	if data.ActiveCount >= maxParticles {
		return
	}

	// Preserve fractional emission so low scales still work for presets that
	// spawn one particle at a time. For example, scale 0.5 emits one particle
	// every other configured spawn tick instead of rounding down to zero.
	data.emissionRemainder += float32(data.ParticlesPerSpawn) * emissionScale
	particlesToSpawn := int(data.emissionRemainder)
	if particlesToSpawn <= 0 {
		return
	}
	data.emissionRemainder -= float32(particlesToSpawn)

	dur := &data.AnimParams.Duration
	pos := &data.AnimParams.Position
	app := &data.AnimParams.Appearance
	clr := &data.AnimParams.Color
	currentTime := data.CurrentTime

	for i := 0; i < particlesToSpawn && data.ActiveCount < maxParticles; i++ {
		if data.ActiveCount >= len(data.ParticlePool) {
			break
		}

		// The pool is dense: the next free slot sits right after the actives.
		particle := &data.ParticlePool[data.ActiveCount]
		particle.TrailPoints = particle.TrailPoints[:0]

		spawnX, spawnY := sampleEmitterPosition(data.EmitterX, data.EmitterY, data.EmitterShape, data.EmitterVector, i, particlesToSpawn)

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
			sinA, cosA := fastSincos(angle)
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
		assignParticleColor(particle, clr)

		particle.Active = true

		// Initialize per-property sequence snapshots, reusing pooled slices
		particle.HasPosXSeq = data.PosXSeq != nil
		if particle.HasPosXSeq {
			FillSnapshot(data.PosXSeq, &particle.PosXSnap, spawnX)
		}
		particle.HasPosYSeq = data.PosYSeq != nil
		if particle.HasPosYSeq {
			FillSnapshot(data.PosYSeq, &particle.PosYSnap, spawnY)
		}
		particle.HasScaleSeq = data.ScaleSeq != nil
		if particle.HasScaleSeq {
			FillSnapshot(data.ScaleSeq, &particle.ScaleSnap, 0)
		}
		particle.HasRotSeq = data.RotSeq != nil
		if particle.HasRotSeq {
			FillSnapshot(data.RotSeq, &particle.RotSnap, 0)
		}
		particle.HasAlphaSeq = data.AlphaSeq != nil
		if particle.HasAlphaSeq {
			FillSnapshot(data.AlphaSeq, &particle.AlphaSnap, 0)
		}

		data.ActiveCount++
		data.Metrics.SpawnCount++
	}
}

func clampEmissionScale(scale float32) float32 {
	if math.IsNaN(float64(scale)) {
		return 1
	}
	if scale < 0 {
		return 0
	}
	if scale > 1 {
		return 1
	}
	return scale
}

func scaledMaxParticles(maxParticles int, scale float32) int {
	if maxParticles <= 0 || scale <= 0 {
		return 0
	}
	scaled := int(float32(maxParticles) * scale)
	if scaled < 1 {
		return 1
	}
	return scaled
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
		sin, cos := fastSincos(angle)
		return emitterX + radius*cos, emitterY + radius*sin
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
	sin, cos := fastSincos(rotation)
	return originX + offsetX*cos - offsetY*sin, originY + offsetX*sin + offsetY*cos
}

func (sys *System) updateParticles(data *SystemData, deltaTime float32) {
	currentTime := data.CurrentTime

	// Simulate first (parallel-capable), then sweep expired particles.
	simulateActiveParticles(data, deltaTime)

	for i := 0; i < data.ActiveCount; i++ {
		particle := &data.ParticlePool[i]

		elapsed := currentTime - particle.SpawnTime
		if elapsed >= particle.Duration {
			particle.Active = false
			if data.Trail.Params.Mode == "particle" {
				detachParticleTrail(&data.Trail, particle.TrailPoints)
			}
			particle.TrailPoints = particle.TrailPoints[:0]
			data.ActiveCount--
			data.Metrics.DeactivateCount++
			// Swap the last active particle into this slot to keep the pool
			// dense; the freed slot keeps its recycled buffers.
			if i != data.ActiveCount {
				data.ParticlePool[i], data.ParticlePool[data.ActiveCount] = data.ParticlePool[data.ActiveCount], data.ParticlePool[i]
				i--
			}
		}
	}
}

// Active-particle counts above which the simulation phase is split across
// goroutines. Below them, goroutine and synchronization overhead outweighs
// the gain. Flow-field sampling is an order of magnitude heavier per
// particle, so it pays off much earlier.
const (
	parallelSimulateThresholdFlow  = 256
	parallelSimulateThresholdPlain = 4096
)

// simulateActiveParticles advances flow state and caches positions for all
// active particles. Each particle only touches its own state, so the work is
// split across goroutines for large pools.
func simulateActiveParticles(data *SystemData, deltaTime float32) {
	n := data.ActiveCount
	if n == 0 {
		return
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > 8 {
		workers = 8
	}
	threshold := parallelSimulateThresholdPlain
	if data.AnimParams.Position.HasFlow {
		threshold = parallelSimulateThresholdFlow
	}
	if n < threshold || workers <= 1 {
		simulateParticleRange(data, data.ParticlePool[:n], deltaTime)
		return
	}

	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for start := 0; start < n; start += chunk {
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(particles []Instance) {
			defer wg.Done()
			simulateParticleRange(data, particles, deltaTime)
		}(data.ParticlePool[start:end])
	}
	wg.Wait()
}

func simulateParticleRange(data *SystemData, particles []Instance, deltaTime float32) {
	currentTime := data.CurrentTime
	hasFlow := data.AnimParams.Position.HasFlow
	for i := range particles {
		particle := &particles[i]
		elapsed := currentTime - particle.SpawnTime
		if particle.HasFlow && hasFlow {
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
	}
}

// Draw renders all particles using GPU batch rendering
func (sys *System) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	for entry := range sys.query.Iter(ecs.World) {
		data := Component.Get(entry)

		if data.Trail.Params.Enabled {
			drawTrail(screen, data)
		}

		if data.SourceImage == nil {
			continue
		}

		if data.ActiveCount == 0 {
			continue
		}

		startTime := time.Now()

		// Build vertex buffer; the quad index pattern is static and grown once.
		data.Vertices = data.Vertices[:0]
		batchQuads := data.ActiveCount
		if batchQuads > maxParticleBatchVertices/4 {
			batchQuads = maxParticleBatchVertices / 4
		}
		ensureQuadIndices(data, batchQuads)

		currentTime := data.CurrentTime
		imgW := data.ImageWidth
		imgH := data.ImageHeight
		halfW := imgW / 2
		halfH := imgH / 2

		// Colors are fully evaluated on the CPU into vertex data, so the
		// shader-less path renders with plain DrawTriangles and benefits
		// from Ebiten's internal draw-command batching across systems.
		var shaderOpts *ebiten.DrawTrianglesShaderOptions
		var plainOpts *ebiten.DrawTrianglesOptions
		if data.Shader != nil {
			shaderOpts = &ebiten.DrawTrianglesShaderOptions{
				Uniforms: data.ShaderUniforms,
				Images:   [4]*ebiten.Image{data.SourceImage},
				Blend:    data.Blend,
			}
		} else {
			plainOpts = &ebiten.DrawTrianglesOptions{
				ColorScaleMode: ebiten.ColorScaleModeStraightAlpha,
				Blend:          data.Blend,
			}
		}
		flush := func() {
			indices := data.Indices[:len(data.Vertices)/4*6]
			if data.Shader != nil {
				screen.DrawTrianglesShader(data.Vertices, indices, data.Shader, shaderOpts)
			} else {
				screen.DrawTriangles(data.Vertices, indices, data.SourceImage, plainOpts)
			}
			data.Vertices = data.Vertices[:0]
		}

		for particleIdx := 0; particleIdx < data.ActiveCount; particleIdx++ {
			// Flush before the vertex count overflows uint16 indices.
			if len(data.Vertices) >= maxParticleBatchVertices {
				flush()
			}
			p := &data.ParticlePool[particleIdx]

			// Calculate normalized time
			elapsed := currentTime - p.SpawnTime
			normalizedT := elapsed / p.Duration
			if normalizedT < 0 {
				normalizedT = 0
			}
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
				sin, cos = fastSincos(rotation)
			}

			// Evaluate alpha and tint per particle; vertex colors carry the
			// final straight-alpha color (custom.x carries normalized time
			// for effect shaders such as blur).
			var alpha float32
			if p.HasAlphaSeq {
				alpha = EvaluateSequence(data.AlphaSeq, &p.AlphaSnap, elapsed)
			} else {
				alpha = lerp(p.StartAlpha, p.EndAlpha, ApplyEasing(normalizedT, p.AlphaEasing))
			}
			colorT := ApplyEasing(normalizedT, p.ColorEasing)
			tintR := lerp(p.StartR, p.EndR, colorT)
			tintG := lerp(p.StartG, p.EndG, colorT)
			tintB := lerp(p.StartB, p.EndB, colorT)

			// Rotated half extents; the four corners are +/- combinations.
			// Top-left, Top-right, Bottom-left, Bottom-right
			wx := scaledHalfW * cos
			wy := scaledHalfW * sin
			hx := -scaledHalfH * sin
			hy := scaledHalfH * cos

			vertex := ebiten.Vertex{
				ColorR:  tintR,
				ColorG:  tintG,
				ColorB:  tintB,
				ColorA:  alpha,
				Custom0: normalizedT,
			}
			vertex.DstX, vertex.DstY = x-wx-hx, y-wy-hy
			vertex.SrcX, vertex.SrcY = 0, 0
			data.Vertices = append(data.Vertices, vertex)
			vertex.DstX, vertex.DstY = x+wx-hx, y+wy-hy
			vertex.SrcX, vertex.SrcY = imgW, 0
			data.Vertices = append(data.Vertices, vertex)
			vertex.DstX, vertex.DstY = x-wx+hx, y-wy+hy
			vertex.SrcX, vertex.SrcY = 0, imgH
			data.Vertices = append(data.Vertices, vertex)
			vertex.DstX, vertex.DstY = x+wx+hx, y+wy+hy
			vertex.SrcX, vertex.SrcY = imgW, imgH
			data.Vertices = append(data.Vertices, vertex)
		}

		if len(data.Vertices) > 0 {
			flush()
		}

		data.Metrics.DrawTimeUs = time.Since(startTime).Microseconds()
	}
}

// ensureQuadIndices grows the static quad index buffer to cover quadCount
// quads. The pattern (two triangles per quad) never changes, so it is built
// once and sliced per draw call instead of being rebuilt every frame.
func ensureQuadIndices(data *SystemData, quadCount int) {
	if quadCount > maxParticleBatchVertices/4 {
		quadCount = maxParticleBatchVertices / 4
	}
	for i := len(data.Indices) / 6; i < quadCount; i++ {
		base := uint16(i * 4)
		data.Indices = append(data.Indices,
			base+0, base+1, base+2,
			base+1, base+3, base+2,
		)
	}
}

// assignParticleColor sets a particle's color pair from config, mixing toward
// the variation pair by one random factor when variation is enabled.
func assignParticleColor(particle *Instance, clr *ColorParams) {
	particle.ColorVariationMix = 0
	if clr.HasVariation {
		particle.ColorVariationMix = rand.Float32()
	}
	applyParticleColor(particle, clr)
}

func applyParticleColor(particle *Instance, clr *ColorParams) {
	if clr.HasVariation {
		t := particle.ColorVariationMix
		particle.StartR = lerp(clr.StartR, clr.Start2R, t)
		particle.StartG = lerp(clr.StartG, clr.Start2G, t)
		particle.StartB = lerp(clr.StartB, clr.Start2B, t)
		particle.EndR = lerp(clr.EndR, clr.End2R, t)
		particle.EndG = lerp(clr.EndG, clr.End2G, t)
		particle.EndB = lerp(clr.EndB, clr.End2B, t)
	} else {
		particle.StartR = clr.StartR
		particle.StartG = clr.StartG
		particle.StartB = clr.StartB
		particle.EndR = clr.EndR
		particle.EndG = clr.EndG
		particle.EndB = clr.EndB
	}
	particle.ColorEasing = clr.Easing
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
			sin, cos := fastSincos(a)
			return p.StartX + cos*dist,
				p.StartY + sin*dist
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

// sampleCurlNoiseField returns the curl (ddy, -ddx) of the flow scalar field.
// The field is a sum of sin/cos terms, so its gradient has a closed form:
// evaluating it analytically needs 4 trig calls per octave instead of the 16
// a central-difference approximation costs.
func sampleCurlNoiseField(x, y, t float32, octaves int, persistence float32) (float32, float32) {
	amp := float32(1)
	freq := float32(1)
	sumAmp := float32(0)
	ddx := float32(0)
	ddy := float32(0)
	for i := 0; i < octaves; i++ {
		px := x * freq
		py := y * freq
		pt := t * (flowTimeBaseFactor + flowTimeFrequencyGain*freq)

		// d/dx sin(px + pt*k) = cos(...) * freq
		cos1 := fastCos(px + pt*flowPrimaryTimeScale)
		ddx += cos1 * freq * amp

		// d/dy 0.7*cos(py*1.3 - pt*k) = -0.7*sin(...) * 1.3 * freq
		sin2 := fastSin(py*flowSecondarySpaceScale - pt*flowSecondaryTimeScale)
		ddy += -flowSecondaryAmplitude * sin2 * flowSecondarySpaceScale * freq * amp

		// d/d{x,y} 0.5*sin(px*0.8 + py*1.1 + pt*k) = 0.5*cos(...) * {0.8,1.1} * freq
		cos3 := fastCos(px*flowTertiaryXScale + py*flowTertiaryYScale + pt*flowTertiaryTimeScale)
		ddx += flowTertiaryAmplitude * cos3 * flowTertiaryXScale * freq * amp
		ddy += flowTertiaryAmplitude * cos3 * flowTertiaryYScale * freq * amp

		// d/d{x,y} 0.35*cos(px*1.7 - py*0.6 - pt*k) = 0.35*-sin(...) * {1.7,-0.6} * freq
		sin4 := fastSin(px*flowQuaternaryXScale - py*flowQuaternaryYScale - pt*flowQuaternaryTimeScale)
		ddx += -flowQuaternaryAmplitude * sin4 * flowQuaternaryXScale * freq * amp
		ddy += flowQuaternaryAmplitude * sin4 * flowQuaternaryYScale * freq * amp

		sumAmp += amp
		amp *= persistence
		freq *= 2
	}
	if sumAmp == 0 {
		return 0, 0
	}
	return ddy / sumAmp, -ddx / sumAmp
}
