package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// 开场动画常量
	OpeningIdleDuration      = 0.5  // Idle 状态持续时间（秒）
	OpeningShowZombieTime    = 2.0  // 展示僵尸时间（秒）
	OpeningCameraSpeed       = 300.0 // 镜头移动速度（像素/秒）
	OpeningZombiePreviewX    = 1200.0 // 僵尸预告位置X坐标
	OpeningCameraRightTarget = 800.0  // 镜头右移目标位置
)

// OpeningAnimationSystem 管理关卡开场动画流程。
// 负责镜头移动、僵尸预告展示、跳过功能等。
type OpeningAnimationSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	levelConfig     *config.LevelConfig
	cameraSystem    *CameraSystem
	openingEntity   ecs.EntityID
}

// NewOpeningAnimationSystem 创建开场动画系统。
// 如果关卡不需要开场动画，返回 nil。
func NewOpeningAnimationSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, levelConfig *config.LevelConfig, cameraSystem *CameraSystem) *OpeningAnimationSystem {
	// 检查是否需要开场动画
	if levelConfig.SkipOpening {
		log.Println("[OpeningAnimationSystem] SkipOpening=true, 不创建开场动画系统")
		return nil
	}

	if levelConfig.OpeningType == "tutorial" {
		log.Println("[OpeningAnimationSystem] Tutorial level, 不创建开场动画系统")
		return nil
	}

	if levelConfig.SpecialRules != "" {
		log.Printf("[OpeningAnimationSystem] Special rules level (%s), 不创建开场动画系统", levelConfig.SpecialRules)
		return nil
	}

	log.Println("[OpeningAnimationSystem] 创建开场动画系统")

	oas := &OpeningAnimationSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		levelConfig:     levelConfig,
		cameraSystem:    cameraSystem,
		openingEntity:   0,
	}

	// 创建开场动画实体
	oas.openingEntity = em.CreateEntity()
	ecs.AddComponent(em, oas.openingEntity, &components.OpeningAnimationComponent{
		State:          "idle",
		ElapsedTime:    0,
		ZombieEntities: []ecs.EntityID{},
		IsSkipped:      false,
		IsCompleted:    false,
	})

	return oas
}

// Update 更新开场动画系统。
func (oas *OpeningAnimationSystem) Update(dt float64) {
	if oas == nil {
		return
	}

	// 获取开场动画组件
	openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](oas.entityManager, oas.openingEntity)
	if !ok || openingComp.IsCompleted {
		return
	}

	// 检查快捷键跳过（ESC 或 Space）
	if oas.checkSkipInput() {
		oas.Skip()
		return
	}

	// 更新已用时间
	openingComp.ElapsedTime += dt

	// 状态机处理
	switch openingComp.State {
	case "idle":
		oas.updateIdleState(openingComp)
	case "cameraMoveRight":
		oas.updateCameraMoveRightState(openingComp)
	case "showZombies":
		oas.updateShowZombiesState(openingComp)
	case "cameraMoveLeft":
		oas.updateCameraMoveLeftState(openingComp)
	case "gameStart":
		oas.updateGameStartState(openingComp)
	}
}

// updateIdleState 处理 Idle 状态。
func (oas *OpeningAnimationSystem) updateIdleState(openingComp *components.OpeningAnimationComponent) {
	if openingComp.ElapsedTime >= OpeningIdleDuration {
		// 切换到镜头右移状态
		openingComp.State = "cameraMoveRight"
		openingComp.ElapsedTime = 0

		// 触发镜头右移
		oas.cameraSystem.MoveTo(OpeningCameraRightTarget, 0, OpeningCameraSpeed)

		log.Println("[OpeningAnimationSystem] State: idle → cameraMoveRight")
	}
}

// updateCameraMoveRightState 处理镜头右移状态。
func (oas *OpeningAnimationSystem) updateCameraMoveRightState(openingComp *components.OpeningAnimationComponent) {
	// 检查镜头是否移动完成
	if !oas.cameraSystem.IsAnimating() {
		// 切换到展示僵尸状态
		openingComp.State = "showZombies"
		openingComp.ElapsedTime = 0

		// 生成预告僵尸
		oas.spawnPreviewZombies(openingComp)

		log.Println("[OpeningAnimationSystem] State: cameraMoveRight → showZombies")
	}
}

// updateShowZombiesState 处理展示僵尸状态。
func (oas *OpeningAnimationSystem) updateShowZombiesState(openingComp *components.OpeningAnimationComponent) {
	// 等待一定时间展示僵尸
	if openingComp.ElapsedTime >= OpeningShowZombieTime {
		// 切换到镜头返回状态
		openingComp.State = "cameraMoveLeft"
		openingComp.ElapsedTime = 0

		// 触发镜头返回
		oas.cameraSystem.MoveTo(0, 0, OpeningCameraSpeed)

		log.Println("[OpeningAnimationSystem] State: showZombies → cameraMoveLeft")
	}
}

// updateCameraMoveLeftState 处理镜头返回状态。
func (oas *OpeningAnimationSystem) updateCameraMoveLeftState(openingComp *components.OpeningAnimationComponent) {
	// 检查镜头是否移动完成
	if !oas.cameraSystem.IsAnimating() {
		// 切换到游戏开始状态
		openingComp.State = "gameStart"
		openingComp.ElapsedTime = 0

		log.Println("[OpeningAnimationSystem] State: cameraMoveLeft → gameStart")
	}
}

// updateGameStartState 处理游戏开始状态。
func (oas *OpeningAnimationSystem) updateGameStartState(openingComp *components.OpeningAnimationComponent) {
	// 清理预告僵尸
	oas.clearPreviewZombies(openingComp)

	// 标记开场动画完成
	openingComp.IsCompleted = true

	// TODO: 启用其他游戏系统（WaveSpawnSystem, InputSystem 等）

	log.Println("[OpeningAnimationSystem] Opening animation completed, game starting!")
}

// spawnPreviewZombies 生成预告僵尸实体。
func (oas *OpeningAnimationSystem) spawnPreviewZombies(openingComp *components.OpeningAnimationComponent) {
	// 从关卡配置获取所有僵尸类型
	zombieTypes := oas.getUniqueZombieTypes()

	log.Printf("[OpeningAnimationSystem] Spawning %d preview zombies", len(zombieTypes))

	// 计算僵尸位置布局
	screenHeight := 600.0 // TODO: 从配置获取
	zombieCount := len(zombieTypes)
	if zombieCount == 0 {
		return
	}

	// 生成僵尸实体
	for i, zombieType := range zombieTypes {
		// 计算Y坐标（均匀分布）
		var y float64
		if zombieCount == 1 {
			y = screenHeight / 2
		} else {
			spacing := screenHeight / float64(zombieCount+1)
			y = spacing * float64(i+1)
		}

		// 创建僵尸实体
		zombieEntity := oas.entityManager.CreateEntity()

		// 添加位置组件
		ecs.AddComponent(oas.entityManager, zombieEntity, &components.PositionComponent{
			X: OpeningZombiePreviewX,
			Y: y,
		})

		// 添加行为组件（特殊的预告行为，不移动）
		ecs.AddComponent(oas.entityManager, zombieEntity, &components.BehaviorComponent{
			Type: oas.zombieTypeToBehaviorType(zombieType),
			// 其他字段根据需要设置
		})

		// TODO: 添加 ReanimComponent 播放 idle 动画
		// TODO: 添加 HealthComponent（可选）

		// 保存僵尸实体ID
		openingComp.ZombieEntities = append(openingComp.ZombieEntities, zombieEntity)

		log.Printf("[OpeningAnimationSystem] Spawned preview zombie: type=%s, x=%.0f, y=%.0f", zombieType, OpeningZombiePreviewX, y)
	}
}

// clearPreviewZombies 清理预告僵尸实体。
func (oas *OpeningAnimationSystem) clearPreviewZombies(openingComp *components.OpeningAnimationComponent) {
	for _, entityID := range openingComp.ZombieEntities {
		oas.entityManager.DestroyEntity(entityID)
	}
	openingComp.ZombieEntities = []ecs.EntityID{}

	log.Println("[OpeningAnimationSystem] Cleared all preview zombies")
}

// getUniqueZombieTypes 获取关卡中所有唯一的僵尸类型。
func (oas *OpeningAnimationSystem) getUniqueZombieTypes() []string {
	uniqueTypes := make(map[string]bool)
	var result []string

	for _, wave := range oas.levelConfig.Waves {
		for _, zombie := range wave.Zombies {
			if !uniqueTypes[zombie.Type] {
				uniqueTypes[zombie.Type] = true
				result = append(result, zombie.Type)
			}
		}
	}

	return result
}

// zombieTypeToBehaviorType 将僵尸类型字符串转换为 BehaviorType。
func (oas *OpeningAnimationSystem) zombieTypeToBehaviorType(zombieType string) components.BehaviorType {
	switch zombieType {
	case "basic":
		return components.BehaviorZombieBasic
	case "conehead":
		return components.BehaviorZombieConehead
	case "buckethead":
		return components.BehaviorZombieBuckethead
	default:
		return components.BehaviorZombieBasic
	}
}

// checkSkipInput 检查是否按下跳过快捷键。
func (oas *OpeningAnimationSystem) checkSkipInput() bool {
	return ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsKeyPressed(ebiten.KeySpace)
}

// Skip 跳过开场动画。
func (oas *OpeningAnimationSystem) Skip() {
	openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](oas.entityManager, oas.openingEntity)
	if !ok {
		return
	}

	log.Println("[OpeningAnimationSystem] Skipping opening animation")

	// 停止镜头动画
	oas.cameraSystem.StopAnimation()

	// 清理预告僵尸
	oas.clearPreviewZombies(openingComp)

	// 设置跳过标志
	openingComp.IsSkipped = true

	// 直接切换到游戏开始状态
	openingComp.State = "gameStart"
	openingComp.IsCompleted = true
}

// IsCompleted 返回开场动画是否已完成。
func (oas *OpeningAnimationSystem) IsCompleted() bool {
	if oas == nil {
		return true
	}

	openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](oas.entityManager, oas.openingEntity)
	if !ok {
		return true
	}

	return openingComp.IsCompleted
}
