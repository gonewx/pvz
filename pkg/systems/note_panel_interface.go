package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// NotePanelModule 定义来信面板模块的接口（Story 8.14）
//
// 该接口用于打破 systems 和 modules 包之间的循环依赖。
// 具体实现由 modules.ZombieNotePanelModule 提供。
//
// 设计原则：
//   - 依赖倒置：systems 包依赖接口而非具体实现
//   - 低耦合：modules 包可以独立演化
type NotePanelModule interface {
	// Update 更新面板状态
	Update(deltaTime float64)

	// Draw 渲染面板
	Draw(screen *ebiten.Image)

	// Show 显示面板
	Show()

	// Hide 隐藏面板
	Hide()

	// SetFadeAlpha 设置淡入透明度（0.0 - 1.0）
	SetFadeAlpha(alpha float64)

	// IsActive 返回面板是否激活
	IsActive() bool

	// Cleanup 清理面板资源
	Cleanup()

	// IsHoveringNextButton 返回鼠标是否悬停在"下一关"按钮上
	IsHoveringNextButton() bool

	// IsHoveringMainMenuButton 返回鼠标是否悬停在"主菜单"按钮上
	IsHoveringMainMenuButton() bool
}

// NotePanelFactory 定义创建来信面板的工厂函数类型
//
// 参数:
//   - noteID: 来信 ID（如 "zombienote1"）
//   - onNextLevel: 点击"下一关"按钮回调
//   - onMainMenu: 点击"主菜单"按钮回调
//
// 返回:
//   - NotePanelModule: 来信面板模块实例
//   - error: 如果创建失败
type NotePanelFactory func(noteID string, onNextLevel func(), onMainMenu func()) (NotePanelModule, error)
