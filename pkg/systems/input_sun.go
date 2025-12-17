package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
)

// ========================================
// 阳光点击和悬停处理
// ========================================

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

// updateSunHover 更新阳光的悬停状态
// 检测鼠标是否悬停在可点击的阳光上，更新 ClickableComponent.IsHovered 状态
// 用于 updateMouseCursor() 读取状态以设置手形光标
func (s *InputSystem) updateSunHover(cameraX float64) {
	mouseScreenX, mouseScreenY := utils.GetPointerPosition()
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
