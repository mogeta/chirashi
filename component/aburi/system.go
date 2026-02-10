package aburi

import (
	"fmt"
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

func NewSystem() *System {
	return &System{
		query: donburi.NewQuery(filter.Contains(Component)),
		cnt:   0,
	}
}

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

	params := &data.AnimParams
	currentTime := data.CurrentTime

	// Debug: log first spawn
	if data.Metrics.SpawnCount == 0 {
		fmt.Printf("First spawn: UsePolar=%v, Angle(%.2f-%.2f), Dist(%.0f-%.0f), Color(%.1f,%.1f,%.1f)->(%.1f,%.1f,%.1f)\n",
			params.UsePolar,
			params.AngleMin, params.AngleMax,
			params.DistanceMin, params.DistanceMax,
			params.StartR, params.StartG, params.StartB,
			params.EndR, params.EndG, params.EndB)
	}

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
		particle.Duration = params.DurationBase
		if params.DurationRange > 0 {
			particle.Duration += (rand.Float32()*2 - 1) * params.DurationRange
		}

		// Position
		if params.UsePolar {
			// Polar mode: convert to cartesian at spawn time (no per-frame cost)
			angle := rangeFloat32(params.AngleMin, params.AngleMax)
			dist := rangeFloat32(params.DistanceMin, params.DistanceMax)
			cos, sin := cosf(angle), sinf(angle)

			particle.StartX = data.EmitterX
			particle.StartY = data.EmitterY
			particle.EndX = data.EmitterX + dist*cos
			particle.EndY = data.EmitterY + dist*sin
		} else {
			// Cartesian mode
			particle.StartX = data.EmitterX + rangeFloat32(params.StartXMin, params.StartXMax)
			particle.EndX = data.EmitterX + rangeFloat32(params.EndXMin, params.EndXMax)
			particle.StartY = data.EmitterY + rangeFloat32(params.StartYMin, params.StartYMax)
			particle.EndY = data.EmitterY + rangeFloat32(params.EndYMin, params.EndYMax)
		}
		particle.PositionEasing = params.PositionEasing

		// Alpha
		particle.StartAlpha = params.StartAlpha
		particle.EndAlpha = params.EndAlpha
		particle.AlphaEasing = params.AlphaEasing

		// Scale
		particle.StartScale = params.StartScale
		particle.EndScale = params.EndScale
		particle.ScaleEasing = params.ScaleEasing

		// Rotation
		particle.StartRotation = params.StartRotation
		particle.EndRotation = params.EndRotation
		particle.RotationEasing = params.RotationEasing

		// Color
		particle.StartR = params.StartR
		particle.StartG = params.StartG
		particle.StartB = params.StartB
		particle.EndR = params.EndR
		particle.EndG = params.EndG
		particle.EndB = params.EndB
		particle.ColorEasing = params.ColorEasing

		particle.Active = true

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

			// Apply CPU easing for position, scale, rotation
			posT := ApplyEasing(normalizedT, p.PositionEasing)
			scaleT := ApplyEasing(normalizedT, p.ScaleEasing)
			rotT := ApplyEasing(normalizedT, p.RotationEasing)

			// Interpolate values
			x := lerp(p.StartX, p.EndX, posT)
			y := lerp(p.StartY, p.EndY, posT)
			scale := lerp(p.StartScale, p.EndScale, scaleT)
			rotation := lerp(p.StartRotation, p.EndRotation, rotT)

			// Calculate scaled dimensions
			scaledHalfW := halfW * scale
			scaledHalfH := halfH * scale

			// Calculate rotated corner positions
			cos := float32(1.0)
			sin := float32(0.0)
			if rotation != 0 {
				cos = cosf(rotation)
				sin = sinf(rotation)
			}

			// 4 corners relative to center, then rotated and translated
			// Top-left, Top-right, Bottom-left, Bottom-right
			corners := [4][2]float32{
				{-scaledHalfW, -scaledHalfH},
				{scaledHalfW, -scaledHalfH},
				{-scaledHalfW, scaledHalfH},
				{scaledHalfW, scaledHalfH},
			}

			// Prepare vertex custom data for GPU interpolation
			// color.r = startAlpha, color.g = endAlpha
			// color.b = alphaEasing (normalized), color.a = colorEasing (normalized)
			// custom.x = spawnTime, custom.y = duration
			// custom.z = startColor (packed RGB), custom.w = endColor (packed RGB)
			alphaEasingNorm := float32(p.AlphaEasing) / 25.0
			colorEasingNorm := float32(p.ColorEasing) / 25.0
			startColorPacked := packRGB(p.StartR, p.StartG, p.StartB)
			endColorPacked := packRGB(p.EndR, p.EndG, p.EndB)

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
					ColorR:  p.StartAlpha,
					ColorG:  p.EndAlpha,
					ColorB:  alphaEasingNorm,
					ColorA:  colorEasingNorm,
					Custom0: p.SpawnTime,
					Custom1: p.Duration,
					Custom2: startColorPacked,
					Custom3: endColorPacked,
				})
			}

			// Add indices for two triangles (0,1,2), (1,3,2)
			data.Indices = append(data.Indices,
				vertexBase+0, vertexBase+1, vertexBase+2,
				vertexBase+1, vertexBase+3, vertexBase+2,
			)
		}

		// Draw all particles in a single batch
		opts := &ebiten.DrawTrianglesShaderOptions{
			Uniforms: map[string]interface{}{
				"Time": currentTime,
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

// packRGB packs RGB (0-1) into a single float for shader transfer
func packRGB(r, g, b float32) float32 {
	ri := uint32(clampf(r, 0, 1) * 255)
	gi := uint32(clampf(g, 0, 1) * 255)
	bi := uint32(clampf(b, 0, 1) * 255)
	return float32(ri<<16 | gi<<8 | bi)
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

func cosf(radians float32) float32 {
	// Fast cosine approximation using Taylor series
	// For better accuracy, use math.Cos
	x := radians
	for x > 3.14159265 {
		x -= 6.28318530
	}
	for x < -3.14159265 {
		x += 6.28318530
	}
	x2 := x * x
	return 1 - x2/2 + x2*x2/24 - x2*x2*x2/720
}

func sinf(radians float32) float32 {
	// Fast sine approximation using Taylor series
	x := radians
	for x > 3.14159265 {
		x -= 6.28318530
	}
	for x < -3.14159265 {
		x += 6.28318530
	}
	x2 := x * x
	return x - x*x2/6 + x*x2*x2/120 - x*x2*x2*x2/5040
}
