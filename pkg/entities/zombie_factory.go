package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// NewZombieEntity 创建普通僵尸实体
// 僵尸从屏幕右侧外生成，以恒定速度从右向左移动
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载僵尸动画帧）
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置（通常在屏幕右侧外）
//
// 返回:
//   - ecs.EntityID: 创建的僵尸实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewZombieEntity(em *ecs.EntityManager, rm *game.ResourceManager, row int, spawnX float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 计算僵尸Y坐标（世界坐标，基于行）
	// 使用和植物相同的Y坐标计算，确保同一行的实体在同一高度
	// 使用 config.ZombieVerticalOffset 以便手工调整
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + config.ZombieVerticalOffset

	// 加载僵尸走路动画帧
	frames := make([]*ebiten.Image, config.ZombieWalkAnimationFrames)
	for i := 0; i < config.ZombieWalkAnimationFrames; i++ {
		framePath := fmt.Sprintf("assets/images/Zombies/Zombie/Zombie_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			return 0, fmt.Errorf("failed to load zombie animation frame %d: %w", i+1, err)
		}
		frames[i] = frameImage
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	})

	// 添加精灵组件（初始化为第一帧）
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: frames[0],
	})

	// 添加动画组件（走路动画，循环播放）
	em.AddComponent(entityID, &components.AnimationComponent{
		Frames:       frames,
		FrameSpeed:   config.ZombieWalkFrameSpeed, // 0.1秒/帧，完整动画约2.2秒
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,  // 循环播放走路动画
		IsFinished:   false, // 动画一直播放
	})

	// 添加速度组件（从右向左移动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed, // 负值表示向左移动
		VY: 0.0,
	})

	// 添加行为组件（标识为普通僵尸）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
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
//   - rm: 资源管理器（用于加载僵尸动画帧）
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置（通常在屏幕右侧外）
//
// 返回:
//   - ecs.EntityID: 创建的路障僵尸实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewConeheadZombieEntity(em *ecs.EntityManager, rm *game.ResourceManager, row int, spawnX float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 计算僵尸Y坐标（世界坐标，基于行）
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + config.ZombieVerticalOffset

	// 加载路障僵尸走路动画帧
	frames, err := utils.LoadConeheadZombieWalkAnimation(rm)
	if err != nil {
		return 0, fmt.Errorf("failed to load conehead zombie walk animation: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	})

	// 添加精灵组件（初始化为第一帧）
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: frames[0],
	})

	// 添加动画组件（走路动画，循环播放）
	em.AddComponent(entityID, &components.AnimationComponent{
		Frames:       frames,
		FrameSpeed:   config.ZombieWalkFrameSpeed, // 0.1秒/帧
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,  // 循环播放走路动画
		IsFinished:   false, // 动画一直播放
	})

	// 添加速度组件（从右向左移动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed, // 负值表示向左移动
		VY: 0.0,
	})

	// 添加行为组件（标识为路障僵尸）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorZombieConehead,
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
//   - rm: 资源管理器（用于加载僵尸动画帧）
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置（通常在屏幕右侧外）
//
// 返回:
//   - ecs.EntityID: 创建的铁桶僵尸实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewBucketheadZombieEntity(em *ecs.EntityManager, rm *game.ResourceManager, row int, spawnX float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 计算僵尸Y坐标（世界坐标，基于行）
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + config.ZombieVerticalOffset

	// 加载铁桶僵尸走路动画帧
	frames, err := utils.LoadBucketheadZombieWalkAnimation(rm)
	if err != nil {
		return 0, fmt.Errorf("failed to load buckethead zombie walk animation: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	})

	// 添加精灵组件（初始化为第一帧）
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: frames[0],
	})

	// 添加动画组件（走路动画，循环播放）
	em.AddComponent(entityID, &components.AnimationComponent{
		Frames:       frames,
		FrameSpeed:   config.ZombieWalkFrameSpeed, // 0.1秒/帧
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,  // 循环播放走路动画
		IsFinished:   false, // 动画一直播放
	})

	// 添加速度组件（从右向左移动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed, // 负值表示向左移动
		VY: 0.0,
	})

	// 添加行为组件（标识为铁桶僵尸）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBuckethead,
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
