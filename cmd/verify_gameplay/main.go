package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/managers"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	verbose = flag.Bool("verbose", false, "显示详细调试信息")
)

// 每行对应的僵尸类型（可配置）
var zombieTypesByRow = []string{
	"flag",        // 行0: 旗帜僵尸
	"basic",       // 行1: 普通僵尸
	"polevaulter", // 行2: 撑杆跳僵尸
	"conehead",    // 行3: 路障僵尸
	"buckethead",  // 行4: 铁桶僵尸
}

// VerifyGameplayGame 统一验证游戏
// Story 22.1: 使用 SystemManager 统一管理 ECS 系统
type VerifyGameplayGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// Story 22.1: SystemManager 统一管理所有 ECS 系统
	// 新增系统时只需在 SystemManager 中添加，无需修改此文件
	systemManager *managers.SystemManager

	// 需要直接访问的系统引用（从 SystemManager 获取）
	reanimSystem          *systems.ReanimSystem
	renderSystem          *systems.RenderSystem
	rewardSystem          *systems.RewardAnimationSystem
	plantPreviewSystem    *systems.PlantPreviewSystem
	flagWaveWarningSystem *systems.FlagWaveWarningSystem
	lawnGridSystem        *systems.LawnGridSystem
	inputSystem           *systems.InputSystem

	// 植物卡片渲染
	plantCardRenderSystem *systems.PlantCardRenderSystem
	plantCardSystem       *systems.PlantCardSystem // 卡片冷却更新
	sunCounterFont        *text.GoTextFace

	// 调试字体
	debugFont *text.GoTextFace

	// 植物卡片实体
	plantCards []ecs.EntityID

	// 僵尸生成计时器（每行独立）
	zombieSpawnTimers [5]float64

	// 背景图片
	background *ebiten.Image
	seedBank   *ebiten.Image

	// 草坪网格实体
	lawnGridEntityID ecs.EntityID
}

// NewVerifyGameplayGame 创建验证游戏实例
// Story 22.1: 使用 SystemManager 创建所有系统
func NewVerifyGameplayGame() (*VerifyGameplayGame, error) {
	// 创建 ECS 管理器
	em := ecs.NewEntityManager()

	// 创建音频上下文
	audioContext := audio.NewContext(48000)

	// 创建资源管理器
	rm := game.NewResourceManager(audioContext)

	// 加载资源配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	// 加载所有资源组
	log.Println("Loading all resources...")
	if err := rm.LoadAllResources(); err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	// 加载奖励面板资源
	log.Println("Loading reward panel resources...")
	if err := rm.LoadResourceGroup("DelayLoad_AwardScreen"); err != nil {
		log.Printf("Warning: Failed to load reward panel resources: %v", err)
	}

	// 加载 Reanim 资源
	log.Println("Loading Reanim resources...")
	if err := rm.LoadReanimResources(); err != nil {
		return nil, fmt.Errorf("failed to load Reanim resources: %w", err)
	}

	// 加载 Reanim 配置
	log.Println("Loading Reanim config...")
	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		return nil, fmt.Errorf("failed to load Reanim config: %w", err)
	}
	// 关键：将 ReanimConfigManager 设置到 ResourceManager 中
	// 这样 SystemManager 创建 ReanimSystem 时才能获取配置
	rm.SetReanimConfigManager(reanimConfigManager)

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX
	gs.Sun = 9990 // 设置大量阳光用于测试

	// 创建音频管理器并设置到 GameState
	audioManager := game.NewAudioManager(rm, nil)
	gs.SetAudioManager(audioManager)

	// 创建草坪网格实体（在创建 SystemManager 之前）
	lawnGridEntityID := em.CreateEntity()
	ecs.AddComponent(em, lawnGridEntityID, &components.LawnGridComponent{})

	// Story 22.1: 使用 SystemManager 创建所有系统
	// 配置系统依赖
	deps := managers.SystemDependencies{
		EntityManager:        em,
		ResourceManager:      rm,
		GameState:            gs,
		SceneManager:         nil, // verify_gameplay 不需要场景管理
		LawnGridEntityID:     lawnGridEntityID,
		EnabledLanes:         []int{1, 2, 3, 4, 5},
		SunCollectionTargetX: float64(config.SeedBankX + config.SunPoolOffsetX),
		SunCollectionTargetY: float64(config.SeedBankY + config.SunPoolOffsetY),
		SunSpawnMinX:         config.SkyDropSunMinX,
		SunSpawnMaxX:         config.SkyDropSunMaxX,
		SunSpawnMinTargetY:   config.SkyDropSunMinTargetY,
		SunSpawnMaxTargetY:   config.SkyDropSunMaxTargetY,
		LevelConfig:          nil, // verify_gameplay 不需要关卡配置
		SpawnRulesConfig:     nil,
		ZombiePhysicsConfig:  nil,
		WindowWidth:          screenWidth,
		WindowHeight:         screenHeight,
	}

	// 使用 verify_gameplay 专用的系统选项
	opts := managers.VerifyGameplayOptions()

	// 创建 SystemManager
	systemManager := managers.NewSystemManager(deps, opts)
	log.Println("[VerifyGameplay] SystemManager created with all systems")

	// 加载字体
	sunCounterFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.SunCounterFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun counter font: %v", err)
	}

	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
	}

	plantCardFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PlantCardSunCostFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load plant card font: %v", err)
	}

	plantCardRenderSystem := systems.NewPlantCardRenderSystem(em, plantCardFont)

	// 创建 PlantCardSystem（负责卡片冷却更新）
	plantCardSystem := systems.NewPlantCardSystem(em, gs, rm)

	// 加载背景图片
	background, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		log.Printf("Warning: Failed to load background: %v", err)
	}

	seedBank, err := rm.LoadImageByID("IMAGE_SEEDBANK")
	if err != nil {
		log.Printf("Warning: Failed to load seed bank: %v", err)
	}

	// 创建 FlagWaveWarningSystem（手动触发模式）
	flagWaveWarningSystem := systems.NewFlagWaveWarningSystem(em, nil, rm)

	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║              统一游戏验证程序                          ║")
	log.Println("║        Story 22.1: SystemManager 统一管理             ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Println()
	log.Println("【功能说明】")
	log.Println("  - 植物选择栏：显示所有可用植物")
	log.Println("  - 僵尸生成：每行固定类型，每2秒生成一个")
	log.Println("  - 行0: 旗帜僵尸  行1: 普通僵尸")
	log.Println("  - 行2: 撑杆跳僵尸  行3: 路障僵尸")
	log.Println("  - 行4: 铁桶僵尸")
	log.Println()
	log.Println("【快捷键】")
	log.Println("  1-5       - 选择对应植物卡片")
	log.Println("  F1-F5     - 触发对应行的除草车")
	log.Println("  Shift+1-5 - 在对应行生成一个僵尸")
	log.Println("  R         - 触发奖励动画")
	log.Println("  H         - 触发「一大波僵尸正在接近」提示")
	log.Println("  G         - 开启/关闭自动生成僵尸")
	log.Println("  C         - 清除所有僵尸")
	log.Println("  ESC       - 退出程序")
	log.Println("  右键      - 取消植物选择")
	log.Println("════════════════════════════════════════════════════════")

	vg := &VerifyGameplayGame{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,

		// Story 22.1: SystemManager 统一管理
		systemManager: systemManager,

		// 从 SystemManager 获取需要直接访问的系统
		reanimSystem:       systemManager.GetReanimSystem(),
		renderSystem:       systemManager.GetRenderSystem(),
		rewardSystem:       systemManager.GetRewardSystem(),
		plantPreviewSystem: systemManager.GetPlantPreviewSystem(),
		lawnGridSystem:     systemManager.GetLawnGridSystem(),
		inputSystem:        systemManager.GetInputSystem(),

		// 手动创建的系统（不在 SystemManager 中）
		flagWaveWarningSystem: flagWaveWarningSystem,

		// 其他字段
		plantCardRenderSystem: plantCardRenderSystem,
		plantCardSystem:       plantCardSystem,
		sunCounterFont:        sunCounterFont,
		debugFont:             debugFont,
		background:            background,
		seedBank:              seedBank,
		lawnGridEntityID:      lawnGridEntityID,
	}

	// 验证工具启用忽略冷却模式，可以随意种植
	if vg.inputSystem != nil {
		vg.inputSystem.SetIgnoreCooldown(true)
		log.Println("[VerifyGameplay] 已启用忽略冷却模式，可以随意种植植物")
	}

	// 初始化场景
	vg.setupScene()

	return vg, nil
}

// setupScene 设置测试场景
func (vg *VerifyGameplayGame) setupScene() {
	// 创建植物卡片（所有可用植物）
	vg.createPlantCards()

	// 创建除草车（每行一台）
	vg.createLawnmowers()

	// 自动生成一个旗帜僵尸用于测试（在第0行）
	log.Println("[VerifyGameplay] 自动生成旗帜僵尸进行测试...")
	vg.spawnZombie(0)
}

// createPlantCards 创建所有植物卡片
func (vg *VerifyGameplayGame) createPlantCards() {
	allPlants := []components.PlantType{
		components.PlantSunflower,
		components.PlantPeashooter,
		components.PlantWallnut,
		components.PlantCherryBomb,
		components.PlantPotatoMine,
		components.PlantSnowPea,
	}

	startX := float64(config.SeedBankX + config.PlantCardStartOffsetX)
	startY := float64(config.SeedBankY + config.PlantCardOffsetY)

	for i, plantType := range allPlants {
		x := startX + float64(i)*float64(config.PlantCardSpacing)
		cardID, err := entities.NewPlantCardEntity(
			vg.entityManager,
			vg.resourceManager,
			vg.reanimSystem,
			plantType,
			x, startY,
			config.PlantCardScale,
		)
		if err != nil {
			log.Printf("Warning: Failed to create plant card %d: %v", plantType, err)
			continue
		}
		vg.plantCards = append(vg.plantCards, cardID)
		log.Printf("[VerifyGameplay] Created plant card: type=%d, entity=%d", plantType, cardID)
	}
}

// createLawnmowers 创建除草车
func (vg *VerifyGameplayGame) createLawnmowers() {
	for lane := 1; lane <= 5; lane++ {
		lawnmowerID, err := entities.NewLawnmowerEntity(vg.entityManager, vg.resourceManager, lane)
		if err != nil {
			log.Printf("Warning: Failed to create lawnmower for lane %d: %v", lane, err)
			continue
		}
		log.Printf("[VerifyGameplay] Created lawnmower: lane=%d, entity=%d", lane, lawnmowerID)
	}
}

// spawnZombie 在指定行生成僵尸
func (vg *VerifyGameplayGame) spawnZombie(row int) {
	if row < 0 || row >= 5 {
		return
	}

	zombieType := zombieTypesByRow[row]
	// 僵尸出生在屏幕可视范围最右端（世界坐标 = 相机X + 屏幕宽度 - 边距）
	spawnX := config.GameCameraX + float64(screenWidth) - 30.0

	var zombieID ecs.EntityID
	var err error

	switch zombieType {
	case "basic":
		zombieID, err = entities.NewZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	case "conehead":
		zombieID, err = entities.NewConeheadZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	case "buckethead":
		zombieID, err = entities.NewBucketheadZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	case "flag":
		zombieID, err = entities.NewFlagZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	case "polevaulter":
		zombieID, err = entities.NewPolevaulterZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	default:
		zombieID, err = entities.NewZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	}

	if err != nil {
		log.Printf("Warning: Failed to spawn zombie: %v", err)
		return
	}

	// 根据僵尸类型使用对应的激活函数
	if zombieType == "polevaulter" {
		entities.ActivatePolevaulterZombie(vg.entityManager, zombieID)
	} else {
		entities.ActivateZombie(vg.entityManager, zombieID)
	}

	if *verbose {
		log.Printf("[VerifyGameplay] Spawned %s zombie on row %d (entity=%d)", zombieType, row, zombieID)
	}
}

// triggerLawnmower 触发指定行的除草车
func (vg *VerifyGameplayGame) triggerLawnmower(lane int) {
	lawnmowers := ecs.GetEntitiesWith2[*components.LawnmowerComponent, *components.PositionComponent](vg.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](vg.entityManager, lawnmowerID)

		if lawnmower.Lane == lane && !lawnmower.IsTriggered {
			lawnmower.IsTriggered = true
			lawnmower.IsMoving = true

			// 播放音效
			player := vg.resourceManager.GetAudioPlayer("SOUND_LAWNMOWER")
			if player != nil {
				player.Rewind()
				player.Play()
			}

			// 恢复动画播放
			if reanim, ok := ecs.GetComponent[*components.ReanimComponent](vg.entityManager, lawnmowerID); ok {
				reanim.IsPaused = false
			}

			log.Printf("[VerifyGameplay] Triggered lawnmower on lane %d", lane)
			break
		}
	}
}

// clearAllZombies 清除所有僵尸
func (vg *VerifyGameplayGame) clearAllZombies() {
	zombies := ecs.GetEntitiesWith1[*components.BehaviorComponent](vg.entityManager)

	count := 0
	for _, entityID := range zombies {
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](vg.entityManager, entityID)
		if behavior.Type == components.BehaviorZombieBasic ||
			behavior.Type == components.BehaviorZombieConehead ||
			behavior.Type == components.BehaviorZombieBuckethead ||
			behavior.Type == components.BehaviorZombieFlag ||
			behavior.Type == components.BehaviorZombiePolevaulter {
			vg.entityManager.DestroyEntity(entityID)
			count++
		}
	}
	vg.entityManager.RemoveMarkedEntities()
	log.Printf("[VerifyGameplay] Cleared %d zombies", count)
}

// 僵尸生成开关
var zombieSpawnEnabled = false // 默认关闭自动生成

// Update 更新游戏逻辑
// Story 22.1: 使用 SystemManager.Update() 统一更新所有系统
func (vg *VerifyGameplayGame) Update() error {
	dt := 1.0 / 60.0

	// ESC 退出（优先于 InputSystem 的暂停处理）
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		log.Println("[VerifyGameplay] Exiting...")
		return fmt.Errorf("quit")
	}

	// verify_gameplay 特有的调试快捷键
	// F1-F5 触发对应行的除草车
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		vg.triggerLawnmower(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		vg.triggerLawnmower(2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		vg.triggerLawnmower(3)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		vg.triggerLawnmower(4)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		vg.triggerLawnmower(5)
	}

	// Shift+1-5 在指定行手动生成一个僵尸
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		if inpututil.IsKeyJustPressed(ebiten.Key1) {
			vg.spawnZombie(0)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key2) {
			vg.spawnZombie(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			vg.spawnZombie(2)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key4) {
			vg.spawnZombie(3)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key5) {
			vg.spawnZombie(4)
		}
	}

	// R 触发奖励动画
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		log.Println("[VerifyGameplay] Triggering reward animation...")
		vg.rewardSystem.TriggerReward("plant", "sunflower")
	}

	// H 触发「一大波僵尸正在接近」提示
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		if vg.flagWaveWarningSystem.IsWarningActive() {
			log.Println("[VerifyGameplay] Dismissing huge wave warning...")
			vg.flagWaveWarningSystem.DismissWarning()
		} else {
			log.Println("[VerifyGameplay] Triggering huge wave warning...")
			vg.flagWaveWarningSystem.TriggerWarning()
		}
	}

	// G 暂停/继续僵尸生成（改用 G 键避免与 InputSystem 的 P 键冲突）
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		zombieSpawnEnabled = !zombieSpawnEnabled
		if zombieSpawnEnabled {
			log.Println("[VerifyGameplay] Zombie spawn ENABLED")
		} else {
			log.Println("[VerifyGameplay] Zombie spawn DISABLED")
		}
	}

	// C 清除所有僵尸
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		vg.clearAllZombies()
	}

	// 调用正式的 InputSystem 处理植物选择和种植逻辑
	// 这会复用正式的逻辑，包括种植粒子效果
	if vg.inputSystem != nil {
		vg.inputSystem.Update(dt, vg.gameState.CameraX)
	}

	// 僵尸自动生成（每行每5秒生成一个）
	if zombieSpawnEnabled {
		for row := 0; row < 5; row++ {
			vg.zombieSpawnTimers[row] += dt
			if vg.zombieSpawnTimers[row] >= 5.0 {
				vg.spawnZombie(row)
				vg.zombieSpawnTimers[row] = 0
			}
		}
	}

	// Story 22.1: 使用 SystemManager 统一更新所有系统
	// 这确保了所有系统都会被更新，不会遗漏任何系统
	vg.systemManager.Update(dt)

	// 更新手动管理的系统
	vg.flagWaveWarningSystem.Update(dt)
	vg.plantCardSystem.Update(dt) // 更新卡片冷却

	// 更新鼠标光标
	cursorShape := vg.rewardSystem.GetCursorShape()
	ebiten.SetCursorShape(cursorShape)

	return nil
}

// Draw 绘制游戏画面
func (vg *VerifyGameplayGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 绘制背景
	if vg.background != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vg.gameState.CameraX, 0)
		screen.DrawImage(vg.background, opts)
	}

	// 绘制游戏世界元素（使用 RenderSystem）
	vg.renderSystem.DrawGameWorld(screen, vg.gameState.CameraX)

	// 绘制游戏世界粒子
	vg.renderSystem.DrawGameWorldParticles(screen, vg.gameState.CameraX)

	// 绘制 UI 元素
	vg.drawUI(screen)

	// 绘制 UI 层的 Reanim 实体
	vg.renderSystem.DrawUIElements(screen)

	// 绘制红字警告（一大波僵尸正在接近）
	vg.drawHugeWaveWarning(screen)

	// 绘制植物预览（选择植物后跟随鼠标的预览图像）
	// 使用 SystemManager 提供的 PlantPreviewRenderSystem
	if plantPreviewRenderSystem := vg.systemManager.GetPlantPreviewRenderSystem(); plantPreviewRenderSystem != nil {
		plantPreviewRenderSystem.Draw(screen, vg.gameState.CameraX)
	}

	// 绘制奖励动画
	vg.rewardSystem.Draw(screen)

	// 绘制调试信息
	vg.drawDebugInfo(screen)
}

// drawHugeWaveWarning 渲染红字警告 "A Huge Wave of Zombies is Approaching!"
func (vg *VerifyGameplayGame) drawHugeWaveWarning(screen *ebiten.Image) {
	// 查询红字警告实体
	warningEntities := ecs.GetEntitiesWith1[*components.FlagWaveWarningComponent](vg.entityManager)
	if len(warningEntities) == 0 {
		return
	}

	// 获取警告组件
	warningComp, ok := ecs.GetComponent[*components.FlagWaveWarningComponent](vg.entityManager, warningEntities[0])
	if !ok || !warningComp.IsActive {
		return
	}

	// 闪烁效果：不可见时跳过渲染
	if !warningComp.FlashVisible {
		return
	}

	// 优先使用预渲染的位图字体图片
	if warningComp.TextImage != nil {
		vg.drawHugeWaveWarningBitmap(screen, warningComp)
		return
	}

	// 回退：使用 debugFont 渲染文本
	vg.drawHugeWaveWarningText(screen, warningComp)
}

// drawHugeWaveWarningBitmap 使用预渲染的位图字体图片绘制警告
func (vg *VerifyGameplayGame) drawHugeWaveWarningBitmap(screen *ebiten.Image, warningComp *components.FlagWaveWarningComponent) {
	textImage := warningComp.TextImage
	imgWidth := float64(textImage.Bounds().Dx())
	imgHeight := float64(textImage.Bounds().Dy())

	// 计算缩放后的尺寸
	scaledWidth := imgWidth * warningComp.Scale
	scaledHeight := imgHeight * warningComp.Scale

	// 绘制半透明黑色背景（提高可读性）
	bgPadding := 15.0 * warningComp.Scale
	bgX := warningComp.X - scaledWidth/2 - bgPadding
	bgY := warningComp.Y - scaledHeight/2 - bgPadding
	bgW := scaledWidth + bgPadding*2
	bgH := scaledHeight + bgPadding*2

	bgImage := ebiten.NewImage(int(bgW), int(bgH))
	bgImage.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 128})
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(bgX, bgY)
	screen.DrawImage(bgImage, bgOp)

	// 绘制文字图片
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(warningComp.Scale, warningComp.Scale)
	op.GeoM.Translate(-scaledWidth/2, -scaledHeight/2)
	op.GeoM.Translate(warningComp.X, warningComp.Y)

	// 应用透明度
	op.ColorScale.ScaleAlpha(float32(warningComp.Alpha))

	screen.DrawImage(textImage, op)
}

// drawHugeWaveWarningText 使用系统字体绘制警告（回退方案）
func (vg *VerifyGameplayGame) drawHugeWaveWarningText(screen *ebiten.Image, warningComp *components.FlagWaveWarningComponent) {
	if vg.debugFont == nil {
		return
	}

	warningText := warningComp.Text

	// 测量文本宽度以便居中
	textWidth := text.Advance(warningText, vg.debugFont)

	// 居中绘制
	x := warningComp.X - textWidth/2
	y := warningComp.Y

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(color.RGBA{R: 255, G: 0, B: 0, A: 255}) // 红色
	text.Draw(screen, warningText, vg.debugFont, op)
}

// drawUI 绘制 UI 元素
func (vg *VerifyGameplayGame) drawUI(screen *ebiten.Image) {
	// 绘制种子栏背景
	if vg.seedBank != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(config.SeedBankX), float64(config.SeedBankY))
		screen.DrawImage(vg.seedBank, opts)
	}

	// 绘制植物卡片
	vg.plantCardRenderSystem.Draw(screen)

	// 绘制阳光计数
	if vg.sunCounterFont != nil {
		sunText := fmt.Sprintf("%d", vg.gameState.GetSun())
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(config.SeedBankX+config.SunCounterOffsetX), float64(config.SeedBankY+config.SunCounterOffsetY))
		op.ColorScale.ScaleWithColor(color.Black)
		text.Draw(screen, sunText, vg.sunCounterFont, op)
	}
}

// drawDebugInfo 绘制调试信息
func (vg *VerifyGameplayGame) drawDebugInfo(screen *ebiten.Image) {
	if vg.debugFont == nil {
		return
	}

	y := float64(screenHeight - 120)
	clr := color.RGBA{255, 255, 255, 200}

	// 快捷键提示
	hints := []string{
		"1-5=选择植物 | F1-F5=除草车 | Shift+1-5=生成僵尸 | R=奖励 | H=一大波 | G=自动生成 | C=清除 | ESC=退出",
	}

	isPlanting, plantType := vg.gameState.GetPlantingMode()
	if isPlanting {
		hints = append(hints, fmt.Sprintf("已选择植物: %d (点击草坪种植)", plantType))
	}

	spawnStatus := "已启用"
	if !zombieSpawnEnabled {
		spawnStatus = "已暂停"
	}
	hints = append(hints, fmt.Sprintf("僵尸生成: %s", spawnStatus))

	for _, hint := range hints {
		op := &text.DrawOptions{}
		op.GeoM.Translate(10, y)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, hint, vg.debugFont, op)
		y += 18
	}
}

// Layout 设置屏幕布局
func (vg *VerifyGameplayGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	flag.Parse()

	if !*verbose {
		log.SetOutput(os.Stdout)
	}

	verifyGame, err := NewVerifyGameplayGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	ebiten.SetWindowTitle("统一游戏验证程序 (Story 22.1: SystemManager)")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	if err := ebiten.RunGame(verifyGame); err != nil {
		if err.Error() != "quit" {
			log.Fatal(err)
		}
	}
}
