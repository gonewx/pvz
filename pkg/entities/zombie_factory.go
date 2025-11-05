package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// NewZombieEntity 创建普通僵尸实体
// 僵尸从屏幕右侧外生成，可选择是否立即开始移动
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载僵尸 Reanim 资源）
//   - rs: Reanim 系统（用于初始化动画）
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置（通常在屏幕右侧外）
//
// 返回:
//   - ecs.EntityID: 创建的僵尸实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
//
// 注意：僵尸默认创建时速度为0（待命状态），需要通过 WaveSpawnSystem.ActivateWave() 激活
func NewZombieEntity(em *ecs.EntityManager, rm ResourceLoader, rs ReanimSystemInterface, row int, spawnX float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 计算僵尸Y坐标（世界坐标，基于行）
	// 使用和植物相同的Y坐标计算，确保同一行的实体在同一高度
	// 行中心 = GridWorldStartY + row*CellHeight + CellHeight/2.0
	// 使用 config.ZombieVerticalOffset 以便手工调整
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	})

	// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
	// 从 ResourceManager 获取普通僵尸的 Reanim 数据和部件图片
	reanimXML := rm.GetReanimXML("Zombie")
	partImages := rm.GetReanimPartImages("Zombie")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load Zombie Reanim resources")
	}

	// 添加 ReanimComponent
	// 普通僵尸：使用白名单方式，只显示基础身体部件和右手（anim_innerarm）
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "Zombie",
		Reanim:     reanimXML,
		PartImages: partImages,
		VisibleTracks: map[string]bool{
			// 基础身体部件
			"Zombie_body":           true,
			"Zombie_neck":           true,
			"Zombie_outerarm_upper": true,
			"Zombie_outerarm_lower": true,
			"Zombie_outerarm_hand":  true,
			"Zombie_outerleg_upper": true,
			"Zombie_outerleg_lower": true,
			"Zombie_outerleg_foot":  true,
			"Zombie_innerleg_upper": true,
			"Zombie_innerleg_lower": true,
			"Zombie_innerleg_foot":  true,
			// 头部
			"anim_head1": true, // 头部
			"anim_head2": true, // 下巴
			// 右手（内侧手臂）
			"anim_innerarm1": true, // 内侧手臂上部
			"anim_innerarm2": true, // 内侧手臂下部
			"anim_innerarm3": true, // 内侧手
		},
		// Story 6.3: 部件组配置（数据驱动，业务系统通过语义接口操作）
		// 这样 BehaviorSystem 只需要调用 HidePartGroup("arm") 而不需要知道具体轨道名
		PartGroups: map[string][]string{
			"arm": { // 左手（外侧手臂）- 受伤时会掉落
				"Zombie_outerarm_hand",
				"Zombie_outerarm_upper",
				"Zombie_outerarm_lower",
			},
			"head": { // 头部 - 死亡时会掉落
				"anim_head1",
				"anim_head2",
			},
		},
	})

	// Story 8.3: 使用 ReanimSystem 初始化动画（播放 idle 动画，等待激活）
	// 激活后会由 WaveSpawnSystem.ActivateWave() 切换为 walk 动画
	if err := rs.PlayAnimation(entityID, "anim_idle"); err != nil {
		return 0, fmt.Errorf("failed to play Zombie idle animation: %w", err)
	}

	// 添加速度组件（初始速度为0，待命状态）
	// Story 8.3: 僵尸在预生成时不移动，等待 WaveSpawnSystem.ActivateWave() 激活
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: 0.0, // 待命状态：不向左移动
		VY: 0.0, // 待命状态：不垂直移动
	})

	// 添加行为组件（标识为普通僵尸，初始为 idle 状态）
	// Story 8.3: 僵尸初始为静止状态，等待 WaveSpawnSystem.ActivateWave() 激活后切换为 Walking
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type:            components.BehaviorZombieBasic,
		ZombieAnimState: components.ZombieAnimIdle,
	})

	// 添加生命值组件（本Story定义但不使用，为Story 4.4准备）
	em.AddComponent(entityID, &components.HealthComponent{
		CurrentHealth: config.ZombieDefaultHealth,
		MaxHealth:     config.ZombieDefaultHealth,
	})

	// 添加碰撞组件（用于检测子弹碰撞）
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.ZombieCollisionWidth,
		Height: config.ZombieCollisionHeight,
	})

	return entityID, nil
}

// NewConeheadZombieEntity 创建路障僵尸实体
// 路障僵尸拥有370点护甲值，总生命值为640（护甲370+身体270）
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载僵尸 Reanim 资源）
//   - rs: Reanim 系统（用于初始化动画）
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置（通常在屏幕右侧外）
//
// 返回:
//   - ecs.EntityID: 创建的路障僵尸实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewConeheadZombieEntity(em *ecs.EntityManager, rm ResourceLoader, rs ReanimSystemInterface, row int, spawnX float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 计算僵尸Y坐标（世界坐标，基于行）
	// 行中心 = GridWorldStartY + row*CellHeight + CellHeight/2.0
	// 使用 config.ZombieVerticalOffset 以便手工调整
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	})

	// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
	// 从 ResourceManager 获取僵尸的 Reanim 数据和部件图片
	// 注意：路障僵尸使用基础僵尸的动画
	reanimXML := rm.GetReanimXML("Zombie")
	partImages := rm.GetReanimPartImages("Zombie")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load Zombie Reanim resources for Conehead")
	}

	// 添加 ReanimComponent
	// 路障僵尸：基础部件 + 路障
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "Zombie",
		Reanim:     reanimXML,
		PartImages: partImages,
		VisibleTracks: map[string]bool{
			// 基础身体部件
			"Zombie_body":           true,
			"Zombie_neck":           true,
			"Zombie_outerarm_upper": true,
			"Zombie_outerarm_lower": true,
			"Zombie_outerarm_hand":  true,
			"Zombie_outerleg_upper": true,
			"Zombie_outerleg_lower": true,
			"Zombie_outerleg_foot":  true,
			"Zombie_innerleg_upper": true,
			"Zombie_innerleg_lower": true,
			"Zombie_innerleg_foot":  true,
			// 头部
			"anim_head1": true, // 头部
			"anim_head2": true, // 下巴
			// 右手（内侧手臂）
			"anim_innerarm1": true,
			"anim_innerarm2": true,
			"anim_innerarm3": true,
			// 路障
			"anim_cone": true,
		},
		// Story 6.3: 部件组配置（数据驱动）
		PartGroups: map[string][]string{
			"arm": { // 左手（外侧手臂）
				"Zombie_outerarm_hand",
				"Zombie_outerarm_upper",
				"Zombie_outerarm_lower",
			},
			"head": { // 头部
				"anim_head1",
				"anim_head2",
			},
			"armor": { // 护甲（路障）
				"anim_cone",
			},
		},
	})

	// Story 8.3: 使用 ReanimSystem 初始化动画（播放 idle 动画，等待激活）
	// 激活后会由 WaveSpawnSystem.ActivateWave() 切换为 walk 动画
	if err := rs.PlayAnimation(entityID, "anim_idle"); err != nil {
		return 0, fmt.Errorf("failed to play ZombieConeHead idle animation: %w", err)
	}

	// 添加速度组件（初始速度为0，待命状态）
	// Story 8.3: 僵尸在预生成时不移动，等待 WaveSpawnSystem.ActivateWave() 激活
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: 0.0, // 待命状态：不向左移动
		VY: 0.0, // 待命状态：不垂直移动
	})

	// 添加行为组件（标识为路障僵尸，初始为 idle 状态）
	// Story 8.3: 僵尸初始为静止状态，等待 WaveSpawnSystem.ActivateWave() 激活后切换为 Walking
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type:            components.BehaviorZombieConehead,
		ZombieAnimState: components.ZombieAnimIdle,
	})

	// 添加护甲组件（路障僵尸的关键特性）
	em.AddComponent(entityID, &components.ArmorComponent{
		CurrentArmor: config.ConeheadZombieArmorHealth,
		MaxArmor:     config.ConeheadZombieArmorHealth,
	})

	// 添加生命值组件（身体生命值270）
	em.AddComponent(entityID, &components.HealthComponent{
		CurrentHealth: config.ZombieDefaultHealth,
		MaxHealth:     config.ZombieDefaultHealth,
	})

	// 添加碰撞组件（用于检测子弹碰撞）
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.ZombieCollisionWidth,
		Height: config.ZombieCollisionHeight,
	})

	return entityID, nil
}

// NewBucketheadZombieEntity 创建铁桶僵尸实体
// 铁桶僵尸拥有1100点护甲值，总生命值为1370（护甲1100+身体270）
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载僵尸 Reanim 资源）
//   - rs: Reanim 系统（用于初始化动画）
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置（通常在屏幕右侧外）
//
// 返回:
//   - ecs.EntityID: 创建的铁桶僵尸实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewBucketheadZombieEntity(em *ecs.EntityManager, rm ResourceLoader, rs ReanimSystemInterface, row int, spawnX float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 计算僵尸Y坐标（世界坐标，基于行）
	// 行中心 = GridWorldStartY + row*CellHeight + CellHeight/2.0
	// 使用 config.ZombieVerticalOffset 以便手工调整
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	})

	// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
	// 从 ResourceManager 获取僵尸的 Reanim 数据和部件图片
	// 注意：铁桶僵尸使用基础僵尸的动画
	reanimXML := rm.GetReanimXML("Zombie")
	partImages := rm.GetReanimPartImages("Zombie")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load Zombie Reanim resources for Buckethead")
	}

	// 添加 ReanimComponent
	// 铁桶僵尸：基础部件 + 铁桶
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "Zombie",
		Reanim:     reanimXML,
		PartImages: partImages,
		VisibleTracks: map[string]bool{
			// 基础身体部件
			"Zombie_body":           true,
			"Zombie_neck":           true,
			"Zombie_outerarm_upper": true,
			"Zombie_outerarm_lower": true,
			"Zombie_outerarm_hand":  true,
			"Zombie_outerleg_upper": true,
			"Zombie_outerleg_lower": true,
			"Zombie_outerleg_foot":  true,
			"Zombie_innerleg_upper": true,
			"Zombie_innerleg_lower": true,
			"Zombie_innerleg_foot":  true,
			// 头部
			"anim_head1": true, // 头部
			"anim_head2": true, // 下巴
			// 右手（内侧手臂）
			"anim_innerarm1": true,
			"anim_innerarm2": true,
			"anim_innerarm3": true,
			// 铁桶
			"anim_bucket": true,
		},
		// Story 6.3: 部件组配置（数据驱动）
		PartGroups: map[string][]string{
			"arm": { // 左手（外侧手臂）
				"Zombie_outerarm_hand",
				"Zombie_outerarm_upper",
				"Zombie_outerarm_lower",
			},
			"head": { // 头部
				"anim_head1",
				"anim_head2",
			},
			"armor": { // 护甲（铁桶）
				"anim_bucket",
			},
		},
	})

	// Story 8.3: 使用 ReanimSystem 初始化动画（播放 idle 动画，等待激活）
	// 激活后会由 WaveSpawnSystem.ActivateWave() 切换为 walk 动画
	if err := rs.PlayAnimation(entityID, "anim_idle"); err != nil {
		return 0, fmt.Errorf("failed to play ZombieBucketHead idle animation: %w", err)
	}

	// 添加速度组件（初始速度为0，待命状态）
	// Story 8.3: 僵尸在预生成时不移动，等待 WaveSpawnSystem.ActivateWave() 激活
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: 0.0, // 待命状态：不向左移动
		VY: 0.0, // 待命状态：不垂直移动
	})

	// 添加行为组件（标识为铁桶僵尸，初始为 idle 状态）
	// Story 8.3: 僵尸初始为静止状态，等待 WaveSpawnSystem.ActivateWave() 激活后切换为 Walking
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type:            components.BehaviorZombieBuckethead,
		ZombieAnimState: components.ZombieAnimIdle,
	})

	// 添加护甲组件（铁桶僵尸的关键特性）
	em.AddComponent(entityID, &components.ArmorComponent{
		CurrentArmor: config.BucketheadZombieArmorHealth,
		MaxArmor:     config.BucketheadZombieArmorHealth,
	})

	// 添加生命值组件（身体生命值270）
	em.AddComponent(entityID, &components.HealthComponent{
		CurrentHealth: config.ZombieDefaultHealth,
		MaxHealth:     config.ZombieDefaultHealth,
	})

	// 添加碰撞组件（用于检测子弹碰撞）
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.ZombieCollisionWidth,
		Height: config.ZombieCollisionHeight,
	})

	return entityID, nil
}
