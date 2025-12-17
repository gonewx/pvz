package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
)

// ========================================
// 胜利/失败条件检测
// ========================================

// checkVictoryCondition 检查胜利条件
//
// 胜利条件：
// 1. 所有波次已生成 且 所有僵尸已消灭（GameState.CheckVictory()）
// 2. 没有活跃的（移动中的）除草车（Story 10.2）- 车完全消失后再显示胜利动画
//
// 如果达成胜利条件，设置游戏结果为 "win"
func (s *LevelSystem) checkVictoryCondition() {
	// 检查是否有活跃的除草车
	// 原版行为：除草车完全消失后，才显示胜利动画
	hasActiveLawnmowers := false
	if s.lawnmowerSystem != nil {
		hasActiveLawnmowers = s.lawnmowerSystem.HasActiveLawnmowers()
	}

	// Story 18.3 调试：计算场上僵尸数量
	zombiesOnField := s.gameState.TotalZombiesSpawned - s.gameState.ZombiesKilled
	if zombiesOnField <= 0 && s.gameState.TotalZombiesSpawned > 0 {
		// 场上没有僵尸了，输出详细状态
		log.Printf("[LevelSystem] checkVictoryCondition: ZombiesOnField=%d, Killed=%d/%d, SpawnedWaves=%v, hasActiveLawnmowers=%v",
			zombiesOnField, s.gameState.ZombiesKilled, s.gameState.TotalZombiesInLevel,
			s.gameState.SpawnedWaves, hasActiveLawnmowers)
	}

	// 只有在没有活跃除草车的情况下才能胜利
	if s.gameState.CheckVictory() && !hasActiveLawnmowers {
		s.gameState.SetGameResult("win")
		log.Println("[LevelSystem] Victory! All zombies defeated!")

		// 淡出 BGM（胜利音乐在点击卡包后播放，见 reward_animation_system.go）
		if s.resourceManager != nil {
			s.resourceManager.FadeOutMusic(0.3)
		}

		// 清理场上所有阳光实体（避免在奖励动画阶段继续显示）
		s.cleanupAllSunEntities()

		// 保存关卡进度
		if s.gameState.CurrentLevel != nil {
			levelID := s.gameState.CurrentLevel.ID
			rewardPlant := s.gameState.CurrentLevel.RewardPlant
			unlockTools := s.gameState.CurrentLevel.UnlockTools

			// 保存进度（包括关卡完成、植物解锁、工具解锁）
			if err := s.gameState.CompleteLevel(levelID, rewardPlant, unlockTools); err != nil {
				log.Printf("[LevelSystem] Warning: Failed to save progress: %v", err)
			} else {
				log.Printf("[LevelSystem] Progress saved: level=%s, plant=%s, tools=%v", levelID, rewardPlant, unlockTools)
			}
		}

		// 检查是否有新植物解锁，触发奖励动画
		s.triggerRewardIfNeeded()
	}
}

// triggerRewardIfNeeded 检查是否有新植物或工具解锁，如果有则触发奖励动画
func (s *LevelSystem) triggerRewardIfNeeded() {
	if s.rewardSystem == nil {
		return
	}

	hasReward := false

	// 优先检查工具奖励（工具比植物更稀有，优先显示）
	if s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.UnlockTools) > 0 {
		toolID := s.gameState.CurrentLevel.UnlockTools[0] // 通常只解锁一个工具
		log.Printf("[LevelSystem] Triggering reward animation for tool: %s", toolID)
		s.rewardSystem.TriggerToolReward(toolID)
		hasReward = true
	}

	// 如果没有工具奖励，检查植物奖励
	if !hasReward {
		lastUnlocked := s.gameState.GetPlantUnlockManager().GetLastUnlocked()
		if lastUnlocked != "" {
			log.Printf("[LevelSystem] Triggering reward animation for plant: %s", lastUnlocked)
			s.rewardSystem.TriggerPlantReward(lastUnlocked)
			hasReward = true
			// 清除最后解锁标记（避免重复触发）
			s.gameState.GetPlantUnlockManager().ClearLastUnlocked()
		}
	}

	if !hasReward {
		log.Println("[LevelSystem] No new reward unlocked, skipping reward animation")
	}
}

// checkDefeatCondition 检查失败条件
//
// 失败条件（Story 10.2 增强）：
// 1. 如果启用了除草车系统：僵尸到达左侧边界 && 该行除草车已使用 → 游戏失败
// 2. 如果未启用除草车系统：僵尸到达左侧边界 → 游戏失败（原逻辑）
//
// 这样设计的原因：
// - 除草车是每行的最后防线，只在僵尸到达左侧时触发一次
// - 如果除草车未使用，僵尸到达左侧会触发除草车（不触发失败）
// - 如果除草车已使用，僵尸再次到达左侧直接失败（无最后防线）
func (s *LevelSystem) checkDefeatCondition() {
	// 如果启用了除草车系统，检查除草车状态
	if s.lawnmowerSystem != nil {
		s.checkDefeatWithLawnmower()
		return
	}

	// 原逻辑：未启用除草车系统，僵尸到达左侧直接失败
	s.checkDefeatWithoutLawnmower()
}

// checkDefeatWithLawnmower 检查失败条件（有除草车）
// Story 17.9: 使用类型化进家边界
func (s *LevelSystem) checkDefeatWithLawnmower() {
	// 获取除草车状态组件
	stateEntityID := s.lawnmowerSystem.GetStateEntityID()
	state, ok := ecs.GetComponent[*components.LawnmowerStateComponent](s.entityManager, stateEntityID)
	if !ok {
		log.Printf("[LevelSystem] Warning: LawnmowerStateComponent not found, falling back to original defeat logic")
		s.checkDefeatWithoutLawnmower()
		return
	}

	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 检查是否有僵尸到达左边界
	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只检查僵尸类型的实体
		if !isZombieType(behavior.Type) {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// Story 17.9: 获取类型化的进家边界
		zombieTypeStr := s.behaviorTypeToString(behavior.Type)
		defeatBoundary := s.getDefeatBoundary(zombieTypeStr)

		// 僵尸到达左边界
		if pos.X < defeatBoundary {
			// 计算僵尸所在行
			lane := s.getEntityLane(pos.Y)

			// 检查该行除草车是否已使用
			if state.UsedLanes[lane] {
				// 除草车已使用，游戏失败
				s.gameState.SetGameResult("lose")
				log.Printf("[LevelSystem] Defeat! Zombie (ID:%d, type:%s) reached the left boundary on lane %d (lawnmower used, boundary=%.1f)", entityID, zombieTypeStr, lane, defeatBoundary)

				// 触发完整的僵尸获胜流程（Story 8.8）
				s.triggerZombiesWonFlow(entityID)
				return
			} else {
				// 除草车未使用，不触发失败（让除草车触发）
				log.Printf("[LevelSystem] Zombie (ID:%d, type:%s) reached left boundary on lane %d, waiting for lawnmower to trigger", entityID, zombieTypeStr, lane)
				// 注意：不 return，继续检查其他行是否有除草车用完的情况
			}
		}
	}
}

// checkDefeatWithoutLawnmower 检查失败条件（无除草车，原逻辑）
// Story 17.9: 使用类型化进家边界
func (s *LevelSystem) checkDefeatWithoutLawnmower() {
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if !isZombieType(behavior.Type) {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// Story 17.9: 获取类型化的进家边界
		zombieTypeStr := s.behaviorTypeToString(behavior.Type)
		defeatBoundary := s.getDefeatBoundary(zombieTypeStr)

		// 僵尸到达左边界，游戏失败
		if pos.X < defeatBoundary {
			s.gameState.SetGameResult("lose")
			log.Printf("[LevelSystem] Defeat! Zombie (ID:%d, type:%s) reached the left boundary at X=%.0f (boundary=%.1f)", entityID, zombieTypeStr, pos.X, defeatBoundary)

			// 触发完整的僵尸获胜流程（Story 8.8）
			s.triggerZombiesWonFlow(entityID)
			return
		}
	}
}

// getDefeatBoundary 获取僵尸类型对应的进家边界（世界坐标）
//
// Story 17.9: 使用类型化进家边界配置
// 如果未加载物理配置，使用现有 DefeatBoundaryX 常量（向后兼容）
//
// 参数:
//   - zombieType: 僵尸类型字符串（如 "basic", "polevaulter"）
//
// 返回:
//   - float64: 进家判定边界（世界坐标）
func (s *LevelSystem) getDefeatBoundary(zombieType string) float64 {
	if s.zombiePhysics != nil {
		boundary := s.zombiePhysics.GetDefeatBoundary(zombieType)
		return config.GridToWorldX(boundary)
	}
	// 兼容现有常量
	return DefeatBoundaryX
}

// behaviorTypeToString 将 BehaviorType 转换为僵尸类型字符串
//
// Story 17.9: 用于查找进家边界配置
//
// 参数:
//   - behaviorType: 行为类型枚举
//
// 返回:
//   - string: 僵尸类型字符串
func (s *LevelSystem) behaviorTypeToString(behaviorType components.BehaviorType) string {
	switch behaviorType {
	case components.BehaviorZombieBasic, components.BehaviorZombieEating, components.BehaviorZombieDying:
		return "basic"
	case components.BehaviorZombieConehead:
		return "conehead"
	case components.BehaviorZombieBuckethead:
		return "buckethead"
	// 以下为未来扩展的僵尸类型，当前返回默认值
	default:
		return "default"
	}
}

// getEntityLane 根据实体的Y坐标计算所在行（1-5）
func (s *LevelSystem) getEntityLane(y float64) int {
	// 使用与 LawnmowerSystem 相同的计算方法
	offsetY := y - config.GridWorldStartY
	row := int(offsetY / config.CellHeight)
	lane := row + 1

	// 限制范围
	if lane < 1 {
		lane = 1
	}
	if lane > 5 {
		lane = 5
	}

	return lane
}

// isZombieType 判断行为类型是否是僵尸（包括各种活跃状态）
// 返回 true 的状态会被除草车消灭
func isZombieType(behaviorType components.BehaviorType) bool {
	return behaviorType == components.BehaviorZombieBasic ||
		behaviorType == components.BehaviorZombieEating ||
		behaviorType == components.BehaviorZombieConehead ||
		behaviorType == components.BehaviorZombieBuckethead ||
		behaviorType == components.BehaviorZombieFlag ||
		behaviorType == components.BehaviorZombiePolevaulter
}

// triggerFinalWaveWarning 已废弃：统一由 FlagWaveWarningSystem 处理
//
// @deprecated Story 17.6+统一后，此方法不再被调用
func (s *LevelSystem) triggerFinalWaveWarning() {
	log.Printf("[LevelSystem] [DEPRECATED] triggerFinalWaveWarning called - should use FlagWaveWarningSystem")

	s.gameState.ShowingFinalWave = true

	// 使用 AudioManager 统一管理音效（Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_AWOOGA")
	}

	// FinalWave.reanim 使用绝对屏幕坐标，传入 (0, 0) 让动画按原始坐标渲染
	warningEntity, err := entities.NewFinalWaveWarningEntity(
		s.entityManager,
		s.resourceManager,
		0, // 屏幕原点 X
		0, // 屏幕原点 Y
	)

	if err != nil {
		log.Printf("[LevelSystem] ERROR: Failed to create FinalWave warning entity: %v", err)
		return
	}

	ecs.AddComponent(s.entityManager, warningEntity, &components.AnimationCommandComponent{
		UnitID:    "finalwave",
		ComboName: "warning",
		Processed: false,
	})
}

// triggerZombiesWonFlow 触发完整的四阶段僵尸获胜流程（Story 8.8）
//
// 相比旧版 triggerZombiesWon()，新流程包含：
// - Phase 1: 游戏冻结（1.5秒）- 植物停止攻击、子弹消失、UI隐藏、音乐淡出
// - Phase 2: 僵尸入侵动画 - 触发僵尸继续行走至屏幕外
// - Phase 3: 惨叫与"吃脑子"动画（3-4秒）- 惨叫音效、咀嚼音效、屏幕抖动
// - Phase 4: 游戏结束对话框 - 显示"再次尝试"/"返回主菜单"按钮
//
// 参数：
//   - triggerZombieID: 触发失败的僵尸实体ID（在 Phase 2 继续移动）
//
// 执行步骤：
//  1. 创建流程控制实体
//  2. 添加 ZombiesWonPhaseComponent（阶段状态机）
//  3. 添加 GameFreezeComponent（冻结标记）
//  4. 交由 ZombiesWonPhaseSystem 管理流程推进
//
// 注意：
//   - 流程由 ZombiesWonPhaseSystem 自动推进，无需手动管理
//   - 遵循 ECS 零耦合原则，通过组件通信
func (s *LevelSystem) triggerZombiesWonFlow(triggerZombieID ecs.EntityID) {
	log.Printf("[LevelSystem] Triggering zombies won flow (4-phase), zombie ID:%d", triggerZombieID)

	// 使用 ZombiesWonPhaseSystem 提供的标准接口启动流程
	// 遵循 DRY 原则，复用业务逻辑
	flowEntityID := StartZombiesWonFlow(s.entityManager, triggerZombieID)

	log.Printf("[LevelSystem] Flow entity created (ID:%d), zombie ID:%d", flowEntityID, triggerZombieID)
}

// cleanupAllSunEntities 清理场上所有阳光实体
//
// 在游戏胜利时调用，清理所有天空掉落和向日葵生成的阳光
// 确保奖励动画阶段场上不会残留阳光
func (s *LevelSystem) cleanupAllSunEntities() {
	// 查询所有阳光实体
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](s.entityManager)

	if len(sunEntities) == 0 {
		return
	}

	// 销毁所有阳光实体
	for _, entityID := range sunEntities {
		s.entityManager.DestroyEntity(entityID)
	}

	// 立即清理标记的实体
	s.entityManager.RemoveMarkedEntities()

	log.Printf("[LevelSystem] Cleaned up %d sun entities on victory", len(sunEntities))
}
