package systems

import (
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/types"
)

// LawnmowerSystem 除草车系统
// 职责：
// - 检测触发条件（僵尸到达左侧边界）
// - 触发除草车移动
// - 更新除草车位置
// - 检测并消灭路径上的僵尸
// - 管理除草车状态（未触发/移动中/已使用）
type LawnmowerSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager // 用于播放音效
	gameState       *game.GameState       // 用于增加消灭僵尸计数
	stateEntityID   ecs.EntityID          // 全局状态实体ID
}

// NewLawnmowerSystem 创建除草车系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于播放音效和粒子效果）
//   - gs: GameState 实例（用于记录游戏统计）
//
// 返回:
//   - *LawnmowerSystem: 除草车系统实例
func NewLawnmowerSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState) *LawnmowerSystem {
	// 创建全局状态实体
	stateEntity := em.CreateEntity()
	ecs.AddComponent(em, stateEntity, &components.LawnmowerStateComponent{
		UsedLanes: make(map[int]bool),
	})

	return &LawnmowerSystem{
		entityManager:   em,
		resourceManager: rm,
		gameState:       gs,
		stateEntityID:   stateEntity,
	}
}

// Update 更新除草车系统
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) Update(deltaTime float64) {
	// 0. 更新入场动画
	s.updateEnterAnimations(deltaTime)

	// 1. 检测触发条件
	s.checkTriggerConditions()

	// 2. 更新移动中的除草车位置
	s.updateLawnmowerPositions(deltaTime)

	// 3. 更新压扁动画
	s.updateSquashAnimations(deltaTime)

	// 4. 检测并消灭僵尸
	s.checkZombieCollisions()

	// 5. 检测除草车离开屏幕
	s.checkLawnmowerCompletion()
}

// updateEnterAnimations 更新除草车入场动画
// 除草车从屏幕左侧外开出来到达初始位置
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) updateEnterAnimations(deltaTime float64) {
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		// 只处理正在入场的除草车
		if !lawnmower.IsEntering {
			continue
		}

		// 累积入场计时器
		lawnmower.EnterTimer += deltaTime

		// 检查是否还在延迟期间
		if lawnmower.EnterTimer < lawnmower.EnterDelay {
			continue
		}

		// 向右移动到目标位置
		pos.X += lawnmower.EnterSpeed * deltaTime

		// 检查是否到达目标位置
		if pos.X >= lawnmower.EnterTargetX {
			// 到达目标位置，结束入场动画
			pos.X = lawnmower.EnterTargetX
			lawnmower.IsEntering = false

			// 入场完成后暂停动画（静止等待触发）
			if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, lawnmowerID); ok {
				reanim.IsPaused = true
			}

			log.Printf("[LawnmowerSystem] Lawnmower on lane %d enter animation completed at X=%.1f",
				lawnmower.Lane, pos.X)
		}
	}
}

// checkTriggerConditions 检测僵尸与静止除草车的碰撞，触发除草车
// 使用 AABB 碰撞检测替代简单的边界检测
func (s *LawnmowerSystem) checkTriggerConditions() {
	// 获取所有未触发的除草车
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 获取所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		lawnmowerPos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		// 只检查未触发且非入场中的除草车
		if lawnmower.IsTriggered || lawnmower.IsEntering {
			continue
		}

		// 计算除草车的碰撞边界
		lawnmowerLeft := lawnmowerPos.X - config.LawnmowerWidth/2
		lawnmowerRight := lawnmowerPos.X + config.LawnmowerWidth/2
		laneTop := config.GridWorldStartY + float64(lawnmower.Lane-1)*config.CellHeight
		laneBottom := laneTop + config.CellHeight

		// 检测该除草车与所有僵尸的碰撞
		for _, zombieID := range zombieEntities {
			behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
			zombiePos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)

			// 只检查僵尸类型（跳过植物等其他实体）
			if !isZombieType(behavior.Type) {
				continue
			}

			// 只检查已激活的僵尸
			waveState, hasWaveState := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, zombieID)
			if hasWaveState && !waveState.IsActivated {
				continue
			}

			// Y 方向碰撞检测：检查僵尸是否在同一行
			if zombiePos.Y < laneTop || zombiePos.Y > laneBottom {
				continue
			}

			// X 方向碰撞检测：使用 AABB 碰撞检测
			zombieLeft := zombiePos.X - config.ZombieCollisionWidth/2
			zombieRight := zombiePos.X + config.ZombieCollisionWidth/2

			// 碰撞条件：除草车与僵尸在 X 方向有重叠
			if lawnmowerRight >= zombieLeft && lawnmowerLeft <= zombieRight {
				// 触发该除草车
				s.triggerLawnmowerByID(lawnmowerID, lawnmower)
				break // 该除草车已触发，跳出僵尸循环
			}
		}
	}
}

// triggerLawnmower 触发指定行的除草车（通过行号查找）
// 参数:
//   - lane: 行号（1-5）
func (s *LawnmowerSystem) triggerLawnmower(lane int) {
	// 查询该行的除草车
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)

		// 跳过入场中的除草车（入场完成后才能触发）
		if lawnmower.IsEntering {
			continue
		}

		if lawnmower.Lane == lane && !lawnmower.IsTriggered {
			s.triggerLawnmowerByID(lawnmowerID, lawnmower)
			break
		}
	}
}

// triggerLawnmowerByID 通过实体ID直接触发除草车
// 参数:
//   - lawnmowerID: 除草车实体ID
//   - lawnmower: 除草车组件（已获取，避免重复查询）
func (s *LawnmowerSystem) triggerLawnmowerByID(lawnmowerID ecs.EntityID, lawnmower *components.LawnmowerComponent) {
	// 触发除草车
	lawnmower.IsTriggered = true
	lawnmower.IsMoving = true

	// 播放音效（使用 AudioManager 统一管理 - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_LAWNMOWER")
	}

	// 恢复动画播放（触发后开始播放车轮滚动动画）
	// 注意：不切换动画，继续使用 anim_normal，只是取消暂停
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, lawnmowerID); ok {
		reanim.IsPaused = false // 恢复播放，车轮开始滚动、轻微晃动
		log.Printf("[LawnmowerSystem] Resumed animation for lawnmower on lane %d", lawnmower.Lane)
	}

	log.Printf("[LawnmowerSystem] Lawnmower triggered on lane %d (collision detected)", lawnmower.Lane)
}

// updateLawnmowerPositions 更新移动中的除草车位置
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) updateLawnmowerPositions(deltaTime float64) {
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		if lawnmower.IsMoving {
			// 向右移动
			pos.X += lawnmower.Speed * deltaTime
		}
	}
}

// checkZombieCollisions 检测除草车与僵尸的碰撞，消灭僵尸
// 对应 Task 6: 实现僵尸消灭逻辑
func (s *LawnmowerSystem) checkZombieCollisions() {
	// 获取所有移动中的除草车
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 获取所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.HealthComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		lawnmowerPos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		// 只处理移动中的除草车
		if !lawnmower.IsMoving {
			continue
		}

		// 检测同行的僵尸碰撞
		for _, zombieID := range zombieEntities {
			behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
			zombiePos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
			health, _ := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)

			// 只检查僵尸类型
			if !isZombieType(behavior.Type) {
				continue
			}

			// 跳过已死亡的僵尸
			if health.CurrentHealth <= 0 {
				continue
			}

			// 只杀死已激活的僵尸（跳过预生成但未激活的僵尸）
			waveState, hasWaveState := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, zombieID)
			if hasWaveState && !waveState.IsActivated {
				// 预生成的僵尸尚未激活，不应被除草车杀死
				continue
			}

			// Y 方向碰撞检测：检查僵尸中心点是否在除草车所在行的网格范围内
			// 除草车所在行的 Y 范围
			laneTop := config.GridWorldStartY + float64(lawnmower.Lane-1)*config.CellHeight
			laneBottom := laneTop + config.CellHeight

			// 使用僵尸中心点判断是否在同一行
			if zombiePos.Y < laneTop || zombiePos.Y > laneBottom {
				continue
			}

			// X 方向碰撞检测：使用 AABB 碰撞检测，考虑实体的实际宽度
			// 除草车前沿（右边缘）= lawnmowerPos.X + LawnmowerWidth/2
			// 僵尸身体左边缘 = zombiePos.X - ZombieCollisionWidth/2
			lawnmowerFront := lawnmowerPos.X + config.LawnmowerWidth/2
			lawnmowerBack := lawnmowerPos.X - config.LawnmowerWidth/2
			zombieLeft := zombiePos.X - config.ZombieCollisionWidth/2
			zombieRight := zombiePos.X + config.ZombieCollisionWidth/2

			// 碰撞条件：除草车与僵尸在 X 方向有重叠
			if lawnmowerFront >= zombieLeft && lawnmowerBack <= zombieRight {
				// 除草车碾压僵尸，触发死亡动画
				// 不再直接删除，而是播放死亡动画和粒子效果

				// 注意：不在这里调用 IncrementZombiesKilled()
				// 计数统一在 BehaviorSystem.handleZombieDyingBehavior() 中进行
				// 避免重复计数（除草车触发一次 + 死亡动画完成一次）

				log.Printf("[LawnmowerSystem] Lawnmower on lane %d killed zombie at (%.1f, %.1f)",
					lawnmower.Lane, zombiePos.X, zombiePos.Y)

				// 触发僵尸死亡（播放动画和粒子效果）
				s.triggerZombieDeath(zombieID)
			}
		}
	}
}

// checkLawnmowerCompletion 检测除草车是否离开屏幕
func (s *LawnmowerSystem) checkLawnmowerCompletion() {
	// 获取全局状态组件
	state, ok := ecs.GetComponent[*components.LawnmowerStateComponent](s.entityManager, s.stateEntityID)
	if !ok {
		log.Printf("[LawnmowerSystem] Warning: LawnmowerStateComponent not found")
		return
	}

	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		// 检测除草车是否离开屏幕（X > 删除边界）
		if lawnmower.IsMoving && pos.X > config.LawnmowerDeletionBoundary {
			// 标记该行除草车为已使用
			state.UsedLanes[lawnmower.Lane] = true

			// 删除除草车实体
			s.entityManager.DestroyEntity(lawnmowerID)

			log.Printf("[LawnmowerSystem] Lawnmower on lane %d completed and removed", lawnmower.Lane)
		}
	}
}

// getEntityLane 根据实体的Y坐标计算所在行（1-5）
// 参数:
//   - y: 实体的世界坐标Y
//
// 返回:
//   - int: 行号（1-5）
func (s *LawnmowerSystem) getEntityLane(y float64) int {
	// 计算相对于网格起点的Y偏移
	offsetY := y - config.GridWorldStartY

	// 计算行号（0-4）
	row := int(offsetY / config.CellHeight)

	// 转换为行号（1-5）
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

// GetStateEntityID 返回全局状态实体ID（用于 LevelSystem 访问）
func (s *LawnmowerSystem) GetStateEntityID() ecs.EntityID {
	return s.stateEntityID
}

// HasActiveLawnmowers 检查是否有活跃的（移动中的）除草车
// 用于胜利条件判断：除草车完全消失后才能显示胜利动画
// 返回:
//   - bool: 如果有任何除草车正在移动，返回 true
func (s *LawnmowerSystem) HasActiveLawnmowers() bool {
	lawnmowers := ecs.GetEntitiesWith1[*components.LawnmowerComponent](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, ok := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		if ok && lawnmower.IsMoving {
			return true
		}
	}

	return false
}

// triggerZombieDeath 触发僵尸压扁动画
// 除草车碾压僵尸时调用此方法，不再直接切换为死亡状态
// 而是先播放压扁动画，动画结束后再触发死亡
func (s *LawnmowerSystem) triggerZombieDeath(zombieID ecs.EntityID) {
	// 1. 获取僵尸当前位置和行为组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 僵尸 %d 缺少 PositionComponent，无法触发压扁动画", zombieID)
		return
	}

	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发压扁动画", zombieID)
		return
	}

	// 2. 获取对应的除草车实体ID（用于跟随移动）
	lawnmowers := ecs.GetEntitiesWith1[*components.LawnmowerComponent](s.entityManager)
	var lawnmowerID ecs.EntityID = 0
	zombieLane := s.getEntityLane(position.Y)

	for _, lwID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lwID)
		if lawnmower.Lane == zombieLane && lawnmower.IsMoving {
			lawnmowerID = lwID
			break
		}
	}

	// 3. 加载 locator 轨道数据
	locatorFrames, err := s.loadLocatorFrames()
	if err != nil || len(locatorFrames) == 0 {
		log.Printf("[LawnmowerSystem] 加载 locator 轨道失败，回退到普通死亡动画")
		// 回退到旧逻辑：直接切换为 BehaviorZombieDying
		s.triggerZombieDeathFallback(zombieID)
		return
	}

	// 4. 添加 SquashAnimationComponent（开始压扁动画）
	duration := float64(len(locatorFrames)) / 12.0 // 8 帧 / 12 FPS ≈ 0.667 秒

	ecs.AddComponent(s.entityManager, zombieID, &components.SquashAnimationComponent{
		ElapsedTime:       0.0,
		Duration:          duration,
		LocatorFrames:     locatorFrames,
		CurrentFrameIndex: 0,
		OriginalPosX:      position.X,
		OriginalPosY:      position.Y,
		LawnmowerEntityID: lawnmowerID,
		IsCompleted:       false,
	})

	log.Printf("[LawnmowerSystem] 僵尸 %d 开始压扁动画（时长 %.2f 秒，%d 帧）",
		zombieID, duration, len(locatorFrames))

	// 5. 移除速度组件（僵尸停止自主移动）
	if ecs.HasComponent[*components.VelocityComponent](s.entityManager, zombieID) {
		ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, zombieID)
		log.Printf("[LawnmowerSystem] 僵尸 %d 移除 VelocityComponent（停止移动）", zombieID)
	}

	// 6. 暂停僵尸当前动画（定格为当前帧）
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		reanim.IsPaused = true
		log.Printf("[LawnmowerSystem] 僵尸 %d 暂停动画（定格当前帧）", zombieID)
	}

	// 7. 切换行为类型为 BehaviorZombieSquashing（新状态）
	// 注意：不立即切换为 BehaviorZombieDying，避免 BehaviorSystem 干扰
	behavior.Type = components.BehaviorZombieSquashing

	// 注意：不在这里触发粒子效果和死亡动画
	// 这些会在压扁动画结束后由 triggerDeathAfterSquash() 触发
}

// triggerZombieDeathFallback 回退到旧逻辑：直接触发死亡动画和粒子效果
// 当 locator 轨道加载失败时使用
func (s *LawnmowerSystem) triggerZombieDeathFallback(zombieID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", zombieID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[LawnmowerSystem] 僵尸 %d 行为切换为 BehaviorZombieDying（回退模式）", zombieID)

	// 2. 移除速度组件（僵尸停止移动）
	if ecs.HasComponent[*components.VelocityComponent](s.entityManager, zombieID) {
		ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, zombieID)
		log.Printf("[LawnmowerSystem] 僵尸 %d 移除 VelocityComponent（停止移动）", zombieID)
	}

	// 3. 播放死亡动画（单次播放，不循环）
	// 使用 AnimationCommand 组件播放配置的动画组合（自动隐藏装备轨道）
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombie,
		ComboName: "death",
		Processed: false,
	})
	// 设置为不循环
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		reanim.IsLooping = false
	}
	log.Printf("[LawnmowerSystem] 僵尸 %d 开始播放死亡动画（回退模式）", zombieID)

	// 4. 触发粒子效果（手臂和头部掉落）
	// Story 10.6: 添加安全检查，避免测试环境中的 ResourceManager 为空或未初始化
	if s.resourceManager == nil {
		log.Printf("[LawnmowerSystem] ResourceManager 未初始化，跳过粒子效果")
		return
	}

	// 这与 BehaviorSystem 的逻辑一致
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", zombieID)
		return
	}

	// 除草车碾压的僵尸通常向左走，需要翻转粒子方向
	angleOffset := 180.0 // 默认翻转（适合僵尸向左走）

	// 触发僵尸手臂掉落粒子效果
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieArm",
		position.X, position.Y,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建僵尸手臂掉落粒子效果失败: %v", err)
	} else {
		log.Printf("[LawnmowerSystem] 僵尸 %d 触发手臂掉落粒子效果", zombieID)
	}

	// 触发僵尸头部掉落粒子效果
	_, err = entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieHead",
		position.X, position.Y,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建僵尸头部掉落粒子效果失败: %v", err)
	} else {
		log.Printf("[LawnmowerSystem] 僵尸 %d 触发头部掉落粒子效果", zombieID)
	}
}

// loadLocatorFrames 从 LawnMoweredZombie.reanim 加载 locator 轨道的帧数据
// 参数:
//   - 无（直接从 ResourceManager 读取）
//
// 返回:
//   - []components.LocatorFrame: locator 轨道的帧数据数组
//   - error: 如果加载失败返回错误
func (s *LawnmowerSystem) loadLocatorFrames() ([]components.LocatorFrame, error) {
	// 检查 ResourceManager 是否存在
	if s.resourceManager == nil {
		log.Printf("[LawnmowerSystem] 警告：ResourceManager 未初始化，无法加载 locator 轨道")
		return nil, nil
	}

	// 从 ResourceManager 获取 LawnMoweredZombie 的 ReanimXML
	reanimXML := s.resourceManager.GetReanimXML("LawnMoweredZombie")
	if reanimXML == nil {
		log.Printf("[LawnmowerSystem] 错误：未找到 LawnMoweredZombie.reanim 数据")
		return nil, nil
	}

	// 查找 locator 轨道
	var locatorTrack *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == "locator" {
			locatorTrack = &reanimXML.Tracks[i]
			break
		}
	}

	if locatorTrack == nil {
		log.Printf("[LawnmowerSystem] 错误：LawnMoweredZombie.reanim 中未找到 locator 轨道")
		return nil, nil
	}

	// 转换为 LocatorFrame 数组（支持属性继承）
	// Reanim 格式中，如果帧未定义某个属性，则继承上一帧的值（Sparse Frames）
	// 如果不处理继承，会导致后续帧的 Rotation 突然归零（出现"垂直压扁" bug）
	frames := make([]components.LocatorFrame, len(locatorTrack.Frames))

	// 初始状态
	var curX, curY, curSkewX, curSkewY float64 = 0, 0, 0, 0
	var curScaleX, curScaleY float64 = 1, 1

	for i, frame := range locatorTrack.Frames {
		// 更新当前值（如果有定义）
		if frame.X != nil {
			curX = *frame.X
		}
		if frame.Y != nil {
			curY = *frame.Y
		}
		if frame.SkewX != nil {
			curSkewX = *frame.SkewX
		}
		if frame.SkewY != nil {
			curSkewY = *frame.SkewY
		}
		if frame.ScaleX != nil {
			curScaleX = *frame.ScaleX
		}
		if frame.ScaleY != nil {
			curScaleY = *frame.ScaleY
		}

		frames[i] = components.LocatorFrame{
			X:      curX,
			Y:      curY,
			SkewX:  curSkewX,
			SkewY:  curSkewY,
			ScaleX: curScaleX,
			ScaleY: curScaleY,
		}
	}

	log.Printf("[LawnmowerSystem] 成功加载 locator 轨道：%d 帧", len(frames))

	// Debug: 打印前 4 帧的数据（包含旋转信息）
	for i := 0; i < 4 && i < len(frames); i++ {
		log.Printf("[LawnmowerSystem]   帧 %d: X=%.1f, Y=%.1f, SkewX=%.1f°, SkewY=%.1f°, ScaleX=%.3f, ScaleY=%.3f",
			i, frames[i].X, frames[i].Y, frames[i].SkewX, frames[i].SkewY, frames[i].ScaleX, frames[i].ScaleY)
	}

	return frames, nil
}

// updateSquashAnimations 更新所有正在播放的压扁动画
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) updateSquashAnimations(deltaTime float64) {
	// 查询所有拥有 SquashAnimationComponent 的实体（僵尸）
	squashEntities := ecs.GetEntitiesWith2[
		*components.SquashAnimationComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, zombieID := range squashEntities {
		squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombieID)
		position, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)

		// 获取 ReanimComponent (用于设置旋转和缩放)
		reanimComp, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID)

		// 累积已播放时间
		squashAnim.ElapsedTime += deltaTime

		// 检查动画是否已完成
		if squashAnim.IsComplete() {
			// Debug: 打印完成时的详细信息
			log.Printf("[LawnmowerSystem] 僵尸 %d 压扁动画判定完成: ElapsedTime=%.3f, Duration=%.3f, Progress=%.1f%%, IsCompleted=%v",
				zombieID, squashAnim.ElapsedTime, squashAnim.Duration, squashAnim.GetProgress()*100, squashAnim.IsCompleted)
			// 动画结束，触发死亡
			s.triggerDeathAfterSquash(zombieID)
			continue
		}

		// 计算当前帧索引
		frameIndex := squashAnim.GetCurrentFrameIndex()
		if frameIndex >= len(squashAnim.LocatorFrames) {
			frameIndex = len(squashAnim.LocatorFrames) - 1
		}
		if frameIndex < 0 || len(squashAnim.LocatorFrames) == 0 {
			// 边界保护：没有帧数据，跳过此僵尸
			continue
		}

		frame := squashAnim.LocatorFrames[frameIndex]

		// 应用 locator 变换到僵尸

		// 1. 位置：使用原始位置 + locator 偏移
		// Story 10.6: 修复"车顶着僵尸"的问题
		// LawnMoweredZombie.reanim 的 locator 轨道 X 值已经包含了位移（先快后慢）
		// 如果叠加除草车的位置（+lawnmowerPos.X），僵尸会移动得比车还快（飞到车前面）
		// 正确逻辑是：僵尸相对于地面（OriginalPosX）移动
		// - Phase 1: 僵尸被铲起向前抛（速度 > 车速），飞到车前
		// - Phase 2: 僵尸落地被压扁（速度 < 车速），车追上并碾过僵尸
		position.X = squashAnim.OriginalPosX + frame.X
		position.Y = squashAnim.OriginalPosY + frame.Y

		// 2. 旋转和缩放：设置 ReanimComponent 的整体变换属性
		// Story 10.6: 修复压扁动画垂直问题
		// 使用 ReanimComponent.Rotation/Scale 实现整体旋转和缩放，而非手动修改 CachedRenderData
		if hasReanim {
			reanimComp.Rotation = frame.SkewX
			reanimComp.ScaleX = frame.ScaleX
			reanimComp.ScaleY = frame.ScaleY
		}

		// Story 10.6 优化：在压扁开始时（frame 4，scaleX开始变小）触发粒子效果和隐藏肢体
		// 避免除草车已经开过头了才出现粒子效果
		if !squashAnim.ParticlesTriggered && frameIndex >= 4 {
			s.triggerSquashParticles(zombieID, squashAnim)
		}

		// 更新组件状态
		squashAnim.CurrentFrameIndex = frameIndex

		// Debug 日志（前 3 帧）
		if frameIndex < 3 {
			log.Printf("[LawnmowerSystem] 僵尸 %d 压扁动画: 帧=%d/%d, 进度=%.1f%%, scaleX=%.3f, skewX=%.1f°",
				zombieID, frameIndex, len(squashAnim.LocatorFrames),
				squashAnim.GetProgress()*100,
				frame.ScaleX, frame.SkewX)
		}
	}
}

// triggerSquashParticles 触发压扁粒子效果并隐藏僵尸肢体
func (s *LawnmowerSystem) triggerSquashParticles(zombieID ecs.EntityID, squashAnim *components.SquashAnimationComponent) {
	if squashAnim.ParticlesTriggered {
		return
	}
	squashAnim.ParticlesTriggered = true

	// 1. 隐藏僵尸头部和手臂
	if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		if reanimComp.HiddenTracks == nil {
			reanimComp.HiddenTracks = make(map[string]bool)
		}
		// 隐藏头部
		reanimComp.HiddenTracks["anim_head1"] = true
		reanimComp.HiddenTracks["anim_head2"] = true

		// 隐藏外侧手臂
		reanimComp.HiddenTracks["Zombie_outerarm_upper"] = true
		reanimComp.HiddenTracks["Zombie_outerarm_lower"] = true
		reanimComp.HiddenTracks["Zombie_outerarm_hand"] = true

		log.Printf("[LawnmowerSystem] 僵尸 %d 隐藏头部和手臂，触发粒子效果", zombieID)
	}

	if s.resourceManager == nil {
		return
	}

	// 2. 触发粒子效果
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		return
	}

	// 粒子生成位置修正
	spawnX := position.X
	spawnY := squashAnim.OriginalPosY

	// 尝试获取除草车位置，使粒子生成与除草车绑定
	if squashAnim.LawnmowerEntityID != 0 {
		if lmPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, squashAnim.LawnmowerEntityID); ok {
			// 使用除草车位置生成粒子（车轮下方）
			// 除草车宽度约 80px，中心锚点。车头在 X + 40，车尾在 X - 40
			// 碾压发生在车底，我们取稍微靠后的位置，模拟"被压扁后爆出"
			spawnX = lmPos.X - 10.0
			log.Printf("[LawnmowerSystem] 使用除草车位置生成粒子: %.1f (僵尸位置: %.1f)", spawnX, position.X)
		}
	}

	// 粒子方向修正
	angleOffset := -90.0

	// 触发手臂掉落粒子
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieArm",
		spawnX, spawnY,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建手臂粒子失败: %v", err)
	}

	// 触发头部掉落粒子
	_, err = entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieHead",
		spawnX, spawnY,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建头部粒子失败: %v", err)
	}
}

// triggerDeathAfterSquash 压扁动画结束后触发僵尸死亡
// Story 10.6: 压扁动画本身就是完整的死亡过程
// 动画结束后应该直接删除僵尸（而不是再播放 BehaviorZombieDying 动画）
//
// 参数:
//   - zombieID: 僵尸实体ID
func (s *LawnmowerSystem) triggerDeathAfterSquash(zombieID ecs.EntityID) {
	// 0. 获取组件信息（在移除组件前）
	squashAnim, ok := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombieID)
	if !ok {
		// 容错处理：如果没有组件，直接删除僵尸
		s.entityManager.DestroyEntity(zombieID)
		return
	}

	// 1. 确保粒子效果已触发（如果动画非常快或者被跳过，可能没触发）
	if !squashAnim.ParticlesTriggered {
		s.triggerSquashParticles(zombieID, squashAnim)
	}

	// 2. 移除 SquashAnimationComponent
	ecs.RemoveComponent[*components.SquashAnimationComponent](s.entityManager, zombieID)
	log.Printf("[LawnmowerSystem] 僵尸 %d 压扁动画完成，移除 SquashAnimationComponent", zombieID)

	// Story 10.6 修复：压扁动画结束后直接删除僵尸
	// 原因：压扁动画本身就是完整的死亡过程（铲起→旋转→压扁）
	//       不需要再播放 BehaviorZombieDying 动画（头部掉落）

	// 3. 增加僵尸消灭计数（必须在删除实体之前）
	// 注意：这里手动计数，因为僵尸不会经过 BehaviorSystem 的死亡流程
	if s.gameState != nil {
		s.gameState.IncrementZombiesKilled()
		log.Printf("[LawnmowerSystem] 僵尸 %d 被除草车消灭，计数+1", zombieID)
	}

	// 4. 直接删除僵尸实体
	s.entityManager.DestroyEntity(zombieID)
	log.Printf("[LawnmowerSystem] 僵尸 %d 压扁完成，已删除", zombieID)
}
