package systems

import (
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/utils"
)

// 重力加速度常量（像素/秒²）
const SunGravity = 200.0

// SunMovementSystem 管理阳光的移动逻辑
type SunMovementSystem struct {
	entityManager *ecs.EntityManager
}

// NewSunMovementSystem 创建一个新的阳光移动系统
func NewSunMovementSystem(em *ecs.EntityManager) *SunMovementSystem {
	return &SunMovementSystem{
		entityManager: em,
	}
}

// Update 更新所有阳光的位置
func (s *SunMovementSystem) Update(deltaTime float64) {
	// 查询所有拥有 SunComponent, PositionComponent, VelocityComponent 的实体
	entities := ecs.GetEntitiesWith3[
		*components.SunComponent,
		*components.PositionComponent,
		*components.VelocityComponent,
	](s.entityManager)

	for _, id := range entities {
		// 获取组件
		sun, ok := ecs.GetComponent[*components.SunComponent](s.entityManager, id)
		if !ok {
			continue
		}
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		if !ok {
			continue
		}
		vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, id)
		if !ok {
			continue
		}

		// 根据阳光状态处理移动
		switch sun.State {
		case components.SunFalling:
			// 下落中:更新位置
			pos.Y += vel.VY * deltaTime

			// 检查是否到达目标位置
			if pos.Y >= sun.TargetY {
				// 落地:设置为精确的目标位置
				pos.Y = sun.TargetY
				sun.State = components.SunLanded
				vel.VY = 0 // 停止移动
			}

		case components.SunRising:
			// 上升中（向日葵生产的阳光）：抛物线运动，受重力影响
			// 更新速度：重力加速度向下
			vel.VY += SunGravity * deltaTime // 重力向下（正方向）

			// 更新位置
			pos.X += vel.VX * deltaTime
			pos.Y += vel.VY * deltaTime

			// 检查是否到达或超过目标位置
			// 判断条件：Y坐标超过目标且速度向下（VY > 0）
			if pos.Y >= sun.TargetY && vel.VY > 0 {
				// 到达目标位置：设置为精确的目标位置
				pos.Y = sun.TargetY
				sun.State = components.SunLanded
				vel.VX = 0
				vel.VY = 0 // 停止移动
			}

		case components.SunLanded:
			// 已落地:保持静止(不移动)
			// 无需任何操作

		case components.SunCollecting:
			// 正在被收集：执行飞向阳光计数器的缓动动画
			// 检查是否有 SunCollectionAnimationComponent（新的缓动动画系统）
			animComp, hasAnimComp := ecs.GetComponent[*components.SunCollectionAnimationComponent](s.entityManager, id)
			if hasAnimComp {
				// 新的缓动动画系统
				// 1. 递增进度
				animComp.Progress += deltaTime / animComp.Duration
				if animComp.Progress > 1.0 {
					animComp.Progress = 1.0
				}

				// 2. 使用缓动函数计算缓动后的进度（EaseOutCubic：开始快，结束慢）
				easedProgress := utils.EaseOutCubic(animComp.Progress)

				// 3. 使用线性插值计算新位置
				pos.X = utils.Lerp(animComp.StartX, animComp.TargetX, easedProgress)
				pos.Y = utils.Lerp(animComp.StartY, animComp.TargetY, easedProgress)

				// 4. 更新缩放（从 1.0 渐变到 0.6，营造"被吸入"效果）
				scale, hasScale := ecs.GetComponent[*components.ScaleComponent](s.entityManager, id)
				if hasScale {
					// 缩放也使用缓动进度，确保与位置同步
					scale.ScaleX = utils.Lerp(1.0, 0.6, easedProgress)
					scale.ScaleY = utils.Lerp(1.0, 0.6, easedProgress)
				}
			} else {
				// 兼容旧代码：使用速度向量（如果没有缓动组件）
				pos.X += vel.VX * deltaTime
				pos.Y += vel.VY * deltaTime
			}
		}
	}
}
