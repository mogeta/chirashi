package aburi

import (
	"math"
	"strings"
)

// EasingType represents the type of easing function
type EasingType int

const (
	EasingLinear EasingType = iota
	EasingInQuad
	EasingOutQuad
	EasingInOutQuad
	EasingInCubic
	EasingOutCubic
	EasingInOutCubic
	EasingInQuart
	EasingOutQuart
	EasingInOutQuart
	EasingInQuint
	EasingOutQuint
	EasingInOutQuint
	EasingInSine
	EasingOutSine
	EasingInOutSine
	EasingInExpo
	EasingOutExpo
	EasingInOutExpo
	EasingInCirc
	EasingOutCirc
	EasingInOutCirc
	EasingInBack
	EasingOutBack
	EasingInOutBack
)

// ParseEasing converts a string to EasingType
func ParseEasing(s string) EasingType {
	switch strings.ToLower(s) {
	case "linear":
		return EasingLinear
	case "inquad":
		return EasingInQuad
	case "outquad":
		return EasingOutQuad
	case "inoutquad":
		return EasingInOutQuad
	case "incubic":
		return EasingInCubic
	case "outcubic":
		return EasingOutCubic
	case "inoutcubic":
		return EasingInOutCubic
	case "inquart":
		return EasingInQuart
	case "outquart":
		return EasingOutQuart
	case "inoutquart":
		return EasingInOutQuart
	case "inquint":
		return EasingInQuint
	case "outquint":
		return EasingOutQuint
	case "inoutquint":
		return EasingInOutQuint
	case "insine":
		return EasingInSine
	case "outsine":
		return EasingOutSine
	case "inoutsine":
		return EasingInOutSine
	case "inexpo":
		return EasingInExpo
	case "outexpo":
		return EasingOutExpo
	case "inoutexpo":
		return EasingInOutExpo
	case "incirc":
		return EasingInCirc
	case "outcirc":
		return EasingOutCirc
	case "inoutcirc":
		return EasingInOutCirc
	case "inback":
		return EasingInBack
	case "outback":
		return EasingOutBack
	case "inoutback":
		return EasingInOutBack
	default:
		return EasingLinear
	}
}

// String returns the string representation of EasingType
func (e EasingType) String() string {
	switch e {
	case EasingLinear:
		return "Linear"
	case EasingInQuad:
		return "InQuad"
	case EasingOutQuad:
		return "OutQuad"
	case EasingInOutQuad:
		return "InOutQuad"
	case EasingInCubic:
		return "InCubic"
	case EasingOutCubic:
		return "OutCubic"
	case EasingInOutCubic:
		return "InOutCubic"
	case EasingInQuart:
		return "InQuart"
	case EasingOutQuart:
		return "OutQuart"
	case EasingInOutQuart:
		return "InOutQuart"
	case EasingInQuint:
		return "InQuint"
	case EasingOutQuint:
		return "OutQuint"
	case EasingInOutQuint:
		return "InOutQuint"
	case EasingInSine:
		return "InSine"
	case EasingOutSine:
		return "OutSine"
	case EasingInOutSine:
		return "InOutSine"
	case EasingInExpo:
		return "InExpo"
	case EasingOutExpo:
		return "OutExpo"
	case EasingInOutExpo:
		return "InOutExpo"
	case EasingInCirc:
		return "InCirc"
	case EasingOutCirc:
		return "OutCirc"
	case EasingInOutCirc:
		return "InOutCirc"
	case EasingInBack:
		return "InBack"
	case EasingOutBack:
		return "OutBack"
	case EasingInOutBack:
		return "InOutBack"
	default:
		return "Linear"
	}
}

// ApplyEasing applies the easing function to a normalized time value (0-1)
func ApplyEasing(t float32, easing EasingType) float32 {
	switch easing {
	case EasingLinear:
		return t
	case EasingInQuad:
		return t * t
	case EasingOutQuad:
		return t * (2 - t)
	case EasingInOutQuad:
		if t < 0.5 {
			return 2 * t * t
		}
		return -1 + (4-2*t)*t
	case EasingInCubic:
		return t * t * t
	case EasingOutCubic:
		t1 := t - 1
		return t1*t1*t1 + 1
	case EasingInOutCubic:
		if t < 0.5 {
			return 4 * t * t * t
		}
		return (t-1)*(2*t-2)*(2*t-2) + 1
	case EasingInQuart:
		return t * t * t * t
	case EasingOutQuart:
		t1 := t - 1
		return 1 - t1*t1*t1*t1
	case EasingInOutQuart:
		if t < 0.5 {
			return 8 * t * t * t * t
		}
		t1 := t - 1
		return 1 - 8*t1*t1*t1*t1
	case EasingInQuint:
		return t * t * t * t * t
	case EasingOutQuint:
		t1 := t - 1
		return 1 + t1*t1*t1*t1*t1
	case EasingInOutQuint:
		if t < 0.5 {
			return 16 * t * t * t * t * t
		}
		t1 := t - 1
		return 1 + 16*t1*t1*t1*t1*t1
	case EasingInSine:
		return 1 - float32(math.Cos(float64(t)*math.Pi/2))
	case EasingOutSine:
		return float32(math.Sin(float64(t) * math.Pi / 2))
	case EasingInOutSine:
		return -(float32(math.Cos(float64(t)*math.Pi)) - 1) / 2
	case EasingInExpo:
		if t == 0 {
			return 0
		}
		return float32(math.Pow(2, float64(10*(t-1))))
	case EasingOutExpo:
		if t == 1 {
			return 1
		}
		return 1 - float32(math.Pow(2, float64(-10*t)))
	case EasingInOutExpo:
		if t == 0 {
			return 0
		}
		if t == 1 {
			return 1
		}
		if t < 0.5 {
			return float32(math.Pow(2, float64(20*t-10))) / 2
		}
		return (2 - float32(math.Pow(2, float64(-20*t+10)))) / 2
	case EasingInCirc:
		return 1 - float32(math.Sqrt(float64(1-t*t)))
	case EasingOutCirc:
		t1 := t - 1
		return float32(math.Sqrt(float64(1 - t1*t1)))
	case EasingInOutCirc:
		if t < 0.5 {
			return (1 - float32(math.Sqrt(float64(1-4*t*t)))) / 2
		}
		return (float32(math.Sqrt(float64(1-(2*t-2)*(2*t-2)))) + 1) / 2
	case EasingInBack:
		const c1 = 1.70158
		const c3 = c1 + 1
		return c3*t*t*t - c1*t*t
	case EasingOutBack:
		const c1 = 1.70158
		const c3 = c1 + 1
		t1 := t - 1
		return 1 + c3*t1*t1*t1 + c1*t1*t1
	case EasingInOutBack:
		const c1 = 1.70158
		const c2 = c1 * 1.525
		if t < 0.5 {
			return (4 * t * t * ((c2+1)*2*t - c2)) / 2
		}
		return ((2*t-2)*(2*t-2)*((c2+1)*(2*t-2)+c2) + 2) / 2
	default:
		return t
	}
}
