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
		return
	}

	// 计算僵尸所在的行
	zombieRow := utils.GetEntityRow(zombiePos.Y, config.GridWorldStartY, config.CellHeight)

	// 获取所有植物实体
	plants := ecs.GetEntitiesWith2[*components.PlantComponent, *components.PositionComponent](s.entityManager)

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

		// 检测碰撞：僵尸左边缘是否接触到植物
		// 僵尸左边缘 = zombiePos.X - zombieCol.Width/2
		// 植物右边缘 = plantPos.X + CellWidth/2
		zombieLeftEdge := zombiePos.X - zombieCol.Width/2
		plantRightEdge := plantPos.X + config.CellWidth/2

		// 检测距离阈值（当僵尸接近植物时触发跳跃）
		jumpTriggerDistance := 20.0

		if zombieLeftEdge <= plantRightEdge+jumpTriggerDistance && zombieLeftEdge > plantRightEdge-config.CellWidth {
			// 触发跳跃
			s.startJump(zombieID, poleVault, zombiePos.X, plantPos.X, plantID)
			return
		}
	}
}

// startJump 开始跳跃
func (s *PoleVaultSystem) startJump(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent, currentX, plantX float64, plantID ecs.EntityID) {
	log.Printf("[PoleVaultSystem] 撑杆僵尸 %d 检测到植物 %d，开始跳跃", zombieID, plantID)

	// 设置跳跃状态
	poleVault.IsJumping = true
	poleVault.JumpProgress = 0.0
	poleVault.JumpStartX = currentX
	// 跳跃目标位置：植物位置左侧 + 跳跃距离
	poleVault.JumpTargetX = plantX - config.PolevaulterZombieJumpDistance
	poleVault.TargetPlantEntityID = uint64(plantID)

	// 停止移动
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, zombieID); ok {
		vel.VX = 0
		vel.VY = 0
	}

	// 播放跳跃动画
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:        types.UnitIDZombiePolevaulter,
		AnimationName: "anim_jump",
		Processed:     false,
	})
}

// updateJumping 更新跳跃中的僵尸
func (s *PoleVaultSystem) updateJumping(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent, deltaTime float64) {
	// 更新跳跃进度
	poleVault.JumpProgress += deltaTime / config.PolevaulterZombieJumpDuration
	if poleVault.JumpProgress > 1.0 {
		poleVault.JumpProgress = 1.0
	}

	// 计算当前位置（线性插值）
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		return
	}

	// X 位置插值
	pos.X = poleVault.JumpStartX + (poleVault.JumpTargetX-poleVault.JumpStartX)*poleVault.JumpProgress

	// Y 位置使用抛物线（跳跃弧度）
	// 最高点在进度 0.5 时
	jumpHeight := 80.0 // 跳跃最大高度（像素）
	t := poleVault.JumpProgress

	// 抛物线公式：-4h * (t - 0.5)^2 + h，在 t=0.5 时达到最高点 h
	verticalOffset := -4.0 * jumpHeight * (t - 0.5) * (t - 0.5) + jumpHeight

	// 应用跳跃弧度：向上是负Y方向
	// 使用行的基准Y坐标 + 垂直偏移
	row := utils.GetEntityRow(pos.Y+verticalOffset, config.GridWorldStartY, config.CellHeight)
	baseY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset
	pos.Y = baseY - verticalOffset

	// 跳跃完成
	if poleVault.JumpProgress >= 1.0 {
		s.finishJump(zombieID, poleVault)
	}
}

// finishJump 完成跳跃
func (s *PoleVaultSystem) finishJump(zombieID ecs.EntityID, poleVault *components.PoleVaultComponent) {
	log.Printf("[PoleVaultSystem] 撑杆僵尸 %d 跳跃完成，丢弃撑杆", zombieID)

	// 更新撑杆状态
	poleVault.HasPole = false
	poleVault.IsJumping = false
	poleVault.JumpProgress = 0.0

	// 恢复移动（使用普通速度）
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, zombieID); ok {
		vel.VX = config.PolevaulterZombieWalkSpeed
	}

	// 播放行走动画
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombiePolevaulter,
		ComboName: "walk",
		Processed: false,
	})
}
