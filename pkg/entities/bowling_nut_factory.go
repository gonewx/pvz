package entities

import (
	"fmt"
	"log"
	"math"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
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

	// 加载 Wallnut Reanim 资源
	reanimXML := rm.GetReanimXML("Wallnut")
	partImages := rm.GetReanimPartImages("Wallnut")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load Wallnut Reanim resources for bowling nut")
	}

	// 从动画数据动态计算滚动速度
	rollingSpeed := calculateRollingSpeedFromReanim(reanimXML)

	// 创建实体
	entityID := em.CreateEntity()

	// 添加 PositionComponent
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldX,
		Y: worldY,
	})

	// 添加 BowlingNutComponent
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX:    rollingSpeed,
		Row:          row,
		IsRolling:    true, // 放置后立即滚动
		IsExplosive:  isExplosive,
		BounceCount:  0,     // Story 19.7 使用
		SoundPlaying: false, // 音效由 BowlingNutSystem 管理
	})

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
	// StartFrame 从配置读取，确保从滚动动画起始帧开始循环
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		AnimationName: "anim_face",
		Processed:     false,
		StartFrame:    config.BowlingNutRollingStartFrame,
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
	log.Printf("[BowlingNutFactory] 创建保龄球坚果: entityID=%d, row=%d, col=%d, type=%s, speed=%.1f, worldX=%.1f, worldY=%.1f",
		entityID, row, col, nutType, rollingSpeed, worldX, worldY)

	return entityID, nil
}

// calculateRollingSpeedFromReanim 从 Reanim 数据动态计算滚动速度
// 使位移速度与动画旋转速度同步
//
// 计算逻辑：
//  1. 从 ReanimXML.FPS 获取帧率
//  2. 从 anim_face 轨道计算可见帧数
//  3. 从滚动帧的 x 轨迹计算周长（直径 = max_x - min_x，周长 = π * 直径）
//  4. 速度 = 周长 * FPS / 滚动帧数
//
// 参数:
//   - reanimXML: Reanim 动画数据
//
// 返回:
//   - float64: 滚动速度（像素/秒）
func calculateRollingSpeedFromReanim(reanimXML *reanim.ReanimXML) float64 {
	// 获取动画 FPS
	fps := reanimXML.FPS
	if fps <= 0 {
		fps = 12 // 默认 PVZ 动画帧率
	}

	// 获取 anim_face 轨道的可见帧数
	totalVisibleFrames := countTrackVisibleFrames(reanimXML, "anim_face")
	if totalVisibleFrames <= 0 {
		totalVisibleFrames = 30 // 默认值（17 摇摆 + 13 滚动）
	}

	// 滚动动画帧数 = 可见帧总数 - 起始帧
	rollingFrameCount := totalVisibleFrames - config.BowlingNutRollingStartFrame
	if rollingFrameCount <= 0 {
		rollingFrameCount = 13 // 默认滚动帧数
	}

	// 从滚动帧的 x 轨迹计算周长
	circumference := calculateCircumferenceFromTrack(reanimXML, "anim_face", config.BowlingNutRollingStartFrame)
	if circumference <= 0 {
		circumference = 220.0 // 默认周长（直径约 70 像素）
	}

	// 计算同步速度
	speed := config.CalculateBowlingNutSpeed(fps, rollingFrameCount, circumference)

	log.Printf("[BowlingNutFactory] 动态计算滚动速度: fps=%d, totalFrames=%d, rollingFrames=%d, circumference=%.1f, speed=%.1f",
		fps, totalVisibleFrames, rollingFrameCount, circumference, speed)

	return speed
}

// countTrackVisibleFrames 计算指定轨道的可见帧数
// 可见帧定义：FrameNum 为 nil 或 >= 0 的帧
//
// 参数:
//   - reanimXML: Reanim 动画数据
//   - trackName: 轨道名称
//
// 返回:
//   - int: 可见帧数量
func countTrackVisibleFrames(reanimXML *reanim.ReanimXML, trackName string) int {
	// 查找指定轨道
	var track *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == trackName {
			track = &reanimXML.Tracks[i]
			break
		}
	}
	if track == nil {
		return 0
	}

	// 统计可见帧数
	// 逻辑：遍历所有帧，累积 FrameNum 状态
	// - FrameNum == nil：继承上一帧状态
	// - FrameNum == -1：隐藏
	// - FrameNum >= 0：可见
	visibleCount := 0
	currentVisible := true // 默认可见

	for _, frame := range track.Frames {
		if frame.FrameNum != nil {
			currentVisible = *frame.FrameNum >= 0
		}
		if currentVisible {
			visibleCount++
		}
	}

	return visibleCount
}

// calculateCircumferenceFromTrack 从轨道的 x 轨迹计算滚动周长
// 周长 = π * 直径 = π * (max_x - min_x)
//
// 参数:
//   - reanimXML: Reanim 动画数据
//   - trackName: 轨道名称
//   - startFrame: 滚动动画起始帧（逻辑帧索引）
//
// 返回:
//   - float64: 周长（像素）
func calculateCircumferenceFromTrack(reanimXML *reanim.ReanimXML, trackName string, startFrame int) float64 {
	// 查找指定轨道
	var track *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == trackName {
			track = &reanimXML.Tracks[i]
			break
		}
	}
	if track == nil {
		return 0
	}

	// 遍历滚动帧，找到 x 的最大值和最小值
	minX := math.MaxFloat64
	maxX := -math.MaxFloat64
	currentX := 0.0
	visibleFrameIndex := 0
	currentVisible := true

	for _, frame := range track.Frames {
		// 更新可见状态
		if frame.FrameNum != nil {
			currentVisible = *frame.FrameNum >= 0
		}

		if currentVisible {
			// 更新 x 值（继承机制）
			if frame.X != nil {
				currentX = *frame.X
			}

			// 只统计滚动帧（从 startFrame 开始）
			if visibleFrameIndex >= startFrame {
				if currentX < minX {
					minX = currentX
				}
				if currentX > maxX {
					maxX = currentX
				}
			}
			visibleFrameIndex++
		}
	}

	// 如果没有找到有效数据，返回 0
	if minX == math.MaxFloat64 || maxX == -math.MaxFloat64 {
		return 0
	}

	// 周长 = π * 直径
	diameter := maxX - minX
	circumference := math.Pi * diameter

	return circumference
}
