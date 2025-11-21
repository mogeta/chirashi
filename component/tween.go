package component

import (
	"github.com/tanema/gween"
	"github.com/yohamta/donburi"
)

type TweenData struct {
	SequenceX *gween.Sequence
	SequenceY *gween.Sequence
}

var Tween = donburi.NewComponentType[TweenData]()
