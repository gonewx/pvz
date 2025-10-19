package systems

import (
	"log"
	"math"

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
	reanimSystem       entities.ReanimSystemInterface // Story 6.3: Reanim 系统（用于初始化植物动画）
	sunCounterX        float64                        // 阳光计数器X坐标（收集动画目标）
	sunCounterY        float64                        // 阳光计数器Y坐标（收集动画目标）
	collectSoundPlayer *audio.Player                  // 收集阳光音效播放器
	lawnGridSystem     *LawnGridSystem                // 草坪网格管理系统
	lawnGridEntityID   ecs.EntityID                   // 草坪网格实体ID
	plantSoundPlayer   *audio.Player                  // 种植音效播放器
}

// NewInputSystem 创建一个新的输入系统
// Story 6.3: 添加 reanimSystem 参数用于初始化植物动画
func NewInputSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, rs entities.ReanimSystemInterface, sunCounterX, sunCounterY float64, lawnGridSystem *LawnGridSystem, lawnGridEntityID ecs.EntityID) *InputSystem {
	system := &InputSystem{
		entityManager:    em,
		resourceManager:  rm,
		gameState:        gs,
		reanimSystem:     rs,
		sunCounterX:      sunCounterX,
		sunCounterY:      sunCounterY,
		lawnGridSystem:   lawnGridSystem,
		lawnGridEntityID: lawnGridEntityID,
	}

	// 加载收集阳光音效（使用 LoadSoundEffect 而非 LoadAudio 以避免循环播放）
	// Note: Using hardcoded path as sound resource ID loading not yet implemented
	player, err := rm.LoadSoundEffect("assets/sounds/points.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load sun collect sound: %v", err)
	} else {
		system.collectSoundPlayer = player
	}

	// 加载种植音效
	// Note: Using hardcoded path as sound resource ID loading not yet implemented
	plantPlayer, err := rm.LoadSoundEffect("assets/sounds/plant.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load plant sound: %v", err)
	} else {
		system.plantSoundPlayer = plantPlayer
	}

	return system
}

// Update 处理用户输入
// 参数:
//   - deltaTime: 时间增量（秒）
//   - cameraX: 摄像机的世界坐标X位置（用于屏幕坐标到世界坐标的转换）
func (s *InputSystem) Update(deltaTime float64, cameraX float64) {
	// 注意：植物预览位置现在由 PlantPreviewSystem 统一管理，无需在这里更新

	// DEBUG: 按 P 键在鼠标位置生成 PeaSplat 粒子效果（测试用）
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		mouseScreenX, mouseScreenY := ebiten.CursorPosition()
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY)

		_, err := entities.CreateParticleEffect(s.entityManager, s.resourceManager, "PeaSplat", mouseWorldX, mouseWorldY)
		if err != nil {
			log.Printf("[InputSystem] DEBUG: 生成粒子效果失败: %v", err)
		} else {
			log.Printf("[InputSystem] DEBUG: 在位置 (%.1f, %.1f) 生成 PeaSplat 粒子效果", mouseWorldX, mouseWorldY)
		}
	}

	// DEBUG: 按 B 键在鼠标位置生成 BossExplosion 粒子效果（测试用）
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		mouseScreenX, mouseScreenY := ebiten.CursorPosition()
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY)

		_, err := entities.CreateParticleEffect(s.entityManager, s.resourceManager, "BossExplosion", mouseWorldX, mouseWorldY)
		if err != nil {
			log.Printf("[InputSystem] DEBUG: 生成粒子效果失败: %v", err)
		} else {
			log.Printf("[InputSystem] DEBUG: 在位置 (%.1f, %.1f) 生成 BossExplosion 粒子效果", mouseWorldX, mouseWorldY)
		}
	}

	// DEBUG: 按 A 键在鼠标位置生成 Award 粒子效果（测试用）
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		mouseScreenX, mouseScreenY := ebiten.CursorPosition()
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY)

		_, err := entities.CreateParticleEffect(s.entityManager, s.resourceManager, "Award", mouseWorldX, mouseWorldY)
		if err != nil {
			log.Printf("[InputSystem] DEBUG: 生成粒子效果失败: %v", err)
		} else {
			log.Printf("[InputSystem] DEBUG: 在位置 (%.1f, %.1f) 生成 Award 粒子效果", mouseWorldX, mouseWorldY)
		}
	}

	// DEBUG: 按 Z 键在鼠标位置生成 ZombieHead 粒子效果（测试用）
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		mouseScreenX, mouseScreenY := ebiten.CursorPosition()
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY)

		_, err := entities.CreateParticleEffect(s.entityManager, s.resourceManager, "ZombieHead", mouseWorldX, mouseWorldY, 0.0)
		if err != nil {
			log.Printf("[InputSystem] DEBUG: 生成粒子效果失败: %v", err)
		} else {
			log.Printf("[InputSystem] DEBUG: 在位置 (%.1f, %.1f) 生成 ZombieHead 粒子效果", mouseWorldX, mouseWorldY)
		}
	}

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
		mouseScreenX, mouseScreenY := ebiten.CursorPosition()

		// 将鼠标屏幕坐标转换为世界坐标
		// worldX = screenX + cameraX (摄像机向右移动时，世界坐标增大)
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY) // Y轴不受摄像机水平移动影响

		// DEBUG: 鼠标点击日志（每次点击都打印会刷屏，已禁用）
		// log.Printf("[InputSystem] 鼠标点击: 屏幕(%d, %d) -> 世界(%.1f, %.1f)",
		// 	mouseScreenX, mouseScreenY, mouseWorldX, mouseWorldY)

		// 优先检查植物卡片点击（UI元素不受摄像机影响，使用屏幕坐标）
		cardHandled := s.handlePlantCardClick(mouseScreenX, mouseScreenY, cameraX)
		if cardHandled {
			return // 已处理卡片点击，不继续处理其他点击
		}

		// 检查是否在种植模式下点击草坪
		// 注意：handleLawnClick 内部会调用 MouseToGridCoords 进行坐标转换，所以传递屏幕坐标
		lawnHandled := s.handleLawnClick(mouseScreenX, mouseScreenY)
		if lawnHandled {
			return // 已处理草坪种植，不继续处理阳光
		}

		// 查询所有可点击的阳光实体（使用世界坐标）
		entities := ecs.GetEntitiesWith3[
			*components.PositionComponent,
			*components.ClickableComponent,
			*components.SunComponent,
		](s.entityManager)

		// DEBUG: 阳光实体数量日志（每次点击都打印会刷屏，已禁用）
		// log.Printf("[InputSystem] 找到 %d 个阳光实体", len(entities))

		// 从后向前遍历，确保点击最上层的阳光（假设后面的实体渲染在上层）
		for i := len(entities) - 1; i >= 0; i-- {
			id := entities[i]

			// 获取组件
			pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
			clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, id)
			sun, _ := ecs.GetComponent[*components.SunComponent](s.entityManager, id)
			reanim, _ := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)

			// 只处理可点击的阳光（允许下落中和已落地的阳光）
			if !clickable.IsEnabled {
				continue
			}

			// 不允许点击已被收集的阳光
			if sun.State == components.SunCollecting {
				continue
			}

			// 计算点击检测的中心位置
			//
			// 问题分析：阳光的视觉中心（Sun1 核心）和几何中心不一致
			// - Sun.reanim 有3个部件：Sun1(小核心), Sun2(中), Sun3(大光晕)
			// - Sun1 (36x36) 在 (-14.2, -14.2)，是视觉上的"阳光核心"
			// - Sun3 (117x116) 在 (16, -70.6)，是大光晕，占据右侧大量空间
			// - calculateCenterOffset 计算的是几何中心 ≈ (44.2, -19.4)
			// - 但玩家点击的是视觉中心（Sun1），而不是几何中心
			//
			// 修复方案：点击中心应该对齐到实际的视觉中心（渲染后的中心）
			// - 渲染原点 = pos - CenterOffset
			// - Sun1 中心 ≈ 渲染原点 + 0 (Sun1 的中心接近部件坐标原点)
			// - 因此点击中心 = pos - CenterOffset (而不是 pos)
			//
			// 但是这会让所有使用 Reanim 的实体都偏移，只有阳光需要特殊处理
			// 因为阳光的部件分布不均匀（Sun3 拉偏了几何中心）
			//
			// 临时方案：如果有 Reanim，点击中心向左偏移 CenterOffset
			clickCenterX := pos.X
			clickCenterY := pos.Y

			// 修复阳光点击偏移：考虑 CenterOffset
			if reanim != nil && sun != nil {
				// 对于阳光，点击中心应该在渲染原点附近（Sun1 的位置）
				// 而不是 pos（几何中心）
				clickCenterX = pos.X - reanim.CenterOffsetX
				clickCenterY = pos.Y - reanim.CenterOffsetY
				log.Printf("[InputSystem] 阳光 %d: 调整点击中心 pos=(%.1f, %.1f) -> click=(%.1f, %.1f), offset=(%.1f, %.1f)",
					id, pos.X, pos.Y, clickCenterX, clickCenterY, reanim.CenterOffsetX, reanim.CenterOffsetY)
			}

			halfWidth := clickable.Width / 2.0
			halfHeight := clickable.Height / 2.0

			if mouseWorldX >= clickCenterX-halfWidth && mouseWorldX <= clickCenterX+halfWidth &&
				mouseWorldY >= clickCenterY-halfHeight && mouseWorldY <= clickCenterY+halfHeight {
				// 点击命中！
				log.Printf("[InputSystem] ✓ 点击命中阳光! 鼠标=(%.1f, %.1f), 点击中心=(%.1f, %.1f)",
					mouseWorldX, mouseWorldY, clickCenterX, clickCenterY)
				s.handleSunClick(id, pos)
				break // 只处理第一个命中的阳光
			}
		}
	}
}

// handleSunClick 处理阳光被点击的逻辑
func (s *InputSystem) handleSunClick(sunID ecs.EntityID, pos *components.PositionComponent) {
	// 1. 更新阳光状态为收集中
	sun, _ := ecs.GetComponent[*components.SunComponent](s.entityManager, sunID)
	sun.State = components.SunCollecting
	log.Printf("[InputSystem] 阳光开始收集动画")

	// 2. 禁用点击，防止重复点击
	clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, sunID)
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
	ecs.RemoveComponent[*components.LifetimeComponent](s.entityManager, sunID)

	// 5. 计算飞向阳光计数器的速度向量
	// 注意：sunCounterX/Y 是屏幕坐标，需要转换为世界坐标
	// 世界坐标 = 屏幕坐标 + cameraX（仅X轴）
	targetWorldX := s.sunCounterX + s.gameState.CameraX
	targetWorldY := s.sunCounterY // Y轴不受摄像机影响

	dx := targetWorldX - pos.X
	dy := targetWorldY - pos.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// 飞行速度: 600像素/秒
	speed := 600.0
	vx := (dx / distance) * speed
	vy := (dy / distance) * speed

	// 6. 更新 VelocityComponent（如果不存在则添加）
	vel, exists := ecs.GetComponent[*components.VelocityComponent](s.entityManager, sunID)
	if exists {
		// 更新现有速度
		vel.VX = vx
		vel.VY = vy
	} else {
		// 添加新的速度组件（理论上应该已存在，但防御性编程）
		ecs.AddComponent(s.entityManager, sunID, &components.VelocityComponent{
			VX: vx,
			VY: vy,
		})
	}

	// 注意: 阳光数值会在 SunCollectionSystem 检测到阳光到达目标位置时增加
	// 这样可以确保视觉效果（阳光飞行 → 到达 → 数值增加）的正确时序
}

// handlePlantCardClick 处理植物卡片点击逻辑
// 返回 true 表示处理了点击，false 表示未处理
func (s *InputSystem) handlePlantCardClick(mouseX, mouseY int, cameraX float64) bool {
	// 查询所有植物卡片实体
	entities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// DEBUG: 卡片点击检查日志（每次点击都打印会刷屏，已禁用）
	// log.Printf("[InputSystem] 检查植物卡片点击: 鼠标(%d, %d), 找到 %d 个卡片", mouseX, mouseY, len(entities))

	// 遍历卡片实体，检测点击
	for _, entityID := range entities {
		// 获取组件
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

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

				// 创建植物预览实体（转换为世界坐标）
				mouseWorldX := float64(mouseX) + cameraX
				mouseWorldY := float64(mouseY)
				s.createPlantPreview(card.PlantType, mouseWorldX, mouseWorldY)

				// 可选：设置卡片状态为 Clicked（视觉反馈）
				ui.State = components.UIClicked
			}

			return true // 已处理点击
		}
	}

	return false // 未处理任何卡片点击
}

// createPlantPreview 创建植物预览实体（使用 Reanim）
func (s *InputSystem) createPlantPreview(plantType components.PlantType, x, y float64) {
	// 先删除现有预览
	s.destroyPlantPreview()

	// 获取植物对应的 Reanim 资源名称
	var reanimName string
	switch plantType {
	case components.PlantSunflower:
		reanimName = "SunFlower"
	case components.PlantPeashooter:
		reanimName = "PeaShooter"
	case components.PlantWallnut:
		reanimName = "Wallnut"
	case components.PlantCherryBomb:
		reanimName = "CherryBomb"
	default:
		log.Printf("[InputSystem] Unknown plant type for preview: %v", plantType)
		return
	}

	// 从 ResourceManager 获取 Reanim 数据和部件图片
	reanimXML := s.resourceManager.GetReanimXML(reanimName)
	partImages := s.resourceManager.GetReanimPartImages(reanimName)

	if reanimXML == nil || partImages == nil {
		log.Printf("[InputSystem] Failed to load Reanim resources for preview: %s", reanimName)
		return
	}

	// 创建预览实体
	entityID := s.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(s.entityManager, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加 ReanimComponent
	ecs.AddComponent(s.entityManager, entityID, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
	})

	// 添加植物预览组件
	ecs.AddComponent(s.entityManager, entityID, &components.PlantPreviewComponent{
		PlantType: plantType,
		Alpha:     0.5, // 半透明效果
	})

	// 初始化动画
	var animName string
	if plantType == components.PlantPeashooter {
		animName = "anim_full_idle" // 豌豆射手使用完整待机动画
	} else {
		animName = "anim_idle"
	}

	if err := s.reanimSystem.PlayAnimation(entityID, animName); err != nil {
		log.Printf("[InputSystem] Failed to play preview animation: %v", err)
		// 不删除实体，让它以静态方式显示
	}

	log.Printf("[InputSystem] Created plant preview (ID: %d, Type: %v, Reanim: %s) at (%.1f, %.1f)",
		entityID, plantType, reanimName, x, y)
}

// destroyPlantPreview 删除所有植物预览实体
func (s *InputSystem) destroyPlantPreview() {
	// 查询所有拥有 PlantPreviewComponent 的实体
	entities := ecs.GetEntitiesWith1[*components.PlantPreviewComponent](s.entityManager)

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
		// DEBUG: 网格外点击日志（已禁用避免刷屏）
		// log.Printf("[InputSystem] 鼠标点击在网格外: (%d, %d)", mouseX, mouseY)
		return false // 点击在网格外
	}

	// DEBUG: 草坪点击日志（只在种植时保留，已优化）
	// log.Printf("[InputSystem] 草坪点击: col=%d, row=%d", col, row)

	// Story 8.1: 检查该行是否启用（教学关卡可能禁用部分行）
	// 注意：row 是 0-based (0-4)，IsLaneEnabled 使用 1-based (1-5)
	lane := row + 1
	if !s.lawnGridSystem.IsLaneEnabled(lane) {
		log.Printf("[InputSystem] 行 %d 已被禁用，无法种植", lane)
		return true // 处理了点击，但该行禁用
	}

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

	// 重置植物卡片的选择状态
	s.resetPlantCardSelection(plantType)

	// 退出种植模式
	s.gameState.ExitPlantingMode()
	log.Printf("[InputSystem] 种植完成，退出种植模式")

	return true // 已处理点击
}

// createPlantEntity 创建植物实体的辅助方法
// 根据植物类型选择合适的工厂函数
func (s *InputSystem) createPlantEntity(plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	// Story 6.3: 传递 reanimSystem 给工厂函数以初始化动画
	// 坚果墙和樱桃炸弹使用专用的工厂函数
	if plantType == components.PlantWallnut {
		return entities.NewWallnutEntity(s.entityManager, s.resourceManager, s.gameState, s.reanimSystem, col, row)
	}
	if plantType == components.PlantCherryBomb {
		return entities.NewCherryBombEntity(s.entityManager, s.resourceManager, s.gameState, s.reanimSystem, col, row)
	}
	// 其他植物使用通用工厂函数
	return entities.NewPlantEntity(s.entityManager, s.resourceManager, s.gameState, s.reanimSystem, plantType, col, row)
}

// getPlantCost 获取植物的阳光消耗
func (s *InputSystem) getPlantCost(plantType components.PlantType) int {
	switch plantType {
	case components.PlantSunflower:
		return config.SunflowerSunCost // 50
	case components.PlantPeashooter:
		return config.PeashooterSunCost // 100
	case components.PlantWallnut:
		return config.WallnutCost // 50
	case components.PlantCherryBomb:
		return config.CherryBombSunCost // 150
	default:
		return 0
	}
}

// triggerPlantCardCooldown 触发指定植物类型的卡片进入冷却状态
func (s *InputSystem) triggerPlantCardCooldown(plantType components.PlantType) {
	// 查询所有植物卡片实体
	entities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)

	// 找到匹配的卡片并触发冷却
	for _, entityID := range entities {
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)

		if card.PlantType == plantType {
			// 触发冷却
			card.CurrentCooldown = card.CooldownTime
			log.Printf("[InputSystem] 触发植物卡片冷却: PlantType=%v, CooldownTime=%.1f", plantType, card.CooldownTime)
			break // 只触发第一个匹配的卡片
		}
	}
}

// resetPlantCardSelection 重置指定植物类型的卡片选择状态
func (s *InputSystem) resetPlantCardSelection(plantType components.PlantType) {
	// 查询所有植物卡片实体
	entities := ecs.GetEntitiesWith2[
		*components.PlantCardComponent,
		*components.UIComponent,
	](s.entityManager)

	// 找到匹配的卡片并重置UI状态
	for _, entityID := range entities {
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		if card.PlantType == plantType {
			// 重置为正常状态
			ui.State = components.UINormal
			log.Printf("[InputSystem] 重置植物卡片选择状态: PlantType=%v", plantType)
			break // 只重置第一个匹配的卡片
		}
	}
}
