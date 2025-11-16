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

// OptionsPanel Module 选项面板模块（主菜单版）
//
// 职责：
//   - 复用游戏场景的暂停菜单样式（墓碑背景 + 遮罩）
//   - 显示游戏设置选项（音乐、音效、3D 加速、全屏）
//   - 不显示"继续"和"重新开始"按钮（主菜单无需这些功能）
//   - 只显示"确定"按钮（原"返回游戏"按钮的文本修改）
//
// 与 PauseMenuModule 的区别：
//   - PauseMenuModule: 游戏中的暂停菜单（3 个按钮：继续、重新开始、主菜单）
//   - OptionsPanelModule: 主菜单的选项面板（1 个按钮：确定）
//
// 使用场景：
//   - 主菜单场景：点击选项按钮时显示
//
// 设计原则：
//   - 模块化：可在不同场景复用
//   - 自包含：封装所有选项面板相关功能
//   - 低耦合：通过回调与外部交互
//
// Story 12.3: 对话框系统基础 - 选项面板实现
type OptionsPanelModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// 系统（内部管理）
	pauseMenuRenderSystem *systems.PauseMenuRenderSystem // 暂停菜单渲染
	buttonSystem          *systems.ButtonSystem          // 按钮交互（不拥有，只引用）
	buttonRenderSystem    *systems.ButtonRenderSystem    // 按钮渲染（不拥有，只引用）

	// 选项面板实体
	optionsPanelEntity ecs.EntityID

	// 按钮实体（只有1个："确定"）
	confirmButtonEntity ecs.EntityID

	// UI 元素实体
	musicSliderEntity ecs.EntityID // 音乐滑动条
	soundSliderEntity ecs.EntityID // 音效滑动条
	enable3DEntity    ecs.EntityID // 3D加速复选框
	fullscreenEntity  ecs.EntityID // 全屏复选框

	// UI 文字字体
	labelFont *text.GoTextFace // 标签文字字体

	// 回调函数
	onClose func() // 关闭面板回调

	// 屏幕尺寸
	windowWidth  int
	windowHeight int
}

// NewOptionsPanelModule 创建选项面板模块
//
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于加载资源）
//   - buttonSystem: 按钮交互系统（引用，不拥有）
//   - buttonRenderSystem: 按钮渲染系统（引用，不拥有）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - onClose: 关闭面板回调函数（可选）
//
// 返回:
//   - *OptionsPanelModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// Story 12.3: 对话框系统基础
func NewOptionsPanelModule(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	buttonSystem *systems.ButtonSystem,
	buttonRenderSystem *systems.ButtonRenderSystem,
	windowWidth, windowHeight int,
	onClose func(),
) (*OptionsPanelModule, error) {
	module := &OptionsPanelModule{
		entityManager:      em,
		buttonSystem:       buttonSystem,
		buttonRenderSystem: buttonRenderSystem,
		onClose:            onClose,
		windowWidth:        windowWidth,
		windowHeight:       windowHeight,
	}

	// 1. 加载暂停菜单背景图（原版墓碑背景，双图层）
	menuBackImage := rm.GetImageByID("IMAGE_OPTIONS_MENUBACK")
	if menuBackImage == nil {
		log.Printf("[OptionsPanelModule] Warning: Failed to load pause menu background image")
	}

	// 加载遮罩图片（用于边缘透明效果，类似草皮渲染）
	menuBackMaskImage := rm.GetImageByID("IMAGE_OPTIONS_MENUBACK_MASK")
	if menuBackMaskImage == nil {
		log.Printf("[OptionsPanelModule] Warning: Failed to load pause menu mask image, edge transparency may not work")
	}

	// 2. 创建选项面板实体
	module.optionsPanelEntity = em.CreateEntity()

	// 3. 加载标签文字字体（用于滑动条和复选框）
	labelFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuLabelFontSize)
	if err != nil {
		log.Printf("[OptionsPanelModule] Warning: Failed to load label font: %v", err)
		labelFont = nil
	}
	module.labelFont = labelFont

	// 4. 初始化暂停菜单渲染系统（复用游戏场景的渲染逻辑）
	module.pauseMenuRenderSystem = systems.NewPauseMenuRenderSystem(em, windowWidth, windowHeight, menuBackImage, menuBackMaskImage)

	// 5. 创建"确定"按钮（替代"返回游戏"按钮）
	if err := module.createConfirmButton(rm); err != nil {
		return nil, fmt.Errorf("failed to create confirm button: %w", err)
	}

	// 6. 创建UI元素（滑动条和复选框）
	if err := module.createUIElements(rm); err != nil {
		log.Printf("[OptionsPanelModule] Warning: Failed to create UI elements: %v", err)
	}

	// 7. 添加 PauseMenuComponent（复用组件）
	pauseMenuComp := &components.PauseMenuComponent{
		IsActive:     false, // 初始状态：未激活
		OverlayAlpha: config.PauseMenuOverlayAlpha,
	}
	ecs.AddComponent(em, module.optionsPanelEntity, pauseMenuComp)

	log.Printf("[OptionsPanelModule] Initialized successfully")

	return module, nil
}

// createConfirmButton 创建"确定"按钮
func (m *OptionsPanelModule) createConfirmButton(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 加载"返回游戏"按钮图片（原版资源）
	backToGameNormal := rm.GetImageByID("IMAGE_OPTIONS_BACKTOGAMEBUTTON0")
	backToGameHover := rm.GetImageByID("IMAGE_OPTIONS_BACKTOGAMEBUTTON2")

	if backToGameNormal == nil {
		return fmt.Errorf("failed to load back to game button images")
	}

	log.Printf("[OptionsPanelModule] Loaded confirm button images: Normal=%v, Hover=%v",
		backToGameNormal != nil, backToGameHover != nil)

	// 加载按钮文字字体（中文，使用配置的字号）
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuBackToGameButtonFontSize)
	if err != nil {
		log.Printf("[OptionsPanelModule] Warning: Failed to load button font: %v", err)
		buttonFont = nil
	}

	// 创建"确定"按钮实体
	m.confirmButtonEntity = m.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(m.entityManager, m.confirmButtonEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	// 添加按钮组件（简单图片按钮）
	ecs.AddComponent(m.entityManager, m.confirmButtonEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  backToGameNormal,
		HoverImage:   backToGameHover,
		PressedImage: backToGameHover,
		Text:         "确定",                    // 修改文本为"确定"
		Font:         buttonFont,               // 中文字体
		TextColor:    [4]uint8{0, 200, 0, 255}, // 绿色文字（与其他按钮一致）
		State:        components.UINormal,
		Enabled:      true,
		OnClick: func() {
			log.Printf("[OptionsPanelModule] Confirm button clicked!")
			m.Hide()
			if m.onClose != nil {
				m.onClose()
			}
		},
	})

	log.Printf("[OptionsPanelModule] Confirm button created")

	return nil
}

// createUIElements 创建UI元素（滑动条和复选框）
func (m *OptionsPanelModule) createUIElements(rm *game.ResourceManager) error {
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
			log.Printf("[OptionsPanelModule] Music volume changed: %.2f", value)
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
			log.Printf("[OptionsPanelModule] Sound volume changed: %.2f", value)
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
			log.Printf("[OptionsPanelModule] 3D acceleration toggled: %v", isChecked)
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
			log.Printf("[OptionsPanelModule] Fullscreen toggled: %v", isChecked)
			// TODO: 实际切换全屏
		},
	})

	return nil
}

// Update 更新选项面板状态
//
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 职责：
//   - 同步选项面板激活状态
//   - 控制按钮和 UI 元素显示/隐藏
func (m *OptionsPanelModule) Update(deltaTime float64) {
	// 获取选项面板组件
	optionsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.optionsPanelEntity)
	if !ok {
		return
	}

	// 根据激活状态更新 UI 元素位置
	if optionsPanel.IsActive {
		m.showUIElements()
	} else {
		m.hideUIElements()
	}
}

// showUIElements 显示 UI 元素（滑动条、复选框、按钮）
func (m *OptionsPanelModule) showUIElements() {
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0

	// UI元素在墓碑内部的垂直排列（与 PauseMenuModule 一致）
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

	// "确定"按钮：墓碑下方（与"返回游戏"按钮位置一致）
	if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.confirmButtonEntity); ok {
		buttonWidth := float64(button.NormalImage.Bounds().Dx())
		buttonX := screenCenterX - buttonWidth/2.0
		buttonY := screenCenterY + config.PauseMenuBackToGameButtonOffsetY

		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.confirmButtonEntity); ok {
			pos.X = buttonX
			pos.Y = buttonY
		}
	}
}

// hideUIElements 隐藏 UI 元素（移动到屏幕外）
func (m *OptionsPanelModule) hideUIElements() {
	hiddenX := -1000.0
	hiddenY := -1000.0

	uiElements := []ecs.EntityID{
		m.musicSliderEntity,
		m.soundSliderEntity,
		m.enable3DEntity,
		m.fullscreenEntity,
		m.confirmButtonEntity,
	}

	for _, entityID := range uiElements {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, entityID); ok {
			pos.X = hiddenX
			pos.Y = hiddenY
		}
	}
}

// Draw 渲染选项面板到屏幕
//
// 参数:
//   - screen: 目标渲染屏幕
//
// 渲染顺序：
//  1. 半透明遮罩
//  2. 墓碑背景面板
//  3. UI元素（滑动条、复选框）
//  4. "确定"按钮（在背景上方）
func (m *OptionsPanelModule) Draw(screen *ebiten.Image) {
	// 获取选项面板组件
	optionsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.optionsPanelEntity)
	if !ok || !optionsPanel.IsActive {
		return
	}

	// 1. 渲染遮罩和墓碑背景
	m.pauseMenuRenderSystem.Draw(screen)

	// 2. 渲染UI元素
	m.drawUIElements(screen)

	// 3. 渲染"确定"按钮
	if m.buttonRenderSystem != nil {
		m.buttonRenderSystem.DrawButton(screen, m.confirmButtonEntity)
	}
}

// drawUIElements 渲染UI元素（滑动条和复选框）
func (m *OptionsPanelModule) drawUIElements(screen *ebiten.Image) {
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

// drawSlider 渲染单个滑动条（复用 PauseMenuModule 的逻辑）
func (m *OptionsPanelModule) drawSlider(screen *ebiten.Image, slider *components.SliderComponent, x, y float64) {
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

// drawCheckbox 渲染单个复选框（复用 PauseMenuModule 的逻辑）
func (m *OptionsPanelModule) drawCheckbox(screen *ebiten.Image, checkbox *components.CheckboxComponent, x, y float64) {
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

// Show 显示选项面板
//
// 效果：
//   - 设置 PauseMenuComponent.IsActive = true
//   - UI 元素移动到正确位置
func (m *OptionsPanelModule) Show() {
	if optionsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.optionsPanelEntity); ok {
		optionsPanel.IsActive = true
		m.showUIElements()
		log.Printf("[OptionsPanelModule] Options panel shown")
	}
}

// Hide 隐藏选项面板
//
// 效果：
//   - 设置 PauseMenuComponent.IsActive = false
//   - UI 元素移动到屏幕外
func (m *OptionsPanelModule) Hide() {
	if optionsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.optionsPanelEntity); ok {
		optionsPanel.IsActive = false
		m.hideUIElements()
		log.Printf("[OptionsPanelModule] Options panel hidden")
	}
}

// IsActive 检查选项面板是否激活
//
// 返回:
//   - bool: 如果选项面板当前激活，返回 true
func (m *OptionsPanelModule) IsActive() bool {
	if optionsPanel, ok := ecs.GetComponent[*components.PauseMenuComponent](m.entityManager, m.optionsPanelEntity); ok {
		return optionsPanel.IsActive
	}
	return false
}

// Cleanup 清理模块资源
//
// 用途：
//   - 场景切换时清理所有选项面板实体
//   - 避免内存泄漏
func (m *OptionsPanelModule) Cleanup() {
	// 清理选项面板实体
	m.entityManager.DestroyEntity(m.optionsPanelEntity)

	// 清理按钮实体
	m.entityManager.DestroyEntity(m.confirmButtonEntity)

	// 清理 UI 元素实体
	m.entityManager.DestroyEntity(m.musicSliderEntity)
	m.entityManager.DestroyEntity(m.soundSliderEntity)
	m.entityManager.DestroyEntity(m.enable3DEntity)
	m.entityManager.DestroyEntity(m.fullscreenEntity)

	log.Printf("[OptionsPanelModule] Cleaned up")
}
