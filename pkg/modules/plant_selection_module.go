package modules

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// PlantSelectionModule 植物选择栏模块
// 封装所有与植物选择栏相关的功能，包括：
//   - 植物卡片实体的创建和管理
//   - 植物选择逻辑（选择、取消、确认）
//   - 卡片状态更新（冷却、可用性）
//   - 卡片渲染
//
// 设计原则：
//   - 高内聚：所有选卡功能封装在单一模块中
//   - 低耦合：通过清晰的接口与外部系统交互
//   - 可复用：支持在不同场景（游戏中、选卡界面、图鉴）使用
//
// 使用场景：
//   - GameScene: 游戏中的植物选择栏（当前）
//   - PlantSelectionScene: 关卡开始前的选卡界面（Epic 8）
//   - 图鉴界面、商店界面（未来扩展）
type PlantSelectionModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// 系统（内部管理）
	selectionSystem  *systems.PlantSelectionSystem  // 植物选择逻辑
	cardSystem       *systems.PlantCardSystem       // 卡片状态更新
	cardRenderSystem *systems.PlantCardRenderSystem // 卡片渲染

	// 卡片实体列表（用于清理）
	cardEntities []ecs.EntityID

	// 外部依赖
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	reanimSystem    *systems.ReanimSystem
}

// NewPlantSelectionModule 创建一个新的植物选择栏模块
//
// 参数:
//   - em: EntityManager 实例
//   - gs: GameState 实例（用于阳光判断、保存选择结果）
//   - rm: ResourceManager 实例（用于加载卡片资源）
//   - rs: ReanimSystem 实例（用于渲染植物预览）
//   - levelConfig: 当前关卡配置（用于获取可用植物列表）
//   - plantCardFont: 植物卡片阳光数字字体
//   - seedBankX, seedBankY: 植物选择栏位置
//
// 返回:
//   - *PlantSelectionModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// 注意：
//   - 此函数会自动创建 PlantSelectionComponent 实体
//   - 根据 levelConfig.AvailablePlants 创建所有卡片实体
//   - 初始化所有子系统
func NewPlantSelectionModule(
	em *ecs.EntityManager,
	gs *game.GameState,
	rm *game.ResourceManager,
	rs *systems.ReanimSystem,
	levelConfig *config.LevelConfig,
	plantCardFont *text.GoTextFace,
	seedBankX, seedBankY float64,
) (*PlantSelectionModule, error) {
	module := &PlantSelectionModule{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		reanimSystem:    rs,
		cardEntities:    make([]ecs.EntityID, 0),
	}

	// 1. 创建 PlantSelectionComponent 实体（存储选择状态）
	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{},
		MaxSlots:       6, // 默认最多6个植物槽位
		IsConfirmed:    false,
	})

	// 2. 初始化 PlantSelectionSystem（必须在创建卡片前初始化）
	module.selectionSystem = systems.NewPlantSelectionSystem(em, rm, gs, levelConfig)

	// 3. 根据关卡配置创建植物卡片
	if err := module.createPlantCards(levelConfig, seedBankX, seedBankY); err != nil {
		return nil, fmt.Errorf("failed to create plant cards: %w", err)
	}

	// 4. 初始化 PlantCardSystem（卡片状态更新）
	module.cardSystem = systems.NewPlantCardSystem(em, gs, rm)

	// 5. 初始化 PlantCardRenderSystem（卡片渲染）
	// 注意：系统内部会自动过滤奖励卡片（有 RewardCardComponent 标记的卡片）
	module.cardRenderSystem = systems.NewPlantCardRenderSystem(em, plantCardFont)

	log.Printf("[PlantSelectionModule] Initialized with %d plant cards", len(module.cardEntities))

	return module, nil
}

// createPlantCards 根据关卡配置创建植物卡片实体
// 内部方法，由 NewPlantSelectionModule 调用
func (m *PlantSelectionModule) createPlantCards(levelConfig *config.LevelConfig, seedBankX, seedBankY float64) error {
	// 植物类型映射（从字符串ID到组件类型）
	// 注意：只包含当前已实现的植物类型
	plantTypeMap := map[string]components.PlantType{
		"sunflower":  components.PlantSunflower,
		"peashooter": components.PlantPeashooter,
		"wallnut":    components.PlantWallnut,
		"cherrybomb": components.PlantCherryBomb,
		// TODO: 未来添加更多植物类型（Epic 8+）
		// "potatomine":    components.PlantPotatoMine,
		// "snowpea":       components.PlantSnowPea,
		// "chomper":       components.PlantChomper,
		// "repeater":      components.PlantRepeater,
		// "puffshroom":    components.PlantPuffShroom,
		// "sunshroom":     components.PlantSunShroom,
		// "fumeshroom":    components.PlantFumeShroom,
		// "gravebuster":   components.PlantGraveBuster,
		// "hypnoshroom":   components.PlantHypnoShroom,
		// "scaredyshroom": components.PlantScaredyshroom,
		// "iceshroom":     components.PlantIceShroom,
		// "doomshroom":    components.PlantDoomShroom,
	}

	// 获取本关可用植物列表
	availablePlants := levelConfig.AvailablePlants

	// Story 19.5: 保龄球模式（initialSun == 0）使用传送带，不创建植物卡片
	if levelConfig.InitialSun == 0 && len(availablePlants) == 0 {
		log.Printf("[PlantSelectionModule] Bowling mode: skipping plant card creation (uses conveyor belt)")
		return nil
	}

	if len(availablePlants) == 0 {
		// 如果未配置，使用默认植物列表（向日葵、豌豆射手、坚果墙、樱桃炸弹）
		availablePlants = []string{"sunflower", "peashooter", "wallnut", "cherrybomb"}
		log.Printf("[PlantSelectionModule] No available plants in level config, using defaults: %v", availablePlants)
	}

	// 卡片布局配置（与 GameScene 保持一致）
	const (
		PlantCardStartOffsetX = 84                    // 第一张卡片相对于 SeedBank 的 X 偏移量
		PlantCardOffsetY      = 8                     // 卡片相对于 SeedBank 的 Y 偏移量
		PlantCardSpacing      = 60                    // 卡片槽之间的间距
		PlantCardScale        = config.PlantCardScale // 卡片缩放因子（0.54）
	)

	firstCardX := seedBankX + PlantCardStartOffsetX
	cardY := seedBankY + PlantCardOffsetY

	// 创建所有可用植物的卡片
	for i, plantName := range availablePlants {
		plantType, ok := plantTypeMap[plantName]
		if !ok {
			log.Printf("[PlantSelectionModule] Warning: Unknown plant type '%s', skipping", plantName)
			continue
		}

		// 计算卡片位置（水平排列）
		cardX := firstCardX + float64(i)*PlantCardSpacing

		// 创建卡片实体
		cardEntity, err := entities.NewPlantCardEntity(
			m.entityManager,
			m.resourceManager,
			m.reanimSystem,
			plantType,
			cardX,
			cardY,
			PlantCardScale,
		)
		if err != nil {
			log.Printf("[PlantSelectionModule] Warning: Failed to create %s card: %v", plantName, err)
			continue // 继续创建其他卡片
		}

		// 记录卡片实体ID（用于清理）
		m.cardEntities = append(m.cardEntities, cardEntity)
	}

	if len(m.cardEntities) == 0 {
		return fmt.Errorf("no plant cards created")
	}

	return nil
}

// Update 更新所有子系统
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 调用顺序：
//  1. PlantSelectionSystem - 处理选择逻辑
//  2. PlantCardSystem - 更新卡片状态（冷却、可用性）
func (m *PlantSelectionModule) Update(deltaTime float64) {
	// 注意：PlantSelectionSystem.Update() 当前是空实现
	// 选择逻辑由辅助方法（SelectPlant/DeselectPlant）直接处理
	// 此调用预留给未来的动画或自动逻辑
	m.selectionSystem.Update(deltaTime)

	// 更新卡片状态（冷却时间递减、可用性判断）
	m.cardSystem.Update(deltaTime)
}

// Draw 渲染所有植物卡片到屏幕
// 参数:
//   - screen: 目标渲染屏幕
//
// 注意：
//   - 只渲染选择栏卡片，不渲染奖励卡片（renderRewardCards=false）
//   - 自动应用卡片缩放、冷却遮罩、禁用遮罩等效果
func (m *PlantSelectionModule) Draw(screen *ebiten.Image) {
	m.cardRenderSystem.Draw(screen)
}

// DrawWithOffset 渲染所有植物卡片到屏幕，支持 Y 轴偏移
// 用于植物选择栏滑入动画
// 参数:
//   - screen: 目标渲染屏幕
//   - yOffset: Y 轴偏移量（正值向下，负值向上）
func (m *PlantSelectionModule) DrawWithOffset(screen *ebiten.Image, yOffset float64) {
	m.cardRenderSystem.DrawWithOffset(screen, yOffset)
}

// GetSelectedPlants 获取当前已选择的植物列表
// 返回:
//   - []string: 已选择的植物ID列表（如 ["peashooter", "sunflower"]）
//
// 用途：
//   - 用于保存到 GameState
//   - 用于场景切换时传递选择结果
func (m *PlantSelectionModule) GetSelectedPlants() []string {
	return m.selectionSystem.GetSelectedPlants()
}

// SelectPlant 选择一株植物
// 参数:
//   - plantID: 要选择的植物ID（如 "peashooter"）
//
// 返回:
//   - error: 如果选择失败（槽位已满、植物未解锁等）
//
// 注意：
//   - 此方法直接委托给 PlantSelectionSystem
//   - 会自动检查槽位数量、解锁状态
func (m *PlantSelectionModule) SelectPlant(plantID string) error {
	return m.selectionSystem.SelectPlant(plantID)
}

// DeselectPlant 取消选择一株植物
// 参数:
//   - plantID: 要取消选择的植物ID
//
// 返回:
//   - error: 如果取消失败（植物未被选择）
func (m *PlantSelectionModule) DeselectPlant(plantID string) error {
	return m.selectionSystem.DeselectPlant(plantID)
}

// ConfirmSelection 确认植物选择（点击"开战"按钮时调用）
// 返回:
//   - error: 如果确认失败（至少需要选择1株植物）
//
// 注意：
//   - 此方法会将选中植物保存到 GameState
//   - 调用后，GetSelectedPlants() 可获取最终选择结果
func (m *PlantSelectionModule) ConfirmSelection() error {
	return m.selectionSystem.ConfirmSelection()
}

// Cleanup 清理模块资源
// 用途：
//   - 场景切换时清理所有卡片实体
//   - 避免内存泄漏
//
// 注意：
//   - 只清理卡片实体，不清理系统实例
//   - EntityManager 会自动处理组件清理
func (m *PlantSelectionModule) Cleanup() {
	for _, entityID := range m.cardEntities {
		m.entityManager.DestroyEntity(entityID)
	}
	m.entityManager.RemoveMarkedEntities()
	m.cardEntities = nil

	log.Printf("[PlantSelectionModule] Cleaned up plant cards")
}
