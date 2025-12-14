package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/types"
	"github.com/gonewx/pvz/pkg/utils"
)

// PoleVaultSystem 撑杆僵尸跳跃系统
// Story 8.9: 处理撑杆僵尸的植物检测和跳跃逻辑
type PoleVaultSystem struct {
	entityManager *ecs.EntityManager
}

// NewPoleVaultSystem 创建新的撑杆僵尸跳跃系统
func NewPoleVaultSystem(em *ecs.EntityManager) *PoleVaultSystem {
	return &PoleVaultSystem{
		entityManager: em,
	}
}

// Update 更新撑杆僵尸的跳跃状态
func (s *PoleVaultSystem) Update(deltaTime float64) {
	// 获取所有拥有 PoleVaultComponent 的僵尸
	polevaulters := ecs.GetEntitiesWith2[*components.PoleVaultComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range polevaulters {
		poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 检查是否需要应用延迟的位移补偿
		// 这必须在动画切换后的下一帧执行，确保 body1 已重置
		if poleVault.NeedPositionCompensation {
			s.applyPositionCompensation(entityID, poleVault)
			continue
		}

		// 处理跳跃中的僵尸
		if poleVault.IsJumping {
			s.updateJumping(entityID, poleVault, deltaTime)
			continue
		}

		// 只有持杆的僵尸需要检测植物触发跳跃
		if !poleVault.HasPole {
			continue
		}

		// 检查僵尸是否正在移动（避免对静止的僵尸进行检测）
		vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
		if !ok || vel.VX >= 0 {
			log.Printf("[PoleVaultSystem] 僵尸 %d 无速度或未移动，跳过检测", entityID)
			continue
		}

		// 检测前方是否有植物
		s.checkPlantCollision(entityID, poleVault)
	}
}

// checkPlantCollision 检测撑杆僵尸前方是否有植物
func (s *PoleVaultSystem) checkPlantCollision(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent) {
	zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		return
	}

	zombieCol, ok := ecs.GetComponent[*components.CollisionComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[PoleVaultSystem] 僵尸 %d 无 CollisionComponent", zombieID)
		return
	}

	// 计算僵尸所在的行
	zombieRow := utils.GetEntityRow(zombiePos.Y, config.GridWorldStartY, config.CellHeight)

	// 获取所有植物实体
	plants := ecs.GetEntitiesWith2[*components.PlantComponent, *components.PositionComponent](s.entityManager)

	// 僵尸左边缘
	zombieLeftEdge := zombiePos.X - zombieCol.Width/2

	for _, plantID := range plants {
		plantPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, plantID)
		if !ok {
			continue
		}

		plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID)
		if !ok {
			continue
		}

		// 只检测同一行的植物
		if plant.GridRow != zombieRow {
			continue
		}

		// 植物右边缘
		plantRightEdge := plantPos.X + config.CellWidth/2

		// 检测距离阈值（当僵尸接近植物时触发跳跃）
		jumpTriggerDistance := 20.0

		log.Printf("[PoleVaultSystem] 检测: 僵尸%d(X=%.1f,左边缘=%.1f,行=%d) 植物%d(X=%.1f,右边缘=%.1f,行=%d)",
			zombieID, zombiePos.X, zombieLeftEdge, zombieRow,
			plantID, plantPos.X, plantRightEdge, plant.GridRow)

		if zombieLeftEdge <= plantRightEdge+jumpTriggerDistance && zombieLeftEdge > plantRightEdge-config.CellWidth {
			// 触发跳跃
			s.startJump(zombieID, poleVault, zombiePos.X, plantPos.X, plantID)
			return
		}
	}
}

// startJump 开始跳跃
func (s *PoleVaultSystem) startJump(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent, currentX, plantX float64, plantID ecs.EntityID) {
	log.Printf("[PoleVaultSystem] 撑杆僵尸 %d 检测到植物 %d，开始跳跃 (当前 X=%.1f)", zombieID, plantID, currentX)

	// 设置跳跃状态
	poleVault.IsJumping = true
	poleVault.TargetPlantEntityID = uint64(plantID)

	// 播放跳跃动画组合（使用 ComboName 以应用 loop: false 设置）
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombiePolevaulter,
		ComboName: "jump",
		Processed: false,
	})
}

// updateJumping 更新跳跃中的僵尸
// 跳跃期间位置保持不变，动画内部的 body1 位移制造跳跃视觉效果
// 动画结束后需要补偿位移，保持视觉连续
func (s *PoleVaultSystem) updateJumping(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent, deltaTime float64) {
	// 获取 Reanim 组件，检测动画是否播放完成
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID)
	if !ok {
		// 没有动画组件，直接结束跳跃
		s.finishJump(zombieID, poleVault)
		return
	}

	// 检测动画是否播放完成
	if reanim.IsFinished {
		s.finishJump(zombieID, poleVault)
		return
	}

	// 跳跃期间位置保持不变，等待动画完成
}

// finishJump 完成跳跃
func (s *PoleVaultSystem) finishJump(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent) {
	log.Printf("[PoleVaultSystem] 撑杆僵尸 %d 跳跃动画完成，准备应用位移补偿", zombieID)

	// 更新撑杆状态
	poleVault.HasPole = false
	poleVault.IsJumping = false

	// 标记需要位移补偿，延迟到下一帧应用
	// 这样可以确保动画切换（body1 重置）在位移补偿之前完成
	poleVault.NeedPositionCompensation = true

	// 恢复移动（使用普通速度）
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, zombieID); ok {
		vel.VX = config.PolevaulterZombieWalkSpeed
	}

	// 播放行走动画（AnimationCommandComponent 会被 ReanimSystem 在本帧或下一帧处理）
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombiePolevaulter,
		ComboName: "walk",
		Processed: false,
	})
}

// applyPositionCompensation 应用延迟的位移补偿
func (s *PoleVaultSystem) applyPositionCompensation(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent) {
	// 应用跳跃位移补偿
	// jump 动画期间，body1 轨道的 X 从 42.0 变化到 -109.2（视觉上向左移动）
	// 切换到 walk 动画时，body1.X 重置为 40.1（视觉上向右跳回）
	// 为保持视觉连续，需要将实体的世界坐标向左移动，补偿 body1 的重置
	if pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID); ok {
		oldX := pos.X
		pos.X += config.PolevaulterZombieJumpDistance
		log.Printf("[PoleVaultSystem] 撑杆僵尸 %d 应用位移补偿: %.1f → %.1f (补偿量=%.1f)",
			zombieID, oldX, pos.X, config.PolevaulterZombieJumpDistance)
	}

	// 清除补偿标记
	poleVault.NeedPositionCompensation = false
}
