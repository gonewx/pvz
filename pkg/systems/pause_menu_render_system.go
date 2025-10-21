package systems

import (
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// PauseMenuRenderSystem 负责渲染暂停菜单
// Story 10.1: 绘制半透明遮罩和暂停菜单面板
type PauseMenuRenderSystem struct {
	entityManager *ecs.EntityManager
	windowWidth   int
	windowHeight  int
}

// NewPauseMenuRenderSystem 创建暂停菜单渲染系统
func NewPauseMenuRenderSystem(em *ecs.EntityManager, windowWidth, windowHeight int) *PauseMenuRenderSystem {
	return &PauseMenuRenderSystem{
		entityManager: em,
		windowWidth:   windowWidth,
		windowHeight:  windowHeight,
	}
}

// Draw 渲染暂停菜单
// 渲染顺序：
// 1. 半透明黑色遮罩（覆盖整个游戏画面）
// 2. 暂停菜单面板背景（深色矩形）
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

		// 2. 绘制暂停菜单面板背景
		panelX := (float64(s.windowWidth) - config.PauseMenuPanelWidth) / 2.0
		panelY := (float64(s.windowHeight) - config.PauseMenuPanelHeight) / 2.0

		panelImage := ebiten.NewImage(int(config.PauseMenuPanelWidth), int(config.PauseMenuPanelHeight))
		panelImage.Fill(color.RGBA{R: 40, G: 40, B: 40, A: 230}) // 深灰色半透明背景

		panelOp := &ebiten.DrawImageOptions{}
		panelOp.GeoM.Translate(panelX, panelY)
		screen.DrawImage(panelImage, panelOp)

		// 注意：按钮由 ButtonRenderSystem 自动渲染，无需在这里绘制
	}
}
