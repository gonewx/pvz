package systems

import (
	"log"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// AnimationSystem 管理所有实体的帧动画
type AnimationSystem struct {
	entityManager *ecs.EntityManager
}

// NewAnimationSystem 创建一个新的动画系统
func NewAnimationSystem(em *ecs.EntityManager) *AnimationSystem {
	return &AnimationSystem{
		entityManager: em,
	}
}

// Update 更新所有动画实体的帧
func (s *AnimationSystem) Update(deltaTime float64) {
	// 查询所有拥有 AnimationComponent 和 SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.AnimationComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	for _, id := range entities {
		// 获取组件
		animComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.AnimationComponent{}))
		spriteComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))

		// 类型断言
		anim := animComp.(*components.AnimationComponent)
		sprite := spriteComp.(*components.SpriteComponent)

		// 如果动画已完成且非循环,跳过
		if anim.IsFinished {
			continue
		}

		// 调试：检测到动画开始播放
		if anim.CurrentFrame == 0 && anim.FrameCounter == 0 {
			log.Printf("[AnimationSystem] 开始播放动画 (实体ID: %d, 帧数: %d, 循环: %v)", id, len(anim.Frames), anim.IsLooping)
		}

		// 如果没有帧,跳过
		if len(anim.Frames) == 0 {
			continue
		}

		// 增加帧计时器
		anim.FrameCounter += deltaTime

		// 检查是否需要切换到下一帧
		if anim.FrameCounter >= anim.FrameSpeed {
			// 重置计时器
			anim.FrameCounter = 0

			// 前进到下一帧
			anim.CurrentFrame++

			// 检查是否超出帧数
			if anim.CurrentFrame >= len(anim.Frames) {
				if anim.IsLooping {
					// 循环动画: 重置到第0帧
					anim.CurrentFrame = 0
				} else {
					// 非循环动画: 停在最后一帧并标记完成
					anim.CurrentFrame = len(anim.Frames) - 1
					anim.IsFinished = true
					log.Printf("[AnimationSystem] 动画播放完成 (实体ID: %d)，停在最后一帧", id)
				}
			}

			// 更新 SpriteComponent 的图像为当前帧
			sprite.Image = anim.Frames[anim.CurrentFrame]
		}
	}
}
