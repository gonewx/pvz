package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/types"
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
// Story 14.3: Epic 14 - 移除 ReanimSystem 依赖，动画通过 AnimationCommand 组件初始化
func NewZombieEntity(em *ecs.EntityManager, rm ResourceLoader, row int, spawnX float64) (ecs.EntityID, error) {
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

	// 添加基础 ReanimComponent
	// LastGroundX/Y 初始化为 0.0，用于根运动计算
	// LastAnimFrame 初始化为 -1，表示尚未开始动画
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName:        "Zombie",
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		LastGroundX:       0.0,
		LastGroundY:       0.0,
		LastAnimFrame:     -1,
		AccumulatedDeltaX: 0.0,
		AccumulatedDeltaY: 0.0,
	})

	// ✅ Epic 14: 使用 AnimationCommand 触发动画（替代直接调用 ReanimSystem）
	// 添加动画命令组件，让 ReanimSystem 在 Update 中处理
	// Story 17.10: 使用配置驱动的 ComboName 而不是直接指定 AnimationName
	// 这样可以确保正确应用 hidden_tracks（例如隐藏路障/铁桶）
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombie,
		ComboName: "idle",
		Processed: false,
	})

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

	// Story 10.7: 为僵尸添加阴影组件
	shadowSize := config.GetShadowSize("zombie")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
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
//
// Story 14.3: Epic 14 - 移除 ReanimSystem 依赖，动画通过 AnimationCommand 组件初始化
func NewConeheadZombieEntity(em *ecs.EntityManager, rm ResourceLoader, row int, spawnX float64) (ecs.EntityID, error) {
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

	// 添加 ReanimComponent（路障僵尸：基础部件 + 路障）
	// Story 13.7: 使用配置驱动，不再硬编码 VisibleTracks
	// LastGroundX/Y 初始化为 0.0，用于根运动计算
	// LastAnimFrame 初始化为 -1，表示尚未开始动画
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName:        "Zombie",
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		LastGroundX:       0.0,
		LastGroundY:       0.0,
		LastAnimFrame:     -1,
		AccumulatedDeltaX: 0.0,
		AccumulatedDeltaY: 0.0,
	})

	// ✅ Epic 14: 使用 AnimationCommand 触发动画（替代直接调用 ReanimSystem）
	// 添加动画命令组件，让 ReanimSystem 在 Update 中处理
	// Story 17.10: 使用配置驱动的 ComboName 而不是直接指定 AnimationName
	// 这样可以确保正确显示路障（不被隐藏）同时隐藏其他装备
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombieConehead,
		ComboName: "idle",
		Processed: false,
	})

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

	// Story 10.7: 为路障僵尸添加阴影组件
	shadowSize := config.GetShadowSize("zombie_cone")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
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
//
// Story 14.3: Epic 14 - 移除 ReanimSystem 依赖，动画通过 AnimationCommand 组件初始化
func NewBucketheadZombieEntity(em *ecs.EntityManager, rm ResourceLoader, row int, spawnX float64) (ecs.EntityID, error) {
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

	// 添加 ReanimComponent（铁桶僵尸：基础部件 + 铁桶）
	// Story 13.7: 使用配置驱动，不再硬编码 VisibleTracks
	// LastGroundX/Y 初始化为 0.0，用于根运动计算
	// LastAnimFrame 初始化为 -1，表示尚未开始动画
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName:        "Zombie",
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		LastGroundX:       0.0,
		LastGroundY:       0.0,
		LastAnimFrame:     -1,
		AccumulatedDeltaX: 0.0,
		AccumulatedDeltaY: 0.0,
	})

	// ✅ Epic 14: 使用 AnimationCommand 触发动画（替代直接调用 ReanimSystem）
	// 添加动画命令组件，让 ReanimSystem 在 Update 中处理
	// Story 17.10: 使用配置驱动的 ComboName 而不是直接指定 AnimationName
	// 这样可以确保正确显示铁桶（不被隐藏）同时隐藏其他装备
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombieBuckethead,
		ComboName: "idle",
		Processed: false,
	})

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

	// Story 10.7: 为铁桶僵尸添加阴影组件
	shadowSize := config.GetShadowSize("zombie_bucket")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
	})

	return entityID, nil
}
