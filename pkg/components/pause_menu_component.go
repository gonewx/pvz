package components

import "github.com/gonewx/pvz/pkg/ecs"

// PauseMenuComponent - 暂停菜单状态组件
// Story 10.1: 管理暂停菜单的激活状态和按钮实体ID
type PauseMenuComponent struct {
	IsActive       bool         // 暂停菜单是否激活
	ContinueButton ecs.EntityID // "继续"按钮实体ID
	RestartButton  ecs.EntityID // "重新开始"按钮实体ID
	MainMenuButton ecs.EntityID // "返回主菜单"按钮实体ID
	OverlayAlpha   uint8        // 遮罩透明度 (0-255)
}
