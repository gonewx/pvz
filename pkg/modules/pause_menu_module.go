package modules

import (
	"fmt"
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

	// UI 元素实体
	musicSliderEntity ecs.EntityID // 音乐滑动条
	soundSliderEntity ecs.EntityID // 音效滑动条
	enable3DEntity    ecs.EntityID // 3D加速复选框
	fullscreenEntity  ecs.EntityID // 全屏复选框

	// UI 文字字体
	labelFont *text.GoTextFace // 标签文字字体

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

	// 1. 加载暂停菜单背景图（原版墓碑背景，双图层）
	menuBackImage := rm.GetImageByID("IMAGE_OPTIONS_MENUBACK")
	if menuBackImage == nil {
		log.Printf("[PauseMenuModule] Warning: Failed to load pause menu background image")
	}

	// 加载遮罩图片（用于边缘透明效果，类似草皮渲染）
	menuBackMaskImage := rm.GetImageByID("IMAGE_OPTIONS_MENUBACK_MASK")
	if menuBackMaskImage == nil {
		log.Printf("[PauseMenuModule] Warning: Failed to load pause menu mask image, edge transparency may not work")
	}

	// 2. 创建暂停菜单实体
	module.pauseMenuEntity = em.CreateEntity()

	// 3. 加载标签文字字体（用于滑动条和复选框）
	labelFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuLabelFontSize)
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to load label font: %v", err)
		labelFont = nil
	}
	module.labelFont = labelFont

	// 4. 初始化暂停菜单渲染系统（传入背景图和遮罩图）
	module.pauseMenuRenderSystem = systems.NewPauseMenuRenderSystem(em, windowWidth, windowHeight, menuBackImage, menuBackMaskImage)

	// 5. 创建暂停菜单按钮
	if err := module.createPauseMenuButtons(rm); err != nil {
		return nil, fmt.Errorf("failed to create pause menu buttons: %w", err)
	}

	// 6. 创建UI元素（滑动条和复选框）
	if err := module.createUIElements(rm); err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to create UI elements: %v", err)
	}

	// 7. 添加暂停菜单组件
	pauseMenuComp := module.getPauseMenuComponent()
	ecs.AddComponent(em, module.pauseMenuEntity, pauseMenuComp)

	log.Printf("[PauseMenuModule] Initialized with %d buttons", len(module.buttonEntities))

	return module, nil
}

// createPauseMenuButtons 创建暂停菜单按钮
// 内部方法，由 NewPauseMenuModule 调用
// Story 10.1: 使用原版资源 - "返回游戏"按钮使用 options_backtogamebutton 图片
func (m *PauseMenuModule) createPauseMenuButtons(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 加载"返回游戏"按钮图片（原版资源）
	backToGameNormal := rm.GetImageByID("IMAGE_OPTIONS_BACKTOGAMEBUTTON0")
	backToGameHover := rm.GetImageByID("IMAGE_OPTIONS_BACKTOGAMEBUTTON2")

	if backToGameNormal == nil {
		return fmt.Errorf("failed to load back to game button images")
	}

	log.Printf("[PauseMenuModule] Loaded back to game button images: Normal=%v, Hover=%v",
		backToGameNormal != nil, backToGameHover != nil)

	// 加载"返回游戏"按钮文字字体（中文，使用配置的字号）
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuBackToGameButtonFontSize)
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to load button font: %v", err)
		buttonFont = nil
	}

	// 创建"返回游戏"按钮实体
	backToGameEntity := m.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(m.entityManager, backToGameEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	// 添加按钮组件（简单图片按钮）
	ecs.AddComponent(m.entityManager, backToGameEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  backToGameNormal,
		HoverImage:   backToGameHover,
		PressedImage: backToGameHover,
		Text:         "返回游戏",                   // 渲染中文文字
		Font:         buttonFont,               // 中文字体
		TextColor:    [4]uint8{0, 200, 0, 255}, // 绿色文字（与其他按钮一致）
		State:        components.UINormal,
		Enabled:      true,
		OnClick: func() {
			log.Printf("[PauseMenuModule] Back to game button clicked!")
			m.Hide() // 隐藏暂停菜单（会自动触发 onResumeMusic）
			if m.onContinue != nil {
				m.onContinue()
			}
		},
	})

	m.buttonEntities = append(m.buttonEntities, backToGameEntity)

	// 4. 创建"重新开始"按钮（使用三段式按钮）
	restartEntity, err := m.createThreeSliceButton(rm, hiddenX, hiddenY, "重新开始", func() {
		log.Printf("[PauseMenuModule] Restart button clicked!")
		m.Hide()
		if m.onRestart != nil {
			m.onRestart()
		}
	})
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to create restart button: %v", err)
	} else {
		m.buttonEntities = append(m.buttonEntities, restartEntity)
	}

	// 5. 创建"主菜单"按钮（使用三段式按钮）
	mainMenuEntity, err := m.createThreeSliceButton(rm, hiddenX, hiddenY, "主菜单", func() {
		log.Printf("[PauseMenuModule] Main menu button clicked!")
		m.Hide()
		if m.onMainMenu != nil {
			m.onMainMenu()
		}
	})
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to create main menu button: %v", err)
	} else {
		m.buttonEntities = append(m.buttonEntities, mainMenuEntity)
	}

	return nil
}

// createThreeSliceButton 创建三段式按钮（复用 entities.NewMenuButton）
func (m *PauseMenuModule) createThreeSliceButton(rm *game.ResourceManager, x, y float64, text string, onClick func()) (ecs.EntityID, error) {
	// 复用 entities.NewMenuButton 工厂函数
	return entities.NewMenuButton(
		m.entityManager,
		rm,
		x,
		y,
		text,
		config.PauseMenuInnerButtonFontSize, // 使用内部按钮的字体大小
		[4]uint8{0, 200, 0, 255},            // 绿色文字
		config.PauseMenuInnerButtonWidth-40, // 减去左右边框宽度
		onClick,
	)
}

// createUIElements 创建UI元素（滑动条和复选框）
// Story 10.1: 完整实现暂停菜单UI
func (m *PauseMenuModule) createUIElements(rm *game.ResourceManager) error {
	// 初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 1. 创建音乐滑动条
	m.musicSliderEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.musicSliderEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.musicSliderEntity, &components.SliderComponent{
		SlotImage: rm.GetImageByID("IMAGE_OPTIONS_SLIDERSLOT"),
		KnobImage: rm.GetImageByID("IMAGE_OPTIONS_SLIDERKNOB2"),
		Value:     0.8, // 默认80%音量
		Label:     "音乐",
		OnValueChange: func(value float64) {
			log.Printf("[PauseMenuModule] Music volume changed: %.2f", value)
			// TODO: 实际控制音乐音量
		},
	})

	// 2. 创建音效滑动条
	m.soundSliderEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.soundSliderEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.soundSliderEntity, &components.SliderComponent{
		SlotImage: rm.GetImageByID("IMAGE_OPTIONS_SLIDERSLOT"),
		KnobImage: rm.GetImageByID("IMAGE_OPTIONS_SLIDERKNOB2"),
		Value:     0.8, // 默认80%音量
		Label:     "音效",
		OnValueChange: func(value float64) {
			log.Printf("[PauseMenuModule] Sound volume changed: %.2f", value)
			// TODO: 实际控制音效音量
		},
	})

	// 3. 创建3D加速复选框
	m.enable3DEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.enable3DEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.enable3DEntity, &components.CheckboxComponent{
		UncheckedImage: rm.GetImageByID("IMAGE_OPTIONS_CHECKBOX0"),
		CheckedImage:   rm.GetImageByID("IMAGE_OPTIONS_CHECKBOX1"),
		IsChecked:      true, // 默认开启
		Label:          "3D 加速",
		OnToggle: func(isChecked bool) {
			log.Printf("[PauseMenuModule] 3D acceleration toggled: %v", isChecked)
			// TODO: 实际控制3D加速
		},
	})

	// 4. 创建全屏复选框
	m.fullscreenEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.fullscreenEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.fullscreenEntity, &components.CheckboxComponent{
		UncheckedImage: rm.GetImageByID("IMAGE_OPTIONS_CHECKBOX0"),
		CheckedImage:   rm.GetImageByID("IMAGE_OPTIONS_CHECKBOX1"),
		IsChecked:      false, // 默认窗口模式
		Label:          "全屏",
		OnToggle: func(isChecked bool) {
			log.Printf("[PauseMenuModule] Fullscreen toggled: %v", isChecked)
			// TODO: 实际切换全屏
		},
	})

	return nil
}

// getPauseMenuComponent 构建暂停菜单组件
// 内部方法，用于初始化组件
func (m *PauseMenuModule) getPauseMenuComponent() *components.PauseMenuComponent {
	comp := &components.PauseMenuComponent{
		IsActive:     false, // 初始状态：未激活
		OverlayAlpha: config.PauseMenuOverlayAlpha,
	}

	// 设置按钮ID（如果存在）
	if len(m.buttonEntities) > 0 {
		comp.ContinueButton = m.buttonEntities[0] // "返回游戏"按钮
	}

	return comp
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
// Story 10.1: 完整布局 - "返回游戏"在下方，"重新开始"和"主菜单"在上方
func (m *PauseMenuModule) showButtons() {
	if len(m.buttonEntities) == 0 {
		return
	}

	// 屏幕中心位置
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0

	// 按钮索引：
	// 0: "返回游戏"按钮（在墓碑下方）
	// 1: "重新开始"按钮（在墓碑内部，上方）
	// 2: "主菜单"按钮（在墓碑内部，下方）

	// 1. "返回游戏"按钮（在墓碑底部外侧）
	if len(m.buttonEntities) > 0 {
		backToGameEntity := m.buttonEntities[0]
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, backToGameEntity); ok {
			buttonWidth := float64(button.NormalImage.Bounds().Dx())
			buttonHeight := float64(button.NormalImage.Bounds().Dy())
			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuBackToGameButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, backToGameEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
				log.Printf("[PauseMenuModule] Back to game button positioned at (%.1f, %.1f), size: %.1fx%.1f",
					buttonX, buttonY, buttonWidth, buttonHeight)
			}
		}
	}

	// 2. "重新开始"按钮（在墓碑内部，顶部）
	if len(m.buttonEntities) > 1 {
		restartEntity := m.buttonEntities[1]
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, restartEntity); ok {
			// 计算实际按钮宽度：左边缘 + 中间 + 右边缘
			var buttonWidth float64
			if button.Type == components.ButtonTypeNineSlice {
				leftWidth := float64(button.LeftImage.Bounds().Dx())
				rightWidth := float64(button.RightImage.Bounds().Dx())
				buttonWidth = leftWidth + button.MiddleWidth + rightWidth
			} else {
				buttonWidth = config.PauseMenuInnerButtonWidth
			}

			// 水平居中
			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuRestartButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, restartEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
			}
		}
	}

	// 3. "主菜单"按钮（在墓碑内部，中间偏下）
	if len(m.buttonEntities) > 2 {
		mainMenuEntity := m.buttonEntities[2]
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, mainMenuEntity); ok {
			// 计算实际按钮宽度：左边缘 + 中间 + 右边缘
			var buttonWidth float64
			if button.Type == components.ButtonTypeNineSlice {
				leftWidth := float64(button.LeftImage.Bounds().Dx())
				rightWidth := float64(button.RightImage.Bounds().Dx())
				buttonWidth = leftWidth + button.MiddleWidth + rightWidth
			} else {
				buttonWidth = config.PauseMenuInnerButtonWidth
			}

			// 水平居中
			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuMainMenuButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, mainMenuEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
			}
		}
	}

	// 4. 显示UI元素（滑动条和复选框）
	m.showUIElements()
}

// showUIElements 显示UI元素（滑动条和复选框）
func (m *PauseMenuModule) showUIElements() {
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0

	// UI元素在墓碑内部的垂直排列
	// 音乐滑动条：第1行
	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.musicSliderEntity); ok {
		pos.X = screenCenterX + config.PauseMenuMusicSliderOffsetX
		pos.Y = screenCenterY + config.PauseMenuMusicSliderOffsetY
	}

	// 音效滑动条：第2行
	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.soundSliderEntity); ok {
		pos.X = screenCenterX + config.PauseMenuSoundSliderOffsetX
		pos.Y = screenCenterY + config.PauseMenuSoundSliderOffsetY
	}

	// 3D加速复选框：第3行
	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.enable3DEntity); ok {
		pos.X = screenCenterX + config.PauseMenu3DCheckboxOffsetX
		pos.Y = screenCenterY + config.PauseMenu3DCheckboxOffsetY
	}

	// 全屏复选框：第4行
	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.fullscreenEntity); ok {
		pos.X = screenCenterX + config.PauseMenuFullscreenCheckboxOffsetX
		pos.Y = screenCenterY + config.PauseMenuFullscreenCheckboxOffsetY
	}
}

// hideUIElements 隐藏UI元素（移动到屏幕外）
func (m *PauseMenuModule) hideUIElements() {
	hiddenX := -1000.0
	hiddenY := -1000.0

	uiElements := []ecs.EntityID{
		m.musicSliderEntity,
		m.soundSliderEntity,
		m.enable3DEntity,
		m.fullscreenEntity,
	}

	for _, entityID := range uiElements {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, entityID); ok {
			pos.X = hiddenX
			pos.Y = hiddenY
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

	// 同时隐藏UI元素
	m.hideUIElements()
}

// Draw 渲染暂停菜单到屏幕
// 参数:
//   - screen: 目标渲染屏幕
//
// 渲染顺序：
//  1. 半透明遮罩
//  2. 墓碑背景面板
//  3. UI元素（滑动条、复选框）
//  4. 菜单按钮（在背景上方）
func (m *PauseMenuModule) Draw(screen *ebiten.Image) {
	// 1. 渲染遮罩和墓碑背景
	m.pauseMenuRenderSystem.Draw(screen)

	// 2. 渲染UI元素
	m.drawUIElements(screen)

	// 3. 渲染暂停菜单的按钮（在背景上方）
	if m.buttonRenderSystem != nil {
		// 只渲染暂停菜单的按钮
		for _, buttonEntity := range m.buttonEntities {
			m.buttonRenderSystem.DrawButton(screen, buttonEntity)
		}
	}
}

// drawUIElements 渲染UI元素（滑动条和复选框）
func (m *PauseMenuModule) drawUIElements(screen *ebiten.Image) {
	// 渲染音乐滑动条
	if slider, ok := ecs.GetComponent[*components.SliderComponent](m.entityManager, m.musicSliderEntity); ok {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.musicSliderEntity); ok {
			m.drawSlider(screen, slider, pos.X, pos.Y)
		}
	}

	// 渲染音效滑动条
	if slider, ok := ecs.GetComponent[*components.SliderComponent](m.entityManager, m.soundSliderEntity); ok {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.soundSliderEntity); ok {
			m.drawSlider(screen, slider, pos.X, pos.Y)
		}
	}

	// 渲染3D加速复选框
	if checkbox, ok := ecs.GetComponent[*components.CheckboxComponent](m.entityManager, m.enable3DEntity); ok {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.enable3DEntity); ok {
			m.drawCheckbox(screen, checkbox, pos.X, pos.Y)
		}
	}

	// 渲染全屏复选框
	if checkbox, ok := ecs.GetComponent[*components.CheckboxComponent](m.entityManager, m.fullscreenEntity); ok {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.fullscreenEntity); ok {
			m.drawCheckbox(screen, checkbox, pos.X, pos.Y)
		}
	}
}

// drawSlider 渲染单个滑动条
func (m *PauseMenuModule) drawSlider(screen *ebiten.Image, slider *components.SliderComponent, x, y float64) {
	if slider.SlotImage == nil || slider.KnobImage == nil {
		return
	}

	// 渲染文字标签（在滑槽左侧，带阴影效果）
	if m.labelFont != nil && slider.Label != "" {
		labelX := x + config.PauseMenuLabelOffsetX
		labelY := y + config.PauseMenuLabelOffsetY

		// 阴影偏移量
		shadowOffsetX := 2.0
		shadowOffsetY := 2.0
		visualCenterOffsetY := -shadowOffsetY / 2.0

		// 1. 先绘制阴影
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(labelX+shadowOffsetX, labelY+shadowOffsetY+visualCenterOffsetY)
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 180})
		text.Draw(screen, slider.Label, m.labelFont, shadowOp)

		// 2. 再绘制主文字（浅蓝紫色，参考截图）
		op := &text.DrawOptions{}
		op.GeoM.Translate(labelX, labelY+visualCenterOffsetY)
		op.ColorScale.ScaleWithColor(color.RGBA{180, 180, 255, 255}) // 浅蓝紫色
		text.Draw(screen, slider.Label, m.labelFont, op)
	}

	// 渲染滑槽
	slotOp := &ebiten.DrawImageOptions{}
	slotOp.GeoM.Translate(x, y)
	screen.DrawImage(slider.SlotImage, slotOp)

	// 渲染滑块（根据value位置）
	slotWidth := float64(slider.SlotImage.Bounds().Dx())
	slotHeight := float64(slider.SlotImage.Bounds().Dy())
	knobHeight := float64(slider.KnobImage.Bounds().Dy())

	// 水平位置：根据value值
	knobX := x + slotWidth*slider.Value - float64(slider.KnobImage.Bounds().Dx())/2.0
	// 垂直位置：相对于滑槽居中
	knobY := y + (slotHeight-knobHeight)/2.0

	knobOp := &ebiten.DrawImageOptions{}
	knobOp.GeoM.Translate(knobX, knobY)
	screen.DrawImage(slider.KnobImage, knobOp)
}

// drawCheckbox 渲染单个复选框
func (m *PauseMenuModule) drawCheckbox(screen *ebiten.Image, checkbox *components.CheckboxComponent, x, y float64) {
	var image *ebiten.Image
	if checkbox.IsChecked {
		image = checkbox.CheckedImage
	} else {
		image = checkbox.UncheckedImage
	}

	if image == nil {
		return
	}

	// 渲染文字标签（在复选框左侧，带阴影效果）
	if m.labelFont != nil && checkbox.Label != "" {
		labelX := x + config.PauseMenuLabelOffsetX
		labelY := y + config.PauseMenuLabelOffsetY

		// 阴影偏移量
		shadowOffsetX := 2.0
		shadowOffsetY := 2.0
		visualCenterOffsetY := -shadowOffsetY / 2.0

		// 1. 先绘制阴影
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(labelX+shadowOffsetX, labelY+shadowOffsetY+visualCenterOffsetY)
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 180})
		text.Draw(screen, checkbox.Label, m.labelFont, shadowOp)

		// 2. 再绘制主文字（浅蓝紫色，参考截图）
		op := &text.DrawOptions{}
		op.GeoM.Translate(labelX, labelY+visualCenterOffsetY)
		op.ColorScale.ScaleWithColor(color.RGBA{180, 180, 255, 255}) // 浅蓝紫色
		text.Draw(screen, checkbox.Label, m.labelFont, op)
	}

	// 渲染复选框
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(image, op)
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

// GetButtonEntities 返回所有按钮实体ID（调试用）
func (m *PauseMenuModule) GetButtonEntities() []ecs.EntityID {
	return m.buttonEntities
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
