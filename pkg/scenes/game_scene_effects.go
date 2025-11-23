package scenes

import (
	"image/color"

	"github.com/decker502/pvz/pkg/config"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawLawnFlash 绘制草坪闪烁效果（Story 8.2 教学）
// 在玩家选择植物卡片后，已铺设草皮的行会有明暗变化的闪烁效果（由明变暗）
// 使用黑色半透明遮罩实现草皮颜色变暗
// 只在关卡指定的启用行（enabledLanes）上显示闪烁效果
func (s *GameScene) drawLawnFlash(screen *ebiten.Image) {
	alpha := s.lawnGridSystem.GetFlashAlpha()
	if alpha <= 0 {
		return // 没有闪烁效果，直接返回
	}

	// 获取启用的行列表
	enabledLanes := s.lawnGridSystem.EnabledLanes
	if len(enabledLanes) == 0 {
		return // 没有启用的行
	}

	// 为每个启用的行单独绘制闪烁效果
	for _, lane := range enabledLanes {
		// 计算该行的世界坐标范围
		// lane 是 1-based (1-5)，需要转换为 0-based (0-4)
		rowIndex := lane - 1

		// 行的Y坐标范围
		rowStartY := config.GridWorldStartY + float64(rowIndex)*config.CellHeight
		rowEndY := rowStartY + config.CellHeight

		// 行的X坐标范围（整个草坪宽度）
		rowStartX := config.GridWorldStartX
		rowEndX := config.GridWorldStartX + float64(config.GridColumns)*config.CellWidth

		// 转换为屏幕坐标
		screenStartX := rowStartX - s.cameraX
		screenStartY := rowStartY
		width := rowEndX - rowStartX
		height := rowEndY - rowStartY

		// 创建黑色半透明遮罩（让草皮变暗）
		flashImage := ebiten.NewImage(int(width), int(height))
		flashImage.Fill(color.RGBA{0, 0, 0, uint8(alpha * 255)}) // 黑色，alpha 0.0-0.3

		// 绘制到屏幕
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenStartX, screenStartY)
		screen.DrawImage(flashImage, op)
	}
}
