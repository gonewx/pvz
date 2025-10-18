package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// 僵尸行转换常量
const (
	// ZombieAttackStartX 僵尸开始进攻的X坐标阈值
	// 僵尸必须在到达此X坐标之前完成行转换
	ZombieAttackStartX = 900.0

	// LaneTransitionTolerance 行转换完成的Y坐标容差（像素）
	LaneTransitionTolerance = 5.0
)

// ZombieLaneTransitionSystem 僵尸行转换系统
//
// 职责：
//   - 处理僵尸从非有效行移动到目标有效行的逻辑
//   - 监控僵尸是否已到达目标行
//   - 到达目标行后，启动僵尸的正常移动（向左移动）
//
// 工作流程：
//  1. 检测所有带 ZombieTargetLaneComponent 且未到达目标行的僵尸
//  2. 更新僵尸位置，使其向目标行移动（通过 VelocityComponent.VY）
//  3. 检查僵尸是否到达目标行（Y坐标在容差范围内）
//  4. 到达后：标记 HasReachedTargetLane = true，清除Y轴速度，启动正常移动
type ZombieLaneTransitionSystem struct {
	entityManager *ecs.EntityManager
}

// NewZombieLaneTransitionSystem 创建僵尸行转换系统
func NewZombieLaneTransitionSystem(em *ecs.EntityManager) *ZombieLaneTransitionSystem {
	return &ZombieLaneTransitionSystem{
		entityManager: em,
	}
}

// Update 更新僵尸行转换系统
//
// 参数：
//
//	deltaTime - 自上一帧以来经过的时间（秒）
func (s *ZombieLaneTransitionSystem) Update(deltaTime float64) {
	// 查询所有带 ZombieTargetLaneComponent 和 PositionComponent 的僵尸
	entities := ecs.GetEntitiesWith2[
		*components.ZombieTargetLaneComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range entities {
		targetLaneComp, ok := ecs.GetComponent[*components.ZombieTargetLaneComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 如果已到达目标行，跳过
		if targetLaneComp.HasReachedTargetLane {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 计算目标行的中心Y坐标
		targetY := config.GridWorldStartY + float64(targetLaneComp.TargetRow)*config.CellHeight + config.CellHeight/2.0

		// 检查是否已到达目标行（Y坐标在容差范围内）
		deltaY := math.Abs(pos.Y - targetY)
		if deltaY <= LaneTransitionTolerance {
			// 到达目标行
			s.onReachedTargetLane(entityID, targetLaneComp, pos, targetY)
			continue
		}

		// 还未到达，继续移动（速度已在 WaveSpawnSystem 中设置）
		// 这里只需要记录日志
		if deltaY > 50 { // 只在距离较远时记录日志，避免刷屏
			log.Printf("[ZombieLaneTransitionSystem] Zombie %d moving to target lane %d, current Y=%.2f, target Y=%.2f, delta=%.2f",
				entityID, targetLaneComp.TargetRow+1, pos.Y, targetY, deltaY)
		}
	}
}

// onReachedTargetLane 僵尸到达目标行时的处理
//
// 参数：
//
//	entityID - 僵尸实体ID
//	targetLaneComp - 目标行组件
//	pos - 位置组件
//	targetY - 目标行的中心Y坐标
func (s *ZombieLaneTransitionSystem) onReachedTargetLane(
	entityID ecs.EntityID,
	targetLaneComp *components.ZombieTargetLaneComponent,
	pos *components.PositionComponent,
	targetY float64,
) {
	log.Printf("[ZombieLaneTransitionSystem] Zombie %d reached target lane %d",
		entityID, targetLaneComp.TargetRow+1)

	// 标记已到达目标行
	targetLaneComp.HasReachedTargetLane = true

	// 精确对齐到目标行中心
	pos.Y = targetY

	// 清除Y轴速度，启动X轴移动（正常僵尸移动）
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
		vel.VY = 0

		// 如果僵尸还没有X轴速度（说明之前暂停移动等待到达目标行）
		// 启动向左移动
		if vel.VX == 0 {
			// 获取僵尸行为组件，确定僵尸类型
			if behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
				// 根据僵尸类型设置移动速度
				// 所有僵尸的基础速度都是 -23.0（向左移动）
				vel.VX = -23.0
				log.Printf("[ZombieLaneTransitionSystem] Started zombie %d normal movement, behavior=%d, VX=%.2f",
					entityID, behavior.Type, vel.VX)
			}
		}
	}

	// 可选：移除 ZombieTargetLaneComponent 以节省内存（已完成任务）
	// ecs.RemoveComponent[*components.ZombieTargetLaneComponent](s.entityManager, entityID)
}
