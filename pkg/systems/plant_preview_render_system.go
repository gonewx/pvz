package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// PlantPreviewRenderSystem 渲染植物预览的双图像（使用静态图像）
// 渲染两个独立的植物图像：
//  1. 鼠标光标处的不透明图像（Alpha=1.0）
//  2. 网格格子中心的半透明预览图像（Alpha=0.5，仅当鼠标在网格内时）
type PlantPreviewRenderSystem struct {
	entityManager      *ecs.EntityManager
	plantPreviewSystem *PlantPreviewSystem // 用于获取两个渲染位置

	// 红线限制（保龄球关卡）
	redLineEnabled bool // 是否启用红线限制
}

// NewPlantPreviewRenderSystem 创建植物预览渲染系统
func NewPlantPreviewRenderSystem(em *ecs.EntityManager, pps *PlantPreviewSystem) *PlantPreviewRenderSystem {
	return &PlantPreviewRenderSystem{
		entityManager:      em,
		plantPreviewSystem: pps,
		redLineEnabled:     false,
	}
}

// SetRedLineEnabled 设置是否启用红线限制
// 启用后，网格预览在红线右侧不显示
func (s *PlantPreviewRenderSystem) SetRedLineEnabled(enabled bool) {
	s.redLineEnabled = enabled
	log.Printf("[PlantPreviewRenderSystem] 红线限制已设置: %v", enabled)
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

	// 红线检查：如果启用红线限制且网格位置在红线右侧，不显示网格预览
	showGridPreview := isInGrid
	if s.redLineEnabled && isInGrid {
		// 计算红线 X 位置
		redLineX := config.GridWorldStartX + float64(config.BowlingRedLineColumn)*config.CellWidth
		if gridX >= redLineX {
			showGridPreview = false
		}
	}

	for _, entityID := range entities {
		// 获取组件
		preview, ok := ecs.GetComponent[*components.PlantPreviewComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		sprite, ok := ecs.GetComponent[*components.SpriteComponent](s.entityManager, entityID)
		if !ok || sprite.Image == nil {
			continue
		}

		// 1️⃣ 渲染鼠标光标处的不透明图像（Alpha=1.0）
		s.drawStaticPreview(screen, sprite.Image, mouseX, mouseY, 1.0, cameraX, preview.IsExplosive)

		// 2️⃣ 如果在网格内且不在红线右侧，渲染格子中心的半透明预览图像（Alpha=0.5）
		if showGridPreview {
			s.drawStaticPreview(screen, sprite.Image, gridX, gridY, 0.5, cameraX, preview.IsExplosive)
		}
	}
}

// drawStaticPreview 渲染单个静态预览图像
func (s *PlantPreviewRenderSystem) drawStaticPreview(
	screen *ebiten.Image,
	img *ebiten.Image,
	worldX, worldY, alpha, cameraX float64,
	isExplosive bool,
) {
	// 转换为屏幕坐标
	screenX := worldX - cameraX
	screenY := worldY

	// 居中对齐
	w, h := img.Size()
	drawX := screenX - float64(w)/2
	drawY := screenY - float64(h)/2

	// 应用透明度和颜色
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(drawX, drawY)

	// 爆炸坚果使用红色染色
	if isExplosive {
		// 红色染色：增加红色通道，降低绿蓝通道
		opts.ColorM.Scale(1.0, 0.6, 0.6, alpha)
	} else {
		opts.ColorM.Scale(1, 1, 1, alpha)
	}

	screen.DrawImage(img, opts)
}
