package systems

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// PlantSelectionSystem 管理选卡界面的植物选择逻辑
// 负责处理植物选择、取消选择、验证选择合法性等功能
type PlantSelectionSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	gameState       *game.GameState
	levelConfig     *config.LevelConfig
}

// NewPlantSelectionSystem 创建一个新的植物选择系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
//   - gs: GameState 实例
//   - levelConfig: 当前关卡配置
//
// 返回:
//   - *PlantSelectionSystem: 新创建的植物选择系统实例
func NewPlantSelectionSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, levelConfig *config.LevelConfig) *PlantSelectionSystem {
	return &PlantSelectionSystem{
		entityManager:   em,
		resourceManager: rm,
		gameState:       gs,
		levelConfig:     levelConfig,
	}
}

// Update 更新植物选择系统（每帧调用）
// 参数:
//   - dt: 距离上一帧的时间间隔（秒）
//
// 注意: 当前实现为空，因为选择逻辑由辅助方法（SelectPlant/DeselectPlant）直接处理
// 未来可以在此添加UI动画更新等逻辑
func (s *PlantSelectionSystem) Update(dt float64) {
	// 查询 PlantSelectionComponent 实体
	// 当前实现中，选择逻辑由外部（InputSystem 或 UI 事件）调用辅助方法处理
	// 此方法预留给未来的动画或自动逻辑
}

// SelectPlant 选择一株植物
// 参数:
//   - plantID: 要选择的植物ID（如 "peashooter"）
//
// 返回:
//   - error: 如果选择失败（槽位已满、植物未解锁等），返回错误信息
func (s *PlantSelectionSystem) SelectPlant(plantID string) error {
	// 查询 PlantSelectionComponent 实体
	entities := ecs.GetEntitiesWith1[*components.PlantSelectionComponent](s.entityManager)

	if len(entities) == 0 {
		return fmt.Errorf("no plant selection component found")
	}

	// 获取第一个选卡组件（通常只有一个）
	selectionEntity := entities[0]
	selectionComp, ok := ecs.GetComponent[*components.PlantSelectionComponent](s.entityManager, selectionEntity)
	if !ok {
		return fmt.Errorf("failed to get plant selection component")
	}

	// 检查是否已选择
	for _, selected := range selectionComp.SelectedPlants {
		if selected == plantID {
			return fmt.Errorf("plant %s is already selected", plantID)
		}
	}

	// 检查槽位是否已满
	if len(selectionComp.SelectedPlants) >= selectionComp.MaxSlots {
		return fmt.Errorf("selection slots are full (max %d)", selectionComp.MaxSlots)
	}

	// 检查植物是否已解锁
	unlockManager := s.gameState.GetPlantUnlockManager()
	if unlockManager != nil && !unlockManager.IsUnlocked(plantID) {
		return fmt.Errorf("plant %s is not unlocked", plantID)
	}

	// 添加到选择列表
	selectionComp.SelectedPlants = append(selectionComp.SelectedPlants, plantID)

	return nil
}

// DeselectPlant 取消选择一株植物
// 参数:
//   - plantID: 要取消选择的植物ID
//
// 返回:
//   - error: 如果取消失败（植物未被选择），返回错误信息
func (s *PlantSelectionSystem) DeselectPlant(plantID string) error {
	// 查询 PlantSelectionComponent 实体
	entities := ecs.GetEntitiesWith1[*components.PlantSelectionComponent](s.entityManager)

	if len(entities) == 0 {
		return fmt.Errorf("no plant selection component found")
	}

	selectionEntity := entities[0]
	selectionComp, ok := ecs.GetComponent[*components.PlantSelectionComponent](s.entityManager, selectionEntity)
	if !ok {
		return fmt.Errorf("failed to get plant selection component")
	}

	// 查找并移除植物
	found := false
	newSelected := make([]string, 0, len(selectionComp.SelectedPlants)-1)
	for _, selected := range selectionComp.SelectedPlants {
		if selected != plantID {
			newSelected = append(newSelected, selected)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("plant %s is not selected", plantID)
	}

	selectionComp.SelectedPlants = newSelected

	return nil
}

// ConfirmSelection 确认植物选择（点击"开战"按钮时调用）
// 返回:
//   - error: 如果确认失败（至少需要选择1株植物），返回错误信息
func (s *PlantSelectionSystem) ConfirmSelection() error {
	// 查询 PlantSelectionComponent 实体
	entities := ecs.GetEntitiesWith1[*components.PlantSelectionComponent](s.entityManager)

	if len(entities) == 0 {
		return fmt.Errorf("no plant selection component found")
	}

	selectionEntity := entities[0]
	selectionComp, ok := ecs.GetComponent[*components.PlantSelectionComponent](s.entityManager, selectionEntity)
	if !ok {
		return fmt.Errorf("failed to get plant selection component")
	}

	// 验证至少选择了1株植物
	if len(selectionComp.SelectedPlants) < 1 {
		return fmt.Errorf("at least one plant must be selected")
	}

	// 设置确认标志
	selectionComp.IsConfirmed = true

	// 将选中植物保存到 GameState（供 GameScene 使用）
	s.gameState.SetSelectedPlants(selectionComp.SelectedPlants)

	return nil
}

// GetSelectedPlants 获取当前已选择的植物列表
// 返回:
//   - []string: 已选择的植物ID列表
func (s *PlantSelectionSystem) GetSelectedPlants() []string {
	// 查询 PlantSelectionComponent 实体
	entities := ecs.GetEntitiesWith1[*components.PlantSelectionComponent](s.entityManager)

	if len(entities) == 0 {
		return []string{}
	}

	selectionEntity := entities[0]
	selectionComp, ok := ecs.GetComponent[*components.PlantSelectionComponent](s.entityManager, selectionEntity)
	if !ok {
		return []string{}
	}

	// 返回副本，避免外部修改
	result := make([]string, len(selectionComp.SelectedPlants))
	copy(result, selectionComp.SelectedPlants)
	return result
}
