package modules

import (
	"fmt"
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SettingsPanelModule 通用设置面板模块
//
// 职责：
//   - 管理游戏设置 UI 元素（音乐、音效、3D 加速、全屏）
//   - 提供统一的渲染和交互逻辑
//   - 可被其他模块组合使用（组合优于继承）
//
// 设计原则：
//   - 高内聚：封装所有设置 UI 相关功能
//   - 低耦合：通过回调与外部交互
//   - 可组合：被 PauseMenuModule、OptionsPanelModule 等复用
//
// 使用场景：
//   - PauseMenuModule: 游戏中的暂停菜单设置
//   - OptionsPanelModule: 主菜单的选项面板设置
//   - 其他需要设置 UI 的场景
//
// Epic: 架构重构 - 消除代码重复
// Story 20.5: 添加 SettingsManager 集成
type SettingsPanelModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// Story 20.5: 设置管理器（用于读取/保存设置）
	settingsManager *game.SettingsManager

	// 系统（用于渲染墓碑背景和遮罩）
	pauseMenuRenderSystem *systems.PauseMenuRenderSystem // 墓碑背景渲染
	buttonRenderSystem    *systems.ButtonRenderSystem    // 按钮渲染（用于渲染底部按钮）

	// UI 元素实体
	musicSliderEntity ecs.EntityID // 音乐滑动条
	soundSliderEntity ecs.EntityID // 音效滑动条
	enable3DEntity    ecs.EntityID // 3D加速复选框
	fullscreenEntity  ecs.EntityID // 全屏复选框

	// 底部按钮（可选）
	bottomButtonEntity ecs.EntityID // 底部按钮实体（如 "返回游戏"、"确定"）
	hasBottomButton    bool         // 是否有底部按钮

	// UI 文字字体
	labelFont *text.GoTextFace // 标签文字字体

	// 设置面板实体（用于存储 PauseMenuComponent）
	settingsPanelEntity ecs.EntityID

	// 回调函数（可选）
	onMusicVolumeChange func(volume float64) // 音乐音量变化回调（日志/通知）
	onSoundVolumeChange func(volume float64) // 音效音量变化回调（日志/通知）
	on3DToggle          func(enabled bool)   // 3D加速切换回调
	onFullscreenToggle  func(enabled bool)   // 全屏切换回调

	// Story 20.5: 音量实际应用回调
	onMusicVolumeApply func(volume float64) // 音乐音量实际应用回调
	onSoundVolumeApply func(volume float64) // 音效音量实际应用回调

	// 屏幕尺寸
	windowWidth  int
	windowHeight int
}

// SettingsPanelCallbacks 设置面板回调函数集合
type SettingsPanelCallbacks struct {
	OnMusicVolumeChange func(volume float64) // 音乐音量变化回调（可选，用于日志/通知）
	OnSoundVolumeChange func(volume float64) // 音效音量变化回调（可选，用于日志/通知）
	On3DToggle          func(enabled bool)   // 3D加速切换回调（可选）
	OnFullscreenToggle  func(enabled bool)   // 全屏切换回调（可选）

	// Story 20.5: 音量实际应用回调（用于将音量应用到音频系统）
	OnMusicVolumeApply func(volume float64) // 音乐音量实际应用回调（可选）
	OnSoundVolumeApply func(volume float64) // 音效音量实际应用回调（可选）
}

// BottomButtonConfig 底部按钮配置
type BottomButtonConfig struct {
	Text    string // 按钮文字（如 "返回游戏"、"确定"）
	OnClick func() // 点击回调
}

// NewSettingsPanelModule 创建通用设置面板模块
//
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于加载资源）
//   - buttonRenderSystem: 按钮渲染系统（用于渲染底部按钮）
//   - settingsManager: 设置管理器（可选，用于读取/保存设置）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - callbacks: 设置面板回调函数集合（可选）
//   - bottomButtonConfig: 底部按钮配置（可选，传 nil 表示不创建底部按钮）
//
// 返回:
//   - *SettingsPanelModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// Story 20.5: 添加 settingsManager 参数用于读取当前设置值
// 架构重构：提取自 PauseMenuModule 和 OptionsPanelModule 的通用逻辑
func NewSettingsPanelModule(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	buttonRenderSystem *systems.ButtonRenderSystem,
	settingsManager *game.SettingsManager,
	windowWidth, windowHeight int,
	callbacks SettingsPanelCallbacks,
	bottomButtonConfig *BottomButtonConfig,
) (*SettingsPanelModule, error) {
	module := &SettingsPanelModule{
		entityManager:       em,
		buttonRenderSystem:  buttonRenderSystem,
		settingsManager:     settingsManager,
		windowWidth:         windowWidth,
		windowHeight:        windowHeight,
		onMusicVolumeChange: callbacks.OnMusicVolumeChange,
		onSoundVolumeChange: callbacks.OnSoundVolumeChange,
		on3DToggle:          callbacks.On3DToggle,
		onFullscreenToggle:  callbacks.OnFullscreenToggle,
		onMusicVolumeApply:  callbacks.OnMusicVolumeApply,
		onSoundVolumeApply:  callbacks.OnSoundVolumeApply,
	}

	// 1. 加载暂停菜单背景图（原版墓碑背景，双图层）
	menuBackImage, err := rm.LoadImage("assets/images/options_menuback.jpg")
	if err != nil {
		log.Printf("[SettingsPanelModule] Warning: Failed to load menu background image: %v", err)
		menuBackImage = nil
	}

	// 加载遮罩图片（用于边缘透明效果）
	menuBackMaskImage, err := rm.LoadImage("assets/images/options_menuback_.png")
	if err != nil {
		log.Printf("[SettingsPanelModule] Warning: Failed to load menu mask image: %v", err)
		menuBackMaskImage = nil
	}

	// 2. 初始化墓碑背景渲染系统
	module.pauseMenuRenderSystem = systems.NewPauseMenuRenderSystem(em, windowWidth, windowHeight, menuBackImage, menuBackMaskImage)

	// 3. 加载标签文字字体（用于滑动条和复选框）
	labelFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuLabelFontSize)
	if err != nil {
		log.Printf("[SettingsPanelModule] Warning: Failed to load label font: %v", err)
		labelFont = nil
	}
	module.labelFont = labelFont

	// 4. 创建设置面板实体
	module.settingsPanelEntity = em.CreateEntity()

	// 添加 PauseMenuComponent（复用现有组件）
	ecs.AddComponent(em, module.settingsPanelEntity, &components.PauseMenuComponent{
		IsActive:     false, // 初始状态：未激活
		OverlayAlpha: config.PauseMenuOverlayAlpha,
	})

	// 5. 创建UI元素（滑动条和复选框）
	if err := module.createUIElements(rm); err != nil {
		log.Printf("[SettingsPanelModule] Warning: Failed to create UI elements: %v", err)
	}

	// 6. 创建底部按钮（如果配置了）
	if bottomButtonConfig != nil {
		if err := module.createBottomButton(rm, bottomButtonConfig); err != nil {
			log.Printf("[SettingsPanelModule] Warning: Failed to create bottom button: %v", err)
		} else {
			module.hasBottomButton = true
		}
	}

	log.Printf("[SettingsPanelModule] Initialized successfully")

	return module, nil
}

// createUIElements 创建UI元素（滑动条和复选框）
//
// Story 20.5: 从 SettingsManager 加载当前设置值，如果为 nil 则使用默认值
func (m *SettingsPanelModule) createUIElements(rm *game.ResourceManager) error {
	// 初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// Story 20.5: 从 SettingsManager 读取当前设置值
	// 如果 settingsManager 为 nil，使用默认值（降级处理）
	musicVolume := 0.8
	soundVolume := 0.8
	fullscreen := false

	if m.settingsManager != nil {
		settings := m.settingsManager.GetSettings()
		musicVolume = settings.MusicVolume
		soundVolume = settings.SoundVolume
		fullscreen = settings.Fullscreen
		log.Printf("[SettingsPanelModule] Loaded settings from SettingsManager: music=%.2f, sound=%.2f, fullscreen=%v",
			musicVolume, soundVolume, fullscreen)
	} else {
		log.Printf("[SettingsPanelModule] SettingsManager is nil, using default values: music=%.2f, sound=%.2f, fullscreen=%v",
			musicVolume, soundVolume, fullscreen)
	}

	// 1. 创建音乐滑动条
	m.musicSliderEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.musicSliderEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	// 加载滑动条图片
	slotImage, _ := rm.LoadImage("assets/images/options_sliderslot.png")
	knobImage, _ := rm.LoadImage("assets/images/options_sliderknob2.png")

	ecs.AddComponent(m.entityManager, m.musicSliderEntity, &components.SliderComponent{
		SlotImage:    slotImage,
		KnobImage:    knobImage,
		Value:        musicVolume, // Story 20.5: 从设置加载
		Label:        "音乐",
		ClickSoundID: "SOUND_BUTTONCLICK", // 滑块点击音效
		OnValueChange: func(value float64) {
			log.Printf("[SettingsPanelModule] Music volume changed: %.2f", value)
			// Story 20.5: 更新 SettingsManager
			if m.settingsManager != nil {
				m.settingsManager.SetMusicVolume(value)
			}
			// 触发变化回调（日志/通知）
			if m.onMusicVolumeChange != nil {
				m.onMusicVolumeChange(value)
			}
			// Story 20.5: 触发实际应用回调
			if m.onMusicVolumeApply != nil {
				m.onMusicVolumeApply(value)
			}
		},
	})

	// 2. 创建音效滑动条
	m.soundSliderEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.soundSliderEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.soundSliderEntity, &components.SliderComponent{
		SlotImage:    slotImage,
		KnobImage:    knobImage,
		Value:        soundVolume, // Story 20.5: 从设置加载
		Label:        "音效",
		ClickSoundID: "SOUND_BUTTONCLICK", // 滑块点击音效
		OnValueChange: func(value float64) {
			log.Printf("[SettingsPanelModule] Sound volume changed: %.2f", value)
			// Story 20.5: 更新 SettingsManager
			if m.settingsManager != nil {
				m.settingsManager.SetSoundVolume(value)
			}
			// 触发变化回调（日志/通知）
			if m.onSoundVolumeChange != nil {
				m.onSoundVolumeChange(value)
			}
			// Story 20.5: 触发实际应用回调
			if m.onSoundVolumeApply != nil {
				m.onSoundVolumeApply(value)
			}
		},
	})

	// 3. 加载复选框图片
	checkboxUnchecked, _ := rm.LoadImage("assets/images/options_checkbox0.png")
	checkboxChecked, _ := rm.LoadImage("assets/images/options_checkbox1.png")

	// 4. 创建3D加速复选框
	m.enable3DEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.enable3DEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.enable3DEntity, &components.CheckboxComponent{
		UncheckedImage: checkboxUnchecked,
		CheckedImage:   checkboxChecked,
		IsChecked:      true, // 默认开启
		Label:          "3D 加速",
		ClickSoundID:   "SOUND_BUTTONCLICK", // 复选框点击音效
		OnToggle: func(isChecked bool) {
			log.Printf("[SettingsPanelModule] 3D acceleration toggled: %v", isChecked)
			if m.on3DToggle != nil {
				m.on3DToggle(isChecked)
			}
		},
	})

	// 5. 创建全屏复选框
	m.fullscreenEntity = m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, m.fullscreenEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, m.fullscreenEntity, &components.CheckboxComponent{
		UncheckedImage: checkboxUnchecked,
		CheckedImage:   checkboxChecked,
		IsChecked:      fullscreen, // Story 20.5: 从设置加载
		Label:          "全屏",
		ClickSoundID:   "SOUND_BUTTONCLICK", // 复选框点击音效
		OnToggle: func(isChecked bool) {
			log.Printf("[SettingsPanelModule] Fullscreen toggled: %v", isChecked)
			// Story 20.5: 更新 SettingsManager 并立即切换全屏
			if m.settingsManager != nil {
				m.settingsManager.SetFullscreen(isChecked)
			}
			// Story 20.5: 立即切换全屏状态
			if isChecked {
				ebiten.SetFullscreen(true)
			} else {
				// 退出全屏时，恢复窗口尺寸为游戏逻辑尺寸
				ebiten.SetFullscreen(false)
				if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
					ebiten.RestoreWindow()
				}
				ebiten.SetWindowSize(config.GameWindowWidth, config.GameWindowHeight)
			}
			// 触发回调（如果有）
			if m.onFullscreenToggle != nil {
				m.onFullscreenToggle(isChecked)
			}
		},
	})

	return nil
}

// createBottomButton 创建底部按钮（使用与暂停菜单"返回游戏"按钮一致的样式）
func (m *SettingsPanelModule) createBottomButton(rm *game.ResourceManager, buttonConfig *BottomButtonConfig) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 加载按钮图片（与暂停菜单"返回游戏"按钮一致）
	backToGameNormal, err := rm.LoadImage("assets/images/options_backtogamebutton0.png")
	if err != nil {
		return fmt.Errorf("failed to load options_backtogamebutton0.png: %w", err)
	}

	backToGamePressed, err := rm.LoadImage("assets/images/options_backtogamebutton2.png")
	if err != nil {
		log.Printf("[SettingsPanelModule] Warning: Failed to load pressed button image: %v", err)
		backToGamePressed = backToGameNormal // 使用 Normal 图片作为后备
	}

	// 加载按钮文字字体
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuBackToGameButtonFontSize)
	if err != nil {
		log.Printf("[SettingsPanelModule] Warning: Failed to load button font: %v", err)
		buttonFont = nil
	}

	// 创建底部按钮实体
	m.bottomButtonEntity = m.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(m.entityManager, m.bottomButtonEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	// 计算按钮尺寸（从图片获取）
	buttonWidth := float64(backToGameNormal.Bounds().Dx())
	buttonHeight := float64(backToGameNormal.Bounds().Dy())

	// 添加按钮组件
	ecs.AddComponent(m.entityManager, m.bottomButtonEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  backToGameNormal,
		HoverImage:   backToGameNormal,         // ✅ 悬停时不换图（backtogamebutton 系列没有悬停状态）
		PressedImage: backToGamePressed,        // ✅ 按下时使用 button2（下陷效果）
		Text:         buttonConfig.Text,        // 使用配置的文字
		Font:         buttonFont,               // 中文字体
		TextColor:    [4]uint8{0, 200, 0, 255}, // 绿色文字
		Width:        buttonWidth,              // ✅ 初始化按钮尺寸
		Height:       buttonHeight,             // ✅ 初始化按钮尺寸
		State:        components.UINormal,
		Enabled:      true,
		OnClick:        buttonConfig.OnClick,   // 使用配置的回调
		ClickSoundID:   "SOUND_BUTTONCLICK",    // 释放时播放的音效
		PressedSoundID: "SOUND_GRAVEBUTTON",    // 按下时播放的音效（墓碑样式）
	})

	log.Printf("[SettingsPanelModule] Bottom button created with text: %s", buttonConfig.Text)

	return nil
}

// Update 更新设置面板状态
//
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 职责：
//   - 同步设置面板激活状态
//   - 控制 UI 元素显示/隐藏
func (m *SettingsPanelModule) Update(deltaTime float64) {
	// 获取设置面板组件
	settingsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.settingsPanelEntity)
	if !ok {
		return
	}

	// 根据激活状态更新 UI 元素位置
	if settingsPanel.IsActive {
		m.showUIElements()
	} else {
		m.hideUIElements()
	}
}

// showUIElements 显示 UI 元素（滑动条、复选框）
func (m *SettingsPanelModule) showUIElements() {
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

	// 底部按钮（如果有）
	if m.hasBottomButton {
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.bottomButtonEntity); ok {
			buttonWidth := float64(button.NormalImage.Bounds().Dx())
			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuBackToGameButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.bottomButtonEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
			}
		}
	}
}

// hideUIElements 隐藏 UI 元素（移动到屏幕外）
func (m *SettingsPanelModule) hideUIElements() {
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

	// 隐藏底部按钮（如果有）
	if m.hasBottomButton {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.bottomButtonEntity); ok {
			pos.X = hiddenX
			pos.Y = hiddenY
		}
	}
}

// Draw 渲染设置面板到屏幕
//
// 参数:
//   - screen: 目标渲染屏幕
//
// 渲染顺序：
//  1. 半透明遮罩
//  2. 墓碑背景面板
//  3. UI元素（滑动条、复选框）
//  4. 底部按钮（如果有）
func (m *SettingsPanelModule) Draw(screen *ebiten.Image) {
	// 获取设置面板组件
	settingsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.settingsPanelEntity)
	if !ok || !settingsPanel.IsActive {
		return
	}

	// 1. 渲染遮罩和墓碑背景
	m.pauseMenuRenderSystem.Draw(screen)

	// 2. 渲染UI元素
	m.drawUIElements(screen)

	// 3. 渲染底部按钮（如果有）
	if m.hasBottomButton && m.buttonRenderSystem != nil {
		m.buttonRenderSystem.DrawButton(screen, m.bottomButtonEntity)
	}
}

// drawUIElements 渲染UI元素（滑动条和复选框）
func (m *SettingsPanelModule) drawUIElements(screen *ebiten.Image) {
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
func (m *SettingsPanelModule) drawSlider(screen *ebiten.Image, slider *components.SliderComponent, x, y float64) {
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
func (m *SettingsPanelModule) drawCheckbox(screen *ebiten.Image, checkbox *components.CheckboxComponent, x, y float64) {
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

// Show 显示设置面板
//
// 效果：
//   - 设置 PauseMenuComponent.IsActive = true
//   - UI 元素移动到正确位置
func (m *SettingsPanelModule) Show() {
	if settingsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.settingsPanelEntity); ok {
		settingsPanel.IsActive = true
		m.showUIElements()
		log.Printf("[SettingsPanelModule] Settings panel shown")
	}
}

// Hide 隐藏设置面板
//
// 效果：
//   - 设置 PauseMenuComponent.IsActive = false
//   - UI 元素移动到屏幕外
func (m *SettingsPanelModule) Hide() {
	if settingsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.settingsPanelEntity); ok {
		settingsPanel.IsActive = false
		m.hideUIElements()
		log.Printf("[SettingsPanelModule] Settings panel hidden")
	}
}

// IsActive 检查设置面板是否激活
//
// 返回:
//   - bool: 如果设置面板当前激活，返回 true
func (m *SettingsPanelModule) IsActive() bool {
	if settingsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.settingsPanelEntity); ok {
		return settingsPanel.IsActive
	}
	return false
}

// Cleanup 清理模块资源
//
// 用途：
//   - 场景切换时清理所有设置面板实体
//   - 避免内存泄漏
func (m *SettingsPanelModule) Cleanup() {
	// 清理设置面板实体
	m.entityManager.DestroyEntity(m.settingsPanelEntity)

	// 清理 UI 元素实体
	m.entityManager.DestroyEntity(m.musicSliderEntity)
	m.entityManager.DestroyEntity(m.soundSliderEntity)
	m.entityManager.DestroyEntity(m.enable3DEntity)
	m.entityManager.DestroyEntity(m.fullscreenEntity)

	// 清理底部按钮（如果有）
	if m.hasBottomButton {
		m.entityManager.DestroyEntity(m.bottomButtonEntity)
	}

	log.Printf("[SettingsPanelModule] Cleaned up")
}
