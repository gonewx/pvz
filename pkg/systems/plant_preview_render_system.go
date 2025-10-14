package systems

import (
	"log"
	"math"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// PlantPreviewRenderSystem 渲染植物预览的双图像（使用 Reanim）
// 渲染两个独立的植物图像：
//   1. 鼠标光标处的不透明图像（Alpha=1.0）
//   2. 网格格子中心的半透明预览图像（Alpha=0.5，仅当鼠标在网格内时）
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

// Draw 渲染所有植物预览实体（双图像渲染）
// 参数:
//   - screen: 目标渲染画布
//   - cameraX: 摄像机的世界坐标X位置（用于世界坐标到屏幕坐标的转换）
// 渲染逻辑：
//   1. 在鼠标光标位置渲染不透明图像（Alpha=1.0）
//   2. 在网格格子中心渲染半透明预览图像（Alpha=0.5，仅当鼠标在网格内时）
func (s *PlantPreviewRenderSystem) Draw(screen *ebiten.Image, cameraX float64) {
	// 查询所有拥有 PlantPreviewComponent, PositionComponent, ReanimComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantPreviewComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.ReanimComponent{}),
	)

	// 获取两个渲染位置
	mouseX, mouseY, gridX, gridY, isInGrid := s.plantPreviewSystem.GetPreviewPositions()

	for _, entityID := range entities {
		// 获取组件
		reanimComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.ReanimComponent{}))
		reanimData := reanimComp.(*components.ReanimComponent)

		// 1️⃣ 渲染鼠标光标处的不透明图像（Alpha=1.0）
		tempPosForCursor := &components.PositionComponent{X: mouseX, Y: mouseY}
		s.drawReanimPreview(screen, reanimData, tempPosForCursor, 1.0, cameraX, entityID)

		// 2️⃣ 如果在网格内，渲染格子中心的半透明预览图像（Alpha=0.5）
		if isInGrid {
			tempPosForGrid := &components.PositionComponent{X: gridX, Y: gridY}
			s.drawReanimPreview(screen, reanimData, tempPosForGrid, 0.5, cameraX, entityID)
		}
	}
}

// drawReanimPreview 渲染单个 Reanim 预览（半透明）
func (s *PlantPreviewRenderSystem) drawReanimPreview(screen *ebiten.Image, reanim *components.ReanimComponent, pos *components.PositionComponent, alpha float64, cameraX float64, entityID ecs.EntityID) {
	if reanim.Reanim == nil || reanim.CurrentAnim == "" {
		return
	}

	// 将逻辑帧映射到物理帧索引
	physicalIndex := s.findPhysicalFrameIndex(reanim, reanim.CurrentFrame)
	if physicalIndex < 0 {
		return
	}

	// 将世界坐标转换为屏幕坐标，并应用 Reanim 的中心偏移
	screenX := pos.X - cameraX - reanim.CenterOffsetX
	screenY := pos.Y - reanim.CenterOffsetY

	// 调试：输出预览位置信息（每60帧输出一次，避免刷屏）
	if reanim.CurrentFrame%60 == 0 {
		log.Printf("[PlantPreviewRender] 预览 %d: 世界坐标(%.1f, %.1f), 屏幕坐标(%.1f, %.1f), CenterOffset(%.1f, %.1f)",
			entityID, pos.X, pos.Y, screenX, screenY, reanim.CenterOffsetX, reanim.CenterOffsetY)
	}

	// 按 AnimTracks 顺序渲染部件（保证 Z-order 正确）
	for _, track := range reanim.AnimTracks {
		// 如果设置了 VisibleTracks，只渲染白名单中的轨道
		if reanim.VisibleTracks != nil && len(reanim.VisibleTracks) > 0 {
			if !reanim.VisibleTracks[track.Name] {
				continue
			}
		}

		mergedFrames, ok := reanim.MergedTracks[track.Name]
		if !ok || physicalIndex >= len(mergedFrames) {
			continue
		}

		mergedFrame := mergedFrames[physicalIndex]

		// 跳过隐藏的帧
		if mergedFrame.FrameNum != nil && *mergedFrame.FrameNum == -1 {
			continue
		}

		// 跳过没有图片的帧
		if mergedFrame.ImagePath == "" {
			continue
		}

		// 使用 IMAGE 引用查找图片
		img, exists := reanim.PartImages[mergedFrame.ImagePath]
		if !exists || img == nil {
			continue
		}

		// 获取图片尺寸
		bounds := img.Bounds()
		w := bounds.Dx()
		h := bounds.Dy()
		fw := float64(w)
		fh := float64(h)

		// 获取变换参数
		scaleX := 1.0
		scaleY := 1.0
		if mergedFrame.ScaleX != nil {
			scaleX = *mergedFrame.ScaleX
		}
		if mergedFrame.ScaleY != nil {
			scaleY = *mergedFrame.ScaleY
		}

		kx := 0.0
		ky := 0.0
		if mergedFrame.SkewX != nil {
			kx = *mergedFrame.SkewX
		}
		if mergedFrame.SkewY != nil {
			ky = *mergedFrame.SkewY
		}

		// 构建变换矩阵
		a := math.Cos(kx*math.Pi/180.0) * scaleX
		b := math.Sin(kx*math.Pi/180.0) * scaleX
		c := -math.Sin(ky*math.Pi/180.0) * scaleY
		d := math.Cos(ky*math.Pi/180.0) * scaleY

		// 平移分量
		tx := 0.0
		ty := 0.0
		if mergedFrame.X != nil {
			tx = *mergedFrame.X
		}
		if mergedFrame.Y != nil {
			ty = *mergedFrame.Y
		}
		tx += screenX
		ty += screenY

		// 应用变换矩阵到图片的四个角
		x0 := tx
		y0 := ty
		x1 := a*fw + tx
		y1 := b*fw + ty
		x2 := c*fh + tx
		y2 := d*fh + ty
		x3 := a*fw + c*fh + tx
		y3 := b*fw + d*fh + ty

		// 构建顶点数组（两个三角形组成矩形）
		// 重要：应用半透明 alpha 值
		alphaF := float32(alpha)
		vs := []ebiten.Vertex{
			{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: alphaF, ColorG: alphaF, ColorB: alphaF, ColorA: alphaF},
			{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: alphaF, ColorG: alphaF, ColorB: alphaF, ColorA: alphaF},
			{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: alphaF, ColorG: alphaF, ColorB: alphaF, ColorA: alphaF},
			{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: alphaF, ColorG: alphaF, ColorB: alphaF, ColorA: alphaF},
		}
		is := []uint16{0, 1, 2, 1, 3, 2}
		screen.DrawTriangles(vs, is, img, nil)
	}
}

// findPhysicalFrameIndex 查找当前逻辑帧对应的物理帧索引
func (s *PlantPreviewRenderSystem) findPhysicalFrameIndex(comp *components.ReanimComponent, logicalFrame int) int {
	if logicalFrame < 0 || logicalFrame >= len(comp.AnimVisibles) {
		return -1
	}

	visibleCount := 0
	for i, visible := range comp.AnimVisibles {
		if visible == 0 {
			if visibleCount == logicalFrame {
				return i
			}
			visibleCount++
		}
	}
	return -1
}
