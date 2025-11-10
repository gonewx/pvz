package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// PlantPreviewRenderSystem 渲染植物预览的双图像（使用静态图像）
// 渲染两个独立的植物图像：
//  1. 鼠标光标处的不透明图像（Alpha=1.0）
//  2. 网格格子中心的半透明预览图像（Alpha=0.5，仅当鼠标在网格内时）
type PlantPreviewRenderSystem struct {
	entityManager      *ecs.EntityManager
	plantPreviewSystem *PlantPreviewSystem // 用于获取两个渲染位置
}

// NewPlantPreviewRenderSystem 创建植物预览渲染系统
func NewPlantPreviewRenderSystem(em *ecs.EntityManager, pps *PlantPreviewSystem) *PlantPreviewRenderSystem {
	return &PlantPreviewRenderSystem{
		entityManager:      em,
		plantPreviewSystem: pps,
	}
}

// Draw 渲染所有植物预览实体（双图像渲染，使用静态图像）
// 参数:
//   - screen: 目标渲染画布
//   - cameraX: 摄像机的世界坐标X位置（用于世界坐标到屏幕坐标的转换）
//
// 渲染逻辑：
//  1. 在鼠标光标位置渲染不透明图像（Alpha=1.0）
//  2. 在网格格子中心渲染半透明预览图像（Alpha=0.5，仅当鼠标在网格内时）
func (s *PlantPreviewRenderSystem) Draw(screen *ebiten.Image, cameraX float64) {
	// 查询所有拥有 PlantPreviewComponent, PositionComponent, SpriteComponent 的实体
	entities := ecs.GetEntitiesWith3[
		*components.PlantPreviewComponent,
		*components.PositionComponent,
		*components.SpriteComponent,
	](s.entityManager)

	if len(entities) == 0 {
		return
	}

	// 获取两个渲染位置
	mouseX, mouseY, gridX, gridY, isInGrid := s.plantPreviewSystem.GetPreviewPositions()

	for _, entityID := range entities {
		// 获取组件
		sprite, ok := ecs.GetComponent[*components.SpriteComponent](s.entityManager, entityID)
		if !ok || sprite.Image == nil {
			continue
		}

		// 1️⃣ 渲染鼠标光标处的不透明图像（Alpha=1.0）
		s.drawStaticPreview(screen, sprite.Image, mouseX, mouseY, 1.0, cameraX)

		// 2️⃣ 如果在网格内，渲染格子中心的半透明预览图像（Alpha=0.5）
		if isInGrid {
			s.drawStaticPreview(screen, sprite.Image, gridX, gridY, 0.5, cameraX)
		}
	}
}

// drawStaticPreview 渲染单个静态预览图像
func (s *PlantPreviewRenderSystem) drawStaticPreview(
	screen *ebiten.Image,
	img *ebiten.Image,
	worldX, worldY, alpha, cameraX float64,
) {
	// 转换为屏幕坐标
	screenX := worldX - cameraX
	screenY := worldY

	// 居中对齐
	w, h := img.Size()
	drawX := screenX - float64(w)/2
	drawY := screenY - float64(h)/2

	// 应用透明度
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(drawX, drawY)
	opts.ColorM.Scale(1, 1, 1, alpha)

	screen.DrawImage(img, opts)
}
