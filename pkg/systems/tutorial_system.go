package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// TutorialSystem 教学系统
// 管理关卡 1-1 的分步教学引导流程
// 检测触发条件，显示教学文本，推进教学步骤
type TutorialSystem struct {
	entityManager        *ecs.EntityManager
	gameState            *game.GameState
	resourceManager      *game.ResourceManager
	lawnGridSystem       *LawnGridSystem  // 用于控制草坪闪烁效果（Story 8.2）
	sunSpawnSystem       *SunSpawnSystem  // 用于启用阳光自动生成（Story 8.2）
	waveSpawnSystem      *WaveSpawnSystem // 用于激活预生成的僵尸（替代自己创建僵尸）
	tutorialEntity       ecs.EntityID     // 教学实体ID
	textEntity           ecs.EntityID     // 教学文本实体ID（用于显示/隐藏）
	arrowIndicatorEntity ecs.EntityID     // 箭头指示符实体ID（用于显示/隐藏）
	cardHighlightEntity  ecs.EntityID     // 卡片闪烁效果实体ID（用于显示/隐藏）

	// 状态跟踪变量（用于检测变化）
	lastSunAmount       int     // 上一帧的阳光数量
	lastPlantCount      int     // 上一帧的植物数量
	lastZombieCount     int     // 上一帧的僵尸数量
	plantCount          int     // 当前种植的植物总数（用于第二次种植检测）
	initialized         bool    // 是否已初始化（用于 gameStart 触发）
	soddingComplete     bool    // 铺草皮动画是否完成（用于延迟 gameStart 触发）
	sunSpawned          bool    // 第一颗阳光是否已生成（用于 sunSpawned 触发）
	lastSunCount        int     // 上一帧的阳光实体数量（用于检测新阳光生成）
	lastTextDisplayTime float64 // 上次教学文本显示的时间（用于时长检测，防止文本闪烁）

	// 粒子效果重复显示定时器
	arrowRepeatTimer    float64 // 箭头重复显示计时器（秒）
	arrowRepeatInterval float64 // 箭头重复间隔（秒），粒子效果播放1秒后重新创建

	// 僵尸波次管理（教学关卡专用）
	waveDelayTimer float64 // 波次延迟计时器（秒）
	lastWaveKilled bool    // 上一波是否已全部击杀
}

// NewTutorialSystem 创建教学系统实例
// 参数：
//   - em: EntityManager 实例
//   - gs: GameState 实例
//   - rm: ResourceManager 实例
//   - lgs: LawnGridSystem 实例（用于控制草坪闪烁）
//   - sss: SunSpawnSystem 实例（用于启用阳光自动生成）
//   - wss: WaveSpawnSystem 实例（用于激活预生成的僵尸）
//   - levelConfig: 关卡配置（包含 tutorialSteps）
//
// 返回：
//   - *TutorialSystem: 系统实例
func NewTutorialSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, lgs *LawnGridSystem, sss *SunSpawnSystem, wss *WaveSpawnSystem, levelConfig *config.LevelConfig) *TutorialSystem {
	// 创建教学实体
	tutorialEntity := em.CreateEntity()
	ecs.AddComponent(em, tutorialEntity, &components.TutorialComponent{
		CurrentStepIndex:      0,
		CompletedSteps:        make(map[string]bool),
		IsActive:              true,
		TutorialSteps:         levelConfig.TutorialSteps, // 复制配置
		HighlightedCardEntity: 0,                         // Story 8.2.1: 初始无高亮卡片
		FlashTimer:            0,
		FlashCycleDuration:    0.8, // Story 8.2.1: 闪烁周期0.8秒
	})

	log.Printf("[TutorialSystem] Initialized with %d tutorial steps", len(levelConfig.TutorialSteps))

	return &TutorialSystem{
		entityManager:        em,
		gameState:            gs,
		resourceManager:      rm,
		lawnGridSystem:       lgs,
		sunSpawnSystem:       sss, // 保存 SunSpawnSystem 引用
		waveSpawnSystem:      wss, // 保存 WaveSpawnSystem 引用
		tutorialEntity:       tutorialEntity,
		textEntity:           0, // 未创建
		arrowIndicatorEntity: 0, // 未创建
		cardHighlightEntity:  0, // 未创建
		lastSunAmount:        gs.GetSun(),
		lastPlantCount:       0,
		lastZombieCount:      0,
		plantCount:           0, // 初始化植物计数
		initialized:          false,
		sunSpawned:           false, // 阳光未生成
		lastSunCount:         0,     // 初始化阳光实体计数
		lastTextDisplayTime:  0,     // 初始化文本显示时间
		arrowRepeatTimer:     0,     // 定时器初始化
		arrowRepeatInterval:  1.0,   // Story 8.2.1: 箭头每1.0秒重复一次（粒子播放1秒），无缝衔接避免闪烁
		waveDelayTimer:       0,     // 波次延迟计时器初始化
		lastWaveKilled:       false, // 初始化为false
	}
}

// Update 更新教学系统状态
// 参数：
//   - dt: 时间增量（秒）
//
// 功能：
//   - 检查当前步骤的触发条件
//   - 显示/隐藏教学文本
//   - 推进教学步骤
func (s *TutorialSystem) Update(dt float64) {
	// 更新文本显示时间计时器
	if s.lastTextDisplayTime > 0 {
		s.lastTextDisplayTime += dt
	}

	// 教学关卡专用：管理后续波次生成（在教学步骤完成后）
	s.manageWaveSpawning(dt)

	// 获取教学组件
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, s.tutorialEntity)
	if !ok || !tutorial.IsActive {
		return // 教学已完成或不存在
	}

	// 检查是否还有未完成的步骤
	if tutorial.CurrentStepIndex >= len(tutorial.TutorialSteps) {
		// 所有步骤已完成，禁用教学系统
		tutorial.IsActive = false
		s.hideTutorialText()
		log.Printf("[TutorialSystem] All tutorial steps completed")
		return
	}

	// 获取当前步骤
	currentStep := tutorial.TutorialSteps[tutorial.CurrentStepIndex]

	// 检查触发条件
	if s.checkTriggerCondition(currentStep.Trigger) {
		// 标记步骤已完成
		tutorial.CompletedSteps[currentStep.Trigger] = true
		log.Printf("[TutorialSystem] Step %d triggered: %s", tutorial.CurrentStepIndex, currentStep.Trigger)

		// 从 LawnStrings 获取文本
		text := ""
		if s.gameState.LawnStrings != nil {
			text = s.gameState.LawnStrings.GetString(currentStep.TextKey)
		} else {
			// 如果 LawnStrings 未加载，显示键名（调试用）
			text = "[" + currentStep.TextKey + "]"
		}

		// 显示教学文本
		s.showTutorialText(text)
		s.lastTextDisplayTime = 0.01 // 重置计时器（从0.01开始，避免0值判断）

		// 方案A+：根据步骤触发器显示/隐藏箭头指示符和卡片闪烁
		switch currentStep.Trigger {
		case "gameStart":
			// 步骤1：游戏开始，显示箭头指向豌豆射手卡片（原版设计）
			cardID := s.findPeashooterCard()
			log.Printf("[TutorialSystem] gameStart: findPeashooterCard returned ID=%d", cardID)
			if cardID != 0 {
				s.showArrowIndicator(cardID)
				// Story 8.2.1: 启用卡片闪光效果
				s.highlightPlantCard(cardID)
			} else {
				log.Println("[TutorialSystem] WARNING: Cannot find peashooter card!")
			}

		case "plantPlaced":
			// 步骤3：种植第一个豌豆射手后，禁用草坪闪烁，启用阳光自动生成，生成一颗阳光
			s.lawnGridSystem.DisableFlash() // 禁用草坪闪烁
			log.Printf("[TutorialSystem] Lawn flash disabled (plantPlaced)")
			s.sunSpawnSystem.Enable() // 启用阳光自动生成（原版机制：种植后开始掉落阳光）
			s.spawnSkyFallingSun()
			s.sunSpawned = true // 标记阳光已生成（触发下一步骤）
			log.Println("[TutorialSystem] Spawned first sun after planting peashooter, sunSpawned=true, auto spawn ENABLED")

		case "sunClicked":
			// 步骤5：收集第一颗阳光后，生成第二颗阳光（阳光自动生成已在 plantPlaced 启用）
			s.spawnSkyFallingSun()
			log.Println("[TutorialSystem] Spawned second sun after clicking first sun")

		case "secondPlantPlaced":
			// 步骤9：种植第二个豌豆射手后，禁用草坪闪烁，开始生成僵尸
			s.lawnGridSystem.DisableFlash() // 禁用草坪闪烁
			log.Printf("[TutorialSystem] Lawn flash disabled (secondPlantPlaced)")
			s.spawnTutorialZombies()
			log.Println("[TutorialSystem] Started spawning zombies after second plant")

		case "seedClicked":
			// 步骤2：点击卡片后，隐藏箭头，启用草坪闪烁
			s.hideArrowIndicator()
			s.unhighlightPlantCard()       // Story 8.2.1: 隐藏卡片闪光
			s.lawnGridSystem.EnableFlash() // 启用草坪闪烁效果（由明变暗）
			log.Printf("[TutorialSystem] Lawn flash enabled (seedClicked)")

		case "cooldownFinished":
			// 步骤6（旧版）：卡片冷却完成，再次显示箭头
			if cardID := s.findPeashooterCard(); cardID != 0 {
				s.showArrowIndicator(cardID)
				// Story 8.2.1: 启用卡片闪光效果
				s.highlightPlantCard(cardID)
			}

		case "enoughSunAndCooldown":
			// 步骤6（新版）：阳光足够且卡片冷却完成，显示箭头指向豌豆射手卡片
			if cardID := s.findPeashooterCard(); cardID != 0 {
				s.showArrowIndicator(cardID)
				s.highlightPlantCard(cardID) // Story 8.2.1: 启用卡片闪光效果
				log.Printf("[TutorialSystem] enoughSunAndCooldown: showing arrow to peashooter card")
			}

		case "secondSeedClicked":
			// 步骤8：第二次点击卡片，隐藏箭头，启用草坪闪烁
			s.hideArrowIndicator()
			s.unhighlightPlantCard()       // Story 8.2.1: 隐藏卡片闪光
			s.lawnGridSystem.EnableFlash() // 启用草坪闪烁效果（由明变暗）
			log.Printf("[TutorialSystem] Lawn flash enabled (secondSeedClicked)")
		}

		// 移动到下一步
		tutorial.CurrentStepIndex++
	}

	// 更新教学文本显示时间
	s.updateTextDisplayTime(dt)

	// Story 8.2.1: 更新卡片闪烁计时器
	if tutorial.HighlightedCardEntity != 0 {
		tutorial.FlashTimer += dt
	}

	// 重复显示箭头和闪光效果（因为粒子效果只播放1秒）
	s.updateArrowRepeat(dt, currentStep)

	// 更新状态跟踪变量（用于下一帧检测变化）
	s.updateTrackingState()
}

// checkTriggerCondition 检查触发条件是否满足
// 参数：
//   - trigger: 触发器ID（如 "sunClicked", "plantPlaced"）
//
// 返回：
//   - bool: 条件是否满足
func (s *TutorialSystem) checkTriggerCondition(trigger string) bool {
	switch trigger {
	case "gameStart":
		// 游戏开始时触发（只触发一次，且需要铺草皮动画完成后）
		if !s.initialized && s.soddingComplete {
			s.initialized = true
			return true
		}
		return false

	case "sunClicked":
		// 检查阳光是否增加（玩家点击了阳光）
		currentSun := s.gameState.GetSun()
		if currentSun > s.lastSunAmount {
			return true
		}
		return false

	case "sunSpawned":
		// 检查是否有新的阳光实体生成（由TutorialSystem在种植后触发）
		// 直接检查标志位（在 plantPlaced case 中设置）
		if s.sunSpawned {
			s.sunSpawned = false // 重置标志（避免重复触发）
			return true
		}
		return false

	case "enoughSun":
		// 检查阳光是否达到 100（足够种植豌豆射手）
		return s.gameState.GetSun() >= 100

	case "seedClicked":
		// 检查是否进入种植模式（玩家点击了植物卡片）
		isPlanting, _ := s.gameState.GetPlantingMode()
		return isPlanting

	case "plantPlaced":
		// 检查是否有新的植物实体创建
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		if len(plantEntities) > s.lastPlantCount {
			return true
		}
		return false

	case "zombieSpawned":
		// 检查是否有僵尸生成（通过 BehaviorComponent 查询）
		behaviorEntities := ecs.GetEntitiesWith1[*components.BehaviorComponent](s.entityManager)
		for _, entity := range behaviorEntities {
			behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entity)
			if ok && (behavior.Type == components.BehaviorZombieBasic ||
				behavior.Type == components.BehaviorZombieConehead ||
				behavior.Type == components.BehaviorZombieBuckethead) {
				return true
			}
		}
		return false

	case "cooldownFinished":
		// 检测豌豆射手卡片冷却完成
		cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)
		for _, cardID := range cardEntities {
			card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, cardID)
			if ok && card.PlantType == components.PlantPeashooter && card.CurrentCooldown <= 0 {
				return true
			}
		}
		return false

	case "enoughSunAndCooldown":
		// 检测阳光≥100 且 豌豆射手卡片冷却完成（两个条件同时满足）
		if s.gameState.GetSun() < 100 {
			return false
		}
		cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)
		for _, cardID := range cardEntities {
			card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, cardID)
			if ok && card.PlantType == components.PlantPeashooter && card.CurrentCooldown <= 0 {
				return true
			}
		}
		return false

	case "enoughSunNotPlanting":
		// 阳光≥100 且未进入种植模式（提醒玩家继续种植）
		isPlanting, _ := s.gameState.GetPlantingMode()
		return s.gameState.GetSun() >= 100 && !isPlanting

	case "sunClickedWhenEnough":
		// 阳光≥100时触发（简化逻辑，修复时序bug）
		// Bug修复（v1.11）：原逻辑要求"阳光增加 + ≥100 + 文本显示≥3秒"同一帧满足
		// 但阳光增加只有1帧窗口，如果那一帧文本时长不足3秒，就永远错过触发
		// 新逻辑：阳光≥100时立即触发，不依赖"点击收集"的精确帧
		currentSun := s.gameState.GetSun()
		return currentSun >= 100

	case "secondSeedClicked":
		// 第二次点击豌豆射手卡片（需检查已种植1个植物）
		isPlanting, _ := s.gameState.GetPlantingMode()
		return s.plantCount == 1 && isPlanting

	case "secondPlantPlaced":
		// 种植第二个植物
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		return len(plantEntities) >= 2

	default:
		return false
	}
}

// showTutorialText 显示教学文本
// 参数：
//   - text: 教学文本内容
func (s *TutorialSystem) showTutorialText(text string) {
	// 如果教学文本实体已存在，更新文本
	if s.textEntity != 0 {
		if textComp, ok := ecs.GetComponent[*components.TutorialTextComponent](s.entityManager, s.textEntity); ok {
			textComp.Text = text
			textComp.DisplayTime = 0
			log.Printf("[TutorialSystem] Updated tutorial text: %s", text)
			return
		}
	}

	// 创建新的教学文本实体
	s.textEntity = s.entityManager.CreateEntity()
	ecs.AddComponent(s.entityManager, s.textEntity, &components.TutorialTextComponent{
		Text:            text,
		DisplayTime:     0,
		MaxDisplayTime:  0, // 无限显示，直到步骤完成
		BackgroundAlpha: 0.7,
	})
	ecs.AddComponent(s.entityManager, s.textEntity, &components.UIComponent{
		State: components.UINormal,
	})
	// 位置将在 RenderSystem 中根据屏幕尺寸动态计算（底部居中）

	log.Printf("[TutorialSystem] Displayed tutorial text: %s", text)
}

// hideTutorialText 隐藏教学文本
func (s *TutorialSystem) hideTutorialText() {
	if s.textEntity != 0 {
		s.entityManager.DestroyEntity(s.textEntity)
		s.textEntity = 0
		log.Printf("[TutorialSystem] Hidden tutorial text")
	}
}

// updateTextDisplayTime 更新教学文本显示时间
// 参数：
//   - dt: 时间增量（秒）
func (s *TutorialSystem) updateTextDisplayTime(dt float64) {
	if s.textEntity == 0 {
		return
	}

	textComp, ok := ecs.GetComponent[*components.TutorialTextComponent](s.entityManager, s.textEntity)
	if !ok {
		return
	}

	textComp.DisplayTime += dt

	// 如果有最大显示时间限制（当前为0，表示无限显示）
	if textComp.MaxDisplayTime > 0 && textComp.DisplayTime >= textComp.MaxDisplayTime {
		s.hideTutorialText()
	}
}

// updateTrackingState 更新状态跟踪变量（用于下一帧检测变化）
func (s *TutorialSystem) updateTrackingState() {
	s.lastSunAmount = s.gameState.GetSun()

	plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
	currentPlantCount := len(plantEntities)

	// 检测新植物种植（植物数量增加）
	if currentPlantCount > s.lastPlantCount {
		s.plantCount++ // 增加种植计数
		log.Printf("[TutorialSystem] Plant placed, total plantCount: %d", s.plantCount)
	}

	s.lastPlantCount = currentPlantCount

	// 更新阳光实体计数（用于检测新阳光生成）
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](s.entityManager)
	s.lastSunCount = len(sunEntities)

	// 统计僵尸数量（通过 BehaviorComponent）
	zombieCount := 0
	behaviorEntities := ecs.GetEntitiesWith1[*components.BehaviorComponent](s.entityManager)
	for _, entity := range behaviorEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entity)
		if ok && (behavior.Type == components.BehaviorZombieBasic ||
			behavior.Type == components.BehaviorZombieConehead ||
			behavior.Type == components.BehaviorZombieBuckethead) {
			zombieCount++
		}
	}
	s.lastZombieCount = zombieCount
}

// showArrowIndicator 在目标实体上方显示箭头指示符（使用粒子效果）
// 参数：
//   - targetEntity: 目标实体ID（如植物卡片）
func (s *TutorialSystem) showArrowIndicator(targetEntity ecs.EntityID) {
	// 隐藏已存在的箭头
	s.hideArrowIndicator()

	// 获取目标实体位置
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, targetEntity)
	if !ok {
		log.Printf("[TutorialSystem] Target entity has no PositionComponent, cannot show arrow")
		return
	}

	// Story 8.4 Bug修复：从 PlantCardComponent 读取实际的 CardScale，而不是使用硬编码常量
	// 因为不同场景可能使用不同的缩放因子
	card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, targetEntity)
	if !ok {
		log.Printf("[TutorialSystem] Target entity has no PlantCardComponent, cannot show arrow")
		return
	}

	// 使用实际的卡片缩放计算尺寸（用于调试日志）
	// 卡片背景图原始尺寸：100x140
	const cardOriginalWidth = 100.0
	const cardOriginalHeight = 140.0
	cardScaledWidth := cardOriginalWidth * card.CardScale
	cardScaledHeight := cardOriginalHeight * card.CardScale

	// 箭头位置说明：
	// - 发射器位置直接使用卡片左上角坐标（pos.X, pos.Y）
	// - UpsellArrow.xml 中的 EmitterOffsetX=25, EmitterOffsetY=80 会自动调整粒子位置
	// - 最终粒子出现在：(pos.X + 25, pos.Y + 80)，刚好在卡片下方指向卡片
	// - 这是原版设计的正确位置，无需手动调整
	arrowX := pos.X // 发射器X坐标（卡片左上角）
	arrowY := pos.Y // 发射器Y坐标（卡片左上角）

	log.Printf("[TutorialSystem] Arrow indicator: cardScale=%.2f, cardSize=(%.1f x %.1f), cardPos=(%.1f, %.1f), emitterPos=(%.1f, %.1f)",
		card.CardScale, cardScaledWidth, cardScaledHeight, pos.X, pos.Y, arrowX, arrowY)

	// 创建箭头粒子效果（使用 UpsellArrow.xml - 教学专用箭头）
	// 根据原版设计："箭头从草坪指向豌豆射手卡片"，箭头应在卡片下方向上指
	arrowEntity, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"UpsellArrow", // 粒子效果名称（教学专用箭头，不带光晕）
		arrowX, arrowY,
		// 注意：AngleOffset 只影响运动方向，不影响图片旋转
		// 图片旋转需要手动设置粒子的 Rotation 字段（见下方）
	)

	if err != nil {
		log.Printf("[TutorialSystem] Failed to create arrow indicator: %v", err)
		return
	}

	// 标记为UI粒子（不受摄像机影响）
	// 添加 UIComponent 到发射器，这样生成的粒子会自动继承 UI 标记
	ecs.AddComponent(s.entityManager, arrowEntity, &components.UIComponent{
		State: components.UINormal,
	})

	// 手动设置粒子旋转角度（因为 UpsellArrow.xml 中没有 ParticleSpinAngle）
	// IMAGE_DOWNARROW 向下，需要旋转180度让它向上
	if emitter, ok := ecs.GetComponent[*components.EmitterComponent](s.entityManager, arrowEntity); ok {
		emitter.ParticleRotationOverride = 180.0 // 旋转180度，向下箭头变向上箭头
		log.Printf("[TutorialSystem] Set particle rotation override to 180°")
	}

	// 保存箭头实体ID，用于后续移除
	s.arrowIndicatorEntity = arrowEntity
	log.Printf("[TutorialSystem] Arrow indicator shown at (%.1f, %.1f), cardPos=(%.1f, %.1f)", arrowX, arrowY, pos.X, pos.Y)
}

// hideArrowIndicator 隐藏箭头指示符
func (s *TutorialSystem) hideArrowIndicator() {
	if s.arrowIndicatorEntity != 0 {
		s.entityManager.DestroyEntity(s.arrowIndicatorEntity)
		s.arrowIndicatorEntity = 0
		log.Printf("[TutorialSystem] Arrow indicator hidden")
	}
}

// OnSoddingComplete 通知教学系统铺草皮动画完成
// 由 GameScene 在铺草皮动画完成回调中调用
func (s *TutorialSystem) OnSoddingComplete() {
	s.soddingComplete = true
	log.Printf("[TutorialSystem] Sodding animation complete, gameStart can now trigger")
}

// findPeashooterCard 查找豌豆射手卡片实体
// 返回：
//   - ecs.EntityID: 卡片实体ID（0表示未找到）
func (s *TutorialSystem) findPeashooterCard() ecs.EntityID {
	cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)
	for _, cardID := range cardEntities {
		card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, cardID)
		if ok && card.PlantType == components.PlantPeashooter {
			return cardID
		}
	}
	return 0
}

// highlightPlantCard 显示卡片闪烁效果（Story 8.2.1: 使用遮罩式闪烁）
// 参数：
//   - targetEntity: 目标卡片实体ID
func (s *TutorialSystem) highlightPlantCard(targetEntity ecs.EntityID) {
	// 获取教学组件
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, s.tutorialEntity)
	if !ok {
		log.Printf("[TutorialSystem] Cannot get TutorialComponent for highlighting card")
		return
	}

	// 设置高亮卡片
	tutorial.HighlightedCardEntity = targetEntity
	tutorial.FlashTimer = 0 // 重置闪烁计时器
	log.Printf("[TutorialSystem] Card highlight enabled for entity %d", targetEntity)
}

// unhighlightPlantCard 隐藏卡片闪烁效果
func (s *TutorialSystem) unhighlightPlantCard() {
	// 获取教学组件
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, s.tutorialEntity)
	if !ok {
		return
	}

	// 清除高亮卡片
	tutorial.HighlightedCardEntity = 0
	tutorial.FlashTimer = 0
	log.Printf("[TutorialSystem] Card highlight disabled")
}

// updateArrowRepeat 更新箭头重复显示逻辑
// 参数：
//   - dt: 时间增量（秒）
//   - currentStep: 当前教学步骤（未使用，保留用于扩展）
//
// 功能：
//   - 因为粒子效果只播放1秒（SystemLoops=1, SystemDuration=100厘秒）
//   - 只要箭头实体存在，就定期重新创建箭头和闪光效果
func (s *TutorialSystem) updateArrowRepeat(dt float64, currentStep config.TutorialStep) {
	// 只要箭头实体存在，就重复显示（不再检查闪光实体）
	needsRepeat := (s.arrowIndicatorEntity != 0)

	if !needsRepeat {
		s.arrowRepeatTimer = 0 // 重置定时器
		return
	}

	// Story 8.2.1 修复：移除阳光检查逻辑，箭头应该持续显示直到玩家点击卡片
	// 原有的 L609-614 逻辑会导致箭头在阳光<100时被错误隐藏
	// 教学流程中，箭头在 gameStart 时显示（阳光=0），应该持续到 seedClicked

	// 更新定时器
	s.arrowRepeatTimer += dt

	// 每隔 arrowRepeatInterval 秒重新创建箭头
	if s.arrowRepeatTimer >= s.arrowRepeatInterval {
		s.arrowRepeatTimer = 0 // 重置定时器

		// 找到豌豆射手卡片并重新显示箭头
		cardID := s.findPeashooterCard()
		if cardID != 0 {
			// 先隐藏旧的箭头（避免重复创建）
			s.hideArrowIndicator()

			// 重新创建箭头（不创建闪光）
			s.showArrowIndicator(cardID)
			log.Printf("[TutorialSystem] Repeated arrow (timer reset)")
		}
	}
}

// spawnSkyFallingSun 在教学关卡中生成一颗从天空掉落的阳光
// 这是教学关卡的特殊机制：阳光不是定时生成，而是由教学步骤触发
func (s *TutorialSystem) spawnSkyFallingSun() {
	// 生成随机X坐标（草坪范围内）
	minX := 100.0
	maxX := 700.0
	startX := minX + rand.Float64()*(maxX-minX) // 随机位置

	// 生成随机Y坐标（落地位置）
	minY := 150.0
	maxY := 450.0
	targetY := minY + rand.Float64()*(maxY-minY) // 随机位置

	// 边界检查：确保阳光完整显示在屏幕内
	sunRadius := 40.0 // 阳光半径
	if startX < sunRadius {
		startX = sunRadius
	}
	if startX > 800-sunRadius {
		startX = 800 - sunRadius
	}
	if targetY < sunRadius {
		targetY = sunRadius
	}
	if targetY > 600-sunRadius {
		targetY = 600 - sunRadius
	}

	// 创建阳光实体
	sunID := entities.NewSunEntity(s.entityManager, s.resourceManager, startX, targetY)
	log.Printf("[TutorialSystem] Created tutorial sun entity ID=%d at X=%.1f, targetY=%.1f", sunID, startX, targetY)

	// Sun.reanim 只有轨道(Sun1, Sun2, Sun3),没有动画定义
	// 使用 AnimationCommand 组件播放配置的"idle"组合（包含所有3个轨道）
	ecs.AddComponent(s.entityManager, sunID, &components.AnimationCommandComponent{
		UnitID:    "sun",
		ComboName: "idle",
		Processed: false,
	})
}

// spawnTutorialZombies 开始生成教学关卡的僵尸波次
// 激活第一波预生成的僵尸，后续波次由 manageWaveSpawning 管理
func (s *TutorialSystem) spawnTutorialZombies() {
	// 获取关卡配置
	levelConfig := s.gameState.CurrentLevel
	if levelConfig == nil || len(levelConfig.Waves) == 0 {
		log.Printf("[TutorialSystem] ERROR: No level config or waves found")
		return
	}

	if len(levelConfig.Waves) > 0 && !s.gameState.IsWaveSpawned(0) {
		// 使用 WaveSpawnSystem 激活第一波僵尸
		activatedCount := s.waveSpawnSystem.ActivateWave(0)
		log.Printf("[TutorialSystem] Activated wave 1 (%d pre-spawned zombies)", activatedCount)

		// 标记波次已生成
		s.gameState.MarkWaveSpawned(0)
		// s.gameState.IncrementZombiesSpawned(activatedCount) // BUG: 导致僵尸计数重复增加
	}

	// 后续波次将由 manageWaveSpawning 自动处理
	log.Printf("[TutorialSystem] Zombie spawning started, %d waves total", len(levelConfig.Waves))
}

// manageWaveSpawning 管理教学关卡的后续波次生成
// 机制：上一波僵尸全部消灭后，等待 MinDelay 秒后触发下一波
func (s *TutorialSystem) manageWaveSpawning(dt float64) {
	// 检查关卡配置
	if s.gameState.CurrentLevel == nil || len(s.gameState.CurrentLevel.Waves) == 0 {
		return
	}

	// 获取当前场上的僵尸数量
	zombiesOnField := s.gameState.TotalZombiesSpawned - s.gameState.ZombiesKilled
	currentWaveIndex := s.gameState.CurrentWaveIndex

	// 检查是否上一波已全部击杀
	if zombiesOnField == 0 && s.gameState.TotalZombiesSpawned > 0 {
		if !s.lastWaveKilled {
			// 刚击杀完毕，开始延迟计时
			s.lastWaveKilled = true
			s.waveDelayTimer = 0
			log.Printf("[TutorialSystem] Wave cleared, starting delay timer (WaveIndex=%d, Spawned=%d, Killed=%d)",
				currentWaveIndex, s.gameState.TotalZombiesSpawned, s.gameState.ZombiesKilled)
		} else {
			// 延迟计时中
			s.waveDelayTimer += dt
		}
	} else {
		// 如果场上有僵尸，重置标志
		if zombiesOnField > 0 {
			s.lastWaveKilled = false
		}
	}

	// 检查是否需要生成下一波
	if currentWaveIndex < len(s.gameState.CurrentLevel.Waves) &&
		!s.gameState.IsWaveSpawned(currentWaveIndex) &&
		s.lastWaveKilled {

		// 获取波次配置
		waveConfig := s.gameState.CurrentLevel.Waves[currentWaveIndex]

		// 检查延迟时间是否已过
		if s.waveDelayTimer >= waveConfig.MinDelay {
			// 显示"最后一波"提示（在最后一波前）
			if currentWaveIndex == len(s.gameState.CurrentLevel.Waves)-1 {
				s.showFinalWaveWarning()
			}

			log.Printf("[TutorialSystem] Activating wave %d after %.1f seconds delay", currentWaveIndex+1, s.waveDelayTimer)
			activatedCount := s.waveSpawnSystem.ActivateWave(currentWaveIndex)
			log.Printf("[TutorialSystem] Spawned wave %d (%d zombies activated)", currentWaveIndex+1, activatedCount)

			// 标记波次已生成
			s.gameState.MarkWaveSpawned(currentWaveIndex)
			// s.gameState.IncrementZombiesSpawned(activatedCount) // BUG: 导致僵尸计数重复增加

			// 重置延迟计时器和标志
			s.waveDelayTimer = 0
			s.lastWaveKilled = false
		}
	}
}

// showFinalWaveWarning 显示"最后一波"提示
// 使用 FinalWave.reanim 动画效果
func (s *TutorialSystem) showFinalWaveWarning() {
	// 加载 FinalWave.reanim
	reanimXML := s.resourceManager.GetReanimXML("FinalWave")
	partImages := s.resourceManager.GetReanimPartImages("FinalWave")

	if reanimXML == nil {
		log.Printf("[TutorialSystem] WARNING: FinalWave.reanim not found, skipping animation")
		return
	}

	// 创建动画实体
	finalWaveEntity := s.entityManager.CreateEntity()

	// 添加位置组件（屏幕中央）
	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2
	ecs.AddComponent(s.entityManager, finalWaveEntity, &components.PositionComponent{
		X: centerX,
		Y: centerY,
	})

	// 添加 Reanim 组件（初始状态，将被 PlayCombo 覆盖）
	ecs.AddComponent(s.entityManager, finalWaveEntity, &components.ReanimComponent{
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		CurrentAnimations: []string{}, // 将由 PlayCombo 设置
		IsLooping:         true,       // 将由 PlayCombo 根据配置设置为 false
	})

	// 添加 FinalWaveWarningComponent（用于自动删除）
	ecs.AddComponent(s.entityManager, finalWaveEntity, &components.FinalWaveWarningComponent{
		DisplayTime: 2.0, // 显示 2 秒后自动删除
		ElapsedTime: 0.0,
	})

	// 标记为UI元素（不受摄像机影响）
	ecs.AddComponent(s.entityManager, finalWaveEntity, &components.UIComponent{
		State: components.UINormal,
	})

	// 使用配置化的 combo 播放非循环动画
	ecs.AddComponent(s.entityManager, finalWaveEntity, &components.AnimationCommandComponent{
		UnitID:    "finalwave",
		ComboName: "warning",
		Processed: false,
	})
	log.Printf("[TutorialSystem] Final wave warning displayed at (%.1f, %.1f), using combo: finalwave/warning (loop: false)", centerX, centerY)
}
