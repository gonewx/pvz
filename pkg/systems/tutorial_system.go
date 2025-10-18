package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TutorialSystem 教学系统
// 管理关卡 1-1 的分步教学引导流程
// 检测触发条件，显示教学文本，推进教学步骤
type TutorialSystem struct {
	entityManager  *ecs.EntityManager
	gameState      *game.GameState
	tutorialEntity ecs.EntityID // 教学实体ID
	textEntity     ecs.EntityID // 教学文本实体ID（用于显示/隐藏）

	// 状态跟踪变量（用于检测变化）
	lastSunAmount   int  // 上一帧的阳光数量
	lastPlantCount  int  // 上一帧的植物数量
	lastZombieCount int  // 上一帧的僵尸数量
	initialized     bool // 是否已初始化（用于 gameStart 触发）
}

// NewTutorialSystem 创建教学系统实例
// 参数：
//   - em: EntityManager 实例
//   - gs: GameState 实例
//   - levelConfig: 关卡配置（包含 tutorialSteps）
//
// 返回：
//   - *TutorialSystem: 系统实例
func NewTutorialSystem(em *ecs.EntityManager, gs *game.GameState, levelConfig *config.LevelConfig) *TutorialSystem {
	// 创建教学实体
	tutorialEntity := em.CreateEntity()
	ecs.AddComponent(em, tutorialEntity, &components.TutorialComponent{
		CurrentStepIndex: 0,
		CompletedSteps:   make(map[string]bool),
		IsActive:         true,
		TutorialSteps:    levelConfig.TutorialSteps, // 复制配置
	})

	log.Printf("[TutorialSystem] Initialized with %d tutorial steps", len(levelConfig.TutorialSteps))

	return &TutorialSystem{
		entityManager:   em,
		gameState:       gs,
		tutorialEntity:  tutorialEntity,
		textEntity:      0, // 未创建
		lastSunAmount:   gs.GetSun(),
		lastPlantCount:  0,
		lastZombieCount: 0,
		initialized:     false,
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

		// 移动到下一步
		tutorial.CurrentStepIndex++
	}

	// 更新教学文本显示时间
	s.updateTextDisplayTime(dt)

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
		// 游戏开始时立即触发（只触发一次）
		if !s.initialized {
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
	s.lastPlantCount = len(plantEntities)

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
