package tween

import (
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"github.com/yohamta/donburi"
)

// PositionComponent は system から移動
type PositionData struct {
	X, Y float64
}

var PositionComponent = donburi.NewComponentType[PositionData]()

// CreateMoveTween creates a movement tween animation for an entity
func CreateMoveTween(entry *donburi.Entry, fromX, fromY, toX, toY float64, duration float32, easingFunc func(float32, float32, float32, float32) float32) {
	if !entry.HasComponent(Component) {
		entry.AddComponent(Component)
	}

	tweenData := Component.Get(entry)

	// X軸の移動Tween
	tweenData.SequenceX = gween.NewSequence(
		gween.New(float32(fromX), float32(toX), duration, easingFunc),
	)

	// Y軸の移動Tween
	tweenData.SequenceY = gween.NewSequence(
		gween.New(float32(fromY), float32(toY), duration, easingFunc),
	)
}

// CreateShakeTween creates a shake effect tween animation for an entity
func CreateShakeTween(entry *donburi.Entry, centerX, centerY, shakeAmount float64, duration float32) {
	if !entry.HasComponent(Component) {
		entry.AddComponent(Component)
	}

	tweenData := Component.Get(entry)

	// シェイクエフェクト用のランダム関数
	shakeFunc := func(t, b, c, d float32) float32 {
		// ランダムなシェイク値を生成
		offset := float32(shakeAmount * (2*float64(t/d) - 1))
		return b + offset
	}

	tweenData.SequenceX = gween.NewSequence(
		gween.New(float32(centerX), float32(centerX), duration, shakeFunc),
	)

	tweenData.SequenceY = gween.NewSequence(
		gween.New(float32(centerY), float32(centerY), duration, shakeFunc),
	)
}

// CreateAttackMoveTween creates a complete attack animation sequence (move -> attack -> return)
func CreateAttackMoveTween(entry *donburi.Entry, originalX, originalY, targetX, targetY float64) {
	if !entry.HasComponent(Component) {
		entry.AddComponent(Component)
	}

	// Position コンポーネントが必要
	if !entry.HasComponent(PositionComponent) {
		entry.AddComponent(PositionComponent)
		pos := PositionComponent.Get(entry)
		pos.X = originalX
		pos.Y = originalY
	}

	tweenData := Component.Get(entry)

	// 攻撃アニメーション: 移動 -> 少し待機 -> 復帰
	tweenData.SequenceX = gween.NewSequence(
		// 1. 攻撃対象へ移動 (0.3秒, イーズアウト)
		gween.New(float32(originalX), float32(targetX), 0.3, ease.OutQuad),
		// 2. 攻撃位置で少し待機 (0.1秒)
		gween.New(float32(targetX), float32(targetX), 1.0, ease.Linear),
		// 3. 元位置へ復帰 (0.4秒, イーズイン)
		gween.New(float32(targetX), float32(originalX), 0.4, ease.InQuad),
	)

	tweenData.SequenceY = gween.NewSequence(
		// Y軸も同様のシーケンス
		gween.New(float32(originalY), float32(targetY), 0.3, ease.OutQuad),
		gween.New(float32(targetY), float32(targetY), 1.0, ease.Linear),
		gween.New(float32(targetY), float32(originalY), 0.4, ease.InQuad),
	)
}

// CreateBounceBackTween creates a bounce back animation (damage effect)
func CreateBounceBackTween(entry *donburi.Entry, centerX, centerY, bounceDistance float64) {
	if !entry.HasComponent(Component) {
		entry.AddComponent(Component)
	}

	tweenData := Component.Get(entry)

	// バウンスバック: 後ろに下がってから元の位置に戻る
	tweenData.SequenceX = gween.NewSequence(
		// 後ろに下がる (0.1秒, イーズアウト)
		gween.New(float32(centerX), float32(centerX-bounceDistance), 0.1, ease.OutQuad),
		// 元の位置に戻る (0.2秒, イーズバック)
		gween.New(float32(centerX-bounceDistance), float32(centerX), 0.2, ease.OutBack),
	)

	tweenData.SequenceY = gween.NewSequence(
		gween.New(float32(centerY), float32(centerY), 0.1, ease.Linear),
		gween.New(float32(centerY), float32(centerY), 0.2, ease.Linear),
	)
}
