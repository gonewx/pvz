package systems

import (
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// PhysicsSystem 处理游戏物理逻辑
// 主要负责碰撞检测（子弹与僵尸的碰撞）
type PhysicsSystem struct {
	em *ecs.EntityManager
	rm *game.ResourceManager
}

// NewPhysicsSystem 创建物理系统
//
// 参数:
//   - em: 实体管理器，用于查询和操作实体组件
//   - rm: 资源管理器，用于创建击中效果时加载资源
//
// 返回:
//   - *PhysicsSystem: 物理系统实例
func NewPhysicsSystem(em *ecs.EntityManager, rm *game.ResourceManager) *PhysicsSystem {
	return &PhysicsSystem{
		em: em,
		rm: rm,
	}
}

// checkAABBCollision 检查两个实体的AABB（轴对齐边界框）是否发生碰撞
// 假设碰撞盒中心对齐实体位置
//
// 参数:
//   - pos1: 第一个实体的位置组件
//   - col1: 第一个实体的碰撞组件
//   - pos2: 第二个实体的位置组件
//   - col2: 第二个实体的碰撞组件
//
// 返回:
//   - bool: 如果两个碰撞盒重叠返回 true，否则返回 false
func (ps *PhysicsSystem) checkAABBCollision(
	pos1 *components.PositionComponent, col1 *components.CollisionComponent,
	pos2 *components.PositionComponent, col2 *components.CollisionComponent) bool {

	// 计算第一个碰撞盒的边界（中心对齐）
	left1 := pos1.X - col1.Width/2
	right1 := pos1.X + col1.Width/2
	top1 := pos1.Y - col1.Height/2
	bottom1 := pos1.Y + col1.Height/2

	// 计算第二个碰撞盒的边界（中心对齐）
	left2 := pos2.X - col2.Width/2
	right2 := pos2.X + col2.Width/2
	top2 := pos2.Y - col2.Height/2
	bottom2 := pos2.Y + col2.Height/2

	// AABB碰撞检测：检查两个矩形是否重叠
	// 如果任一轴上没有重叠，则没有碰撞
	return right1 >= left2 &&
		left1 <= right2 &&
		bottom1 >= top2 &&
		top1 <= bottom2
}

// Update 更新物理系统，处理碰撞检测
// 检测所有豌豆子弹与僵尸的碰撞
//
// 参数:
//   - deltaTime: 自上一帧以来经过的时间（秒），本系统暂不使用
func (ps *PhysicsSystem) Update(deltaTime float64) {
	// 查询所有拥有必要组件的实体（子弹和僵尸都需要这些组件）
	allEntities := ps.em.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.CollisionComponent{}),
	)

	// 分离子弹和僵尸
	bullets := make([]ecs.EntityID, 0)
	zombies := make([]ecs.EntityID, 0)

	for _, entityID := range allEntities {
		behaviorComp, ok := ps.em.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		if !ok {
			continue
		}
		behavior := behaviorComp.(*components.BehaviorComponent)

		if behavior.Type == components.BehaviorPeaProjectile {
			bullets = append(bullets, entityID)
		} else if behavior.Type == components.BehaviorZombieBasic {
			zombies = append(zombies, entityID)
		}
	}

	// 嵌套遍历检测碰撞
	for _, bulletID := range bullets {
		// 获取子弹的位置和碰撞组件
		bulletPosComp, ok := ps.em.GetComponent(bulletID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			continue
		}
		bulletPos := bulletPosComp.(*components.PositionComponent)

		bulletColComp, ok := ps.em.GetComponent(bulletID, reflect.TypeOf(&components.CollisionComponent{}))
		if !ok {
			continue
		}
		bulletCol := bulletColComp.(*components.CollisionComponent)

		// 检查子弹与所有僵尸的碰撞
		for _, zombieID := range zombies {
			// 获取僵尸的位置和碰撞组件
			zombiePosComp, ok := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				continue
			}
			zombiePos := zombiePosComp.(*components.PositionComponent)

			zombieColComp, ok := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.CollisionComponent{}))
			if !ok {
				continue
			}
			zombieCol := zombieColComp.(*components.CollisionComponent)

			// 执行AABB碰撞检测
			if ps.checkAABBCollision(bulletPos, bulletCol, zombiePos, zombieCol) {
				// 碰撞发生！
				// 1. 创建击中效果实体（在子弹位置）
				_, err := entities.NewPeaBulletHitEffect(ps.em, ps.rm, bulletPos.X, bulletPos.Y)
				if err != nil {
					// 如果创建击中效果失败，记录错误但继续处理碰撞
					// 在实际项目中可以使用日志系统记录错误
					// 这里为了简化，忽略错误
				}

				// 2. 标记子弹实体待删除
				ps.em.DestroyEntity(bulletID)

				// 3. 一个子弹只能击中一个僵尸，跳出内层循环
				// （注意：不减少僵尸生命值，伤害逻辑在 Story 4.4 中实现）
				break
			}
		}
	}
}
