package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
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

// checkTriggerConditions 检测是否有僵尸到达左侧，触发除草车
// 对应 Task 5: 实现除草车触发逻辑
func (s *LawnmowerSystem) checkTriggerConditions() {
	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, zombieID := range zombieEntities {
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)

		// 只检查僵尸类型（跳过植物等其他实体）
		if !isZombieType(behavior.Type) {
			continue
		}

		// 僵尸到达左侧边界
		if pos.X < config.LawnmowerTriggerBoundary {
			// 计算僵尸所在行（1-5）
			lane := s.getEntityLane(pos.Y)

			// 触发该行的除草车
			s.triggerLawnmower(lane)
		}
	}
}

// triggerLawnmower 触发指定行的除草车
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

		if lawnmower.Lane == lane && !lawnmower.IsTriggered {
			// 触发除草车
			lawnmower.IsTriggered = true
			lawnmower.IsMoving = true

			// 播放音效（使用资源 ID 而不是相对路径）
			if s.resourceManager != nil {
				player := s.resourceManager.GetAudioPlayer("SOUND_LAWNMOWER")
				if player != nil {
					player.Rewind()
					player.Play()
				} else {
					log.Printf("[LawnmowerSystem] Warning: Lawnmower audio (SOUND_LAWNMOWER) not loaded")
				}
			}

			// 恢复动画播放（触发后开始播放车轮滚动动画）
			// 注意：不切换动画，继续使用 anim_normal，只是取消暂停
			if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, lawnmowerID); ok {
				reanim.IsPaused = false // 恢复播放，车轮开始滚动、轻微晃动
				log.Printf("[LawnmowerSystem] Resumed animation for lawnmower on lane %d", lane)
			}

			log.Printf("[LawnmowerSystem] Lawnmower triggered on lane %d", lane)
			break
		}
	}
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

			// 计算僵尸所在行
			zombieLane := s.getEntityLane(zombiePos.Y)

			// 检查是否在同一行
			if zombieLane != lawnmower.Lane {
				continue
			}

			// 碰撞检测：僵尸 X 坐标在除草车 X ± CollisionRange 范围内
			distance := math.Abs(zombiePos.X - lawnmowerPos.X)
			if distance < config.LawnmowerCollisionRange {
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
		UnitID:    "zombie",
		ComboName: "death",
		Processed: false,
	})
	// 设置为不循环
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		reanim.IsLooping = false
	}
	log.Printf("[LawnmowerSystem] 僵尸 %d 开始播放死亡动画（回退模式）", zombieID)

	// 4. 触发粒子效果（手臂和头部掉落）
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

	// 转换为 LocatorFrame 数组
	frames := make([]components.LocatorFrame, len(locatorTrack.Frames))
	for i, frame := range locatorTrack.Frames {
		frames[i] = components.LocatorFrame{
			X:      getFloatValue(frame.X),
			Y:      getFloatValue(frame.Y),
			SkewX:  getFloatValue(frame.SkewX),
			SkewY:  getFloatValue(frame.SkewY),
			ScaleX: getFloatValue(frame.ScaleX),
			ScaleY: getFloatValue(frame.ScaleY),
		}

		// 设置默认值
		if frames[i].ScaleX == 0 {
			frames[i].ScaleX = 1.0
		}
		if frames[i].ScaleY == 0 {
			frames[i].ScaleY = 1.0
		}
	}

	log.Printf("[LawnmowerSystem] 成功加载 locator 轨道：%d 帧", len(frames))
	return frames, nil
}

// getFloatValue 获取浮点指针的值，如果为 nil 则返回 0.0
func getFloatValue(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}

// updateSquashAnimations 更新所有正在播放的压扁动画
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) updateSquashAnimations(deltaTime float64) {
	// 查询所有拥有 SquashAnimationComponent 的实体（僵尸）
	squashEntities := ecs.GetEntitiesWith3[
		*components.SquashAnimationComponent,
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	for _, zombieID := range squashEntities {
		squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombieID)
		position, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		reanim, _ := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID)

		// 累积已播放时间
		squashAnim.ElapsedTime += deltaTime

		// 检查动画是否已完成
		if squashAnim.IsComplete() {
			// 动画结束，触发死亡
			s.triggerDeathAfterSquash(zombieID)
			continue
		}

		// 计算当前帧索引
		frameIndex := squashAnim.GetCurrentFrameIndex()
		if frameIndex >= len(squashAnim.LocatorFrames) {
			frameIndex = len(squashAnim.LocatorFrames) - 1
		}

		frame := squashAnim.LocatorFrames[frameIndex]

		// 应用 locator 变换到僵尸

		// 1. 位置：跟随除草车移动 + locator 偏移
		//    如果有关联的除草车，跟随除草车的 X 坐标
		if squashAnim.LawnmowerEntityID != 0 {
			if lawnmowerPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, squashAnim.LawnmowerEntityID); ok {
				position.X = lawnmowerPos.X + frame.X
			} else {
				// 除草车已删除，使用原始位置 + 偏移
				position.X = squashAnim.OriginalPosX + frame.X
			}
		} else {
			position.X = squashAnim.OriginalPosX + frame.X
		}
		position.Y = squashAnim.OriginalPosY + frame.Y

		// 2. 应用变换到 Reanim 动画的所有可见轨道
		// 关键：我们需要修改僵尸 Reanim 的每个部件帧，应用 locator 的缩放和倾斜
		if len(reanim.CachedRenderData) > 0 {
			// 应用 locator 变换到每个渲染部件
			for i := range reanim.CachedRenderData {
				// 应用缩放
				reanim.CachedRenderData[i].Frame.ScaleX = &frame.ScaleX
				reanim.CachedRenderData[i].Frame.ScaleY = &frame.ScaleY

				// 应用旋转（skew）
				reanim.CachedRenderData[i].Frame.SkewX = &frame.SkewX
				reanim.CachedRenderData[i].Frame.SkewY = &frame.SkewY
			}

			// 标记缓存失效，强制重新渲染
			reanim.LastRenderFrame = -1
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

// triggerDeathAfterSquash 压扁动画结束后触发僵尸死亡
// 参数:
//   - zombieID: 僵尸实体ID
func (s *LawnmowerSystem) triggerDeathAfterSquash(zombieID ecs.EntityID) {
	// 1. 移除 SquashAnimationComponent
	ecs.RemoveComponent[*components.SquashAnimationComponent](s.entityManager, zombieID)
	log.Printf("[LawnmowerSystem] 僵尸 %d 压扁动画完成，移除 SquashAnimationComponent", zombieID)

	// 2. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 警告：僵尸 %d 缺少 BehaviorComponent", zombieID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[LawnmowerSystem] 僵尸 %d 切换为 BehaviorZombieDying", zombieID)

	// 3. 播放死亡动画（单次播放，不循环）
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: false,
	})
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		reanim.IsLooping = false
		reanim.IsPaused = false // 恢复播放（之前暂停了）

		// 重置变换（移除压扁效果）
		if len(reanim.CachedRenderData) > 0 {
			for i := range reanim.CachedRenderData {
				reanim.CachedRenderData[i].Frame.ScaleX = nil
				reanim.CachedRenderData[i].Frame.ScaleY = nil
				reanim.CachedRenderData[i].Frame.SkewX = nil
				reanim.CachedRenderData[i].Frame.SkewY = nil
			}
			reanim.LastRenderFrame = -1
		}
	}
	log.Printf("[LawnmowerSystem] 僵尸 %d 开始播放死亡动画（压扁后）", zombieID)

	// 4. 触发粒子效果（手臂和头部掉落）
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", zombieID)
		return
	}

	angleOffset := 180.0 // 粒子向左飞出

	// 触发手臂掉落粒子
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieArm",
		position.X, position.Y,
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
		position.X, position.Y,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建头部粒子失败: %v", err)
	}

	log.Printf("[LawnmowerSystem] 僵尸 %d 触发死亡粒子效果", zombieID)
}
