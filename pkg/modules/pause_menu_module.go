package modules

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// PauseMenuModule 暂停菜单模块
// 封装所有与暂停菜单相关的功能，包括：
//   - 暂停菜单实体的创建和管理
//   - 暂停菜单按钮的交互逻辑
//   - 暂停菜单的渲染（遮罩、面板、按钮）
//   - 暂停状态的控制（暂停/恢复）
//
// 设计原则：
//   - 高内聚：所有暂停菜单功能封装在单一模块中
//   - 低耦合：通过清晰的接口与外部系统交互
//   - 可复用：支持在不同场景（游戏中、小游戏、挑战模式）使用
//
// 使用场景：
//   - GameScene: 游戏中的暂停菜单（当前）
//   - MiniGameScene: 小游戏中的暂停菜单（未来扩展）
//   - ChallengeScene: 挑战模式中的暂停菜单（未来扩展）
//
// Story 10.1: 暂停菜单系统
type PauseMenuModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// 系统（内部管理）
	pauseMenuRenderSystem *systems.PauseMenuRenderSystem // 暂停菜单渲染
	buttonSystem          *systems.ButtonSystem          // 按钮交互（不拥有，只引用）
	buttonRenderSystem    *systems.ButtonRenderSystem    // 按钮渲染（不拥有，只引用）

	// 暂停菜单实体
	pauseMenuEntity ecs.EntityID

	// 按钮实体列表（用于显示/隐藏控制）
	buttonEntities []ecs.EntityID

	// 外部依赖
	gameState *game.GameState

	// 回调函数（由外部场景提供）
	onContinue    func() // "继续"按钮回调
	onRestart     func() // "重新开始"按钮回调
	onMainMenu    func() // "返回主菜单"按钮回调
	onPauseMusic  func() // 暂停音乐回调
	onResumeMusic func() // 恢复音乐回调

	// 内部状态（用于检测状态变化）
	wasActive bool // 上一帧的激活状态

	// 屏幕尺寸（用于渲染）
	windowWidth  int
	windowHeight int
}

// PauseMenuCallbacks 暂停菜单回调函数集合
// 用于外部场景传递回调逻辑
type PauseMenuCallbacks struct {
	OnContinue    func() // "继续"按钮回调
	OnRestart     func() // "重新开始"按钮回调
	OnMainMenu    func() // "返回主菜单"按钮回调
	OnPauseMusic  func() // 暂停音乐回调（可选，如果BGM系统未实现，传nil）
	OnResumeMusic func() // 恢复音乐回调（可选）
}

// NewPauseMenuModule 创建一个新的暂停菜单模块
//
// 参数:
//   - em: EntityManager 实例
//   - gs: GameState 实例（用于控制暂停状态）
//   - rm: ResourceManager 实例（用于加载按钮资源）
//   - buttonSystem: 按钮交互系统（引用，不拥有）
//   - buttonRenderSystem: 按钮渲染系统（引用，不拥有）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - callbacks: 暂停菜单回调函数集合
//
// 返回:
//   - *PauseMenuModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// 注意：
//   - 此函数会自动创建 PauseMenuComponent 实体
//   - 创建三个菜单按钮（继续、重新开始、返回主菜单）
//   - 初始化暂停菜单渲染系统
//   - 按钮初始状态为隐藏（位置在屏幕外）
func NewPauseMenuModule(
	em *ecs.EntityManager,
	gs *game.GameState,
	rm *game.ResourceManager,
	buttonSystem *systems.ButtonSystem,
	buttonRenderSystem *systems.ButtonRenderSystem,
	windowWidth, windowHeight int,
	callbacks PauseMenuCallbacks,
) (*PauseMenuModule, error) {
	module := &PauseMenuModule{
		entityManager:      em,
		gameState:          gs,
		buttonSystem:       buttonSystem,
		buttonRenderSystem: buttonRenderSystem,
		onContinue:         callbacks.OnContinue,
		onRestart:          callbacks.OnRestart,
		onMainMenu:         callbacks.OnMainMenu,
		onPauseMusic:       callbacks.OnPauseMusic,
		onResumeMusic:      callbacks.OnResumeMusic,
		windowWidth:        windowWidth,
		windowHeight:       windowHeight,
		buttonEntities:     make([]ecs.EntityID, 0, 3),
	}

	// 1. 创建暂停菜单实体
	module.pauseMenuEntity = em.CreateEntity()

	// 2. 初始化暂停菜单渲染系统
	module.pauseMenuRenderSystem = systems.NewPauseMenuRenderSystem(em, windowWidth, windowHeight)

	// 3. 创建暂停菜单按钮
	if err := module.createPauseMenuButtons(rm); err != nil {
		return nil, fmt.Errorf("failed to create pause menu buttons: %w", err)
	}

	// 4. 添加暂停菜单组件
	pauseMenuComp := module.getPauseMenuComponent()
	ecs.AddComponent(em, module.pauseMenuEntity, pauseMenuComp)

	log.Printf("[PauseMenuModule] Initialized with %d buttons", len(module.buttonEntities))

	return module, nil
}

// createPauseMenuButtons 创建暂停菜单的三个按钮
// 内部方法，由 NewPauseMenuModule 调用
func (m *PauseMenuModule) createPauseMenuButtons(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 计算按钮中间宽度
	buttonMiddleWidth := config.PauseMenuButtonWidth - 40 // 扣除左右边框

	// 1. 创建"继续"按钮
	continueButton, err := entities.NewMenuButton(
		m.entityManager,
		rm,
		hiddenX, // 初始隐藏
		hiddenY,
		"继续",
		24.0,
		[4]uint8{255, 255, 255, 255}, // 白色文字
		buttonMiddleWidth,
		func() {
			log.Printf("[PauseMenuModule] Continue button clicked!")
			m.Hide() // 隐藏暂停菜单（会自动触发 onResumeMusic）
			if m.onContinue != nil {
				m.onContinue()
			}
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create continue button: %w", err)
	}
	m.buttonEntities = append(m.buttonEntities, continueButton)

	// 2. 创建"重新开始"按钮
	restartButton, err := entities.NewMenuButton(
		m.entityManager,
		rm,
		hiddenX, // 初始隐藏
		hiddenY,
		"重新开始",
		24.0,
		[4]uint8{255, 255, 255, 255}, // 白色文字
		buttonMiddleWidth,
		func() {
			log.Printf("[PauseMenuModule] Restart button clicked!")
			m.Hide() // 隐藏暂停菜单
			if m.onRestart != nil {
				m.onRestart()
			}
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create restart button: %w", err)
	}
	m.buttonEntities = append(m.buttonEntities, restartButton)

	// 3. 创建"返回主菜单"按钮
	mainMenuButton, err := entities.NewMenuButton(
		m.entityManager,
		rm,
		hiddenX, // 初始隐藏
		hiddenY,
		"返回主菜单",
		24.0,
		[4]uint8{255, 255, 255, 255}, // 白色文字
		buttonMiddleWidth,
		func() {
			log.Printf("[PauseMenuModule] Main menu button clicked!")
			m.Hide() // 隐藏暂停菜单
			if m.onMainMenu != nil {
				m.onMainMenu()
			}
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create main menu button: %w", err)
	}
	m.buttonEntities = append(m.buttonEntities, mainMenuButton)

	return nil
}

// getPauseMenuComponent 构建暂停菜单组件
// 内部方法，用于初始化组件
func (m *PauseMenuModule) getPauseMenuComponent() *components.PauseMenuComponent {
	return &components.PauseMenuComponent{
		IsActive:       false, // 初始状态：未激活
		ContinueButton: m.buttonEntities[0],
		RestartButton:  m.buttonEntities[1],
		MainMenuButton: m.buttonEntities[2],
		OverlayAlpha:   config.PauseMenuOverlayAlpha,
	}
}

// Update 更新暂停菜单状态
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 职责：
//   - 同步暂停菜单激活状态（从 GameState.IsPaused 读取）
//   - 控制按钮显示/隐藏
//   - 检测状态变化，触发音乐回调
//
// 注意：
//   - 按钮交互由 ButtonSystem 自动处理（外部调用）
//   - 当检测到 GameState.IsPaused 变化时（如 ESC 键触发），自动触发音乐回调
func (m *PauseMenuModule) Update(deltaTime float64) {
	// 从 GameState 同步暂停状态
	isPaused := m.gameState.IsPaused

	// 检测状态变化（处理 ESC 键触发的暂停/恢复）
	if isPaused != m.wasActive {
		if isPaused {
			// 刚刚暂停
			if m.onPauseMusic != nil {
				m.onPauseMusic()
			}
			log.Printf("[PauseMenuModule] Paused (triggered by external state change)")
		} else {
			// 刚刚恢复
			if m.onResumeMusic != nil {
				m.onResumeMusic()
			}
			log.Printf("[PauseMenuModule] Resumed (triggered by external state change)")
		}
		m.wasActive = isPaused
	}

	// 更新 PauseMenuComponent 的 IsActive 状态
	if pauseMenu, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.pauseMenuEntity); ok {
		if pauseMenu.IsActive != isPaused {
			pauseMenu.IsActive = isPaused
			m.updateButtonsVisibility(isPaused)
		}
	}
}

// updateButtonsVisibility 控制按钮显示/隐藏
// 内部方法，由 Update 调用
func (m *PauseMenuModule) updateButtonsVisibility(visible bool) {
	if visible {
		// 显示按钮：移动到正确位置
		m.showButtons()
	} else {
		// 隐藏按钮：移动到屏幕外
		m.hideButtons()
	}
}

// showButtons 显示所有暂停菜单按钮（移动到正确位置）
func (m *PauseMenuModule) showButtons() {
	// 计算菜单面板中心位置
	panelCenterX := float64(m.windowWidth) / 2.0
	panelCenterY := float64(m.windowHeight) / 2.0

	// 计算按钮位置（垂直居中排列）
	buttonStartY := panelCenterY - config.PauseMenuButtonHeight - config.PauseMenuButtonSpacing
	buttonX := panelCenterX - config.PauseMenuButtonWidth/2.0 // 居中

	// 按钮位置数组
	buttonPositions := []struct {
		x, y float64
	}{
		{buttonX, buttonStartY}, // 继续按钮
		{buttonX, buttonStartY + config.PauseMenuButtonHeight + config.PauseMenuButtonSpacing},   // 重新开始按钮
		{buttonX, buttonStartY + (config.PauseMenuButtonHeight+config.PauseMenuButtonSpacing)*2}, // 返回主菜单按钮
	}

	// 更新所有按钮位置
	for i, entityID := range m.buttonEntities {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, entityID); ok {
			pos.X = buttonPositions[i].x
			pos.Y = buttonPositions[i].y
		}
	}
}

// hideButtons 隐藏所有暂停菜单按钮（移动到屏幕外）
func (m *PauseMenuModule) hideButtons() {
	hiddenX := -1000.0
	hiddenY := -1000.0

	for _, entityID := range m.buttonEntities {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, entityID); ok {
			pos.X = hiddenX
			pos.Y = hiddenY
		}
	}
}

// Draw 渲染暂停菜单到屏幕
// 参数:
//   - screen: 目标渲染屏幕
//
// 注意：
//   - 只渲染暂停菜单遮罩和面板
//   - 按钮由 ButtonRenderSystem 自动渲染（外部调用）
func (m *PauseMenuModule) Draw(screen *ebiten.Image) {
	m.pauseMenuRenderSystem.Draw(screen)
}

// Show 显示暂停菜单
// 用途：
//   - 点击菜单按钮时调用
//
// 效果：
//   - 设置 GameState.IsPaused = true
//   - 更新 wasActive 状态（避免 Update() 重复触发回调）
//   - 暂停 BGM（如果回调函数已设置）
func (m *PauseMenuModule) Show() {
	m.gameState.SetPaused(true)
	m.wasActive = true // 同步内部状态，避免 Update() 重复触发
	if m.onPauseMusic != nil {
		m.onPauseMusic()
	}
	log.Printf("[PauseMenuModule] Pause menu shown")
}

// Hide 隐藏暂停菜单
// 用途：
//   - 点击"继续"按钮时调用
//
// 效果：
//   - 设置 GameState.IsPaused = false
//   - 更新 wasActive 状态（避免 Update() 重复触发回调）
//   - 恢复 BGM（如果回调函数已设置）
func (m *PauseMenuModule) Hide() {
	m.gameState.SetPaused(false)
	m.wasActive = false // 同步内部状态，避免 Update() 重复触发
	if m.onResumeMusic != nil {
		m.onResumeMusic()
	}
	log.Printf("[PauseMenuModule] Pause menu hidden")
}

// Toggle 切换暂停菜单显示/隐藏
// 用途：
//   - 按 ESC 键时调用
//
// 效果：
//   - 如果当前已暂停，则恢复游戏
//   - 如果当前未暂停，则暂停游戏
func (m *PauseMenuModule) Toggle() {
	if m.gameState.IsPaused {
		m.Hide()
	} else {
		m.Show()
	}
}

// IsActive 检查暂停菜单是否激活
// 返回:
//   - bool: 如果暂停菜单当前激活，返回 true
func (m *PauseMenuModule) IsActive() bool {
	return m.gameState.IsPaused
}

// Cleanup 清理模块资源
// 用途：
//   - 场景切换时清理所有暂停菜单实体
//   - 避免内存泄漏
//
// 注意：
//   - 清理暂停菜单实体和按钮实体
//   - 不清理系统实例（ButtonSystem、ButtonRenderSystem 由外部管理）
func (m *PauseMenuModule) Cleanup() {
	// 清理暂停菜单实体
	m.entityManager.DestroyEntity(m.pauseMenuEntity)

	// 清理所有按钮实体
	for _, entityID := range m.buttonEntities {
		m.entityManager.DestroyEntity(entityID)
	}

	log.Printf("[PauseMenuModule] Cleaned up")
}
