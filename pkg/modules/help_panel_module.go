package modules

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// HelpPanelModule 帮助面板模块
//
// 职责：
//   - 加载草皮背景（从游戏背景图裁剪并缩放）
//   - 加载便笺背景和帮助文本（使用 Alpha 蒙板叠加）
//   - 创建和管理"主菜单"按钮
//   - 处理面板显示/隐藏逻辑
//   - 渲染草皮背景、便笺背景、帮助文本和按钮
//
// 资源构成：
//   - 草皮背景：从 IMAGE_BACKGROUND1 裁剪并缩放到全屏
//   - 便笺背景：ZombieNote.jpg + ZombieNote_.png（Alpha 蒙板）
//   - 帮助文本：ZombieNoteHelp.png（黑底白字 → 透明底黑字）
//
// 使用场景：
//   - 主菜单场景：点击帮助按钮时显示
//   - 其他场景：需要显示帮助信息时复用
//
// 设计原则：
//   - 模块化：可在不同场景复用
//   - 自包含：封装所有帮助面板相关功能
//   - 低耦合：通过回调与外部交互
//
// Story 12.3: 对话框系统基础 - 帮助面板实现
type HelpPanelModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// 按钮渲染回调（避免循环导入）
	drawButtonFunc func(screen *ebiten.Image, buttonEntity ecs.EntityID)

	// 帮助面板实体
	helpPanelEntity ecs.EntityID

	// 按钮实体
	confirmButtonEntity ecs.EntityID // "主菜单"按钮

	// 草皮背景（从 IMAGE_BACKGROUND1 裁剪并缩放到全屏）
	lawnBackground *ebiten.Image

	// 便笺背景原始图片
	bgJPG  *ebiten.Image // 便笺背景 JPG
	bgMask *ebiten.Image // 便笺背景 Alpha 蒙板

	// 帮助文本原始图片
	textPNG *ebiten.Image // 帮助文本 PNG

	// 合成后的图片（首次 Draw 时生成）
	backgroundImage *ebiten.Image // 便笺背景（RGB + Alpha 蒙板合成）
	helpTextImage   *ebiten.Image // 帮助文本（黑底白字 → 透明底黑字）
	composited      bool          // 是否已经合成（避免重复处理）

	// 回调函数
	onClose func() // 关闭面板回调

	// 屏幕尺寸
	windowWidth  int
	windowHeight int

	// 便笺面板尺寸
	panelWidth  float64
	panelHeight float64
}

// NewHelpPanelModule 创建帮助面板模块
//
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于加载图片资源）
//   - drawButtonFunc: 按钮渲染回调函数（避免循环导入）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - onClose: 关闭面板回调函数（可选）
//
// 返回:
//   - *HelpPanelModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// 初始化流程：
//  1. 加载草皮背景（从 IMAGE_BACKGROUND1 裁剪并缩放到全屏）
//  2. 加载便笺背景和 Alpha 蒙板
//  3. 加载帮助文本
//  4. 创建"主菜单"按钮实体
//  5. 创建帮助面板实体
//
// Story 12.3: 对话框系统基础
func NewHelpPanelModule(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	drawButtonFunc func(screen *ebiten.Image, buttonEntity ecs.EntityID),
	windowWidth, windowHeight int,
	onClose func(),
) (*HelpPanelModule, error) {
	module := &HelpPanelModule{
		entityManager:  em,
		drawButtonFunc: drawButtonFunc,
		onClose:        onClose,
		windowWidth:    windowWidth,
		windowHeight:   windowHeight,
	}

	var err error

	// 1. 加载草皮背景（从 IMAGE_BACKGROUND1 裁剪并缩放到全屏）
	log.Printf("[HelpPanelModule] Loading lawn background...")
	bgImage, err := rm.LoadImageByID(config.NotePanelLawnBackgroundImage)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", config.NotePanelLawnBackgroundImage, err)
	}
	module.lawnBackground = cropAndScaleLawnBackground(bgImage, windowWidth, windowHeight)
	log.Printf("[HelpPanelModule] Lawn background loaded and scaled to %dx%d", windowWidth, windowHeight)

	// 2. 加载便笺背景和 Alpha 蒙板
	log.Printf("[HelpPanelModule] Loading note background images...")
	module.bgJPG, err = rm.LoadImage(config.ZombieNoteBackgroundJPG)
	if err != nil {
		return nil, fmt.Errorf("failed to load ZombieNote.jpg: %w", err)
	}

	module.bgMask, err = rm.LoadImage(config.ZombieNoteBackgroundMask)
	if err != nil {
		log.Printf("[HelpPanelModule] Warning: Failed to load ZombieNote_.png: %v", err)
		module.bgMask = nil
	}

	// 3. 加载帮助文本（黑底白字 → 透明底黑字）
	log.Printf("[HelpPanelModule] Loading help text image...")
	module.textPNG, err = rm.LoadImage("assets/images/ZombieNoteHelp.png")
	if err != nil {
		return nil, fmt.Errorf("failed to load ZombieNoteHelp.png: %w", err)
	}

	// 4. 设置便笺面板尺寸（基于便笺背景图）
	bgBounds := module.bgJPG.Bounds()
	module.panelWidth = float64(bgBounds.Dx())
	module.panelHeight = float64(bgBounds.Dy())

	// Alpha Mask 合成将在首次 Draw() 时执行
	module.composited = false

	// 5. 创建"主菜单"按钮
	if err := module.createConfirmButton(rm); err != nil {
		return nil, fmt.Errorf("failed to create confirm button: %w", err)
	}

	// 6. 创建帮助面板实体
	module.helpPanelEntity = em.CreateEntity()

	// 添加 HelpPanelComponent
	ecs.AddComponent(em, module.helpPanelEntity, &components.HelpPanelComponent{
		BackgroundImage:     module.lawnBackground,
		HelpTextImage:       nil, // 延迟合成
		ConfirmButtonEntity: uint64(module.confirmButtonEntity),
		IsActive:            false,
		Width:               float64(windowWidth),
		Height:              float64(windowHeight),
	})

	log.Printf("[HelpPanelModule] Initialized successfully")

	return module, nil
}

// cropAndScaleLawnBackground 从背景图裁剪并缩放草皮区域到全屏
//
// 裁剪逻辑：
//  1. 使用指定的起始点 (NotePanelLawnCropX, NotePanelLawnCropY) 和宽度 (NotePanelLawnCropWidth)
//  2. 根据目标屏幕的宽高比自动计算裁剪高度
//  3. 缩放到目标尺寸（因为比例相同，不会变形）
//
// 参数:
//   - bgImage: 原始背景图 (IMAGE_BACKGROUND1)
//   - targetWidth, targetHeight: 目标尺寸（窗口大小）
//
// 返回:
//   - 裁剪并缩放后的图片
func cropAndScaleLawnBackground(bgImage *ebiten.Image, targetWidth, targetHeight int) *ebiten.Image {
	// 目标宽高比
	targetRatio := float64(targetWidth) / float64(targetHeight)

	// 起始点和宽度（从配置读取）
	cropX := config.NotePanelLawnCropX
	cropY := config.NotePanelLawnCropY
	cropWidth := config.NotePanelLawnCropWidth

	// 根据目标比例自动计算裁剪高度
	cropHeight := int(float64(cropWidth) / targetRatio)

	// 裁剪区域
	cropRect := image.Rect(cropX, cropY, cropX+cropWidth, cropY+cropHeight)

	// 裁剪图片
	croppedImage := bgImage.SubImage(cropRect).(*ebiten.Image)

	// 创建目标尺寸的图片
	scaledImage := ebiten.NewImage(targetWidth, targetHeight)

	// 计算缩放比例
	scale := float64(targetWidth) / float64(cropWidth)

	// 绘制缩放后的图片
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	scaledImage.DrawImage(croppedImage, op)

	return scaledImage
}

// createConfirmButton 创建"主菜单"按钮（使用与奖励面板"下一关"按钮相同的样式）
func (m *HelpPanelModule) createConfirmButton(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 加载大按钮图片（与奖励面板下一关按钮一致）
	buttonImage, err := rm.LoadImage("assets/images/SeedChooser_Button.png")
	if err != nil {
		return fmt.Errorf("failed to load SeedChooser_Button.png: %w", err)
	}

	// 加载按钮发光图片（悬停状态）
	buttonGlowImage, err := rm.LoadImage("assets/images/SeedChooser_Button_Glow.png")
	if err != nil {
		log.Printf("[HelpPanelModule] Warning: Failed to load SeedChooser_Button_Glow.png: %v", err)
		buttonGlowImage = buttonImage // 降级使用普通图片
	}

	// 加载按钮文字字体（使用奖励面板按钮字体大小）
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.RewardPanelButtonTextFontSize)
	if err != nil {
		log.Printf("[HelpPanelModule] Warning: Failed to load button font: %v", err)
		buttonFont = nil
	}

	// 创建"主菜单"按钮实体
	m.confirmButtonEntity = m.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(m.entityManager, m.confirmButtonEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})

	// 计算按钮尺寸（从图片获取）
	buttonWidth := float64(buttonImage.Bounds().Dx())
	buttonHeight := float64(buttonImage.Bounds().Dy())

	// 添加按钮组件（大按钮样式，与奖励面板下一关按钮一致）
	ecs.AddComponent(m.entityManager, m.confirmButtonEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  buttonImage,
		HoverImage:   buttonGlowImage,
		PressedImage: buttonImage,
		Text:         "主菜单",
		Font:         buttonFont,
		TextColor:    [4]uint8{255, 200, 0, 255}, // 橙黄色文字
		Width:        buttonWidth,
		Height:       buttonHeight,
		State:        components.UINormal,
		Enabled:      true,
		OnClick: func() {
			log.Printf("[HelpPanelModule] Main menu button clicked!")
			m.Hide()
			if m.onClose != nil {
				m.onClose()
			}
		},
	})

	log.Printf("[HelpPanelModule] Main menu button created")

	return nil
}

// Update 更新帮助面板状态
//
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 职责：
//   - 同步帮助面板激活状态
//   - 控制按钮显示/隐藏
func (m *HelpPanelModule) Update(deltaTime float64) {
	// 获取帮助面板组件
	helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity)
	if !ok {
		return
	}

	// 根据激活状态更新按钮位置
	if helpPanel.IsActive {
		m.showButton()
	} else {
		m.hideButton()
	}
}

// showButton 显示"主菜单"按钮（移动到正确位置）
func (m *HelpPanelModule) showButton() {
	// 获取按钮组件
	button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.confirmButtonEntity)
	if !ok {
		return
	}

	// 屏幕中心位置
	screenCenterX := float64(m.windowWidth) / 2.0

	// 计算按钮宽度（简单图片按钮）
	buttonWidth := float64(button.NormalImage.Bounds().Dx())

	// 按钮位置：在屏幕下方居中
	buttonX := screenCenterX - buttonWidth/2.0
	buttonY := float64(m.windowHeight) - 80.0 // 距离屏幕底部 80 像素

	// 更新按钮位置
	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.confirmButtonEntity); ok {
		pos.X = buttonX
		pos.Y = buttonY
	}
}

// hideButton 隐藏"确定"按钮（移动到屏幕外）
func (m *HelpPanelModule) hideButton() {
	hiddenX := -1000.0
	hiddenY := -1000.0

	if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.confirmButtonEntity); ok {
		pos.X = hiddenX
		pos.Y = hiddenY
	}
}

// applyAlphaMasks 应用 Alpha 蒙板合成图片
//
// 职责：
//   - 在首次 Draw 时调用（此时游戏已经开始，可以使用 ReadPixels）
//   - 合成便笺背景：ZombieNote.jpg + ZombieNote_.png（Alpha 蒙板）
//   - 处理帮助文本：用亮度作为 Alpha，反转颜色（黑底白字 → 透明底黑字）
//   - 更新 HelpPanelComponent 的图片引用
//
// 注意：
//   - 必须在游戏主循环开始后调用（否则 ReadPixels 会 panic）
//   - 只执行一次（通过 composited 标记）
func (m *HelpPanelModule) applyAlphaMasks() {
	if m.composited {
		return // 已经合成过了
	}

	// 1. 合成便笺背景（ZombieNote.jpg + ZombieNote_.png）
	if m.bgMask != nil {
		m.backgroundImage = utils.ApplyAlphaMask(m.bgJPG, m.bgMask)
		log.Printf("[HelpPanelModule] Applied alpha mask to note background")
	} else {
		// 没有蒙板，直接使用原图
		m.backgroundImage = m.bgJPG
		log.Printf("[HelpPanelModule] Using note background without mask")
	}

	// 2. 处理帮助文本：使用蒙板（像草皮渲染一样）
	// 原图：黑底白字
	// 策略：用原图自身作为蒙板（白色文字→不透明，黑色背景→透明）
	// 然后将所有非透明像素设为黑色（不反转，直接设黑）
	maskedText := utils.ApplyAlphaMask(m.textPNG, m.textPNG)

	// 3. 将白色文字转为黑色（不反转，直接设为黑色）
	m.helpTextImage = m.convertToBlack(maskedText)
	log.Printf("[HelpPanelModule] Applied alpha mask (self as mask) and converted to black")

	// 4. 更新 HelpPanelComponent 的图片引用
	helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity)
	if ok {
		helpPanel.HelpTextImage = m.helpTextImage
	}

	// 5. 标记为已合成
	m.composited = true
	log.Printf("[HelpPanelModule] Image composition completed")
}

// convertToBlack 将透明底白字转换为透明底黑字（使用预乘 Alpha）
//
// 处理方法：
//   - 将所有像素的 RGB 设置为黑色，并应用预乘 Alpha
//   - 预乘 Alpha：finalRGB = targetRGB * (alpha / 255)
//   - 这可消除边缘的白色残留（参考 LoadCompositedImage 实现）
//
// 为什么使用预乘 Alpha：
//   - ApplyAlphaMask 输出的是非预乘 Alpha
//   - 半透明边缘（如 R=200, A=200）在非预乘模式下会显示为浅灰色
//   - 预乘后（R=200*200/255=157, A=200）渲染时边缘更暗，消除白边
//
// 参数：
//   - src: 原图（透明底白字，非预乘 Alpha）
//
// 返回：
//   - 处理后的图片（透明底黑字，预乘 Alpha）
func (m *HelpPanelModule) convertToBlack(src *ebiten.Image) *ebiten.Image {
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

// Draw 渲染帮助面板到屏幕
//
// 参数:
//   - screen: 目标渲染屏幕
//
// 渲染顺序：
//  1. 草皮背景（全屏）
//  2. 便笺背景（居中）
//  3. 帮助文本（在便笺上居中）
//  4. "主菜单"按钮（在屏幕下方）
func (m *HelpPanelModule) Draw(screen *ebiten.Image) {
	// 获取帮助面板组件
	helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity)
	if !ok || !helpPanel.IsActive {
		return
	}

	// 延迟合成 Alpha Mask（首次 Draw 时执行）
	// 必须在游戏主循环开始后才能调用 ReadPixels
	if !m.composited {
		m.applyAlphaMasks()
	}

	// 计算便笺面板居中位置
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0
	panelX := screenCenterX - m.panelWidth/2.0
	panelY := screenCenterY - m.panelHeight/2.0

	// 1. 绘制草皮背景（全屏，位置 0,0）
	lawnOp := &ebiten.DrawImageOptions{}
	screen.DrawImage(helpPanel.BackgroundImage, lawnOp)

	// 2. 绘制便笺背景（居中）
	if m.backgroundImage != nil {
		bgOp := &ebiten.DrawImageOptions{}
		bgOp.GeoM.Translate(panelX, panelY)
		screen.DrawImage(m.backgroundImage, bgOp)
	}

	// 3. 绘制帮助文本（在便笺上居中）
	if helpPanel.HelpTextImage != nil {
		textWidth := float64(helpPanel.HelpTextImage.Bounds().Dx())
		textHeight := float64(helpPanel.HelpTextImage.Bounds().Dy())
		textX := screenCenterX - textWidth/2.0
		textY := screenCenterY - textHeight/2.0

		textOp := &ebiten.DrawImageOptions{}
		textOp.GeoM.Translate(textX, textY)
		screen.DrawImage(helpPanel.HelpTextImage, textOp)
	}

	// 4. 绘制"主菜单"按钮
	if m.drawButtonFunc != nil {
		m.drawButtonFunc(screen, m.confirmButtonEntity)
	}
}

// drawOverlay 绘制半透明遮罩
func (m *HelpPanelModule) drawOverlay(screen *ebiten.Image) {
	// 创建半透明黑色遮罩
	overlay := ebiten.NewImage(m.windowWidth, m.windowHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 128}) // 50% 透明度
	screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
}

// Show 显示帮助面板
//
// 效果：
//   - 设置 HelpPanelComponent.IsActive = true
//   - 按钮移动到正确位置
func (m *HelpPanelModule) Show() {
	if helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity); ok {
		helpPanel.IsActive = true
		m.showButton()
		log.Printf("[HelpPanelModule] Help panel shown")
	}
}

// Hide 隐藏帮助面板
//
// 效果：
//   - 设置 HelpPanelComponent.IsActive = false
//   - 按钮移动到屏幕外
func (m *HelpPanelModule) Hide() {
	if helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity); ok {
		helpPanel.IsActive = false
		m.hideButton()
		log.Printf("[HelpPanelModule] Help panel hidden")
	}
}

// IsActive 检查帮助面板是否激活
//
// 返回:
//   - bool: 如果帮助面板当前激活，返回 true
func (m *HelpPanelModule) IsActive() bool {
	if helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity); ok {
		return helpPanel.IsActive
	}
	return false
}

// Cleanup 清理模块资源
//
// 用途：
//   - 场景切换时清理所有帮助面板实体
//   - 避免内存泄漏
func (m *HelpPanelModule) Cleanup() {
	// 清理帮助面板实体
	m.entityManager.DestroyEntity(m.helpPanelEntity)

	// 清理按钮实体
	m.entityManager.DestroyEntity(m.confirmButtonEntity)

	log.Printf("[HelpPanelModule] Cleaned up")
}
