package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/decker502/pvz/pkg/systems/behavior"
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
	"flag",       // 行0: 旗帜僵尸
	"basic",      // 行1: 普通僵尸
	"conehead",   // 行2: 路障僵尸
	"conehead",   // 行3: 路障僵尸
	"buckethead", // 行4: 铁桶僵尸
}

// VerifyGameplayGame 统一验证游戏
type VerifyGameplayGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// 核心系统
	reanimSystem        *systems.ReanimSystem
	renderSystem        *systems.RenderSystem
	behaviorSystem      *behavior.BehaviorSystem
	physicsSystem       *systems.PhysicsSystem
	lawnmowerSystem     *systems.LawnmowerSystem
	rewardSystem        *systems.RewardAnimationSystem
	particleSystem      *systems.ParticleSystem
	lawnGridSystem      *systems.LawnGridSystem
	sunCollectionSystem *systems.SunCollectionSystem
	sunMovementSystem   *systems.SunMovementSystem
	flashEffectSystem   *systems.FlashEffectSystem

	// 红字警告系统（一大波僵尸正在接近）
	flagWaveWarningSystem *systems.FlagWaveWarningSystem

	// 植物预览系统
	plantPreviewSystem       *systems.PlantPreviewSystem
	plantPreviewRenderSystem *systems.PlantPreviewRenderSystem

	// 植物卡片渲染
	plantCardRenderSystem *systems.PlantCardRenderSystem
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

	// 选中的植物类型
	selectedPlantType components.PlantType

	// 植物预览实体
	previewEntityID ecs.EntityID
}

// NewVerifyGameplayGame 创建验证游戏实例
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

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX
	gs.Sun = 9990 // 设置大量阳光用于测试

	// 创建系统
	reanimSystem := systems.NewReanimSystem(em)
	reanimSystem.SetConfigManager(reanimConfigManager)
	reanimSystem.SetResourceLoader(rm) // 设置资源加载器，用于运行时单位切换

	renderSystem := systems.NewRenderSystem(em)
	renderSystem.SetReanimSystem(reanimSystem)
	renderSystem.SetResourceManager(rm)

	particleSystem := systems.NewParticleSystem(em, rm)

	// 创建草坪网格系统（启用所有行）
	enabledLanes := []int{1, 2, 3, 4, 5}
	lawnGridSystem := systems.NewLawnGridSystem(em, enabledLanes)

	// 创建除草车系统
	lawnmowerSystem := systems.NewLawnmowerSystem(em, rm, gs)

	// 创建行为系统（传入草坪网格系统）
	behaviorSystem := behavior.NewBehaviorSystem(em, rm, gs, lawnGridSystem, 0)

	// 创建物理系统
	physicsSystem := systems.NewPhysicsSystem(em, rm)

	// 创建奖励动画系统
	rewardSystem := systems.NewRewardAnimationSystem(em, gs, rm, nil, reanimSystem, particleSystem, renderSystem)

	// 计算阳光收集目标位置
	sunTargetX := float64(config.SeedBankX + config.SunPoolOffsetX)
	sunTargetY := float64(config.SeedBankY + config.SunPoolOffsetY)

	// 创建阳光收集和移动系统
	sunCollectionSystem := systems.NewSunCollectionSystem(em, gs, sunTargetX, sunTargetY)
	sunMovementSystem := systems.NewSunMovementSystem(em)

	// 创建闪烁效果系统
	flashEffectSystem := systems.NewFlashEffectSystem(em)

	// 创建红字警告系统（一大波僵尸正在接近）
	// 传入 nil 作为 WaveTimingSystem，使用手动触发模式
	flagWaveWarningSystem := systems.NewFlagWaveWarningSystem(em, nil, rm)

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

	// 加载背景图片
	background, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		log.Printf("Warning: Failed to load background: %v", err)
	}

	seedBank, err := rm.LoadImageByID("IMAGE_SEEDBANK")
	if err != nil {
		log.Printf("Warning: Failed to load seed bank: %v", err)
	}

	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║              统一游戏验证程序                          ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Println()
	log.Println("【功能说明】")
	log.Println("  - 植物选择栏：显示所有可用植物")
	log.Println("  - 僵尸生成：每行固定类型，每2秒生成一个")
	log.Println("  - 行0: 旗帜僵尸  行1: 普通僵尸")
	log.Println("  - 行2: 路障僵尸  行3: 路障僵尸")
	log.Println("  - 行4: 铁桶僵尸")
	log.Println()
	log.Println("【快捷键】")
	log.Println("  1-5       - 触发对应行的除草车")
	log.Println("  Shift+1-5 - 在对应行生成一个僵尸")
	log.Println("  R         - 触发奖励动画")
	log.Println("  H         - 触发「一大波僵尸正在接近」提示")
	log.Println("  P         - 开启/关闭自动生成僵尸")
	log.Println("  C         - 清除所有僵尸")
	log.Println("  Q         - 退出程序")
	log.Println("  右键      - 取消植物选择")
	log.Println("════════════════════════════════════════════════════════")

	// 创建植物预览系统
	plantPreviewSystem := systems.NewPlantPreviewSystem(em, gs, lawnGridSystem)
	plantPreviewRenderSystem := systems.NewPlantPreviewRenderSystem(em, plantPreviewSystem)

	vg := &VerifyGameplayGame{
		entityManager:            em,
		gameState:                gs,
		resourceManager:          rm,
		reanimSystem:             reanimSystem,
		renderSystem:             renderSystem,
		behaviorSystem:           behaviorSystem,
		physicsSystem:            physicsSystem,
		lawnmowerSystem:          lawnmowerSystem,
		rewardSystem:             rewardSystem,
		particleSystem:           particleSystem,
		lawnGridSystem:           lawnGridSystem,
		sunCollectionSystem:      sunCollectionSystem,
		sunMovementSystem:        sunMovementSystem,
		flashEffectSystem:        flashEffectSystem,
		flagWaveWarningSystem:    flagWaveWarningSystem,
		plantPreviewSystem:       plantPreviewSystem,
		plantPreviewRenderSystem: plantPreviewRenderSystem,
		plantCardRenderSystem:    plantCardRenderSystem,
		sunCounterFont:           sunCounterFont,
		debugFont:                debugFont,
		background:               background,
		seedBank:                 seedBank,
	}

	// 初始化场景
	vg.setupScene()

	return vg, nil
}

// setupScene 设置测试场景
func (vg *VerifyGameplayGame) setupScene() {
	// 创建草坪网格实体
	vg.lawnGridEntityID = vg.entityManager.CreateEntity()

	// 创建植物卡片（所有可用植物）
	vg.createPlantCards()

	ecs.AddComponent(vg.entityManager, vg.lawnGridEntityID, &components.LawnGridComponent{})

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
	default:
		zombieID, err = entities.NewZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
	}

	if err != nil {
		log.Printf("Warning: Failed to spawn zombie: %v", err)
		return
	}

	// 立即激活僵尸（设置速度）
	vel, ok := ecs.GetComponent[*components.VelocityComponent](vg.entityManager, zombieID)
	if ok {
		vel.VX = config.ZombieWalkSpeed // 僵尸向左移动
	}

	// 切换到行走动画
	behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](vg.entityManager, zombieID)
	if ok {
		behaviorComp.ZombieAnimState = components.ZombieAnimWalking
	}

	// 播放行走动画
	unitID := "zombie"
	switch zombieType {
	case "conehead":
		unitID = "zombie_conehead"
	case "buckethead":
		unitID = "zombie_buckethead"
	case "flag":
		unitID = "zombie_flag"
	}
	ecs.AddComponent(vg.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    unitID,
		ComboName: "walk",
		Processed: false,
	})

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
			behavior.Type == components.BehaviorZombieFlag {
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
func (vg *VerifyGameplayGame) Update() error {
	dt := 1.0 / 60.0

	// 快捷键处理
	// 1-5 触发对应行的除草车（不按Shift时）
	if !ebiten.IsKeyPressed(ebiten.KeyShift) {
		if inpututil.IsKeyJustPressed(ebiten.Key1) {
			vg.triggerLawnmower(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key2) {
			vg.triggerLawnmower(2)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			vg.triggerLawnmower(3)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key4) {
			vg.triggerLawnmower(4)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key5) {
			vg.triggerLawnmower(5)
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

	// P 暂停/继续僵尸生成
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
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

	// Q 退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyGameplay] Exiting...")
		return fmt.Errorf("quit")
	}

	// 处理植物卡片点击
	vg.handlePlantCardClick()

	// 处理右键取消选择
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && vg.selectedPlantType != components.PlantUnknown {
		vg.cancelPlantSelection()
	}

	// 处理草坪格子点击（种植植物）
	vg.handleLawnClick()

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

	// 更新各个系统
	vg.reanimSystem.Update(dt)
	vg.behaviorSystem.Update(dt)
	vg.physicsSystem.Update(dt)
	vg.lawnmowerSystem.Update(dt)
	vg.particleSystem.Update(dt)
	vg.lawnGridSystem.Update(dt)
	vg.sunCollectionSystem.Update(dt)
	vg.sunMovementSystem.Update(dt)
	vg.rewardSystem.Update(dt)
	vg.flashEffectSystem.Update(dt)
	vg.plantPreviewSystem.Update(dt)

	// 清理已删除的实体（必须在所有系统更新后调用）
	vg.entityManager.RemoveMarkedEntities()

	// 更新鼠标光标
	cursorShape := vg.rewardSystem.GetCursorShape()
	ebiten.SetCursorShape(cursorShape)

	return nil
}

// handlePlantCardClick 处理植物卡片点击
func (vg *VerifyGameplayGame) handlePlantCardClick() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mx, my := ebiten.CursorPosition()

	for _, cardID := range vg.plantCards {
		pos, ok := ecs.GetComponent[*components.PositionComponent](vg.entityManager, cardID)
		if !ok {
			continue
		}
		card, ok := ecs.GetComponent[*components.PlantCardComponent](vg.entityManager, cardID)
		if !ok {
			continue
		}
		click, ok := ecs.GetComponent[*components.ClickableComponent](vg.entityManager, cardID)
		if !ok {
			continue
		}

		// 检查点击是否在卡片范围内
		if float64(mx) >= pos.X && float64(mx) <= pos.X+click.Width &&
			float64(my) >= pos.Y && float64(my) <= pos.Y+click.Height {
			// 如果点击同一个卡片，取消选择
			if vg.selectedPlantType == card.PlantType {
				vg.cancelPlantSelection()
				return
			}

			// 选择新植物前，先销毁旧的预览实体
			if vg.previewEntityID != 0 {
				vg.entityManager.DestroyEntity(vg.previewEntityID)
				vg.previewEntityID = 0
			}

			vg.selectedPlantType = card.PlantType
			log.Printf("[VerifyGameplay] Selected plant: %d", card.PlantType)

			// 创建预览实体
			vg.createPlantPreview(card.PlantType)
			return
		}
	}
}

// createPlantPreview 创建植物预览实体
func (vg *VerifyGameplayGame) createPlantPreview(plantType components.PlantType) {
	// 渲染植物图标（直接传入 plantType）
	plantIcon, err := entities.RenderPlantIcon(
		vg.entityManager,
		vg.resourceManager,
		vg.reanimSystem,
		plantType,
	)
	if err != nil {
		log.Printf("[VerifyGameplay] Failed to render plant preview: %v", err)
		return
	}

	// 创建预览实体
	vg.previewEntityID = vg.entityManager.CreateEntity()

	// 添加位置组件（初始位置，会被 PlantPreviewSystem 更新）
	ecs.AddComponent(vg.entityManager, vg.previewEntityID, &components.PositionComponent{
		X: 0,
		Y: 0,
	})

	// 添加精灵组件
	ecs.AddComponent(vg.entityManager, vg.previewEntityID, &components.SpriteComponent{
		Image: plantIcon,
	})

	// 添加预览组件
	ecs.AddComponent(vg.entityManager, vg.previewEntityID, &components.PlantPreviewComponent{
		PlantType: plantType,
		Alpha:     0.5,
	})

	log.Printf("[VerifyGameplay] Created plant preview entity: %d", vg.previewEntityID)
}

// cancelPlantSelection 取消植物选择
func (vg *VerifyGameplayGame) cancelPlantSelection() {
	if vg.previewEntityID != 0 {
		vg.entityManager.DestroyEntity(vg.previewEntityID)
		vg.previewEntityID = 0
	}
	vg.selectedPlantType = components.PlantUnknown
	log.Printf("[VerifyGameplay] Cancelled plant selection")
}

// handleLawnClick 处理草坪点击（种植植物）
func (vg *VerifyGameplayGame) handleLawnClick() {
	if vg.selectedPlantType == components.PlantUnknown {
		return
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mx, my := ebiten.CursorPosition()

	// 将屏幕坐标转换为世界坐标
	worldX := float64(mx) + vg.gameState.CameraX
	worldY := float64(my)

	// 计算格子坐标
	col := int((worldX - config.GridWorldStartX) / config.CellWidth)
	row := int((worldY - config.GridWorldStartY) / config.CellHeight)

	// 检查是否在有效范围内
	if col < 0 || col >= config.GridColumns || row < 0 || row >= config.GridRows {
		return
	}

	// 检查格子是否被占用（使用 LawnGridSystem API）
	if vg.lawnGridSystem.IsOccupied(vg.lawnGridEntityID, col, row) {
		log.Printf("[VerifyGameplay] Cell (%d, %d) is occupied", row, col)
		return
	}

	// 检查阳光是否足够
	sunCost := vg.getSunCost(vg.selectedPlantType)
	if vg.gameState.GetSun() < sunCost {
		log.Printf("[VerifyGameplay] Not enough sun: need %d, have %d", sunCost, vg.gameState.GetSun())
		return
	}

	// 根据植物类型调用对应的工厂函数
	var plantID ecs.EntityID
	var err error

	switch vg.selectedPlantType {
	case components.PlantSunflower, components.PlantPeashooter:
		plantID, err = entities.NewPlantEntity(
			vg.entityManager,
			vg.resourceManager,
			vg.gameState,
			vg.reanimSystem,
			vg.selectedPlantType,
			col, row,
		)
	case components.PlantWallnut:
		plantID, err = entities.NewWallnutEntity(
			vg.entityManager,
			vg.resourceManager,
			vg.gameState,
			vg.reanimSystem,
			col, row,
		)
	case components.PlantCherryBomb:
		plantID, err = entities.NewCherryBombEntity(
			vg.entityManager,
			vg.resourceManager,
			vg.gameState,
			col, row,
		)
	default:
		log.Printf("[VerifyGameplay] Unknown plant type: %d", vg.selectedPlantType)
		return
	}

	if err != nil {
		log.Printf("Warning: Failed to plant: %v", err)
		return
	}

	// 更新网格（使用 LawnGridSystem API）
	vg.lawnGridSystem.OccupyCell(vg.lawnGridEntityID, col, row, plantID)

	// 扣除阳光
	vg.gameState.SpendSun(sunCost)

	log.Printf("[VerifyGameplay] Planted type=%d at (%d, %d), entity=%d", vg.selectedPlantType, row, col, plantID)

	// 销毁预览实体
	if vg.previewEntityID != 0 {
		vg.entityManager.DestroyEntity(vg.previewEntityID)
		vg.previewEntityID = 0
	}

	// 重置选中状态
	vg.selectedPlantType = components.PlantUnknown
}

// getSunCost 获取植物阳光消耗
func (vg *VerifyGameplayGame) getSunCost(plantType components.PlantType) int {
	switch plantType {
	case components.PlantSunflower:
		return config.SunflowerSunCost
	case components.PlantPeashooter:
		return config.PeashooterSunCost
	case components.PlantWallnut:
		return config.WallnutCost
	case components.PlantCherryBomb:
		return config.CherryBombSunCost
	default:
		return 0
	}
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
	vg.plantPreviewRenderSystem.Draw(screen, vg.gameState.CameraX)

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
		"1-5=除草车 | Shift+1-5=生成僵尸 | R=奖励 | H=一大波僵尸 | P=自动生成 | C=清除 | Q=退出 | 右键=取消",
	}

	if vg.selectedPlantType != components.PlantUnknown {
		hints = append(hints, fmt.Sprintf("已选择植物: %d (点击草坪种植)", vg.selectedPlantType))
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

	ebiten.SetWindowTitle("统一游戏验证程序")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	if err := ebiten.RunGame(verifyGame); err != nil {
		if err.Error() != "quit" {
			log.Fatal(err)
		}
	}
}
