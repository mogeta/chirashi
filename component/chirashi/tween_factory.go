package chirashi

import (
	"fmt"

	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

// TweenFactory creates tween animations from configuration
type TweenFactory struct{}

// NewTweenFactory creates a new TweenFactory instance
func NewTweenFactory() *TweenFactory {
	return &TweenFactory{}
}

// CreateSequence creates a gween.Sequence from TweenConfig
func (f *TweenFactory) CreateSequence(config TweenConfig, baseValue float64) *gween.Sequence {
	if len(config.Steps) == 0 {
		// Return a simple static sequence if no steps
		return gween.NewSequence(gween.New(float32(baseValue), float32(baseValue), 1, ease.Linear))
	}

	if config.Type == "single" && len(config.Steps) > 0 {
		// Create a single tween from the first step
		step := config.Steps[0]
		tween := f.CreateSingle(step, baseValue)
		return gween.NewSequence(tween)
	}

	// Create sequence with multiple steps
	var tweens []*gween.Tween
	currentValue := baseValue

	for _, step := range config.Steps {
		fromValue := currentValue
		if step.From != 0 || len(tweens) == 0 {
			// Use explicit from value for first step or when specified (relative to baseValue)
			fromValue = baseValue + step.From
		}

		// Calculate to value as relative to baseValue
		toValue := baseValue + step.To

		// Apply random range if specified
		if step.ToRange != nil {
			randomOffset := rangeFloat(step.ToRange.Min, step.ToRange.Max)
			toValue = baseValue + randomOffset
		}

		easingFunc := f.ParseEasing(step.Easing)
		tween := gween.New(float32(fromValue), float32(toValue), float32(step.Duration), easingFunc)
		tweens = append(tweens, tween)
		currentValue = toValue
	}

	return gween.NewSequence(tweens...)
}

// CreateSingle creates a single gween.Tween from TweenStep
func (f *TweenFactory) CreateSingle(step TweenStep, baseValue float64) *gween.Tween {
	fromValue := baseValue
	if step.From != 0 {
		// Use explicit from value (relative to baseValue)
		fromValue = baseValue + step.From
	}

	// Calculate to value as relative to baseValue
	toValue := baseValue + step.To

	// Apply random range if specified
	if step.ToRange != nil {
		randomOffset := rangeFloat(step.ToRange.Min, step.ToRange.Max)
		toValue = baseValue + randomOffset
	}

	easingFunc := f.ParseEasing(step.Easing)
	return gween.New(float32(fromValue), float32(toValue), float32(step.Duration), easingFunc)
}

// ParseEasing converts easing function name to ease.TweenFunc
func (f *TweenFactory) ParseEasing(name string) ease.TweenFunc {
	switch name {
	case "Linear":
		return ease.Linear
	case "InQuad":
		return ease.InQuad
	case "OutQuad":
		return ease.OutQuad
	case "InOutQuad":
		return ease.InOutQuad
	case "InCubic":
		return ease.InCubic
	case "OutCubic":
		return ease.OutCubic
	case "InOutCubic":
		return ease.InOutCubic
	case "InQuart":
		return ease.InQuart
	case "OutQuart":
		return ease.OutQuart
	case "InOutQuart":
		return ease.InOutQuart
	case "InQuint":
		return ease.InQuint
	case "OutQuint":
		return ease.OutQuint
	case "InOutQuint":
		return ease.InOutQuint
	case "InSine":
		return ease.InSine
	case "OutSine":
		return ease.OutSine
	case "InOutSine":
		return ease.InOutSine
	case "InExpo":
		return ease.InExpo
	case "OutExpo":
		return ease.OutExpo
	case "InOutExpo":
		return ease.InOutExpo
	case "InCirc":
		return ease.InCirc
	case "OutCirc":
		return ease.OutCirc
	case "InOutCirc":
		return ease.InOutCirc
	case "InBack":
		return ease.InBack
	case "OutBack":
		return ease.OutBack
	case "InOutBack":
		return ease.InOutBack
	default:
		// Default to Linear if unknown easing
		return ease.Linear
	}
}

// CreateSequenceFactory creates a SequenceFunc from TweenConfig
func (f *TweenFactory) CreateSequenceFactory(config TweenConfig, baseX, baseY float64) SequenceFunc {
	return func() *gween.Sequence {
		return f.CreateSequence(config, baseX)
	}
}

// CreateMovementFactories creates X and Y sequence factories from MovementConfig
func (f *TweenFactory) CreateMovementFactories(movement MovementConfig, baseX, baseY float64) (SequenceFunc, SequenceFunc) {
	xFactory := func() *gween.Sequence {
		return f.CreateSequence(movement.X, baseX)
	}

	yFactory := func() *gween.Sequence {
		return f.CreateSequence(movement.Y, baseY)
	}

	return xFactory, yFactory
}

// CreateDefaultSequence creates a default alpha sequence from AppearanceConfig
func (f *TweenFactory) CreateDefaultSequence(appearance AppearanceConfig) *gween.Sequence {
	if len(appearance.Alpha.Steps) == 0 {
		// Default fade out sequence
		return gween.NewSequence(gween.New(1.0, 1.0, 1.0, ease.OutCirc))
	}

	return f.CreateSequence(appearance.Alpha, 1.0)
}

// ValidateConfig validates that a TweenConfig can be properly converted
func (f *TweenFactory) ValidateConfig(config TweenConfig) error {
	if config.Type != "single" && config.Type != "sequence" {
		return fmt.Errorf("invalid tween type: %s", config.Type)
	}

	if len(config.Steps) == 0 {
		return fmt.Errorf("tween config must have at least one step")
	}

	for i, step := range config.Steps {
		if step.Duration <= 0 {
			return fmt.Errorf("step %d duration must be positive", i)
		}

		// Validate easing name by trying to parse it
		_ = f.ParseEasing(step.Easing)
	}

	return nil
}
