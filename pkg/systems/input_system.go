package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputSystem 处理所有用户输入，包括鼠标点击和键盘输入
type InputSystem struct {
	entityManager      *ecs.EntityManager
	resourceManager    *game.ResourceManager
	gameState          *game.GameState
	reanimSystem       entities.ReanimSystemInterface // Reanim 系统（用于初始化植物动画）
	sunCounterX        float64                        // 阳光计数器X坐标（收集动画目标）
	sunCounterY        float64                        // 阳光计数器Y坐标（收集动画目标）
	lawnGridSystem     *LawnGridSystem                // 草坪网格管理系统
	lawnGridEntityID   ecs.EntityID                   // 草坪网格实体ID
	lastBuzzerPlayTime float64                        // 上次播放无效操作音效的时间 (Story 10.8)
	buzzerCooldownTime float64                        // 无效操作音效冷却时间（秒）(Story 10.8)
	gameTime           float64                        // 游戏时间累计（秒）(Story 10.8)
	tooltipEntity      ecs.EntityID                   // Tooltip 实体ID (Story 10.8)
}

// NewInputSystem 创建一个新的输入系统
// 添加 reanimSystem 参数用于初始化植物动画
func NewInputSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, rs entities.ReanimSystemInterface, sunCounterX, sunCounterY float64, lawnGridSystem *LawnGridSystem, lawnGridEntityID ecs.EntityID) *InputSystem {
	system := &InputSystem{
		entityManager:      em,
		resourceManager:    rm,
		gameState:          gs,
		reanimSystem:       rs,
		sunCounterX:        sunCounterX,
		sunCounterY:        sunCounterY,
		lawnGridSystem:     lawnGridSystem,
		lawnGridEntityID:   lawnGridEntityID,
		buzzerCooldownTime: 0.5, // Story 10.8: 0.5秒冷却时间，防止连续点击播放多次
	}

	// 音效统一由 AudioManager 管理（Story 10.9）

	return system
}

// Update 处理用户输入
// 参数:
//   - deltaTime: 时间增量（秒）
//   - cameraX: 摄像机的世界坐标X位置（用于屏幕坐标到世界坐标的转换）
func (s *InputSystem) Update(deltaTime float64, cameraX float64) {
	// Story 10.8: 更新游戏时间（用于音效冷却）
	s.gameTime += deltaTime

	// ESC 键切换暂停/恢复
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.gameState.TogglePause()
		if s.gameState.IsPaused {
			log.Printf("[InputSystem] 游戏暂停 (ESC)")
			// 播放暂停音效
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_PAUSE")
			}
		} else {
			log.Printf("[InputSystem] 游戏恢复 (ESC)")
		}
		return // 处理暂停切换后立即返回，避免响应其他输入
	}

	// 暂停时屏蔽游戏世界交互
	if s.gameState.IsPaused {
		return // 暂停时不处理任何游戏输入
	}

	// 注意：植物预览位置现在由 PlantPreviewSystem 统一管理，无需在这里更新

	// Story 8.2.1: 更新植物卡片悬停状态（每帧检测）
	s.updatePlantCardHover()

	// 更新阳光悬停状态（每帧检测，用于手形光标）
	s.updateSunHover(cameraX)

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

	// DEBUG: 按 L 键在鼠标位置生成 Planting 粒子效果（测试种植粒子）
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		mouseScreenX, mouseScreenY := ebiten.CursorPosition()
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY)

		_, err := entities.NewPlantingParticleEffect(s.entityManager, s.resourceManager, mouseWorldX, mouseWorldY)
		if err != nil {
			log.Printf("[InputSystem] DEBUG: 生成种植粒子效果失败: %v", err)
		} else {
			log.Printf("[InputSystem] DEBUG: 在位置 (%.1f, %.1f) 生成种植粒子效果", mouseWorldX, mouseWorldY)
		}
	}

	// 处理植物选择快捷键（数字键 1-9）
	s.handlePlantCardHotkeys(cameraX)

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

		// 优先检查阳光点击（阳光可能飘到UI区域，优先级高于植物卡片）
		sunHandled := s.handleSunClickAt(mouseWorldX, mouseWorldY)
		if sunHandled {
			return // 已处理阳光点击，不继续处理其他点击
		}

		// 检查植物卡片点击（UI元素不受摄像机影响，使用屏幕坐标）
		cardHandled := s.handlePlantCardClick(mouseScreenX, mouseScreenY, cameraX)
		if cardHandled {
			return // 已处理卡片点击，不继续处理其他点击
		}

		// 检查是否在种植模式下点击草坪
		// 注意：handleLawnClick 内部会调用 MouseToGridCoords 进行坐标转换，所以传递屏幕坐标
		lawnHandled := s.handleLawnClick(mouseScreenX, mouseScreenY)
		if lawnHandled {
			return // 已处理草坪种植
		}
	}
}

// handleSunClickAt 检测并处理阳光点击
// 返回 true 表示处理了点击，false 表示未处理
func (s *InputSystem) handleSunClickAt(mouseWorldX, mouseWorldY float64) bool {
	// 查询所有可点击的阳光实体（使用世界坐标）
	sunEntities := ecs.GetEntitiesWith3[
		*components.PositionComponent,
		*components.ClickableComponent,
		*components.SunComponent,
	](s.entityManager)

	// 从后向前遍历，确保点击最上层的阳光（假设后面的实体渲染在上层）
	for i := len(sunEntities) - 1; i >= 0; i-- {
		id := sunEntities[i]

		// 获取组件
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, id)
		sun, _ := ecs.GetComponent[*components.SunComponent](s.entityManager, id)

		// 只处理可点击的阳光（允许下落中和已落地的阳光）
		if !clickable.IsEnabled {
			continue
		}

		// 不允许点击已被收集的阳光
		if sun.State == components.SunCollecting {
			continue
		}

		// 使用坐标转换工具库计算点击中心
		clickCenterX, clickCenterY, err := utils.GetClickableCenter(s.entityManager, id, pos)
		if err != nil {
			// 实体没有 ReanimComponent，使用 Position 作为默认中心
			clickCenterX = pos.X
			clickCenterY = pos.Y
		}

		halfWidth := clickable.Width / 2.0
		halfHeight := clickable.Height / 2.0

		if mouseWorldX >= clickCenterX-halfWidth && mouseWorldX <= clickCenterX+halfWidth &&
			mouseWorldY >= clickCenterY-halfHeight && mouseWorldY <= clickCenterY+halfHeight {
			// 点击命中！
			log.Printf("[InputSystem] ✓ 点击命中阳光! 鼠标=(%.1f, %.1f), 点击中心=(%.1f, %.1f)",
				mouseWorldX, mouseWorldY, clickCenterX, clickCenterY)
			s.handleSunClick(id, pos)
			return true // 已处理阳光点击
		}
	}

	return false // 未点击到任何阳光
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

	// 3. 播放收集音效（使用 AudioManager 统一管理 - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_POINTS")
		log.Printf("[InputSystem] 播放收集音效")
	}

	// 4. 移除 LifetimeComponent，防止收集过程中过期消失
	ecs.RemoveComponent[*components.LifetimeComponent](s.entityManager, sunID)

	// 5. 添加收集动画组件（缓动动画）
	// 注意：sunCounterX/Y 是屏幕坐标，需要转换为世界坐标
	// 世界坐标 = 屏幕坐标 + cameraX（仅X轴）
	targetWorldX := s.sunCounterX + s.gameState.CameraX
	targetWorldY := s.sunCounterY // Y轴不受摄像机影响

	ecs.AddComponent(s.entityManager, sunID, &components.SunCollectionAnimationComponent{
		StartX:   pos.X,
		StartY:   pos.Y,
		TargetX:  targetWorldX,
		TargetY:  targetWorldY,
		Progress: 0.0,
		Duration: 0.6, // 动画时长0.6秒（原速度600px/s，平均距离约360px）
	})

	// 6. 添加缩放组件（用于收集过程中的缩小效果）
	ecs.AddComponent(s.entityManager, sunID, &components.ScaleComponent{
		ScaleX: 1.0, // 初始缩放 = 1.0（原始大小）
		ScaleY: 1.0,
	})

	// 注意: 阳光数值会在 SunCollectionSystem 检测到阳光到达目标位置时增加
	// 这样可以确保视觉效果（阳光飞行 → 到达 → 数值增加）的正确时序
	// 运动逻辑由 SunMovementSystem 根据 SunCollectionAnimationComponent 计算缓动位置和缩放
}

// handlePlantCardClick 处理植物卡片点击逻辑
// 返回 true 表示处理了点击，false 表示未处理
// Story 19.3: 强引导模式下阻止卡片点击
func (s *InputSystem) handlePlantCardClick(mouseX, mouseY int, cameraX float64) bool {
	// Story 19.3: 检查强引导模式是否阻止卡片点击
	if IsGuidedTutorialBlocking("click_plant_card") {
		return false // 静默忽略，不处理卡片点击
	}
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
		// 跳过奖励卡片（奖励卡片不应被点击选择）
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		log.Printf("[InputSystem] 卡片 %d: PlantType=%v, 位置=(%.1f, %.1f), 可点击区域=%.1fx%.1f, IsAvailable=%v, IsEnabled=%v",
			entityID, card.PlantType, pos.X, pos.Y, clickable.Width, clickable.Height, card.IsAvailable, clickable.IsEnabled)

		// AABB 碰撞检测（先检测点击，再判断卡片状态）
		if float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+clickable.Width &&
			float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+clickable.Height {
			// 点击命中卡片！
			log.Printf("[InputSystem] 点击植物卡片: PlantType=%v, IsPlantingMode=%v, IsAvailable=%v",
				card.PlantType, s.gameState.IsPlantingMode, card.IsAvailable)

			// Story 10.8: 检查卡片状态
			currentSun := s.gameState.GetSun()
			plantCost := card.SunCost

			// 1. 冷却中 - 不做任何反应
			if card.CurrentCooldown > 0 {
				log.Printf("[InputSystem] 卡片冷却中，跳过 (剩余 %.1f 秒)", card.CurrentCooldown)
				return true // 已处理点击，但卡片冷却中
			}

			// 2. 阳光不足 - 触发闪烁反馈
			if currentSun < plantCost {
				log.Printf("[InputSystem] 阳光不足: 需要 %d, 当前 %d", plantCost, currentSun)
				s.gameState.TriggerSunFlash()

				// Story 10.8: 播放无效操作音效（带冷却，使用 AudioManager - Story 10.9）
				timeSinceLastBuzzer := s.gameTime - s.lastBuzzerPlayTime
				if timeSinceLastBuzzer >= s.buzzerCooldownTime {
					if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
						audioManager.PlaySound("SOUND_BUZZER")
						s.lastBuzzerPlayTime = s.gameTime
						log.Printf("[InputSystem] 播放无效操作音效 (buzzer)")
					}
				} else {
					log.Printf("[InputSystem] 无效操作音效冷却中 (%.2f/%.2f)", timeSinceLastBuzzer, s.buzzerCooldownTime)
				}

				return true // 已处理点击，但阻止卡片选择
			}

			// 3. 其他不可用原因 - 跳过
			if !clickable.IsEnabled {
				log.Printf("[InputSystem] 卡片不可点击，跳过")
				return true
			}

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

				// Story 10.9: 播放选中植物卡片音效 (seedlift.ogg)
				if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
					audioManager.PlaySound("SOUND_SEEDLIFT")
				}

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

// createPlantPreview 创建植物预览实体（复用植物卡片的渲染逻辑）
// 使用与植物卡片相同的 Reanim 离屏渲染技术,确保预览和卡片显示一致
func (s *InputSystem) createPlantPreview(plantType components.PlantType, x, y float64) {
	// 先删除现有预览
	s.destroyPlantPreview()

	// 使用 RenderPlantIcon 复用卡片渲染逻辑（直接传入 plantType）
	plantIcon, err := entities.RenderPlantIcon(s.entityManager, s.resourceManager, s.reanimSystem, plantType)
	if err != nil {
		log.Printf("[InputSystem] Failed to render plant icon for preview: %v", err)
		return
	}

	// 创建预览实体
	entityID := s.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(s.entityManager, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加静态精灵组件（使用渲染好的图标）
	ecs.AddComponent(s.entityManager, entityID, &components.SpriteComponent{
		Image: plantIcon,
	})

	// 添加植物预览组件
	ecs.AddComponent(s.entityManager, entityID, &components.PlantPreviewComponent{
		PlantType: plantType,
		Alpha:     0.5, // 半透明效果
	})

	log.Printf("[InputSystem] Created plant preview (ID: %d, Type: %v) at (%.1f, %.1f)",
		entityID, plantType, x, y)
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
// Story 19.3: 强引导模式下阻止草坪种植
func (s *InputSystem) handleLawnClick(mouseX, mouseY int) bool {
	// Story 19.3: 检查强引导模式是否阻止草坪点击（种植）
	if IsGuidedTutorialBlocking("click_lawn_empty") {
		return false // 静默忽略，不处理草坪点击
	}

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

	// 检查该行是否启用（教学关卡可能禁用部分行）
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

	// 触发种植粒子效果
	worldX, worldY := utils.GridToWorldCoords(
		col, row,
		config.GridWorldStartX, config.GridWorldStartY,
		config.CellWidth, config.CellHeight,
	)
	_, err = entities.NewPlantingParticleEffect(s.entityManager, s.resourceManager, worldX, worldY)
	if err != nil {
		log.Printf("[InputSystem] 警告：创建种植粒子效果失败: %v", err)
		// 不阻塞游戏逻辑，继续进行
	} else {
		log.Printf("[InputSystem] 触发种植粒子效果，位置: (%.1f, %.1f)", worldX, worldY)
	}

	// 标记格子为占用
	err = s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, col, row, plantID)
	if err != nil {
		log.Printf("[InputSystem] 标记格子占用失败: %v", err)
		// 失败时删除植物实体并返还阳光
		s.entityManager.DestroyEntity(plantID)
		s.gameState.AddSun(sunCost)
		return true
	}

	// 播放种植音效（使用 AudioManager 统一管理 - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_PLANT")
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
	// 传递 reanimSystem 给工厂函数以初始化动画
	// 坚果墙和樱桃炸弹使用专用的工厂函数
	if plantType == components.PlantWallnut {
		return entities.NewWallnutEntity(s.entityManager, s.resourceManager, s.gameState, s.reanimSystem, col, row)
	}
	if plantType == components.PlantCherryBomb {
		return entities.NewCherryBombEntity(s.entityManager, s.resourceManager, s.gameState, col, row)
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

// updatePlantCardHover 更新植物卡片的悬停状态
// Story 8.2.1: 每帧检测鼠标是否悬停在植物卡片上，设置 UIComponent.State 为 UIHovered
// Story 10.8: 添加 Tooltip 显示和鼠标光标切换
func (s *InputSystem) updatePlantCardHover() {
	mouseX, mouseY := ebiten.CursorPosition()

	// 查询所有植物卡片实体
	entities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// 记录是否有卡片被悬停
	hoveredEntityID := ecs.EntityID(0)
	var hoveredCard *components.PlantCardComponent

	// 遍历所有卡片，检测悬停
	for _, entityID := range entities {
		// 跳过奖励卡片
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		// AABB 碰撞检测
		isHovering := float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+clickable.Width &&
			float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+clickable.Height

		// Story 10.8: 悬停任何卡片都显示 Tooltip（不管是否可用）
		if isHovering {
			hoveredEntityID = entityID
			hoveredCard = card

			// 只有可用的卡片才显示悬停效果（UI状态变化）
			if card.IsAvailable && clickable.IsEnabled {
				// 只有当前状态不是 Clicked 时才设置为 Hovered（避免覆盖点击状态）
				if ui.State != components.UIClicked {
					ui.State = components.UIHovered
				}
			}
			break // 找到悬停卡片后停止遍历
		} else {
			// 如果不再悬停且当前是 Hovered 状态，恢复为 Normal
			if ui.State == components.UIHovered {
				if card.IsAvailable {
					ui.State = components.UINormal
				} else {
					ui.State = components.UIDisabled
				}
			}
		}
	}

	// Story 10.8: 更新 Tooltip（鼠标光标由 GameScene.updateMouseCursor() 统一管理）
	if hoveredEntityID != 0 {
		// 有卡片被悬停: 显示 Tooltip
		s.showTooltip(hoveredEntityID, hoveredCard)
	} else {
		// 没有卡片被悬停: 隐藏 Tooltip
		s.hideTooltip()
	}
}

// showTooltip 显示 Tooltip
// Story 10.8: 鼠标悬停植物卡片时显示提示信息
func (s *InputSystem) showTooltip(cardEntityID ecs.EntityID, card *components.PlantCardComponent) {
	// 获取或创建 Tooltip 实体
	if s.tooltipEntity == 0 {
		s.tooltipEntity = s.entityManager.CreateEntity()
		tooltip := components.NewTooltipComponent()
		s.entityManager.AddComponent(s.tooltipEntity, tooltip)
	}

	// 获取 Tooltip 组件
	tooltip, ok := ecs.GetComponent[*components.TooltipComponent](s.entityManager, s.tooltipEntity)
	if !ok {
		log.Printf("[InputSystem] 警告: Tooltip 组件不存在")
		return
	}

	// 获取卡片位置信息
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, cardEntityID)
	if !ok {
		return
	}
	clickable, ok := ecs.GetComponent[*components.ClickableComponent](s.entityManager, cardEntityID)
	if !ok {
		return
	}

	// 更新 Tooltip 内容
	tooltip.IsVisible = true
	tooltip.TargetEntity = cardEntityID
	tooltip.PlantName = s.getPlantName(card.PlantType)

	// 设置状态提示文本
	// 优先级：冷却中 > 阳光不足 > 可用
	currentSun := s.gameState.GetSun()
	if card.CurrentCooldown > 0 {
		tooltip.StatusText = "重新装填中..."
	} else if currentSun < card.SunCost {
		tooltip.StatusText = "没有足够的阳光"
	} else {
		tooltip.StatusText = "" // 可用状态不显示状态提示
	}

	// 计算 Tooltip 位置（卡片下方居中，因为卡片在屏幕顶部）
	// tooltip.X: 卡片中心 X 坐标
	// tooltip.Y: 卡片底部 Y 坐标
	tooltip.X = pos.X + clickable.Width/2
	tooltip.Y = pos.Y + clickable.Height // 卡片底部 Y 坐标
}

// hideTooltip 隐藏 Tooltip
// Story 10.8: 鼠标离开卡片时隐藏提示信息
func (s *InputSystem) hideTooltip() {
	if s.tooltipEntity == 0 {
		return
	}

	tooltip, ok := ecs.GetComponent[*components.TooltipComponent](s.entityManager, s.tooltipEntity)
	if ok {
		tooltip.IsVisible = false
	}
}

// isCardClickable 检测卡片是否可点击
// Story 10.8: 判断卡片是否处于可点击状态（决定鼠标光标样式）
func (s *InputSystem) isCardClickable(card *components.PlantCardComponent) bool {
	return card.IsAvailable && card.CurrentCooldown <= 0 && s.gameState.Sun >= card.SunCost
}

// getPlantName 获取植物名称
// Story 10.8: 根据植物类型返回中文名称
func (s *InputSystem) getPlantName(plantType components.PlantType) string {
	switch plantType {
	case components.PlantPeashooter:
		return "豌豆射手"
	case components.PlantSunflower:
		return "向日葵"
	case components.PlantCherryBomb:
		return "樱桃炸弹"
	case components.PlantWallnut:
		return "坚果墙"
	default:
		return "未知植物"
	}
}

// handlePlantCardHotkeys 处理植物卡片快捷键（数字键 1-9）
// 按卡片在卡槽中的位置从左到右对应数字键 1-9
func (s *InputSystem) handlePlantCardHotkeys(cameraX float64) {
	// 定义快捷键映射
	hotkeys := []ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3,
		ebiten.Key4, ebiten.Key5, ebiten.Key6,
		ebiten.Key7, ebiten.Key8, ebiten.Key9,
	}

	// 检测按下的数字键
	pressedIndex := -1
	for i, key := range hotkeys {
		if inpututil.IsKeyJustPressed(key) {
			pressedIndex = i
			break
		}
	}

	if pressedIndex < 0 {
		return // 没有按下快捷键
	}

	// 查询所有植物卡片实体，按 X 坐标排序
	cardEntities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// 过滤并收集非奖励卡片
	type cardInfo struct {
		entityID ecs.EntityID
		posX     float64
		card     *components.PlantCardComponent
		ui       *components.UIComponent
	}
	var cards []cardInfo

	for _, entityID := range cardEntities {
		// 跳过奖励卡片
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		cards = append(cards, cardInfo{
			entityID: entityID,
			posX:     pos.X,
			card:     card,
			ui:       ui,
		})
	}

	// 按 X 坐标排序（从左到右）
	for i := 0; i < len(cards)-1; i++ {
		for j := i + 1; j < len(cards); j++ {
			if cards[i].posX > cards[j].posX {
				cards[i], cards[j] = cards[j], cards[i]
			}
		}
	}

	// 检查索引是否有效
	if pressedIndex >= len(cards) {
		log.Printf("[InputSystem] 快捷键 %d 无效：只有 %d 张卡片", pressedIndex+1, len(cards))
		return
	}

	// 获取对应卡片
	targetCard := cards[pressedIndex]

	log.Printf("[InputSystem] 快捷键选择卡片: 按键=%d, 植物类型=%v", pressedIndex+1, targetCard.card.PlantType)

	// 检查卡片状态（与鼠标点击逻辑保持一致）
	// 1. 冷却中 - 不做任何反应
	if targetCard.card.CurrentCooldown > 0 {
		log.Printf("[InputSystem] 卡片冷却中，跳过 (剩余 %.1f 秒)", targetCard.card.CurrentCooldown)
		return
	}

	// 2. 阳光不足 - 触发闪烁反馈
	currentSun := s.gameState.GetSun()
	plantCost := targetCard.card.SunCost
	if currentSun < plantCost {
		log.Printf("[InputSystem] 阳光不足: 需要 %d, 当前 %d", plantCost, currentSun)
		s.gameState.TriggerSunFlash()

		// 播放无效操作音效（带冷却）
		timeSinceLastBuzzer := s.gameTime - s.lastBuzzerPlayTime
		if timeSinceLastBuzzer >= s.buzzerCooldownTime {
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_BUZZER")
				s.lastBuzzerPlayTime = s.gameTime
			}
		}
		return
	}

	// 3. 卡片不可用 - 跳过
	if !targetCard.card.IsAvailable {
		log.Printf("[InputSystem] 卡片不可用，跳过")
		return
	}

	// 检查当前是否已在种植模式
	if s.gameState.IsPlantingMode {
		// 如果选择的是同一种植物，退出种植模式
		_, currentPlantType := s.gameState.GetPlantingMode()
		if currentPlantType == targetCard.card.PlantType {
			log.Printf("[InputSystem] 快捷键退出种植模式（重复选择同一卡片）")
			s.gameState.ExitPlantingMode()
			s.destroyPlantPreview()
			targetCard.ui.State = components.UINormal
			return
		}
		// 如果选择不同植物，切换到新植物
		log.Printf("[InputSystem] 快捷键切换种植模式: %v -> %v", currentPlantType, targetCard.card.PlantType)
		s.destroyPlantPreview()
	}

	// 进入种植模式
	log.Printf("[InputSystem] 快捷键进入种植模式: PlantType=%v", targetCard.card.PlantType)
	s.gameState.EnterPlantingMode(targetCard.card.PlantType)

	// 播放选中植物卡片音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SEEDLIFT")
	}

	// 创建植物预览实体（使用鼠标位置）
	mouseX, mouseY := ebiten.CursorPosition()
	mouseWorldX := float64(mouseX) + cameraX
	mouseWorldY := float64(mouseY)
	s.createPlantPreview(targetCard.card.PlantType, mouseWorldX, mouseWorldY)

	// 设置卡片状态为 Clicked（视觉反馈）
	targetCard.ui.State = components.UIClicked
}

// updateSunHover 更新阳光的悬停状态
// 检测鼠标是否悬停在可点击的阳光上，更新 ClickableComponent.IsHovered 状态
// 用于 updateMouseCursor() 读取状态以设置手形光标
func (s *InputSystem) updateSunHover(cameraX float64) {
	mouseScreenX, mouseScreenY := ebiten.CursorPosition()
	mouseWorldX := float64(mouseScreenX) + cameraX
	mouseWorldY := float64(mouseScreenY)

	// 查询所有可点击的阳光实体
	sunEntities := ecs.GetEntitiesWith3[
		*components.PositionComponent,
		*components.ClickableComponent,
		*components.SunComponent,
	](s.entityManager)

	for _, id := range sunEntities {
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, id)
		sun, _ := ecs.GetComponent[*components.SunComponent](s.entityManager, id)

		// 跳过不可点击或正在被收集的阳光
		if !clickable.IsEnabled || sun.State == components.SunCollecting {
			clickable.IsHovered = false
			continue
		}

		// 使用坐标转换工具库计算点击中心
		clickCenterX, clickCenterY, err := utils.GetClickableCenter(s.entityManager, id, pos)
		if err != nil {
			clickCenterX = pos.X
			clickCenterY = pos.Y
		}

		halfWidth := clickable.Width / 2.0
		halfHeight := clickable.Height / 2.0

		// 检测悬停
		isHovered := mouseWorldX >= clickCenterX-halfWidth && mouseWorldX <= clickCenterX+halfWidth &&
			mouseWorldY >= clickCenterY-halfHeight && mouseWorldY <= clickCenterY+halfHeight

		clickable.IsHovered = isHovered
	}
}
