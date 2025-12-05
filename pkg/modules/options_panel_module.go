package modules

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// OptionsPanelModule 选项面板模块（主菜单版）
//
// 职责：
//   - 复用游戏场景的暂停菜单样式（墓碑背景 + 遮罩）
//   - 显示游戏设置选项（音乐、音效、3D 加速、全屏）
//   - 只显示"确定"按钮（通过 SettingsPanelModule 的底部按钮配置）
//   - 完全组合 SettingsPanelModule，消除代码重复
//
// 与 PauseMenuModule 的区别：
//   - PauseMenuModule: 游戏中的暂停菜单（自己管理 3 个按钮）
//   - OptionsPanelModule: 主菜单的选项面板（使用 SettingsPanelModule 的底部按钮）
//
// 使用场景：
//   - 主菜单场景：点击选项按钮时显示
//
// 设计原则：
//   - 组合优于继承：使用 SettingsPanelModule 组合模式
//   - 模块化：可在不同场景复用
//   - 低耦合：通过回调与外部交互
//
// Story 12.3: 对话框系统基础 - 选项面板实现
// Story 20.5: 添加 SettingsManager 支持
// 重构：使用组合模式消除代码重复，底部按钮完全由 SettingsPanelModule 管理
type OptionsPanelModule struct {
	// Story 20.5: 设置管理器（用于保存设置）
	settingsManager *game.SettingsManager

	// 组合：通用设置面板（复用）
	settingsPanelModule *SettingsPanelModule
}

// NewOptionsPanelModule 创建选项面板模块
//
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于加载资源）
//   - buttonSystem: 按钮交互系统（引用，不拥有）
//   - buttonRenderSystem: 按钮渲染系统（引用，不拥有）
//   - settingsManager: 设置管理器（可为 nil）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - onClose: 关闭面板回调函数（可选）
//
// 返回:
//   - *OptionsPanelModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// Story 12.3: 对话框系统基础
// Story 20.5: 添加 settingsManager 参数
// 重构：完全组合 SettingsPanelModule，底部按钮由其管理
func NewOptionsPanelModule(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	buttonSystem *systems.ButtonSystem,
	buttonRenderSystem *systems.ButtonRenderSystem,
	settingsManager *game.SettingsManager,
	windowWidth, windowHeight int,
	onClose func(),
) (*OptionsPanelModule, error) {
	module := &OptionsPanelModule{
		settingsManager: settingsManager,
	}

	// 创建通用设置面板模块（组合），配置底部"确定"按钮
	settingsPanelModule, err := NewSettingsPanelModule(
		em,
		rm,
		buttonRenderSystem, // 传递按钮渲染系统（用于渲染底部按钮）
		settingsManager,    // Story 20.5: 传递 SettingsManager
		windowWidth,
		windowHeight,
		SettingsPanelCallbacks{
			OnMusicVolumeChange: func(volume float64) {
				// 日志/通知回调
			},
			OnSoundVolumeChange: func(volume float64) {
				// 日志/通知回调
			},
			On3DToggle: func(enabled bool) {
				// TODO: 实际控制3D加速
			},
			OnFullscreenToggle: func(enabled bool) {
				// 全屏切换由 SettingsPanelModule 内部处理
			},
			// 音量应用回调：连接到 AudioManager
			OnMusicVolumeApply: func(volume float64) {
				if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
					audioManager.SetMusicVolume(volume)
					log.Printf("[OptionsPanelModule] Applied music volume: %.2f", volume)
				}
			},
			OnSoundVolumeApply: func(volume float64) {
				if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
					audioManager.SetSoundVolume(volume)
					log.Printf("[OptionsPanelModule] Applied sound volume: %.2f", volume)
				}
			},
		},
		&BottomButtonConfig{
			Text: "确定",
			OnClick: func() {
				log.Printf("[OptionsPanelModule] Confirm button clicked!")
				// Story 20.5: 关闭时自动保存设置
				if module.settingsManager != nil {
					if err := module.settingsManager.Save(); err != nil {
						log.Printf("[OptionsPanelModule] Warning: Failed to save settings: %v", err)
					} else {
						log.Printf("[OptionsPanelModule] Settings saved successfully")
					}
				}
				module.Hide()
				if onClose != nil {
					onClose()
				}
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create settings panel module: %w", err)
	}
	module.settingsPanelModule = settingsPanelModule

	log.Printf("[OptionsPanelModule] Initialized successfully")

	return module, nil
}

// Update 更新选项面板状态
//
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 职责：
//   - 委托给 SettingsPanelModule
func (m *OptionsPanelModule) Update(deltaTime float64) {
	m.settingsPanelModule.Update(deltaTime)
}

// Draw 渲染选项面板到屏幕
//
// 参数:
//   - screen: 目标渲染屏幕
//
// 职责：
//   - 委托给 SettingsPanelModule（包括底部按钮）
func (m *OptionsPanelModule) Draw(screen *ebiten.Image) {
	m.settingsPanelModule.Draw(screen)
}

// Show 显示选项面板
//
// 效果：
//   - 委托给 SettingsPanelModule
func (m *OptionsPanelModule) Show() {
	log.Printf("[OptionsPanelModule] Show() called")
	m.settingsPanelModule.Show()
	log.Printf("[OptionsPanelModule] Options panel shown")
}

// Hide 隐藏选项面板
//
// 效果：
//   - 委托给 SettingsPanelModule
func (m *OptionsPanelModule) Hide() {
	m.settingsPanelModule.Hide()
	log.Printf("[OptionsPanelModule] Options panel hidden")
}

// IsActive 检查选项面板是否激活
//
// 返回:
//   - bool: 如果选项面板当前激活，返回 true
func (m *OptionsPanelModule) IsActive() bool {
	return m.settingsPanelModule.IsActive()
}

// Cleanup 清理模块资源
//
// 用途：
//   - 场景切换时清理所有选项面板实体
//   - 委托给 SettingsPanelModule
func (m *OptionsPanelModule) Cleanup() {
	if m.settingsPanelModule != nil {
		m.settingsPanelModule.Cleanup()
	}
	log.Printf("[OptionsPanelModule] Cleaned up")
}
