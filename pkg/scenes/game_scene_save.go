package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
)

// ============================================================================
// 存档保存相关方法
// Story 15.5: 从 game_scene_init.go 拆分出存档保存逻辑（约150行）
// ============================================================================

// saveBattleState 保存当前战斗状态
func (s *GameScene) saveBattleState() {
	// 获取当前用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot save battle state")
		return
	}

	// 获取 gdata Manager
	gdataManager := s.gameState.GetGdataManager()
	if gdataManager == nil {
		log.Printf("[GameScene] Warning: gdata Manager not available, cannot save battle state")
		return
	}

	// 创建序列化器并保存
	serializer := game.NewBattleSerializer(gdataManager)
	if err := serializer.SaveBattle(s.entityManager, s.gameState, currentUser); err != nil {
		log.Printf("[GameScene] ERROR: Failed to save battle state: %v", err)
		return
	}

	log.Printf("[GameScene] Battle state saved successfully for user: %s", currentUser)
}

// deleteBattleSave 删除当前用户的战斗存档
func (s *GameScene) deleteBattleSave() {
	// 获取当前用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot delete battle save")
		return
	}

	// 删除存档
	if err := saveManager.DeleteBattleSave(currentUser); err != nil {
		log.Printf("[GameScene] ERROR: Failed to delete battle save: %v", err)
		return
	}

	log.Printf("[GameScene] Battle save deleted for user: %s", currentUser)
}

// showBattleSaveDialog 显示战斗存档选择对话框（继续/重玩/取消）
func (s *GameScene) showBattleSaveDialog() {
	log.Printf("[GameScene] 检测到战斗存档，立即恢复场景数据")

	// 1. 立即恢复存档数据（场景完整显示）
	s.restoreBattleState()
	s.skipOpeningAnimation()

	// 2. 立即处理一次动画命令（让实体能正确渲染），但不推进动画帧
	// 使用 deltaTime=0 确保动画数据初始化，但保持静止状态
	if s.reanimSystem != nil {
		s.reanimSystem.Update(0)
	}

	log.Printf("[GameScene] 场景数据已恢复，显示对话框")

	// 3. 显示对话框让玩家选择
	dialogEntity, err := entities.NewContinueGameDialogEntity(
		s.entityManager,
		s.resourceManager,
		s.battleSaveInfo,
		WindowWidth,
		WindowHeight,
		func() {
			log.Printf("[GameScene] 用户选择继续游戏，删除存档并开始")
			s.battleSaveDialogID = 0
			s.deleteBattleSave()
			s.createOpeningDaveDialogueIfNeeded()
		},
		func() {
			log.Printf("[GameScene] 用户选择重玩关卡，重新创建场景")
			s.battleSaveDialogID = 0
			s.deleteBattleSave()
			currentLevelID := "1-1"
			if s.gameState.CurrentLevel != nil {
				currentLevelID = s.gameState.CurrentLevel.ID
			}
			s.sceneManager.SwitchTo(NewGameScene(s.resourceManager, s.sceneManager, currentLevelID))
		},
		func() {
			log.Printf("[GameScene] 用户选择取消，返回主菜单")
			s.battleSaveDialogID = 0
			s.sceneManager.SwitchTo(NewMainMenuScene(s.resourceManager, s.sceneManager))
		},
	)

	if err != nil {
		log.Printf("[GameScene] Warning: Failed to create continue game dialog: %v", err)
		return
	}

	s.battleSaveDialogID = dialogEntity
	log.Printf("[GameScene] 继续游戏对话框已显示 (对话框ID: %d)", dialogEntity)
}
