package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/types"
)

// ============================================================================
// 单个僵尸生成和激活
// ============================================================================

// spawnZombieForWave 为指定波次生成僵尸（预生成模式）
//
// 生成的僵尸初始状态为"待命"（不移动），需要调用 ActivateWave() 激活
//
// Story 17.3: 添加生成限制检查
// Story 17.9: 使用精确的出生坐标
//
// 参数：
//
//	zombieType - 僵尸类型
//	lane - 行号（1-5）
//	waveIndex - 所属波次索引（0-based）
//	indexInWave - 在本波中的索引（0, 1, 2...）
//
// 返回：
//
//	僵尸实体ID
func (s *WaveSpawnSystem) spawnZombieForWave(zombieType string, lane int, waveIndex int, indexInWave int) ecs.EntityID {
	// Story 17.3: 生成前检查限制规则
	if s.spawnRules != nil && s.constraintID != 0 {
		constraint, ok := ecs.GetComponent[*components.SpawnConstraintComponent](s.entityManager, s.constraintID)
		if ok {
			// 计算当前轮数
			// 公式: RoundNumber = TotalCompletedFlags / 2 - 1
			// 对于当前关卡，暂时假设 CompletedFlags = 0，即 RoundNumber = -1
			roundNumber := -1 // 默认第一轮
			if s.gameState.TotalCompletedFlags > 0 {
				roundNumber = s.gameState.TotalCompletedFlags/2 - 1
			}

			// 转换 lane (1-5) 为 row (0-4)
			row := lane - 1
			if row < 0 {
				row = 0
			}
			if row > 4 {
				row = 4
			}

			// 使用纯函数验证
			valid, reason := ValidateZombieSpawn(zombieType, lane, constraint, roundNumber, s.spawnRules)
			if !valid {
				log.Printf("[WaveSpawnSystem] Zombie spawn rejected: type=%s, lane=%d, wave=%d, reason=%s",
					zombieType, lane, waveIndex+1, reason)
				return 0 // 跳过生成
			}

			// 如果是红眼巨人，增加计数
			if zombieType == "gargantuar_redeye" {
				constraint.RedEyeCount++
				log.Printf("[WaveSpawnSystem] Red eye count increased: %d (round %d)",
					constraint.RedEyeCount, roundNumber)
			}
		}
	}

	// Story 17.9: 判断是否为旗帜波
	isFlagWave := false
	if s.levelConfig != nil && waveIndex >= 0 && waveIndex < len(s.levelConfig.Waves) {
		isFlagWave = s.levelConfig.Waves[waveIndex].IsFlag
	}

	// 转换 lane (1-5) 为 row (0-4) 用于分散分配
	targetRow := lane - 1
	if targetRow < 0 {
		targetRow = 0
	}
	if targetRow > 4 {
		targetRow = 4
	}

	// 使用分散分配器计算激活延迟和X偏移
	// 确保同行僵尸有时间间隔，不同行有空间错开
	activationDelay, xOffset := s.dispersionAllocator.AllocateZombie(targetRow)

	// 开场预览期间，僵尸随机分布在5行上
	// 激活后，僵尸会移动到随机选择的有效行

	// 预览行：随机选择一行（0-4）用于开场预览展示
	previewRow := rand.Intn(5)

	// Story 17.9: 计算预览位置（僵尸初始站位）
	// X坐标：根据波次类型和僵尸类型使用精确坐标 + 分散偏移
	baseSpawnX := s.getZombieSpawnXForWave(zombieType, isFlagWave, false)
	spawnX := baseSpawnX + xOffset // 应用跨行空间错开
	spawnY := s.getZombieSpawnY(previewRow) // 使用随机行的Y坐标

	// 使用统一的工厂注册表创建僵尸
	// 新增僵尸时只需在 entities/zombie_factory_registry.go 中添加记录
	entityID, err := entities.CreateZombieByID(
		zombieType,
		s.entityManager,
		s.resourceManager,
		previewRow,
		spawnX,
	)

	if err != nil {
		log.Printf("[WaveSpawnSystem] ERROR: Failed to spawn zombie: %v", err)
		return 0
	}

	// 添加波次状态组件（标记为待命状态）
	// 使用分散分配器计算的激活延迟，实现同行僵尸时间错开效果
	ecs.AddComponent(s.entityManager, entityID, &components.ZombieWaveStateComponent{
		WaveIndex:           waveIndex,
		IsActivated:         false, // 初始状态：待命
		IndexInWave:         indexInWave,
		ActivationDelay:     activationDelay,
		ActivationTimer:     0,
		IsPendingActivation: false,
	})

	// 移除或清零速度组件（僵尸初始不移动）
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
		vel.VX = 0 // 待命状态：不向左移动
		vel.VY = 0 // 待命状态：不垂直移动
		log.Printf("[WaveSpawnSystem] Cleared zombie %d velocity: VX=%.2f, VY=%.2f", entityID, vel.VX, vel.VY)
	}

	// 切换到 idle 动画（预览期间僵尸静止）
	// 激活时会切换回 walk 动画
	if behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
		behavior.ZombieAnimState = components.ZombieAnimIdle

		// 读取当前动画状态（调试用）
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			// 使用新的简化结构
			currentFrame := reanimComp.CurrentFrame
			currentAnim := ""
			if len(reanimComp.CurrentAnimations) > 0 {
				currentAnim = reanimComp.CurrentAnimations[0]
			}
			log.Printf("[WaveSpawnSystem] Zombie %d 切换前动画: %s, 帧: %d", entityID, currentAnim, currentFrame)
		}

		// Determine correct unit ID based on behavior type
		unitID := types.UnitIDZombie
		switch behavior.Type {
		case components.BehaviorZombieConehead:
			unitID = types.UnitIDZombieConehead
		case components.BehaviorZombieBuckethead:
			unitID = types.UnitIDZombieBuckethead
		}

		// 使用组件通信替代直接调用
		// 僵尸使用配置驱动的动画组合（自动隐藏装备轨道）
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    unitID,
			ComboName: "idle",
			Processed: false,
		})

		// 验证切换后的状态
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			// 使用新的简化结构
			currentFrame := reanimComp.CurrentFrame
			currentAnim := ""
			if len(reanimComp.CurrentAnimations) > 0 {
				currentAnim = reanimComp.CurrentAnimations[0]
			}
			log.Printf("[WaveSpawnSystem] Zombie %d 添加空闲动画命令: %s, 帧: %d (预生成阶段使用 idle)", entityID, currentAnim, currentFrame)
		}
	}

	// 注意：不在创建时添加目标行组件
	// 目标行将在激活时（ActivateWave）才随机选择并添加

	log.Printf("[WaveSpawnSystem] Pre-spawned zombie %d: wave=%d, index=%d, previewRow=%d, pos=(%.1f, %.1f)",
		entityID, waveIndex, indexInWave, previewRow, spawnX, spawnY)

	return entityID
}

// spawnAndActivateZombie 生成并直接激活单个僵尸（实时生成模式）
//
// 生成僵尸后立即设置为激活状态，开始移动和播放行走动画
//
// 参数：
//
//	zombieType - 僵尸类型
//	lane - 行号（1-5）
//	waveIndex - 所属波次索引（0-based）
//	indexInWave - 在本波中的索引
//
// 返回：
//
//	僵尸实体ID
func (s *WaveSpawnSystem) spawnAndActivateZombie(zombieType string, lane int, waveIndex int, indexInWave int) ecs.EntityID {
	// Story 17.3: 生成前检查限制规则
	if s.spawnRules != nil && s.constraintID != 0 {
		constraint, ok := ecs.GetComponent[*components.SpawnConstraintComponent](s.entityManager, s.constraintID)
		if ok {
			roundNumber := -1
			if s.gameState.TotalCompletedFlags > 0 {
				roundNumber = s.gameState.TotalCompletedFlags/2 - 1
			}

			valid, reason := ValidateZombieSpawn(zombieType, lane, constraint, roundNumber, s.spawnRules)
			if !valid {
				log.Printf("[WaveSpawnSystem] Zombie spawn rejected: type=%s, lane=%d, wave=%d, reason=%s",
					zombieType, lane, waveIndex+1, reason)
				return 0
			}

			if zombieType == "gargantuar_redeye" {
				constraint.RedEyeCount++
			}
		}
	}

	// 将行号转换为行索引
	row := lane - 1
	if row < 0 {
		row = 0
	}
	if row > 4 {
		row = 4
	}

	// 使用分散分配器计算X偏移（实时模式不使用延迟，但仍需空间错开）
	_, xOffset := s.dispersionAllocator.AllocateZombie(row)

	// Story 17.9: 判断是否为旗帜波
	isFlagWave := false
	if s.levelConfig != nil && waveIndex >= 0 && waveIndex < len(s.levelConfig.Waves) {
		isFlagWave = s.levelConfig.Waves[waveIndex].IsFlag
	}

	// Story 17.9: 计算生成位置 + 分散偏移
	baseSpawnX := s.getZombieSpawnXForWave(zombieType, isFlagWave, false)
	spawnX := baseSpawnX + xOffset // 应用跨行空间错开
	spawnY := s.getZombieSpawnY(row)

	// 使用统一的工厂注册表创建僵尸
	// 新增僵尸时只需在 entities/zombie_factory_registry.go 中添加记录
	entityID, err := entities.CreateZombieByID(
		zombieType,
		s.entityManager,
		s.resourceManager,
		row,
		spawnX,
	)

	if err != nil {
		log.Printf("[WaveSpawnSystem] ERROR: Failed to spawn zombie: %v", err)
		return 0
	}

	// 添加波次状态组件（直接标记为已激活）
	ecs.AddComponent(s.entityManager, entityID, &components.ZombieWaveStateComponent{
		WaveIndex:   waveIndex,
		IsActivated: true, // 直接激活
		IndexInWave: indexInWave,
	})

	// 使用公共函数激活僵尸（复用正式逻辑）
	// Story 8.9: 撑杆僵尸使用专用激活函数
	if zombieType == "polevaulter" {
		entities.ActivatePolevaulterZombie(s.entityManager, entityID)
	} else {
		entities.ActivateZombie(s.entityManager, entityID)
	}

	log.Printf("[WaveSpawnSystem] Spawned and activated zombie %d: type=%s, wave=%d, index=%d, row=%d, pos=(%.1f, %.1f)",
		entityID, zombieType, waveIndex+1, indexInWave, row, spawnX, spawnY)

	return entityID
}

// spawnZombieWithOffset 生成单个僵尸（带索引偏移避免重叠）
//
// 根据僵尸类型字符串调用对应的工厂函数
//
// 参数：
//
//	zombieType - 僵尸类型字符串："basic", "conehead", "buckethead"
//	lane - 行号（1-5，对应游戏界面的5行）
//	index - 僵尸在同一波中的索引（0, 1, 2...），用于计算额外的X偏移
//
// 返回：
//
//	生成的僵尸实体ID，如果失败返回 0
func (s *WaveSpawnSystem) spawnZombieWithOffset(zombieType string, lane int, index int) ecs.EntityID {
	// 将行号从1-5转换为数组索引0-4
	row := lane - 1
	if row < 0 || row > 4 {
		log.Printf("[WaveSpawnSystem] ERROR: Invalid lane %d (must be 1-5)", lane)
		return 0
	}

	// 计算生成位置
	// X坐标：在配置的范围内随机生成，根据行号的最大X值
	spawnX := s.getZombieSpawnX(row)
	spawnY := s.getZombieSpawnY(row)

	// 查找目标有效行（如果当前行无效）
	targetLane := s.findNearestEnabledLane(lane)
	targetRow := targetLane - 1

	// 使用统一的工厂注册表创建僵尸
	// 新增僵尸时只需在 entities/zombie_factory_registry.go 中添加记录
	entityID, err := entities.CreateZombieByID(
		zombieType,
		s.entityManager,
		s.resourceManager,
		row,
		spawnX,
	)

	// 检查是否创建成功
	if err != nil {
		log.Printf("[WaveSpawnSystem] ERROR: Failed to spawn zombie type '%s': %v", zombieType, err)
		return 0
	}

	// 如果生成行不是有效行，添加目标行组件
	if row != targetRow {
		s.addTargetLaneComponent(entityID, targetRow, spawnY)
		log.Printf("[WaveSpawnSystem] Zombie %d spawned at lane %d (disabled), will move to lane %d before attacking",
			entityID, lane, targetLane)
	}

	// 输出生成位置信息（用于调试重叠问题）
	log.Printf("[WaveSpawnSystem] Zombie %d spawned at position: X=%.1f (base+index*120+random), Y=%.1f, lane=%d, index=%d",
		entityID, spawnX, spawnY, lane, index)

	return entityID
}

// activateZombieNow 立即激活单个僵尸
//
// 执行僵尸的实际激活逻辑：设置速度、切换动画、分配目标行
func (s *WaveSpawnSystem) activateZombieNow(entityID ecs.EntityID, waveState *components.ZombieWaveStateComponent) {
	// 标记为已激活
	waveState.IsActivated = true
	waveState.IsPendingActivation = false

	// 获取僵尸当前位置
	pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if hasPos {
		// 计算当前所在行（0-4）
		currentRow := int((pos.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)
		if currentRow < 0 {
			currentRow = 0
		}
		if currentRow > 4 {
			currentRow = 4
		}

		// 从 enabledLanes 中随机选择一个有效行作为目标行
		targetRow := s.randomEnabledLane()

		log.Printf("[WaveSpawnSystem] Zombie %d activating: currentRow=%d, targetRow=%d", entityID, currentRow, targetRow)

		// 如果当前行不是目标行，添加目标行组件
		if currentRow != targetRow {
			s.addTargetLaneComponent(entityID, targetRow, pos.Y)
			log.Printf("[WaveSpawnSystem] Zombie %d will transition from row %d to row %d", entityID, currentRow, targetRow)
		}
	}

	// 使用公共函数激活僵尸（复用正式逻辑：设置速度和动画）
	entities.ActivateZombie(s.entityManager, entityID)

	log.Printf("[WaveSpawnSystem] Activated zombie %d (wave %d, index %d)",
		entityID, waveState.WaveIndex, waveState.IndexInWave)

	// 增加已激活僵尸计数（用于计算场上僵尸数）
	// zombiesOnField = TotalZombiesSpawned - ZombiesKilled
	s.gameState.IncrementZombiesSpawned(1)
}

// UpdatePendingActivations 更新等待激活的僵尸
//
// 每帧调用，检查所有等待激活的僵尸，当延迟时间到达时激活它们
// 这样可以实现同一波次僵尸错开入场的散落效果
//
// 参数：
//
//	dt - 距离上一帧的时间间隔（秒）
func (s *WaveSpawnSystem) UpdatePendingActivations(dt float64) {
	// 查询所有带 ZombieWaveStateComponent 的僵尸
	zombieEntities := ecs.GetEntitiesWith1[*components.ZombieWaveStateComponent](s.entityManager)

	for _, entityID := range zombieEntities {
		waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只处理等待激活中的僵尸
		if waveState.IsPendingActivation && !waveState.IsActivated {
			// 更新计时器
			waveState.ActivationTimer += dt

			// 检查是否达到激活延迟
			if waveState.ActivationTimer >= waveState.ActivationDelay {
				// 执行实际激活
				s.activateZombieNow(entityID, waveState)
				log.Printf("[WaveSpawnSystem] Zombie %d activated after %.2fs delay", entityID, waveState.ActivationDelay)
			}
		}
	}
}
