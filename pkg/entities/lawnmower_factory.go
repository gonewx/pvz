package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// NewLawnmowerEntity 创建除草车实体
// 除草车是游戏中每行的最后防线，当僵尸到达屏幕左侧时自动触发
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载除草车 Reanim 资源）
//   - rs: Reanim 系统（用于初始化动画）
//   - lane: 所在行（1-5，与 EnabledLanes 一致）
//
// 返回:
//   - ecs.EntityID: 创建的除草车实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
//
// 注意：除草车默认创建时处于静止状态（未触发），需要通过 LawnmowerSystem 检测触发条件
func NewLawnmowerEntity(
	em *ecs.EntityManager,
	rm ResourceLoader,
	rs ReanimSystemInterface,
	lane int,
) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}
	if rs == nil {
		return 0, fmt.Errorf("reanim system cannot be nil")
	}
	if lane < 1 || lane > 5 {
		return 0, fmt.Errorf("invalid lane %d, must be between 1 and 5", lane)
	}

	// 计算除草车位置（世界坐标）
	// X坐标：使用配置的起始X位置（左侧台阶）
	posX := config.LawnmowerStartX

	// Y坐标：对应行的中心Y坐标
	// 行中心 = GridWorldStartY + (lane-1)*CellHeight + CellHeight/2.0
	posY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: posX,
		Y: posY,
	})

	// Story 10.2: 使用 ReanimComponent 加载除草车动画
	// 从 ResourceManager 获取除草车的 Reanim 数据和部件图片
	reanimXML := rm.GetReanimXML("LawnMower")
	partImages := rm.GetReanimPartImages("LawnMower")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load LawnMower Reanim resources")
	}

	// 添加 ReanimComponent
	// 除草车：显示所有部件（包括车身、轮子、引擎等）
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimXML: reanimXML,
		PartImages: partImages,
		IsLooping:  true, // 除草车动画循环播放
	})

	// Story 13.6: LawnMower 动画播放
	// 注意：LawnMower.reanim 使用 anim_normal 动画
	// 由于 lawnmower 可能没有在配置文件中定义，使用降级方案
	// anim_tricked 是另一种状态（可能是被僵尸推着走），不在正常游戏流程中使用
	if err := rs.PlayAnimation(entityID, "anim_normal"); err != nil {
		return 0, fmt.Errorf("failed to play LawnMower normal animation: %w", err)
	}

	// 暂停动画，直到除草车被触发（原版行为：静止时完全不动）
	// 触发后 LawnmowerSystem 会设置 IsPaused = false 恢复播放
	// Story 13.2: 移除 CurrentFrame 设置（PlayAnimation 会自动初始化动画从第 0 帧开始）
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID); ok {
		reanim.IsPaused = true
	}

	// 添加除草车组件
	ecs.AddComponent(em, entityID, &components.LawnmowerComponent{
		Lane:        lane,
		IsTriggered: false,
		IsMoving:    false,
		Speed:       config.LawnmowerSpeed,
	})

	log.Printf("[LawnmowerFactory] Created lawnmower entity %d on lane %d at (%.1f, %.1f)",
		entityID, lane, posX, posY)

	return entityID, nil
}
