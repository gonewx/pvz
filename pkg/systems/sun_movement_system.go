package systems

import (
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

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
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.VelocityComponent{}),
	)

	for _, id := range entities {
		// 获取组件
		sunComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
		posComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
		velComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.VelocityComponent{}))

		// 类型断言
		sun := sunComp.(*components.SunComponent)
		pos := posComp.(*components.PositionComponent)
		vel := velComp.(*components.VelocityComponent)

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

		case components.SunLanded:
			// 已落地:保持静止(不移动)
			// 无需任何操作

		case components.SunCollecting:
			// 正在被收集:由 Story 2.4 实现收集动画
			// 本故事暂不处理
		}
	}
}




