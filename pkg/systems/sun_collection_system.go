package systems

import (
	"log"
	"math"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// SunCollectionSystem 管理阳光收集动画的完成检测
// 检查正在收集的阳光是否到达目标位置，并在到达时增加阳光数值并删除实体
type SunCollectionSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState // 游戏状态（用于增加阳光数值）
	targetX       float64         // 阳光计数器X坐标
	targetY       float64         // 阳光计数器Y坐标
}

// NewSunCollectionSystem 创建一个新的阳光收集系统
func NewSunCollectionSystem(em *ecs.EntityManager, gs *game.GameState, targetX, targetY float64) *SunCollectionSystem {
	return &SunCollectionSystem{
		entityManager: em,
		gameState:     gs,
		targetX:       targetX,
		targetY:       targetY,
	}
}

// Update 检查所有正在收集的阳光是否到达目标位置
func (s *SunCollectionSystem) Update(deltaTime float64) {
	// 查询所有正在收集的阳光实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	for _, id := range entities {
		// 获取组件
		sunComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
		posComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))

		// 类型断言
		sun := sunComp.(*components.SunComponent)
		pos := posComp.(*components.PositionComponent)

		// 只处理正在收集的阳光
		if sun.State != components.SunCollecting {
			continue
		}

		// 计算到目标位置的距离
		dx := s.targetX - pos.X
		dy := s.targetY - pos.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		// 如果距离小于阈值（10像素），认为已到达
		if distance < 10.0 {
			// 增加阳光数值（在阳光到达时才增加，而非点击时）
			// 自然掉落的阳光固定为 25 点
			oldSun := s.gameState.GetSun()
			s.gameState.AddSun(25)
			log.Printf("[SunCollectionSystem] 阳光到达目标! 阳光数量: %d -> %d, 删除实体", oldSun, s.gameState.GetSun())

			// 删除阳光实体
			s.entityManager.DestroyEntity(id)
		}
	}
}
