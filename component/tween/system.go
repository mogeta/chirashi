package tween

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

type System struct {
	query *donburi.Query
}

func NewSystem() *System {
	return &System{
		query: donburi.NewQuery(filter.Contains(Component)),
	}
}

func (t *System) Update(ecs *ecs.ECS) {
	for entry := range t.query.Iter(ecs.World) {
		tweenData := Component.Get(entry)

		// Position コンポーネントの取得
		if !entry.HasComponent(PositionComponent) {
			continue
		}
		pos := PositionComponent.Get(entry)

		if tweenData.SequenceX != nil && tweenData.SequenceY != nil {
			x, _, isFinishedX := tweenData.SequenceX.Update(1.0 / 60.0)
			y, _, isFinishedY := tweenData.SequenceY.Update(1.0 / 60.0)
			pos.X = float64(x)
			pos.Y = float64(y)

			if isFinishedX && isFinishedY {
				entry.RemoveComponent(Component)
			}
		}
	}
}
