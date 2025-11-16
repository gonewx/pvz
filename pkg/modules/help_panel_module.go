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
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// HelpPanelModule 帮助面板模块
//
// 职责：
//   - 加载便笺背景和帮助文本（使用 Alpha 蒙板叠加）
//   - 创建和管理"确定"按钮
//   - 处理面板显示/隐藏逻辑
//   - 渲染遮罩、便笺背景、帮助文本和按钮
//
// 资源构成：
//   - 便笺背景：ZombieNote.jpg + Alpha 蒙板 ZombieNote_.png
//   - 帮助文本：ZombieNoteHelp.png + Alpha 蒙板 ZombieNoteHelpBlack.png
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

	// 系统（内部管理）
	buttonSystem       *systems.ButtonSystem       // 按钮交互（引用，不拥有）
	buttonRenderSystem *systems.ButtonRenderSystem // 按钮渲染（引用，不拥有）

	// 帮助面板实体
	helpPanelEntity ecs.EntityID

	// 按钮实体
	confirmButtonEntity ecs.EntityID // "确定"按钮

	// 原始图片（未合成，延迟处理避免 ReadPixels 错误）
	bgJPG    *ebiten.Image // 便笺背景 JPG
	bgMask   *ebiten.Image // 便笺背景 Alpha 蒙板
	textPNG  *ebiten.Image // 帮助文本 PNG
	textMask *ebiten.Image // 帮助文本 Alpha 蒙板

	// 合成后的图片（首次 Draw 时生成）
	backgroundImage *ebiten.Image // 便笺背景（RGB + Alpha 蒙板合成）
	helpTextImage   *ebiten.Image // 帮助文本（RGB + Alpha 蒙板合成）
	composited      bool          // 是否已经合成（避免重复处理）

	// 回调函数
	onClose func() // 关闭面板回调

	// 屏幕尺寸
	windowWidth  int
	windowHeight int
}

// NewHelpPanelModule 创建帮助面板模块
//
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于加载图片资源）
//   - buttonSystem: 按钮交互系统（引用，不拥有）
//   - buttonRenderSystem: 按钮渲染系统（引用，不拥有）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - onClose: 关闭面板回调函数（可选）
//
// 返回:
//   - *HelpPanelModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// 初始化流程：
//   1. 加载便笺背景和 Alpha 蒙板，合成
//   2. 加载帮助文本和 Alpha 蒙板，合成
//   3. 创建"确定"按钮实体
//   4. 创建帮助面板实体，添加 HelpPanelComponent
//
// Story 12.3: 对话框系统基础
func NewHelpPanelModule(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	buttonSystem *systems.ButtonSystem,
	buttonRenderSystem *systems.ButtonRenderSystem,
	windowWidth, windowHeight int,
	onClose func(),
) (*HelpPanelModule, error) {
	module := &HelpPanelModule{
		entityManager:      em,
		buttonSystem:       buttonSystem,
		buttonRenderSystem: buttonRenderSystem,
		onClose:            onClose,
		windowWidth:        windowWidth,
		windowHeight:       windowHeight,
	}

	var err error

	// 1. 加载便笺背景和 Alpha 蒙板（延迟处理，避免 ReadPixels 在游戏开始前调用）
	log.Printf("[HelpPanelModule] Loading background images...")
	module.bgJPG, err = rm.LoadImage("assets/images/ZombieNote.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to load ZombieNote.jpg: %w", err)
	}

	module.bgMask, err = rm.LoadImage("assets/images/ZombieNote_.png")
	if err != nil {
		log.Printf("[HelpPanelModule] Warning: Failed to load ZombieNote_.png: %v", err)
		module.bgMask = nil // 没有蒙板就直接使用原图
	}

	// 2. 加载帮助文本（不需要蒙板，使用原图亮度作为 Alpha）
	// ZombieNoteHelp.png：黑底白字
	// ZombieNoteHelpBlack.png：全黑（无效的占位符）
	// 处理目标：白字→黑字，黑底→透明
	log.Printf("[HelpPanelModule] Loading help text image...")
	module.textPNG, err = rm.LoadImage("assets/images/ZombieNoteHelp.png")
	if err != nil {
		return nil, fmt.Errorf("failed to load ZombieNoteHelp.png: %w", err)
	}

	// 不需要加载蒙板（使用原图亮度作为 Alpha）
	module.textMask = nil

	// Alpha Mask 合成将在首次 Draw() 时执行（此时游戏已经开始）
	module.composited = false

	// 3. 创建"确定"按钮
	if err := module.createConfirmButton(rm); err != nil {
		return nil, fmt.Errorf("failed to create confirm button: %w", err)
	}

	// 4. 创建帮助面板实体
	module.helpPanelEntity = em.CreateEntity()

	// 获取背景图片尺寸（用于居中）
	bgBounds := module.bgJPG.Bounds()
	width := float64(bgBounds.Dx())
	height := float64(bgBounds.Dy())

	// 添加 HelpPanelComponent（此时图片还未合成，将在 Draw 时处理）
	ecs.AddComponent(em, module.helpPanelEntity, &components.HelpPanelComponent{
		BackgroundImage:     nil, // 延迟合成
		HelpTextImage:       nil, // 延迟合成
		ConfirmButtonEntity: uint64(module.confirmButtonEntity),
		IsActive:            false, // 初始状态：未激活
		Width:               width,
		Height:              height,
	})

	log.Printf("[HelpPanelModule] Initialized successfully")

	return module, nil
}

// createConfirmButton 创建"确定"按钮
func (m *HelpPanelModule) createConfirmButton(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 按钮文本
	buttonText := "确定"

	// 使用三段式按钮（复用 entities.NewMenuButton）
	entity, err := entities.NewMenuButton(
		m.entityManager,
		rm,
		hiddenX,
		hiddenY,
		buttonText,
		config.PauseMenuInnerButtonFontSize,      // 使用与暂停菜单一致的字体大小
		[4]uint8{0, 200, 0, 255},                 // 绿色文字
		config.PauseMenuInnerButtonWidth-40,      // 减去左右边框宽度
		func() {                                  // 点击回调
			log.Printf("[HelpPanelModule] Confirm button clicked!")
			m.Hide()
			if m.onClose != nil {
				m.onClose()
			}
		},
	)

	if err != nil {
		return err
	}

	m.confirmButtonEntity = entity
	log.Printf("[HelpPanelModule] Confirm button created")

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

// showButton 显示"确定"按钮（移动到正确位置）
func (m *HelpPanelModule) showButton() {
	// 获取帮助面板组件（用于计算按钮位置）
	helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity)
	if !ok {
		return
	}

	// 获取按钮组件
	button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, m.confirmButtonEntity)
	if !ok {
		return
	}

	// 屏幕中心位置
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0

	// 计算按钮宽度（三段式按钮）
	var buttonWidth float64
	if button.Type == components.ButtonTypeNineSlice {
		leftWidth := float64(button.LeftImage.Bounds().Dx())
		rightWidth := float64(button.RightImage.Bounds().Dx())
		buttonWidth = leftWidth + button.MiddleWidth + rightWidth
	} else {
		buttonWidth = config.PauseMenuInnerButtonWidth
	}

	// 按钮位置：在便笺下方居中
	// 便笺底部 Y 坐标 = 屏幕中心 Y + 便笺高度/2
	panelBottomY := screenCenterY + helpPanel.Height/2.0
	buttonX := screenCenterX - buttonWidth/2.0
	buttonY := panelBottomY + 20.0 // 便笺下方 20 像素

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
//   - 合成便笺背景
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

	// 1. 合成便笺背景
	if m.bgMask != nil {
		m.backgroundImage = utils.ApplyAlphaMask(m.bgJPG, m.bgMask)
		log.Printf("[HelpPanelModule] Applied alpha mask to background")
	} else {
		m.backgroundImage = m.bgJPG
		log.Printf("[HelpPanelModule] Using original background (no mask)")
	}

	// 2. 处理帮助文本：用亮度作为 Alpha + 反转颜色
	// 原图：黑底白字 → 目标：透明底黑字
	m.helpTextImage = m.convertWhiteTextToBlack(m.textPNG)
	log.Printf("[HelpPanelModule] Converted help text (white on black → black on transparent)")

	// 3. 更新 HelpPanelComponent 的图片引用
	helpPanel, ok := ecs.GetComponent[*components.HelpPanelComponent](m.entityManager, m.helpPanelEntity)
	if ok {
		helpPanel.BackgroundImage = m.backgroundImage
		helpPanel.HelpTextImage = m.helpTextImage
	}

	// 4. 标记为已合成
	m.composited = true
	log.Printf("[HelpPanelModule] Image composition completed")
}

// convertWhiteTextToBlack 将黑底白字转换为透明底黑字
//
// 处理流程：
//   1. 计算每个像素的亮度（作为 Alpha）
//   2. 应用阈值：暗色区域（亮度 < 阈值）→ 完全透明
//   3. 反转 RGB（白色 → 黑色）
//   4. 背景（黑色，低亮度）变透明，文字（白色，高亮度）变不透明
//
// 参数：
//   - src: 原图（黑底白字）
//
// 返回：
//   - 处理后的图片（透明底黑字）
func (m *HelpPanelModule) convertWhiteTextToBlack(src *ebiten.Image) *ebiten.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建新图片
	result := ebiten.NewImage(width, height)

	// 读取源图像素
	srcPixels := make([]byte, width*height*4)
	src.ReadPixels(srcPixels)

	// 处理每个像素
	resultPixels := make([]byte, width*height*4)
	const brightnessThreshold = 30 // 亮度阈值：低于此值的像素被视为背景（完全透明）

	for i := 0; i < len(srcPixels); i += 4 {
		r := srcPixels[i+0]
		g := srcPixels[i+1]
		b := srcPixels[i+2]

		// 计算亮度（灰度值）
		brightness := (uint16(r) + uint16(g) + uint16(b)) / 3

		// 应用阈值：暗色区域（背景）设为完全透明
		var alpha byte
		if brightness < brightnessThreshold {
			alpha = 0 // 完全透明（背景）
		} else {
			alpha = byte(brightness) // 使用亮度作为 Alpha（文字）
		}

		// 反转 RGB（白色文字变成黑色）
		resultPixels[i+0] = 255 - r // R
		resultPixels[i+1] = 255 - g // G
		resultPixels[i+2] = 255 - b // B
		resultPixels[i+3] = alpha   // A（亮度，应用阈值）
	}

	// 写入结果
	result.WritePixels(resultPixels)
	return result
}

// Draw 渲染帮助面板到屏幕
//
// 参数:
//   - screen: 目标渲染屏幕
//
// 渲染顺序：
//  1. 半透明遮罩（覆盖整个屏幕）
//  2. 便笺背景（居中）
//  3. 帮助文本（叠加在便笺上）
//  4. "确定"按钮（在便笺下方）
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

	// 1. 绘制半透明遮罩
	m.drawOverlay(screen)

	// 2. 计算居中位置
	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0

	panelX := screenCenterX - helpPanel.Width/2.0
	panelY := screenCenterY - helpPanel.Height/2.0

	// 3. 绘制便笺背景
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(panelX, panelY)
	screen.DrawImage(helpPanel.BackgroundImage, bgOp)

	// 4. 绘制帮助文本（叠加在便笺上）
	textOp := &ebiten.DrawImageOptions{}
	textOp.GeoM.Translate(panelX, panelY)
	screen.DrawImage(helpPanel.HelpTextImage, textOp)

	// 5. 绘制"确定"按钮
	if m.buttonRenderSystem != nil {
		m.buttonRenderSystem.DrawButton(screen, m.confirmButtonEntity)
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
