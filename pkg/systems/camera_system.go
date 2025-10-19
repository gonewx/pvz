package systems

import (
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

const (
	// 镜头移动范围限制（防止移出背景边界）
	CameraMinX = 0.0
	// CameraMaxX 根据背景宽度1400和窗口宽度800计算：1400 - 800 = 600
	CameraMaxX = 600.0
)

// CameraSystem 管理镜头移动和平滑动画。
// 负责将镜头从当前位置平滑移动到目标位置，支持多种缓动效果。
type CameraSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState
	cameraEntity  ecs.EntityID // 镜头实体ID
}

// NewCameraSystem 创建镜头控制系统。
func NewCameraSystem(em *ecs.EntityManager, gs *game.GameState) *CameraSystem {
	cs := &CameraSystem{
		entityManager: em,
		gameState:     gs,
		cameraEntity:  0,
	}

	// 创建镜头实体
	cs.cameraEntity = em.CreateEntity()
	ecs.AddComponent(em, cs.cameraEntity, &components.CameraComponent{
		TargetX:        0,
		TargetY:        0,
		AnimationSpeed: 300, // 默认速度 300 px/s
		IsAnimating:    false,
		EasingType:     "easeInOut",
		StartX:         0,
		TotalDistance:  0,
	})

	return cs
}

// Update 更新镜头系统，处理镜头移动动画。
func (cs *CameraSystem) Update(dt float64) {
	// 获取镜头组件
	cameraComp, ok := ecs.GetComponent[*components.CameraComponent](cs.entityManager, cs.cameraEntity)
	if !ok || !cameraComp.IsAnimating {
		return
	}

	// 计算当前位置到目标位置的距离
	currentX := cs.gameState.CameraX
	distanceToTarget := cameraComp.TargetX - currentX

	// 检查是否已到达目标（距离 < 5px 视为到达）
	if math.Abs(distanceToTarget) < 5.0 {
		// 到达目标，停止动画
		cs.gameState.CameraX = cameraComp.TargetX
		cameraComp.IsAnimating = false
		return
	}

	// 计算当前进度 (0.0 - 1.0)
	movedDistance := math.Abs(currentX - cameraComp.StartX)
	progress := 0.0
	if cameraComp.TotalDistance > 0 {
		progress = movedDistance / cameraComp.TotalDistance
	}

	// 应用缓动函数
	easedSpeed := cameraComp.AnimationSpeed
	switch cameraComp.EasingType {
	case "easeInOut":
		// 二次缓动：先加速后减速
		easingFactor := cs.easeInOutQuad(progress)
		// 使用缓动因子调整速度（在中间部分加速）
		easedSpeed = cameraComp.AnimationSpeed * (1.0 + easingFactor)
	case "easeOut":
		// 减速运动
		easingFactor := cs.easeOutQuad(1.0 - progress)
		easedSpeed = cameraComp.AnimationSpeed * easingFactor
	case "linear":
		// 线性运动，无需调整
	default:
		// 默认线性
	}

	// 计算移动方向
	direction := 1.0
	if distanceToTarget < 0 {
		direction = -1.0
	}

	// 计算本帧移动的距离
	moveDistance := easedSpeed * dt * direction

	// 更新镜头位置
	newX := currentX + moveDistance

	// 防止越过目标
	if direction > 0 && newX > cameraComp.TargetX {
		newX = cameraComp.TargetX
	} else if direction < 0 && newX < cameraComp.TargetX {
		newX = cameraComp.TargetX
	}

	// 范围限制
	newX = math.Max(CameraMinX, math.Min(CameraMaxX, newX))

	// 应用到 GameState
	cs.gameState.CameraX = newX
}

// MoveTo 移动镜头到目标位置。
// 参数:
//   - targetX: 目标X坐标（世界坐标）
//   - targetY: 目标Y坐标（世界坐标，当前未使用）
//   - speed: 移动速度（像素/秒）
func (cs *CameraSystem) MoveTo(targetX, targetY, speed float64) {
	cameraComp, ok := ecs.GetComponent[*components.CameraComponent](cs.entityManager, cs.cameraEntity)
	if !ok {
		return
	}

	// 设置目标位置和速度
	cameraComp.TargetX = targetX
	cameraComp.TargetY = targetY
	cameraComp.AnimationSpeed = speed
	cameraComp.IsAnimating = true

	// 记录起点和总距离
	cameraComp.StartX = cs.gameState.CameraX
	cameraComp.TotalDistance = math.Abs(targetX - cameraComp.StartX)
}

// StopAnimation 停止镜头动画，立即设置到目标位置。
func (cs *CameraSystem) StopAnimation() {
	cameraComp, ok := ecs.GetComponent[*components.CameraComponent](cs.entityManager, cs.cameraEntity)
	if !ok {
		return
	}

	// 停止动画
	cameraComp.IsAnimating = false

	// 立即设置到目标位置
	cs.gameState.CameraX = cameraComp.TargetX
}

// IsAnimating 返回镜头是否正在动画中。
func (cs *CameraSystem) IsAnimating() bool {
	cameraComp, ok := ecs.GetComponent[*components.CameraComponent](cs.entityManager, cs.cameraEntity)
	if !ok {
		return false
	}
	return cameraComp.IsAnimating
}

// easeInOutQuad 二次缓动函数（先加速后减速）。
func (cs *CameraSystem) easeInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// easeOutQuad 减速缓动函数。
func (cs *CameraSystem) easeOutQuad(t float64) float64 {
	return t * (2 - t)
}
