# aburi - GPU Particle System

GPU最適化されたパーティクルシステム。`DrawTrianglesShader`による1回のドローコールでバッチ描画を実現。

## 特徴

- **GPUバッチ描画**: 全パーティクルを1回のドローコールで描画
- **極座標/直交座標**: 円形パターンも簡単に作成
- **色アニメーション**: RGB補間による色変化
- **イージング**: 25種類のイージング関数をGPU/CPUで適用
- **ワンショット対応**: ヒットエフェクトなど一時的なパーティクル

## クイックスタート

### 1. セットアップ

```go
import (ええええええ
    "chirashi/assets"
    "chirashi/component/aburi"
)

// シェーダーとイメージを準備
shader, _ := ebiten.NewShader(assets.ParticleShader)
img := ebiten.NewImage(8, 8)
img.Fill(color.White)

// パーティクルマネージャー作成
pm := aburi.NewParticleManager(shader, img)

// エフェクトを事前ロード
pm.Preload("hit", "assets/particles/aburi/hit_spark.yaml")
pm.Preload("explosion", "assets/particles/aburi/rocket_thrust.yaml")
pm.Preload("flame", "assets/particles/aburi/burner_flame.yaml")
```

### 2. ECSシステム登録

```go
world := donburi.NewWorld()
ecs := ecs.NewECS(world)

particleSys := aburi.NewSystem()
ecs.AddSystem(particleSys.Update)
ecs.AddRenderer(0, particleSys.Draw)
```

### 3. パーティクル生成

```go
// ワンショット（60フレームで自動削除）
pm.SpawnOneShot(world, "hit", x, y, 60)

// ループ（手動削除）
entity, _ := pm.SpawnLoop(world, "flame", x, y)
// 削除時: world.Remove(entity)
```

## YAML設定

### 基本構造

```yaml
name: "effect_name"
description: "説明"

emitter:
  x: 0      # エミッター位置オフセット
  y: 0

animation:
  duration:
    value: 1.0
    range:           # オプション: ランダム範囲
      min: 0.8
      max: 1.2

  position:
    type: "polar"    # "polar" or "cartesian"
    # Polar mode
    angle:
      min: 0
      max: 6.28      # 0-2π で全方向
    distance:
      min: 50
      max: 150
    easing: "OutCirc"

  alpha:
    start: 1.0
    end: 0.0
    easing: "Linear"

  scale:
    start: 1.0
    end: 0.5
    easing: "OutQuad"

  rotation:
    start: 0
    end: 3.14
    easing: "Linear"

  color:             # オプション
    start_r: 1.0
    start_g: 1.0
    start_b: 1.0
    end_r: 1.0
    end_g: 0.2
    end_b: 0.0
    easing: "Linear"

spawn:
  interval: 1           # フレーム間隔
  particles_per_spawn: 10
  max_particles: 1000
  is_loop: true
  life_time: 60         # is_loop=false時のみ有効
```

### 極座標 vs 直交座標

```yaml
# 極座標（円形パターン向け）
position:
  type: "polar"
  angle: { min: 0, max: 6.28 }
  distance: { min: 50, max: 100 }

# 直交座標（矩形パターン向け）
position:
  type: "cartesian"
  end_x: { min: -100, max: 100 }
  end_y: { min: -100, max: 100 }
```

### 角度の参考値

| 方向 | ラジアン | 度 |
|------|----------|-----|
| 右 | 0 | 0° |
| 下 | 1.57 | 90° |
| 左 | 3.14 | 180° |
| 上 | 4.71 | 270° |
| 全方向 | 0〜6.28 | 0〜360° |

## イージング一覧

```
Linear
InQuad, OutQuad, InOutQuad
InCubic, OutCubic, InOutCubic
InQuart, OutQuart, InOutQuart
InQuint, OutQuint, InOutQuint
InSine, OutSine, InOutSine
InExpo, OutExpo, InOutExpo
InCirc, OutCirc, InOutCirc
InBack, OutBack, InOutBack
```

## エフェクト例

### 爆発（全方向）

```yaml
position:
  type: "polar"
  angle: { min: 0, max: 6.28 }
  distance: { min: 50, max: 200 }
  easing: "OutQuad"
scale:
  start: 1.0
  end: 0.2
color:
  start_r: 1.0, start_g: 1.0, start_b: 0.8  # 白黄
  end_r: 1.0, end_g: 0.2, end_b: 0.0        # 赤
```

### ロケット噴射（下向き）

```yaml
position:
  type: "polar"
  angle: { min: 1.2, max: 1.94 }  # 下向き±20°
  distance: { min: 80, max: 200 }
color:
  start_r: 1.0, start_g: 0.95, start_b: 0.7
  end_r: 1.0, end_g: 0.3, end_b: 0.0
```

### バーナー（先細り）

```yaml
position:
  type: "polar"
  angle: { min: 1.47, max: 1.67 }  # 狭い角度
  distance: { min: 100, max: 180 }
scale:
  start: 1.2
  end: 0.1       # 先細り
color:
  start_r: 0.7, start_g: 0.85, start_b: 1.0  # 青白
  end_r: 1.0, end_g: 0.5, end_b: 0.1         # オレンジ
```

### ヒットスパーク

```yaml
duration:
  value: 0.3
position:
  type: "polar"
  angle: { min: 0, max: 6.28 }
  distance: { min: 20, max: 80 }
  easing: "OutQuad"
spawn:
  particles_per_spawn: 20
  is_loop: false
  life_time: 30
```

## API リファレンス

### ParticleManager

```go
// 作成
pm := aburi.NewParticleManager(shader, image)

// 事前ロード
pm.Preload(name, path string) error
pm.PreloadFromBytes(name string, data []byte) error

// 生成
pm.SpawnOneShot(world, name string, x, y float32, lifetimeFrames int) error
pm.SpawnLoop(world, name string, x, y float32) (donburi.Entity, error)

// 設定変更
pm.SetShader(shader *ebiten.Shader)
pm.SetImage(image *ebiten.Image)
```

### 直接生成（低レベルAPI）

```go
aburi.NewParticlesFromConfig(world, shader, image, config, x, y)
aburi.NewParticlesFromFile(world, shader, image, path, x, y)
```

## エディタ

```bash
go run . -aburi
```

- パラメータをリアルタイム編集
- YAMLの保存/読み込み
- デバッグ情報表示（FPS、アクティブ数、描画時間）

## パフォーマンス

| 項目 | 実装 |
|------|------|
| 描画 | 1回のDrawTrianglesShader |
| 位置/スケール/回転 | CPU（spawn時 or 毎フレーム） |
| アルファ/色 | GPU（シェーダー内で補間） |
| 極座標変換 | spawn時のみ（毎フレームコストなし） |
| メモリ | プール + インデックス管理（O(1)操作） |
