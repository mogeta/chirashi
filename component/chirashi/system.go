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
			data.LifeTime--
			if data.LifeTime <= 0 {
				ecs.World.Remove(entry.Entity())
			}
		}
	}
}

func (sys *System) spawn(data *SystemData) {
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
			particle.StartX = data.EmitterX
			particle.StartY = data.EmitterY
			particle.ControlX = data.EmitterX + rangeFloat32(pos.ControlXMin, pos.ControlXMax)
			particle.ControlY = data.EmitterY + rangeFloat32(pos.ControlYMin, pos.ControlYMax)
			particle.HasAttractor = true
		case pos.UsePolar:
			// Polar mode: convert to cartesian at spawn time (no per-frame cost)
			angle := rangeFloat32(pos.AngleMin, pos.AngleMax)
			dist := rangeFloat32(pos.DistMin, pos.DistMax)
			cos, sin := float32(math.Cos(float64(angle))), float32(math.Sin(float64(angle)))
			particle.StartX = data.EmitterX
			particle.StartY = data.EmitterY
			particle.EndX = data.EmitterX + dist*cos
			particle.EndY = data.EmitterY + dist*sin
			particle.HasAttractor = false
		default:
			// Cartesian mode
			particle.StartX = data.EmitterX + rangeFloat32(pos.StartXMin, pos.StartXMax)
			particle.EndX = data.EmitterX + rangeFloat32(pos.EndXMin, pos.EndXMax)
			particle.StartY = data.EmitterY + rangeFloat32(pos.StartYMin, pos.StartYMax)
			particle.EndY = data.EmitterY + rangeFloat32(pos.EndYMin, pos.EndYMax)
			particle.HasAttractor = false
		}
		particle.PositionEasing = pos.Easing

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
			particle.PosXSnap = GenerateSnapshot(data.PosXSeq, data.EmitterX)
		}
		particle.HasPosYSeq = data.PosYSeq != nil
		if particle.HasPosYSeq {
			particle.PosYSnap = GenerateSnapshot(data.PosYSeq, data.EmitterY)
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

func (sys *System) updateParticles(data *SystemData) {
	currentTime := data.CurrentTime
	indicesToRemove := []int{}

	// Check for expired particles
	for i := 0; i < len(data.ActiveIndices); i++ {
		particleIdx := data.ActiveIndices[i]
		particle := &data.ParticlePool[particleIdx]

		elapsed := currentTime - particle.SpawnTime
		if elapsed >= particle.Duration {
			particle.Active = false
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
			posT := ApplyEasing(normalizedT, p.PositionEasing)
			var x, y float32
			switch {
			case p.HasAttractor:
				// Quadratic bezier: B(t) = (1-t)²·P0 + 2(1-t)t·P1 + t²·P2
				u := 1 - posT
				x = u*u*p.StartX + 2*u*posT*p.ControlX + posT*posT*data.AttractorX
				y = u*u*p.StartY + 2*u*posT*p.ControlY + posT*posT*data.AttractorY
			case p.HasPosXSeq:
				x = EvaluateSequence(data.PosXSeq, &p.PosXSnap, elapsed)
				if p.HasPosYSeq {
					y = EvaluateSequence(data.PosYSeq, &p.PosYSnap, elapsed)
				} else {
					y = lerp(p.StartY, p.EndY, posT)
				}
			default:
				x = lerp(p.StartX, p.EndX, posT)
				if p.HasPosYSeq {
					y = EvaluateSequence(data.PosYSeq, &p.PosYSnap, elapsed)
				} else {
					y = lerp(p.StartY, p.EndY, posT)
				}
			}

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

func clampf(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}
