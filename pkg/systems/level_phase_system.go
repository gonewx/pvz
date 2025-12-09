package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
)

// LevelPhaseSystem 关卡阶段转场系统
// Story 19.4: 管理关卡多阶段流程和转场动画
//
// 此系统负责：
// - 监控阶段状态变化
// - 处理转场动画（传送带滑入、红线显示等）
// - 协调转场序列（关闭强引导 → Dave 对话 → 动画 → 激活下一阶段）
//
// 遵循零耦合原则：
// - 通过回调函数与外部系统通信
// - 使用组件查询而非直接调用其他系统
type LevelPhaseSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// phaseEntity 阶段管理实体ID
	phaseEntity ecs.EntityID

	// 外部系统回调（遵循零耦合原则）
	onDisableGuidedTutorial func()                  // 关闭强引导模式回调
	onActivateBowling       func()                  // 激活保龄球阶段回调
	onTransitionComplete    func()                  // 转场完成回调
	daveDialogueKeys        []string                // Dave 对话文本 keys
	resourceLoader          entities.ResourceLoader // 资源加载器接口
}

// NewLevelPhaseSystem 创建关卡阶段转场系统
//
// 参数：
//   - em: 实体管理器
//   - gs: 游戏状态
//   - rm: 资源管理器
//
// 返回：
//   - 关卡阶段转场系统实例
func NewLevelPhaseSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *LevelPhaseSystem {
	system := &LevelPhaseSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		daveDialogueKeys: []string{
			"CRAZY_DAVE_2410", // "好的，干得不错，现在给你个惊喜……"
			"CRAZY_DAVE_2411", // "我们去玩保龄球！"
			"CRAZY_DAVE_2412", // "嘿，拿好这个坚果墙！" {SHOW_WALLNUT}
			"CRAZY_DAVE_2413", // "为什么我要给你坚果墙？"
			"CRAZY_DAVE_2414", // "因为我发~~~疯了！！！！！" {SHAKE}
			"CRAZY_DAVE_2415", // "现在出发！给我赢个冠军回来！" {SCREAM}
		},
	}

	// 只有当 rm 非 nil 时才设置 resourceLoader
	// 避免接口变量持有 nil 指针的问题
	if rm != nil {
		system.resourceLoader = rm
	}

	// 创建阶段管理实体
	system.phaseEntity = em.CreateEntity()

	// 初始化阶段组件
	phaseComp := &components.LevelPhaseComponent{
		CurrentPhase:        1, // 初始为阶段 1（铲子教学）
		PhaseState:          components.PhaseStateActive,
		TransitionProgress:  0,
		TransitionStep:      components.TransitionStepNone,
		ConveyorBeltY:       config.ConveyorBeltStartY,
		ConveyorBeltVisible: false,
		ShowRedLine:         false,
	}
	em.AddComponent(system.phaseEntity, phaseComp)

	log.Printf("[LevelPhaseSystem] Initialized (Entity ID: %d), starting at Phase %d",
		system.phaseEntity, phaseComp.CurrentPhase)

	return system
}

// Update 更新系统状态
//
// 参数：
//   - dt: 时间增量（秒）
func (s *LevelPhaseSystem) Update(dt float64) {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return
	}

	// 只在转场状态下更新
	if phaseComp.PhaseState != components.PhaseStateTransitioning {
		return
	}

	// 根据当前转场步骤处理
	switch phaseComp.TransitionStep {
	case components.TransitionStepDisableGuided:
		// 步骤 1: 关闭强引导模式
		s.executeDisableGuided(phaseComp)

	case components.TransitionStepDaveDialogue:
		// 步骤 2: 等待 Dave 对话完成（由回调推进）
		// 检查 Dave 实体是否还存在
		s.checkDaveDialogueComplete(phaseComp)

	case components.TransitionStepConveyorSlide:
		// 步骤 3: 传送带滑入动画
		s.updateConveyorSlideIn(dt, phaseComp)

	case components.TransitionStepShowRedLine:
		// 步骤 4: 显示红线
		s.executeShowRedLine(phaseComp)

	case components.TransitionStepActivateBowling:
		// 步骤 5: 激活保龄球阶段
		s.executeActivateBowling(phaseComp)
	}
}

// StartPhaseTransition 启动阶段转场
//
// 参数：
//   - from: 源阶段编号
//   - to: 目标阶段编号
func (s *LevelPhaseSystem) StartPhaseTransition(from, to int) {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		log.Printf("[LevelPhaseSystem] ERROR: Cannot start transition, phase component not found")
		return
	}

	// Bug Fix: 如果已经在转场中，忽略重复的转场请求
	// 这可能发生在从存档恢复后，GuidedTutorialSystem 再次触发回调的情况
	if phaseComp.PhaseState == components.PhaseStateTransitioning {
		log.Printf("[LevelPhaseSystem] Already transitioning, ignoring duplicate transition request")
		return
	}

	// 验证转场有效性
	if phaseComp.CurrentPhase != from {
		log.Printf("[LevelPhaseSystem] WARNING: Current phase is %d, not %d, ignoring transition request",
			phaseComp.CurrentPhase, from)
		return
	}

	log.Printf("[LevelPhaseSystem] Starting phase transition: %d -> %d", from, to)

	// 设置转场状态
	phaseComp.PhaseState = components.PhaseStateTransitioning
	phaseComp.TransitionProgress = 0
	phaseComp.TransitionStep = components.TransitionStepDisableGuided

	log.Printf("[LevelPhaseSystem] Transition started, first step: DisableGuided")
}

// executeDisableGuided 执行步骤1：关闭强引导模式
func (s *LevelPhaseSystem) executeDisableGuided(phaseComp *components.LevelPhaseComponent) {
	log.Printf("[LevelPhaseSystem] Step 1: Disabling guided tutorial mode")

	// 调用回调关闭强引导
	if s.onDisableGuidedTutorial != nil {
		s.onDisableGuidedTutorial()
	}

	// 推进到下一步：创建 Dave 对话
	phaseComp.TransitionStep = components.TransitionStepDaveDialogue
	s.createTransitionDaveDialogue(phaseComp)
}

// createTransitionDaveDialogue 创建转场 Dave 对话
func (s *LevelPhaseSystem) createTransitionDaveDialogue(phaseComp *components.LevelPhaseComponent) {
	log.Printf("[LevelPhaseSystem] Step 2: Creating Dave dialogue for transition")

	// Bug Fix: 检查是否已存在 Dave 实体（从存档恢复的情况）
	// 如果已存在，使用现有实体而不是创建新的
	existingDaves := ecs.GetEntitiesWith1[*components.DaveDialogueComponent](s.entityManager)
	if len(existingDaves) > 0 {
		phaseComp.DaveDialogueEntityID = int(existingDaves[0])
		log.Printf("[LevelPhaseSystem] Using existing Dave entity from save restore: %d", existingDaves[0])
		return
	}

	// 检查资源加载器是否可用（测试环境下可能为 nil）
	if s.resourceLoader == nil {
		log.Printf("[LevelPhaseSystem] WARNING: ResourceLoader is nil, skipping Dave dialogue")
		s.advanceToConveyorSlide()
		return
	}

	// 创建 Dave 实体
	daveEntity, err := entities.NewCrazyDaveEntity(
		s.entityManager,
		s.resourceLoader,
		s.daveDialogueKeys,
		func() {
			// Dave 对话完成回调
			log.Printf("[LevelPhaseSystem] Dave dialogue completed, advancing to conveyor slide")
			s.advanceToConveyorSlide()
		},
	)

	if err != nil {
		log.Printf("[LevelPhaseSystem] ERROR: Failed to create Dave entity: %v", err)
		// 跳过 Dave 对话，直接进入下一步
		s.advanceToConveyorSlide()
		return
	}

	phaseComp.DaveDialogueEntityID = int(daveEntity)
	log.Printf("[LevelPhaseSystem] Dave entity created: %d", daveEntity)
}

// checkDaveDialogueComplete 检查 Dave 对话是否完成
// Story 19.x QA: 监听对话进度，当 Dave 说第二句话时显示红线
func (s *LevelPhaseSystem) checkDaveDialogueComplete(phaseComp *components.LevelPhaseComponent) {
	// 获取 Dave 对话实体
	if phaseComp.DaveDialogueEntityID == 0 {
		return
	}

	daveEntityID := ecs.EntityID(phaseComp.DaveDialogueEntityID)
	dialogueComp, ok := ecs.GetComponent[*components.DaveDialogueComponent](s.entityManager, daveEntityID)
	if !ok {
		return
	}

	// Story 19.x QA: 当 Dave 说第二句话（CRAZY_DAVE_2411 "我们去玩保龄球！"）时显示红线
	// CurrentLineIndex == 1 表示正在显示第二句（索引从 0 开始）
	if dialogueComp.CurrentLineIndex >= 1 && !phaseComp.ShowRedLine {
		log.Printf("[LevelPhaseSystem] Dave is saying line %d, showing red line", dialogueComp.CurrentLineIndex+1)
		phaseComp.ShowRedLine = true
	}
}

// advanceToConveyorSlide 推进到传送带滑入步骤
func (s *LevelPhaseSystem) advanceToConveyorSlide() {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return
	}

	log.Printf("[LevelPhaseSystem] Step 3: Starting conveyor belt slide-in animation")
	phaseComp.TransitionStep = components.TransitionStepConveyorSlide
	phaseComp.ConveyorBeltVisible = true
	phaseComp.ConveyorBeltY = config.ConveyorBeltStartY
	phaseComp.TransitionProgress = 0
}

// updateConveyorSlideIn 更新传送带滑入动画
func (s *LevelPhaseSystem) updateConveyorSlideIn(dt float64, phaseComp *components.LevelPhaseComponent) {
	// 计算动画进度
	duration := config.PhaseTransitionConveyorSlideDuration
	phaseComp.TransitionProgress += dt / duration

	if phaseComp.TransitionProgress >= 1.0 {
		phaseComp.TransitionProgress = 1.0
	}

	// 使用 ease-out 缓动计算 Y 位置
	progress := easeOutQuad(phaseComp.TransitionProgress)
	startY := config.ConveyorBeltStartY
	targetY := config.ConveyorBeltTargetY
	phaseComp.ConveyorBeltY = startY + (targetY-startY)*progress

	// 检查动画是否完成
	if phaseComp.TransitionProgress >= 1.0 {
		log.Printf("[LevelPhaseSystem] Conveyor slide-in complete, Y=%.1f", phaseComp.ConveyorBeltY)
		phaseComp.TransitionStep = components.TransitionStepShowRedLine
	}
}

// executeShowRedLine 执行步骤4：显示红线
// Story 19.x QA: 红线可能已在 Dave 对话期间显示，这里做兼容检查
func (s *LevelPhaseSystem) executeShowRedLine(phaseComp *components.LevelPhaseComponent) {
	if !phaseComp.ShowRedLine {
		log.Printf("[LevelPhaseSystem] Step 4: Showing red line")
		phaseComp.ShowRedLine = true
	} else {
		log.Printf("[LevelPhaseSystem] Step 4: Red line already shown during Dave dialogue")
	}
	phaseComp.TransitionStep = components.TransitionStepActivateBowling
}

// executeActivateBowling 执行步骤5：激活保龄球阶段
func (s *LevelPhaseSystem) executeActivateBowling(phaseComp *components.LevelPhaseComponent) {
	log.Printf("[LevelPhaseSystem] Step 5: Activating bowling phase")

	// 更新阶段状态
	phaseComp.CurrentPhase = 2
	phaseComp.PhaseState = components.PhaseStateActive
	phaseComp.TransitionStep = components.TransitionStepNone

	// 调用回调激活保龄球阶段
	if s.onActivateBowling != nil {
		s.onActivateBowling()
	}

	// 调用转场完成回调
	if s.onTransitionComplete != nil {
		s.onTransitionComplete()
	}

	// 调用���件内的回调
	if phaseComp.OnPhaseTransitionComplete != nil {
		phaseComp.OnPhaseTransitionComplete()
	}

	log.Printf("[LevelPhaseSystem] Phase transition completed! Now in Phase %d", phaseComp.CurrentPhase)
}

// ============================================================================
// 公共查询方法
// ============================================================================

// IsInPhase 检查当前是否在指定阶段
func (s *LevelPhaseSystem) IsInPhase(phase int) bool {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return false
	}
	return phaseComp.CurrentPhase == phase
}

// IsTransitioning 检查是否正在转场
func (s *LevelPhaseSystem) IsTransitioning() bool {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return false
	}
	return phaseComp.PhaseState == components.PhaseStateTransitioning
}

// GetCurrentPhase 获取当前阶段编号
func (s *LevelPhaseSystem) GetCurrentPhase() int {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return 0
	}
	return phaseComp.CurrentPhase
}

// GetConveyorBeltY 获取传送带当前 Y 位置
func (s *LevelPhaseSystem) GetConveyorBeltY() float64 {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return config.ConveyorBeltTargetY
	}
	return phaseComp.ConveyorBeltY
}

// IsConveyorBeltVisible 检查传送带是否可见
func (s *LevelPhaseSystem) IsConveyorBeltVisible() bool {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return false
	}
	return phaseComp.ConveyorBeltVisible
}

// ShouldShowRedLine 检查是否应该显示红线
func (s *LevelPhaseSystem) ShouldShowRedLine() bool {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return false
	}
	return phaseComp.ShowRedLine
}

// ============================================================================
// 回调设置方法
// ============================================================================

// SetOnDisableGuidedTutorial 设置关闭强引导模式回调
func (s *LevelPhaseSystem) SetOnDisableGuidedTutorial(callback func()) {
	s.onDisableGuidedTutorial = callback
}

// SetOnActivateBowling 设置激活保龄球阶段回调
func (s *LevelPhaseSystem) SetOnActivateBowling(callback func()) {
	s.onActivateBowling = callback
}

// SetOnTransitionComplete 设置转场完成回调
func (s *LevelPhaseSystem) SetOnTransitionComplete(callback func()) {
	s.onTransitionComplete = callback
}

// GetPhaseEntity 获取阶段管理实体ID
func (s *LevelPhaseSystem) GetPhaseEntity() ecs.EntityID {
	return s.phaseEntity
}

// SetDaveDialogueEntityID 设置转场 Dave 对话实体ID
//
// Story 19.x Bug Fix: 从存档恢复时，需要更新 Dave 对话实体ID
// 因为恢复时创建的 Dave 实体是新的，与存档中保存的实体ID不同
func (s *LevelPhaseSystem) SetDaveDialogueEntityID(entityID ecs.EntityID) {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return
	}
	phaseComp.DaveDialogueEntityID = int(entityID)
	log.Printf("[LevelPhaseSystem] Updated DaveDialogueEntityID to %d", entityID)
}

// AdvanceToConveyorSlide 公开方法：推进到传送带滑入步骤
//
// Story 19.x Bug Fix: 从存档恢复 Dave 对话完成时调用
// 用于替代私有方法 advanceToConveyorSlide()，允许外部调用
func (s *LevelPhaseSystem) AdvanceToConveyorSlide() {
	s.advanceToConveyorSlide()
}

// IsInDaveDialogueStep 检查是否正在转场的 Dave 对话步骤
//
// Story 19.x Bug Fix: 用于存档恢复时判断是否需要设置特殊回调
func (s *LevelPhaseSystem) IsInDaveDialogueStep() bool {
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, s.phaseEntity)
	if !ok {
		return false
	}
	return phaseComp.PhaseState == components.PhaseStateTransitioning &&
		phaseComp.TransitionStep == components.TransitionStepDaveDialogue
}
