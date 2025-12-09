package entities

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// ReanimSystemInterface 定义 ReanimSystem 的接口，用于工厂函数依赖注入
// 这样可以避免循环依赖，同时方便测试
// Story 13.8: 简化接口，只保留核心 API
type ReanimSystemInterface interface {
	// 核心动画播放 API
	PlayAnimation(entityID ecs.EntityID, animName string) error
	PlayCombo(entityID ecs.EntityID, unitID, comboName string) error
	// RenderToTexture 将指定实体的 Reanim 渲染到目标纹理（离屏渲染）
	RenderToTexture(entityID ecs.EntityID, target *ebiten.Image) error
	// PrepareStaticPreview prepares a Reanim entity for static preview (Story 11.1)
	PrepareStaticPreview(entityID ecs.EntityID, plantType types.PlantType) error
}

// NewPlantEntity 创建植物实体
// 根据植物类型和网格位置创建一个完整的植物实体，包含位置、图像和植物组件
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载植物图像和 Reanim 资源）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - rs: Reanim 系统（用于初始化动画）
//   - plantType: 植物类型（向日葵、豌豆射手等）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的植物实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPlantEntity(em *ecs.EntityManager, rm ResourceLoader, gs *game.GameState, rs ReanimSystemInterface, plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	// 计算植物原点坐标（使用世界坐标系统）
	// Reanim 坐标系统：部件坐标从原点开始绘制
	// 中心偏移会由 ReanimSystem 自动计算并在渲染时应用
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2 + config.PlantOffsetY

	// Story 6.3: Reanim 迁移完成
	// 注意：旧版代码使用 SpriteComponent 和 GetPlantImagePath()
	// 现在所有植物都使用 ReanimComponent，不再需要加载 sprite 图片
	// SpriteComponent 已被移除，植物渲染完全由 ReanimComponent 处理

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// 添加植物组件
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType:       plantType,
		GridRow:         row,
		GridCol:         col,
		AttackAnimState: components.AttackAnimIdle, // Story 10.3: 初始化为空闲状态
		LastFiredFrame:  -1,                        // 初始化为 -1，表示还未发射过
		BlinkTimer:      3.0,                       // Story 6.4: 初始化眨眼计时器为3秒
	})

	// 为向日葵添加特定组件
	if plantType == components.PlantSunflower {
		// 添加生命值组件
		em.AddComponent(entityID, &components.HealthComponent{
			CurrentHealth: config.SunflowerDefaultHealth,
			MaxHealth:     config.SunflowerDefaultHealth,
		})

		// 添加行为组件
		em.AddComponent(entityID, &components.BehaviorComponent{
			Type: components.BehaviorSunflower,
		})

		// 添加计时器组件（首次生产周期为 7 秒）
		em.AddComponent(entityID, &components.TimerComponent{
			Name:        "sun_production",
			TargetTime:  7.0,
			CurrentTime: 0,
			IsReady:     false,
		})

		// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
		// 从 ResourceManager 获取向日葵的 Reanim 数据和部件图片
		reanimXML := rm.GetReanimXML("SunFlower")
		partImages := rm.GetReanimPartImages("SunFlower")

		if reanimXML == nil || partImages == nil {
			return 0, fmt.Errorf("failed to load SunFlower Reanim resources")
		}

		// 添加 ReanimComponent
		em.AddComponent(entityID, &components.ReanimComponent{
			ReanimName: "SunFlower",
			ReanimXML:  reanimXML,
			PartImages: partImages,
		})

		// Story 13.8: 使用 PlayCombo API 播放默认动画
		if err := rs.PlayCombo(entityID, "sunflower", ""); err != nil {
			return 0, fmt.Errorf("failed to play SunFlower default animation: %w", err)
		}
		log.Printf("[PlantFactory] 向日葵 %d: 成功添加 ReanimComponent 并初始化动画", entityID)
	}

	// 为豌豆射手添加特定组件
	if plantType == components.PlantPeashooter {
		// 添加生命值组件
		em.AddComponent(entityID, &components.HealthComponent{
			CurrentHealth: config.PeashooterDefaultHealth,
			MaxHealth:     config.PeashooterDefaultHealth,
		})

		// Story 10.3: 添加植物组件（用于攻击动画状态管理）
		em.AddComponent(entityID, &components.PlantComponent{
			PlantType:         components.PlantPeashooter,
			GridRow:           row, // ✅ 添加缺失的 GridRow
			GridCol:           col, // ✅ 添加缺失的 GridCol
			AttackAnimState:   components.AttackAnimIdle,
			PendingProjectile: false,
			LastFiredFrame:    -1, // 初始化为 -1，表示还未发射过
			LastMouthX:        0,
		})

		// 添加行为组件
		em.AddComponent(entityID, &components.BehaviorComponent{
			Type: components.BehaviorPeashooter,
		})

		// 添加攻击冷却计时器
		em.AddComponent(entityID, &components.TimerComponent{
			Name:        "attack_cooldown",
			TargetTime:  1.4, // 攻击间隔 1.4 秒
			CurrentTime: 0,
			IsReady:     false,
		})

		// Story 13.6: 使用集中配置文件创建豌豆射手动画
		// 从 ResourceManager 获取豌豆射手的 Reanim 数据和部件图片
		reanimXML := rm.GetReanimXML("PeaShooterSingle")
		partImages := rm.GetReanimPartImages("PeaShooterSingle")

		if reanimXML == nil || partImages == nil {
			return 0, fmt.Errorf("failed to load PeaShooterSingle Reanim resources")
		}

		// 添加基础的 ReanimComponent
		em.AddComponent(entityID, &components.ReanimComponent{
			ReanimName: "PeaShooterSingle",
			ReanimXML:  reanimXML,
			PartImages: partImages,
		})

		// Story 13.8: 使用 PlayCombo API 播放默认动画
		// PlayCombo 会自动从 data/reanim_config.yaml 读取配置
		if err := rs.PlayCombo(entityID, "peashootersingle", ""); err != nil {
			return 0, fmt.Errorf("failed to play peashooter default animation: %w", err)
		}

		log.Printf("[PlantFactory] 豌豆射手 %d: 成功使用集中配置文件创建动画", entityID)
	}

	// Story 10.7: 为植物添加阴影组件
	// 根据植物类型从配置获取阴影尺寸
	var shadowEntityType string
	switch plantType {
	case components.PlantSunflower:
		shadowEntityType = "sunflower"
	case components.PlantPeashooter:
		shadowEntityType = "peashooter"
	case components.PlantWallnut:
		shadowEntityType = "wallnut"
	case components.PlantCherryBomb:
		shadowEntityType = "cherrybomb"
	default:
		shadowEntityType = "" // 使用默认尺寸,暂不支持 repeater 和 snowpea
	}

	shadowSize := config.GetShadowSize(shadowEntityType)
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
	})

	return entityID, nil
}

// NewWallnutEntity 创建坚果墙实体
// 坚果墙是一种高生命值的防御植物，没有攻击能力，根据生命值百分比显示不同的外观状态
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载坚果墙图像和 Reanim 资源）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - rs: Reanim 系统（用于初始化动画）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的坚果墙实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewWallnutEntity(em *ecs.EntityManager, rm ResourceLoader, gs *game.GameState, rs ReanimSystemInterface, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标（使用世界坐标系统）
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
	// 从 ResourceManager 获取坚果墙的 Reanim 数据和部件图片
	// 注意：ResourceManager 加载时使用 "Wallnut"（与文件名匹配）
	reanimXML := rm.GetReanimXML("Wallnut")
	partImages := rm.GetReanimPartImages("Wallnut")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load Wallnut Reanim resources")
	}

	// Clone partImages to avoid shared state issues when modifying images (e.g. cracking)
	clonedPartImages := make(map[string]*ebiten.Image, len(partImages))
	for k, v := range partImages {
		clonedPartImages[k] = v
	}

	// 添加植物组件（用于碰撞检测和网格位置追踪）
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType:       components.PlantWallnut,
		GridRow:         row,
		GridCol:         col,
		AttackAnimState: components.AttackAnimIdle, // Story 10.3: 初始化为空闲状态
	})

	// 添加生命值组件（坚果墙拥有极高的生命值）
	em.AddComponent(entityID, &components.HealthComponent{
		CurrentHealth: config.WallnutDefaultHealth, // 4000
		MaxHealth:     config.WallnutDefaultHealth,
	})

	// 添加行为组件（坚果墙行为）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorWallnut,
	})

	// 添加 ReanimComponent
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "Wallnut",
		ReanimXML:  reanimXML,
		PartImages: clonedPartImages,
	})

	// Story 13.8: 使用 PlayCombo API 播放默认动画
	if err := rs.PlayCombo(entityID, "wallnut", ""); err != nil {
		return 0, fmt.Errorf("failed to play WallNut default animation: %w", err)
	}

	// 添加碰撞组件（用于僵尸碰撞检测）
	// 坚果墙的碰撞盒与普通植物类似
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.CellWidth * 0.8,  // 碰撞盒宽度略小于格子宽度
		Height: config.CellHeight * 0.8, // 碰撞盒高度略小于格子高度
	})

	// Story 10.7: 为坚果墙添加阴影组件
	shadowSize := config.GetShadowSize("wallnut")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
	})

	return entityID, nil
}

// NewCherryBombEntity 创建樱桃炸弹实体
// 樱桃炸弹是一种高成本的一次性爆炸植物，种植后经过引信时间（1.5秒）后爆炸，
// 对以自身为中心的3x3范围内的所有僵尸造成1800点伤害（足以秒杀所有僵尸）
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载樱桃炸弹图像和 Reanim 资源）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的樱桃炸弹实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
//
// Story 14.3: Epic 14 - 移除 ReanimSystem 依赖，动画通过 AnimationCommand 组件初始化
func NewCherryBombEntity(em *ecs.EntityManager, rm ResourceLoader, gs *game.GameState, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标（使用世界坐标系统）
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// 从 ResourceManager 获取樱桃炸弹的 Reanim 数据和部件图片
	reanimXML := rm.GetReanimXML("CherryBomb")
	partImages := rm.GetReanimPartImages("CherryBomb")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load CherryBomb Reanim resources")
	}

	// 添加 ReanimComponent
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "CherryBomb",
		ReanimXML:  reanimXML,
		PartImages: partImages,
	})

	// ✅ Epic 14: 使用 AnimationCommand 触发动画（替代直接调用 ReanimSystem）
	// 添加动画命令组件，让 ReanimSystem 在 Update 中处理
	// 樱桃炸弹播放 anim_idle（引信动画）
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		AnimationName: "anim_idle",
		Processed:     false,
	})
	log.Printf("[PlantFactory] 樱桃炸弹 %d: 成功添加 ReanimComponent 并初始化引信动画", entityID)

	// 添加植物组件（用于碰撞检测和网格位置追踪）
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType:       components.PlantCherryBomb,
		GridRow:         row,
		GridCol:         col,
		AttackAnimState: components.AttackAnimIdle, // Story 10.3: 初始化为空闲状态
	})

	// 添加行为组件（樱桃炸弹行为）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorCherryBomb,
	})

	// 添加引信计时器组件（1.5秒后爆炸）
	em.AddComponent(entityID, &components.TimerComponent{
		Name:        "fuse_timer",
		TargetTime:  config.CherryBombFuseTime, // 1.5秒
		CurrentTime: 0,
		IsReady:     false,
	})

	// 添加碰撞组件（用于后续爆炸范围检测）
	// 碰撞盒大小与格子大小一致
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.CellWidth,
		Height: config.CellHeight,
	})

	// Story 10.7: 为樱桃炸弹添加阴影组件
	shadowSize := config.GetShadowSize("cherrybomb")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
	})

	return entityID, nil
}

// NewPotatoMineEntity 创建土豆雷实体
// 土豆雷是一种低成本的地雷植物，需要一定时间武装后才能触发爆炸
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载土豆雷图像和 Reanim 资源）
//   - gs: 游戏状态
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的土豆雷实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPotatoMineEntity(em *ecs.EntityManager, rm ResourceLoader, gs *game.GameState, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标（使用世界坐标系统）
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// 从 ResourceManager 获取土豆雷的 Reanim 数据和部件图片
	reanimXML := rm.GetReanimXML("PotatoMine")
	partImages := rm.GetReanimPartImages("PotatoMine")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load PotatoMine Reanim resources")
	}

	// 添加 ReanimComponent
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "PotatoMine",
		ReanimXML:  reanimXML,
		PartImages: partImages,
	})

	// 使用 AnimationCommand 触发默认动画（anim_armed）
	// 设置 UnitID 以便 PlayAnimationWithConfig 能从配置中获取 Scale
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:        "potatomine",
		AnimationName: "anim_armed",
		Processed:     false,
	})
	log.Printf("[PlantFactory] 土豆雷 %d: 成功添加 ReanimComponent 并初始化动画", entityID)

	// 添加植物组件（用于碰撞检测和网格位置追踪）
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType:       components.PlantPotatoMine,
		GridRow:         row,
		GridCol:         col,
		AttackAnimState: components.AttackAnimIdle,
	})

	// 添加行为组件（土豆雷行为）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorPotatoMine,
	})

	// 添加碰撞组件（用于后续爆炸范围检测）
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.CellWidth,
		Height: config.CellHeight,
	})

	// 添加阴影组件
	shadowSize := config.GetShadowSize("potatomine")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: 0,
	})

	return entityID, nil
}
