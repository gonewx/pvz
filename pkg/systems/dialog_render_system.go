package systems

import (
	"image/color"
	"log"
	"sort"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// DialogRenderSystem 对话框渲染系统
// 负责渲染所有对话框实体
//
// 职责：
//   - 渲染半透明遮罩（覆盖整个屏幕）
//   - 渲染对话框背景（九宫格拉伸）
//   - 渲染对话框装饰（骷髅头）
//   - 渲染对话框标题和消息
//   - 渲染对话框按钮
//   - 渲染对话框的子实体（如输入框） - Story 12.4
type DialogRenderSystem struct {
	entityManager         *ecs.EntityManager
	windowWidth           int
	windowHeight          int
	titleFont             *text.GoTextFace       // 标题字体
	messageFont           *text.GoTextFace       // 消息字体
	buttonFont            *text.GoTextFace       // 按钮字体
	textInputRenderSystem *TextInputRenderSystem // 文本输入框渲染系统（用于渲染子实体）
}

// NewDialogRenderSystem 创建对话框渲染系统
func NewDialogRenderSystem(em *ecs.EntityManager, windowWidth, windowHeight int, titleFont, messageFont, buttonFont *text.GoTextFace) *DialogRenderSystem {
	return &DialogRenderSystem{
		entityManager:         em,
		windowWidth:           windowWidth,
		windowHeight:          windowHeight,
		titleFont:             titleFont,
		messageFont:           messageFont,
		buttonFont:            buttonFont,
		textInputRenderSystem: nil, // 稍后通过 SetTextInputRenderSystem 设置
	}
}

// SetTextInputRenderSystem 设置文本输入框渲染系统
// Story 12.4: 用于渲染对话框的子实体（输入框）
func (s *DialogRenderSystem) SetTextInputRenderSystem(tirs *TextInputRenderSystem) {
	s.textInputRenderSystem = tirs
}

// Draw 渲染所有对话框
// 查询所有拥有 DialogComponent 和 PositionComponent 的实体并渲染
func (s *DialogRenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有对话框实体
	dialogEntities := ecs.GetEntitiesWith2[*components.DialogComponent, *components.PositionComponent](s.entityManager)

	if len(dialogEntities) == 0 {
		return
	}

	log.Printf("[DialogRenderSystem] 正在渲染 %d 个对话框", len(dialogEntities))

	// 绘制半透明遮罩（只需要绘制一次）
	s.drawOverlay(screen)

	// ✅ Story 12.4: 按实体 ID 排序，确保渲染顺序固定（防止闪烁）
	// ID 小的先渲染（在底层），ID 大的后渲染（在上层）
	sort.Slice(dialogEntities, func(i, j int) bool {
		return dialogEntities[i] < dialogEntities[j]
	})

	// 渲染每个对话框
	for _, entityID := range dialogEntities {
		dialogComp, ok := ecs.GetComponent[*components.DialogComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if !dialogComp.IsVisible {
			log.Printf("[DialogRenderSystem] 对话框 %d 不可见，跳过渲染", entityID)
			continue
		}

		log.Printf("[DialogRenderSystem] 渲染对话框: 位置=(%.0f, %.0f), 大小=(%.0f, %.0f), 标题=%s, 消息=%s",
			posComp.X, posComp.Y, dialogComp.Width, dialogComp.Height, dialogComp.Title, dialogComp.Message)

		// 1. 绘制对话框背景（九宫格）
		// 根据 UseBigBottom 选择使用大底部区域或标准底部
		if dialogComp.UseBigBottom {
			utils.RenderNinePatchWithBigBottom(screen, dialogComp.Parts, posComp.X, posComp.Y, dialogComp.Width, dialogComp.Height)
		} else {
			utils.RenderNinePatch(screen, dialogComp.Parts, posComp.X, posComp.Y, dialogComp.Width, dialogComp.Height)
		}

		// 2. 绘制骷髅头装饰（顶部中央）
		s.drawHeader(screen, dialogComp, posComp.X, posComp.Y)

		// 3. 绘制标题文字
		s.drawTitle(screen, dialogComp, posComp.X, posComp.Y)

		// 4. 绘制消息文字
		s.drawMessage(screen, dialogComp, posComp.X, posComp.Y)

		// 4.5. ✅ Story 12.4: 绘制用户列表（如果有 UserListComponent）
		s.drawUserList(screen, entityID, dialogComp, posComp.X, posComp.Y)

		// 5. 绘制按钮
		s.drawButtons(screen, dialogComp, posComp.X, posComp.Y)

		// 6. ✅ Story 12.4: 绘制对话框的子实体（如输入框）
		// 在对话框之后立即渲染其子实体，确保层级正确
		if len(dialogComp.ChildEntities) > 0 {
			log.Printf("[DialogRenderSystem] 对话框有 %d 个子实体，开始渲染", len(dialogComp.ChildEntities))
			s.drawChildEntities(screen, dialogComp)
		} else {
			log.Printf("[DialogRenderSystem] 对话框没有子实体")
		}

		log.Printf("[DialogRenderSystem] 对话框渲染完成")
	}
}

// drawChildEntities 绘制对话框的子实体（如输入框）
// Story 12.4: 确保子实体跟随父对话框的z-order
func (s *DialogRenderSystem) drawChildEntities(screen *ebiten.Image, dialogComp *components.DialogComponent) {
	if s.textInputRenderSystem == nil {
		log.Printf("[DialogRenderSystem] ⚠️ textInputRenderSystem is nil, cannot render child entities")
		return
	}

	log.Printf("[DialogRenderSystem] 开始渲染 %d 个子实体", len(dialogComp.ChildEntities))

	// 遍历所有子实体
	for _, childID := range dialogComp.ChildEntities {
		// 检查是否是文本输入框
		input, ok := ecs.GetComponent[*components.TextInputComponent](s.entityManager, childID)
		if !ok {
			log.Printf("[DialogRenderSystem] 子实体 %d 不是文本输入框", childID)
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, childID)
		if !ok {
			log.Printf("[DialogRenderSystem] 子实体 %d 没有 PositionComponent", childID)
			continue
		}

		log.Printf("[DialogRenderSystem] 渲染输入框子实体 %d at (%.0f, %.0f)", childID, pos.X, pos.Y)

		// 直接调用 TextInputRenderSystem 的绘制方法
		s.textInputRenderSystem.DrawInputBox(screen, input, pos)
	}

	log.Printf("[DialogRenderSystem] 子实体渲染完成")
}

// drawOverlay 绘制半透明遮罩
func (s *DialogRenderSystem) drawOverlay(screen *ebiten.Image) {
	// 创建半透明黑色遮罩
	overlay := ebiten.NewImage(s.windowWidth, s.windowHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 128}) // 50% 透明度
	screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
}

// drawHeader 绘制骷髅头装饰
func (s *DialogRenderSystem) drawHeader(screen *ebiten.Image, dialog *components.DialogComponent, dialogX, dialogY float64) {
	if dialog.Parts == nil || dialog.Parts.Header == nil {
		return
	}

	// 骷髅头装饰居中显示在对话框顶部
	headerBounds := dialog.Parts.Header.Bounds()
	headerWidth := float64(headerBounds.Dx())
	headerHeight := float64(headerBounds.Dy())

	// 居中位置
	headerX := dialogX + dialog.Width/2 - headerWidth/2
	headerY := dialogY - headerHeight/2 // 一半在对话框上方，一半在对话框内

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(headerX, headerY)
	screen.DrawImage(dialog.Parts.Header, op)
}

// drawTitle 绘制标题文字
func (s *DialogRenderSystem) drawTitle(screen *ebiten.Image, dialog *components.DialogComponent, dialogX, dialogY float64) {
	if dialog.Title == "" || s.titleFont == nil {
		return
	}

	// 标题居中显示在对话框顶部
	centerX := dialogX + dialog.Width/2
	centerY := dialogY + 60

	log.Printf("[DialogRenderSystem] 绘制标题: '%s' at (%.0f, %.0f)", dialog.Title, centerX, centerY)

	// 1. 先绘制阴影（黑色，稍微偏移）
	shadowOp := &text.DrawOptions{}
	shadowOp.LayoutOptions.PrimaryAlign = text.AlignCenter
	shadowOp.LayoutOptions.SecondaryAlign = text.AlignCenter
	shadowOp.GeoM.Translate(centerX+2, centerY+2)                // 阴影偏移2像素
	shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 128}) // 半透明黑色
	text.Draw(screen, dialog.Title, s.titleFont, shadowOp)

	// 2. 再绘制主文字（橙黄色 - 和奖励面板标题一样）
	op := &text.DrawOptions{}
	op.LayoutOptions.PrimaryAlign = text.AlignCenter
	op.LayoutOptions.SecondaryAlign = text.AlignCenter
	op.GeoM.Translate(centerX, centerY)
	op.ColorScale.ScaleWithColor(color.RGBA{255, 200, 0, 255}) // 橙黄色
	text.Draw(screen, dialog.Title, s.titleFont, op)
}

// drawMessage 绘制消息文字（支持多行）
func (s *DialogRenderSystem) drawMessage(screen *ebiten.Image, dialog *components.DialogComponent, dialogX, dialogY float64) {
	if dialog.Message == "" || s.messageFont == nil {
		return
	}

	// ✅ Story 12.4: 支持多行文本显示
	// 将长文本按宽度限制自动换行
	const maxLineWidth = 380.0 // 对话框宽度 - 左右边距
	const lineHeight = 25.0    // 行高

	// 对话框布局常量（与 dialog_factory.go 保持一致）
	const titleAreaHeight = 60.0  // 标题区域高度（骷髅头装饰 + 标题文字）
	const buttonAreaHeight = 70.0 // 按钮区域高度

	// 将消息文本按 maxLineWidth 分割成多行
	lines := s.wrapText(dialog.Message, maxLineWidth)

	// 计算总高度（用于垂直居中）
	totalHeight := float64(len(lines)) * lineHeight

	// 计算消息可用区域（在标题和按钮之间）
	messageAreaTop := dialogY + titleAreaHeight
	messageAreaBottom := dialogY + dialog.Height - buttonAreaHeight
	messageAreaHeight := messageAreaBottom - messageAreaTop

	// 在可用区域内垂直居中
	startY := messageAreaTop + (messageAreaHeight-totalHeight)/2

	log.Printf("[DialogRenderSystem] 绘制消息 (%d 行): '%s' at (%.0f, %.0f)", len(lines), dialog.Message, dialogX+dialog.Width/2, startY)

	// 逐行绘制
	for i, line := range lines {
		if line == "" {
			continue
		}

		centerX := dialogX + dialog.Width/2
		lineY := startY + float64(i)*lineHeight

		// 1. 先绘制阴影（黑色，稍微偏移）
		shadowOp := &text.DrawOptions{}
		shadowOp.LayoutOptions.PrimaryAlign = text.AlignCenter
		shadowOp.LayoutOptions.SecondaryAlign = text.AlignStart
		shadowOp.GeoM.Translate(centerX+2, lineY+2)                  // 阴影偏移2像素
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 128}) // 半透明黑色
		text.Draw(screen, line, s.messageFont, shadowOp)

		// 2. 再绘制主文字（橙黄色）
		op := &text.DrawOptions{}
		op.LayoutOptions.PrimaryAlign = text.AlignCenter
		op.LayoutOptions.SecondaryAlign = text.AlignStart
		op.GeoM.Translate(centerX, lineY)
		op.ColorScale.ScaleWithColor(color.RGBA{255, 200, 0, 255}) // 橙黄色
		text.Draw(screen, line, s.messageFont, op)
	}
}

// wrapText 将文本按指定宽度分割成多行
// Story 12.4: 自动换行支持
func (s *DialogRenderSystem) wrapText(textStr string, maxWidth float64) []string {
	if s.messageFont == nil {
		return []string{textStr}
	}

	words := []rune(textStr)
	var lines []string
	var currentLine []rune

	for _, ch := range words {
		// 尝试添加字符到当前行
		testLine := append(currentLine, ch)
		testLineStr := string(testLine)

		// 测量宽度
		width, _ := text.Measure(testLineStr, s.messageFont, 0)

		if width > maxWidth && len(currentLine) > 0 {
			// 当前行已满，保存并开始新行
			lines = append(lines, string(currentLine))
			currentLine = []rune{ch}
		} else {
			currentLine = testLine
		}
	}

	// 添加最后一行
	if len(currentLine) > 0 {
		lines = append(lines, string(currentLine))
	}

	return lines
}

// drawButtons 绘制按钮
func (s *DialogRenderSystem) drawButtons(screen *ebiten.Image, dialog *components.DialogComponent, dialogX, dialogY float64) {
	for i, btn := range dialog.Buttons {
		// 按钮绝对位置
		btnX := dialogX + btn.X
		btnY := dialogY + btn.Y

		log.Printf("[DialogRenderSystem] 绘制按钮: '%s' at (%.0f, %.0f), 大小=(%.0f, %.0f)",
			btn.Label, btnX, btnY, btn.Width, btn.Height)

		// ✅ 检查是否是按下的按钮（下陷效果）
		isPressed := dialog.PressedButtonIdx == i
		pressOffsetY := 0.0
		if isPressed {
			pressOffsetY = 2.0 // 按钮按下时向下偏移 2 像素
		}

		// 如果有三段式按钮图片，使用三段式渲染
		if btn.LeftImage != nil && btn.MiddleImage != nil && btn.RightImage != nil {
			s.drawNineSliceButton(screen, &btn, btnX, btnY+pressOffsetY)
		} else {
			// 降级：使用纯色矩形
			s.drawFallbackButton(screen, &btn, btnX, btnY+pressOffsetY)
		}

		// 绘制按钮文字（居中，使用游戏内菜单面板的按钮样式）
		if s.buttonFont != nil {
			centerX := btnX + btn.Width/2
			centerY := btnY + btn.Height/2 + pressOffsetY // ✅ 文字也跟随按钮下陷

			log.Printf("[DialogRenderSystem] 按钮文字位置: (%.0f, %.0f)", centerX, centerY)

			// 阴影偏移量
			shadowOffsetX := 2.0
			shadowOffsetY := 2.0

			// 为了让"文字+阴影"整体看起来垂直居中，将主文字向上偏移阴影的一半
			visualCenterOffsetY := -shadowOffsetY / 2.0

			// 1. 先绘制阴影（半透明黑色，偏移位置）
			shadowOp := &text.DrawOptions{}
			shadowOp.LayoutOptions.PrimaryAlign = text.AlignCenter
			shadowOp.LayoutOptions.SecondaryAlign = text.AlignCenter
			shadowOp.GeoM.Translate(centerX+shadowOffsetX, centerY+shadowOffsetY+visualCenterOffsetY)
			shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 180}) // 半透明黑色阴影（游戏内菜单样式）
			text.Draw(screen, btn.Label, s.buttonFont, shadowOp)

			// 2. 再绘制主文字（绿色 - 游戏内菜单按钮样式）
			op := &text.DrawOptions{}
			op.LayoutOptions.PrimaryAlign = text.AlignCenter
			op.LayoutOptions.SecondaryAlign = text.AlignCenter
			op.GeoM.Translate(centerX, centerY+visualCenterOffsetY)
			op.ColorScale.ScaleWithColor(color.RGBA{0, 200, 0, 255}) // 绿色文字（游戏内菜单按钮样式）
			text.Draw(screen, btn.Label, s.buttonFont, op)
		}
	}
}

// drawNineSliceButton 绘制三段式按钮（左、中、右）
func (s *DialogRenderSystem) drawNineSliceButton(screen *ebiten.Image, btn *components.DialogButton, x, y float64) {
	leftWidth := float64(btn.LeftImage.Bounds().Dx())
	middleWidth := btn.MiddleWidth

	// 绘制左边缘
	leftOp := &ebiten.DrawImageOptions{}
	leftOp.GeoM.Translate(x, y)
	screen.DrawImage(btn.LeftImage, leftOp)

	// 绘制中间（拉伸）
	middleOp := &ebiten.DrawImageOptions{}
	middleOp.GeoM.Scale(middleWidth/float64(btn.MiddleImage.Bounds().Dx()), 1.0)
	middleOp.GeoM.Translate(x+leftWidth, y)
	screen.DrawImage(btn.MiddleImage, middleOp)

	// 绘制右边缘
	rightOp := &ebiten.DrawImageOptions{}
	rightOp.GeoM.Translate(x+leftWidth+middleWidth, y)
	screen.DrawImage(btn.RightImage, rightOp)
}

// drawFallbackButton 绘制降级按钮（纯色矩形）
func (s *DialogRenderSystem) drawFallbackButton(screen *ebiten.Image, btn *components.DialogButton, x, y float64) {
	// 绘制按钮背景（灰色矩形）
	btnImage := ebiten.NewImage(int(btn.Width), int(btn.Height))
	btnImage.Fill(color.RGBA{80, 80, 80, 255}) // 深灰色背景

	btnOp := &ebiten.DrawImageOptions{}
	btnOp.GeoM.Translate(x, y)
	screen.DrawImage(btnImage, btnOp)

	// 绘制按钮边框（增强可见性）
	// 上边框
	topBorder := ebiten.NewImage(int(btn.Width), 2)
	topBorder.Fill(color.RGBA{200, 200, 200, 255})
	topOp := &ebiten.DrawImageOptions{}
	topOp.GeoM.Translate(x, y)
	screen.DrawImage(topBorder, topOp)

	// 下边框
	bottomBorder := ebiten.NewImage(int(btn.Width), 2)
	bottomBorder.Fill(color.RGBA{200, 200, 200, 255})
	bottomOp := &ebiten.DrawImageOptions{}
	bottomOp.GeoM.Translate(x, y+btn.Height-2)
	screen.DrawImage(bottomBorder, bottomOp)

	// 左边框
	leftBorder := ebiten.NewImage(2, int(btn.Height))
	leftBorder.Fill(color.RGBA{200, 200, 200, 255})
	leftOp := &ebiten.DrawImageOptions{}
	leftOp.GeoM.Translate(x, y)
	screen.DrawImage(leftBorder, leftOp)

	// 右边框
	rightBorder := ebiten.NewImage(2, int(btn.Height))
	rightBorder.Fill(color.RGBA{200, 200, 200, 255})
	rightOp := &ebiten.DrawImageOptions{}
	rightOp.GeoM.Translate(x+btn.Width-2, y)
	screen.DrawImage(rightBorder, rightOp)
}

// drawUserList 绘制用户列表（Story 12.4）
// 如果对话框有 UserListComponent，则绘制用户列表，否则跳过
func (s *DialogRenderSystem) drawUserList(screen *ebiten.Image, entityID ecs.EntityID, dialog *components.DialogComponent, dialogX, dialogY float64) {
	// 检查是否有 UserListComponent
	userList, ok := ecs.GetComponent[*components.UserListComponent](s.entityManager, entityID)
	if !ok {
		// 没有用户列表组件，跳过
		return
	}

	log.Printf("[DialogRenderSystem] 绘制用户列表: %d 个用户", len(userList.Users))

	// 列表区域配置（使用统一常量）
	const listBgPadding = 2.0  // 列表背景的内边距
	const listBgHeight = 200.0 // 列表背景固定高度
	listX := dialogX + components.UserListPadding
	listWidth := dialog.Width - components.UserListPadding*2
	itemHeight := userList.ItemHeight

	// 绘制列表区域背景（使用黑色填充，固定高度）
	bgWidth := listWidth + listBgPadding*2
	bgHeight := listBgHeight
	bgX := listX - listBgPadding
	bgY := dialogY + components.UserListStartY - listBgPadding

	// 创建黑色背景
	bgCanvas := ebiten.NewImage(int(bgWidth), int(bgHeight))
	bgCanvas.Fill(color.RGBA{0, 0, 0, 255}) // 黑色

	// 绘制背景到屏幕
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(bgX, bgY)
	screen.DrawImage(bgCanvas, bgOp)

	log.Printf("[DialogRenderSystem] 绘制列表背景（黑色，固定高度）: (%.0f, %.0f), 大小=(%.0f, %.0f)", bgX, bgY, bgWidth, bgHeight)

	// 绘制用户列表项
	for i, user := range userList.Users {
		itemY := dialogY + components.UserListStartY + float64(i)*itemHeight

		// 判断是否是选中的用户（使用 SelectedIndex 而不是 CurrentUser）
		isSelected := (i == userList.SelectedIndex)

		// 判断是否是悬停的用户
		isHovered := (i == userList.HoveredIndex)

		// 只有选中的用户项绘制背景（绿色）
		if isSelected {
			bgImage := ebiten.NewImage(int(listWidth), int(itemHeight)-2)
			bgImage.Fill(color.RGBA{0, 150, 0, 200}) // 绿色
			bgOp := &ebiten.DrawImageOptions{}
			bgOp.GeoM.Translate(listX, itemY)
			screen.DrawImage(bgImage, bgOp)
		}

		// 绘制用户名文字（水平和垂直居中）
		textX := listX + listWidth/2  // 水平居中
		textY := itemY + itemHeight/2 // 垂直居中
		textOp := &text.DrawOptions{}
		textOp.GeoM.Translate(textX, textY)
		textOp.LayoutOptions.PrimaryAlign = text.AlignCenter   // 水平居中
		textOp.LayoutOptions.SecondaryAlign = text.AlignCenter // 垂直居中

		// 根据悬停状态设置文字颜色：悬停时白色，正常时黄色
		if isHovered {
			textOp.ColorScale.ScaleWithColor(color.White) // 悬停时白色
		} else {
			textOp.ColorScale.ScaleWithColor(color.RGBA{255, 200, 0, 255}) // 正常时黄色
		}

		if s.messageFont != nil {
			text.Draw(screen, user.Username, s.messageFont, textOp)
		}
	}

	// 绘制"建立一位新用户"选项（列表末尾）
	newUserIndex := len(userList.Users)
	newUserItemY := dialogY + components.UserListStartY + float64(newUserIndex)*itemHeight

	// 判断是否悬停
	isNewUserHovered := (newUserIndex == userList.HoveredIndex)

	// 如果选中了"建立一位新用户"项，绘制背景
	if userList.SelectedIndex == newUserIndex {
		bgImage := ebiten.NewImage(int(listWidth), int(itemHeight)-2)
		bgImage.Fill(color.RGBA{0, 150, 0, 200}) // 绿色
		bgOp := &ebiten.DrawImageOptions{}
		bgOp.GeoM.Translate(listX, newUserItemY)
		screen.DrawImage(bgImage, bgOp)
	}

	// 绘制文字"建立一位新用户"（水平和垂直居中）
	newUserTextX := listX + listWidth/2         // 水平居中
	newUserTextY := newUserItemY + itemHeight/2 // 垂直居中
	newUserTextOp := &text.DrawOptions{}
	newUserTextOp.GeoM.Translate(newUserTextX, newUserTextY)
	newUserTextOp.LayoutOptions.PrimaryAlign = text.AlignCenter   // 水平居中
	newUserTextOp.LayoutOptions.SecondaryAlign = text.AlignCenter // 垂直居中

	// 根据悬停状态设置文字颜色：悬停时白色，正常时黄色
	if isNewUserHovered {
		newUserTextOp.ColorScale.ScaleWithColor(color.White) // 悬停时白色
	} else {
		newUserTextOp.ColorScale.ScaleWithColor(color.RGBA{255, 200, 0, 255}) // 正常时黄色
	}

	if s.messageFont != nil {
		text.Draw(screen, "（建立一位新用户）", s.messageFont, newUserTextOp)
	}

	log.Printf("[DialogRenderSystem] 用户列表渲染完成")
}
