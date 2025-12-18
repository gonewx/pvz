package scenes

// 主菜单场景核心逻辑
// 拆分文件：
// - main_menu_buttons.go: 按钮系统 (高亮、可见性、点击处理、底部按钮栏)
// - main_menu_user_ui.go: 用户管理UI (用户名木牌、用户管理对话框)
// - main_menu_dialogs.go: 对话框系统 (解锁对话框、帮助/选项面板、错误提示、战斗存档对话框)
// - main_menu_zombie_hand.go: 僵尸手动画

import (
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/modules"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// WindowWidth is the logical width of the game window in pixels.
	WindowWidth = 800
	// WindowHeight is the logical height of the game window in pixels.
	WindowHeight = 600
)

// MainMenuState represents the state of the main menu scene
type MainMenuState int

const (
	MainMenuStateNormal            MainMenuState = iota // Normal state
	MainMenuStateZombieHandPlaying                      // Zombie hand animation playing
)

// MainMenuScene represents the main menu screen of the game.
// It displays when the game starts and allows the player to navigate to other scenes.
type MainMenuScene struct {
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager
	backgroundImage *ebiten.Image
	bgmStarted      bool // 背景音乐是否已启动
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
	sliderSystem       *systems.SliderSystem       // Slider interaction system (for options panel)
	checkboxSystem     *systems.CheckboxSystem     // Checkbox interaction system (for options panel)
	helpPanelModule    *modules.HelpPanelModule    // Help panel module
	optionsPanelModule *modules.OptionsPanelModule // Options panel module

	// Story 12.2: Bottom function bar (Options/Help/Quit buttons)
	bottomButtonImages      map[components.BottomButtonType][2]*ebiten.Image // [0]=Normal, [1]=Hover
	hoveredBottomButton     components.BottomButtonType                      // Current hovered bottom button (-1 = none)
	lastHoveredBottomButton components.BottomButtonType                      // Story 10.9: 追踪上一帧的悬停状态（用于播放音效）

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

	// Story 12.6: Zombie hand transition animation
	zombieHandEntity ecs.EntityID  // Zombie hand entity ID
	menuState        MainMenuState // Main menu state
	pendingScene     string        // Pending scene to switch to after animation

	// Story 10.9: 延迟播放音效
	pendingSoundDelay float64 // 延迟时间（秒），0表示无待播放音效
	pendingSoundID    string  // 待播放的音效ID

	// Story 21.4: 移动端虚拟键盘
	virtualKeyboardEntity       ecs.EntityID                         // 虚拟键盘实体
	virtualKeyboardSystem       *systems.VirtualKeyboardSystem       // 虚拟键盘系统
	virtualKeyboardRenderSystem *systems.VirtualKeyboardRenderSystem // 虚拟键盘渲染系统
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
		resourceManager:         rm,
		sceneManager:            sm,
		lastCursorShape:         -1, // 初始化为无效值，确保第一次更新光标
		hoveredBottomButton:     components.BottomButtonNone,
		lastHoveredBottomButton: components.BottomButtonNone,       // Story 10.9: 初始化底部按钮悬停追踪
		wasMousePressed:         utils.IsPointerPressed(),          // ✅ 初始化指针状态，支持触摸和鼠标
		wasF1Pressed:            ebiten.IsKeyPressed(ebiten.KeyF1), // ✅ 初始化键盘状态
		wasOPressed:             ebiten.IsKeyPressed(ebiten.KeyO),  // ✅ 初始化键盘状态
		menuState:               MainMenuStateNormal,               // Story 12.6: 初始化为正常状态
	}

	// Story 12.1: Initialize ECS systems for SelectorScreen Reanim
	scene.entityManager = ecs.NewEntityManager()
	scene.reanimSystem = systems.NewReanimSystem(scene.entityManager)

	// Story 13.6: 设置配置管理器
	if configManager := rm.GetReanimConfigManager(); configManager != nil {
		scene.reanimSystem.SetConfigManager(configManager)
	}

	// Story 5.4.1: 设置资源加载器，用于运行时单位切换
	scene.reanimSystem.SetResourceLoader(rm)

	scene.renderSystem = systems.NewRenderSystem(scene.entityManager)
	// ✅ 修复：设置 ReanimSystem 引用，以便 RenderSystem 调用 GetRenderData()
	scene.renderSystem.SetReanimSystem(scene.reanimSystem)
	log.Printf("[MainMenuScene] Initialized ECS systems")

	// 加载主菜单需要的音效资源组（包含 SOUND_EVILLAUGH 等）
	if err := rm.LoadResourceGroup("LoadingSounds"); err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load LoadingSounds group: %v", err)
	}

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
			scene.hasStartedGame = saveManager.GetHasStartedGame()

			// Bug Fix: 优先使用战斗存档中的 LevelID（与 onStartAdventureClicked 保持一致）
			// 如果有战斗存档，必须使用存档中的关卡ID，否则会导致主菜单显示与实际加载的关卡不匹配
			scene.currentLevel = ""
			currentUser := saveManager.GetCurrentUser()
			if currentUser != "" && saveManager.HasBattleSave(currentUser) {
				if battleInfo, err := saveManager.GetBattleSaveInfo(currentUser); err == nil && battleInfo != nil {
					scene.currentLevel = battleInfo.LevelID
					log.Printf("[MainMenuScene] Found battle save for level %s, using it for display", scene.currentLevel)
				}
			}

			// 如果没有战斗存档，使用 GetNextLevelToPlay
			if scene.currentLevel == "" {
				scene.currentLevel = saveManager.GetNextLevelToPlay()
				log.Printf("[MainMenuScene] No battle save, using next level: %s (highest completed: %s)",
					scene.currentLevel, saveManager.GetHighestLevel())
			}

			log.Printf("[MainMenuScene] Display level: %s, hasStartedGame: %v", scene.currentLevel, scene.hasStartedGame)
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

		// Story 12.4 AC8: **关键修复**：在播放动画之前先设置 HiddenTracks
		// 这样首次渲染就不会显示木牌和草叶子
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](scene.entityManager, selectorEntity)
		if ok && scene.isFirstLaunch {
			reanimComp.HiddenTracks = make(map[string]bool)
			// 隐藏木牌轨道
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

		// 播放泥土松动音效（开场动画开始时）
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_DIRT_RISE")
		}

		// 设置木牌滚入音效延迟播放
		// anim_sign 从第13帧开始（约0.65秒后），需要延迟播放
		scene.pendingSoundDelay = 0.65
		scene.pendingSoundID = "SOUND_ROLL_IN"

		// 处理 AnimationCommand（立即初始化动画）
		scene.reanimSystem.Update(0)

		// 修复：SelectorScreen 是全屏 UI，应该使用左上角对齐（Reanim 原始坐标）
		// 而不是中心对齐。禁用 CenterOffset 功能。
		reanimComp, ok = ecs.GetComponent[*components.ReanimComponent](scene.entityManager, selectorEntity)
		if ok {
			reanimComp.CenterOffsetX = 0
			reanimComp.CenterOffsetY = 0
			log.Printf("[MainMenuScene] SelectorScreen 使用左上角对齐（CenterOffset = 0）")

			// 首次启动时，设置开场动画为非循环模式
			// PlayAnimation 默认设置 IsLooping = true，需要手动覆盖
			if scene.isFirstLaunch {
				reanimComp.IsLooping = false
				log.Printf("[MainMenuScene] First launch: set anim_open to non-looping mode")
			}
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

	// 背景音乐由 AudioManager 统一管理（Story 10.9）
	// 音乐将在 Update() 中首次播放

	// Story 12.3: Initialize dialog systems
	// 加载不同大小的字体用于对话框渲染
	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 22)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load dialog title font: %v", err)
	}

	messageFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 18)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load dialog message font: %v", err)
	}

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
	scene.sliderSystem = systems.NewSliderSystem(scene.entityManager)
	scene.checkboxSystem = systems.NewCheckboxSystem(scene.entityManager)

	// Story 12.3: Initialize help panel module
	helpPanel, err := modules.NewHelpPanelModule(
		scene.entityManager,
		rm,
		func(screen *ebiten.Image, buttonEntity ecs.EntityID) {
			scene.buttonRenderSystem.DrawButton(screen, buttonEntity)
		},
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
	// Story 20.5: 从 GameState 获取 SettingsManager，复用已实现的设置保存逻辑
	optionsPanel, err := modules.NewOptionsPanelModule(
		scene.entityManager,
		rm,
		scene.buttonSystem,
		scene.buttonRenderSystem,
		gameState.GetSettingsManager(), // 复用 SettingsManager，支持全屏设置保存
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
	inputFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load input font: %v", err)
	}
	scene.textInputRenderSystem = systems.NewTextInputRenderSystem(scene.entityManager, inputFont)
	log.Printf("[MainMenuScene] Text input systems initialized")

	// ✅ Story 12.4: 设置 DialogRenderSystem 的 TextInputRenderSystem 引用
	scene.dialogRenderSystem.SetTextInputRenderSystem(scene.textInputRenderSystem)
	log.Printf("[MainMenuScene] Set TextInputRenderSystem reference in DialogRenderSystem")

	// Story 12.6: Create zombie hand entity (initially paused, for transition animation)
	zombieHandEntity, err := entities.NewZombieHandEntity(
		scene.entityManager,
		rm,
		config.ZombieHandOffsetX, // 水平偏移（正值向右）
		config.ZombieHandOffsetY, // 垂直偏移（正值向下）
	)
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to create zombie hand entity: %v", err)
		scene.zombieHandEntity = 0
	} else {
		scene.zombieHandEntity = zombieHandEntity
		// Mark as UI element (not affected by camera)
		ecs.AddComponent(scene.entityManager, zombieHandEntity, &components.UIComponent{})
		log.Printf("[MainMenuScene] Zombie hand entity created (ID=%d, offset=(%.1f, %.1f))",
			zombieHandEntity, config.ZombieHandOffsetX, config.ZombieHandOffsetY)
	}

	// Story 21.4: 初始化虚拟键盘（仅移动端）
	if utils.IsMobile() {
		kbEntity, err := entities.NewVirtualKeyboardEntity(
			scene.entityManager,
			rm,
			WindowWidth,
			WindowHeight,
		)
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to create virtual keyboard: %v", err)
		} else {
			scene.virtualKeyboardEntity = kbEntity
			scene.virtualKeyboardSystem = systems.NewVirtualKeyboardSystem(scene.entityManager)
			scene.virtualKeyboardRenderSystem = systems.NewVirtualKeyboardRenderSystem(scene.entityManager, rm)
			log.Printf("[MainMenuScene] Virtual keyboard initialized for mobile (ID=%d)", kbEntity)
		}
	}

	return scene
}

// Update updates the main menu scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (m *MainMenuScene) Update(deltaTime float64) {
	// Debug: Check for GameFreezeComponent
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](m.entityManager)
	if len(freezeEntities) > 0 {
		log.Printf("[MainMenuScene] ⚠️  WARNING: Found %d GameFreezeComponent entities! This should not happen in MainMenu.", len(freezeEntities))
	}

	// 清理上一帧标记删除的实体（确保本帧开始前已删除）
	m.entityManager.RemoveMarkedEntities()

	// Story 10.9: 处理延迟播放音效
	if m.pendingSoundDelay > 0 {
		m.pendingSoundDelay -= deltaTime
		if m.pendingSoundDelay <= 0 {
			// 延迟时间到，播放音效
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound(m.pendingSoundID)
			}
			m.pendingSoundDelay = 0
			m.pendingSoundID = ""
		}
	}

	// Story 12.4: Check for first launch and show new user dialog
	if m.isFirstLaunch && !m.newUserDialogShown {
		m.showNewUserDialogForFirstLaunch()
		m.newUserDialogShown = true
	}

	// Story 12.4: Update text input system (for user dialogs)
	if m.textInputSystem != nil {
		m.textInputSystem.Update(deltaTime)
	}

	// Story 21.4: 更新虚拟键盘系统（移动端）
	if m.virtualKeyboardSystem != nil {
		m.virtualKeyboardSystem.Update(deltaTime)
	}

	// 确保背景音乐正在播放（使用 AudioManager 统一管理 - Story 10.9）
	if !m.bgmStarted {
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlayMusic("SOUND_TITLESCREEN")
			m.bgmStarted = true
		}
	}

	// Story 12.1: Update Reanim system (animate clouds, flowers, etc.)
	if m.reanimSystem != nil {
		m.reanimSystem.Update(deltaTime)

		// Story 12.6: Check if zombie hand animation finished
		if m.menuState == MainMenuStateZombieHandPlaying {
			m.checkZombieHandAnimationFinished()
		}

		// ✅ 检测开场动画完成，切换到循环动画
		if !m.cloudAnimsResumed && m.selectorScreenEntity != 0 {
			reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
			if ok && reanimComp.IsFinished {
				// 初始化 AnimationLoopStates（如果需要）
				if reanimComp.AnimationLoopStates == nil {
					reanimComp.AnimationLoopStates = make(map[string]bool)
				}

				// 开场动画已完成，添加循环动画
				cloudAnims := []string{"anim_cloud1", "anim_cloud2", "anim_cloud4",
					"anim_cloud5", "anim_cloud6", "anim_cloud7"}

				for _, animName := range cloudAnims {
					if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, animName); err != nil {
						log.Printf("[MainMenuScene] Warning: Failed to add %s: %v", animName, err)
					}
					reanimComp.AnimationLoopStates[animName] = true
				}

				// Story 12.4 AC8: 仅在非首次启动时添加 anim_grass
				if !m.isFirstLaunch {
					if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_grass"); err != nil {
						log.Printf("[MainMenuScene] Warning: Failed to add anim_grass: %v", err)
					}
					reanimComp.AnimationLoopStates["anim_grass"] = true
					log.Printf("[MainMenuScene] Added anim_grass (non-first launch)")
				} else {
					log.Printf("[MainMenuScene] Skipped anim_grass (first launch, will add after user creation)")
				}

				// 最后添加 anim_idle（按钮应该在最上层）
				if err := m.reanimSystem.AddAnimation(m.selectorScreenEntity, "anim_idle"); err != nil {
					log.Printf("[MainMenuScene] Warning: Failed to add anim_idle: %v", err)
				}

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

	// 选项面板激活时更新滑块和复选框系统（复用游戏场景的交互逻辑）
	if m.optionsPanelModule != nil && m.optionsPanelModule.IsActive() {
		if m.sliderSystem != nil {
			m.sliderSystem.Update(deltaTime)
		}
		if m.checkboxSystem != nil {
			m.checkboxSystem.Update(deltaTime)
		}
	}

	// 更新最后触摸位置（用于触摸释放时获取位置）
	utils.UpdateLastTouchPosition()

	// Get pointer position (supports both mouse and touch)
	mouseX, mouseY := utils.GetPointerPosition()

	// Check if pointer is currently pressed (mouse or touch)
	isMousePressed := utils.IsPointerPressed()

	// ✅ 检测指针释放（支持触摸和鼠标）
	pointerReleased, releaseX, releaseY := utils.IsPointerJustReleased()
	// 如果有释放事件，使用释放位置；否则使用当前指针位置
	if pointerReleased {
		mouseX, mouseY = releaseX, releaseY
	}
	isMouseReleased := pointerReleased

	// Story 12.2: 键盘快捷键触发面板（临时验证方案）
	// 检查是否有面板或对话框打开
	panelOpen := (m.helpPanelModule != nil && m.helpPanelModule.IsActive()) ||
		(m.optionsPanelModule != nil && m.optionsPanelModule.IsActive()) ||
		m.currentDialog != 0 ||
		m.currentUserDialogID != 0 ||
		m.currentErrorDialogID != 0

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

		// ✅ ECS 架构修复: 对所有对话框都调用 DialogInputSystem.Update()
		if m.currentDialog != 0 || m.currentUserDialogID != 0 || m.currentErrorDialogID != 0 {
			m.dialogInputSystem.Update(deltaTime)
			m.entityManager.RemoveMarkedEntities()

			// Check if dialog was closed
			dialogEntities := ecs.GetEntitiesWith1[*components.DialogComponent](m.entityManager)

			// 检查 currentDialog 是否还存在
			if m.currentDialog != 0 {
				dialogStillExists := false
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

			// 检查 currentUserDialogID 是否还存在
			if m.currentUserDialogID != 0 {
				userDialogExists := false
				for _, entityID := range dialogEntities {
					if entityID == m.currentUserDialogID {
						userDialogExists = true
						break
					}
				}
				if !userDialogExists {
					log.Printf("[MainMenuScene] User dialog closed, clearing currentUserDialogID")
					m.currentUserDialogID = 0
				}
			}
		}

		// Story 12.4: Update mouse cursor for dialog buttons and list items
		m.updateMouseCursor()
		return
	}

	// Story 12.6 Task 2.6: Block all button interactions during zombie hand animation
	if m.menuState == MainMenuStateZombieHandPlaying {
		m.hoveredButton = ""
		m.hoveredBottomButton = components.BottomButtonNone
		m.wasMousePressed = isMousePressed
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

		// Check if mouse is in hitbox
		if inHitbox {
			m.hoveredButton = hitbox.TrackName

			if isMouseReleased {
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
			if isMouseReleased {
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
	m.updateBottomButtons(mouseX, mouseY, isMouseReleased)

	// Story 12.1 Task 5: Update button highlight based on hover state
	m.updateButtonHighlight()

	// Story 12.4 Task 2.3: Update user sign hover state
	hasOpenDialog := m.currentUserDialogID != 0 || m.currentDialog != 0 || m.currentErrorDialogID != 0
	if !hasOpenDialog {
		m.updateUserSignHover(mouseX, mouseY, isMouseReleased)
	} else {
		// 对话框打开时，强制重置木牌悬停状态
		if m.userSignEntity != 0 {
			if userSignComp, ok := ecs.GetComponent[*components.UserSignComponent](m.entityManager, m.userSignEntity); ok {
				userSignComp.IsHovered = false
			}
		}
	}

	// Story 12.1 Task 5: Update mouse cursor based on hover state
	m.updateMouseCursor()

	// Clean up marked entities (e.g., closed dialogs)
	m.entityManager.RemoveMarkedEntities()
}

// Draw renders the main menu scene to the screen.
// If a background image is loaded, it draws the image.
// Otherwise, it uses a dark blue fallback background.
func (m *MainMenuScene) Draw(screen *ebiten.Image) {
	// Story 12.6: Debug menu state
	if m.zombieHandEntity != 0 {
		log.Printf("[MainMenuScene] 🎨 Draw() called: menuState=%d", m.menuState)
	}

	// Story 12.1: Draw SelectorScreen Reanim (contains background, buttons, decorations)
	if m.selectorScreenEntity != 0 {
		// 主菜单使用 Reanim 渲染，直接调用 DrawEntity
		m.renderSystem.DrawEntity(screen, m.selectorScreenEntity, 0)

		// Story 12.1 Task 6: 渲染关卡进度数字
		if m.hasStartedGame && m.currentLevel != "" {
			log.Printf("[MainMenuScene] 🔢 准备渲染关卡数字: %s", m.currentLevel)

			reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
			if ok {
				buttonTrackName := "SelectorScreen_Adventure_button"
				frames, trackExists := reanimComp.MergedTracks[buttonTrackName]

				if trackExists && len(frames) > 0 {
					currentFrameIdx := reanimComp.CurrentFrame
					if currentFrameIdx < 0 {
						currentFrameIdx = 0
					}
					if currentFrameIdx >= len(frames) {
						currentFrameIdx = len(frames) - 1
					}

					buttonFrame := frames[currentFrameIdx]

					posComp, hasPosComp := ecs.GetComponent[*components.PositionComponent](m.entityManager, m.selectorScreenEntity)
					baseX := 0.0
					baseY := 0.0
					if hasPosComp {
						baseX = posComp.X
						baseY = posComp.Y
					}

					const buttonWidth = 330.0
					const buttonHeight = 120.0
					const numberOffsetX = 0.0
					const numberOffsetY = 38.0

					buttonX := 0.0
					buttonY := 0.0
					if buttonFrame.X != nil {
						buttonX = *buttonFrame.X
					}
					if buttonFrame.Y != nil {
						buttonY = *buttonFrame.Y
					}

					buttonCenterX := baseX + buttonX + buttonWidth/2 - reanimComp.CenterOffsetX + numberOffsetX
					buttonCenterY := baseY + buttonY - reanimComp.CenterOffsetY + buttonHeight/2 + numberOffsetY

					angleRadians := 0.0
					if buttonFrame.SkewY != nil && *buttonFrame.SkewY != 0 {
						angleRadians = *buttonFrame.SkewY * 3.14159265359 / 180.0
					} else if buttonFrame.SkewX != nil && *buttonFrame.SkewX != 0 {
						angleRadians = *buttonFrame.SkewX * 3.14159265359 / 180.0
					} else {
						angleRadians = 5.0 * 3.14159265359 / 180.0
					}
					if !m.levelNumbersDebugLogged {
						m.levelNumbersDebugLogged = true
					}

					renderLevelNumbers(screen, m.resourceManager, m.currentLevel, buttonCenterX, buttonCenterY, angleRadians)
				}
			}
		}

		// Story 12.4 Task 2.4: 渲染木牌上的用户名文本
		m.renderUserSignText(screen)
	} else {
		// Fallback: Draw background image if SelectorScreen failed to load
		if m.backgroundImage != nil {
			bounds := m.backgroundImage.Bounds()
			bgWidth := float64(bounds.Dx())
			bgHeight := float64(bounds.Dy())

			scaleX := WindowWidth / bgWidth
			scaleY := WindowHeight / bgHeight

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scaleX, scaleY)
			screen.DrawImage(m.backgroundImage, op)
		} else {
			// Fallback: Fill the screen with a dark blue color (midnight blue)
			screen.Fill(color.RGBA{R: 25, G: 25, B: 112, A: 255})
		}

		// Fallback: Draw old-style buttons only if Reanim failed to load
		for _, btn := range m.buttons {
			if btn.NormalImage == nil {
				continue
			}

			var img *ebiten.Image
			if btn.State == components.UIHovered && btn.HoverImage != nil {
				img = btn.HoverImage
			} else {
				img = btn.NormalImage
			}

			op := &ebiten.DrawImageOptions{}

			if btn.State == components.UIHovered && btn.HoverImage == nil {
				op.ColorScale.Scale(1.2, 1.2, 1.2, 1.0)
			}

			op.GeoM.Translate(btn.X, btn.Y)
			screen.DrawImage(img, op)
		}
	}

	// Story 12.2: Draw bottom function buttons (Options/Help/Quit)
	m.drawBottomButtons(screen)

	// Story 12.6: Draw zombie hand animation (if playing)
	if m.menuState == MainMenuStateZombieHandPlaying && m.zombieHandEntity != 0 {
		log.Printf("[MainMenuScene] 🧟 Drawing zombie hand entity (ID=%d)", m.zombieHandEntity)
		m.renderSystem.DrawEntity(screen, m.zombieHandEntity, 0)
	} else {
		if m.zombieHandEntity != 0 {
			log.Printf("[MainMenuScene] 🧟 NOT drawing zombie hand: menuState=%d (expected %d)",
				m.menuState, MainMenuStateZombieHandPlaying)
		}
	}

	// Story 12.3: Draw dialogs (last, on top of everything)
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

	// Story 21.4: 渲染虚拟键盘（最上层）
	if m.virtualKeyboardRenderSystem != nil {
		m.virtualKeyboardRenderSystem.Draw(screen)
	}
}
