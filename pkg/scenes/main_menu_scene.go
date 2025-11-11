package scenes

import (
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	// WindowWidth is the logical width of the game window in pixels.
	WindowWidth = 800
	// WindowHeight is the logical height of the game window in pixels.
	WindowHeight = 600
)

// MainMenuScene represents the main menu screen of the game.
// It displays when the game starts and allows the player to navigate to other scenes.
type MainMenuScene struct {
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager
	backgroundImage *ebiten.Image
	bgmPlayer       *audio.Player
	buttons         []components.Button
	wasMousePressed bool // Track mouse state from previous frame to detect click edges

	// Story 12.1: SelectorScreen Reanim entity and systems
	entityManager        *ecs.EntityManager
	reanimSystem         *systems.ReanimSystem
	renderSystem         *systems.RenderSystem
	selectorScreenEntity ecs.EntityID

	// Story 12.1: Button state management
	buttonHitboxes []config.MenuButtonHitbox
	hoveredButton  string // Current hovered button track name (empty = no hover)
	currentLevel   string // Current highest level from save (format: "X-Y")

	// Story 12.1 Task 5: Button highlight images
	buttonNormalImages    map[string]*ebiten.Image // Map: track name -> normal button image
	buttonHighlightImages map[string]*ebiten.Image // Map: track name -> highlight button image
	lastHoveredButton     string                   // Track the last hovered button for sound effect (play only once)

	// Cloud animation management
	cloudAnimsResumed bool // Track whether cloud animations have been resumed after opening animation
}

// NewMainMenuScene creates and returns a new MainMenuScene instance.
// It loads the main menu background image and initializes interactive buttons.
//
// Parameters:
//   - rm: The ResourceManager instance used to load game resources.
//   - sm: The SceneManager instance used to switch between scenes.
//
// Returns:
//   - A pointer to the newly created MainMenuScene.
//
// If the background image fails to load, the scene will fall back to a solid color background.
func NewMainMenuScene(rm *game.ResourceManager, sm *game.SceneManager) *MainMenuScene {
	scene := &MainMenuScene{
		resourceManager: rm,
		sceneManager:    sm,
	}

	// Story 12.1: Initialize ECS systems for SelectorScreen Reanim
	scene.entityManager = ecs.NewEntityManager()
	scene.reanimSystem = systems.NewReanimSystem(scene.entityManager)

	// Story 13.6: 设置配置管理器
	if configManager := rm.GetReanimConfigManager(); configManager != nil {
		scene.reanimSystem.SetConfigManager(configManager)
	}

	scene.renderSystem = systems.NewRenderSystem(scene.entityManager)

	// Story 13.4: Enable render cache optimization
	scene.renderSystem.SetReanimSystem(scene.reanimSystem)
	log.Printf("[MainMenuScene] Initialized ECS systems")

	// Story 12.1: Create SelectorScreen Reanim entity
	selectorEntity, err := entities.NewSelectorScreenEntity(scene.entityManager, rm)
	if err != nil {
		log.Printf("Warning: Failed to create SelectorScreen entity: %v", err)
		log.Printf("Main menu will use fallback rendering")
		scene.selectorScreenEntity = 0
	} else {
		scene.selectorScreenEntity = selectorEntity

		// ✅ Story 13.10 Bug Fix: 重新分析轨道类型
		// selector_screen_factory 把所有轨道都放进了 VisualTracks（包括动画定义轨道）
		// 需要使用 ReanimSystem 重新分类，将动画定义轨道移到 LogicalTracks
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](scene.entityManager, selectorEntity)
		if ok && reanimComp.ReanimXML != nil {
			visualTracks, logicalTracks := scene.reanimSystem.AnalyzeTrackTypes(reanimComp.ReanimXML)
			reanimComp.VisualTracks = visualTracks
			reanimComp.LogicalTracks = logicalTracks
			log.Printf("[MainMenuScene] 重新分析轨道类型: VisualTracks=%d, LogicalTracks=%d", len(visualTracks), len(logicalTracks))
		}

		// Story 13.8: 初始化 SelectorScreen 动画
		// ✅ Story 13.10 修正：不要同时播放所有动画，而是只播放开场动画
		// 云朵和草动画应该在开场动画完成后再播放
		// 策略：开场阶段只播放 anim_open，循环阶段再添加其他动画

		// 1. 先播放开场动画（只播放 anim_open）
		if err := scene.reanimSystem.PlayAnimation(selectorEntity, "anim_open"); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to play anim_open: %v", err)
		}

		// 2. 不再添加 anim_sign 和 anim_idle（这些会覆盖背景）
		// scene.reanimSystem.AddAnimation(selectorEntity, "anim_sign")
		// scene.reanimSystem.AddAnimation(selectorEntity, "anim_idle")

		// 3. 云朵和草动画在开场完成后才添加（见 Update() 中的 cloudAnimsResumed 逻辑）

		// 4. 完成动画设置（初始化动画帧索引）
		// ✅ Story 13.10: FinalizeAnimations 不再生成轨道绑定
		// 现在只负责初始化 AnimationFrameIndices，确保每个动画有独立的帧计数器
		if err := scene.reanimSystem.FinalizeAnimations(selectorEntity); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to finalize animations: %v", err)
		}

		// 5. 获取 ReanimComponent 并设置循环状态
		reanimComp, ok = ecs.GetComponent[*components.ReanimComponent](scene.entityManager, selectorEntity)
		if ok {
			// 🔍 调试：输出 AnimationFPS 的值
			log.Printf("[MainMenuScene] 🔍 DEBUG: AnimationFPS = %.1f (全局 FPS)", reanimComp.AnimationFPS)

			// 初始化 AnimationLoopStates、AnimationPausedStates 和 AnimationFPSOverrides
			reanimComp.AnimationLoopStates = make(map[string]bool)
			reanimComp.AnimationPausedStates = make(map[string]bool)
			reanimComp.AnimationFPSOverrides = make(map[string]float64)
			reanimComp.AnimationSpeedOverrides = make(map[string]float64)

			// ✅ 从配置中加载每个动画的独立 FPS 和速度倍率
			if configManager := rm.GetReanimConfigManager(); configManager != nil {
				unitConfig, err := configManager.GetUnit("selectorscreen")
				if err == nil {
					for _, animInfo := range unitConfig.AvailableAnimations {
						if animInfo.FPS > 0 {
							reanimComp.AnimationFPSOverrides[animInfo.Name] = animInfo.FPS
							log.Printf("[MainMenuScene] 动画 %s 使用独立 FPS = %.1f", animInfo.Name, animInfo.FPS)
						}
						if animInfo.Speed > 0 {
							reanimComp.AnimationSpeedOverrides[animInfo.Name] = animInfo.Speed
							log.Printf("[MainMenuScene] 动画 %s 使用速度倍率 = %.2f", animInfo.Name, animInfo.Speed)
						}
					}
				} else {
					log.Printf("[MainMenuScene] Warning: 无法加载 selectorscreen 配置: %v", err)
				}
			}

			// 开场动画设置为非循环
			reanimComp.AnimationLoopStates["anim_open"] = false
			reanimComp.AnimationLoopStates["anim_sign"] = false
			reanimComp.AnimationLoopStates["anim_idle"] = false

			// ✅ Story 13.10: 云朵动画在开场完成后才添加，这里不需要初始化
			// 云朵动画会在 Update() 中检测到 IsFinished 后动态添加

			// 全局设置为循环模式（但具体每个动画由 AnimationLoopStates 控制）
			reanimComp.IsLooping = true

			// ✅ Story 13.10: 不再需要手动绑定轨道
			// 新的渲染逻辑直接从动画遍历到轨道，自然覆盖

			log.Printf("[MainMenuScene] ✅ SelectorScreen 动画初始化完成（开场动画非循环）")
		}

		// 修复：SelectorScreen 是全屏 UI，应该使用左上角对齐（Reanim 原始坐标）
		// 而不是中心对齐。禁用 CenterOffset 功能。
		if ok {
			reanimComp.CenterOffsetX = 0
			reanimComp.CenterOffsetY = 0
			log.Printf("[MainMenuScene] SelectorScreen 使用左上角对齐（CenterOffset = 0）")
		}
	}

	// Story 12.1: Initialize button hitboxes
	scene.buttonHitboxes = config.MenuButtonHitboxes

	// Story 12.1 Task 5: Load button highlight images
	scene.buttonNormalImages = make(map[string]*ebiten.Image)
	scene.buttonHighlightImages = make(map[string]*ebiten.Image)
	scene.loadButtonImages(rm)

	// Story 12.1: Load current level from save
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	if err := saveManager.Load(); err == nil {
		scene.currentLevel = saveManager.GetHighestLevel()
		if scene.currentLevel == "" {
			scene.currentLevel = "1-1" // Default for new players
		}
		log.Printf("[MainMenuScene] Loaded highest level: %s", scene.currentLevel)
	} else {
		scene.currentLevel = "1-1" // Default for new players
		log.Printf("[MainMenuScene] No save file, defaulting to level 1-1")
	}

	// Story 12.1: Update button visibility based on unlock status
	// scene.updateButtonVisibility()

	// Load background image (fallback if SelectorScreen fails)
	// img, err := rm.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_BG")
	// if err != nil {
	// 	log.Printf("Warning: Failed to load main menu background: %v", err)
	// 	log.Printf("The game will use a fallback solid color background")
	// 	// Fallback: keep backgroundImage as nil, will use solid color in Draw()
	// } else {
	// 	scene.backgroundImage = img
	// }

	// Load background music (using titlescreen music from loaderbar group)
	// Note: Need to ensure loaderbar group is loaded before this
	player, err := rm.LoadSoundEffect("assets/sounds/titlescreen.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load main menu music: %v", err)
		// Continue without music
	} else {
		scene.bgmPlayer = player
	}

	// Initialize buttons
	// scene.initButtons()

	return scene
}

// Update updates the main menu scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (m *MainMenuScene) Update(deltaTime float64) {
	// Ensure background music is playing
	if m.bgmPlayer != nil && !m.bgmPlayer.IsPlaying() {
		m.bgmPlayer.Play()
	}

	// Story 12.1: Update Reanim system (animate clouds, flowers, etc.)
	if m.reanimSystem != nil {
		m.reanimSystem.Update(deltaTime)

		// ✅ 检测开场动画完成，切换到循环动画
		if !m.cloudAnimsResumed && m.selectorScreenEntity != 0 {
			reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
			if ok && reanimComp.IsFinished {
				// 开场动画已完成，添加循环动画
				// 策略：
				//   1. 保留 anim_open（停留在最后一帧，提供背景）
				//   2. 添加 anim_idle（提供按钮动画）
				//   3. 添加云朵和草动画（在上层）
				// 原因：anim_idle 从物理帧 41 开始，但背景轨道在帧 41 被隐藏了（f=-1）
				//       anim_open（帧 0-12）提供背景，anim_idle（帧 41+）提供按钮动画

				// ✅ 不移除、不暂停 anim_open，让它自然停留在最后一帧（非循环动画完成后不更新）

				// 1. 添加 anim_idle（提供按钮和装饰物动画）
				if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_idle"); err != nil {
					log.Printf("[MainMenuScene] Warning: Failed to add anim_idle: %v", err)
				}
				reanimComp.AnimationLoopStates["anim_idle"] = true

				// 2. 添加云朵和草动画（在 anim_idle 之后渲染，形成图层）
				cloudAnims := []string{"anim_grass", "anim_cloud1", "anim_cloud2", "anim_cloud4",
					"anim_cloud5", "anim_cloud6", "anim_cloud7"}

				for _, animName := range cloudAnims {
					if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, animName); err != nil {
						log.Printf("[MainMenuScene] Warning: Failed to add animation %s: %v", animName, err)
					}
					reanimComp.AnimationLoopStates[animName] = true
				}

				// 3. 重新完成动画设置（初始化新动画的帧索引）
				if err := m.reanimSystem.FinalizeAnimations(m.selectorScreenEntity); err != nil {
					log.Printf("[MainMenuScene] Warning: Failed to finalize animations: %v", err)
				}

				m.cloudAnimsResumed = true
				log.Printf("[MainMenuScene] ✅ 开场动画完成，已切换到循环模式（保留 anim_open 背景 + anim_idle + 云朵）")
			}
		}
	}

	// Get mouse position
	mouseX, mouseY := ebiten.CursorPosition()

	// Check if mouse button is currently pressed
	isMousePressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Detect click edge (button was just pressed this frame)
	isMouseClicked := isMousePressed && !m.wasMousePressed

	// Story 12.1: Check SelectorScreen button hitboxes
	m.hoveredButton = "" // Reset hovered button
	for _, hitbox := range m.buttonHitboxes {
		// Check if mouse is in hitbox
		if isPointInRect(float64(mouseX), float64(mouseY), hitbox.X, hitbox.Y, hitbox.Width, hitbox.Height) {
			m.hoveredButton = hitbox.TrackName

			if isMouseClicked {
				// Button clicked
				m.onMenuButtonClicked(hitbox.ButtonType)
			}
			break // Only one button can be hovered at a time
		}
	}

	// Update old-style button states based on mouse position and clicks
	for i := range m.buttons {
		btn := &m.buttons[i]

		// Check if mouse is hovering over this button
		if isPointInRect(float64(mouseX), float64(mouseY), btn.X, btn.Y, btn.Width, btn.Height) {
			// Mouse is over the button
			if isMouseClicked {
				// Button was clicked
				btn.State = components.UIClicked
				if btn.OnClick != nil {
					btn.OnClick()
				}
			} else {
				// Button is hovered but not clicked
				btn.State = components.UIHovered
			}
		} else {
			// Mouse is not over the button
			btn.State = components.UINormal
		}
	}

	// Remember mouse state for next frame
	m.wasMousePressed = isMousePressed

	// Story 12.1 Task 5: Update button highlight based on hover state
	m.updateButtonHighlight()
}

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
			switch m.lastHoveredButton {
			case "SelectorScreen_Adventure_button":
				reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"] = normalImg
			case "SelectorScreen_Survival_button":
				reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"] = normalImg
			case "SelectorScreen_Challenges_button":
				reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_BUTTON"] = normalImg
			case "SelectorScreen_ZenGarden_button":
				reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_BUTTON"] = normalImg
			}
		}
	}

	// Step 2: Apply highlight to the currently hovered button (if any and unlocked)
	if m.hoveredButton != "" {
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
		if found && config.IsMenuModeUnlocked(buttonType, m.currentLevel) {
			// Apply highlight image if available
			if highlightImg, exists := m.buttonHighlightImages[m.hoveredButton]; exists {
				// Find the correct image reference for this button and apply highlight
				switch m.hoveredButton {
				case "SelectorScreen_Adventure_button":
					reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"] = highlightImg
				case "SelectorScreen_Survival_button":
					reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"] = highlightImg
				case "SelectorScreen_Challenges_button":
					reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_BUTTON"] = highlightImg
				case "SelectorScreen_ZenGarden_button":
					reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_BUTTON"] = highlightImg
				}
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

// playGraveButtonSound plays the stone grinding sound effect for button hover.
//
// Story 12.1 Task 5: Button Highlight Effect
func (m *MainMenuScene) playGraveButtonSound() {
	// Check if resource manager is available (nil in unit tests)
	if m.resourceManager == nil {
		return
	}

	player, err := m.resourceManager.LoadSoundEffect("assets/sounds/gravebutton.ogg")
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load grave button sound: %v", err)
		return
	}
	player.Rewind()
	player.Play()
}

// Draw renders the main menu scene to the screen.
// If a background image is loaded, it draws the image.
// Otherwise, it uses a dark blue fallback background.
func (m *MainMenuScene) Draw(screen *ebiten.Image) {
	// Story 12.1: Draw SelectorScreen Reanim (contains background, buttons, decorations)
	if m.selectorScreenEntity != 0 {
		// 主菜单使用 Reanim 渲染，直接调用 DrawEntity
		// 使用 cameraX = 0（主菜单没有摄像机偏移）
		m.renderSystem.DrawEntity(screen, m.selectorScreenEntity, 0)

		// Note: Old m.buttons drawing removed - SelectorScreen Reanim handles all button rendering
	} else {
		// Fallback: Draw background image if SelectorScreen failed to load
		if m.backgroundImage != nil {
			// Scale background image to fit window size if needed
			bounds := m.backgroundImage.Bounds()
			bgWidth := float64(bounds.Dx())
			bgHeight := float64(bounds.Dy())

			// Calculate scale factors
			scaleX := WindowWidth / bgWidth
			scaleY := WindowHeight / bgHeight

			// Create draw options with scaling
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scaleX, scaleY)

			// Draw the background image
			screen.DrawImage(m.backgroundImage, op)
		} else {
			// Fallback: Fill the screen with a dark blue color (midnight blue)
			screen.Fill(color.RGBA{R: 25, G: 25, B: 112, A: 255})
		}

		// Fallback: Draw old-style buttons only if Reanim failed to load
		for _, btn := range m.buttons {
			// Skip drawing if button has no image
			if btn.NormalImage == nil {
				continue
			}

			// Select which image to draw based on button state
			var img *ebiten.Image
			if btn.State == components.UIHovered && btn.HoverImage != nil {
				// Use hover image if available
				img = btn.HoverImage
			} else {
				// Use normal image
				img = btn.NormalImage
			}

			// Create draw options
			op := &ebiten.DrawImageOptions{}

			// Apply visual effects for hovered state (if no hover image available)
			if btn.State == components.UIHovered && btn.HoverImage == nil {
				// Make button brighter when hovered
				op.ColorM.Scale(1.2, 1.2, 1.2, 1.0)
			}

			// Position the button
			op.GeoM.Translate(btn.X, btn.Y)

			// Draw the button
			screen.DrawImage(img, op)
		}
	}
}

// initButtons initializes the menu buttons with their positions, images, and click handlers.
func (m *MainMenuScene) initButtons() {
	// Load button images using resource IDs
	adventureNormal, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON")
	if err != nil {
		log.Printf("Warning: Failed to load adventure button normal image: %v", err)
		adventureNormal = nil
	}

	adventureHover, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_HIGHLIGHT")
	if err != nil {
		log.Printf("Warning: Failed to load adventure button hover image: %v", err)
		adventureHover = nil
	}

	// For exit button, we'll use a simple button image
	exitNormal, err := m.resourceManager.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		log.Printf("Warning: Failed to load exit button image: %v", err)
		exitNormal = nil
	}

	// Calculate button positions (centered on screen)
	// Adventure button dimensions (estimate based on typical asset size)
	var adventureWidth, adventureHeight float64 = 200, 80
	if adventureNormal != nil {
		bounds := adventureNormal.Bounds()
		adventureWidth = float64(bounds.Dx())
		adventureHeight = float64(bounds.Dy())
	}

	// Exit button dimensions
	var exitWidth, exitHeight float64 = 150, 60
	if exitNormal != nil {
		bounds := exitNormal.Bounds()
		exitWidth = float64(bounds.Dx())
		exitHeight = float64(bounds.Dy())
	}

	// Position buttons vertically centered with spacing
	const buttonSpacing = 30.0
	adventureX := (WindowWidth - adventureWidth) / 2
	adventureY := WindowHeight/2 - adventureHeight - buttonSpacing/2

	exitX := (WindowWidth - exitWidth) / 2
	exitY := WindowHeight/2 + buttonSpacing/2

	// Initialize button array
	m.buttons = []components.Button{
		{
			X:           adventureX,
			Y:           adventureY,
			Width:       adventureWidth,
			Height:      adventureHeight,
			NormalImage: adventureNormal,
			HoverImage:  adventureHover,
			State:       components.UINormal,
			OnClick:     m.onStartAdventureClicked,
		},
		{
			X:           exitX,
			Y:           exitY,
			Width:       exitWidth,
			Height:      exitHeight,
			NormalImage: exitNormal,
			HoverImage:  nil, // Will use color/scale effects instead
			State:       components.UINormal,
			OnClick:     m.onExitClicked,
		},
	}
}

// onStartAdventureClicked handles the "Start Adventure" button click.
// It switches the current scene to the GameScene.
func (m *MainMenuScene) onStartAdventureClicked() {
	log.Println("Start Adventure button clicked")

	// Story 8.6: Load level from save file or default to 1-1
	levelToLoad := "1-1" // Default to first level
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	if err := saveManager.Load(); err == nil {
		// Save file exists, get highest level
		highestLevel := saveManager.GetHighestLevel()
		if highestLevel != "" {
			levelToLoad = highestLevel
			log.Printf("[MainMenu] Loading from save: highest level = %s", highestLevel)
		}
	}

	// Pass ResourceManager, SceneManager, and levelID to GameScene
	gameScene := NewGameScene(m.resourceManager, m.sceneManager, levelToLoad)
	m.sceneManager.SwitchTo(gameScene)
}

// onExitClicked handles the "Exit Game" button click.
// It terminates the application.
func (m *MainMenuScene) onExitClicked() {
	log.Println("Exit Game button clicked")
	os.Exit(0)
}

// isPointInRect checks if a point (px, py) is inside a rectangle defined by (x, y, width, height).
// Returns true if the point is within the rectangle bounds (inclusive), false otherwise.
func isPointInRect(px, py, x, y, width, height float64) bool {
	return px >= x && px <= x+width && py >= y && py <= y+height
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

	// 实验：完全移除 HiddenTracks 白名单，让所有轨道依赖动画定义轨道的 f 值自然控制
	// 这样可以验证 anim_grass, anim_open 等动画定义轨道是否能正确控制纯视觉轨道
	reanimComp.HiddenTracks = nil

	log.Printf("[MainMenuScene] 实验模式：移除 HiddenTracks 白名单，所有轨道依赖 f 值控制")
	log.Printf("[MainMenuScene] Button visibility (level=%s): Adventure=%v, Challenges=%v, Vasebreaker=%v, Survival=%v",
		m.currentLevel,
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

		// Play button click sound (shadow buttons also have click feedback)
		player, err := m.resourceManager.LoadSoundEffect("assets/sounds/buttonclick.ogg")
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to load button click sound: %v", err)
		} else {
			player.Rewind()
			player.Play()
		}

		// Show unlock dialog (Story 12.3 - temporarily use log)
		log.Printf("[MainMenuScene] TODO Story 12.3: Show unlock dialog for mode %v", buttonType)
		return
	}

	// Play button click sound
	// Note: SOUND_BUTTONCLICK should be loaded in initialization
	player, err := m.resourceManager.LoadSoundEffect("assets/sounds/buttonclick.ogg")
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load button click sound: %v", err)
	} else {
		player.Rewind()
		player.Play()
	}

	// Route to appropriate handler based on button type
	switch buttonType {
	case config.MenuButtonAdventure:
		// Start adventure mode
		log.Printf("[MainMenuScene] Starting Adventure mode")
		m.onStartAdventureClicked()

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
