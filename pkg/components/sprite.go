package components

import "github.com/hajimehoshi/ebiten/v2"

// SpriteComponent 存储实体的视觉表现(当前绘制的图像)
type SpriteComponent struct {
	Image *ebiten.Image
}
