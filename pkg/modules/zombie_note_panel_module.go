package modules

import (
	"fmt"
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ZombieNotePanelModule 僵尸来信面板模块（Story 8.14）
//
// 职责：
//   - 加载便笺背景和来信内容图片（使用 Alpha 蒙板叠加）
//   - 创建和管理"下一关"按钮和"主菜单"按钮
//   - 处理面板显示/隐藏逻辑
//   - 渲染遮罩、便笺背景、来信内容、标题文字和按钮
//
// 资源构成：
//   - 便笺背景：ZombieNote.jpg + Alpha 蒙板 ZombieNote_.png
//   - 来信内容：ZombieNote{N}.png（如 ZombieNote1.png）
//
// 使用场景：
//   - Level 1-9 通关后显示僵尸来信
//   - 后续章节可能有更多来信类型
//
// 设计原则：
//   - 模块化：可在不同场景复用
//   - 自包含：封装所有来信面板相关功能
//   - 低耦合：通过回调与外部交互
type ZombieNotePanelModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// 按钮渲染回调（避免循环导入）
	drawButtonFunc func(screen *ebiten.Image, buttonEntity ecs.EntityID)

	// 面板实体
	panelEntity ecs.EntityID

	// 按钮实体
	nextLevelButtonEntity ecs.EntityID // "下一关"按钮
	mainMenuButtonEntity  ecs.EntityID // "主菜单"按钮

	// 草皮背景（全屏，从 IMAGE_BACKGROUND1 裁剪并缩放）
	lawnBackground *ebiten.Image

	// 原始图片（未合成，延迟处理避免 ReadPixels 错误）
	bgJPG   *ebiten.Image // 便笺背景 JPG
	bgMask  *ebiten.Image // 便笺背景 Alpha 蒙板
	notePNG *ebiten.Image // 来信内容 PNG

	// 合成后的图片（首次 Draw 时生成）
	backgroundImage *ebiten.Image // 便笺背景（RGB + Alpha 蒙板合成）
	noteImage       *ebiten.Image // 来信内容
	composited      bool          // 是否已经合成

	// 标题文字
	titleText string                 // 标题文本（从 LawnStrings 加载）
	titleFont *text.GoTextFaceSource // 标题字体

	// 回调函数
	onNextLevel func() // 点击"下一关"回调
	onMainMenu  func() // 点击"主菜单"回调

	// 屏幕尺寸
	windowWidth  int
	windowHeight int

	// 面板状态
	isActive    bool    // 是否激活
	fadeAlpha   float64 // 淡入透明度（0.0 - 1.0）
	panelWidth  float64 // 便笺面板宽度
	panelHeight float64 // 便笺面板高度
}

// NewZombieNotePanelModule 创建僵尸来信面板模块
//
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于加载图片资源）
//   - gs: GameState 实例（用于获取 LawnStrings）
//   - drawButtonFunc: 按钮渲染回调函数（避免循环导入）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - noteID: 来信 ID（如 "zombienote1"，对应 ZombieNote1.png）
//   - onNextLevel: 点击"下一关"按钮回调
//   - onMainMenu: 点击"主菜单"按钮回调
//
// 返回:
//   - *ZombieNotePanelModule: 新创建的模块实例
//   - error: 如果初始化失败
func NewZombieNotePanelModule(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	gs *game.GameState,
	drawButtonFunc func(screen *ebiten.Image, buttonEntity ecs.EntityID),
	windowWidth, windowHeight int,
	noteID string,
	onNextLevel func(),
	onMainMenu func(),
) (*ZombieNotePanelModule, error) {
	module := &ZombieNotePanelModule{
		entityManager:  em,
		drawButtonFunc: drawButtonFunc,
		onNextLevel:    onNextLevel,
		onMainMenu:     onMainMenu,
		windowWidth:    windowWidth,
		windowHeight:   windowHeight,
		isActive:       false,
		fadeAlpha:      0.0,
	}

	var err error

	// 1. 加载草皮背景（从 IMAGE_BACKGROUND1 裁剪并缩放到全屏）
	log.Printf("[ZombieNotePanelModule] Loading lawn background...")
	bgImage, err := rm.LoadImageByID(config.NotePanelLawnBackgroundImage)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", config.NotePanelLawnBackgroundImage, err)
	}
	module.lawnBackground = cropAndScaleLawnBackground(bgImage, windowWidth, windowHeight)
	log.Printf("[ZombieNotePanelModule] Lawn background loaded and scaled to %dx%d", windowWidth, windowHeight)

	// 2. 加载便笺背景和 Alpha 蒙板
	log.Printf("[ZombieNotePanelModule] Loading note background images...")
	module.bgJPG, err = rm.LoadImage(config.ZombieNoteBackgroundJPG)
	if err != nil {
		return nil, fmt.Errorf("failed to load ZombieNote.jpg: %w", err)
	}

	module.bgMask, err = rm.LoadImage(config.ZombieNoteBackgroundMask)
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load ZombieNote_.png: %v", err)
		module.bgMask = nil // 没有蒙板就直接使用原图
	}

	// 3. 加载来信内容图片
	notePath := config.GetZombieNoteImagePath(noteID)
	log.Printf("[ZombieNotePanelModule] Loading note image: %s", notePath)
	module.notePNG, err = rm.LoadImage(notePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", notePath, err)
	}

	// 4. 加载标题文字（从 LawnStrings）
	if gs != nil && gs.LawnStrings != nil {
		module.titleText = gs.LawnStrings.GetString(config.ZombieNoteTitleKey)
		if module.titleText == "" {
			module.titleText = "你发现了张便签：" // 默认文本
		}
	} else {
		module.titleText = "你发现了张便签："
	}

	// 5. 加载标题字体
	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.ZombieNoteTitleFontSize)
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load title font: %v", err)
		module.titleFont = nil
	} else {
		module.titleFont = titleFont.Source
	}

	// 6. 设置便笺面板尺寸（基于便笺背景图）
	bgBounds := module.bgJPG.Bounds()
	module.panelWidth = float64(bgBounds.Dx())
	module.panelHeight = float64(bgBounds.Dy())

	// 7. 创建按钮
	if err := module.createButtons(rm); err != nil {
		return nil, fmt.Errorf("failed to create buttons: %w", err)
	}

	// 8. 创建面板实体
	module.panelEntity = em.CreateEntity()

	// Alpha Mask 合成将在首次 Draw() 时执行
	module.composited = false

	log.Printf("[ZombieNotePanelModule] Initialized successfully (noteID: %s)", noteID)

	return module, nil
}

// createButtons 创建"下一关"和"主菜单"按钮
func (m *ZombieNotePanelModule) createButtons(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 加载按钮图片
	buttonImage, err := rm.LoadImage("assets/images/SeedChooser_Button.png")
	if err != nil {
		return fmt.Errorf("failed to load SeedChooser_Button.png: %w", err)
	}

	buttonGlowImage, err := rm.LoadImage("assets/images/SeedChooser_Button_Glow.png")
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load SeedChooser_Button_Glow.png: %v", err)
		buttonGlowImage = buttonImage
	}

	// 加载小按钮图片（主菜单按钮）
	smallButtonImage, err := rm.LoadImage("assets/images/SeedChooser_Button2.png")
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load SeedChooser_Button2.png: %v", err)
		smallButtonImage = buttonImage
	}

	smallButtonGlowImage, err := rm.LoadImage("assets/images/SeedChooser_Button2_Glow.png")
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load SeedChooser_Button2_Glow.png: %v", err)
		smallButtonGlowImage = smallButtonImage
	}

	// 加载按钮字体（下一关按钮使用大字体）
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.ZombieNoteButtonTextFontSize)
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load button font: %v", err)
		buttonFont = nil
	}

	// 加载主菜单按钮字体（使用较小字体，与奖励面板一致）
	mainMenuButtonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.ZombieNoteMainMenuButtonFontSize)
	if err != nil {
		log.Printf("[ZombieNotePanelModule] Warning: Failed to load main menu button font: %v", err)
		mainMenuButtonFont = buttonFont // 回退到默认按钮字体
	}

	// === 创建"下一关"按钮 ===
	m.nextLevelButtonEntity = m.entityManager.CreateEntity()

	ecs.AddComponent(m.entityManager, m.nextLevelButtonEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	buttonWidth := float64(buttonImage.Bounds().Dx())
	buttonHeight := float64(buttonImage.Bounds().Dy())

	ecs.AddComponent(m.entityManager, m.nextLevelButtonEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  buttonImage,
		HoverImage:   buttonGlowImage,
		PressedImage: buttonImage,
		Text:         "下一关",
		Font:         buttonFont,
		TextColor:    [4]uint8{255, 200, 0, 255}, // 橙黄色
		Width:        buttonWidth,
		Height:       buttonHeight,
		State:        components.UINormal,
		Enabled:      true,
		OnClick: func() {
			log.Printf("[ZombieNotePanelModule] Next level button clicked!")
			if m.onNextLevel != nil {
				m.onNextLevel()
			}
		},
	})

	// === 创建"主菜单"按钮 ===
	m.mainMenuButtonEntity = m.entityManager.CreateEntity()

	ecs.AddComponent(m.entityManager, m.mainMenuButtonEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	smallButtonWidth := float64(smallButtonImage.Bounds().Dx())
	smallButtonHeight := float64(smallButtonImage.Bounds().Dy())

	ecs.AddComponent(m.entityManager, m.mainMenuButtonEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  smallButtonImage,
		HoverImage:   smallButtonGlowImage,
		PressedImage: smallButtonImage,
		Text:         "主菜单",
		Font:         mainMenuButtonFont,
		TextColor:    [4]uint8{config.ZombieNoteMainMenuButtonTextColor.R, config.ZombieNoteMainMenuButtonTextColor.G, config.ZombieNoteMainMenuButtonTextColor.B, config.ZombieNoteMainMenuButtonTextColor.A},
		NoTextShadow: true, // 主菜单按钮不需要文字阴影
		Width:        smallButtonWidth,
		Height:       smallButtonHeight,
		State:        components.UINormal,
		Enabled:      true,
		OnClick: func() {
			log.Printf("[ZombieNotePanelModule] Main menu button clicked!")
			if m.onMainMenu != nil {
				m.onMainMenu()
			}
		},
	})

	log.Printf("[ZombieNotePanelModule] Buttons created")
	return nil
}

// applyAlphaMasks 应用 Alpha 蒙板合成图片
func (m *ZombieNotePanelModule) applyAlphaMasks() {
	if m.composited {
		return
	}

	// 1. 合成便笺背景（ZombieNote.jpg + ZombieNote_.png）
	if m.bgMask != nil {
		m.backgroundImage = utils.ApplyAlphaMask(m.bgJPG, m.bgMask)
		log.Printf("[ZombieNotePanelModule] Applied alpha mask to background")
	} else {
		// 没有蒙板，直接使用原图
		m.backgroundImage = m.bgJPG
		log.Printf("[ZombieNotePanelModule] Using background without mask")
	}

	// 2. 处理来信内容图片（与 HelpPanelModule 一致）
	// 原图：黑底白字
	// 策略：用原图自身作为蒙板（白色文字→不透明，黑色背景→透明）
	// 然后将所有非透明像素设为黑色
	maskedNote := utils.ApplyAlphaMask(m.notePNG, m.notePNG)
	m.noteImage = m.convertToBlack(maskedNote)
	log.Printf("[ZombieNotePanelModule] Applied alpha mask and converted note to black")

	m.composited = true
	log.Printf("[ZombieNotePanelModule] Image composition completed")
}

// convertToBlack 将透明底白字转换为透明底黑字（使用预乘 Alpha）
//
// 处理流程：
//  1. 读取源图像素
//  2. 将所有像素的 RGB 设置为黑色 (0, 0, 0)
//  3. 保持 Alpha 通道不变
//
// 为什么使用预乘 Alpha：
//   - 半透明边缘（如 R=200, A=200）在非预乘模式下会显示为浅灰色
//   - 预乘后边缘更暗，消除白边
//
// 参数：
//   - src: 原图（透明底白字，非预乘 Alpha）
//
// 返回：
//   - 处理后的图片（透明底黑字，预乘 Alpha）
func (m *ZombieNotePanelModule) convertToBlack(src *ebiten.Image) *ebiten.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建新图片
	result := ebiten.NewImage(width, height)

	// 读取源图像素
	pixels := make([]byte, width*height*4)
	src.ReadPixels(pixels)

	// 将所有像素设为黑色（使用预乘 Alpha）
	for i := 0; i < len(pixels); i += 4 {
		alpha := pixels[i+3]

		// 目标颜色：黑色 (0, 0, 0)
		// 预乘 Alpha：finalRGB = targetRGB * (alpha / 255)
		// 对于黑色：0 * (alpha / 255) = 0
		// 所以预乘 Alpha 后 RGB 仍然是 0
		pixels[i+0] = 0     // R = 0
		pixels[i+1] = 0     // G = 0
		pixels[i+2] = 0     // B = 0
		pixels[i+3] = alpha // A 保持不变
	}

	// 写入结果
	result.WritePixels(pixels)
	return result
}

// Update 更新面板状态
func (m *ZombieNotePanelModule) Update(deltaTime float64) {
	if !m.isActive {
		return
	}

	// 更新按钮位置（确保在正确位置）
	m.updateButtonPositions()
}

// updateButtonPositions 更新按钮位置
func (m *ZombieNotePanelModule) updateButtonPositions() {
	screenWidth := float64(m.windowWidth)
	screenHeight := float64(m.windowHeight)
	screenCenterX := screenWidth / 2.0

	// === "下一关"按钮位置（屏幕下方居中，使用比例配置） ===
	if nextBtn, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.nextLevelButtonEntity); ok {
		buttonWidth := nextBtn.Width
		buttonX := screenCenterX - buttonWidth/2.0
		buttonY := screenHeight * config.ZombieNoteButtonY

		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.nextLevelButtonEntity); ok {
			pos.X = buttonX
			pos.Y = buttonY
		}
	}

	// === "主菜单"按钮位置（屏幕右上角，使用比例配置） ===
	if mainBtn, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.mainMenuButtonEntity); ok {
		buttonWidth := mainBtn.Width
		buttonX := screenWidth*config.ZombieNoteMainMenuButtonX - buttonWidth/2.0
		buttonY := screenHeight * config.ZombieNoteMainMenuButtonY

		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.mainMenuButtonEntity); ok {
			pos.X = buttonX
			pos.Y = buttonY
		}
	}
}

// Draw 渲染面板
func (m *ZombieNotePanelModule) Draw(screen *ebiten.Image) {
	if !m.isActive {
		return
	}

	// 延迟合成 Alpha Mask
	if !m.composited {
		m.applyAlphaMasks()
	}

	// 计算便笺面板居中位置
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0
	panelX := screenCenterX - m.panelWidth/2.0
	panelY := screenCenterY - m.panelHeight/2.0

	// 1. 绘制草皮背景（全屏，应用淡入效果）
	if m.lawnBackground != nil {
		lawnOp := &ebiten.DrawImageOptions{}
		lawnOp.ColorScale.ScaleAlpha(float32(m.fadeAlpha))
		screen.DrawImage(m.lawnBackground, lawnOp)
	}

	// 2. 绘制便笺背景（居中，应用淡入效果）
	if m.backgroundImage != nil {
		bgOp := &ebiten.DrawImageOptions{}
		bgOp.GeoM.Translate(panelX, panelY)
		bgOp.ColorScale.ScaleAlpha(float32(m.fadeAlpha))
		screen.DrawImage(m.backgroundImage, bgOp)
	}

	// 3. 绘制标题文字（"你发现了张便签："）
	m.drawTitle(screen, panelX, panelY)

	// 4. 绘制来信内容图片（在便笺上方居中）
	m.drawNoteImage(screen, panelX, panelY)

	// 5. 绘制按钮
	if m.drawButtonFunc != nil {
		m.drawButtonFunc(screen, m.nextLevelButtonEntity)
		m.drawButtonFunc(screen, m.mainMenuButtonEntity)
	}
}

// drawOverlay 绘制半透明遮罩
func (m *ZombieNotePanelModule) drawOverlay(screen *ebiten.Image) {
	overlay := ebiten.NewImage(m.windowWidth, m.windowHeight)
	overlayAlpha := uint8(float64(config.ZombieNotePanelOverlayAlpha) * m.fadeAlpha)
	overlay.Fill(color.RGBA{0, 0, 0, overlayAlpha})
	screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
}

// drawTitle 绘制标题文字（带阴影效果）
func (m *ZombieNotePanelModule) drawTitle(screen *ebiten.Image, panelX, panelY float64) {
	if m.titleFont == nil || m.titleText == "" {
		return
	}

	// 创建字体 Face
	face := &text.GoTextFace{
		Source: m.titleFont,
		Size:   config.ZombieNoteTitleFontSize,
	}

	// 计算文字宽度（用于居中）
	textWidth, _ := text.Measure(m.titleText, face, 0)

	// 文字位置（屏幕顶部居中，使用屏幕顶部为基准）
	screenCenterX := float64(m.windowWidth) / 2.0
	textX := screenCenterX - textWidth/2.0
	textY := config.ZombieNoteTitleOffsetY // 相对于屏幕顶部的偏移

	// 阴影偏移量
	shadowOffsetX := 2.0
	shadowOffsetY := 2.0

	// 1. 先绘制阴影（深色文字，偏移位置）
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(textX+shadowOffsetX, textY+shadowOffsetY)
	shadowOp.LayoutOptions.PrimaryAlign = text.AlignStart   // 水平从左开始
	shadowOp.LayoutOptions.SecondaryAlign = text.AlignStart // 垂直从顶部开始
	shadowAlpha := float32(0.7) * float32(m.fadeAlpha)      // 阴影透明度
	shadowOp.ColorScale.Scale(0, 0, 0, shadowAlpha)         // 黑色阴影
	text.Draw(screen, m.titleText, face, shadowOp)

	// 2. 再绘制主文字
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(textX, textY)
	textOp.LayoutOptions.PrimaryAlign = text.AlignStart   // 水平从左开始
	textOp.LayoutOptions.SecondaryAlign = text.AlignStart // 垂直从顶部开始
	r := float32(config.ZombieNoteTitleColor.R) / 255.0
	g := float32(config.ZombieNoteTitleColor.G) / 255.0
	b := float32(config.ZombieNoteTitleColor.B) / 255.0
	a := float32(config.ZombieNoteTitleColor.A) / 255.0 * float32(m.fadeAlpha)
	textOp.ColorScale.Scale(r, g, b, a)

	text.Draw(screen, m.titleText, face, textOp)
}

// drawNoteImage 绘制来信内容图片
func (m *ZombieNotePanelModule) drawNoteImage(screen *ebiten.Image, panelX, panelY float64) {
	if m.noteImage == nil {
		return
	}

	// 计算图片居中位置（在面板内居中）
	noteWidth := float64(m.noteImage.Bounds().Dx())
	noteHeight := float64(m.noteImage.Bounds().Dy())

	// 面板中央位置
	panelCenterX := panelX + m.panelWidth/2.0
	panelCenterY := panelY + m.panelHeight/2.0

	noteX := panelCenterX - noteWidth/2.0
	noteY := panelCenterY - noteHeight/2.0

	// 绘制来信图片（应用淡入效果）
	noteOp := &ebiten.DrawImageOptions{}
	noteOp.GeoM.Translate(noteX, noteY)
	noteOp.ColorScale.ScaleAlpha(float32(m.fadeAlpha))
	screen.DrawImage(m.noteImage, noteOp)
}

// Show 显示面板
func (m *ZombieNotePanelModule) Show() {
	m.isActive = true
	m.fadeAlpha = 1.0 // 直接显示（淡入由外部控制）
	m.updateButtonPositions()
	log.Printf("[ZombieNotePanelModule] Panel shown")
}

// Hide 隐藏面板
func (m *ZombieNotePanelModule) Hide() {
	m.isActive = false
	m.hideButtons()
	log.Printf("[ZombieNotePanelModule] Panel hidden")
}

// hideButtons 隐藏按钮
func (m *ZombieNotePanelModule) hideButtons() {
	hiddenX := -1000.0
	hiddenY := -1000.0

	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.nextLevelButtonEntity); ok {
		pos.X = hiddenX
		pos.Y = hiddenY
	}

	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.mainMenuButtonEntity); ok {
		pos.X = hiddenX
		pos.Y = hiddenY
	}
}

// SetFadeAlpha 设置淡入透明度
func (m *ZombieNotePanelModule) SetFadeAlpha(alpha float64) {
	m.fadeAlpha = alpha
}

// IsActive 检查面板是否激活
func (m *ZombieNotePanelModule) IsActive() bool {
	return m.isActive
}

// Cleanup 清理模块资源
func (m *ZombieNotePanelModule) Cleanup() {
	m.entityManager.DestroyEntity(m.panelEntity)
	m.entityManager.DestroyEntity(m.nextLevelButtonEntity)
	m.entityManager.DestroyEntity(m.mainMenuButtonEntity)
	log.Printf("[ZombieNotePanelModule] Cleaned up")
}

// IsHoveringNextButton 检查鼠标是否悬停在"下一关"按钮上
func (m *ZombieNotePanelModule) IsHoveringNextButton() bool {
	if !m.isActive {
		return false
	}

	mouseX, mouseY := utils.GetPointerPosition()

	pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.nextLevelButtonEntity)
	if !ok {
		return false
	}

	btn, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.nextLevelButtonEntity)
	if !ok {
		return false
	}

	return float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+btn.Width &&
		float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+btn.Height
}

// IsHoveringMainMenuButton 检查鼠标是否悬停在"主菜单"按钮上
func (m *ZombieNotePanelModule) IsHoveringMainMenuButton() bool {
	if !m.isActive {
		return false
	}

	mouseX, mouseY := utils.GetPointerPosition()

	pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.mainMenuButtonEntity)
	if !ok {
		return false
	}

	btn, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.mainMenuButtonEntity)
	if !ok {
		return false
	}

	return float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+btn.Width &&
		float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+btn.Height
}
