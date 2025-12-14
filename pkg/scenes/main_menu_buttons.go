package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// loadButtonImages loads normal and highlight images for all menu buttons.
//
// This method extracts normal button images from the SelectorScreen ReanimComponent
// and loads the corresponding highlight images from the resource manager.
//
// Story 12.1 Task 5: Button Highlight Effect
func (m *MainMenuScene) loadButtonImages(rm *game.ResourceManager) {
	// Get ReanimComponent from SelectorScreen entity
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
	if !ok || reanimComp == nil {
		log.Printf("[MainMenuScene] Warning: Failed to get ReanimComponent for button image loading")
		return
	}

	// Define button track name to resource ID mappings
	// Note: Track names don't match actual game modes (see menu_config.go for details)
	buttonMappings := map[string]struct {
		normalImageRef      string // Image reference in PartImages (from .reanim file)
		highlightResourceID string // Resource ID for highlight image
	}{
		"SelectorScreen_Adventure_button": {
			normalImageRef:      "IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON",
			highlightResourceID: "IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_HIGHLIGHT",
		},
		"SelectorScreen_StartAdventure_button": {
			normalImageRef:      "IMAGE_REANIM_SELECTORSCREEN_STARTADVENTURE_BUTTON1",
			highlightResourceID: "IMAGE_REANIM_SELECTORSCREEN_STARTADVENTURE_HIGHLIGHT",
		},
		"SelectorScreen_Survival_button": {
			normalImageRef:      "IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON",
			highlightResourceID: "IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_HIGHLIGHT",
		},
		"SelectorScreen_Challenges_button": {
			normalImageRef:      "IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_BUTTON",
			highlightResourceID: "IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_HIGHLIGHT",
		},
		"SelectorScreen_ZenGarden_button": {
			normalImageRef:      "IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_BUTTON",
			highlightResourceID: "IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_HIGHLIGHT",
		},
	}

	// Load images for each button
	for trackName, mapping := range buttonMappings {
		// Get normal image from PartImages (already loaded by ReanimSystem)
		if normalImg, exists := reanimComp.PartImages[mapping.normalImageRef]; exists {
			m.buttonNormalImages[trackName] = normalImg
			log.Printf("[MainMenuScene] Loaded normal image for %s", trackName)
		} else {
			log.Printf("[MainMenuScene] Warning: Normal image not found for %s (ref: %s)", trackName, mapping.normalImageRef)
		}

		// Load highlight image from resource manager
		highlightImg, err := rm.LoadImageByID(mapping.highlightResourceID)
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to load highlight image for %s: %v", trackName, err)
		} else {
			m.buttonHighlightImages[trackName] = highlightImg
			log.Printf("[MainMenuScene] Loaded highlight image for %s", trackName)
		}
	}

	log.Printf("[MainMenuScene] Button image loading complete: %d normal, %d highlight",
		len(m.buttonNormalImages), len(m.buttonHighlightImages))
}

// updateButtonHighlight updates the button appearance based on hover state.
//
// When the mouse hovers over an unlocked button, this method:
// 1. Replaces the button image with its highlight version in the ReanimComponent
// 2. Plays the stone grinding sound effect (SOUND_GRAVEBUTTON) once
//
// When the mouse leaves a button, it restores the normal image.
//
// Story 12.1 Task 5: Button Highlight Effect
func (m *MainMenuScene) updateButtonHighlight() {
	// Get ReanimComponent from SelectorScreen entity
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
	if !ok || reanimComp == nil {
		return
	}

	// Step 1: Restore the previously highlighted button (if any)
	if m.lastHoveredButton != "" && m.lastHoveredButton != m.hoveredButton {
		// Restore the old button to normal
		if normalImg, exists := m.buttonNormalImages[m.lastHoveredButton]; exists {
			// Find the correct image reference for this button and restore it
			var imageRef string
			switch m.lastHoveredButton {
			case "SelectorScreen_Adventure_button":
				imageRef = "IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"
				reanimComp.PartImages[imageRef] = normalImg
			case "SelectorScreen_StartAdventure_button":
				imageRef = "IMAGE_REANIM_SELECTORSCREEN_STARTADVENTURE_BUTTON1"
				reanimComp.PartImages[imageRef] = normalImg
			case "SelectorScreen_Survival_button":
				imageRef = "IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"
				reanimComp.PartImages[imageRef] = normalImg
			case "SelectorScreen_Challenges_button":
				imageRef = "IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_BUTTON"
				reanimComp.PartImages[imageRef] = normalImg
			case "SelectorScreen_ZenGarden_button":
				imageRef = "IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_BUTTON"
				reanimComp.PartImages[imageRef] = normalImg
			}

			// 强制重建渲染缓存（修改 LastRenderFrame 触发缓存失效）
			reanimComp.LastRenderFrame = -1
		}
	}

	// Step 2: Apply highlight to the currently hovered button (if any and unlocked)
	if m.hoveredButton != "" {
		// 检查轨道是否被隐藏（如果被隐藏则不需要高亮）
		if reanimComp.HiddenTracks != nil && reanimComp.HiddenTracks[m.hoveredButton] {
			m.lastHoveredButton = ""
			return
		}

		// Find the button type for unlock check
		var buttonType config.MenuButtonType
		var found bool
		for _, hitbox := range m.buttonHitboxes {
			if hitbox.TrackName == m.hoveredButton {
				buttonType = hitbox.ButtonType
				found = true
				break
			}
		}

		// Only apply highlight to unlocked buttons
		// 未解锁的按钮不高亮（阴影覆盖在上面，高亮也看不到）
		if found && config.IsMenuModeUnlocked(buttonType, m.currentLevel) {
			// Apply highlight image if available
			if highlightImg, exists := m.buttonHighlightImages[m.hoveredButton]; exists {
				// Find the correct image reference for this button and apply highlight
				var imageRef string
				switch m.hoveredButton {
				case "SelectorScreen_Adventure_button":
					imageRef = "IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"
					reanimComp.PartImages[imageRef] = highlightImg
				case "SelectorScreen_StartAdventure_button":
					imageRef = "IMAGE_REANIM_SELECTORSCREEN_STARTADVENTURE_BUTTON1"
					reanimComp.PartImages[imageRef] = highlightImg
				case "SelectorScreen_Survival_button":
					imageRef = "IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"
					reanimComp.PartImages[imageRef] = highlightImg
				case "SelectorScreen_Challenges_button":
					imageRef = "IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_BUTTON"
					reanimComp.PartImages[imageRef] = highlightImg
				case "SelectorScreen_ZenGarden_button":
					imageRef = "IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_BUTTON"
					reanimComp.PartImages[imageRef] = highlightImg
				}

				// 强制重建渲染缓存（修改 LastRenderFrame 触发缓存失效）
				reanimComp.LastRenderFrame = -1
			}

			// Play sound effect once when entering a new button
			if m.lastHoveredButton != m.hoveredButton {
				m.playGraveButtonSound()
			}

			// Update last hovered button
			m.lastHoveredButton = m.hoveredButton
			return
		}
	}

	// Step 3: If no button is hovered (or button is locked), clear last hovered
	m.lastHoveredButton = ""
}

// updateMouseCursor updates the mouse cursor shape based on hover state.
//
// When the mouse hovers over an unlocked button, bottom function button, or panel button,
// the cursor changes to a pointer hand. Otherwise, the cursor is set to the default arrow shape.
//
// Only updates the cursor when the shape actually changes to avoid unnecessary API calls.
//
// Story 12.1 Task 5: Button Highlight Effect
// Story 12.2: 底部功能栏 - 手形光标
// Story 12.3: 面板按钮光标管理
func (m *MainMenuScene) updateMouseCursor() {
	// Default cursor shape
	cursorShape := ebiten.CursorShapeDefault

	// ✅ 核心修复：对话框打开时，只检查对话框的悬停状态，忽略所有底层 UI
	hasOpenDialog := m.currentUserDialogID != 0 || m.currentDialog != 0 || m.currentErrorDialogID != 0

	if !hasOpenDialog {
		// 只有在没有对话框时才检查底层 UI 元素

		// Check if hovering over a grave button
		if m.hoveredButton != "" {
			// ✅ 修复：所有可见的按钮（包括未解锁的）都显示手形鼠标
			// 未解锁的按钮也可以点击，点击后会提示未解锁
			cursorShape = ebiten.CursorShapePointer
		}

		// Check if hovering over a bottom function button
		if m.hoveredBottomButton != components.BottomButtonNone {
			cursorShape = ebiten.CursorShapePointer
		}

		// Story 12.4 AC2: Check if hovering over user sign
		if m.userSignEntity != 0 {
			if userSignComp, ok := ecs.GetComponent[*components.UserSignComponent](m.entityManager, m.userSignEntity); ok {
				if userSignComp.IsHovered {
					cursorShape = ebiten.CursorShapePointer
				}
			}
		}

		// Check if hovering over any panel button (help/options panel)
		panelButtons := ecs.GetEntitiesWith1[*components.ButtonComponent](m.entityManager)
		for _, entityID := range panelButtons {
			button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, entityID)
			if ok && button.State == components.UIHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}

		// Check if hovering over any slider (settings panel)
		sliders := ecs.GetEntitiesWith1[*components.SliderComponent](m.entityManager)
		for _, entityID := range sliders {
			slider, ok := ecs.GetComponent[*components.SliderComponent](m.entityManager, entityID)
			if ok && slider.IsHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}

		// Check if hovering over any checkbox (settings panel)
		checkboxes := ecs.GetEntitiesWith1[*components.CheckboxComponent](m.entityManager)
		for _, entityID := range checkboxes {
			checkbox, ok := ecs.GetComponent[*components.CheckboxComponent](m.entityManager, entityID)
			if ok && checkbox.IsHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}
	}

	// ✅ ECS 架构重构: 只读取组件状态,不进行碰撞检测
	// DialogInputSystem 负责更新 DialogComponent.HoveredButtonIdx 和 UserListComponent.HoveredIndex
	// 这里只根据状态设置光标

	// 检查所有对话框（用户管理对话框、错误对话框、通用对话框）
	dialogIDs := []ecs.EntityID{m.currentUserDialogID, m.currentDialog, m.currentErrorDialogID}
	for _, dialogID := range dialogIDs {
		if dialogID != 0 {
			dialogComp, ok := ecs.GetComponent[*components.DialogComponent](m.entityManager, dialogID)
			if ok && dialogComp.IsVisible {
				// 检查对话框按钮是否悬停（只读取状态）
				if dialogComp.HoveredButtonIdx >= 0 {
					cursorShape = ebiten.CursorShapePointer
					break
				}

				// 检查用户列表是否有悬停项（只读取状态）
				if userList, ok := ecs.GetComponent[*components.UserListComponent](m.entityManager, dialogID); ok {
					if userList.HoveredIndex >= 0 {
						cursorShape = ebiten.CursorShapePointer
						break
					}
				}
			}
		}
	}

	// Only update cursor if shape changed (避免闪烁)
	if cursorShape != m.lastCursorShape {
		ebiten.SetCursorShape(cursorShape)
		m.lastCursorShape = cursorShape
	}
}

// playGraveButtonSound plays the button hover sound effect.
//
// Story 12.1 Task 5: Button Highlight Effect
func (m *MainMenuScene) playGraveButtonSound() {
	// 使用 AudioManager 播放悬停音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_BLEEP")
	}
}

// updateButtonVisibility updates the visibility of SelectorScreen buttons based on unlock status.
// This method controls which buttons are visible in the Reanim animation by setting the HiddenTracks whitelist.
//
// Unlock rules:
//   - Adventure mode: Always visible
//   - Challenges mode: Visible if level >= 3-2
//   - Vasebreaker mode: Visible if level >= 5-10
//   - Survival mode: Visible if level >= 5-10
//
// Story 12.1: Main Menu Tombstone System Enhancement
func (m *MainMenuScene) updateButtonVisibility() {
	if m.selectorScreenEntity == 0 {
		return // SelectorScreen entity not created, skip
	}

	// Get ReanimComponent from SelectorScreen entity
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
	if !ok {
		log.Printf("[MainMenuScene] Warning: SelectorScreen entity has no ReanimComponent")
		return
	}

	// Step 1: Load hidden tracks from config file (static baseline)
	hiddenTracks := make(map[string]bool)

	if configManager := m.resourceManager.GetReanimConfigManager(); configManager != nil {
		unitConfig, err := configManager.GetUnit("selectorscreen")
		if err == nil {
			// Find "opening" combo and load its hidden_tracks
			for _, combo := range unitConfig.AnimationCombos {
				if combo.Name == "opening" {
					for _, track := range combo.HiddenTracks {
						hiddenTracks[track] = true
						log.Printf("[MainMenuScene] Config hidden track: %s", track)
					}
					break
				}
			}
		} else {
			log.Printf("[MainMenuScene] Warning: Failed to load selectorscreen config: %v", err)
		}
	}

	// Step 2: Merge with code logic (dynamic control based on progress)

	// 2.1 Hide adventure mode button based on whether user has started game
	// New user (!hasStartedGame): Hide "Adventure" button, show "Start Adventure" button
	// Has started game (hasStartedGame): Hide "Start Adventure" button, show "Adventure" button
	// Adventure mode is always unlocked, so both buttons hide their shadows
	if !m.hasStartedGame {
		// 新用户：显示 StartAdventure 按钮
		hiddenTracks["SelectorScreen_Adventure_button"] = true
		hiddenTracks["SelectorScreen_Adventure_shadow"] = true
		hiddenTracks["SelectorScreen_StartAdventure_shadow"] = true // ✅ Adventure 总是解锁，隐藏 StartAdventure 阴影
	} else {
		// 已开始游戏：显示 Adventure 按钮
		hiddenTracks["SelectorScreen_StartAdventure_button"] = true
		hiddenTracks["SelectorScreen_StartAdventure_shadow"] = true
		hiddenTracks["SelectorScreen_Adventure_shadow"] = true // ✅ Adventure 总是解锁，隐藏 Adventure 阴影
	}

	// 2.2 Hide/show other mode buttons based on unlock status

	// Challenges mode (unlocked at 3-2)
	// Note: SelectorScreen_Survival_button track corresponds to Challenges mode
	if config.IsMenuModeUnlocked(config.MenuButtonChallenges, m.currentLevel) {
		hiddenTracks["SelectorScreen_Survival_shadow"] = true
	}
	// 未解锁时：不隐藏按钮和阴影（显示墓碑状态）

	// Vasebreaker mode (unlocked at 5-10)
	// Note: SelectorScreen_Challenges_button track corresponds to Vasebreaker mode
	if config.IsMenuModeUnlocked(config.MenuButtonVasebreaker, m.currentLevel) {
		hiddenTracks["SelectorScreen_Challenges_shadow"] = true
	}
	// 未解锁时：不隐藏按钮和阴影（显示墓碑状态）

	// Survival mode (unlocked at 5-10)
	// Note: SelectorScreen_ZenGarden_button track corresponds to Survival mode
	if config.IsMenuModeUnlocked(config.MenuButtonSurvival, m.currentLevel) {
		hiddenTracks["SelectorScreen_ZenGarden_shadow"] = true
	}
	// 未解锁时：不隐藏按钮和阴影（显示墓碑状态）

	// Step 3: Apply merged HiddenTracks to ReanimComponent
	// Story 12.4: 首次启动时需要保留 leaf 轨道的隐藏状态
	if m.isFirstLaunch && reanimComp.HiddenTracks != nil {
		// 保留首次启动时设置的 leaf 轨道隐藏
		for trackName := range reanimComp.HiddenTracks {
			if !hiddenTracks[trackName] {
				log.Printf("[MainMenuScene] Preserving first-launch hidden track: %s", trackName)
				hiddenTracks[trackName] = true
			}
		}
	}
	reanimComp.HiddenTracks = hiddenTracks

	log.Printf("[MainMenuScene] Updated button visibility (level=%s, %d hidden tracks): Adventure=%v, Challenges=%v, Vasebreaker=%v, Survival=%v",
		m.currentLevel,
		len(hiddenTracks),
		config.IsMenuModeUnlocked(config.MenuButtonAdventure, m.currentLevel),
		config.IsMenuModeUnlocked(config.MenuButtonChallenges, m.currentLevel),
		config.IsMenuModeUnlocked(config.MenuButtonVasebreaker, m.currentLevel),
		config.IsMenuModeUnlocked(config.MenuButtonSurvival, m.currentLevel))
}

// onMenuButtonClicked handles clicks on SelectorScreen menu buttons.
// Checks unlock status and routes to appropriate handler.
//
// Parameters:
//   - buttonType: The type of button that was clicked
//
// Story 12.1: Main Menu Tombstone System Enhancement
func (m *MainMenuScene) onMenuButtonClicked(buttonType config.MenuButtonType) {
	log.Printf("[MainMenuScene] Button clicked: %v", buttonType)

	// Check if button is unlocked
	if !config.IsMenuModeUnlocked(buttonType, m.currentLevel) {
		log.Printf("[MainMenuScene] Button is locked (requires higher level)")

		// 播放按钮点击音效
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_TAP")
		}

		// Story 12.3: Show unlock dialog
		message := getUnlockMessage(buttonType)
		m.showUnlockDialog("未解锁！", message)
		return
	}

	// 播放按钮点击音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_TAP")
	}

	// Route to appropriate handler based on button type
	switch buttonType {
	case config.MenuButtonAdventure:
		// 检查下一关是否有配置文件
		saveManager := game.GetGameState().GetSaveManager()
		nextLevel := saveManager.GetNextLevelToPlay()
		log.Printf("[MainMenuScene] Adventure button clicked - next level: %s", nextLevel)

		// 检查关卡配置文件是否存在
		if !config.LevelConfigExists(nextLevel) {
			// 关卡配置不存在，弹出"敬请期待"对话框
			log.Printf("[MainMenuScene] Level %s config not found, showing coming soon dialog", nextLevel)
			m.showComingSoonDialog()
			return
		}

		// Story 12.6: Trigger zombie hand animation before starting adventure
		log.Printf("[MainMenuScene] Starting adventure - triggering zombie hand animation")
		m.triggerZombieHandAnimation()

	case config.MenuButtonChallenges:
		// TODO: Implement challenges/mini-games mode
		log.Printf("[MainMenuScene] Challenges mode - Not yet implemented")

	case config.MenuButtonVasebreaker:
		// TODO: Implement vasebreaker/puzzle mode
		log.Printf("[MainMenuScene] Vasebreaker mode - Not yet implemented")

	case config.MenuButtonSurvival:
		// TODO: Implement survival mode
		log.Printf("[MainMenuScene] Survival mode - Not yet implemented")

	default:
		log.Printf("[MainMenuScene] Warning: Unknown button type: %v", buttonType)
	}
}

// getUnlockMessage returns the unlock message for a button type
// Story 12.3: Dialog System Implementation
func getUnlockMessage(buttonType config.MenuButtonType) string {
	switch buttonType {
	case config.MenuButtonChallenges:
		return "进行更多新冒险来解锁玩玩小游戏。"
	case config.MenuButtonVasebreaker:
		return "进行更多新冒险来解锁解谜模式。"
	case config.MenuButtonSurvival:
		return "进行更多新冒险来解锁生存模式。"
	default:
		return "此功能尚未解锁。"
	}
}

// ========== Story 12.2: Bottom Function Bar Implementation ==========

// loadBottomButtonImages loads the normal and hover images for bottom function buttons.
//
// This method loads images but does NOT create entities. Buttons are rendered dynamically
// in the Draw method, following the SelectorScreen animation transform.
//
// Story 12.2: 底部功能栏重构（动画跟随版本）
func (m *MainMenuScene) loadBottomButtonImages() {
	m.bottomButtonImages = make(map[components.BottomButtonType][2]*ebiten.Image)
	m.hoveredBottomButton = components.BottomButtonNone // No hover initially

	// Resource ID mapping
	buttonResources := map[components.BottomButtonType][2]string{
		components.BottomButtonOptions: {"IMAGE_SELECTORSCREEN_OPTIONS1", "IMAGE_SELECTORSCREEN_OPTIONS2"},
		components.BottomButtonHelp:    {"IMAGE_SELECTORSCREEN_HELP1", "IMAGE_SELECTORSCREEN_HELP2"},
		components.BottomButtonQuit:    {"IMAGE_SELECTORSCREEN_QUIT1", "IMAGE_SELECTORSCREEN_QUIT2"},
	}

	// Load images for each button
	for btnType, resIDs := range buttonResources {
		normalImg, err := m.resourceManager.LoadImageByID(resIDs[0])
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to load normal image for button %d: %v", btnType, err)
			continue
		}

		hoverImg, err := m.resourceManager.LoadImageByID(resIDs[1])
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to load hover image for button %d: %v", btnType, err)
			continue
		}

		m.bottomButtonImages[btnType] = [2]*ebiten.Image{normalImg, hoverImg}
	}

	log.Printf("[MainMenuScene] Loaded bottom button images (count=%d)", len(m.bottomButtonImages))
}

// calculateBottomButtonScreenPos calculates the screen position of a bottom button,
// following the SelectorScreen animation transform.
//
// This follows the same logic as level numbers, using the background right section to follow animation.
//
// Returns: (screenX, screenY, width, height, ok)
//
// Story 12.2: 底部功能栏重构（动画跟随版本）
func (m *MainMenuScene) calculateBottomButtonScreenPos(buttonType components.BottomButtonType) (float64, float64, float64, float64, bool) {
	// Get SelectorScreen ReanimComponent
	if m.selectorScreenEntity == 0 {
		return 0, 0, 0, 0, false
	}

	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
	if !ok {
		return 0, 0, 0, 0, false
	}

	posComp, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.selectorScreenEntity)
	if !ok {
		return 0, 0, 0, 0, false
	}

	// Get button images to calculate size
	images, ok := m.bottomButtonImages[buttonType]
	if !ok || images[0] == nil {
		return 0, 0, 0, 0, false
	}

	btnWidth := float64(images[0].Bounds().Dx())
	btnHeight := float64(images[0].Bounds().Dy())

	// 底部按钮跟随背景右侧动画移动（与关卡数字类似）
	// 使用 SelectorScreen_BG_Right 轨道的偏移量
	referenceTrackName := "SelectorScreen_BG_Right"
	frames, trackExists := reanimComp.MergedTracks[referenceTrackName]

	// 背景右侧的最终位置（开场动画完成后）
	const finalBgRightX = 71.0
	const finalBgRightY = 41.0

	// 计算按钮的基础位置
	buttonIndex := int(buttonType)
	baseX, baseY := config.CalculateBottomButtonPosition(buttonIndex)

	// 默认使用最终位置（无动画或轨道不存在时）
	screenX := posComp.X + baseX - reanimComp.CenterOffsetX
	screenY := posComp.Y + baseY - reanimComp.CenterOffsetY

	if trackExists && len(frames) > 0 {
		// 获取当前帧索引
		currentFrameIdx := reanimComp.CurrentFrame
		if currentFrameIdx < 0 {
			currentFrameIdx = 0
		}
		if currentFrameIdx >= len(frames) {
			currentFrameIdx = len(frames) - 1
		}

		// 获取当前帧数据
		frame := frames[currentFrameIdx]

		// 获取背景当前的 X 和 Y 坐标
		frameX := finalBgRightX // 默认值
		if frame.X != nil {
			frameX = *frame.X
		}

		frameY := 0.0
		if frame.Y != nil {
			frameY = *frame.Y
		}

		// 计算背景相对于最终位置的偏移
		bgOffsetX := frameX - finalBgRightX
		bgOffsetY := frameY - finalBgRightY

		// 按钮跟随背景的偏移
		screenX = posComp.X + baseX + bgOffsetX - reanimComp.CenterOffsetX
		screenY = posComp.Y + baseY + bgOffsetY - reanimComp.CenterOffsetY
	}

	return screenX, screenY, btnWidth, btnHeight, true
}

// updateBottomButtons updates the hover and click states of bottom buttons
// based on mouse position and input.
//
// Story 12.2: 底部功能栏重构（动画跟随版本）
// Story 10.9: 添加悬停音效支持
func (m *MainMenuScene) updateBottomButtons(mouseX, mouseY int, isMouseReleased bool) {
	m.hoveredBottomButton = components.BottomButtonNone // Reset hover state

	// Check each button in order (Options, Help, Quit)
	buttonTypes := []components.BottomButtonType{
		components.BottomButtonOptions,
		components.BottomButtonHelp,
		components.BottomButtonQuit,
	}

	for _, btnType := range buttonTypes {
		// Calculate button's current screen position (dynamic, follows animation)
		screenX, screenY, btnWidth, btnHeight, ok := m.calculateBottomButtonScreenPos(btnType)
		if !ok {
			continue
		}

		// Skip detection if button is off-screen (still animating in)
		// 只检测屏幕内的按钮，避免动画过程中的不稳定检测
		if screenY > 600 || screenY+btnHeight < 0 || screenX > 800 || screenX+btnWidth < 0 {
			continue
		}

		// Expand clickable area with padding for easier clicking
		padding := config.BottomButtonClickPadding
		expandedX := screenX - padding
		expandedY := screenY - padding
		expandedWidth := btnWidth + padding*2
		expandedHeight := btnHeight + padding*2

		// Check if mouse is over this button (using expanded area)
		if isPointInRect(float64(mouseX), float64(mouseY), expandedX, expandedY, expandedWidth, expandedHeight) {
			// Mouse is over button
			if isMouseReleased {
				// Button clicked
				m.onBottomButtonClicked(btnType)
			} else {
				// Button hovered
				m.hoveredBottomButton = btnType
			}
			break // Only one button can be hovered at a time
		}
	}

	// Story 10.9: 检测悬停状态变化，播放音效
	if m.hoveredBottomButton != m.lastHoveredBottomButton {
		if m.hoveredBottomButton != components.BottomButtonNone {
			// 新进入一个按钮，播放悬停音效
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_BLEEP")
			}
		}
		m.lastHoveredBottomButton = m.hoveredBottomButton
	}
}

// drawBottomButtons renders the 3 bottom function buttons to the screen.
//
// Buttons follow the SelectorScreen animation transform, similar to level numbers.
//
// Story 12.2: 底部功能栏重构（动画跟随版本）
func (m *MainMenuScene) drawBottomButtons(screen *ebiten.Image) {
	// Draw each button in order (Options, Help, Quit)
	buttonTypes := []components.BottomButtonType{
		components.BottomButtonOptions,
		components.BottomButtonHelp,
		components.BottomButtonQuit,
	}

	for _, btnType := range buttonTypes {
		// Get button images
		images, ok := m.bottomButtonImages[btnType]
		if !ok {
			continue
		}

		// Select image based on hover state
		img := images[0] // Normal image
		if m.hoveredBottomButton == btnType && images[1] != nil {
			img = images[1] // Hover image
		}

		if img == nil {
			continue
		}

		// Calculate button's current screen position (dynamic, follows animation)
		screenX, screenY, _, _, ok := m.calculateBottomButtonScreenPos(btnType)
		if !ok {
			continue
		}

		// Draw button
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		screen.DrawImage(img, op)
	}
}

// onBottomButtonClicked handles bottom button click events
//
// Actions:
//   - Options: Opens the options panel
//   - Help: Opens the help panel
//   - Quit: Exits the game
//
// Story 12.2: 底部功能栏重构
func (m *MainMenuScene) onBottomButtonClicked(btnType components.BottomButtonType) {
	// 播放按钮点击音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_TAP")
	}

	switch btnType {
	case components.BottomButtonOptions:
		// Show options panel (Story 12.3)
		log.Printf("[MainMenuScene] Options button clicked")
		m.showOptionsDialog()

	case components.BottomButtonHelp:
		// Show help panel (Story 12.3)
		log.Printf("[MainMenuScene] Help button clicked")
		m.showHelpDialog()

	case components.BottomButtonQuit:
		// Exit game
		log.Printf("[MainMenuScene] Quit button clicked - exiting game")
		exitGame()
	}
}

// isPointInRect checks if a point (px, py) is inside a rectangle defined by (x, y, width, height).
// Returns true if the point is within the rectangle bounds (inclusive), false otherwise.
func isPointInRect(px, py, x, y, width, height float64) bool {
	return px >= x && px <= x+width && py >= y && py <= y+height
}

// disableAllButtons disables all menu buttons during zombie hand animation.
// Story 12.6 Task 2.3 & 2.6
//
// Note: This function is called when zombie hand animation starts.
// The actual button blocking logic is implemented in Update() by checking
// menuState == MainMenuStateZombieHandPlaying and returning early.
func (m *MainMenuScene) disableAllButtons() {
	// Clear hover states
	m.hoveredButton = ""
	m.hoveredBottomButton = components.BottomButtonNone
	log.Printf("[MainMenuScene] 🚫 Disabled all buttons (zombie hand animation playing)")
}
