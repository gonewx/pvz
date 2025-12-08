package systems

import (
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/types"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	// 开场动画常量
	OpeningIdleDuration      = 0.5   // Idle 状态持续时间（秒）
	OpeningShowZombieTime    = 2.0   // 展示僵尸时间（秒）
	OpeningCameraSpeed       = 300.0 // 镜头移动速度（像素/秒）
	OpeningCameraRightTarget = 600.0 // 镜头右移目标位置（背景最右边）

	// ReadySetPlant 动画常量
	ReadySetPlantFPS      = 12   // 动画帧率
	ReadySetPlantDuration = 2.25 // 动画总时长（25帧 / 12FPS ≈ 2.08秒，加缓冲）
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
	usernameFont    *text.GoTextFace // 用户名字体（24号）
	skipKeyConsumed bool             // 跳过按键是否已被消费（防止同一帧 ESC 触发暂停）
}

// NewOpeningAnimationSystem 创建开场动画系统。
// 如果关卡不需要开场动画，返回 nil。
func NewOpeningAnimationSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, levelConfig *config.LevelConfig, cameraSystem *CameraSystem) *OpeningAnimationSystem {
	// 检查 levelConfig 是否为 nil
	if levelConfig == nil {
		log.Println("[OpeningAnimationSystem] levelConfig is nil, 不创建开场动画系统")
		return nil
	}

	// 检查是否需要开场动画
	if levelConfig.SkipOpening {
		log.Println("[OpeningAnimationSystem] SkipOpening=true, 不创建开场动画系统")
		return nil
	}

	// 迷你游戏关卡(如1-5保龄球, 1-10传送带)无开场动画
	if levelConfig.SpecialRules != "" {
		log.Printf("[OpeningAnimationSystem] Special rules level (%s), 不创建开场动画系统", levelConfig.SpecialRules)
		return nil
	}

	log.Printf("[OpeningAnimationSystem] 创建开场动画系统 (OpeningType: %s)", levelConfig.OpeningType)

	oas := &OpeningAnimationSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		levelConfig:     levelConfig,
		cameraSystem:    cameraSystem,
		openingEntity:   0,
		usernameFont:    nil,
		skipKeyConsumed: false,
	}

	// 加载用户名显示字体
	if rm != nil {
		font, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.OpeningUsernameFontSize)
		if err != nil {
			log.Printf("[OpeningAnimationSystem] 警告: 加载用户名字体失败: %v", err)
		} else {
			oas.usernameFont = font
			log.Printf("[OpeningAnimationSystem] 用户名字体加载成功 (size=%.0f)", config.OpeningUsernameFontSize)
		}
	}

	// 创建开场动画实体
	oas.openingEntity = em.CreateEntity()
	openingComp := &components.OpeningAnimationComponent{
		State:          "idle",
		ElapsedTime:    0,
		ZombieEntities: []ecs.EntityID{},
		IsSkipped:      false,
		IsCompleted:    false,
	}
	ecs.AddComponent(em, oas.openingEntity, openingComp)

	// Story 8.3.1: 在系统创建时立即生成预览僵尸（而不是等镜头移动到位）
	// 这样僵尸在开场一开始就存在，镜头移过去时玩家能看到它们
	oas.spawnPreviewZombies(openingComp)
	log.Printf("[OpeningAnimationSystem] 预览僵尸已在开场动画开始时生成")

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

	// 注意：开场动画期间不响应任何按键（ESC/Space）
	// 玩家必须等待动画完成后才能操作
	// 原因：跳过动画可能导致游戏流程异常

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

		// Story 8.3.1: 预览僵尸已在系统创建时生成，这里不需要再生成

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

		// 触发镜头返回到游戏位置（GameCameraX = 220）
		// 僵尸会自然移出屏幕右侧（符合设计）
		oas.cameraSystem.MoveTo(config.GameCameraX, 0, OpeningCameraSpeed)

		log.Println("[OpeningAnimationSystem] State: showZombies → cameraMoveLeft")
	}
}

// updateCameraMoveLeftState 处理镜头返回状态。
func (oas *OpeningAnimationSystem) updateCameraMoveLeftState(openingComp *components.OpeningAnimationComponent) {
	// 检查镜头是否移动完成
	if !oas.cameraSystem.IsAnimating() {
		// 直接切换到游戏开始状态
		// ReadySetPlant 动画将在铺草皮完成、UI 显示后由 GameScene 触发
		openingComp.State = "gameStart"
		openingComp.ElapsedTime = 0

		log.Println("[OpeningAnimationSystem] State: cameraMoveLeft → gameStart")
	}
}

// createReadySetPlantAnimation 创建 ReadySetPlant 动画实体。
func (oas *OpeningAnimationSystem) createReadySetPlantAnimation(openingComp *components.OpeningAnimationComponent) {
	if openingComp.ReadySetPlantStarted {
		return
	}

	// 获取 Reanim 资源
	reanimXML := oas.resourceManager.GetReanimXML("StartReadySetPlant")
	partImages := oas.resourceManager.GetReanimPartImages("StartReadySetPlant")
	if reanimXML == nil || partImages == nil {
		log.Println("[OpeningAnimationSystem] ⚠️ Failed to load StartReadySetPlant reanim resources")
		return
	}

	// 创建动画实体
	entity := oas.entityManager.CreateEntity()
	openingComp.ReadySetPlantEntity = entity
	openingComp.ReadySetPlantStarted = true

	// 添加位置组件（屏幕中心）
	// 屏幕尺寸：800x600，中心点 (400, 300)
	ecs.AddComponent(oas.entityManager, entity, &components.PositionComponent{
		X: config.ScreenWidth / 2,
		Y: config.ScreenHeight / 2,
	})

	// 添加 UI 组件（标记为 UI 元素，不受摄像机影响）
	ecs.AddComponent(oas.entityManager, entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 构建 MergedTracks（关键：这是动画数据的核心）
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 计算总帧数（简单动画的帧数 = 所有轨道中最大的帧数）
	totalFrames := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > totalFrames {
			totalFrames = len(track.Frames)
		}
	}

	// 初始化 AnimVisiblesMap：使用合成动画名 "_root"
	// 对于简单动画文件（没有配置文件定义的动画组合），使用 "_root" 作为动画名
	// prepareRenderCache 会识别 "_root" 并直接使用物理帧，无需映射
	animVisiblesMap := make(map[string][]int)
	visibles := make([]int, totalFrames)
	for i := range visibles {
		visibles[i] = 0 // 所有帧可见
	}
	animVisiblesMap["_root"] = visibles

	// 分析轨道类型（所有轨道都是可视轨道）
	var visualTracks, logicalTracks []string
	for _, track := range reanimXML.Tracks {
		visualTracks = append(visualTracks, track.Name)
	}

	// 添加 ReanimComponent
	reanimComp := &components.ReanimComponent{
		ReanimName:        "StartReadySetPlant",
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		MergedTracks:      mergedTracks,
		VisualTracks:      visualTracks,
		LogicalTracks:     logicalTracks,
		CurrentFrame:      0,
		FrameAccumulator:  0,
		AnimationFPS:      float64(reanimXML.FPS),
		CurrentAnimations: []string{"_root"}, // 使用合成动画名，简单动画直接播放所有帧
		AnimVisiblesMap:   animVisiblesMap,
		IsLooping:         false, // 单次播放
		IsFinished:        false,
		LastRenderFrame:   -1,
	}
	ecs.AddComponent(oas.entityManager, entity, reanimComp)

	// 播放音效
	oas.playReadySetPlantSound()

	log.Printf("[OpeningAnimationSystem] Created ReadySetPlant animation entity: %d (FPS=%d, Tracks=%d, Frames=%d)",
		entity, reanimXML.FPS, len(reanimXML.Tracks), totalFrames)
}

// clearReadySetPlantAnimation 清理 ReadySetPlant 动画实体。
func (oas *OpeningAnimationSystem) clearReadySetPlantAnimation(openingComp *components.OpeningAnimationComponent) {
	if openingComp.ReadySetPlantEntity != 0 {
		oas.entityManager.DestroyEntity(openingComp.ReadySetPlantEntity)
		openingComp.ReadySetPlantEntity = 0
		openingComp.ReadySetPlantStarted = false
		log.Println("[OpeningAnimationSystem] Cleared ReadySetPlant animation entity")
	}
}

// playReadySetPlantSound 播放 ReadySetPlant 音效。
func (oas *OpeningAnimationSystem) playReadySetPlantSound() {
	// 使用 AudioManager 统一管理音效（Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_READYSETPLANT")
		log.Println("[OpeningAnimationSystem] Playing ReadySetPlant sound effect")
	}
}

// updateGameStartState 处理游戏开始状态。
func (oas *OpeningAnimationSystem) updateGameStartState(openingComp *components.OpeningAnimationComponent) {
	// Story 8.3.1: 清理预览僵尸（镜头返回后销毁独立的预览实体）
	oas.clearPreviewZombies(openingComp)

	// 标记开场动画完成
	openingComp.IsCompleted = true

	log.Println("[OpeningAnimationSystem] Opening animation completed, game starting!")
}

// spawnPreviewZombies 生成预告僵尸实体。
// Story 8.3.1: 预览僵尸是独立的展示实体，用于开场动画中的"情报侦察"功能。
// 预览僵尸与实际关卡僵尸完全独立，不会参与游戏逻辑。
// 每种类型的预览僵尸数量不超过关卡配置中该类型的实际数量。
// 确保每种非普通僵尸类型至少显示一次（如果预览数量允许）。
// 优化：尽量避免僵尸站位重叠，分散到不同行和不同X位置。
func (oas *OpeningAnimationSystem) spawnPreviewZombies(openingComp *components.OpeningAnimationComponent) {
	// 获取每种僵尸类型在关卡配置中的数量
	zombieTypeCounts := oas.getZombieTypeCounts()

	if len(zombieTypeCounts) == 0 {
		log.Printf("[OpeningAnimationSystem] No zombie types found, skipping preview spawn")
		return
	}

	// 计算实际预览数量
	previewCount := oas.calculatePreviewZombieCount()

	// 初始化随机数种子（使用当前时间）
	rand.Seed(time.Now().UnixNano())

	// 构建预览类型列表，确保非普通类型优先显示
	// 1. 首先收集所有非 basic 类型（每种至少一个）
	var priorityTypes []string // 非普通类型（优先显示）
	var remainingPool []string // 剩余可用的僵尸池

	for zombieType, count := range zombieTypeCounts {
		if zombieType != "basic" {
			// 非普通类型：第一个加入优先列表，其余加入池
			priorityTypes = append(priorityTypes, zombieType)
			for i := 1; i < count; i++ {
				remainingPool = append(remainingPool, zombieType)
			}
		} else {
			// basic 类型全部加入池
			for i := 0; i < count; i++ {
				remainingPool = append(remainingPool, zombieType)
			}
		}
	}

	// 2. 打乱优先列表和剩余池
	rand.Shuffle(len(priorityTypes), func(i, j int) {
		priorityTypes[i], priorityTypes[j] = priorityTypes[j], priorityTypes[i]
	})
	rand.Shuffle(len(remainingPool), func(i, j int) {
		remainingPool[i], remainingPool[j] = remainingPool[j], remainingPool[i]
	})

	// 3. 构建最终预览列表：优先类型 + 剩余池
	var previewTypes []string
	previewTypes = append(previewTypes, priorityTypes...)
	previewTypes = append(previewTypes, remainingPool...)

	// 限制预览数量
	if previewCount > len(previewTypes) {
		previewCount = len(previewTypes)
	}

	log.Printf("[OpeningAnimationSystem] Spawning %d preview zombies (available types: %v, priority: %v)", previewCount, zombieTypeCounts, priorityTypes)

	// 生成指定数量的预览僵尸，完全随机分配行和位置
	for i := 0; i < previewCount; i++ {
		zombieType := previewTypes[i]

		// 完全随机选择行
		lane := rand.Intn(config.GridRows)

		// 计算Y坐标，加入随机垂直偏移（±12像素）
		baseY := config.GridWorldStartY + float64(lane)*config.CellHeight + config.CellHeight/2 + config.ZombieVerticalOffset
		yJitter := (rand.Float64() - 0.5) * 24
		y := baseY + yJitter

		// X坐标完全随机
		x := config.ZombieSpawnMinX + rand.Float64()*(config.ZombieSpawnMaxX-config.ZombieSpawnMinX)

		// 创建僵尸实体
		zombieEntity := oas.entityManager.CreateEntity()

		// 添加位置组件
		ecs.AddComponent(oas.entityManager, zombieEntity, &components.PositionComponent{
			X: x,
			Y: y,
		})

		// 添加行为组件（特殊的预告行为，不移动、不攻击）
		ecs.AddComponent(oas.entityManager, zombieEntity, &components.BehaviorComponent{
			Type: components.BehaviorZombiePreview, // 使用预览行为，防止僵尸移动
		})

		// 根据僵尸类型获取对应的 UnitID（用于显示正确的装备外观）
		unitID := types.ZombieTypeToUnitID(zombieType)

		// 添加 ReanimComponent 播放 idle 动画
		// 所有僵尸类型都使用基础 "Zombie" 动画资源，通过 UnitID 控制装备显示
		reanimXML := oas.resourceManager.GetReanimXML("Zombie")
		partImages := oas.resourceManager.GetReanimPartImages("Zombie")
		if reanimXML != nil && partImages != nil {
			// Story 8.3.1: 使用精简的初始化
			// 注意：MergedTracks 必须为 nil，让 PlayCombo 自动初始化轨道
			reanimComp := &components.ReanimComponent{
				ReanimName:        "Zombie",
				ReanimXML:         reanimXML,
				PartImages:        partImages,
				MergedTracks:      nil, // Story 8.3.1: 必须为 nil，让 PlayCombo 初始化轨道
				VisualTracks:      nil, // PlayCombo 会自动设置
				LogicalTracks:     nil, // PlayCombo 会自动设置
				CurrentFrame:      0,
				FrameAccumulator:  0,
				AnimationFPS:      12,
				CurrentAnimations: []string{},
				AnimVisiblesMap:   map[string][]int{},
				IsLooping:         true,
				IsFinished:        false,
				LastRenderFrame:   -1, // 确保首次渲染时触发缓存构建
			}
			ecs.AddComponent(oas.entityManager, zombieEntity, reanimComp)

			// 使用 AnimationCommand 组件播放配置的动画组合
			// 根据僵尸类型使用对应的 UnitID，以显示正确的装备（路障/铁桶等）
			ecs.AddComponent(oas.entityManager, zombieEntity, &components.AnimationCommandComponent{
				UnitID:    unitID,
				ComboName: "idle",
				Processed: false,
			})
		}

		// 保存僵尸实体ID
		openingComp.ZombieEntities = append(openingComp.ZombieEntities, zombieEntity)

		log.Printf("[OpeningAnimationSystem] Spawned preview zombie %d: type=%s, unitID=%s, lane=%d, x=%.0f, y=%.0f", i, zombieType, unitID, lane, x, y)
	}
}

// getZombieTypeCounts 获取关卡中每种僵尸类型的总数量。
// 返回一个 map，key 为僵尸类型，value 为该类型在所有波次中的总数量。
func (oas *OpeningAnimationSystem) getZombieTypeCounts() map[string]int {
	counts := make(map[string]int)

	for _, wave := range oas.levelConfig.Waves {
		// 新格式：ZombieGroup
		for _, zombie := range wave.Zombies {
			counts[zombie.Type] += zombie.Count
		}
		// 旧格式：ZombieSpawn (向后兼容)
		for _, zombie := range wave.OldZombies {
			counts[zombie.Type]++
		}
	}

	return counts
}

// calculatePreviewZombieCount 计算预览僵尸数量。
// Story 8.3.1: 如果配置了 PreviewZombieCount 则使用配置值，否则根据关卡难度自动计算：
//   - 简单关卡（≤2波）：3 只预览僵尸
//   - 中等关卡（3-5波）：5 只预览僵尸
//   - 困难关卡（>5波）：8 只预览僵尸
func (oas *OpeningAnimationSystem) calculatePreviewZombieCount() int {
	// 优先使用配置值
	if oas.levelConfig.PreviewZombieCount > 0 {
		return oas.levelConfig.PreviewZombieCount
	}

	// 根据波数自动计算
	waveCount := len(oas.levelConfig.Waves)
	switch {
	case waveCount <= 2:
		return 3 // 简单关卡
	case waveCount <= 5:
		return 5 // 中等关卡
	default:
		return 8 // 困难关卡
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
// Story 8.3.1: 支持新格式 ZombieGroup 和旧格式 ZombieSpawn (向后兼容)
func (oas *OpeningAnimationSystem) getUniqueZombieTypes() []string {
	uniqueTypes := make(map[string]bool)
	var result []string

	for _, wave := range oas.levelConfig.Waves {
		// 新格式：ZombieGroup
		for _, zombie := range wave.Zombies {
			if !uniqueTypes[zombie.Type] {
				uniqueTypes[zombie.Type] = true
				result = append(result, zombie.Type)
			}
		}
		// 旧格式：ZombieSpawn (向后兼容)
		for _, zombie := range wave.OldZombies {
			if !uniqueTypes[zombie.Type] {
				uniqueTypes[zombie.Type] = true
				result = append(result, zombie.Type)
			}
		}
	}

	return result
}

// checkSkipInput 检查是否按下跳过快捷键。
// 使用 IsKeyJustPressed 确保只在按键按下的瞬间触发一次
func (oas *OpeningAnimationSystem) checkSkipInput() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
}

// Skip 跳过开场动画。
func (oas *OpeningAnimationSystem) Skip() {
	openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](oas.entityManager, oas.openingEntity)
	if !ok {
		return
	}

	log.Println("[OpeningAnimationSystem] Skipping opening animation")

	// 标记跳过按键已被消费（防止同一帧 ESC 触发暂停）
	oas.skipKeyConsumed = true

	// 停止镜头动画
	oas.cameraSystem.StopAnimation()

	// Story 8.3.1: 清理预览僵尸（跳过时也需要清理）
	oas.clearPreviewZombies(openingComp)

	// 清理 ReadySetPlant 动画（如果正在播放）
	oas.clearReadySetPlantAnimation(openingComp)

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

// GetEntity 返回开场动画实体ID（用于调试和验证工具）
func (oas *OpeningAnimationSystem) GetEntity() ecs.EntityID {
	if oas == nil {
		return 0
	}
	return oas.openingEntity
}

// WasSkipKeyConsumed 检查跳过按键是否已被消费（用于防止同一帧 ESC 触发暂停）
// 调用此方法后会自动重置状态
func (oas *OpeningAnimationSystem) WasSkipKeyConsumed() bool {
	if oas == nil {
		return false
	}
	if oas.skipKeyConsumed {
		oas.skipKeyConsumed = false
		return true
	}
	return false
}

// Draw 绘制开场动画UI元素。
// 在开场动画期间，在屏幕下方居中显示 "{username}的房子" 文本。
// 文本为白色，带黑色阴影。
func (oas *OpeningAnimationSystem) Draw(screen *ebiten.Image) {
	if oas == nil || oas.usernameFont == nil {
		return
	}

	// 检查开场动画是否正在播放
	openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](oas.entityManager, oas.openingEntity)
	if !ok || openingComp.IsCompleted {
		return
	}

	// 获取当前用户名
	username := ""
	if oas.gameState != nil {
		sm := oas.gameState.GetSaveManager()
		if sm != nil {
			username = sm.GetCurrentUser()
		}
	}

	// 如果没有用户名，不显示
	if username == "" {
		return
	}

	// 构建显示文本
	displayText := username + "的房子"

	// 计算文本宽度，用于居中
	textWidth := text.Advance(displayText, oas.usernameFont)

	// 计算位置：屏幕下方居中
	screenWidth := config.ScreenWidth
	screenHeight := config.ScreenHeight
	x := (screenWidth - textWidth) / 2
	y := screenHeight - config.OpeningUsernameOffsetFromBottom

	// 阴影颜色（黑色）
	shadowColor := color.RGBA{0, 0, 0, 255}
	// 文本颜色（白色）
	textColor := color.RGBA{255, 255, 255, 255}

	// 绘制阴影（偏移）
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(
		x+config.OpeningUsernameShadowOffsetX,
		y+config.OpeningUsernameShadowOffsetY,
	)
	shadowOp.ColorScale.ScaleWithColor(shadowColor)
	text.Draw(screen, displayText, oas.usernameFont, shadowOp)

	// 绘制白色文本
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, displayText, oas.usernameFont, textOp)
}
