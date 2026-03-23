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
	data.EmitterX = x + config.Emitter.X
	data.EmitterY = y + config.Emitter.Y
	data.EmitterLocalSpace = config.Emitter.Space != EmitterSpaceWorld
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
	applyAnimationParamsToActiveParticles(data)
}

func shiftActiveParticlesForEmitterDelta(data *SystemData, dx, dy float32) {
	if !data.EmitterLocalSpace || (dx == 0 && dy == 0) {
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

func applyAnimationParamsToActiveParticles(data *SystemData) {
	pos := data.AnimParams.Position
	app := data.AnimParams.Appearance
	clr := data.AnimParams.Color
	duration := data.AnimParams.Duration

	for _, idx := range data.ActiveIndices {
		p := &data.ParticlePool[idx]
		applyLiveDuration(data, p, duration)
		applyLiveEasing(p, pos, app, clr)
		applyLiveAppearance(data, p, app)
		applyLivePositionSequences(data, p)
		applyLiveColor(p, clr)
		applyLiveFlow(p, pos)
	}
}

func applyLiveDuration(data *SystemData, p *Instance, duration DurationParams) {
	if p.Duration <= 0 || duration.Base <= 0 {
		return
	}
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

func applyLiveEasing(p *Instance, pos PositionParams, app AppearanceParams, clr ColorParams) {
	p.PositionEasing = pos.Easing
	p.AlphaEasing = app.AlphaEasing
	p.ScaleEasing = app.ScaleEasing
	p.RotationEasing = app.RotationEasing
	p.ColorEasing = clr.Easing
}

func applyLiveAppearance(data *SystemData, p *Instance, app AppearanceParams) {
	if data.AlphaSeq == nil {
		p.HasAlphaSeq = false
		p.StartAlpha = app.StartAlpha
		p.EndAlpha = app.EndAlpha
	} else {
		p.HasAlphaSeq = true
		p.AlphaSnap = GenerateSnapshot(data.AlphaSeq, 0)
	}

	if data.ScaleSeq == nil {
		p.HasScaleSeq = false
		p.StartScale = app.StartScale
		p.EndScale = app.EndScale
	} else {
		p.HasScaleSeq = true
		p.ScaleSnap = GenerateSnapshot(data.ScaleSeq, 0)
	}

	if data.RotSeq == nil {
		p.HasRotSeq = false
		p.StartRotation = app.StartRotation
		p.EndRotation = app.EndRotation
	} else {
		p.HasRotSeq = true
		p.RotSnap = GenerateSnapshot(data.RotSeq, 0)
	}
}

func applyLivePositionSequences(data *SystemData, p *Instance) {
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
}

func applyLiveColor(p *Instance, clr ColorParams) {
	p.StartR = clr.StartR
	p.StartG = clr.StartG
	p.StartB = clr.StartB
	p.EndR = clr.EndR
	p.EndG = clr.EndG
	p.EndB = clr.EndB
}

func applyLiveFlow(p *Instance, pos PositionParams) {
	flowGain := float32(0)
	if pos.HasFlow {
		flowGain = (pos.FlowStrengthMin + pos.FlowStrengthMax) / 2
	}
	p.HasAttractor = pos.UseAttractor
	p.HasFlow = pos.HasFlow
	if pos.HasFlow {
		if p.FlowGain == 0 {
			resetParticleFlowState(p, true)
		}
		p.FlowGain = flowGain
		return
	}
	p.FlowGain = 0
	resetParticleFlowState(p, false)
}
