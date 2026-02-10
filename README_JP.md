# chirashi

[English version](README.md)

`chirashi` は Ebitengine 向けのGPU志向パーティクルコンポーネントとエディターです。
ECSライブラリとして [donburi](https://github.com/yohamta/donburi) を利用しています。

## 特徴

- `DrawTrianglesShader` を使ったGPUバッチ描画
- リアルタイムで調整できるビルトインエディター
- `polar` / `cartesian` の位置モード
- イージング + マルチステップシーケンス対応
- YAMLでの保存/読み込み
- donburi (ECS) 連携

## 要件

- Go 1.24+
- Ebitengine が対応する実行環境（デスクトップまたはWebビルド先）

## クイックスタート

エディターを直接起動:

```bash
go run ./cmd/chirashi-editor
```

または Mage タスクを利用:

```bash
mage run
```

利用可能なタスク一覧:

```bash
mage -l
```

主なビルド/実行コマンド:

```bash
mage build         # ネイティブバイナリを build/ に出力
mage buildRelease  # 最適化ビルド
mage buildWeb      # WASM を build/web に出力
mage serve         # Web用ビルド + localhost:8080 で配信
mage test          # go test ./... を実行
```

## コンポーネントとして使う

```go
import (
    "chirashi/component/chirashi"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/yohamta/donburi"
    "github.com/yohamta/donburi/ecs"
)

world := donburi.NewWorld()
gameECS := ecs.NewECS(world)

particleSystem := chirashi.NewSystem()
gameECS.AddSystem(particleSystem.Update)
gameECS.AddRenderer(0, particleSystem.Draw)

shader, _ := ebiten.NewShader([]byte("..."))
image := ebiten.NewImage(8, 8)

manager := chirashi.NewParticleManager(shader, image)
_ = manager.Preload("sample", "assets/particles/sample.yaml")
_, _ = manager.SpawnLoop(world, "sample", 640, 480)
```

## 設定ファイル（YAML）

パーティクルは YAML で定義します（`assets/particles/*.yaml` を参照）:

```yaml
name: "sample"
description: "sample particle"

emitter:
  x: 0
  y: 0

animation:
  duration:
    value: 1.0
  position:
    type: "polar"
    angle: { min: 0.0, max: 6.28 }
    distance: { min: 50, max: 150 }
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
    start: 0.0
    end: 3.14
    easing: "Linear"

spawn:
  interval: 1
  particles_per_spawn: 10
  max_particles: 1000
  is_loop: true
```

## デモ

Web上で動作確認ができます。
[https://muzigen.net/ebiten/chirashi/](https://muzigen.net/ebiten/chirashi/)

## ライセンス

MIT License
