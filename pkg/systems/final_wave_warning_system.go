package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// FinalWaveWarningSystem 最后一波提示动画管理系统
//
// Story 11.3: 最后一波僵尸提示动画
//
// 职责：
//   - 监控提示动画的播放时长
//   - 动画播放完成后自动销毁实体
//
// 架构说明：
//   - 遵循 ECS 架构：系统只处理逻辑，不存储状态
//   - 通过 FinalWaveWarningComponent 管理动画生命周期
type FinalWaveWarningSystem struct {
	entityManager *ecs.EntityManager
}

// NewFinalWaveWarningSystem 创建最后一波提示动画系统
//
// 参数：
//   - em: 实体管理器
//
// 返回：
//   - *FinalWaveWarningSystem: 新创建的系统实例
func NewFinalWaveWarningSystem(em *ecs.EntityManager) *FinalWaveWarningSystem {
	return &FinalWaveWarningSystem{
		entityManager: em,
	}
}

// Update 更新提示动画，播放完成后自动销毁
//
// 执行流程：
//  1. 查询所有带有 FinalWaveWarningComponent 的实体
//  2. 更新每个实体的 ElapsedTime
//  3. 如果 ElapsedTime >= DisplayTime，则销毁实体
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *FinalWaveWarningSystem) Update(deltaTime float64) {
	// 查询所有最后一波提示实体
	entities := ecs.GetEntitiesWith1[*components.FinalWaveWarningComponent](s.entityManager)

	for _, entityID := range entities {
		warningComp, ok := ecs.GetComponent[*components.FinalWaveWarningComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 更新已显示时间
		warningComp.ElapsedTime += deltaTime

		// 检查是否超过显示时长
		if warningComp.ElapsedTime >= warningComp.DisplayTime {
			// 销毁实体
			s.entityManager.DestroyEntity(entityID)
			log.Printf("[FinalWaveWarningSystem] Warning animation completed, entity %d destroyed", entityID)
		}
	}
}
