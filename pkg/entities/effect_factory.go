package entities

import (
	"fmt"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// NewPeaBulletHitEffect 创建豌豆子弹击中效果实体
// 击中效果在指定位置显示短暂的水花动画后自动消失
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载击中效果图像）
//   - x: 击中效果的世界坐标X位置（通常为子弹碰撞位置）
//   - y: 击中效果的世界坐标Y位置（通常为子弹碰撞位置）
//
// 返回:
//   - ecs.EntityID: 创建的击中效果实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPeaBulletHitEffect(em *ecs.EntityManager, rm ResourceLoader, x, y float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 加载击中效果图像
	// 注意：使用 firepea_spark.png 作为临时方案，因为原始的 PeaBulletHit.png 已删除
	// TODO: 考虑为击中效果创建专用图片或使用粒子效果
	hitImage, err := rm.LoadImage("assets/reanim/firepea_spark.png")
	if err != nil {
		return 0, fmt.Errorf("failed to load hit effect image: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// Story 6.3: 游戏世界实体统一使用 ReanimComponent 渲染
	// 为单图片实体创建简化的 Reanim 包装（无动画轨道）
	// 注意：UI 元素（植物卡片）仍使用 SpriteComponent，由专门的渲染系统处理
	em.AddComponent(entityID, createSimpleReanimComponent(hitImage, "hit_effect"))

	// 添加行为组件（标识为击中效果）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorPeaBulletHit,
	})

	// 添加计时器组件（控制效果显示时长）
	em.AddComponent(entityID, &components.TimerComponent{
		Name:        "hit_effect_duration",
		CurrentTime: 0.0,
		TargetTime:  config.HitEffectDuration, // 0.2秒后删除
		IsReady:     false,
	})

	return entityID, nil
}

// NewFallingPartEffect 创建掉落部件效果实体
// 掉落部件（手臂或头部）以抛物线轨迹飞出，一段时间后消失
//
// 参数:
//   - em: 实体管理器
//   - partImage: 部件图片（从 ReanimComponent 获取）
//   - x: 掉落起点的世界坐标X位置
//   - y: 掉落起点的世界坐标Y位置
//   - velocityX: 初始水平速度（向左为负，向右为正）
//   - velocityY: 初始垂直速度（向上为负）
//
// 返回:
//   - ecs.EntityID: 创建的掉落效果实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewFallingPartEffect(em *ecs.EntityManager, partImage *ebiten.Image, x, y, velocityX, velocityY float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if partImage == nil {
		return 0, fmt.Errorf("part image cannot be nil")
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// Story 6.3: 游戏世界实体统一使用 ReanimComponent 渲染
	// 为单图片实体创建简化的 Reanim 包装（无动画轨道）
	// 注意：UI 元素（植物卡片）仍使用 SpriteComponent，由专门的渲染系统处理
	em.AddComponent(entityID, createSimpleReanimComponent(partImage, "falling_part"))

	// 添加速度组件（抛物线运动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: velocityX,
		VY: velocityY,
	})

	// 添加行为组件（标识为掉落部件效果）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorFallingPart,
	})

	// 添加计时器组件（控制效果显示时长，2秒后删除）
	em.AddComponent(entityID, &components.TimerComponent{
		Name:        "falling_part_duration",
		CurrentTime: 0.0,
		TargetTime:  2.0, // 2秒后删除
		IsReady:     false,
	})

	return entityID, nil
}

// CreateFinalWaveEntity 创建最后一波警告动画实体
// 使用 FinalWave.reanim 动画，显示"A huge wave of zombies is approaching!"提示
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载 FinalWave 图片和 reanim 数据）
//   - reanimSystem: Reanim系统（用于初始化动画）
//   - x: 动画的世界坐标X位置（通常为屏幕中心）
//   - y: 动画的世界坐标Y位置（通常为屏幕中心）
//
// 返回:
//   - ecs.EntityID: 创建的动画实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func CreateFinalWaveEntity(em *ecs.EntityManager, rm ResourceLoader, reanimSystem ReanimSystemInterface, x, y float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}
	if reanimSystem == nil {
		return 0, fmt.Errorf("reanim system cannot be nil")
	}

	// 加载 FinalWave.reanim 数据
	reanimXML := rm.GetReanimXML("FinalWave")
	if reanimXML == nil {
		return 0, fmt.Errorf("FinalWave.reanim not found in resource manager")
	}

	// 加载 FinalWave 部件图片
	partImages := rm.GetReanimPartImages("FinalWave")
	if partImages == nil || len(partImages) == 0 {
		return 0, fmt.Errorf("FinalWave part images not found")
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加 ReanimComponent（动画组件）
	reanimComp := &components.ReanimComponent{
		Reanim:          reanimXML,
		PartImages:      partImages,
		CurrentAnim:     "anim",
		CurrentFrame:    0,
		IsLooping:       false, // 最后一波警告只播放一次
		VisibleTracks:   make(map[string]bool),
		MergedTracks:    make(map[string][]reanim.Frame),
		FrameAccumulator: 0.0,
	}
	ecs.AddComponent(em, entityID, reanimComp)

	// 初始化 Reanim 动画
	reanimSystem.PlayAnimation(entityID, "anim")

	// 添加生命周期组件（动画播放完毕后自动删除）
	// FinalWave.reanim 有 27 帧，FPS=12，播放时长约 2.25 秒
	animDuration := float64(27) / float64(reanimXML.FPS)
	ecs.AddComponent(em, entityID, &components.LifetimeComponent{
		MaxLifetime:     animDuration,
		CurrentLifetime: 0.0,
		IsExpired:       false,
	})

	return entityID, nil
}
