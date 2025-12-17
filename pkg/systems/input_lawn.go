package systems

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
)

// ========================================
// 草坪点击种植处理
// ========================================

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
// 使用工厂注册表机制，新增植物无需修改此方法
func (s *InputSystem) createPlantEntity(plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	// 将 PlantType 转换为配置 ID
	plantID := config.PlantTypeToID(plantType)
	if plantID == "" {
		return 0, fmt.Errorf("unknown plant type: %v", plantType)
	}

	// 从工厂注册表获取工厂函数
	factory, ok := entities.GetPlantFactory(plantID)
	if !ok {
		return 0, fmt.Errorf("no factory registered for plant: %s", plantID)
	}

	// 构造工厂依赖
	deps := entities.PlantFactoryDeps{
		EntityManager:  s.entityManager,
		ResourceLoader: s.resourceManager,
		GameState:      s.gameState,
		ReanimSystem:   s.reanimSystem,
	}

	// 调用工厂函数创建实体
	return factory(deps, col, row)
}

// getPlantCost 获取植物的阳光消耗
func (s *InputSystem) getPlantCost(plantType components.PlantType) int {
	return config.GetPlantSunCostByType(plantType)
}
