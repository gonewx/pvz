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

	// DEBUG: 记录系统是否被调用
	if len(entities) > 0 {
		log.Printf("[ZombieLaneTransitionSystem] Update called, processing %d zombies", len(entities))
	}

	for _, entityID := range entities {
		// Story 8.3: 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
		if waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID); ok {
			if !waveState.IsActivated {
				// 僵尸未激活，跳过行转换逻辑（保持静止展示）
				log.Printf("[ZombieLaneTransitionSystem] Zombie %d NOT activated, skipping transition", entityID)
				continue
			}
		}

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
		// 必须与僵尸工厂函数的Y坐标计算公式保持一致
		targetY := config.GridWorldStartY + float64(targetLaneComp.TargetRow)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset

		// Story 8.7: 根据转换模式选择不同的处理逻辑
		switch targetLaneComp.TransitionMode {
		case components.TransitionModeInstant:
			// 瞬间调整模式：立即设置Y坐标
			s.handleInstantTransition(entityID, targetLaneComp, pos, targetY)

		case components.TransitionModeGradual:
			// 渐变动画模式：保留原有逻辑
			s.handleGradualTransition(entityID, targetLaneComp, pos, targetY)

		default:
			// 默认使用瞬间模式（向后兼容）
			s.handleInstantTransition(entityID, targetLaneComp, pos, targetY)
		}
	}
}

// handleGradualTransition 处理渐变行转换模式
//
// Story 8.7: 此方法完全保留 Story 8.3 的原有渐变动画逻辑
//
// 僵尸通过Y轴速度平滑移动到目标行（约3秒）
//
// 参数：
//
//	entityID - 僵尸实体ID
//	targetLaneComp - 目标行组件
//	pos - 位置组件
//	targetY - 目标行的中心Y坐标
func (s *ZombieLaneTransitionSystem) handleGradualTransition(
	entityID ecs.EntityID,
	targetLaneComp *components.ZombieTargetLaneComponent,
	pos *components.PositionComponent,
	targetY float64,
) {
	// Story 8.3 原逻辑：如果僵尸还没有Y轴速度（刚激活），计算并设置Y轴速度
	vel, hasVel := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
	if hasVel && vel.VY == 0 && math.Abs(pos.Y-targetY) > LaneTransitionTolerance {
		// 计算到达目标行所需的Y轴速度
		// 假设需要在3秒内到达目标行
		deltaY := targetY - pos.Y
		vySpeed := deltaY / 3.0
		vel.VY = vySpeed
		log.Printf("[ZombieLaneTransitionSystem] Initialized zombie %d VY speed: %.2f (mode=gradual, deltaY=%.2f)",
			entityID, vySpeed, deltaY)
	}

	// 检查是否已到达目标行（Y坐标在容差范围内）
	deltaY := math.Abs(pos.Y - targetY)
	if deltaY <= LaneTransitionTolerance {
		// 到达目标行
		s.onReachedTargetLane(entityID, targetLaneComp, pos, targetY)
		return
	}

	// 还未到达，继续移动
	// 这里只需要记录日志
	if deltaY > 50 { // 只在距离较远时记录日志，避免刷屏
		log.Printf("[ZombieLaneTransitionSystem] Zombie %d moving to target lane %d (mode=gradual), current Y=%.2f, target Y=%.2f, delta=%.2f",
			entityID, targetLaneComp.TargetRow+1, pos.Y, targetY, deltaY)
	}
}

// handleInstantTransition 处理瞬间行转换模式
//
// Story 8.7: 僵尸立即调整Y坐标到目标行，无过渡动画
//
// 参数：
//
//	entityID - 僵尸实体ID
//	targetLaneComp - 目标行组件
//	pos - 位置组件
//	targetY - 目标行的中心Y坐标
func (s *ZombieLaneTransitionSystem) handleInstantTransition(
	entityID ecs.EntityID,
	targetLaneComp *components.ZombieTargetLaneComponent,
	pos *components.PositionComponent,
	targetY float64,
) {
	// 1. 瞬间调整Y坐标到目标行
	pos.Y = targetY

	log.Printf("[ZombieLaneTransitionSystem] Zombie %d instantly moved to target lane %d (mode=instant, Y=%.2f)",
		entityID, targetLaneComp.TargetRow+1, targetY)

	// 2. 标记已到达目标行
	targetLaneComp.HasReachedTargetLane = true

	// 3. 启动X轴移动（清除Y轴速度，设置X轴速度）
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
				log.Printf("[ZombieLaneTransitionSystem] Started zombie %d movement (VX=-23.0, behavior=%d)",
					entityID, behavior.Type)
			}
		}
	}

	// (可选) 移除 ZombieTargetLaneComponent 以节省内存（已完成任务）
	// ecs.RemoveComponent[*components.ZombieTargetLaneComponent](s.entityManager, entityID)
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
				log.Printf("[ZombieLaneTransitionSystem] ⚠️ SET VX=-23.0 for zombie %d (reached target lane), behavior=%d",
					entityID, behavior.Type)
			}
		}
	}

	// 可选：移除 ZombieTargetLaneComponent 以节省内存（已完成任务）
	// ecs.RemoveComponent[*components.ZombieTargetLaneComponent](s.entityManager, entityID)
}
