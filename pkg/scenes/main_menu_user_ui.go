package scenes

import (
	"image/color"
	"log"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// showNewUserDialogForFirstLaunch 显示首次启动的新建用户对话框
//
// Story 12.4: 首次启动用户创建流程
//
// 当游戏首次启动（无任何用户）时，自动弹出新建用户对话框
// 用户必须创建用户才能继续游戏（不可跳过）
func (m *MainMenuScene) showNewUserDialogForFirstLaunch() {
	log.Printf("[MainMenuScene] Showing new user dialog for first launch")

	// 创建新建用户对话框
	dialogID, inputBoxID, err := entities.NewNewUserDialogEntity(
		m.entityManager,
		m.resourceManager,
		WindowWidth,
		WindowHeight,
		func(result entities.NewUserDialogResult) {
			if result.Confirmed {
				// 用户点击"好"按钮（无论用户名是否为空）
				// onNewUserCreated 内部会验证用户名
				// 验证失败时会显示错误对话框，但不关闭新用户对话框
				m.onNewUserCreated(result.Username)
			} else {
				// 用户点击"取消"按钮
				// 首次启动不允许取消，显示错误提示对话框
				log.Printf("[MainMenuScene] First launch: cannot cancel user creation, showing error dialog")
				m.showErrorDialog("输入你的名字", "请输入你的名字，以创建新的用户档案。档案用于保存游戏积分和进度。")
			}
		},
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create new user dialog: %v", err)
		return
	}

	m.currentUserDialogID = dialogID
	m.currentInputBoxID = inputBoxID
	m.currentDialog = dialogID // 设置 currentDialog 以触发背景交互阻止
	log.Printf("[MainMenuScene] New user dialog created (entity ID: %d)", dialogID)

	// Story 21.4: 移动端显示虚拟键盘
	if utils.IsMobile() && m.virtualKeyboardSystem != nil {
		m.virtualKeyboardSystem.ShowKeyboard(inputBoxID)
	}
}

// onNewUserCreated 处理新用户创建成功的回调
func (m *MainMenuScene) onNewUserCreated(username string) {
	log.Printf("[MainMenuScene] Creating new user: %s", username)

	gameState := getGameState()
	saveManager := gameState.GetSaveManager()

	// 验证用户名
	if err := saveManager.ValidateUsername(username); err != nil {
		log.Printf("[MainMenuScene] Invalid username: %v", err)
		// 显示错误提示对话框
		m.showErrorDialog("无效的用户名", err.Error())
		return
	}

	// 创建用户
	if err := saveManager.CreateUser(username); err != nil {
		log.Printf("[MainMenuScene] Failed to create user: %v", err)
		m.showErrorDialog("创建用户失败", err.Error())
		return
	}

	log.Printf("[MainMenuScene] User created successfully: %s", username)

	// 关闭对话框
	m.closeCurrentDialog()

	// 重新加载存档数据
	if err := saveManager.Load(); err == nil {
		m.currentLevel = saveManager.GetHighestLevel()
		if m.currentLevel == "" {
			m.currentLevel = "1-1"
		}
		m.hasStartedGame = saveManager.GetHasStartedGame()
	}

	// ✅ 修复：先记录是否首次启动，然后立即设置为 false
	// 这样 updateButtonVisibility() 就不会保留首次启动的隐藏轨道
	wasFirstLaunch := m.isFirstLaunch
	m.isFirstLaunch = false

	// Story 12.4 AC8: 创建成功后，首先取消隐藏木牌和草叶子轨道
	if wasFirstLaunch && m.selectorScreenEntity != 0 {
		// 首次启动时，取消隐藏木牌和草叶子轨道
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
		if ok && reanimComp.HiddenTracks != nil {
			// 取消隐藏木牌轨道
			delete(reanimComp.HiddenTracks, "woodsign1")
			delete(reanimComp.HiddenTracks, "woodsign2")
			delete(reanimComp.HiddenTracks, "woodsign3")
			// 取消隐藏草叶子轨道
			delete(reanimComp.HiddenTracks, "leaf1")
			delete(reanimComp.HiddenTracks, "leaf2")
			delete(reanimComp.HiddenTracks, "leaf3")
			delete(reanimComp.HiddenTracks, "leaf4")
			delete(reanimComp.HiddenTracks, "leaf5")
			delete(reanimComp.HiddenTracks, "leaf22")
			delete(reanimComp.HiddenTracks, "leaf_SelectorScreen_Leaves")
			log.Printf("[MainMenuScene] First launch: unhidden woodsign and leaf tracks")

			// ✅ 设置动画循环状态
			if reanimComp.AnimationLoopStates == nil {
				reanimComp.AnimationLoopStates = make(map[string]bool)
			}
			reanimComp.AnimationLoopStates["anim_sign"] = false // 木牌动画非循环
			reanimComp.AnimationLoopStates["anim_grass"] = true // 草动画循环
		}

		// ✅ 修复：直接调用 AddAnimation() 添加到现有动画列表
		// 此时应该已经有：anim_open（背景）、anim_idle（按钮）、云朵动画
		// 现在添加：anim_sign（木牌）、anim_grass（草）
		if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_sign"); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to add anim_sign: %v", err)
		}
		if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_grass"); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to add anim_grass: %v", err)
		}
		log.Printf("[MainMenuScene] First launch: added anim_sign + anim_grass to existing animations")
	}

	// ✅ 修复：在取消隐藏轨道后再更新按钮可见性
	// 这样 updateButtonVisibility() 就不会重新隐藏 woodsign2
	m.updateButtonVisibility()

	// Story 12.4: 初始化木牌（显示用户名）
	m.initUserSign()

	log.Printf("[MainMenuScene] First launch setup completed")
}

// closeCurrentDialog 关闭当前打开的对话框
func (m *MainMenuScene) closeCurrentDialog() {
	if m.currentUserDialogID != 0 {
		m.entityManager.DestroyEntity(m.currentUserDialogID)
		m.currentUserDialogID = 0
	}
	if m.currentInputBoxID != 0 {
		m.entityManager.DestroyEntity(m.currentInputBoxID)
		m.currentInputBoxID = 0
	}
	// 清除 currentDialog 以允许背景交互
	m.currentDialog = 0

	// Story 21.4: 隐藏虚拟键盘
	if m.virtualKeyboardSystem != nil {
		m.virtualKeyboardSystem.HideKeyboard()
	}
}

// initUserSign 初始化木牌UI实体（显示用户名）
// Story 12.4 Task 2.2
func (m *MainMenuScene) initUserSign() {
	// 获取当前用户名
	currentUser := m.saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[MainMenuScene] Warning: No current user, skipping user sign initialization")
		return
	}

	// 加载木牌按下状态图片
	signPressImage, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_WOODSIGN2_PRESS")
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load sign press image: %v", err)
		signPressImage = nil
	}

	// Story 12.4 新方案：将用户名预先绘制到木牌图片上
	// 这样用户名会自然跟随木牌动画，不需要单独处理动画同步
	if m.selectorScreenEntity != 0 {
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
		if ok {
			// 加载原始木牌图片
			originalSignImage, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_WOODSIGN1")
			if err != nil {
				log.Printf("[MainMenuScene] Warning: Failed to load woodsign1 image: %v", err)
				return
			}

			// 创建新图片，将用户名绘制在木牌上
			signWithText := m.createSignWithUsername(originalSignImage, currentUser)
			if signWithText != nil {
				// 替换 PartImages 中的木牌图片
				reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_WOODSIGN1"] = signWithText
				log.Printf("[MainMenuScene] Replaced woodsign1 image with username: %s", currentUser)
			}
		}

		// 添加 UserSignComponent（用于悬停和点击检测）
		ecs.AddComponent(m.entityManager, m.selectorScreenEntity, &components.UserSignComponent{
			CurrentUsername: currentUser,
			IsHovered:       false,
			SignPressImage:  signPressImage,
		})
		m.userSignEntity = m.selectorScreenEntity
		log.Printf("[MainMenuScene] User sign initialized for user: %s", currentUser)
	} else {
		log.Printf("[MainMenuScene] Warning: SelectorScreen entity not found, cannot initialize user sign")
	}
}

// createSignWithUsername 创建带用户名的木牌图片
// 在原始木牌图片上绘制用户名文本（白色泛黄，无描边，26号字体）
func (m *MainMenuScene) createSignWithUsername(originalImage *ebiten.Image, username string) *ebiten.Image {
	if originalImage == nil {
		return nil
	}

	// 获取原始图片尺寸
	bounds := originalImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建新图片
	newImage := ebiten.NewImage(width, height)

	// 先绘制原始木牌图片
	newImage.DrawImage(originalImage, nil)

	// 加载字体
	usernameFont, err := m.resourceManager.LoadFont("assets/fonts/fzse_gbk.ttf", 26)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load username font: %v", err)
		return originalImage
	}

	// 计算用户名位置（木牌中下部分，居中，70% 高度）
	centerX := float64(width) * 0.5
	centerY := float64(height) * 0.60

	// 绘制白色泛黄文本（无描边）
	yellowishWhiteColor := color.RGBA{R: 255, G: 255, B: 200, A: 255}
	drawCenteredTextOnImage(newImage, username, centerX, centerY, usernameFont, yellowishWhiteColor)

	return newImage
}

// drawCenteredTextOnImage 在图片上居中绘制文本
func drawCenteredTextOnImage(img *ebiten.Image, textStr string, centerX, centerY float64, fontFace *text.GoTextFace, clr color.Color) {
	textWidth, _ := text.Measure(textStr, fontFace, 0)
	x := centerX - textWidth/2
	y := centerY

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(img, textStr, fontFace, op)
}

// updateUserSignHover 更新木牌悬停状态和点击检测
// Story 12.4 Task 2.3
func (m *MainMenuScene) updateUserSignHover(mouseX, mouseY int, isMouseReleased bool) {
	// 如果没有木牌实体，跳过
	if m.userSignEntity == 0 {
		return
	}

	// 获取 UserSignComponent
	userSignComp, ok := ecs.GetComponent[*components.UserSignComponent](m.entityManager, m.userSignEntity)
	if !ok {
		return
	}

	// 获取 ReanimComponent 以获取木牌轨道的位置
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.userSignEntity)
	if !ok {
		return
	}

	// Story 12.4 AC2: woodsign2 是 "如果这不是你的存档，请点我" 的木板
	signTrackName := "woodsign2"

	// 检查轨道是否被隐藏
	if reanimComp.HiddenTracks != nil && reanimComp.HiddenTracks[signTrackName] {
		userSignComp.IsHovered = false
		return
	}

	// 获取轨道的当前帧数据
	frames, trackExists := reanimComp.MergedTracks[signTrackName]
	if !trackExists || len(frames) == 0 {
		userSignComp.IsHovered = false
		return
	}

	// ✅ 修复：使用与渲染系统相同的逻辑来获取当前帧
	// 遍历所有动画，找到最后一个有效的 woodsign2 数据
	var selectedFrame *reanim.Frame
	for _, animName := range reanimComp.CurrentAnimations {
		// 获取该动画的当前逻辑帧（支持独立帧索引）
		var logicalFrame float64
		if reanimComp.AnimationFrameIndices != nil {
			if frame, exists := reanimComp.AnimationFrameIndices[animName]; exists {
				logicalFrame = frame
			} else {
				logicalFrame = float64(reanimComp.CurrentFrame)
			}
		} else {
			logicalFrame = float64(reanimComp.CurrentFrame)
		}

		// 获取动画的可见性数组
		animVisibles, ok := reanimComp.AnimVisiblesMap[animName]
		if !ok {
			continue
		}

		// 映射逻辑帧到物理帧
		physicalFrame := systems.MapLogicalToPhysical(int(logicalFrame), animVisibles)
		if physicalFrame < 0 || physicalFrame >= len(frames) {
			continue
		}

		// 检查动画定义轨道是否可见（f != -1）
		animDefTrack, ok := reanimComp.MergedTracks[animName]
		if !ok || physicalFrame >= len(animDefTrack) {
			continue
		}

		defFrame := animDefTrack[physicalFrame]
		if defFrame.FrameNum != nil && *defFrame.FrameNum == -1 {
			// 动画隐藏，跳过
			continue
		}

		// 获取该帧的数据（后面的动画会覆盖前面的）
		selectedFrame = &frames[physicalFrame]
	}

	// 如果没有找到有效的帧数据，跳过
	if selectedFrame == nil {
		userSignComp.IsHovered = false
		return
	}

	// 获取当前帧的变换数据
	frame := *selectedFrame

	// 获取 PositionComponent 的基础位置
	posComp, hasPosComp := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.userSignEntity)
	baseX := 0.0
	baseY := 0.0
	if hasPosComp {
		baseX = posComp.X
		baseY = posComp.Y
	}

	// 计算木牌的屏幕位置（左上角）
	frameX := 0.0
	frameY := 0.0
	if frame.X != nil {
		frameX = *frame.X
	}
	if frame.Y != nil {
		frameY = *frame.Y
	}

	signX := baseX + frameX - reanimComp.CenterOffsetX
	signY := baseY + frameY - reanimComp.CenterOffsetY

	// 从 PartImages 获取木牌图片以确定尺寸
	signImage, hasImage := reanimComp.PartImages[frame.ImagePath]
	if !hasImage || signImage == nil {
		userSignComp.IsHovered = false
		return
	}

	bounds := signImage.Bounds()
	signWidth := float64(bounds.Dx())
	signHeight := float64(bounds.Dy())

	// Story 12.4 AC2: woodsign2 木板的点击检测区域
	// "如果这不是你的存档，请点我" 整个木板都可点击
	clickableTop := signY + signHeight*0.1    // 木板顶部预留 10% 边距
	clickableBottom := signY + signHeight*0.9 // 木板底部预留 10% 边距
	clickableLeft := signX + signWidth*0.05   // 木板左侧预留 5% 边距
	clickableRight := signX + signWidth*0.95  // 木板右侧预留 5% 边距

	// 检查鼠标是否在可点击区域内
	mouseInSign := float64(mouseX) >= clickableLeft &&
		float64(mouseX) <= clickableRight &&
		float64(mouseY) >= clickableTop &&
		float64(mouseY) <= clickableBottom

	// 更新悬停状态，并动态替换木牌图片
	if userSignComp.IsHovered != mouseInSign {
		userSignComp.IsHovered = mouseInSign

		// Story 10.9: 悬停时播放音效
		if mouseInSign {
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_BLEEP")
			}
		}

		// Story 12.4 AC2: 悬停时切换 woodsign2 为 SignPressImage
		if mouseInSign && userSignComp.SignPressImage != nil {
			// 直接使用按下状态图片（不需要绘制用户名，woodsign2 是纯木板）
			reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_WOODSIGN2"] = userSignComp.SignPressImage
			log.Printf("[MainMenuScene] User sign (woodsign2) hovered, switched to press image")
		} else {
			// 恢复正常状态木牌图片
			originalSignImage, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_WOODSIGN2")
			if err == nil {
				reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_WOODSIGN2"] = originalSignImage
				log.Printf("[MainMenuScene] User sign (woodsign2) unhovered, switched to normal image")
			}
		}
	}

	// 如果点击木牌，打开用户管理对话框
	if mouseInSign && isMouseReleased {
		// Story 10.9: 点击时播放音效
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_TAP")
		}
		log.Printf("[MainMenuScene] User sign clicked, showing user management dialog")
		m.showUserManagementDialog()
	}
}

// showUserManagementDialog 显示用户管理对话框
// Story 12.4 AC3, AC4
func (m *MainMenuScene) showUserManagementDialog() {
	// 如果已有对话框打开，先关闭
	if m.currentUserDialogID != 0 {
		m.closeCurrentDialog()
	}

	// 获取用户列表
	users, err := m.saveManager.LoadUserList()
	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to load user list: %v", err)
		m.showErrorDialog("加载失败", "无法加载用户列表")
		return
	}

	// 获取当前用户
	currentUser := m.saveManager.GetCurrentUser()

	// 创建用户管理对话框
	dialogID, err := entities.NewUserManagementDialogEntity(
		m.entityManager,
		m.resourceManager,
		users,
		currentUser,
		WindowWidth,
		WindowHeight,
		m.onUserManagementAction,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create user management dialog: %v", err)
		return
	}

	m.currentUserDialogID = dialogID
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] User management dialog opened (currentUser=%s)", currentUser)
}

// onUserManagementAction 用户管理对话框的操作回调
// Story 12.4 AC4, AC9
func (m *MainMenuScene) onUserManagementAction(result entities.UserManagementDialogResult) {
	// 从 UserListComponent 读取选中的用户
	var selectedUser string
	var isNewUserSelected bool

	if m.currentUserDialogID != 0 {
		userList, ok := ecs.GetComponent[*components.UserListComponent](m.entityManager, m.currentUserDialogID)
		if ok {
			selectedUser = userList.GetSelectedUsername()
			isNewUserSelected = userList.IsNewUserSelected()
			log.Printf("[MainMenuScene] UserList: selectedUser=%s, isNewUserSelected=%v", selectedUser, isNewUserSelected)
		}
	}

	switch result.Action {
	case entities.UserActionSwitch:
		// "好"按钮：切换用户或新建用户
		if isNewUserSelected {
			// 点击了"建立一位新用户"，然后点击"好"按钮
			m.closeCurrentDialog()
			m.showNewUserDialog(false) // force=false，可以关闭
		} else if selectedUser != "" {
			// 切换到选中的用户
			currentUser := m.saveManager.GetCurrentUser()
			if selectedUser == currentUser {
				// 选中的是当前用户，直接关闭对话框
				log.Printf("[MainMenuScene] Selected current user, just close dialog")
				m.closeCurrentDialog()
			} else {
				// 切换用户
				if err := m.saveManager.SwitchUser(selectedUser); err != nil {
					log.Printf("[MainMenuScene] Error: Failed to switch user: %v", err)
					m.showErrorDialog("切换失败", "无法切换到用户: "+selectedUser)
					return
				}
				log.Printf("[MainMenuScene] Switched to user: %s", selectedUser)
				// 重新加载主菜单数据
				m.reloadMainMenuData()
				// 关闭对话框
				m.closeCurrentDialog()
			}
		}

	case entities.UserActionCreateNew:
		// 这个 case 已经不需要了，因为"建立一位新用户"现在在列表中，通过 UserActionSwitch 处理
		// 保留以防万一
		m.closeCurrentDialog()
		m.showNewUserDialog(false) // force=false，可以关闭

	case entities.UserActionRename:
		// 显示重命名对话框（不关闭用户管理对话框，直接叠加）
		if selectedUser != "" && !isNewUserSelected {
			m.showRenameUserDialog(selectedUser)
		} else {
			log.Printf("[MainMenuScene] Warning: Cannot rename when no user selected or new user selected")
		}

	case entities.UserActionDelete:
		// 显示删除确认对话框（不关闭用户管理对话框，直接叠加）
		if selectedUser != "" && !isNewUserSelected {
			m.showDeleteUserDialog(selectedUser)
		} else {
			log.Printf("[MainMenuScene] Warning: Cannot delete when no user selected or new user selected")
		}

	case entities.UserActionNone:
		// 取消，关闭对话框
		m.closeCurrentDialog()
	}
}

// reloadMainMenuData 重新加载主菜单数据（用户切换后）
// Story 12.4 Task 8.2
func (m *MainMenuScene) reloadMainMenuData() {
	// 重新加载存档数据
	if err := m.saveManager.Load(); err != nil {
		log.Printf("[MainMenuScene] Error: Failed to load save after user switch: %v", err)
		m.currentLevel = "1-1"
		m.hasStartedGame = false
	} else {
		m.currentLevel = m.saveManager.GetHighestLevel()
		if m.currentLevel == "" {
			m.currentLevel = "1-1"
		}
		m.hasStartedGame = m.saveManager.GetHasStartedGame()
		log.Printf("[MainMenuScene] Reloaded save: level=%s, hasStartedGame=%v", m.currentLevel, m.hasStartedGame)
	}

	// 更新按钮可见性
	m.updateButtonVisibility()

	// 更新木牌显示的用户名（重新生成木牌图片）
	m.initUserSign()
}

// showNewUserDialog 显示新建用户对话框
// Story 12.4 AC5
func (m *MainMenuScene) showNewUserDialog(force bool) {
	// 关闭现有对话框
	if m.currentUserDialogID != 0 {
		m.closeCurrentDialog()
	}

	// 创建新建用户对话框的回调
	callback := func(result entities.NewUserDialogResult) {
		if result.Confirmed {
			m.onNewUserCreated(result.Username)
		} else if !force {
			// 非强制模式可以取消
			m.closeCurrentDialog()
		}
	}

	// 创建新建用户对话框
	dialogID, inputBoxID, err := entities.NewNewUserDialogEntity(
		m.entityManager,
		m.resourceManager,
		WindowWidth,
		WindowHeight,
		callback,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create new user dialog: %v", err)
		return
	}

	m.currentUserDialogID = dialogID
	m.currentInputBoxID = inputBoxID
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] New user dialog opened (force=%v)", force)

	// Story 21.4: 移动端显示虚拟键盘
	if utils.IsMobile() && m.virtualKeyboardSystem != nil {
		m.virtualKeyboardSystem.ShowKeyboard(inputBoxID)
	}
}

// showRenameUserDialog 显示重命名用户对话框
// Story 12.4 AC6
func (m *MainMenuScene) showRenameUserDialog(oldUsername string) {
	// 用于存储对话框 ID 的变量
	var renameDialogID ecs.EntityID
	var renameInputBoxID ecs.EntityID

	// 创建重命名对话框的回调
	callback := func(result entities.RenameUserDialogResult) {
		if result.Confirmed && result.NewName != "" {
			// 执行重命名
			if err := m.saveManager.RenameUser(oldUsername, result.NewName); err != nil {
				log.Printf("[MainMenuScene] Error: Failed to rename user: %v", err)
				m.showErrorDialog("重命名失败", err.Error())
				return
			}
			log.Printf("[MainMenuScene] User renamed: %s -> %s", oldUsername, result.NewName)
			// 重新加载数据
			m.reloadMainMenuData()
			// 刷新用户管理对话框的列表数据
			m.refreshUserManagementDialog()
		}
		// 无论确认还是取消，都手动销毁重命名对话框
		if renameDialogID != 0 {
			m.entityManager.DestroyEntity(renameDialogID)
			log.Printf("[MainMenuScene] Destroyed rename dialog (ID: %d)", renameDialogID)
		}
		if renameInputBoxID != 0 {
			m.entityManager.DestroyEntity(renameInputBoxID)
			log.Printf("[MainMenuScene] Destroyed rename input box (ID: %d)", renameInputBoxID)
		}
		// 恢复 currentDialog 为用户管理对话框
		m.currentDialog = m.currentUserDialogID
		m.currentInputBoxID = 0
	}

	// 创建重命名对话框
	dialogID, inputBoxID, err := entities.NewRenameUserDialogEntity(
		m.entityManager,
		m.resourceManager,
		oldUsername,
		WindowWidth,
		WindowHeight,
		callback,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create rename user dialog: %v", err)
		return
	}

	// 保存到闭包变量中
	renameDialogID = dialogID
	renameInputBoxID = inputBoxID

	// ✅ 重命名对话框不覆盖 currentUserDialogID
	// 只更新 currentDialog 和 currentInputBoxID
	m.currentInputBoxID = inputBoxID
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] Rename user dialog opened for: %s (dialogID=%d, keeping userDialogID=%d)",
		oldUsername, dialogID, m.currentUserDialogID)

	// Story 21.4: 移动端显示虚拟键盘
	if utils.IsMobile() && m.virtualKeyboardSystem != nil {
		m.virtualKeyboardSystem.ShowKeyboard(inputBoxID)
	}
}

// showDeleteUserDialog 显示删除用户确认对话框
// Story 12.4 AC7
func (m *MainMenuScene) showDeleteUserDialog(username string) {
	// 用于存储对话框 ID 的变量
	var deleteDialogID ecs.EntityID

	// 创建删除确认对话框的回调
	callback := func(result entities.DeleteUserDialogResult) {
		if result.Confirmed {
			// 执行删除
			if err := m.saveManager.DeleteUser(username); err != nil {
				log.Printf("[MainMenuScene] Error: Failed to delete user: %v", err)
				m.showErrorDialog("删除失败", err.Error())
				return
			}
			log.Printf("[MainMenuScene] User deleted: %s", username)

			// 检查是否还有用户
			users, err := m.saveManager.LoadUserList()
			if err != nil || len(users) == 0 {
				// 没有用户了，清空木板显示并进入强制新建用户流程
				m.isFirstLaunch = true
				m.newUserDialogShown = true // 防止 Update 中重复调用 showNewUserDialogForFirstLaunch

				// 清空木板显示：移除 UserSignComponent，恢复原始木牌图片
				if m.userSignEntity != 0 {
					// 移除 UserSignComponent
					if _, ok := ecs.GetComponent[*components.UserSignComponent](m.entityManager, m.userSignEntity); ok {
						ecs.RemoveComponent[*components.UserSignComponent](m.entityManager, m.userSignEntity)
						log.Printf("[MainMenuScene] Removed UserSignComponent from entity %d", m.userSignEntity)
					}

					// 恢复原始木牌图片（不带用户名）
					if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.userSignEntity); ok {
						originalSignImage, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_WOODSIGN1")
						if err == nil {
							reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_WOODSIGN1"] = originalSignImage
							log.Printf("[MainMenuScene] Restored original woodsign1 image (no username)")
						}
					}

					m.userSignEntity = 0
				}

				// 先销毁删除确认对话框
				if deleteDialogID != 0 {
					m.entityManager.DestroyEntity(deleteDialogID)
					log.Printf("[MainMenuScene] Destroyed delete dialog (ID: %d)", deleteDialogID)
				}
				m.currentDialog = m.currentUserDialogID
				// 然后打开新建用户对话框
				m.showNewUserDialogAfterDeleteAll()
				return
			}

			// 重新加载数据
			m.reloadMainMenuData()
			// 刷新用户管理对话框的列表数据
			m.refreshUserManagementDialog()
		}
		// 无论确认还是取消，都手动销毁删除确认对话框
		if deleteDialogID != 0 {
			m.entityManager.DestroyEntity(deleteDialogID)
			log.Printf("[MainMenuScene] Destroyed delete dialog (ID: %d)", deleteDialogID)
		}
		// 恢复 currentDialog 为用户管理对话框
		m.currentDialog = m.currentUserDialogID
	}

	// 创建删除确认对话框
	dialogID, err := entities.NewDeleteUserDialogEntity(
		m.entityManager,
		m.resourceManager,
		username,
		WindowWidth,
		WindowHeight,
		callback,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create delete user dialog: %v", err)
		return
	}

	// 保存到闭包变量中
	deleteDialogID = dialogID

	// ✅ 删除确认对话框不覆盖 currentUserDialogID
	// 只更新 currentDialog
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] Delete user dialog opened for: %s (dialogID=%d, keeping userDialogID=%d)",
		username, dialogID, m.currentUserDialogID)
}

// showNewUserDialogAfterDeleteAll 删除所有用户后显示新建用户对话框
// Story 12.4: 删除最后一个用户后的特殊流程
// 新建用户成功后，关闭两个对话框（新建用户对话框 + 用户管理对话框）
func (m *MainMenuScene) showNewUserDialogAfterDeleteAll() {
	// 保存用户管理对话框 ID，用于后续关闭
	userManagementDialogID := m.currentUserDialogID

	// 用于存储新建用户对话框 ID 的变量（供闭包使用）
	var newUserDialogID ecs.EntityID
	var newUserInputBoxID ecs.EntityID

	// 创建新建用户对话框的回调
	callback := func(result entities.NewUserDialogResult) {
		if result.Confirmed {
			// 用户点击"好"按钮
			m.onNewUserCreatedAfterDeleteAll(result.Username, newUserDialogID, newUserInputBoxID, userManagementDialogID)
		} else {
			// 用户点击"取消"按钮 - 强制创建，显示错误提示
			log.Printf("[MainMenuScene] Cannot cancel: must create a user")
			m.showErrorDialog("输入你的名字", "请输入你的名字，以创建新的用户档案。档案用于保存游戏积分和进度。")
		}
	}

	// 创建新建用户对话框
	dialogID, inputBoxID, err := entities.NewNewUserDialogEntity(
		m.entityManager,
		m.resourceManager,
		WindowWidth,
		WindowHeight,
		callback,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create new user dialog: %v", err)
		return
	}

	// 保存到闭包变量
	newUserDialogID = dialogID
	newUserInputBoxID = inputBoxID

	m.currentInputBoxID = inputBoxID
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] New user dialog opened after deleting all users (entity ID: %d)", dialogID)

	// Story 21.4: 移动端显示虚拟键盘
	if utils.IsMobile() && m.virtualKeyboardSystem != nil {
		m.virtualKeyboardSystem.ShowKeyboard(inputBoxID)
	}
}

// onNewUserCreatedAfterDeleteAll 处理删除所有用户后新建用户的回调
// 需要关闭新建用户对话框和用户管理对话框
func (m *MainMenuScene) onNewUserCreatedAfterDeleteAll(username string, newUserDialogID, newUserInputBoxID, userManagementDialogID ecs.EntityID) {
	log.Printf("[MainMenuScene] Creating new user after delete all: %s", username)

	gameState := getGameState()
	saveManager := gameState.GetSaveManager()

	// 验证用户名
	if err := saveManager.ValidateUsername(username); err != nil {
		log.Printf("[MainMenuScene] Invalid username: %v", err)
		m.showErrorDialog("无效的用户名", err.Error())
		return
	}

	// 创建用户
	if err := saveManager.CreateUser(username); err != nil {
		log.Printf("[MainMenuScene] Failed to create user: %v", err)
		m.showErrorDialog("创建用户失败", err.Error())
		return
	}

	log.Printf("[MainMenuScene] User created successfully: %s", username)

	// 关闭新建用户对话框
	if newUserInputBoxID != 0 {
		m.entityManager.DestroyEntity(newUserInputBoxID)
	}
	if newUserDialogID != 0 {
		m.entityManager.DestroyEntity(newUserDialogID)
	}

	// 关闭用户管理对话框
	if userManagementDialogID != 0 {
		m.entityManager.DestroyEntity(userManagementDialogID)
	}

	// 清理跟踪变量
	m.currentUserDialogID = 0
	m.currentInputBoxID = 0
	m.currentDialog = 0

	// 重新加载存档数据
	if err := saveManager.Load(); err == nil {
		m.currentLevel = saveManager.GetHighestLevel()
		if m.currentLevel == "" {
			m.currentLevel = "1-1"
		}
		m.hasStartedGame = saveManager.GetHasStartedGame()
	}

	// 记录是否首次启动，然后设置为 false
	wasFirstLaunch := m.isFirstLaunch
	m.isFirstLaunch = false

	// 首次启动时，取消隐藏木牌和草叶子轨道
	if wasFirstLaunch && m.selectorScreenEntity != 0 {
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
		if ok && reanimComp.HiddenTracks != nil {
			delete(reanimComp.HiddenTracks, "woodsign1")
			delete(reanimComp.HiddenTracks, "woodsign2")
			delete(reanimComp.HiddenTracks, "woodsign3")
			delete(reanimComp.HiddenTracks, "leaf1")
			delete(reanimComp.HiddenTracks, "leaf2")
			delete(reanimComp.HiddenTracks, "leaf3")
			delete(reanimComp.HiddenTracks, "leaf4")
			delete(reanimComp.HiddenTracks, "leaf5")
			delete(reanimComp.HiddenTracks, "leaf22")
			delete(reanimComp.HiddenTracks, "leaf_SelectorScreen_Leaves")
			log.Printf("[MainMenuScene] First launch: unhidden woodsign and leaf tracks")

			if reanimComp.AnimationLoopStates == nil {
				reanimComp.AnimationLoopStates = make(map[string]bool)
			}
			reanimComp.AnimationLoopStates["anim_sign"] = false
			reanimComp.AnimationLoopStates["anim_grass"] = true
		}

		if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_sign"); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to add anim_sign: %v", err)
		}
		if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_grass"); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to add anim_grass: %v", err)
		}
		log.Printf("[MainMenuScene] First launch: added anim_sign + anim_grass to existing animations")
	}

	m.updateButtonVisibility()
	m.initUserSign()

	log.Printf("[MainMenuScene] New user created after delete all, setup completed")
}

// refreshUserManagementDialog 刷新用户管理对话框的列表数据
// Story 12.4: 重命名/删除后不重新创建对话框，只刷新数据
func (m *MainMenuScene) refreshUserManagementDialog() {
	if m.currentUserDialogID == 0 {
		log.Printf("[MainMenuScene] Warning: No user management dialog to refresh")
		return
	}

	// 获取 UserListComponent
	userList, ok := ecs.GetComponent[*components.UserListComponent](m.entityManager, m.currentUserDialogID)
	if !ok {
		log.Printf("[MainMenuScene] Warning: User management dialog has no UserListComponent")
		return
	}

	// 保存原来的选中索引
	oldSelectedIndex := userList.SelectedIndex
	oldUserCount := len(userList.Users)

	// 重新加载用户列表
	users, err := m.saveManager.LoadUserList()
	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to load user list: %v", err)
		return
	}

	// 更新 UserListComponent 的数据
	userList.Users = make([]components.UserInfo, len(users))
	for i, user := range users {
		userList.Users[i] = components.UserInfo{
			Username:    user.Username,
			CreatedAt:   user.CreatedAt,
			LastLoginAt: user.LastLoginAt,
		}
	}

	// 更新当前用户
	userList.CurrentUser = m.saveManager.GetCurrentUser()

	// ✅ 智能更新选中索引
	// 场景 1: 重命名（用户数量不变） - 保持原索引
	// 场景 2: 删除（用户数量减少） - 调整索引
	if len(users) == oldUserCount {
		// 重命名场景：保持原来的选中索引
		userList.SelectedIndex = oldSelectedIndex
		log.Printf("[MainMenuScene] Refreshed (rename): kept selectedIndex=%d", oldSelectedIndex)
	} else {
		// 删除场景：调整索引
		if oldSelectedIndex >= len(users) {
			// 原索引超出范围，选中最后一个用户
			userList.SelectedIndex = len(users) - 1
			if userList.SelectedIndex < 0 {
				userList.SelectedIndex = 0
			}
			log.Printf("[MainMenuScene] Refreshed (delete): adjusted selectedIndex from %d to %d", oldSelectedIndex, userList.SelectedIndex)
		} else {
			// 原索引仍然有效，保持不变
			userList.SelectedIndex = oldSelectedIndex
			log.Printf("[MainMenuScene] Refreshed (delete): kept selectedIndex=%d", oldSelectedIndex)
		}
	}

	log.Printf("[MainMenuScene] Refreshed user list: %d users, currentUser=%s, selectedIndex=%d",
		len(userList.Users), userList.CurrentUser, userList.SelectedIndex)
}

// renderUserSignText 渲染木牌上的用户名文本
// Story 12.4 Task 2.4
// 新方案：用户名已预先绘制到木牌图片上，这里不需要单独渲染
// 保留此函数用于未来可能的悬停效果（如更换图片）
func (m *MainMenuScene) renderUserSignText(screen *ebiten.Image) {
	// 用户名已预先绘制到木牌图片上（通过 initUserSign），随 Reanim 动画自然移动
	// 此函数暂时为空，保留用于未来扩展
}
