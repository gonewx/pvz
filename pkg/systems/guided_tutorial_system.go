package systems

import (
	"image"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// GuidedTutorialSystem 强引导教学系统
// Story 19.3: 强引导教学系统
//
// 此系统负责 Level 1-5 的强制铲子教学阶段：
//   - 限制玩家操作，只允许白名单中的操作
//   - 监控空闲时间，超时后显示浮动箭头提示
//   - 监控植物数量，所有植物移除后触发转场
type GuidedTutorialSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// guidedEntity 强引导教学实体ID
	guidedEntity ecs.EntityID

	// 内部状态跟踪
	lastPlantCount int     // 上一帧的植物数量
	initialized    bool    // 系统是否已初始化
	lastShovelMode bool    // 上一帧的铲子模式状态（用于检测模式切换）
	totalTime      float64 // 总运行时间（用于调试）
}

// GuidedTutorialStateProvider 强引导教学状态提供者接口
// Story 19.3: 用于外部系统查询和控制强引导状态
// 遵循零耦合原则，系统不直接依赖 GameScene
type GuidedTutorialStateProvider interface {
	// IsGuidedTutorialActive 返回强引导模式是否激活
	IsGuidedTutorialActive() bool

	// IsOperationAllowed 检查操作是否在白名单中
	// 参数：
	//   - operation: 操作标识（如 "click_shovel", "click_plant"）
	// 返回：
	//   - true: 操作允许执行
	//   - false: 操作被屏蔽（静默忽略）
	IsOperationAllowed(operation string) bool

	// NotifyOperation 通知系统发生了某个操作
	// 用于重置空闲计时器
	// 参数：
	//   - operation: 操作标识
	NotifyOperation(operation string)
}

// ShovelSlotBoundsProvider 铲子槽位边界提供者接口
// 用于获取铲子槽位位置，避免直接依赖 GameScene
type ShovelSlotBoundsProvider interface {
	// GetShovelSlotBounds 获取铲子槽位边界（屏幕坐标）
	GetShovelSlotBounds() image.Rectangle
	// GetShovelIconBounds 获取铲子图标边界（屏幕坐标）
	// Story 19.x QA: 铲子图标在卡槽内居中，此方法返回图标的实际位置
	GetShovelIconBounds() image.Rectangle
}

// shovelSlotBoundsProvider 铲子槽位边界提供者引用
var guidedTutorialShovelProvider ShovelSlotBoundsProvider

// guidedTutorialStateProvider 强引导状态提供者引用
// 用于 InputSystem 等其他系统查询强引导状态
var guidedTutorialStateProvider GuidedTutorialStateProvider

// SetGuidedTutorialShovelProvider 设置铲子槽位边界提供者
// 由 GameScene 在初始化后调用此方法
func SetGuidedTutorialShovelProvider(provider ShovelSlotBoundsProvider) {
	guidedTutorialShovelProvider = provider
}

// SetGuidedTutorialStateProvider 设置强引导状态提供者
// 由 GameScene 在初始化后调用此方法
func SetGuidedTutorialStateProvider(provider GuidedTutorialStateProvider) {
	guidedTutorialStateProvider = provider
}

// IsGuidedTutorialBlocking 检查强引导模式是否阻止指定操作
// 供 InputSystem 等其他系统调用
func IsGuidedTutorialBlocking(operation string) bool {
	if guidedTutorialStateProvider == nil {
		return false // 没有提供者，不阻止操作
	}
	if !guidedTutorialStateProvider.IsGuidedTutorialActive() {
		return false // 强引导未激活，不阻止操作
	}
	return !guidedTutorialStateProvider.IsOperationAllowed(operation)
}

// NotifyGuidedTutorialOperation 通知强引导系统发生了某个操作
// 供 InputSystem 等其他系统调用
func NotifyGuidedTutorialOperation(operation string) {
	if guidedTutorialStateProvider != nil {
		guidedTutorialStateProvider.NotifyOperation(operation)
	}
}

// NewGuidedTutorialSystem 创建强引导教学系统
//
// 参数：
//   - em: 实体管理器
//   - gs: 游戏状态
//   - rm: 资源管理器
//
// 返回：
//   - 强引导教学系统实例
func NewGuidedTutorialSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *GuidedTutorialSystem {
	system := &GuidedTutorialSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		lastPlantCount:  0,
		initialized:     false,
		lastShovelMode:  false,
		totalTime:       0,
	}

	// 创建强引导教学实体
	system.guidedEntity = em.CreateEntity()

	// 初始化组件，默认白名单包含铲子教学阶段的操作
	guidedComp := &components.GuidedTutorialComponent{
		IsActive: false, // 默认不激活，由外部系统激活
		AllowedActions: []string{
			"click_shovel", // 点击铲子槽位
			"click_plant",  // 点击植物（铲子模式下）
			"click_screen", // 点击屏幕（推进 Dave 对话）
		},
		IdleTimer:            0,
		IdleThreshold:        config.GuidedTutorialIdleThreshold,
		ShowArrow:            false,
		ArrowTarget:          "shovel",
		ArrowEntityID:        0,
		LastPlantCount:       0,
		TransitionReady:      false,
		OnTransitionCallback: nil,
		TextEntityID:         0,
		TutorialTextKey:      "ADVICE_CLICK_SHOVEL", // Story 19.x QA: 铲子教学默认文本
	}
	em.AddComponent(system.guidedEntity, guidedComp)

	log.Printf("[GuidedTutorialSystem] Initialized (Entity ID: %d)", system.guidedEntity)

	return system
}

// Update 更新强引导教学系统状态
//
// 参数：
//   - dt: 时间增量（秒）
func (s *GuidedTutorialSystem) Update(dt float64) {
	s.totalTime += dt

	// 获取强引导组件
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return
	}

	// 如果未激活，跳过处理
	if !guidedComp.IsActive {
		return
	}

	// 初始化植物数量
	if !s.initialized {
		plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
		s.lastPlantCount = len(plantEntities)
		guidedComp.LastPlantCount = s.lastPlantCount
		s.initialized = true
		log.Printf("[GuidedTutorialSystem] Initialized with %d plants", s.lastPlantCount)
	}

	// 更新空闲计时器
	guidedComp.IdleTimer += dt

	// 监控植物数量变化
	s.monitorPlantCount(guidedComp)

	// 更新箭头显示状态
	s.updateArrowDisplay(guidedComp)
}

// monitorPlantCount 监控植物数量变化
// 检测植物被移除时重置空闲计时器，检测转场条件
func (s *GuidedTutorialSystem) monitorPlantCount(guidedComp *components.GuidedTutorialComponent) {
	// 获取当前植物数量
	plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
	currentPlantCount := len(plantEntities)

	// 检测数量变化（植物被移除）
	if currentPlantCount < s.lastPlantCount {
		// 有植物被移除，重置空闲计时器
		log.Printf("[GuidedTutorialSystem] Plant removed: %d -> %d, resetting idle timer",
			s.lastPlantCount, currentPlantCount)
		s.ResetIdleTimer()
	}

	// 更新计数
	s.lastPlantCount = currentPlantCount
	guidedComp.LastPlantCount = currentPlantCount

	// 检测转场条件（所有植物已移除）
	if currentPlantCount == 0 && !guidedComp.TransitionReady {
		guidedComp.TransitionReady = true
		log.Printf("[GuidedTutorialSystem] All plants removed, transition ready!")

		// 隐藏箭头和教学文本
		s.hideArrow(guidedComp)
		s.hideTutorialText(guidedComp)

		// 调用转场回调
		if guidedComp.OnTransitionCallback != nil {
			log.Printf("[GuidedTutorialSystem] Calling transition callback")
			guidedComp.OnTransitionCallback()
		}
	}
}

// updateArrowDisplay 更新箭头显示状态
// 根据空闲时间显示或隐藏浮动箭头
func (s *GuidedTutorialSystem) updateArrowDisplay(guidedComp *components.GuidedTutorialComponent) {
	// 如果转场已就绪，不显示箭头
	if guidedComp.TransitionReady {
		return
	}

	// 检查是否应该显示箭头
	shouldShowArrow := guidedComp.IdleTimer >= guidedComp.IdleThreshold

	if shouldShowArrow && !guidedComp.ShowArrow {
		// 空闲时间超过阈值，显示箭头
		s.showShovelArrow(guidedComp)
	} else if !shouldShowArrow && guidedComp.ShowArrow {
		// 有操作发生，隐藏箭头
		s.hideArrow(guidedComp)
	}
}

// showShovelArrow 显示指向铲子的浮动箭头
// Story 19.x QA: 箭头指向铲子本身而非卡槽
func (s *GuidedTutorialSystem) showShovelArrow(guidedComp *components.GuidedTutorialComponent) {
	// 如果箭头已存在，先隐藏
	if guidedComp.ArrowEntityID != 0 {
		s.hideArrow(guidedComp)
	}

	// 获取铲子图标位置
	if guidedTutorialShovelProvider == nil {
		log.Printf("[GuidedTutorialSystem] Warning: ShovelSlotBoundsProvider not set, cannot show arrow")
		return
	}

	// Story 19.x QA: 使用 GetShovelIconBounds 获取铲子图标的实际位置
	bounds := guidedTutorialShovelProvider.GetShovelIconBounds()

	// 计算箭头发射器位置
	// UpsellArrow.xml 中 EmitterOffsetX=25, EmitterOffsetY=80
	// 粒子实际出现位置 = 发射器位置 + (25, 80)
	// 目标：箭头指向铲子的水平居中、底部位置
	// 所以：发射器X = 铲子中心X - 25
	//       发射器Y = 铲子底部Y - 80
	shovelCenterX := float64(bounds.Min.X+bounds.Max.X) / 2
	shovelBottomY := float64(bounds.Max.Y)

	arrowX := shovelCenterX - 25 // 抵消 EmitterOffsetX
	arrowY := shovelBottomY - 80 // 抵消 EmitterOffsetY

	log.Printf("[GuidedTutorialSystem] Showing arrow: emitter(%.1f, %.1f), target shovel bottom center(%.1f, %.1f), icon bounds: %v",
		arrowX, arrowY, shovelCenterX, shovelBottomY, bounds)

	// 创建 UpsellArrow 粒子效果
	arrowEntity, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"UpsellArrow",
		arrowX, arrowY,
	)

	if err != nil {
		log.Printf("[GuidedTutorialSystem] Failed to create arrow indicator: %v", err)
		return
	}

	// 标记为 UI 粒子（不受摄像机影响）
	ecs.AddComponent(s.entityManager, arrowEntity, &components.UIComponent{
		State: components.UINormal,
	})

	// 设置粒子旋转（向上指向铲子）
	// UpsellArrow 默认是向下箭头，需要旋转 180 度
	if emitter, ok := ecs.GetComponent[*components.EmitterComponent](s.entityManager, arrowEntity); ok {
		emitter.ParticleRotationOverride = 180.0
		log.Printf("[GuidedTutorialSystem] Set particle rotation override to 180°")
	}

	// 保存箭头实体ID
	guidedComp.ArrowEntityID = arrowEntity
	guidedComp.ShowArrow = true
	guidedComp.ArrowTarget = "shovel"

	log.Printf("[GuidedTutorialSystem] Arrow shown at (%.1f, %.1f), entity ID: %d", arrowX, arrowY, arrowEntity)
}

// hideArrow 隐藏浮动箭头
func (s *GuidedTutorialSystem) hideArrow(guidedComp *components.GuidedTutorialComponent) {
	if guidedComp.ArrowEntityID == 0 {
		return
	}

	// 销毁发射器生成的粒子
	if emitter, ok := ecs.GetComponent[*components.EmitterComponent](s.entityManager, guidedComp.ArrowEntityID); ok {
		for _, particleID := range emitter.ActiveParticles {
			s.entityManager.DestroyEntity(particleID)
		}
		log.Printf("[GuidedTutorialSystem] Destroyed %d particles associated with arrow", len(emitter.ActiveParticles))
	}

	// 销毁箭头实体
	s.entityManager.DestroyEntity(guidedComp.ArrowEntityID)
	guidedComp.ArrowEntityID = 0
	guidedComp.ShowArrow = false

	log.Printf("[GuidedTutorialSystem] Arrow hidden")
}

// IsActive 返回强引导模式是否激活
func (s *GuidedTutorialSystem) IsActive() bool {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return false
	}
	return guidedComp.IsActive
}

// SetActive 设置强引导模式激活状态
// Story 19.x QA: 添加教学文本显示/隐藏
func (s *GuidedTutorialSystem) SetActive(active bool) {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return
	}

	wasActive := guidedComp.IsActive
	guidedComp.IsActive = active

	if active && !wasActive {
		// 激活时重置状态
		guidedComp.IdleTimer = 0
		guidedComp.ShowArrow = false
		guidedComp.TransitionReady = false
		s.initialized = false
		// Story 19.x QA: 显示教学文本
		s.showTutorialText(guidedComp)
		log.Printf("[GuidedTutorialSystem] Activated")
	} else if !active && wasActive {
		// 停用时隐藏箭头和教学文本
		s.hideArrow(guidedComp)
		s.hideTutorialText(guidedComp)
		log.Printf("[GuidedTutorialSystem] Deactivated")
	}
}

// showTutorialText 显示教学文本
// Story 19.x QA: 创建 TutorialTextComponent 显示铲子教学提示
func (s *GuidedTutorialSystem) showTutorialText(guidedComp *components.GuidedTutorialComponent) {
	// 如果已有教学文本实体，先隐藏
	if guidedComp.TextEntityID != 0 {
		s.hideTutorialText(guidedComp)
	}

	// 获取教学文本内容
	textKey := guidedComp.TutorialTextKey
	if textKey == "" {
		textKey = "ADVICE_CLICK_SHOVEL"
	}

	var text string
	if s.gameState.LawnStrings != nil {
		text = s.gameState.LawnStrings.GetString(textKey)
	}
	if text == "" {
		// 使用默认文本
		text = "点击拾取铲子！"
	}

	// 创建教学文本实体
	textEntity := s.entityManager.CreateEntity()
	textComp := &components.TutorialTextComponent{
		Text:            text,
		DisplayTime:     0,
		MaxDisplayTime:  0, // 无限显示，直到教学完成
		BackgroundAlpha: 0.5,
		IsAdvisory:      false, // 使用标准位置
	}
	s.entityManager.AddComponent(textEntity, textComp)

	// 保存实体ID
	guidedComp.TextEntityID = textEntity

	log.Printf("[GuidedTutorialSystem] Tutorial text shown: '%s' (Entity ID: %d)", text, textEntity)
}

// hideTutorialText 隐藏教学文本
// Story 19.x QA: 销毁教学文本实体
func (s *GuidedTutorialSystem) hideTutorialText(guidedComp *components.GuidedTutorialComponent) {
	if guidedComp.TextEntityID == 0 {
		return
	}

	s.entityManager.DestroyEntity(guidedComp.TextEntityID)
	guidedComp.TextEntityID = 0

	log.Printf("[GuidedTutorialSystem] Tutorial text hidden")
}

// IsOperationAllowed 检查操作是否在白名单中
//
// 参数：
//   - operation: 操作标识（如 "click_shovel", "click_plant"）
//
// 返回：
//   - true: 操作允许执行
//   - false: 操作被屏蔽（静默忽略）
func (s *GuidedTutorialSystem) IsOperationAllowed(operation string) bool {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return true // 组件不存在，允许所有操作
	}

	// 如果强引导未激活，允许所有操作
	if !guidedComp.IsActive {
		return true
	}

	// 检查操作是否在白名单中
	for _, allowed := range guidedComp.AllowedActions {
		if allowed == operation {
			return true
		}
	}

	return false
}

// NotifyOperation 通知系统发生了某个操作
// 用于重置空闲计时器
//
// 参数：
//   - operation: 操作标识
func (s *GuidedTutorialSystem) NotifyOperation(operation string) {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok || !guidedComp.IsActive {
		return
	}

	// 检查是否是有效操作
	if s.IsOperationAllowed(operation) {
		log.Printf("[GuidedTutorialSystem] Valid operation: %s, resetting idle timer", operation)
		s.ResetIdleTimer()
	}
}

// ResetIdleTimer 重置空闲计时器
func (s *GuidedTutorialSystem) ResetIdleTimer() {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return
	}

	guidedComp.IdleTimer = 0

	// 如果箭头正在显示，隐藏它
	if guidedComp.ShowArrow {
		s.hideArrow(guidedComp)
	}
}

// SetTransitionCallback 设置转场回调函数
// 当所有植物被移除时调用此回调
//
// 参数：
//   - callback: 回调函数
func (s *GuidedTutorialSystem) SetTransitionCallback(callback func()) {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return
	}
	guidedComp.OnTransitionCallback = callback
	log.Printf("[GuidedTutorialSystem] Transition callback set")
}

// SetAllowedActions 设置允许的操作白名单
// 用于根据当前阶段动态调整白名单
//
// 参数：
//   - actions: 允许的操作列表
func (s *GuidedTutorialSystem) SetAllowedActions(actions []string) {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return
	}
	guidedComp.AllowedActions = actions
	log.Printf("[GuidedTutorialSystem] Allowed actions set: %v", actions)
}

// IsTransitionReady 返回转场条件是否满足
func (s *GuidedTutorialSystem) IsTransitionReady() bool {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if !ok {
		return false
	}
	return guidedComp.TransitionReady
}

// GetPlantCount 获取当前植物数量
func (s *GuidedTutorialSystem) GetPlantCount() int {
	return s.lastPlantCount
}

// Cleanup 清理系统资源
func (s *GuidedTutorialSystem) Cleanup() {
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, s.guidedEntity)
	if ok {
		s.hideArrow(guidedComp)
	}
	guidedTutorialShovelProvider = nil
	log.Printf("[GuidedTutorialSystem] Cleanup completed")
}
