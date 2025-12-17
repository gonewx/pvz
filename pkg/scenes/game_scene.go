package scenes

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/modules"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// GameScene represents the main gameplay screen.
// This is where the actual Plants vs Zombies gameplay will occur.
// It manages the game state, UI elements, and the ECS system.
//
// Architecture Note (Story 22.1):
// GameScene currently manages ECS systems with distributed fields.
// For new tools like verify_gameplay, use pkg/managers.SystemManager
// for unified system creation and updates.
//
// Future migration: Consider migrating to SystemManager for system creation
// while keeping scene-specific control logic (opening animation, sodding,
// tutorial flow, etc.) in GameScene.
type GameScene struct {
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager
	gameState       *game.GameState // Global game state (阳光、关卡进度等)

	// UI Image Resources
	background   *ebiten.Image // Lawn background (草坪背景)
	seedBank     *ebiten.Image // Plant selection bar (植物选择栏)
	sunCounterBG *ebiten.Image // Sun counter background (阳光计数器背景)
	shovelSlot   *ebiten.Image // Shovel slot background (铲子槽位背景)
	shovel       *ebiten.Image // Shovel icon (铲子图标)
	// Progress bar resources (进度条资源)
	flagMeter     *ebiten.Image // Flag meter background (进度条背景，2行切片)
	flagMeterProg *ebiten.Image // Flag meter progress bar (进度条填充)
	flagMeterFlag *ebiten.Image // Flag meter flags/parts (进度条标志，3列切片)

	// Story 19.4: Bowling red line image (保龄球红线图片)
	bowlingRedLine *ebiten.Image // Red line between column 3 and 4 (保龄球关卡红线)

	// Story 19.5: Conveyor belt images (传送带图片)
	conveyorBeltBackdrop *ebiten.Image // Conveyor belt backdrop (传送带背景)
	conveyorBelt         *ebiten.Image // Conveyor belt animation (传送带传动动画，6行纹理)

	// 传送带卡片渲染资源（复用植物卡片渲染逻辑）
	conveyorCardBackground *ebiten.Image // 卡片背景框（SeedPacket_Larger）
	conveyorWallnutIcon    *ebiten.Image // 普通坚果图标
	conveyorExplodeNutIcon *ebiten.Image // 爆炸坚果图标（红色版本）

	// Story 8.2 QA改进：草皮叠加层（随动画进度渐进显示）
	sodRowImage        *ebiten.Image // 草皮叠加图片（sod1row.jpg 或 sod3row.jpg）
	soddedBackground   *ebiten.Image // 已铺草皮完整背景（IMAGE_BACKGROUND1），用于 Level 1-4 双背景叠加
	soddingAnimDelay   float64       // 铺草皮动画延迟时间（秒）
	soddingAnimStarted bool          // 铺草皮动画是否已启动
	soddingAnimTimer   float64       // 铺草皮动画延迟计时器
	sodDebugPrinted    bool          // 草皮叠加图调试日志是否已打印

	// 植物选择栏滑入动画状态
	seedBankSlideInProgress  float64 // 滑入动画进度 (0.0 - 1.0)
	seedBankSlideInStarted   bool    // 滑入动画是否已启动
	seedBankSlideInCompleted bool    // 滑入动画是否已完成

	// Story 8.12: 选卡关卡水平滑动动画状态
	// 选卡确认后，植物选择栏需要从选卡位置（X=0）水平滑动到正常游戏位置（X=SeedBankX）
	seedBankHorizontalSlideProgress  float64 // 水平滑动动画进度 (0.0 - 1.0)
	seedBankHorizontalSlideStarted   bool    // 水平滑动是否已启动
	seedBankHorizontalSlideCompleted bool    // 水平滑动是否已完成

	// Story 8.6 QA修正: 预渲染草皮支持
	preSoddedImage *ebiten.Image // 预渲染的草皮图片(仅包含指定行的草皮)

	// 性能优化：缓存草皮渲染参数，避免每帧重复计算
	sodOverlayX float64 // 草皮世界坐标X（缓存）
	sodOverlayY float64 // 草皮世界坐标Y（缓存）
	sodWidth    int     // 草皮图片宽度（缓存）
	sodHeight   int     // 草皮图片高度（缓存）

	// Story 8.2.1: 草皮闪烁调试标志
	lawnFlashLogged bool // 是否已记录草皮闪烁信息（避免重复日志）

	// Font Resources
	sunCounterFont *text.GoTextFace // Font for sun counter display
	plantCardFont  *text.GoTextFace // Font for plant card sun cost display

	// Camera and Animation
	cameraX            float64 // Camera X position (controls which part of background to show)
	maxCameraX         float64 // Maximum camera X position (rightmost edge of background)
	isIntroAnimPlaying bool    // Whether the intro animation is currently playing
	introAnimTimer     float64 // Timer for intro animation

	// ECS Framework and Systems
	entityManager     *ecs.EntityManager
	sunSpawnSystem    *systems.SunSpawnSystem
	sunMovementSystem *systems.SunMovementSystem
	lifetimeSystem    *systems.LifetimeSystem
	renderSystem      *systems.RenderSystem
	inputSystem       *systems.InputSystem
	// Story 6.3: Reanim 动画系统（替代旧的 AnimationSystem）
	reanimSystem        *systems.ReanimSystem
	sunCollectionSystem *systems.SunCollectionSystem

	// 植物选择栏模块（Story 3.1 架构优化：封装所有选卡功能）
	// 替代原有的分散系统：
	//   - plantCardSystem       *systems.PlantCardSystem       (已移至模块内部)
	//   - plantCardRenderSystem *systems.PlantCardRenderSystem (已移至模块内部)
	// 优点：
	//   - 高内聚：所有选卡功能封装在单一模块中
	//   - 低耦合：通过清晰的接口与其他系统交互
	//   - 可复用：支持在不同场景（游戏中、选卡界面、图鉴）使用
	plantSelectionModule *modules.PlantSelectionModule

	// Story 3.2: Plant Preview Systems
	plantPreviewSystem       *systems.PlantPreviewSystem
	plantPreviewRenderSystem *systems.PlantPreviewRenderSystem

	// Story 3.3: Lawn Grid System
	lawnGridSystem   *systems.LawnGridSystem // 草坪网格管理系统
	lawnGridEntityID ecs.EntityID            // 草坪网格实体ID

	// Story 3.4: Behavior System
	behaviorSystem *behavior.BehaviorSystem // 植物行为系统（向日葵生产阳光等）

	// Story 4.3: Physics System
	physicsSystem *systems.PhysicsSystem // 物理系统（碰撞检测）

	// Story 5.5: Level Management Systems
	levelSystem     *systems.LevelSystem     // 关卡管理系统
	waveSpawnSystem *systems.WaveSpawnSystem // 波次生成系统

	// Zombie Lane Transition System - 僵尸行转换系统
	zombieLaneTransitionSystem *systems.ZombieLaneTransitionSystem // 僵尸移动到目标行系统

	// Story 7.2: Particle System
	particleSystem *systems.ParticleSystem // 粒子系统（粒子特效）

	// 方案A+：Flash Effect System
	flashEffectSystem *systems.FlashEffectSystem // 闪烁效果系统（僵尸受击闪烁）

	// Story 8.2: Tutorial System
	tutorialSystem      *systems.TutorialSystem // 教学系统（关卡 1-1 教学引导）
	tutorialFont        interface{}             // 教学文本字体（*utils.BitmapFont 或 *text.GoTextFace）
	bowlingTutorialFont interface{}             // Level 1-5 保龄球教学字体（42px）

	// Story 8.2 QA改进：完整的铺草皮动画系统
	soddingSystem *systems.SoddingSystem // 铺草皮动画系统（SodRoll 滚动动画）

	// Story 8.3: Camera and Opening Animation Systems
	cameraSystem        *systems.CameraSystem           // 镜头控制系统（镜头移动、缓动）
	openingSystem       *systems.OpeningAnimationSystem // 开场动画系统（僵尸预告、跳过）
	readySetPlantSystem *systems.ReadySetPlantSystem    // ReadySetPlant 动画系统（铺草皮后播放）

	// Story 8.3 + 8.4重构: Reward Animation System (完全封装奖励流程)
	// 内部自动管理卡片包动画、粒子效果、奖励面板渲染等所有细节
	rewardSystem *systems.RewardAnimationSystem

	// Button Systems (按钮系统 - ECS 架构)
	buttonSystem       *systems.ButtonSystem       // 按钮交互系统
	buttonRenderSystem *systems.ButtonRenderSystem // 按钮渲染系统
	menuButtonEntity   ecs.EntityID                // 菜单按钮实体ID

	// Story 20.5: Slider and Checkbox Systems (滑块和复选框系统)
	sliderSystem   *systems.SliderSystem   // 滑块交互系统
	checkboxSystem *systems.CheckboxSystem // 复选框交互系统

	// Story 10.1: Pause Menu Systems (暂停菜单系统)
	pauseMenuModule *modules.PauseMenuModule // 暂停菜单模块（Story 10.1）

	// Story 10.2: Lawnmower System (除草车系统)
	lawnmowerSystem *systems.LawnmowerSystem // 除草车系统（最后防线）

	// Story 11.2: Level Progress Bar (关卡进度条)
	levelProgressBarRenderSystem *systems.LevelProgressBarRenderSystem // 进度条渲染系统
	levelProgressBarEntity       ecs.EntityID                          // 进度条实体ID

	// Story 11.3: Final Wave Warning System (最后一波提示系统)
	finalWaveWarningSystem *systems.FinalWaveWarningSystem // 最后一波提示动画系统

	// Story 8.8: Zombies Won Flow System (僵尸获胜流程系统)
	zombiesWonPhaseSystem *systems.ZombiesWonPhaseSystem // 僵尸获胜四阶段流程系统

	// Dialog Systems (对话框系统 - ECS ���构)
	dialogInputSystem  *systems.DialogInputSystem  // 对话框输入系统（处理对话框交互）
	dialogRenderSystem *systems.DialogRenderSystem // 对话框渲染系统（渲染对话框和按钮）

	// Cursor state tracking (光标状态追踪)
	lastCursorShape ebiten.CursorShapeType // 上一帧的光标形状（避免不必要的API调用）

	// Story 8.3.1: 僵尸预生成状态标志
	// 实际关卡僵尸在开场动画完成后才预生成，与预览僵尸完全独立
	zombiesPreSpawned bool

	// Story 18.3: 战斗存档对话框
	hasBattleSave         bool                 // 是否有战斗存档
	battleSaveInfo        *game.BattleSaveInfo // 战斗存档信息
	battleSaveDialogShown bool                 // 对话框是否已显示
	battleSaveDialogID    ecs.EntityID         // 对话框实体ID

	// Story 19.2: 铲子交互系统
	shovelSelected          bool                             // 铲子是否被选中
	shovelInteractionSystem *systems.ShovelInteractionSystem // 铲子交互系统

	// Story 19.3: 强引导教学系统
	guidedTutorialSystem *systems.GuidedTutorialSystem // 强引导教学系统（Level 1-5 铲子教学）

	// Story 19.4: 阶段转场系统
	levelPhaseSystem *systems.LevelPhaseSystem // 阶段转场系统（铲子教学 → 保龄球）

	// Story 19.5: 传送带系统
	conveyorBeltSystem *systems.ConveyorBeltSystem // 传送带系统（卡片生成与管理）

	// Story 21.4: 传送带卡片拖拽种植状态（移动端触摸拖拽支持）
	isDragConveyorPlanting bool   // 是否处于传送带卡片拖拽种植模式
	dragConveyorCardIndex  int    // 拖拽中的传送带卡片索引
	dragConveyorCardType   string // 拖拽中的传送带卡片类型

	// 铲子拖拽状态（移动端触摸拖拽支持）
	isDragShovel bool // 是否处于铲子拖拽模式

	// Story 19.6: 保龄球坚果滚动系统
	bowlingNutSystem *systems.BowlingNutSystem // 保龄球坚果滚动系统

	// Story 19.1: 疯狂戴夫对话系统
	daveDialogueSystem *systems.DaveDialogueSystem // 疯狂戴夫对话系统

	// 僵尸呻吟音效系统（环境音效，增强游戏氛围）
	zombieGroanSystem *systems.ZombieGroanSystem

	// Story 8.9: 撑杆僵尸跳跃系统
	poleVaultSystem *systems.PoleVaultSystem

	// Story 8.9: 减速效果系统
	slowEffectSystem *systems.SlowEffectSystem

	// Story 8.11: 大嘴花系统
	chomperSystem *systems.ChomperSystem

	// Story 8.12: 选卡界面系统
	seedChooserRenderSystem *systems.SeedChooserRenderSystem // 选卡界面渲染系统
	seedChooserInputSystem  *systems.SeedChooserInputSystem  // 选卡界面交互系统

	// 背景音乐播放状态
	bgmStarted bool // 是否已开始播放背景音乐
}

// NewGameScene creates and returns a new GameScene instance.
// It loads all necessary UI resources and initializes the game scene.
//
// Parameters:
//   - rm: The ResourceManager instance used to load game resources.
//   - sm: The SceneManager instance used to switch between scenes.
//   - levelID: The level ID to load (e.g., "1-1", "1-2"). Will be converted to "data/levels/level-{id}.yaml"
//
// Returns:
//   - A pointer to the newly created GameScene.
//
// If any UI resources fail to load, the scene will use fallback rendering methods.
func NewGameScene(rm *game.ResourceManager, sm *game.SceneManager, levelID string) *GameScene {
	// Phase 1: 创建场景基础结构
	scene := &GameScene{
		resourceManager:    rm,
		sceneManager:       sm,
		gameState:          game.GetGameState(),
		cameraX:            config.GameCameraX,
		isIntroAnimPlaying: false,
		introAnimTimer:     0,
	}

	// Phase 2: 初始化基础状态
	scene.initializeBaseState()

	// Phase 3: 检测战斗存档
	scene.detectBattleSave()

	// Phase 4: 加载资源和关卡配置
	if err := scene.loadGameResources(rm, levelID); err != nil {
		log.Printf("[GameScene] FATAL: Failed to initialize game resources: %v", err)
	}

	// Phase 5: 初始化ECS系统（拆分到 game_scene_systems.go）
	scene.initializeSystems(rm)

	// Phase 6: 初始化关卡实体和回调
	scene.initializeLevelContent(rm)

	return scene
}

// initializeBaseState 初始化基础状态
func (s *GameScene) initializeBaseState() {
	s.gameState.SetPaused(false)
	s.gameState.IsGameOver = false
	s.gameState.GameResult = ""
}

// detectBattleSave 检测战斗存档
func (s *GameScene) detectBattleSave() {
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser != "" && saveManager.HasBattleSave(currentUser) {
		s.hasBattleSave = true
		s.battleSaveInfo, _ = saveManager.GetBattleSaveInfo(currentUser)
		if s.battleSaveInfo != nil {
			log.Printf("[GameScene] 检测到战斗存档: 关卡=%s, 波次=%d, 阳光=%d",
				s.battleSaveInfo.LevelID,
				s.battleSaveInfo.WaveIndex+1,
				s.battleSaveInfo.Sun)
		}
	}
}

// loadGameResources 加载游戏资源和关卡配置
func (s *GameScene) loadGameResources(rm *game.ResourceManager, levelID string) error {
	// 加载 Reanim 资源
	if err := rm.LoadReanimResources(); err != nil {
		log.Printf("[GameScene] FATAL: Failed to load Reanim resources: %v", err)
		return fmt.Errorf("reanim resources: %w", err)
	}
	log.Printf("[GameScene] Successfully loaded all Reanim resources")

	// 初始化 ECS 框架
	s.entityManager = ecs.NewEntityManager()

	// 加载关卡配置
	levelFilePath := fmt.Sprintf("data/levels/level-%s.yaml", levelID)
	levelConfig, err := config.LoadLevelConfig(levelFilePath)
	if err != nil {
		log.Printf("[GameScene] FATAL: Failed to load level config '%s': %v", levelFilePath, err)
		return fmt.Errorf("level config: %w", err)
	}

	s.gameState.LoadLevel(levelConfig)
	log.Printf("[GameScene] Loaded level: %s (%d waves, %d plants available, enabled lanes: %v)",
		levelConfig.Name, len(levelConfig.Waves), len(levelConfig.AvailablePlants), levelConfig.EnabledLanes)

	// 验证战斗存档匹配性
	s.validateBattleSave(levelConfig)

	// 加载 UI 资源
	s.loadResources()

	return nil
}

// validateBattleSave 验证战斗存档的关卡匹配性
func (s *GameScene) validateBattleSave(levelConfig *config.LevelConfig) {
	if !s.hasBattleSave || s.battleSaveInfo == nil {
		return
	}

	if s.battleSaveInfo.LevelID != "" && s.battleSaveInfo.LevelID != levelConfig.ID {
		log.Printf("[GameScene] ⚠️ 战斗存档关卡不匹配! 存档关卡: %s, 当前关卡: %s",
			s.battleSaveInfo.LevelID, levelConfig.ID)
		log.Printf("[GameScene] 清除存档标记，删除不匹配的存档...")

		saveManager := s.gameState.GetSaveManager()
		currentUser := saveManager.GetCurrentUser()
		if err := saveManager.DeleteBattleSave(currentUser); err != nil {
			log.Printf("[GameScene] Warning: Failed to delete mismatched battle save: %v", err)
		}

		s.hasBattleSave = false
		s.battleSaveInfo = nil
	}
}

// initializeSystems 初始化所有 ECS 系统（调用 game_scene_systems.go 中的函数）
func (s *GameScene) initializeSystems(rm *game.ResourceManager) {
	// 核心系统
	s.initCoreSystems(rm)

	// 游戏逻辑系统
	s.initGameplaySystems(rm)

	// Bug Fix: 在初始化 UI 系统之前加载 LoadingImages 资源组
	// 暂停菜单模块需要 IMAGE_OPTIONS_BACKTOGAMEBUTTON0 等按钮图片
	if err := rm.LoadResourceGroup("LoadingImages"); err != nil {
		log.Printf("Warning: Failed to load LoadingImages resources before UI init: %v", err)
	}

	// UI 系统
	s.initUISystems(rm)

	// 加载草皮相关资源
	s.loadSoddingResources()

	// 初始化植物卡片系统
	s.initPlantCardSystems(rm)

	// 关卡特定系统
	s.initLevelSpecificSystems(rm)

	// 设置系统回调
	s.setupSystemCallbacks()
}

// initializeLevelContent 初始化关卡内容（实体、回调等）
func (s *GameScene) initializeLevelContent(rm *game.ResourceManager) {
	// 初始化关卡实体（预设植物、Dave对话）
	s.initLevelEntities(rm)
}

// NewGameSceneFromBattleSave 创建从战斗存档恢复的游戏场景
//
// Story 18.3: 简化实现
//
// 说明：
//
//	现在 NewGameScene 会自动检测战斗存档并处理，
//	此函数仅作为别名保留以保持向后兼容性。
//
// 参数：
//   - rm: 资源管理器
//   - sm: 场景管理器
//   - levelID: 关卡ID（从存档中获取）
//
// 返回：
//   - GameScene 实例（会自动检测存档并显示对话框）
func NewGameSceneFromBattleSave(rm *game.ResourceManager, sm *game.SceneManager, levelID string) *GameScene {
	log.Printf("[GameScene] NewGameSceneFromBattleSave 调用，将使用标准构造函数: level=%s", levelID)
	// 直接使用标准构造函数，它会自动检测存档并处理
	return NewGameScene(rm, sm, levelID)
}

// initPlantCardSystems initializes the plant selection module.
// Story 3.1 架构优化：使用 PlantSelectionModule 统一管理所有选卡功能
// Story 8.3: 使用 PlantUnlockManager 统一管理植物可用性

// This method handles:
//   - Intro animation (camera scrolling left → right → center)
//   - ECS system updates (input, sun spawning, movement, collection, lifetime management)
//   - System execution order ensures correct game logic flow
//   - Story 10.1: Pause menu (只更新 UI 系统，跳过游戏逻辑)
func (s *GameScene) Update(deltaTime float64) {
	// Phase 1: 战斗存档对话框处理（最高优先级）
	if !s.updateBattleSaveDialog(deltaTime) {
		return
	}

	// Phase 2: 暂停菜单处理
	if !s.updatePauseMenu(deltaTime) {
		return
	}

	// Phase 3: 开场动画阶段处理
	if !s.updateOpeningPhase(deltaTime) {
		return
	}

	// Phase 4: 开场动画完成后的一次性设置（返回是否跳过本帧输入）
	skipInputThisFrame := s.updatePostOpeningSetup()

	// Phase 5: 旧的 intro 动画处理（可能是遗留代码）
	if !s.checkIntroAnimation(deltaTime) {
		return
	}

	// Phase 6: 同步摄像机位置
	s.syncCameraPosition()

	// Phase 7: 游戏结束阶段处理
	if !s.updateGameOverPhase(deltaTime) {
		return
	}

	// Phase 8: 背景音乐播放控制
	s.updateBackgroundMusic()

	// Phase 9: 核心游戏循环 - 更新所有 ECS 系统
	s.updateGameplaySystems(deltaTime, s.cameraX, skipInputThisFrame)
}

// Draw renders the game scene to the screen.
// It draws the lawn background, all game entities, and UI elements in the correct order.
// Rendering order (back to front, 符合原版PVZ):
// 1. Background (lawn) - 草坪背景
// 2. UI base layer (seed bank, shovel) - UI基础元素
// 3. Plant cards - 卡片栏（在游戏实体下方）
// 4. Game entities (plants, zombies, projectiles) - 游戏世界实体
// 5. UI overlay (sun counter text) - UI文字（始终可见）
// 6. Plant preview - 植物拖拽预览
// 7. Suns (阳光) - 最顶层，确保可点击
func (s *GameScene) Draw(screen *ebiten.Image) {
	// 计算渲染上下文（UI 可见性状态）
	ctx := s.calculateRenderContext()

	// Layer 1: 背景层
	s.drawBackgroundLayer(screen)

	// Layer 2-3: UI 基础层（SeedBank、Shovel、PlantCards）
	s.drawUIBaseLayer(screen, ctx)

	// Layer 4-4.5: 游戏世界层（Plants、Zombies、LawnFlash）
	s.drawGameWorldLayer(screen)

	// 选卡界面层（特殊渲染）
	s.drawSeedChooserLayer(screen, ctx)

	// Layer 5-8: UI 覆盖层（SunCounter、Particles、Preview、Suns）
	s.drawUIOverlayLayer(screen, ctx)

	// Layer 8.5-10.1: 教学和警告层
	s.drawTutorialAndWarningLayer(screen, ctx)

	// Layer 10.2+: 最顶层 UI（Reward、Dialogs、Tooltip、PauseMenu）
	s.drawTopMostUILayer(screen)
}

// SaveOnExit 实现 game.Saveable 接口
//
// Bug Fix: 游戏关闭时自动保存战斗存档
//
// 在以下情况下保存存档：
//   - 游戏正在进行中（未结束、未暂停在对话框）
//   - 有有效的用户登录
//   - 游戏已经开始（有阳光、植物等）
//
// 返回：
//   - true: 保存成功或无需保存
//   - false: 保存失败
func (s *GameScene) SaveOnExit() bool {
	// 如果游戏已结束，不需要保存
	if s.gameState.IsGameOver {
		log.Printf("[GameScene] SaveOnExit: 游戏已结束，跳过存档")
		return true
	}

	// 如果正在显示战斗存档对话框（即刚从存档恢复），不重复保存
	if s.battleSaveDialogID != 0 {
		log.Printf("[GameScene] SaveOnExit: 正在显示存档对话框，跳过存档")
		return true
	}

	// 检查是否有有效用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] SaveOnExit: 无有效用户，跳过存档")
		return true
	}

	// 检查游戏是否已经开始（有有效的关卡时间）
	if s.gameState.LevelTime <= 0 {
		log.Printf("[GameScene] SaveOnExit: 游戏未开始，跳过存档")
		return true
	}

	// 保存战斗状态
	log.Printf("[GameScene] SaveOnExit: 游戏关闭时自动保存战斗存档")
	s.saveBattleState()
	return true
}

// destroyAllPlantPreviews 销毁所有植物预览实体
// 用于切换到铲子模式时清理残留的植物预览
func (s *GameScene) destroyAllPlantPreviews() {
	previewEntities := ecs.GetEntitiesWith1[*components.PlantPreviewComponent](s.entityManager)
	for _, entityID := range previewEntities {
		s.entityManager.DestroyEntity(entityID)
	}
	// 立即清理标记删除的实体
	s.entityManager.RemoveMarkedEntities()
}

// ============================================================================
// Story 15.5: 铲子系统、教程系统相关方法已移至 game_scene_shovel.go 和 game_scene_tutorial.go
// ============================================================================
