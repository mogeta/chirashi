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
	deltaTime := float32(1.0 / float64(ebiten.TPS()))

	for entry := range sys.query.Iter(ecs.World) {
		data := Component.Get(entry)

		startTime := time.Now()

		// Update current time
		data.CurrentTime += deltaTime

		// Spawn new particles
		sys.spawn(data)

		// Deactivate expired particles
		sys.updateParticles(data)

		// Update metrics
		data.Metrics.UpdateTimeUs = time.Since(startTime).Microseconds()
		data.Metrics.FrameCount++

		// Handle lifetime
		if !data.IsLoop {
			if data.LifeTime > 0 {
				data.LifeTime--
			}
			if data.LifeTime <= 0 && data.ActiveCount == 0 {
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

		spawnX, spawnY := sampleEmitterPosition(data.EmitterX, data.EmitterY, data.EmitterShape)

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
			// Polar mode: convert to cartesian at spawn time (no per-frame cost)
			angle := rangeFloat32(pos.AngleMin, pos.AngleMax)
			dist := rangeFloat32(pos.DistMin, pos.DistMax)
			cos, sin := float32(math.Cos(float64(angle))), float32(math.Sin(float64(angle)))
			particle.StartX = spawnX
			particle.StartY = spawnY
			particle.EndX = spawnX + dist*cos
			particle.EndY = spawnY + dist*sin
			particle.HasAttractor = false
		default:
			// Cartesian mode
			particle.StartX = spawnX + rangeFloat32(pos.StartXMin, pos.StartXMax)
			particle.EndX = spawnX + rangeFloat32(pos.EndXMin, pos.EndXMax)
			particle.StartY = spawnY + rangeFloat32(pos.StartYMin, pos.StartYMax)
			particle.EndY = spawnY + rangeFloat32(pos.EndYMin, pos.EndYMax)
			particle.HasAttractor = false
		}
		particle.PositionEasing = pos.Easing
		particle.NoisePhasePosX = rand.Float32() * 2 * math.Pi
		particle.NoisePhasePosY = rand.Float32() * 2 * math.Pi
		if pos.HasTurbulence {
			particle.TurbulenceGain = rangeFloat32(pos.TurbulenceStrengthMin, pos.TurbulenceStrengthMax)
			particle.TurbulenceOffsetX = rand.Float32()*32 - 16
			particle.TurbulenceOffsetY = rand.Float32()*32 - 16
		} else {
			particle.TurbulenceGain = 0
			particle.TurbulenceOffsetX = 0
			particle.TurbulenceOffsetY = 0
		}
		particle.NoisePhaseAlpha = rand.Float32() * 2 * math.Pi
		particle.NoisePhaseScale = rand.Float32() * 2 * math.Pi
		particle.NoisePhaseRotation = rand.Float32() * 2 * math.Pi

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

func sampleEmitterPosition(emitterX, emitterY float32, shape EmitterShapeParams) (float32, float32) {
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

func evaluateParticleBasePosition(data *SystemData, p *Instance, elapsed, normalizedT float32) (float32, float32) {
	posT := ApplyEasing(normalizedT, p.PositionEasing)
	switch {
	case p.HasAttractor:
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

func normalizedTimeAt(elapsed, duration float32) float32 {
	if duration <= 0 {
		return 1
	}
	t := elapsed / duration
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func sampleTurbulenceField(x, y, t float32, octaves int, persistence float32) (float32, float32) {
	amp := float32(1)
	freq := float32(1)
	sumAmp := float32(0)
	var vx, vy float32
	for i := 0; i < octaves; i++ {
		px := x * freq
		py := y * freq
		pt := t * (0.85 + 0.35*freq)
		vx += (float32(math.Sin(float64(py+pt))) + 0.5*float32(math.Sin(float64(px*0.7-py*1.3-pt*1.1)))) * amp
		vy += (float32(math.Cos(float64(px-pt))) + 0.5*float32(math.Cos(float64(py*0.6+px*1.2+pt*0.9)))) * amp
		sumAmp += amp
		amp *= persistence
		freq *= 2
	}
	if sumAmp == 0 {
		return 0, 0
	}
	return vx / sumAmp, vy / sumAmp
}

func sampleContinuousNoise(t, phase float32, octaves int) float32 {
	amp := float32(1)
	freq := float32(1)
	sumAmp := float32(0)
	value := float32(0)
	for i := 0; i < octaves; i++ {
		value += float32(math.Sin(float64((t+phase)*freq))) * amp
		value += 0.5 * float32(math.Sin(float64((t*0.73+phase*1.37)*freq+0.7))) * amp
		sumAmp += amp * 1.5
		amp *= 0.5
		freq *= 2
	}
	if sumAmp == 0 {
		return 0
	}
	return value / sumAmp
}

func applyNoiseOverlay(base, elapsed float32, noise NoiseParams, phase float32) float32 {
	if !noise.Enabled || noise.Amplitude == 0 {
		return base
	}
	t := elapsed * noise.Frequency * 2 * math.Pi
	return base + sampleContinuousNoise(t, phase+noise.Seed, noise.Octaves)*noise.Amplitude
}

func turbulenceEnvelope(pos PositionParams, normalizedT float32) float32 {
	return lerp(pos.TurbulenceEnvStart, pos.TurbulenceEnvEnd, ApplyEasing(normalizedT, pos.TurbulenceEnvEasing))
}

func applyTurbulence(data *SystemData, p *Instance, baseX, baseY, elapsed, normalizedT float32) (float32, float32) {
	pos := data.AnimParams.Position
	if !pos.HasTurbulence || p.TurbulenceGain == 0 {
		return baseX, baseY
	}

	sampleX := baseX
	sampleY := baseY
	if pos.TurbulenceLocalSpace {
		sampleX -= data.EmitterX
		sampleY -= data.EmitterY
	}

	t := elapsed * pos.TurbulenceTimeScale
	sampleX += pos.DomainDriftX*t + pos.DomainOrbitRadiusX*float32(math.Cos(float64(pos.DomainOrbitFrequency*t+pos.DomainOrbitPhase)))
	sampleY += pos.DomainDriftY*t + pos.DomainOrbitRadiusY*float32(math.Sin(float64(pos.DomainOrbitFrequency*t+pos.DomainOrbitPhase)))
	sampleX = sampleX/pos.TurbulenceScale + p.TurbulenceOffsetX
	sampleY = sampleY/pos.TurbulenceScale + p.TurbulenceOffsetY

	fieldX, fieldY := sampleTurbulenceField(sampleX, sampleY, t, pos.TurbulenceOctaves, pos.TurbulencePersistence)
	strength := p.TurbulenceGain * turbulenceEnvelope(pos, normalizedT)
	return baseX + fieldX*strength, baseY + fieldY*strength
}

func evaluateParticlePosition(data *SystemData, p *Instance, elapsed, normalizedT float32) (float32, float32) {
	baseX, baseY := evaluateParticleBasePosition(data, p, elapsed, normalizedT)
	baseX = applyNoiseOverlay(baseX, elapsed, data.AnimParams.Position.PositionNoiseX, p.NoisePhasePosX)
	baseY = applyNoiseOverlay(baseY, elapsed, data.AnimParams.Position.PositionNoiseY, p.NoisePhasePosY)
	return applyTurbulence(data, p, baseX, baseY, elapsed, normalizedT)
}

func evaluateParticlePositionAndVelocity(data *SystemData, p *Instance, elapsed, normalizedT float32) (float32, float32, float32, float32) {
	x, y := evaluateParticlePosition(data, p, elapsed, normalizedT)
	vx, vy := sampleParticleVelocity(data, p, elapsed, x, y)
	return x, y, vx, vy
}

func resolveParticleAlpha(data *SystemData, p *Instance, elapsed, normalizedT float32) float32 {
	if p.HasAlphaSeq {
		return applyNoiseOverlay(EvaluateSequence(data.AlphaSeq, &p.AlphaSnap, elapsed), elapsed, data.AnimParams.Appearance.AlphaNoise, p.NoisePhaseAlpha)
	}
	base := lerp(p.StartAlpha, p.EndAlpha, ApplyEasing(normalizedT, p.AlphaEasing))
	return applyNoiseOverlay(base, elapsed, data.AnimParams.Appearance.AlphaNoise, p.NoisePhaseAlpha)
}

func resolveParticleScale(data *SystemData, p *Instance, elapsed, normalizedT float32) float32 {
	base := float32(0)
	if p.HasScaleSeq {
		base = EvaluateSequence(data.ScaleSeq, &p.ScaleSnap, elapsed)
	} else {
		base = lerp(p.StartScale, p.EndScale, ApplyEasing(normalizedT, p.ScaleEasing))
	}
	return applyNoiseOverlay(base, elapsed, data.AnimParams.Appearance.ScaleNoise, p.NoisePhaseScale)
}

func resolveParticleRotation(data *SystemData, p *Instance, elapsed, normalizedT float32) float32 {
	base := float32(0)
	if p.HasRotSeq {
		base = EvaluateSequence(data.RotSeq, &p.RotSnap, elapsed)
	} else {
		base = lerp(p.StartRotation, p.EndRotation, ApplyEasing(normalizedT, p.RotationEasing))
	}
	return applyNoiseOverlay(base, elapsed, data.AnimParams.Appearance.RotationNoise, p.NoisePhaseRotation)
}

func sampleParticleVelocity(data *SystemData, p *Instance, elapsed, x, y float32) (float32, float32) {
	if p.Duration <= 0 {
		return p.EndX - p.StartX, p.EndY - p.StartY
	}
	sample := float32(1.0 / 120.0)
	if sample > p.Duration*0.25 {
		sample = p.Duration * 0.25
	}
	if sample <= 0 {
		return p.EndX - p.StartX, p.EndY - p.StartY
	}
	if elapsed+sample <= p.Duration {
		nextElapsed := elapsed + sample
		nextX, nextY := evaluateParticlePosition(data, p, nextElapsed, normalizedTimeAt(nextElapsed, p.Duration))
		return (nextX - x) / sample, (nextY - y) / sample
	}
	if elapsed-sample >= 0 {
		prevElapsed := elapsed - sample
		prevX, prevY := evaluateParticlePosition(data, p, prevElapsed, normalizedTimeAt(prevElapsed, p.Duration))
		return (x - prevX) / sample, (y - prevY) / sample
	}
	return p.EndX - p.StartX, p.EndY - p.StartY
}

func (sys *System) updateParticles(data *SystemData) {
	currentTime := data.CurrentTime
	writeIdx := 0

	for _, particleIdx := range data.ActiveIndices {
		particle := &data.ParticlePool[particleIdx]

		elapsed := currentTime - particle.SpawnTime
		if elapsed >= particle.Duration {
			particle.Active = false
			data.ActiveCount--
			data.Metrics.DeactivateCount++
			data.FreeIndices = append(data.FreeIndices, particleIdx)
			continue
		}
		data.ActiveIndices[writeIdx] = particleIdx
		writeIdx++
	}
	data.ActiveIndices = data.ActiveIndices[:writeIdx]
}

// Draw renders all particles using GPU batch rendering
func (sys *System) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	for entry := range sys.query.Iter(ecs.World) {
		data := Component.Get(entry)

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

			// Calculate position, scale, rotation (per-property sequence or lerp)
			x, y, _, _ := evaluateParticlePositionAndVelocity(data, p, elapsed, normalizedT)

			scale := resolveParticleScale(data, p, elapsed, normalizedT)
			rotation := resolveParticleRotation(data, p, elapsed, normalizedT)

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
			alpha := resolveParticleAlpha(data, p, elapsed, normalizedT)
			startAlpha, endAlpha, alphaEasingNorm := alpha, alpha, float32(0)

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
