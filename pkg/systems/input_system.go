package systems

import (
	"log"
	"math"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputSystem 处理所有用户输入，包括鼠标点击和键盘输入
type InputSystem struct {
	entityManager      *ecs.EntityManager
	resourceManager    *game.ResourceManager
	gameState          *game.GameState
	sunCounterX        float64         // 阳光计数器X坐标（收集动画目标）
	sunCounterY        float64         // 阳光计数器Y坐标（收集动画目标）
	collectSoundPlayer *audio.Player   // 收集阳光音效播放器
	lawnGridSystem     *LawnGridSystem // 草坪网格管理系统
	lawnGridEntityID   ecs.EntityID    // 草坪网格实体ID
	plantSoundPlayer   *audio.Player   // 种植音效播放器
}

// NewInputSystem 创建一个新的输入系统
func NewInputSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, sunCounterX, sunCounterY float64, lawnGridSystem *LawnGridSystem, lawnGridEntityID ecs.EntityID) *InputSystem {
	system := &InputSystem{
		entityManager:    em,
		resourceManager:  rm,
		gameState:        gs,
		sunCounterX:      sunCounterX,
		sunCounterY:      sunCounterY,
		lawnGridSystem:   lawnGridSystem,
		lawnGridEntityID: lawnGridEntityID,
	}

	// 加载收集阳光音效（使用 LoadSoundEffect 而非 LoadAudio 以避免循环播放）
	player, err := rm.LoadSoundEffect("assets/audio/Sound/points.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load sun collect sound: %v", err)
	} else {
		system.collectSoundPlayer = player
	}

	// 加载种植音效
	plantPlayer, err := rm.LoadSoundEffect("assets/audio/Sound/plant.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load plant sound: %v", err)
	} else {
		system.plantSoundPlayer = plantPlayer
	}

	return system
}

// Update 处理用户输入
func (s *InputSystem) Update(deltaTime float64) {
	// 检测鼠标右键取消种植模式
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if s.gameState.IsPlantingMode {
			log.Printf("[InputSystem] 右键取消种植模式")
			s.gameState.ExitPlantingMode()
			s.destroyPlantPreview()
		}
	}

	// 检测鼠标左键是否刚被按下
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		log.Printf("[InputSystem] 鼠标点击: (%d, %d)", mouseX, mouseY)

		// 优先检查植物卡片点击
		cardHandled := s.handlePlantCardClick(mouseX, mouseY)
		if cardHandled {
			return // 已处理卡片点击，不继续处理其他点击
		}

		// 检查是否在种植模式下点击草坪
		lawnHandled := s.handleLawnClick(mouseX, mouseY)
		if lawnHandled {
			return // 已处理草坪种植，不继续处理阳光
		}

		// 查询所有可点击的阳光实体
		entities := s.entityManager.GetEntitiesWith(
			reflect.TypeOf(&components.PositionComponent{}),
			reflect.TypeOf(&components.ClickableComponent{}),
			reflect.TypeOf(&components.SunComponent{}),
		)

		log.Printf("[InputSystem] 找到 %d 个阳光实体", len(entities))

		// 从后向前遍历，确保点击最上层的阳光（假设后面的实体渲染在上层）
		for i := len(entities) - 1; i >= 0; i-- {
			id := entities[i]

			// 获取组件
			posComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
			clickableComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.ClickableComponent{}))
			sunComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))

			// 类型断言
			pos := posComp.(*components.PositionComponent)
			clickable := clickableComp.(*components.ClickableComponent)
			sun := sunComp.(*components.SunComponent)

			// 只处理可点击的阳光（允许下落中和已落地的阳光）
			if !clickable.IsEnabled {
				continue
			}

			// 不允许点击已被收集的阳光
			if sun.State == components.SunCollecting {
				continue
			}

			// AABB 碰撞检测（点在矩形内）
			if float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+clickable.Width &&
				float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+clickable.Height {
				// 点击命中！
				log.Printf("[InputSystem] 点击命中阳光! 位置:(%.1f, %.1f), 状态:%d", pos.X, pos.Y, sun.State)
				s.handleSunClick(id, pos)
				break // 只处理第一个命中的阳光
			}
		}
	}
}

// handleSunClick 处理阳光被点击的逻辑
func (s *InputSystem) handleSunClick(sunID ecs.EntityID, pos *components.PositionComponent) {
	// 1. 更新阳光状态为收集中
	sunComp, _ := s.entityManager.GetComponent(sunID, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)
	sun.State = components.SunCollecting
	log.Printf("[InputSystem] 阳光开始收集动画")

	// 2. 禁用点击，防止重复点击
	clickableComp, _ := s.entityManager.GetComponent(sunID, reflect.TypeOf(&components.ClickableComponent{}))
	clickable := clickableComp.(*components.ClickableComponent)
	clickable.IsEnabled = false

	// 3. 播放收集音效（单次播放，不循环）
	if s.collectSoundPlayer != nil {
		// 重置播放位置到开头（如果之前播放过）
		s.collectSoundPlayer.Rewind()
		// 播放音效（会自动播放到结束后停止）
		s.collectSoundPlayer.Play()
		log.Printf("[InputSystem] 播放收集音效")
	}

	// 4. 移除 LifetimeComponent，防止收集过程中过期消失
	s.entityManager.RemoveComponent(sunID, reflect.TypeOf(&components.LifetimeComponent{}))

	// 5. 计算飞向阳光计数器的速度向量
	dx := s.sunCounterX - pos.X
	dy := s.sunCounterY - pos.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// 飞行速度: 600像素/秒
	speed := 600.0
	vx := (dx / distance) * speed
	vy := (dy / distance) * speed

	// 6. 更新 VelocityComponent（如果不存在则添加）
	velComp, exists := s.entityManager.GetComponent(sunID, reflect.TypeOf(&components.VelocityComponent{}))
	if exists {
		// 更新现有速度
		vel := velComp.(*components.VelocityComponent)
		vel.VX = vx
		vel.VY = vy
	} else {
		// 添加新的速度组件（理论上应该已存在，但防御性编程）
		s.entityManager.AddComponent(sunID, &components.VelocityComponent{
			VX: vx,
			VY: vy,
		})
	}

	// 注意: 阳光数值会在 SunCollectionSystem 检测到阳光到达目标位置时增加
	// 这样可以确保视觉效果（阳光飞行 → 到达 → 数值增加）的正确时序
}

// handlePlantCardClick 处理植物卡片点击逻辑
// 返回 true 表示处理了点击，false 表示未处理
func (s *InputSystem) handlePlantCardClick(mouseX, mouseY int) bool {
	// 查询所有植物卡片实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantCardComponent{}),
		reflect.TypeOf(&components.ClickableComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.UIComponent{}),
	)

	log.Printf("[InputSystem] 检查植物卡片点击: 鼠标(%d, %d), 找到 %d 个卡片", mouseX, mouseY, len(entities))

	// 遍历卡片实体，检测点击
	for _, entityID := range entities {
		// 获取组件
		cardComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PlantCardComponent{}))
		clickableComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.ClickableComponent{}))
		posComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		uiComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.UIComponent{}))

		card := cardComp.(*components.PlantCardComponent)
		clickable := clickableComp.(*components.ClickableComponent)
		pos := posComp.(*components.PositionComponent)
		ui := uiComp.(*components.UIComponent)

		log.Printf("[InputSystem] 卡片 %d: PlantType=%v, 位置=(%.1f, %.1f), 可点击区域=%.1fx%.1f, IsAvailable=%v, IsEnabled=%v",
			entityID, card.PlantType, pos.X, pos.Y, clickable.Width, clickable.Height, card.IsAvailable, clickable.IsEnabled)

		// 只处理可用的卡片
		if !card.IsAvailable || !clickable.IsEnabled {
			log.Printf("[InputSystem] 卡片 %d 不可用，跳过", entityID)
			continue
		}

		// AABB 碰撞检测
		if float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+clickable.Width &&
			float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+clickable.Height {
			// 点击命中卡片！
			log.Printf("[InputSystem] 点击植物卡片: PlantType=%v, IsPlantingMode=%v",
				card.PlantType, s.gameState.IsPlantingMode)

			// 检查当前是否已在种植模式
			if s.gameState.IsPlantingMode {
				// 如果已在种植模式，点击卡片退出种植模式
				log.Printf("[InputSystem] 退出种植模式（点击卡片）")
				s.gameState.ExitPlantingMode()
				s.destroyPlantPreview()
				// 可选：设置卡片状态为 Normal
				ui.State = components.UINormal
			} else {
				// 如果不在种植模式，进入种植模式
				log.Printf("[InputSystem] 进入种植模式: PlantType=%v", card.PlantType)
				s.gameState.EnterPlantingMode(card.PlantType)

				// 创建植物预览实体
				// 注意：需要导入 entities 包
				mouseXf, mouseYf := float64(mouseX), float64(mouseY)
				s.createPlantPreview(card.PlantType, mouseXf, mouseYf)

				// 可选：设置卡片状态为 Clicked（视觉反馈）
				ui.State = components.UIClicked
			}

			return true // 已处理点击
		}
	}

	return false // 未处理任何卡片点击
}

// createPlantPreview 创建植物预览实体
func (s *InputSystem) createPlantPreview(plantType components.PlantType, x, y float64) {
	// 先删除现有预览
	s.destroyPlantPreview()

	// 获取植物预览图像路径
	imagePath := utils.GetPlantPreviewImagePath(plantType)

	// 加载植物图像
	plantImage, err := s.resourceManager.LoadImage(imagePath)
	if err != nil {
		log.Printf("[InputSystem] Failed to load plant preview image %s: %v", imagePath, err)
		return
	}

	// 创建预览实体
	entityID := s.entityManager.CreateEntity()

	// 添加位置组件
	s.entityManager.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加精灵组件
	s.entityManager.AddComponent(entityID, &components.SpriteComponent{
		Image: plantImage,
	})

	// 添加植物预览组件
	s.entityManager.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: plantType,
		Alpha:     0.5, // 半透明效果
	})

	log.Printf("[InputSystem] Created plant preview (ID: %d, Type: %v) at (%.1f, %.1f)",
		entityID, plantType, x, y)
}

// destroyPlantPreview 删除所有植物预览实体
func (s *InputSystem) destroyPlantPreview() {
	// 查询所有拥有 PlantPreviewComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantPreviewComponent{}),
	)

	// 删除所有预览实体
	for _, entityID := range entities {
		s.entityManager.DestroyEntity(entityID)
		log.Printf("[InputSystem] Destroyed plant preview entity (ID: %d)", entityID)
	}

	// 立即清理标记删除的实体
	s.entityManager.RemoveMarkedEntities()
}

// handleLawnClick 处理草坪点击种植逻辑
// 返回 true 表示处理了点击，false 表示未处理
func (s *InputSystem) handleLawnClick(mouseX, mouseY int) bool {
	// 检查当前是否在种植模式
	isPlanting, plantType := s.gameState.GetPlantingMode()
	if !isPlanting {
		return false // 不在种植模式，不处理
	}

	// 转换鼠标坐标到网格坐标（使用世界坐标系统）
	col, row, isValid := utils.MouseToGridCoords(
		mouseX, mouseY,
		s.gameState.CameraX,
		config.GridWorldStartX, config.GridWorldStartY,
		config.GridColumns, config.GridRows,
		config.CellWidth, config.CellHeight,
	)
	if !isValid {
		log.Printf("[InputSystem] 鼠标点击在网格外: (%d, %d)", mouseX, mouseY)
		return false // 点击在网格外
	}

	log.Printf("[InputSystem] 草坪点击: col=%d, row=%d", col, row)

	// 检查格子是否已被占用
	if s.lawnGridSystem.IsOccupied(s.lawnGridEntityID, col, row) {
		log.Printf("[InputSystem] 格子 (%d, %d) 已被占用，无法种植", col, row)
		return true // 处理了点击（虽然没有种植），防止继续处理阳光
	}

	// 获取植物消耗
	sunCost := s.getPlantCost(plantType)

	// 尝试扣除阳光
	if !s.gameState.SpendSun(sunCost) {
		log.Printf("[InputSystem] 阳光不足，需要 %d，当前 %d", sunCost, s.gameState.GetSun())
		return true // 处理了点击，但阳光不足
	}

	log.Printf("[InputSystem] 扣除阳光 %d，剩余 %d", sunCost, s.gameState.GetSun())

	// 创建植物实体（需要导入 entities 包）
	plantID, err := s.createPlantEntity(plantType, col, row)
	if err != nil {
		log.Printf("[InputSystem] 创建植物实体失败: %v", err)
		// 创建失败，返还阳光
		s.gameState.AddSun(sunCost)
		return true
	}

	log.Printf("[InputSystem] 成功创建植物实体 (ID: %d, Type: %v) 在 (%d, %d)", plantID, plantType, col, row)

	// 标记格子为占用
	err = s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, col, row, plantID)
	if err != nil {
		log.Printf("[InputSystem] 标记格子占用失败: %v", err)
		// 失败时删除植物实体并返还阳光
		s.entityManager.DestroyEntity(plantID)
		s.gameState.AddSun(sunCost)
		return true
	}

	// 播放种植音效
	if s.plantSoundPlayer != nil {
		s.plantSoundPlayer.Rewind()
		s.plantSoundPlayer.Play()
		log.Printf("[InputSystem] 播放种植音效")
	}

	// 触发植物卡片冷却
	s.triggerPlantCardCooldown(plantType)

	// 删除预览实体
	s.destroyPlantPreview()

	// 退出种植模式
	s.gameState.ExitPlantingMode()
	log.Printf("[InputSystem] 种植完成，退出种植模式")

	return true // 已处理点击
}

// createPlantEntity 创建植物实体的辅助方法
// 使用 entities.NewPlantEntity 工厂函数以保持一致性
func (s *InputSystem) createPlantEntity(plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	return entities.NewPlantEntity(s.entityManager, s.resourceManager, s.gameState, plantType, col, row)
}

// getPlantCost 获取植物的阳光消耗
func (s *InputSystem) getPlantCost(plantType components.PlantType) int {
	switch plantType {
	case components.PlantSunflower:
		return 50
	case components.PlantPeashooter:
		return 100
	default:
		return 0
	}
}

// triggerPlantCardCooldown 触发指定植物类型的卡片进入冷却状态
func (s *InputSystem) triggerPlantCardCooldown(plantType components.PlantType) {
	// 查询所有植物卡片实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantCardComponent{}),
	)

	// 找到匹配的卡片并触发冷却
	for _, entityID := range entities {
		cardComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PlantCardComponent{}))
		card := cardComp.(*components.PlantCardComponent)

		if card.PlantType == plantType {
			// 触发冷却
			card.CurrentCooldown = card.CooldownTime
			log.Printf("[InputSystem] 触发植物卡片冷却: PlantType=%v, CooldownTime=%.1f", plantType, card.CooldownTime)
			break // 只触发第一个匹配的卡片
		}
	}
}
