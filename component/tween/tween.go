package tween

import (
	"github.com/tanema/gween"
	"github.com/yohamta/donburi"
)

type TweenData struct {
	SequenceX *gween.Sequence
	SequenceY *gween.Sequence
}

var Component = donburi.NewComponentType[TweenData]()
