package scenes

import (
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/modules"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	hasStartedGame bool   // Whether user has started the game (true = show Adventure button, false = show StartAdventure button)

	// Story 12.1 Task 5: Button highlight images
	buttonNormalImages    map[string]*ebiten.Image // Map: track name -> normal button image
	buttonHighlightImages map[string]*ebiten.Image // Map: track name -> highlight button image
	lastHoveredButton     string                   // Track the last hovered button for sound effect (play only once)

	// Cloud animation management
	cloudAnimsResumed bool // Track whether cloud animations have been resumed after opening animation

	// Story 12.1 Task 6: Debug logging
	levelNumbersDebugLogged bool // Track whether debug info has been logged (only log once)

	// Story 12.3: Dialog system
	dialogRenderSystem *systems.DialogRenderSystem // Dialog render system
	dialogInputSystem  *systems.DialogInputSystem  // Dialog input system
	currentDialog      ecs.EntityID                // Current open dialog entity (0 = none)

	// Story 12.3: Help and Options panels
	buttonSystem       *systems.ButtonSystem       // Button interaction system
	buttonRenderSystem *systems.ButtonRenderSystem // Button render system
	helpPanelModule    *modules.HelpPanelModule    // Help panel module
	optionsPanelModule *modules.OptionsPanelModule // Options panel module

	// Story 12.2: Bottom function bar (Options/Help/Quit buttons)
	bottomButtonImages  map[components.BottomButtonType][2]*ebiten.Image // [0]=Normal, [1]=Hover
	hoveredBottomButton components.BottomButtonType                      // Current hovered bottom button (-1 = none)

	// Cursor state tracking
	lastCursorShape ebiten.CursorShapeType // Track last cursor shape to avoid unnecessary updates

	// Keyboard state tracking for edge detection
	wasF1Pressed bool // Track F1 key state from previous frame
	wasOPressed  bool // Track O key state from previous frame

	// Story 12.4: User management UI
	isFirstLaunch         bool                           // Whether this is first launch (no users)
	newUserDialogShown    bool                           // Whether new user dialog has been shown for first launch
	currentUserDialogID   ecs.EntityID                   // Current user dialog entity (0 = none)
	currentInputBoxID     ecs.EntityID                   // Current text input box entity (0 = none)
	currentErrorDialogID  ecs.EntityID                   // Current error dialog entity (0 = none) - Story 12.4: 防止错误对话框叠加
	textInputSystem       *systems.TextInputSystem       // Text input system
	textInputRenderSystem *systems.TextInputRenderSystem // Text input render system
	userSignEntity        ecs.EntityID                   // User sign entity (wood sign showing username)
	saveManager           *game.SaveManager              // Save manager reference for user management
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
		resourceManager:     rm,
		sceneManager:        sm,
		lastCursorShape:     -1, // 初始化为无效值，确保第一次更新光标
		hoveredBottomButton: components.BottomButtonNone,
	}

	// Story 12.1: Initialize ECS systems for SelectorScreen Reanim
	scene.entityManager = ecs.NewEntityManager()
	scene.reanimSystem = systems.NewReanimSystem(scene.entityManager)

	// Story 13.6: 设置配置管理器
	if configManager := rm.GetReanimConfigManager(); configManager != nil {
		scene.reanimSystem.SetConfigManager(configManager)
	}

	scene.renderSystem = systems.NewRenderSystem(scene.entityManager)
	// ✅ 修复：设置 ReanimSystem 引用，以便 RenderSystem 调用 GetRenderData()
	scene.renderSystem.SetReanimSystem(scene.reanimSystem)
	log.Printf("[MainMenuScene] Initialized ECS systems")

	// Story 12.4: Check if this is first launch (BEFORE creating SelectorScreen)
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	scene.saveManager = saveManager // Save reference for user management
	users, err := saveManager.LoadUserList()
	if err != nil || len(users) == 0 {
		// First launch: no users exist
		scene.isFirstLaunch = true
		scene.currentLevel = "1-1"
		scene.hasStartedGame = false
		log.Printf("[MainMenuScene] First launch detected: no users found")
	} else {
		// Not first launch: load current user's save
		scene.isFirstLaunch = false
		if err := saveManager.Load(); err == nil {
			scene.currentLevel = saveManager.GetHighestLevel()
			if scene.currentLevel == "" {
				scene.currentLevel = "1-1"
			}
			scene.hasStartedGame = saveManager.GetHasStartedGame()
			log.Printf("[MainMenuScene] Loaded highest level: %s, hasStartedGame: %v", scene.currentLevel, scene.hasStartedGame)
		} else {
			scene.currentLevel = "1-1"
			scene.hasStartedGame = false
			log.Printf("[MainMenuScene] No save file, defaulting to level 1-1")
		}
	}

	// Story 12.1: Create SelectorScreen Reanim entity
	selectorEntity, err := entities.NewSelectorScreenEntity(scene.entityManager, rm)
	if err != nil {
		log.Printf("Warning: Failed to create SelectorScreen entity: %v", err)
		log.Printf("Main menu will use fallback rendering")
		scene.selectorScreenEntity = 0
	} else {
		scene.selectorScreenEntity = selectorEntity

		// ✅ Epic 14: 移除 AnalyzeTrackTypes 调用（已私有化，由 ReanimSystem 内部处理）
		// PlayAnimation/AddAnimation 会自动调用 analyzeTrackTypes

		// Story 12.4 AC8: **关键修复**：在播放动画之前先设置 HiddenTracks
		// 这样首次渲染就不会显示木牌和草叶子
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](scene.entityManager, selectorEntity)
		if ok && scene.isFirstLaunch {
			reanimComp.HiddenTracks = make(map[string]bool)
			// 隐��木牌轨道
			reanimComp.HiddenTracks["woodsign1"] = true
			reanimComp.HiddenTracks["woodsign2"] = true
			reanimComp.HiddenTracks["woodsign3"] = true
			// 隐藏草叶子轨道
			reanimComp.HiddenTracks["leaf1"] = true
			reanimComp.HiddenTracks["leaf2"] = true
			reanimComp.HiddenTracks["leaf3"] = true
			reanimComp.HiddenTracks["leaf4"] = true
			reanimComp.HiddenTracks["leaf5"] = true
			reanimComp.HiddenTracks["leaf22"] = true
			reanimComp.HiddenTracks["leaf_SelectorScreen_Leaves"] = true
			log.Printf("[MainMenuScene] First launch: hidden woodsign and leaf tracks (**BEFORE** playing animation)")
		}

		// ✅ Story 12.4 AC8: 根据首次启动状态播放不同的动画
		if scene.isFirstLaunch {
			// 首次启动：仅播放 anim_open（背景展开），不播放木牌和草动画
			ecs.AddComponent(scene.entityManager, selectorEntity, &components.AnimationCommandComponent{
				AnimationName: "anim_open", // 单动画模式
				Processed:     false,
			})
			log.Printf("[MainMenuScene] First launch: playing anim_open only")
		} else {
			// 非首次启动：播放完整开场组合（anim_open + anim_sign）
			ecs.AddComponent(scene.entityManager, selectorEntity, &components.AnimationCommandComponent{
				UnitID:    "selectorscreen",
				ComboName: "opening", // 使用配置的组合动画（包含 anim_open 和 anim_sign）
				Processed: false,
			})
			log.Printf("[MainMenuScene] Normal launch: playing opening combo (anim_open + anim_sign)")
		}

		// 处理 AnimationCommand（立即初始化动画）
		scene.reanimSystem.Update(0)

		// 3. 云朵和草动画在开场完成后才添加（见 Update() 中的 cloudAnimsResumed 逻辑）

		// 4. ✅ Epic 14: 移除 FinalizeAnimations 调用（已私有化，由 PlayAnimation/AddAnimation 内部处理）

		// 5. 获取 ReanimComponent 并设置循环状态
		reanimComp, ok = ecs.GetComponent[*components.ReanimComponent](scene.entityManager, selectorEntity)
		if ok {
			// 🔍 调试：输出 AnimationFPS 的值
			log.Printf("[MainMenuScene] ���� DEBUG: AnimationFPS = %.1f (全局 FPS)", reanimComp.AnimationFPS)

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

			// 开场动画设置为非循环（opening 组合包含 anim_open 和 anim_sign）
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

	// 调试日志：显示所有按钮的 hitbox 配置
	log.Printf("[MainMenuScene] 加载了 %d 个按钮 hitbox 配置:", len(scene.buttonHitboxes))
	for i, hitbox := range scene.buttonHitboxes {
		// 计算四边形的宽度和高度（用于日志显示）
		width := hitbox.TopRight.X - hitbox.TopLeft.X
		height := hitbox.BottomLeft.Y - hitbox.TopLeft.Y
		log.Printf("[MainMenuScene]   [%d] %s: 左上角=(%.1f, %.1f), 尺寸=%.1fx%.1f, 类型=%v",
			i, hitbox.TrackName, hitbox.TopLeft.X, hitbox.TopLeft.Y, width, height, hitbox.ButtonType)
	}

	// Story 12.1 Task 5: Load button highlight images
	scene.buttonNormalImages = make(map[string]*ebiten.Image)
	scene.buttonHighlightImages = make(map[string]*ebiten.Image)
	scene.loadButtonImages(rm)

	// Story 12.1: Update button visibility based on unlock status
	scene.updateButtonVisibility()

	// Story 12.4: Initialize user sign (if not first launch)
	if !scene.isFirstLaunch {
		scene.initUserSign()
	}

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

	// Story 12.3: Initialize dialog systems
	// 加载不同大小的字体用于对话框渲染
	// 标题字体（大）
	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 22)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load dialog title font: %v", err)
	}

	// 消息字体（中）
	messageFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 18)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load dialog message font: %v", err)
	}

	// 按钮字体（与奖励面板按钮字体一致）
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load dialog button font: %v", err)
	}

	scene.dialogRenderSystem = systems.NewDialogRenderSystem(scene.entityManager, WindowWidth, WindowHeight, titleFont, messageFont, buttonFont)
	scene.dialogInputSystem = systems.NewDialogInputSystem(scene.entityManager)
	scene.currentDialog = 0 // No dialog initially
	log.Printf("[MainMenuScene] Initialized dialog systems")

	// Story 12.3: Initialize button systems (shared by help and options panels)
	scene.buttonSystem = systems.NewButtonSystem(scene.entityManager)
	scene.buttonRenderSystem = systems.NewButtonRenderSystem(scene.entityManager)

	// Story 12.3: Initialize help panel module
	helpPanel, err := modules.NewHelpPanelModule(
		scene.entityManager,
		rm,
		scene.buttonSystem,
		scene.buttonRenderSystem,
		WindowWidth,
		WindowHeight,
		nil, // onClose callback (no special action needed)
	)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to initialize help panel: %v", err)
	} else {
		scene.helpPanelModule = helpPanel
		log.Printf("[MainMenuScene] Help panel module initialized")
	}

	// Story 12.3: Initialize options panel module
	optionsPanel, err := modules.NewOptionsPanelModule(
		scene.entityManager,
		rm,
		scene.buttonSystem,
		scene.buttonRenderSystem,
		WindowWidth,
		WindowHeight,
		nil, // onClose callback (no special action needed)
	)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to initialize options panel: %v", err)
	} else {
		scene.optionsPanelModule = optionsPanel
		log.Printf("[MainMenuScene] Options panel module initialized")
	}

	// Story 12.2: Load bottom function button images
	scene.loadBottomButtonImages()

	// Story 12.4: Initialize text input systems (for user management dialogs)
	scene.textInputSystem = systems.NewTextInputSystem(scene.entityManager)
	// 加载输入框字体（与消息字体一致）
	inputFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load input font: %v", err)
	}
	scene.textInputRenderSystem = systems.NewTextInputRenderSystem(scene.entityManager, inputFont)
	log.Printf("[MainMenuScene] Text input systems initialized")

	// ✅ Story 12.4: 设置 DialogRenderSystem 的 TextInputRenderSystem 引用
	// 这样 DialogRenderSystem 可以在渲染对话框后立即渲染其子实体（输入框）
	scene.dialogRenderSystem.SetTextInputRenderSystem(scene.textInputRenderSystem)
	log.Printf("[MainMenuScene] Set TextInputRenderSystem reference in DialogRenderSystem")

	return scene
}

// Update updates the main menu scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (m *MainMenuScene) Update(deltaTime float64) {
	// 清理上一帧标记删除的实体（确保本帧开始前已删除）
	m.entityManager.RemoveMarkedEntities()

	// Story 12.4: Check for first launch and show new user dialog
	if m.isFirstLaunch && !m.newUserDialogShown {
		m.showNewUserDialogForFirstLaunch()
		m.newUserDialogShown = true
	}

	// Story 12.4: Update text input system (for user dialogs)
	if m.textInputSystem != nil {
		m.textInputSystem.Update(deltaTime)
	}

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
				//   3. 添加云朵动画（在上层）
				//   4. Story 12.4 AC8: 仅在非首次启动时添加草动画
				// 原因：anim_idle 从物理帧 41 开始，但背景轨道在帧 41 被隐藏了（f=-1）
				//       anim_open（帧 0-12）提供背景，anim_idle（帧 41+）提供按钮动画

				// ✅ 不移除、不暂停 anim_open，让它自然停留在最后一帧（非循环动画完成后不更新）

				// ✅ 渲染顺序说明：
				//   在 Reanim 系统中，动画的添加顺序影响 CachedRenderData 的顺序
				//   但最终的视觉图层由每个轨道/图片本身的绘制顺序决定
				//
				//   理论顺序（从底到顶）：
				//   1. anim_open (背景)
				//   2. 云朵动画 (中间层)
				//   3. anim_grass (草) - 仅非首次启动
				//   4. anim_idle (按钮，最上层)

				// 1. 先添加云朵动画
				cloudAnims := []string{"anim_cloud1", "anim_cloud2", "anim_cloud4",
					"anim_cloud5", "anim_cloud6", "anim_cloud7"}

				for _, animName := range cloudAnims {
					if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, animName); err != nil {
						log.Printf("[MainMenuScene] Warning: Failed to add %s: %v", animName, err)
					}
					reanimComp.AnimationLoopStates[animName] = true
				}

				// 2. Story 12.4 AC8: 仅在非首次启动时添加 anim_grass
				// 首次启动时，草动画会在用户创建成功后手动添加
				if !m.isFirstLaunch {
					if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_grass"); err != nil {
						log.Printf("[MainMenuScene] Warning: Failed to add anim_grass: %v", err)
					}
					reanimComp.AnimationLoopStates["anim_grass"] = true
					log.Printf("[MainMenuScene] Added anim_grass (non-first launch)")
				} else {
					log.Printf("[MainMenuScene] Skipped anim_grass (first launch, will add after user creation)")
				}

				// 3. 最后添加 anim_idle（按钮应该在最上层）
				if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_idle"); err != nil {
					log.Printf("[MainMenuScene] Warning: Failed to add anim_idle: %v", err)
				}

				// 3. ✅ Epic 14: FinalizeAnimations 已集成到 AddAnimation 内部

				m.cloudAnimsResumed = true
				log.Printf("[MainMenuScene] ✅ 开场动画完成，已切换到循环模式（保留 anim_open 背景 + anim_idle + 云朵）")
			}
		}
	}

	// Story 12.3: Update help and options panels
	if m.helpPanelModule != nil {
		m.helpPanelModule.Update(deltaTime)
	}
	if m.optionsPanelModule != nil {
		m.optionsPanelModule.Update(deltaTime)
	}

	// Story 12.3: Update button system (for panel buttons)
	if m.buttonSystem != nil {
		m.buttonSystem.Update(deltaTime)
	}

	// Get mouse position (needed for both dialog and background interaction)
	mouseX, mouseY := ebiten.CursorPosition()

	// Check if mouse button is currently pressed
	isMousePressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Detect click edge (button was just pressed this frame)
	isMouseClicked := isMousePressed && !m.wasMousePressed

	// Story 12.2: 键盘快捷键触发面板（临时验证方案）
	// 检查是否有面板或对话框打开
	// ✅ Story 12.4: 同时检查 currentDialog, currentUserDialogID 和 currentErrorDialogID
	panelOpen := (m.helpPanelModule != nil && m.helpPanelModule.IsActive()) ||
		(m.optionsPanelModule != nil && m.optionsPanelModule.IsActive()) ||
		m.currentDialog != 0 ||
		m.currentUserDialogID != 0 ||
		m.currentErrorDialogID != 0

	// ✅ Story 12.4: 调试日志 - 跟踪对话框状态
	if m.currentUserDialogID != 0 || m.currentErrorDialogID != 0 {
		log.Printf("[MainMenuScene] Dialog state: panelOpen=%v, currentDialog=%d, currentUserDialogID=%d, currentErrorDialogID=%d",
			panelOpen, m.currentDialog, m.currentUserDialogID, m.currentErrorDialogID)
	}

	// 检测按键状态（用于边缘检测）
	isF1Pressed := ebiten.IsKeyPressed(ebiten.KeyF1)
	isOPressed := ebiten.IsKeyPressed(ebiten.KeyO)

	// F1 - 显示帮助面板（边缘触发）
	isF1Clicked := isF1Pressed && !m.wasF1Pressed
	if isF1Clicked && !panelOpen {
		log.Printf("[MainMenuScene] F1 key pressed, showing help panel")
		m.showHelpDialog()
	}

	// O 键 - 显示选项面板（边缘触发）
	isOClicked := isOPressed && !m.wasOPressed
	if isOClicked && !panelOpen {
		log.Printf("[MainMenuScene] O key pressed, showing options panel")
		m.showOptionsDialog()
	}

	// 更新按键状态（用于下一帧的边缘检测）
	m.wasF1Pressed = isF1Pressed
	m.wasOPressed = isOPressed

	// Story 12.3: If a panel or dialog is open, block background interaction
	if panelOpen {
		// 阻止背景交互
		m.wasMousePressed = isMousePressed

		// 对话框输入系统处理（如果有对话框）
		if m.currentDialog != 0 {
			m.dialogInputSystem.Update(deltaTime)
			m.entityManager.RemoveMarkedEntities()

			// Check if dialog was closed
			dialogStillExists := false
			dialogEntities := ecs.GetEntitiesWith1[*components.DialogComponent](m.entityManager)
			for _, entityID := range dialogEntities {
				if entityID == m.currentDialog {
					dialogStillExists = true
					break
				}
			}

			if !dialogStillExists {
				m.currentDialog = 0
				// 如果是错误对话框被关闭，也清除 currentErrorDialogID
				if m.currentErrorDialogID != 0 {
					// 检查错误对话框是否还存在
					errorDialogExists := false
					for _, entityID := range dialogEntities {
						if entityID == m.currentErrorDialogID {
							errorDialogExists = true
							break
						}
					}
					if !errorDialogExists {
						log.Printf("[MainMenuScene] Error dialog closed, clearing currentErrorDialogID")
						m.currentErrorDialogID = 0
					}
				}

				// ✅ Story 12.4: 如果还有其他对话框，将 currentDialog 设置为最上层对话框
				if len(dialogEntities) > 0 {
					// 找到 ID 最大的对话框（最上层）
					var maxDialogID ecs.EntityID = 0
					for _, entityID := range dialogEntities {
						if entityID > maxDialogID {
							maxDialogID = entityID
						}
					}
					m.currentDialog = maxDialogID
					log.Printf("[MainMenuScene] Updated currentDialog to topmost dialog (ID: %d)", maxDialogID)
				}
			}
		}

		return
	}

	// Story 12.1: Check SelectorScreen button hitboxes
	m.hoveredButton = "" // Reset hovered button

	// Get ReanimComponent to check hidden tracks
	var hiddenTracks map[string]bool
	if m.selectorScreenEntity != 0 {
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity); ok {
			hiddenTracks = reanimComp.HiddenTracks
		}
	}

	for _, hitbox := range m.buttonHitboxes {
		// 跳过被隐藏的按钮轨道
		if hiddenTracks != nil && hiddenTracks[hitbox.TrackName] {
			continue
		}

		// 使用四边形点击检测（支持旋转/倾斜按钮）
		inHitbox := config.IsPointInQuadrilateral(float64(mouseX), float64(mouseY), &hitbox)

		// 调试日志：显示每个按钮的 hitbox 信息和鼠标位置
		if hitbox.TrackName == "SelectorScreen_Challenges_button" && (inHitbox || isMouseClicked) {
			log.Printf("[MainMenuScene] 解谜按钮检测: 鼠标=(%.1f, %.1f), 四边形=[(%.1f,%.1f)-(%.1f,%.1f)-(%.1f,%.1f)-(%.1f,%.1f)], 命中=%v",
				float64(mouseX), float64(mouseY),
				hitbox.TopLeft.X, hitbox.TopLeft.Y,
				hitbox.TopRight.X, hitbox.TopRight.Y,
				hitbox.BottomRight.X, hitbox.BottomRight.Y,
				hitbox.BottomLeft.X, hitbox.BottomLeft.Y,
				inHitbox)
		}

		// Check if mouse is in hitbox
		if inHitbox {
			m.hoveredButton = hitbox.TrackName

			if isMouseClicked {
				// Button clicked
				log.Printf("[MainMenuScene] 按钮点击: %s (类型=%v)", hitbox.TrackName, hitbox.ButtonType)
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

	// Story 12.2: Update bottom function buttons (Options/Help/Quit)
	m.updateBottomButtons(mouseX, mouseY, isMouseClicked)

	// Story 12.1 Task 5: Update button highlight based on hover state
	m.updateButtonHighlight()

	// Story 12.4 Task 2.3: Update user sign hover state
	m.updateUserSignHover(mouseX, mouseY, isMouseClicked)

	// Story 12.1 Task 5: Update mouse cursor based on hover state
	m.updateMouseCursor()

	// Clean up marked entities (e.g., closed dialogs)
	m.entityManager.RemoveMarkedEntities()
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

	// Only update cursor if shape changed (避免闪烁)
	if cursorShape != m.lastCursorShape {
		ebiten.SetCursorShape(cursorShape)
		m.lastCursorShape = cursorShape
	}
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

		// Story 12.1 Task 6: 渲染关卡进度数字（在冒险模式按钮上，随动画一起移动）
		// 只在已开始游戏的用户显示关卡数字（新用户显示 StartAdventure 按钮，不需要数字）
		if m.hasStartedGame && m.currentLevel != "" {
			log.Printf("[MainMenuScene] 🔢 准备渲染关卡数字: %s", m.currentLevel)

			// 获取 ReanimComponent 以访问按钮的实时变换
			reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
			if ok {
				// 获取冒险按钮轨道的当前帧数据
				buttonTrackName := "SelectorScreen_Adventure_button"
				frames, trackExists := reanimComp.MergedTracks[buttonTrackName]

				if trackExists && len(frames) > 0 {
					// 获取当前动画的帧索引
					currentFrameIdx := reanimComp.CurrentFrame
					if currentFrameIdx < 0 {
						currentFrameIdx = 0
					}
					if currentFrameIdx >= len(frames) {
						currentFrameIdx = len(frames) - 1
					}

					if !m.levelNumbersDebugLogged {
						log.Printf("[MainMenuScene] 🔍 按钮轨道信息: 轨道=%s, 总帧数=%d, 当前帧=%d", buttonTrackName, len(frames), currentFrameIdx)
					}

					// 获取按钮当前帧的变换数据
					buttonFrame := frames[currentFrameIdx]

					// 打印帧数据（仅一次）
					frameX := 0.0
					frameY := 0.0
					if buttonFrame.X != nil {
						frameX = *buttonFrame.X
					}
					if buttonFrame.Y != nil {
						frameY = *buttonFrame.Y
					}
					if !m.levelNumbersDebugLogged {
						log.Printf("[MainMenuScene] 🔍 按钮帧数据: X=%.1f, Y=%.1f", frameX, frameY)
					}

					// 获取 PositionComponent 的基础位置
					posComp, hasPosComp := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.selectorScreenEntity)
					baseX := 0.0
					baseY := 0.0
					if hasPosComp {
						baseX = posComp.X
						baseY = posComp.Y
					}

					if !m.levelNumbersDebugLogged {
						log.Printf("[MainMenuScene] 🔍 基础位置: baseX=%.1f, baseY=%.1f, CenterOffsetX=%.1f, CenterOffsetY=%.1f",
							baseX, baseY, reanimComp.CenterOffsetX, reanimComp.CenterOffsetY)
					}

					// 计算数字渲染位置（按钮中心下方）
					// 按钮尺寸：宽 330, 高 120
					const buttonWidth = 330.0
					const buttonHeight = 120.0
					const numberOffsetX = 0.0
					const numberOffsetY = 38.0

					// 安全获取按钮位置（检查指针）
					buttonX := 0.0
					buttonY := 0.0
					if buttonFrame.X != nil {
						buttonX = *buttonFrame.X
					}
					if buttonFrame.Y != nil {
						buttonY = *buttonFrame.Y
					}

					// 按钮中心位置 = 基础位置 + 帧位置（左边缘） + 宽度的一半 - 偏移
					// buttonFrame.X 是按钮左边缘，需要加上宽度的一半得到中心
					buttonCenterX := baseX + buttonX + buttonWidth/2 - reanimComp.CenterOffsetX + numberOffsetX
					buttonCenterY := baseY + buttonY - reanimComp.CenterOffsetY + buttonHeight/2 + numberOffsetY

					// 获取按钮的倾斜角度（转换为弧度）
					// Reanim 的 SkewY 单位是度，需要转换为弧度
					// SkewY 是 Y 轴倾斜，影响左右高度（负值表示左高右低）
					angleRadians := 0.0
					if buttonFrame.SkewY != nil && *buttonFrame.SkewY != 0 {
						angleRadians = *buttonFrame.SkewY * 3.14159265359 / 180.0
						if !m.levelNumbersDebugLogged {
							log.Printf("[MainMenuScene] 🔍 使用 SkewY=%.3f度, angleRadians=%.3f弧度", *buttonFrame.SkewY, angleRadians)
						}
					} else if buttonFrame.SkewX != nil && *buttonFrame.SkewX != 0 {
						// 如果 SkewY 为 0，尝试使用 SkewX
						angleRadians = *buttonFrame.SkewX * 3.14159265359 / 180.0
						if !m.levelNumbersDebugLogged {
							log.Printf("[MainMenuScene] 🔍 使用 SkewX=%.3f度, angleRadians=%.3f弧度", *buttonFrame.SkewX, angleRadians)
						}
					} else {
						// Reanim 中无倾斜角度，使用固定倾斜（左高右低，约 5 度）
						angleRadians = 5.0 * 3.14159265359 / 180.0
						if !m.levelNumbersDebugLogged {
							log.Printf("[MainMenuScene] 🔍 Reanim 无倾斜，使用固定角度 -3 度, angleRadians=%.3f弧度", angleRadians)
						}
					}
					if !m.levelNumbersDebugLogged {
						m.levelNumbersDebugLogged = true
					}

					// 渲染关卡进度数字（应用倾斜角度）
					renderLevelNumbers(screen, m.resourceManager, m.currentLevel, buttonCenterX, buttonCenterY, angleRadians)
				} else {
					log.Printf("[MainMenuScene] ⚠️ 未找到按钮轨道或帧数据: %s", buttonTrackName)
				}
			} else {
				log.Println("[MainMenuScene] ⚠️ 未找到 ReanimComponent")
			}
		} else {
			log.Println("[MainMenuScene] ⚠️ currentLevel 为空，不渲染数字")
		}

		// Story 12.4 Task 2.4: 渲染木牌上的用户名文本
		m.renderUserSignText(screen)

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

	// Story 12.2: Draw bottom function buttons (Options/Help/Quit)
	m.drawBottomButtons(screen)

	// Story 12.3: Draw dialogs (last, on top of everything)
	// ✅ Story 12.4: DialogRenderSystem 现在也负责渲染对话框的子实体（输入框）
	// 这样确保输入框跟随父对话框的z-order，不会总是显示在最上层
	if m.dialogRenderSystem != nil {
		m.dialogRenderSystem.Draw(screen)
	}

	// Story 12.3: Draw help and options panels (above dialogs)
	if m.helpPanelModule != nil {
		m.helpPanelModule.Draw(screen)
	}
	if m.optionsPanelModule != nil {
		m.optionsPanelModule.Draw(screen)
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

	// Story 12.1 Task 6: 首次点击"开始冒险吧"时，标记用户已开始游戏
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	if err := saveManager.Load(); err == nil {
		if !saveManager.GetHasStartedGame() {
			log.Println("[MainMenuScene] 首次开始游戏，设置 hasStartedGame = true")
			saveManager.SetHasStartedGame()
			if err := saveManager.Save(); err != nil {
				log.Printf("[MainMenuScene] ⚠️ 保存 hasStartedGame 失败: %v", err)
			}
		}
	}

	// Story 8.6: Load level from save file or default to 1-1
	levelToLoad := "1-1" // Default to first level
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

		// Play button click sound (shadow buttons also have click feedback)
		player, err := m.resourceManager.LoadSoundEffect("assets/sounds/buttonclick.ogg")
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to load button click sound: %v", err)
		} else {
			player.Rewind()
			player.Play()
		}

		// Story 12.3: Show unlock dialog
		message := getUnlockMessage(buttonType)
		m.showUnlockDialog("未解锁！", message)
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

// showUnlockDialog displays a dialog with a title and message
// Story 12.3: Dialog System Implementation
func (m *MainMenuScene) showUnlockDialog(title, message string) {
	// Close existing dialog (if any)
	if m.currentDialog != 0 {
		m.entityManager.DestroyEntity(m.currentDialog)
		m.currentDialog = 0
	}

	// Create new dialog
	dialogEntity, err := entities.NewDialogEntity(
		m.entityManager,
		m.resourceManager,
		title,
		message,
		[]string{"确定"},
		WindowWidth,
		WindowHeight,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to create dialog: %v", err)
		return
	}

	m.currentDialog = dialogEntity
	log.Printf("[MainMenuScene] Dialog created: %s - %s", title, message)
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

// showHelpDialog 显示帮助面板
// Story 12.3: 使用帮助面板模块（便笺背景 + 帮助文本）
func (m *MainMenuScene) showHelpDialog() {
	if m.helpPanelModule != nil {
		m.helpPanelModule.Show()
		log.Printf("[MainMenuScene] Help panel shown")
	}
}

// showOptionsDialog 显示选项面板
// Story 12.3: 使用选项面板模块（复用游戏场景的暂停菜单样式）
func (m *MainMenuScene) showOptionsDialog() {
	if m.optionsPanelModule != nil {
		m.optionsPanelModule.Show()
		log.Printf("[MainMenuScene] Options panel shown")
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
func (m *MainMenuScene) updateBottomButtons(mouseX, mouseY int, isMouseClicked bool) {
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
			if isMouseClicked {
				// Button clicked
				m.onBottomButtonClicked(btnType)
			} else {
				// Button hovered
				m.hoveredBottomButton = btnType
			}
			break // Only one button can be hovered at a time
		}
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
	// Play click sound effect
	if player, err := m.resourceManager.LoadSoundEffect("assets/sounds/buttonclick.ogg"); err == nil {
		player.Play()
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
		os.Exit(0)
	}
}

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
				// 验证失败时会显示错误对话框，但���关闭新用户对话框
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
}

// onNewUserCreated 处理新用户创建成功的回调
func (m *MainMenuScene) onNewUserCreated(username string) {
	log.Printf("[MainMenuScene] Creating new user: %s", username)

	gameState := game.GetGameState()
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

	// 更新按钮可见性
	m.updateButtonVisibility()

	// Story 12.4: 初始化木牌（显示用户名）
	m.initUserSign()

	// 更新标志（不再是首次启动）
	wasFirstLaunch := m.isFirstLaunch
	m.isFirstLaunch = false

	// Story 12.4 AC8: 创建成功后，播放 anim_sign + anim_grass
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
}

// showErrorDialog 显示错误提示对话框
// 注意：错误对话框不会影响 currentDialog/currentUserDialogID 的跟踪
// 这样错误对话框关闭后，新用户对话框仍然保持打开状态
// Story 12.4: 防止错误对话框叠加 - 同一时间只能有一个错误对话框
func (m *MainMenuScene) showErrorDialog(title, message string) {
	// ✅ 如果已有错误对话框，先销毁旧的
	if m.currentErrorDialogID != 0 {
		log.Printf("[MainMenuScene] Destroying old error dialog (entity ID: %d)", m.currentErrorDialogID)
		// 如果 currentDialog 指向错误对话框，也清除
		if m.currentDialog == m.currentErrorDialogID {
			m.currentDialog = 0
		}
		m.entityManager.DestroyEntity(m.currentErrorDialogID)
		m.currentErrorDialogID = 0
	}

	dialogID, err := entities.NewDialogEntity(
		m.entityManager,
		m.resourceManager,
		title,
		message,
		[]string{"确定"},
		WindowWidth,
		WindowHeight,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create error dialog: %v", err)
		return
	}

	// ✅ 记录错误对话框ID，用于下次创建时销毁
	m.currentErrorDialogID = dialogID
	m.currentDialog = dialogID // 设置为当前对话框，触发背景交互阻止
	log.Printf("[MainMenuScene] Error dialog created (entity ID: %d)", dialogID)
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
// 在原始木牌图片上绘制用户名文本（白字黄边，40号字体）
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

	// 绘制黄色描边
	yellowColor := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	drawTextOutlineOnImage(newImage, username, centerX, centerY, usernameFont, yellowColor, 1)

	// 绘制白色文本
	whiteColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	drawCenteredTextOnImage(newImage, username, centerX, centerY, usernameFont, whiteColor)

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

// drawTextOutlineOnImage 在图片上绘制文本描边
func drawTextOutlineOnImage(img *ebiten.Image, textStr string, centerX, centerY float64, fontFace *text.GoTextFace, outlineColor color.Color, thickness int) {
	textWidth, _ := text.Measure(textStr, fontFace, 0)
	baseX := centerX - textWidth/2
	baseY := centerY

	// 绘制描边：在 8 个方向偏移绘制
	offsets := []struct{ dx, dy float64 }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for _, offset := range offsets {
		for t := 1; t <= thickness; t++ {
			op := &text.DrawOptions{}
			op.GeoM.Translate(baseX+offset.dx*float64(t), baseY+offset.dy*float64(t))
			op.ColorScale.ScaleWithColor(outlineColor)
			text.Draw(img, textStr, fontFace, op)
		}
	}
}

// updateUserSignHover 更新木牌悬停状态和点击检测
// Story 12.4 Task 2.3
func (m *MainMenuScene) updateUserSignHover(mouseX, mouseY int, isMouseClicked bool) {
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

	// 获取当前帧索引
	currentFrameIdx := reanimComp.CurrentFrame
	if currentFrameIdx < 0 {
		currentFrameIdx = 0
	}
	if currentFrameIdx >= len(frames) {
		currentFrameIdx = len(frames) - 1
	}

	// 获取当前帧的变换数据
	frame := frames[currentFrameIdx]

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
	if mouseInSign && isMouseClicked {
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
	log.Printf("[MainMenuScene] User management dialog opened")
}

// onUserManagementAction 用户管理对话框的操作回调
// Story 12.4 AC4, AC9
func (m *MainMenuScene) onUserManagementAction(result entities.UserManagementDialogResult) {
	switch result.Action {
	case entities.UserActionSwitch:
		// 切换用户
		if result.SelectedUser != "" {
			if err := m.saveManager.SwitchUser(result.SelectedUser); err != nil {
				log.Printf("[MainMenuScene] Error: Failed to switch user: %v", err)
				m.showErrorDialog("切换失败", "无法切换到用户: "+result.SelectedUser)
				return
			}
			log.Printf("[MainMenuScene] Switched to user: %s", result.SelectedUser)
			// 重新加载主菜单数据
			m.reloadMainMenuData()
			// 关闭对话框
			m.closeCurrentDialog()
		}

	case entities.UserActionCreateNew:
		// 显示新建用户对话框
		m.closeCurrentDialog()
		m.showNewUserDialog(false) // force=false，可以关闭

	case entities.UserActionRename:
		// 显示重命名对话框
		if result.SelectedUser != "" {
			m.closeCurrentDialog()
			m.showRenameUserDialog(result.SelectedUser)
		}

	case entities.UserActionDelete:
		// 显示删除确认对话框
		if result.SelectedUser != "" {
			m.closeCurrentDialog()
			m.showDeleteUserDialog(result.SelectedUser)
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

	// 更新木牌显示的用户名
	if m.userSignEntity != 0 {
		if userSignComp, ok := ecs.GetComponent[*components.UserSignComponent](m.entityManager, m.userSignEntity); ok {
			userSignComp.CurrentUsername = m.saveManager.GetCurrentUser()
			log.Printf("[MainMenuScene] Updated user sign to: %s", userSignComp.CurrentUsername)
		}
	}
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
}

// showRenameUserDialog 显示重命名用户对话框
// Story 12.4 AC6
func (m *MainMenuScene) showRenameUserDialog(oldUsername string) {
	// 关闭现有对话框
	if m.currentUserDialogID != 0 {
		m.closeCurrentDialog()
	}

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
			m.closeCurrentDialog()
			m.reloadMainMenuData()
		} else {
			// 取消
			m.closeCurrentDialog()
		}
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

	m.currentUserDialogID = dialogID
	m.currentInputBoxID = inputBoxID
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] Rename user dialog opened for: %s", oldUsername)
}

// showDeleteUserDialog 显示删除用户确认对话框
// Story 12.4 AC7
func (m *MainMenuScene) showDeleteUserDialog(username string) {
	// 关闭现有对话框
	if m.currentUserDialogID != 0 {
		m.closeCurrentDialog()
	}

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
			m.closeCurrentDialog()

			// 检查是否还有用户
			users, err := m.saveManager.LoadUserList()
			if err != nil || len(users) == 0 {
				// 没有用户了，回到首次启动状态
				m.isFirstLaunch = true
				m.userSignEntity = 0
				m.showNewUserDialog(true) // 强制创建新用户
				return
			}

			// 重新加载数据
			m.reloadMainMenuData()
		} else {
			// 取消
			m.closeCurrentDialog()
		}
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

	m.currentUserDialogID = dialogID
	m.currentDialog = dialogID
	log.Printf("[MainMenuScene] Delete user dialog opened for: %s", username)
}

// renderUserSignText 渲染木牌上的用户名文本
// Story 12.4 Task 2.4
// 新方案：用户名已预先绘制到木牌图片上，这里不需要单独渲染
// 保留此函数用于未来可能的悬停效果（如更换图片）
func (m *MainMenuScene) renderUserSignText(screen *ebiten.Image) {
	// 用户名已预先绘制到木牌图片上，随 Reanim 动画自然移动
	// 此函数暂时为空，保留用于未来扩展
}

// drawCenteredText 在指定位置居中绘制文本
func drawCenteredText(screen *ebiten.Image, textStr string, centerX, centerY float64, fontFace *text.GoTextFace, clr color.Color) {
	// 使用 text.Measure 计算文本宽度
	textWidth, _ := text.Measure(textStr, fontFace, 0)

	x := centerX - textWidth/2
	y := centerY

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, textStr, fontFace, op)
}

// drawTextOutline 绘制文本描边（用于白字黄边效果）
func drawTextOutline(screen *ebiten.Image, textStr string, centerX, centerY float64, fontFace *text.GoTextFace, outlineColor color.Color, thickness int) {
	// 使用 text.Measure 计算文本宽度
	textWidth, _ := text.Measure(textStr, fontFace, 0)
	baseX := centerX - textWidth/2
	baseY := centerY

	// 绘制描边：在 8 个方向偏移绘制
	offsets := []struct{ dx, dy float64 }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for _, offset := range offsets {
		for t := 1; t <= thickness; t++ {
			op := &text.DrawOptions{}
			op.GeoM.Translate(baseX+offset.dx*float64(t), baseY+offset.dy*float64(t))
			op.ColorScale.ScaleWithColor(outlineColor)
			text.Draw(screen, textStr, fontFace, op)
		}
	}
}

// getTrackNames 获取 MergedTracks 中的所有轨道名称（用于调试）
func getTrackNames(tracks map[string][]reanim.Frame) []string {
	names := make([]string, 0, len(tracks))
	for name := range tracks {
		names = append(names, name)
	}
	return names
}

// getPartImageKeys 获取 PartImages 中的所有键（用于调试）
func getPartImageKeys(images map[string]*ebiten.Image) []string {
	keys := make([]string, 0, len(images))
	for key := range images {
		keys = append(keys, key)
	}
	return keys
}
