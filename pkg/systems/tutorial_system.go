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
	lawnGridSystem       *LawnGridSystem // 用于控制草坪闪烁效果（Story 8.2）
	sunSpawnSystem       *SunSpawnSystem // 用于启用阳光自动生成（Story 8.2）
	levelSystem          *LevelSystem    // 用于访问 WaveTimingSystem（统一波次管理）
	tutorialEntity       ecs.EntityID    // 教学实体ID
	textEntity           ecs.EntityID    // 教学文本实体ID（用于显示/隐藏）
	arrowIndicatorEntity ecs.EntityID    // 箭头指示符实体ID（用于显示/隐藏）
	cardHighlightEntity  ecs.EntityID    // 卡片闪烁效果实体ID（用于显示/隐藏）

	// 状态跟踪变量（用于检测变化）
	lastSunAmount       int     // 上一帧的阳光数量
	lastPlantCount      int     // 上一帧的植物数量
	lastZombieCount     int     // 上一帧的僵尸数量
	plantCount          int     // 当前种植的植物总数（用于第二次种植检测）
	newPlantThisFrame   bool    // 本帧是否有新植物种植（用于 plantPlaced 触发器）
	sunClickedThisFrame bool    // 本帧是否收集了阳光（用于 sunClicked 触发器）
	initialized         bool    // 是否已初始化（用于 gameStart 触发）
	soddingComplete     bool    // 铺草皮动画是否完成（用于延迟 gameStart 触发）
	sunSpawned          bool    // 第一颗阳光是否已生成（用于 sunSpawned 触发）- 已废弃，改用 newSunThisFrame
	newSunThisFrame     bool    // 本帧是否有新阳光生成（用于 sunSpawned 触发器）
	lastSunCount        int     // 上一帧的阳光实体数量（用于检测新阳光生成）
	lastTextDisplayTime float64 // 上次教学文本显示的时间（用于时长检测，防止文本闪烁）

	// 向日葵教学相关（Level 1-2）
	sunflowerCount   int     // 向日葵种植计数
	stepTimeElapsed  float64 // 当前步骤经过时间（用于超时触发）
	sunSpawnObserved bool    // 是否观察到了阳光生成事件（用于 sunSpawned 触发器防止事件丢失）
}

// NewTutorialSystem 创建教学系统实例
// 参数：
//   - em: EntityManager 实例
//   - gs: GameState 实例
//   - rm: ResourceManager 实例
//   - lgs: LawnGridSystem 实例（用于控制草坪闪烁）
//   - sss: SunSpawnSystem 实例（用于启用阳光自动生成）
//   - levelConfig: 关卡配置（包含 tutorialSteps）
//
// 返回：
//   - *TutorialSystem: 系统实例
//
// 注意：创建后需调用 SetLevelSystem() 设置 LevelSystem 引用，用于访问 WaveTimingSystem
func NewTutorialSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, lgs *LawnGridSystem, sss *SunSpawnSystem, levelConfig *config.LevelConfig) *TutorialSystem {
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
		levelSystem:          nil, // 通过 SetLevelSystem 设置（避免循环依赖）
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
		sunflowerCount:       0,     // 向日葵计数初始化
		stepTimeElapsed:      0,     // 步骤计时器初始化
		sunSpawnObserved:     false, // 初始化未观察到阳光
	}
}

// SetLevelSystem 设置 LevelSystem 引用
// 用于访问 WaveTimingSystem，实现统一波次管理
// 由于 TutorialSystem 和 LevelSystem 创建顺序的原因，需要通过 setter 设置
func (s *TutorialSystem) SetLevelSystem(ls *LevelSystem) {
	s.levelSystem = ls
	log.Printf("[TutorialSystem] LevelSystem reference set")
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

	// 波次管理现在统一由 WaveTimingSystem 处理（第一波触发后自动恢复计时器）

	// 获取教学组件
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, s.tutorialEntity)
	if !ok || !tutorial.IsActive {
		return // 教学已完成或不存在
	}

	// 检查是否还有未完成的步骤
	if tutorial.CurrentStepIndex >= len(tutorial.TutorialSteps) {
		// 所有步骤已完成，禁用教学系统
		// 但不立即隐藏文本，让最后一条文本显示足够时间
		if s.lastTextDisplayTime >= config.TutorialTextMinDisplayTime {
			tutorial.IsActive = false
			s.hideTutorialText()
			log.Printf("[TutorialSystem] All tutorial steps completed")
		}
		return
	}

	// 获取当前步骤
	currentStep := tutorial.TutorialSteps[tutorial.CurrentStepIndex]

	// 更新状态跟踪变量
	// 注意：必须在检查触发条件之前调用，因为 Level 1-2 的触发器依赖 sunflowerCount
	// 但 plantPlaced 触发器需要特殊处理（见 checkTriggerCondition 中的实现）
	s.updateTrackingState()

	// 更新步骤计时器（用于超时触发器）
	s.stepTimeElapsed += dt

	// 检查触发条件
	if s.checkTriggerCondition(currentStep.Trigger) {
		// 标记步骤已完成
		tutorial.CompletedSteps[currentStep.Trigger] = true
		log.Printf("[TutorialSystem] Step %d triggered: %s", tutorial.CurrentStepIndex, currentStep.Trigger)

		// 特殊处理：sunflowerReminder 触发时，如果已有≥3颗向日葵，跳过文本显示
		// 这是为了让玩家快速种植3颗向日葵时可以跳过提醒步骤
		skipTextDisplay := currentStep.Trigger == "sunflowerReminder" && s.sunflowerCount >= 3
		if skipTextDisplay {
			log.Printf("[TutorialSystem] Skipping sunflowerReminder text display (sunflowerCount=%d >= 3)", s.sunflowerCount)
		}

		// 特殊处理：secondSeedClicked 触发时，如果已有≥2个植物，跳过文本显示
		// 这是为了让玩家快速种植时，优先显示 ADVICE_ZOMBIE_ONSLAUGHT 而不是 ADVICE_CLICK_ON_GRASS
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		if currentStep.Trigger == "secondSeedClicked" && len(plantEntities) >= 2 {
			skipTextDisplay = true
			log.Printf("[TutorialSystem] Skipping secondSeedClicked text display (plantCount=%d >= 2), prioritizing ADVICE_ZOMBIE_ONSLAUGHT", len(plantEntities))
		}

		// 如果 textKey 为空，也跳过文本显示（如 hideSunflowerHint 步骤）
		if currentStep.TextKey == "" {
			skipTextDisplay = true
		}

		// 从 LawnStrings 获取文本
		text := ""
		if !skipTextDisplay {
			if s.gameState.LawnStrings != nil {
				text = s.gameState.LawnStrings.GetString(currentStep.TextKey)
			} else {
				// 如果 LawnStrings 未加载，显示键名（调试用）
				text = "[" + currentStep.TextKey + "]"
			}

			// 显示教学文本（根据 action 类型选择样式）
			if currentStep.Action == "sunflowerHint" {
				s.showTutorialTextAdvisory(text) // 提示性教学：更靠下，自动消失
			} else {
				s.showTutorialText(text) // 强制性教学：标准位置
			}
			s.lastTextDisplayTime = 0.01 // 重置计时器（从0.01开始，避免0值判断）
		}

		// 方案A+：根据步骤触发器显示/隐藏箭头指示符和卡片闪烁
		// 注意：如果 action 是 sunflowerHint，则跳过 trigger 中的箭头处理（由 action switch 处理）
		if currentStep.Action != "sunflowerHint" {
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
				// 不再手动设置 sunSpawned，改为在 updateTrackingState 中检测阳光实体变化
				log.Println("[TutorialSystem] Spawned first sun after planting peashooter, auto spawn ENABLED")

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
		}

		// 根据 action 处理向日葵教学（Level 1-2）
		switch currentStep.Action {
		case "sunflowerHint":
			// 如果是跳过文本显示的情况（已有≥3颗向日葵），不显示箭头
			if !skipTextDisplay {
				// 显示箭头指向向日葵卡片，卡片闪烁
				cardID := s.findSunflowerCard()
				if cardID != 0 {
					s.showArrowIndicator(cardID)
					s.highlightPlantCard(cardID)
					log.Printf("[TutorialSystem] sunflowerHint: showing arrow to sunflower card (ID=%d)", cardID)
				} else {
					log.Println("[TutorialSystem] WARNING: Cannot find sunflower card for sunflowerHint!")
				}
			}
			// 重置步骤计时器
			s.stepTimeElapsed = 0

		case "completeTutorial":
			// 向日葵教学完成，隐藏所有提示
			s.hideArrowIndicator()
			s.unhighlightPlantCard()
			s.hideTutorialText()
			tutorial.IsActive = false // 立即标记教学完成
			log.Printf("[TutorialSystem] Sunflower tutorial completed (sunflowerCount=%d)", s.sunflowerCount)
			return // 教学完成，直接返回，不再更新箭头

		case "hideSunflowerHint":
			// 隐藏箭头和卡片高亮，但保留教学文字让其自然超时消失
			s.hideArrowIndicator()
			s.unhighlightPlantCard()
			tutorial.IsActive = false // 标记教学完成，不再处理后续步骤
			log.Printf("[TutorialSystem] Sunflower hint hidden, text will auto-dismiss (sunflowerCount=%d)", s.sunflowerCount)
			return // 直接返回
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
		// 检查本帧是否收集了阳光（阳光数量增加）
		// 这是事件触发型，不需要等待最小显示时间
		return s.sunClickedThisFrame

	case "sunSpawned":
		// 检查本帧是否有新的阳光实体生成
		// 这是状态检测型，需要等待最小显示时间
		if !s.isMinDisplayTimeElapsed() {
			return false
		}
		// 只要本帧有新阳光，或者之前观察到了阳光生成事件（未被消费），就触发
		if s.newSunThisFrame || s.sunSpawnObserved {
			s.sunSpawnObserved = false // 消费事件
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
		// 检查本帧是否有新植物种植
		// 使用 newPlantThisFrame 标志（在 updateTrackingState 中设置）
		// 这样即使 updateTrackingState 在 checkTriggerCondition 之前调用，也能正确检测
		return s.newPlantThisFrame

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
		// 修复：如果玩家已经种植了≥2个植物，跳过此步骤
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		if len(plantEntities) >= 2 {
			return true // 玩家已种植多个植物，跳过此步骤
		}
		// 这是状态检测型，需要等待最小显示时间
		if !s.isMinDisplayTimeElapsed() {
			return false
		}
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
		// 这是状态检测型，需要等待最小显示时间
		if !s.isMinDisplayTimeElapsed() {
			return false
		}
		isPlanting, _ := s.gameState.GetPlantingMode()
		return s.gameState.GetSun() >= 100 && !isPlanting

	case "sunClickedWhenEnough":
		// 阳光≥100时触发
		// 修复：如果玩家已经种植了≥2个植物，跳过此步骤
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		if len(plantEntities) >= 2 {
			return true // 玩家已种植多个植物，跳过此步骤
		}
		currentSun := s.gameState.GetSun()
		return currentSun >= 100

	case "secondSeedClicked":
		// 第二次点击豌豆射手卡片
		// 修复：如果玩家已经种植了≥2个植物，跳过此步骤直接进入下一步
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		if len(plantEntities) >= 2 {
			return true // 玩家已种植多个植物，跳过此步骤
		}
		// 正常情况：等待玩家点击卡片
		isPlanting, _ := s.gameState.GetPlantingMode()
		return s.plantCount >= 1 && isPlanting

	case "secondPlantPlaced":
		// 种植第二个植物
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		return len(plantEntities) >= 2

	// 向日葵教学触发器（Level 1-2）
	case "sunflowerCount1OrTimeout":
		// 种植1颗向日葵或10秒超时
		return s.sunflowerCount >= 1 || s.stepTimeElapsed >= 10.0

	case "sunflowerCount2OrTimeout":
		// 种植2颗向日葵或10秒超时
		return s.sunflowerCount >= 2 || s.stepTimeElapsed >= 10.0

	case "sunflowerReminder":
		// 20秒后仍不足3颗向日葵，或已经种植≥3颗（跳过提醒直接完成）
		return (s.stepTimeElapsed >= 20.0 && s.sunflowerCount < 3) || s.sunflowerCount >= 3

	case "sunflowerCount3":
		// 种植≥3颗向日葵
		return s.sunflowerCount >= 3

	default:
		return false
	}
}

// showTutorialText 显示教学文本
// 参数：
//   - text: 教学文本内容
func (s *TutorialSystem) showTutorialText(text string) {
	s.showTutorialTextWithStyle(text, false)
}

// showTutorialTextAdvisory 显示提示性教学文本（更靠下，自动消失）
// 参数：
//   - text: 教学文本内容
func (s *TutorialSystem) showTutorialTextAdvisory(text string) {
	s.showTutorialTextWithStyle(text, true)
}

// showTutorialTextWithStyle 显示教学文本（内部方法）
// 参数：
//   - text: 教学文本内容
//   - isAdvisory: 是否为提示性教学（更靠下，自动消失）
func (s *TutorialSystem) showTutorialTextWithStyle(text string, isAdvisory bool) {
	// 如果教学文本实体已存在，更新文本
	if s.textEntity != 0 {
		if textComp, ok := ecs.GetComponent[*components.TutorialTextComponent](s.entityManager, s.textEntity); ok {
			textComp.Text = text
			textComp.DisplayTime = 0
			textComp.IsAdvisory = isAdvisory
			if isAdvisory {
				textComp.MaxDisplayTime = config.AdvisoryTutorialTextDisplayDuration
			} else {
				textComp.MaxDisplayTime = 0 // 无限显示
			}
			log.Printf("[TutorialSystem] Updated tutorial text: %s (advisory=%v)", text, isAdvisory)
			return
		}
	}

	// 计算最大显示时间
	maxDisplayTime := 0.0
	if isAdvisory {
		maxDisplayTime = config.AdvisoryTutorialTextDisplayDuration
	}

	// 创建新的教学文本实体
	s.textEntity = s.entityManager.CreateEntity()
	ecs.AddComponent(s.entityManager, s.textEntity, &components.TutorialTextComponent{
		Text:            text,
		DisplayTime:     0,
		MaxDisplayTime:  maxDisplayTime,
		BackgroundAlpha: 0.7,
		IsAdvisory:      isAdvisory,
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

// isMinDisplayTimeElapsed 检查是否已经过了最小文本显示时间
// 用于状态检测型触发器，确保事件触发型文本有足够时间显示
func (s *TutorialSystem) isMinDisplayTimeElapsed() bool {
	return s.lastTextDisplayTime >= config.TutorialTextMinDisplayTime
}

// updateTrackingState 更新状态跟踪变量（用于下一帧检测变化）
func (s *TutorialSystem) updateTrackingState() {
	// 检测阳光变化（玩家收集了阳光）
	currentSun := s.gameState.GetSun()
	if currentSun > s.lastSunAmount {
		s.sunClickedThisFrame = true
	} else {
		s.sunClickedThisFrame = false
	}
	s.lastSunAmount = currentSun

	plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
	currentPlantCount := len(plantEntities)

	// 检测新植物种植（植物数量增加）
	// 设置 newPlantThisFrame 标志，供 plantPlaced 触发器使用
	if currentPlantCount > s.lastPlantCount {
		s.plantCount++             // 增加种植计数
		s.newPlantThisFrame = true // 本帧有新植物
		log.Printf("[TutorialSystem] Plant placed, total plantCount: %d", s.plantCount)
	} else {
		s.newPlantThisFrame = false // 本帧无新植物
	}

	s.lastPlantCount = currentPlantCount

	// 统计向日葵数量（Level 1-2 教学用）
	s.sunflowerCount = 0
	for _, plantID := range plantEntities {
		plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID)
		if ok && plant.PlantType == components.PlantSunflower {
			s.sunflowerCount++
		}
	}

	// 更新阳光实体计数（用于检测新阳光生成）
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](s.entityManager)
	currentSunCount := len(sunEntities)
	if currentSunCount > s.lastSunCount {
		s.newSunThisFrame = true
		s.sunSpawnObserved = true // 记录事件，供 sunSpawned 触发器使用（防止事件在等待期间丢失）
	} else {
		s.newSunThisFrame = false
	}
	s.lastSunCount = currentSunCount

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
		// 修复：同时销毁发射器生成的粒子
		// 如果发射器被销毁但粒子是循环的(ParticleLoops=true)，粒子会残留导致重影
		if emitter, ok := ecs.GetComponent[*components.EmitterComponent](s.entityManager, s.arrowIndicatorEntity); ok {
			for _, particleID := range emitter.ActiveParticles {
				s.entityManager.DestroyEntity(particleID)
			}
			log.Printf("[TutorialSystem] Destroyed %d particles associated with arrow indicator", len(emitter.ActiveParticles))
		}

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

// findSunflowerCard 查找向日葵卡片实体
// 返回：
//   - ecs.EntityID: 卡片实体ID（0表示未找到）
func (s *TutorialSystem) findSunflowerCard() ecs.EntityID {
	cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)
	for _, cardID := range cardEntities {
		card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, cardID)
		if ok && card.PlantType == components.PlantSunflower {
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
//   - currentStep: 当前教学步骤
//
// 功能：
//   - SeedPacket/UpsellArrow 粒子效果配置了 SystemLoops=1，会自动循环
//   - 不再需要手动重建箭头，粒子系统会自动处理循环
//   - 此函数保留用于未来可能的位置更新需求
func (s *TutorialSystem) updateArrowRepeat(dt float64, currentStep config.TutorialStep) {
	// SystemLoops=1 已正确实现，粒子效果会自动循环
	// 不再需要定期重建箭头，否则会导致重复箭头
	// 保留此函数框架，以备未来需要动态更新箭头位置时使用
	_ = dt
	_ = currentStep
}

// spawnSkyFallingSun 在教学关卡中生成一颗从天空掉落的阳光
// 这是教学关卡的特殊机制：阳光不是定时生成，而是由教学步骤触发
func (s *TutorialSystem) spawnSkyFallingSun() {
	// 使用配置常量生成随机X坐标（与 SunSpawnSystem 一致）
	startX := config.SkyDropSunMinX + rand.Float64()*(config.SkyDropSunMaxX-config.SkyDropSunMinX)

	// 生成随机Y坐标（落地位置）- 使用更保守的范围确保阳光完整显示
	// 考虑阳光半径40px，避免阳光落在屏幕边缘
	targetY := config.SkyDropSunMinTargetY + rand.Float64()*(config.SkyDropSunMaxTargetY-config.SkyDropSunMinTargetY-80)

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
// 通过 WaveTimingSystem 触发第一波，后续波次由 WaveTimingSystem 统一管理
func (s *TutorialSystem) spawnTutorialZombies() {
	// 获取关卡配置
	levelConfig := s.gameState.CurrentLevel
	if levelConfig == nil || len(levelConfig.Waves) == 0 {
		log.Printf("[TutorialSystem] ERROR: No level config or waves found")
		return
	}

	// 通过 WaveTimingSystem 统一触发第一波
	if s.levelSystem != nil {
		wts := s.levelSystem.GetWaveTimingSystem()
		if wts != nil {
			// 立即触发第一波，同时恢复计时器管理后续波次
			waveIndex := wts.TriggerNextWaveImmediately()
			if waveIndex >= 0 {
				log.Printf("[TutorialSystem] Triggered wave %d via WaveTimingSystem", waveIndex+1)
			} else {
				log.Printf("[TutorialSystem] ERROR: Failed to trigger first wave via WaveTimingSystem")
			}
		} else {
			log.Printf("[TutorialSystem] ERROR: WaveTimingSystem not available")
		}
	} else {
		log.Printf("[TutorialSystem] ERROR: LevelSystem not set, cannot trigger waves")
	}

	log.Printf("[TutorialSystem] Zombie spawning started, %d waves total", len(levelConfig.Waves))
}
