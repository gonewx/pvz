package systems

import (
	"log"
	"math"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputSystem 处理所有用户输入，包括鼠标点击和键盘输入
type InputSystem struct {
	entityManager      *ecs.EntityManager
	resourceManager    *game.ResourceManager
	gameState          *game.GameState
	sunCounterX        float64       // 阳光计数器X坐标（收集动画目标）
	sunCounterY        float64       // 阳光计数器Y坐标（收集动画目标）
	collectSoundPlayer *audio.Player // 收集阳光音效播放器
}

// NewInputSystem 创建一个新的输入系统
func NewInputSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, sunCounterX, sunCounterY float64) *InputSystem {
	system := &InputSystem{
		entityManager:   em,
		resourceManager: rm,
		gameState:       gs,
		sunCounterX:     sunCounterX,
		sunCounterY:     sunCounterY,
	}

	// 加载收集阳光音效（使用 LoadSoundEffect 而非 LoadAudio 以避免循环播放）
	player, err := rm.LoadSoundEffect("assets/audio/Sound/points.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load sun collect sound: %v", err)
	} else {
		system.collectSoundPlayer = player
	}

	return system
}

// Update 处理用户输入
func (s *InputSystem) Update(deltaTime float64) {
	// 检测鼠标左键是否刚被按下
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		log.Printf("[InputSystem] 鼠标点击: (%d, %d)", mouseX, mouseY)

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
