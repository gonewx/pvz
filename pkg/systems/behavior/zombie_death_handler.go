package behavior

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/types"
)

// triggerZombieDeath 触发僵尸死亡状态转换
// 当僵尸生命值 <= 0 时调用，将僵尸从正常行为状态切换到死亡动画播放状态
// 根据 DeathEffectType 决定死亡效果：
// - DeathEffectNormal: 正常死亡，播放头部、手臂掉落效果
// - DeathEffectInstant: 瞬间死亡（保龄球坚果撞击），跳过手臂掉落效果，但保留头部掉落
// 注意：手臂掉落效果在 updateZombieDamageState 中根据 DeathEffectType 判断是否触发

func (s *BehaviorSystem) triggerZombieDeath(entityID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", entityID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[BehaviorSystem] 僵尸 %d 行为切换为 BehaviorZombieDying", entityID)

	// 获取僵尸位置（用于回退方案）
	position, hasPosition := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !hasPosition {
		log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", entityID)
	} else {
		// 【重要】先获取头部轨道位置，再隐藏轨道
		// GetTrackWorldPosition 从 CachedRenderData 中查找轨道，被隐藏的轨道不在缓存中
		// 使用 AnchorBottomCenter 获取脖子位置（头部底部中心）
		headX, headY := position.X, position.Y // 回退值
		if s.reanimSystem != nil {
			if trackX, trackY, found := s.reanimSystem.GetTrackWorldPosition(entityID, "anim_head1", systems.AnchorBottomCenter); found {
				headX, headY = trackX, trackY
				log.Printf("[BehaviorSystem] 僵尸 %d 头部轨道位置: (%.1f, %.1f)", entityID, headX, headY)
			} else {
				log.Printf("[BehaviorSystem] 警告：僵尸 %d 无法获取头部轨道位置，使用回退值", entityID)
			}
		}

		// 播放头部掉落音效
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_LIMBS_POP")
		}

		// 触发僵尸头部掉落粒子效果
		// 从头部轨道位置（脖子）发射粒子
		var err error
		if behavior.UnitID == types.UnitIDZombiePolevaulter {
			// 撑杆僵尸使用专用头部图片
			_, err = entities.CreateParticleEffectWithImage(
				s.entityManager,
				s.resourceManager,
				"ZombieHead", // 复用 ZombieHead 粒子配置
				headX, headY,
				"IMAGE_ZOMBIEPOLEVAULTERHEAD", // 使用撑杆僵尸专用头部图片
				0.0,                           // 使用定义文件中的角度
			)
		} else {
			// 其他僵尸使用默认头部图片
			_, err = entities.CreateParticleEffect(
				s.entityManager,
				s.resourceManager,
				"ZombieHead", // 粒子效果名称（不带.xml后缀）
				headX, headY,
			)
		}
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建僵尸头部掉落粒子效果失败: %v", err)
			// 不阻塞游戏逻辑，游戏继续运行
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 触发头部掉落粒子效果，位置: (%.1f, %.1f)", entityID, headX, headY)
		}

		// 旗帜僵尸特殊处理：触发旗帜掉落粒子效果
		if behavior.UnitID == "zombie_flag" {
			_, err := entities.CreateParticleEffect(
				s.entityManager,
				s.resourceManager,
				"ZombieFlag", // 旗帜掉落粒子效果
				position.X, position.Y, // 旗帜使用脚底位置
			)
			if err != nil {
				log.Printf("[BehaviorSystem] 警告：创建旗帜掉落粒子效果失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 旗帜僵尸 %d 触发旗帜掉落粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
			}
		}
	}

	// 2. 隐藏头部轨道（头掉落效果）
	// 直接修改 HiddenTracks 字段而不调用废弃的 HideTrack API
	// 注意：旗帜僵尸的旗帜隐藏在 zombie_flag.yaml 的 death/death_damaged 配置中处理
	// 无论是普通死亡还是瞬间死亡，都隐藏头部轨道
	{
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if reanim.HiddenTracks == nil {
				reanim.HiddenTracks = make(map[string]bool)
			}
			// 隐藏所有头部相关轨道
			headTracks := []string{"anim_head1", "anim_head2"}
			for _, trackName := range headTracks {
				reanim.HiddenTracks[trackName] = true
			}
			log.Printf("[BehaviorSystem] 僵尸 %d 头部掉落，隐藏轨道: %v", entityID, headTracks)
		}
	}

	// 3. 移除 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 4. 使用 AnimationCommand 组件播放死亡动画（不循环）
	// 使用组件通信替代直接调用
	// 使用配置驱动的动画组合（自动隐藏装备轨道）
	// 旗帜僵尸特殊处理：根据 ArmLost 选择死亡动画
	// 随机选择 death 或 death2 动画
	deathComboName := "death"
	if rand.Float32() < 0.5 {
		deathComboName = "death2"
	}
	unitID := behavior.UnitID
	if unitID == "" {
		unitID = types.UnitIDZombie // 后备默认值
	}
	if unitID == "zombie_flag" {
		if health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			if health.ArmLost {
				deathComboName = deathComboName + "_damaged"
				log.Printf("[BehaviorSystem] 旗帜僵尸 %d 使用受损死亡动画 (%s)", entityID, deathComboName)
			}
		}
	}
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    unitID,
		ComboName: deathComboName,
		Processed: false,
	})
	log.Printf("[BehaviorSystem] 僵尸 %d 添加死亡动画命令 (%s/%s)", entityID, unitID, deathComboName)

	// 设置为不循环
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
		reanim.IsLooping = false
	}

	log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画已开始播放 (anim_death, 不循环)", entityID)
}

// handleZombieDyingBehavior 处理僵尸死亡动画播放
// 当死亡动画完成后，删除僵尸实体并增加击杀计数

func (s *BehaviorSystem) handleZombieDyingBehavior(entityID ecs.EntityID) {
	// 获取 ReanimComponent
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		// 如果没有 ReanimComponent，直接删除僵尸
		log.Printf("[BehaviorSystem] 死亡中的僵尸 %d 缺少 ReanimComponent，直接删除", entityID)
		// 僵尸死亡，增加计数
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
		return
	}

	// 检查死亡动画是否完成
	// 使用 IsFinished 标志来判断非循环动画是否已完成
	if reanim.IsFinished {
		// 使用 CurrentFrame 替代 AnimStates
		log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画完成 (frame %d)，删除实体",
			entityID, reanim.CurrentFrame)
		// 僵尸死亡，增加计数
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
	}
}

// triggerZombieExplosionDeath 触发僵尸爆炸烧焦死亡动画
//
// 当僵尸被爆炸类攻击（樱桃炸弹、土豆雷、辣椒等）杀死时调用此方法
// 切换为烧焦死亡行为，播放 Zombie_charred.reanim 动画
//
// 特殊处理：
//   - 撑杆僵尸在跳跃中被炸死时直接消失，不显示烧焦效果
//
// 参数:
//   - entityID: 僵尸实体ID
//
// 使用场景:
//   - 樱桃炸弹爆炸杀死僵尸 (Story 5.4)
//   - 土豆雷爆炸杀死僵尸（未来实现）
//
// 技术说明:
//   - 使用 AnimationCommand 组件触发动画切换
//   - ReanimSystem 的 PlayCombo 负责处理单位切换（Story 5.4.1 重构）
//   - 不隐藏头部轨道（烧焦动画中僵尸整体烧焦，头不掉落）
//   - 不触发粒子效果（爆炸效果已在爆炸时播放）
//   - 参考实现: triggerZombieDeath() (普通死亡)
func (s *BehaviorSystem) triggerZombieExplosionDeath(entityID ecs.EntityID) {
	// 撑杆僵尸在跳跃中被炸死时直接消失，不显示烧焦效果
	if poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, entityID); ok {
		if poleVault.IsJumping {
			log.Printf("[BehaviorSystem] 撑杆僵尸 %d 跳跃中被炸死，直接消失", entityID)
			position, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			posX, posY := 0.0, 0.0
			if position != nil {
				posX, posY = position.X, position.Y
			}
			s.triggerZombieInstantDeath(entityID, posX, posY)
			return
		}
	}

	// 1. 切换行为类型为 BehaviorZombieDyingExplosion
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发爆炸死亡", entityID)
		return
	}
	behavior.Type = components.BehaviorZombieDyingExplosion
	log.Printf("[BehaviorSystem] 僵尸 %d 行为切换为 BehaviorZombieDyingExplosion", entityID)

	// 2. 移除 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 3. 使用 AnimationCommand 触发烧焦死亡动画
	//    Story 5.4.1: ReanimSystem.PlayCombo 现在支持单位切换
	//    当 UnitID 与当前 ReanimName 不同时，自动重新加载 Reanim 数据
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombieCharred, // 指向 zombie_charred 配置
		ComboName: "death",                   // 配置中的 death 组合
		Processed: false,
	})
	log.Printf("[BehaviorSystem] 僵尸 %d 添加烧焦死亡动画命令 (zombie_charred/death)", entityID)

	log.Printf("[BehaviorSystem] 僵尸 %d 烧焦死亡动画已开始播放 (zombie_charred/death, 不循环)", entityID)
}

// handleZombieDyingExplosionBehavior 处理僵尸爆炸烧焦死亡动画播放
//
// 当僵尸被爆炸类攻击杀死时，播放专用的烧焦黑化动画
// 动画播放完成后删除僵尸实体并增加消灭计数
//
// 参数:
//   - entityID: 僵尸实体ID
//
// 技术说明:
//   - 烧焦动画为非循环动画，ReanimSystem 会自动推进帧
//   - 当 reanim.IsFinished = true 时，动画完成
//   - 必须在删除实体前增加计数，否则计数丢失
//   - 参考实现: handleZombieDyingBehavior() (普通死亡)
func (s *BehaviorSystem) handleZombieDyingExplosionBehavior(entityID ecs.EntityID) {
	// 获取 ReanimComponent
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		// 如果没有 ReanimComponent，直接删除僵尸
		log.Printf("[BehaviorSystem] 爆炸死亡中的僵尸 %d 缺少 ReanimComponent，直接删除", entityID)
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
		return
	}

	// 检查烧焦死亡动画是否完成
	if reanim.IsFinished {
		log.Printf("[BehaviorSystem] 僵尸 %d 烧焦死亡动画完成，删除实体", entityID)

		// 增加僵尸消灭计数
		s.gameState.IncrementZombiesKilled()

		// 删除僵尸实体
		s.entityManager.DestroyEntity(entityID)
	}
}

// triggerZombieInstantDeath 触发僵尸瞬间消失死亡
//
// 当僵尸被土豆地雷等爆炸攻击杀死时，直接删除实体
// 不播放变焦动画，僵尸瞬间消失（爆炸粒子效果已在爆炸时播放）
//
// 参数:
//   - entityID: 僵尸实体ID
//   - x, y: 僵尸位置（用于日志）
func (s *BehaviorSystem) triggerZombieInstantDeath(entityID ecs.EntityID, x, y float64) {
	log.Printf("[BehaviorSystem] 僵尸 %d 瞬间消失，位置: (%.1f, %.1f)", entityID, x, y)

	// 先将行为设置为"已删除"状态，防止同一帧内被其他系统重复处理
	// 这样在后续的 handleZombieBasicBehavior 检测到 health <= 0 时会跳过
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if ok {
		behavior.Type = components.BehaviorZombieDying
	}

	// 增加僵尸消灭计数
	s.gameState.IncrementZombiesKilled()

	// 直接删除僵尸实体（不播放死亡动画，爆炸粒子效果已在爆炸时播放）
	s.entityManager.DestroyEntity(entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 已删除", entityID)
}

// updateZombieDamageState 根据生命值更新僵尸的受伤状态
// 僵尸有三个受伤阶段：
// 1. 健康（HP > 90）：完整外观
// 2. 掉手臂（HP <= 90 且 HP > 0）：隐藏外侧手臂
// 3. 掉头（HP <= 0）：无头状态（在 triggerZombieDeath 中处理）
//
// 特殊情况：
// - DeathEffectInstant（保龄球坚果撞击）：跳过手臂掉落效果，直接进入死亡状态

func (s *BehaviorSystem) updateZombieDamageState(entityID ecs.EntityID, health *components.HealthComponent) {
	// 生命值阈值：90（33%，根据原版游戏数据）
	const armLostThreshold = 90

	// 检查是否应该掉手臂（生命值 <= 90 且手臂尚未掉落）
	if health.CurrentHealth <= armLostThreshold && !health.ArmLost {
		// 标记手臂已掉落，防止重复触发
		health.ArmLost = true

		// 如果是瞬间死亡（保龄球坚果撞击），跳过手臂掉落的视觉效果
		// 僵尸直接进入死亡状态，不需要播放手臂掉落动画和粒子
		if health.DeathEffectType == components.DeathEffectInstant {
			log.Printf("[BehaviorSystem] 僵尸 %d 瞬间死亡，跳过手臂掉落效果", entityID)
			return
		}

		// 播放手臂掉落音效
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_LIMBS_POP")
		}

		// 获取行为组件，检查僵尸类型
		behavior, hasBehavior := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 【重要】先获取轨道位置，再隐藏轨道
		// GetTrackWorldPosition 从 CachedRenderData 中查找轨道，被隐藏的轨道不在缓存中
		// 获取僵尸位置（用于回退方案）
		position, hasPosition := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !hasPosition {
			log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发手臂掉落粒子", entityID)
			return
		}

		// 使用轨道位置方案获取手臂的世界坐标
		// 修复：僵尸使用脚底锚点（CenterOffsetY = maxY），position.Y 是脚底位置
		// 手臂实际在脚底上方 60-100 像素，需要通过轨道位置获取精确坐标
		// 使用 AnchorBottomCenter 获取手部底部中心位置，更适合粒子效果生成
		armX, armY := position.X, position.Y // 回退值
		if s.reanimSystem != nil {
			if trackX, trackY, found := s.reanimSystem.GetTrackWorldPosition(entityID, "Zombie_outerarm_hand", systems.AnchorBottomCenter); found {
				armX, armY = trackX, trackY
				log.Printf("[BehaviorSystem] 僵尸 %d 手臂轨道位置: (%.1f, %.1f)", entityID, armX, armY)
			} else {
				log.Printf("[BehaviorSystem] 警告：僵尸 %d 无法获取手臂轨道位置，使用回退值", entityID)
			}
		}

		// 隐藏手臂轨道（手臂掉落效果）
		// 直接修改 HiddenTracks 字段而不调用废弃的 HideTrack API
		// 根据僵尸类型选择正确的轨道名称
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if reanim.HiddenTracks == nil {
				reanim.HiddenTracks = make(map[string]bool)
			}
			var armTracks []string
			if hasBehavior && behavior.UnitID == types.UnitIDZombiePolevaulter {
				// 撑杆僵尸使用专用轨道名
				armTracks = []string{"Zombie_outerarm_hand", "Zombie_polevaulter_outerarm_upper", "Zombie_polevaulter_outerarm_lower"}
			} else {
				// 普通僵尸使用默认轨道名
				armTracks = []string{"Zombie_outerarm_hand", "Zombie_outerarm_upper", "Zombie_outerarm_lower"}
			}
			for _, trackName := range armTracks {
				reanim.HiddenTracks[trackName] = true
			}
			log.Printf("[BehaviorSystem] 僵尸 %d 手臂掉落，隐藏轨道: %v", entityID, armTracks)
		}

		log.Printf("[BehaviorSystem] 僵尸 %d 手臂掉落 (HP=%d/%d)",
			entityID, health.CurrentHealth, health.MaxHealth)

		// 旗帜僵尸特殊处理：根据当前状态切换到对应的受损动画
		if hasBehavior && behavior.UnitID == "zombie_flag" {
			var damagedComboName string
			switch behavior.Type {
			case components.BehaviorZombieEating:
				damagedComboName = "eat_damaged"
			case components.BehaviorZombieBasic, components.BehaviorZombieFlag:
				damagedComboName = "walk_damaged"
			}
			if damagedComboName != "" {
				ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
					UnitID:    "zombie_flag",
					ComboName: damagedComboName,
					Processed: false,
				})
				log.Printf("[BehaviorSystem] 旗帜僵尸 %d 受损，切换到 %s 动画", entityID, damagedComboName)
			}
		}

		// 触发僵尸手臂掉落粒子效果
		// 根据僵尸类型选择正确的手臂图片
		var err error
		if hasBehavior && behavior.UnitID == types.UnitIDZombiePolevaulter {
			// 撑杆僵尸只显示手的掉落（因为胳膊部分与普通僵尸不同）
			_, err = entities.CreateParticleEffectWithImage(
				s.entityManager,
				s.resourceManager,
				"ZombieArm", // 复用 ZombieArm 粒子配置
				armX, armY,
				"IMAGE_REANIM_ZOMBIE_OUTERARM_HAND", // 只显示手
				0.0,                                 // 使用定义文件中的角度
			)
		} else {
			// 其他僵尸使用完整的手臂+手图片
			_, err = entities.CreateParticleEffect(
				s.entityManager,
				s.resourceManager,
				"ZombieArm", // 粒子效果名称（不带.xml后缀）
				armX, armY,
			)
		}
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建僵尸手臂掉落粒子效果失败: %v", err)
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 触发手臂掉落粒子效果，位置: (%.1f, %.1f)", entityID, armX, armY)
		}
	}
}
