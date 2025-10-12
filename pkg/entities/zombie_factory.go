package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// ZombieVerticalOffset 僵尸在格子中的垂直偏移量
	// 使僵尸在格子中央偏下的位置，视觉效果更自然
	ZombieVerticalOffset = 30.0

	// ZombieWalkAnimationFrames 普通僵尸走路动画的总帧数
	ZombieWalkAnimationFrames = 22

	// ZombieWalkFrameSpeed 僵尸走路动画的帧速率（秒/帧）
	ZombieWalkFrameSpeed = 0.1

	// ZombieWalkSpeed 普通僵尸的移动速度（像素/秒）
	// 负值表示从右向左移动
	ZombieWalkSpeed = -30.0

	// ZombieDefaultHealth 普通僵尸的默认生命值
	ZombieDefaultHealth = 270
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
	// 僵尸应该在格子中央偏下的位置
	spawnY := config.GridWorldStartY + float64(row)*config.CellHeight + ZombieVerticalOffset

	// 加载僵尸走路动画帧
	frames := make([]*ebiten.Image, ZombieWalkAnimationFrames)
	for i := 0; i < ZombieWalkAnimationFrames; i++ {
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
		FrameSpeed:   ZombieWalkFrameSpeed, // 0.1秒/帧，完整动画约2.2秒
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,  // 循环播放走路动画
		IsFinished:   false, // 动画一直播放
	})

	// 添加速度组件（从右向左移动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: ZombieWalkSpeed, // 负值表示向左移动
		VY: 0.0,
	})

	// 添加行为组件（标识为普通僵尸）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})

	// 添加生命值组件（本Story定义但不使用，为Story 4.4准备）
	em.AddComponent(entityID, &components.HealthComponent{
		CurrentHealth: ZombieDefaultHealth,
		MaxHealth:     ZombieDefaultHealth,
	})

	return entityID, nil
}
