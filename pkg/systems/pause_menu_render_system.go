package systems

import (
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// PauseMenuRenderSystem 负责渲染暂停菜单
// 绘制半透明遮罩和暂停菜单面板（使用原版墓碑背景，双图层叠加）
type PauseMenuRenderSystem struct {
	entityManager     *ecs.EntityManager
	windowWidth       int
	windowHeight      int
	menuBackImage     *ebiten.Image // 暂停菜单背景图（墓碑形状，彩色层）
	menuBackMaskImage *ebiten.Image // 暂停菜单遮罩图（墓碑形状，用于实现边缘透明）
}

// NewPauseMenuRenderSystem 创建暂停菜单渲染系统
// 参数:
//   - menuBackImage: 背景彩色层图片
//   - menuBackMaskImage: 遮罩层图片（用于边缘透明效果，如果为nil则不使用遮罩）
func NewPauseMenuRenderSystem(em *ecs.EntityManager, windowWidth, windowHeight int, menuBackImage, menuBackMaskImage *ebiten.Image) *PauseMenuRenderSystem {
	return &PauseMenuRenderSystem{
		entityManager:     em,
		windowWidth:       windowWidth,
		windowHeight:      windowHeight,
		menuBackImage:     menuBackImage,
		menuBackMaskImage: menuBackMaskImage,
	}
}

// Draw 渲染暂停菜单
// 渲染顺序：
// 1. 半透明黑色遮罩（覆盖整个游戏画面）
// 2. 暂停菜单面板背景（原版墓碑形状背景图）
// 3. 按钮由 ButtonRenderSystem 自动渲染
func (s *PauseMenuRenderSystem) Draw(screen *ebiten.Image) {
	// 查询暂停菜单实体
	entities := ecs.GetEntitiesWith1[*components.PauseMenuComponent](s.entityManager)

	for _, entityID := range entities {
		pauseMenu, _ := ecs.GetComponent[*components.PauseMenuComponent](s.entityManager, entityID)

		// 只渲染激活的暂停菜单
		if !pauseMenu.IsActive {
			continue
		}

		// 1. 绘制半透明黑色遮罩（覆盖整个屏幕）
		overlayImage := ebiten.NewImage(s.windowWidth, s.windowHeight)
		overlayImage.Fill(color.RGBA{R: 0, G: 0, B: 0, A: pauseMenu.OverlayAlpha})
		screen.DrawImage(overlayImage, &ebiten.DrawImageOptions{})

		// 2. 绘制暂停菜单面板背景（双图层叠加实现边缘透明，类似草皮渲染）
		// 原版实现：彩色层 + 白色遮罩层，使用遮罩的亮度值作为Alpha通道
		if s.menuBackImage != nil {
			// 计算背景图片的中心位置
			bounds := s.menuBackImage.Bounds()
			menuWidth := float64(bounds.Dx())
			menuHeight := float64(bounds.Dy())

			panelX := (float64(s.windowWidth) - menuWidth) / 2.0
			panelY := (float64(s.windowHeight) - menuHeight) / 2.0

			// 如果有遮罩图片，使用CPU处理合成（参考草皮渲染）
			if s.menuBackMaskImage != nil {
				// 创建临时图片用于像素级合成
				tempImage := ebiten.NewImage(int(menuWidth), int(menuHeight))

				// 读取彩色背景像素
				colorPixels := make([]byte, 4*int(menuWidth)*int(menuHeight))
				s.menuBackImage.ReadPixels(colorPixels)

				// 读取遮罩像素
				maskPixels := make([]byte, 4*int(menuWidth)*int(menuHeight))
				s.menuBackMaskImage.ReadPixels(maskPixels)

				// 合成：使用遮罩的亮度值作为Alpha通道
				resultPixels := make([]byte, 4*int(menuWidth)*int(menuHeight))
				for i := 0; i < len(colorPixels); i += 4 {
					// 使用遮罩的R通道（灰度图的亮度）作为Alpha
					alpha := maskPixels[i] // R通道

					// 复制RGB from 彩色层，Alpha from 遮罩层
					resultPixels[i] = colorPixels[i]     // R
					resultPixels[i+1] = colorPixels[i+1] // G
					resultPixels[i+2] = colorPixels[i+2] // B
					resultPixels[i+3] = alpha            // A（来自遮罩）
				}

				// 写入合成后的像素
				tempImage.WritePixels(resultPixels)

				// 绘制合成后的图片到屏幕
				finalOp := &ebiten.DrawImageOptions{}
				finalOp.GeoM.Translate(panelX, panelY)
				screen.DrawImage(tempImage, finalOp)
			} else {
				// 没有遮罩图，直接绘制彩色背景
				panelOp := &ebiten.DrawImageOptions{}
				panelOp.GeoM.Translate(panelX, panelY)
				screen.DrawImage(s.menuBackImage, panelOp)
			}
		}

		// 注意：按钮由 ButtonRenderSystem 自动渲染，无需在这里绘制
	}
}
