package chirashi

import "github.com/yohamta/donburi"

// ApplyConfigLive updates an existing particle entity in place from config values.
// This preserves active particles where possible and updates spawn parameters for future particles.
func ApplyConfigLive(world donburi.World, entity donburi.Entity, config *ParticleConfig, x, y float32) {
	if !world.Valid(entity) {
		return
	}

	normalizeParticleConfig(config)

	entry := world.Entry(entity)
	data := Component.Get(entry)

	prevEmitterX := data.EmitterX
	prevEmitterY := data.EmitterY
	prevAnim := data.AnimParams

	data.EmitterX = x + config.Emitter.X
	data.EmitterY = y + config.Emitter.Y
	data.EmitterShape = buildEmitterShapeParams(config.Emitter.Shape)
	data.SpawnInterval = config.Spawn.Interval
	data.ParticlesPerSpawn = config.Spawn.ParticlesPerSpawn
	data.IsLoop = config.Spawn.IsLoop
	if !data.IsLoop {
		data.LifeTime = config.Spawn.LifeTime
	}

	data.AnimParams = buildAnimationParams(config)
	buildSequenceConfigs(config, data)

	shiftActiveParticlesForEmitterDelta(data, data.EmitterX-prevEmitterX, data.EmitterY-prevEmitterY)
	applyAnimationParamsToActiveParticles(data, prevAnim)
}

func shiftActiveParticlesForEmitterDelta(data *SystemData, dx, dy float32) {
	if dx == 0 && dy == 0 {
		return
	}
	for _, idx := range data.ActiveIndices {
		p := &data.ParticlePool[idx]
		p.StartX += dx
		p.EndX += dx
		p.StartY += dy
		p.EndY += dy
		p.ControlX += dx
		p.ControlY += dy
	}
}

func applyAnimationParamsToActiveParticles(data *SystemData, prev AnimationParams) {
	pos := data.AnimParams.Position
	app := data.AnimParams.Appearance
	clr := data.AnimParams.Color
	duration := data.AnimParams.Duration
	flowGain := float32(0)
	if pos.HasFlow {
		flowGain = (pos.FlowStrengthMin + pos.FlowStrengthMax) / 2
	}

	for _, idx := range data.ActiveIndices {
		p := &data.ParticlePool[idx]

		if p.Duration > 0 && duration.Base > 0 {
			elapsed := data.CurrentTime - p.SpawnTime
			progress := elapsed / p.Duration
			if progress < 0 {
				progress = 0
			}
			if progress > 1 {
				progress = 1
			}
			p.Duration = duration.Base
			p.SpawnTime = data.CurrentTime - progress*p.Duration
		}

		p.PositionEasing = pos.Easing
		p.AlphaEasing = app.AlphaEasing
		p.ScaleEasing = app.ScaleEasing
		p.RotationEasing = app.RotationEasing
		p.ColorEasing = clr.Easing

		if data.AlphaSeq == nil {
			p.HasAlphaSeq = false
			p.StartAlpha = app.StartAlpha
			p.EndAlpha = app.EndAlpha
		} else if prev.Appearance != app || prev.Duration != duration {
			p.HasAlphaSeq = true
			p.AlphaSnap = GenerateSnapshot(data.AlphaSeq, 0)
		}

		if data.ScaleSeq == nil {
			p.HasScaleSeq = false
			p.StartScale = app.StartScale
			p.EndScale = app.EndScale
		} else if prev.Appearance != app || prev.Duration != duration {
			p.HasScaleSeq = true
			p.ScaleSnap = GenerateSnapshot(data.ScaleSeq, 0)
		}

		if data.RotSeq == nil {
			p.HasRotSeq = false
			p.StartRotation = app.StartRotation
			p.EndRotation = app.EndRotation
		} else if prev.Appearance != app || prev.Duration != duration {
			p.HasRotSeq = true
			p.RotSnap = GenerateSnapshot(data.RotSeq, 0)
		}

		if data.PosXSeq != nil {
			p.HasPosXSeq = true
			p.PosXSnap = GenerateSnapshot(data.PosXSeq, p.StartX)
		} else {
			p.HasPosXSeq = false
		}
		if data.PosYSeq != nil {
			p.HasPosYSeq = true
			p.PosYSnap = GenerateSnapshot(data.PosYSeq, p.StartY)
		} else {
			p.HasPosYSeq = false
		}

		p.StartR = clr.StartR
		p.StartG = clr.StartG
		p.StartB = clr.StartB
		p.EndR = clr.EndR
		p.EndG = clr.EndG
		p.EndB = clr.EndB

		p.HasAttractor = pos.UseAttractor
		p.HasFlow = pos.HasFlow
		if pos.HasFlow {
			if !prev.Position.HasFlow {
				resetParticleFlowState(p, true)
			}
			p.FlowGain = flowGain
		} else {
			p.FlowGain = 0
			resetParticleFlowState(p, false)
		}
	}
}
