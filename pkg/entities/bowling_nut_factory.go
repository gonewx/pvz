package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// NewBowlingNutEntity 创建保龄球坚果实体
// Story 19.6: 从传送带放置的坚果，自动滚动向右移动
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源加载器（用于加载 Reanim 资源）
//   - row: 放置行号 (0-4)
//   - col: 放置列号 (0-8)
//   - isExplosive: 是否为爆炸坚果
//
// 返回:
//   - ecs.EntityID: 创建的保龄球坚果实体ID
//   - error: 如果创建失败返回错误信息
func NewBowlingNutEntity(em *ecs.EntityManager, rm ResourceLoader, row, col int, isExplosive bool) (ecs.EntityID, error) {
	// 计算世界坐标（基于 row, col）
	worldX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 创建实体
	entityID := em.CreateEntity()

	// 添加 PositionComponent
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldX,
		Y: worldY,
	})

	// 添加 BowlingNutComponent
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX:    config.BowlingNutSpeed,
		Row:          row,
		IsRolling:    true, // 放置后立即滚动
		IsExplosive:  isExplosive,
		BounceCount:  0,     // Story 19.7 使用
		SoundPlaying: false, // 音效由 BowlingNutSystem 管理
	})

	// 加载 Wallnut Reanim 资源
	reanimXML := rm.GetReanimXML("Wallnut")
	partImages := rm.GetReanimPartImages("Wallnut")

	if reanimXML == nil || partImages == nil {
		em.DestroyEntity(entityID)
		return 0, fmt.Errorf("failed to load Wallnut Reanim resources for bowling nut")
	}

	// Clone partImages to avoid shared state issues
	clonedPartImages := make(map[string]*ebiten.Image, len(partImages))
	for k, v := range partImages {
		clonedPartImages[k] = v
	}

	// 添加 ReanimComponent
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "Wallnut",
		ReanimXML:  reanimXML,
		PartImages: clonedPartImages,
	})

	// 添加 AnimationCommandComponent 触发滚动动画
	// 使用 anim_face 动画（包含摇摆和滚动两部分）
	// 帧结构分析：
	//   - 逻辑帧 0-16 → 物理帧 0-16（摇摆动画）
	//   - 逻辑帧 17 → 物理帧 43（起始帧，kx=0°）
	//   - 逻辑帧 18-29 → 物理帧 44-55（kx=27.6°→360°）
	// StartFrame=17 确保完整的360°循环：0° → 360° → 循环回0°
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		AnimationName: "anim_face",
		Processed:     false,
		StartFrame:    17, // 从0°开始，确保360°→0°循环连续
	})

	// 添加 CollisionComponent（为 Story 19.7 预留）
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.BowlingNutCollisionWidth,
		Height: config.BowlingNutCollisionHeight,
	})

	// 记录日志
	nutType := components.BowlingNutTypeNormal
	if isExplosive {
		nutType = components.BowlingNutTypeExplosive
	}
	log.Printf("[BowlingNutFactory] 创建保龄球坚果: entityID=%d, row=%d, col=%d, type=%s, worldX=%.1f, worldY=%.1f",
		entityID, row, col, nutType, worldX, worldY)

	return entityID, nil
}

