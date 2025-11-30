package scenes

import (
	"log"
	"os"

	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// showUnlockDialog displays a dialog with a title and message
// Story 12.3: Dialog System Implementation
func (m *MainMenuScene) showUnlockDialog(title, message string) {
	// Close existing dialog (if any)
	if m.currentDialog != 0 {
		m.entityManager.DestroyEntity(m.currentDialog)
		m.currentDialog = 0
	}

	// Create new dialog
	dialogEntity, err := entities.NewDialogEntity(
		m.entityManager,
		m.resourceManager,
		title,
		message,
		[]string{"确定"},
		WindowWidth,
		WindowHeight,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to create dialog: %v", err)
		return
	}

	m.currentDialog = dialogEntity
	log.Printf("[MainMenuScene] Dialog created: %s - %s", title, message)
}

// showHelpDialog 显示帮助面板
// Story 12.3: 使用帮助面板模块（便笺背景 + 帮助文本）
func (m *MainMenuScene) showHelpDialog() {
	if m.helpPanelModule != nil {
		m.helpPanelModule.Show()
		log.Printf("[MainMenuScene] Help panel shown")
	}
}

// showOptionsDialog 显示选项面板
// Story 12.3: 使用选项面板模块（复用游戏场景的暂停菜单样式）
func (m *MainMenuScene) showOptionsDialog() {
	if m.optionsPanelModule != nil {
		m.optionsPanelModule.Show()
		log.Printf("[MainMenuScene] Options panel shown")
	}
}

// showErrorDialog 显示错误提示对话框
// 注意：错误对话框不会影响 currentDialog/currentUserDialogID 的跟踪
// 这样错误对话框关闭后，新用户对话框仍然保持打开状态
// Story 12.4: 防止错误对话框叠加 - 同一时间只能有一个错误对话框
func (m *MainMenuScene) showErrorDialog(title, message string) {
	// ✅ 如果已有错误对话框，先销毁旧的
	if m.currentErrorDialogID != 0 {
		log.Printf("[MainMenuScene] Destroying old error dialog (entity ID: %d)", m.currentErrorDialogID)
		// 如果 currentDialog 指向错误对话框，也清除
		if m.currentDialog == m.currentErrorDialogID {
			m.currentDialog = 0
		}
		m.entityManager.DestroyEntity(m.currentErrorDialogID)
		m.currentErrorDialogID = 0
	}

	dialogID, err := entities.NewDialogEntity(
		m.entityManager,
		m.resourceManager,
		title,
		message,
		[]string{"确定"},
		WindowWidth,
		WindowHeight,
	)

	if err != nil {
		log.Printf("[MainMenuScene] Error: Failed to create error dialog: %v", err)
		return
	}

	// ✅ 记录错误对话框ID，用于下次创建时销毁
	m.currentErrorDialogID = dialogID
	m.currentDialog = dialogID // 设置为当前对话框，触发背景交互阻止
	log.Printf("[MainMenuScene] Error dialog created (entity ID: %d)", dialogID)
}

// showBattleSaveDialog 显示战斗存档选择对话框
//
// Story 18.2: 战斗存档自动加载
//
// 当检测到未完成的战斗存档时，显示对话框让玩家选择：
//   - "继续游戏": 从存档恢复战斗状态
//   - "重新开始": 删除存档并开始新游戏
//
// 对话框显示信息：
//   - 关卡ID
//   - 波次进度
//   - 阳光数量
func (m *MainMenuScene) showBattleSaveDialog() {
	// 关闭现有对话框
	if m.currentDialog != 0 {
		m.entityManager.DestroyEntity(m.currentDialog)
		m.currentDialog = 0
	}

	// Story 18.3: 使用新的三按钮两行布局对话框
	dialogEntity, err := entities.NewContinueGameDialogEntity(
		m.entityManager,
		m.resourceManager,
		m.battleSaveInfo,
		WindowWidth,
		WindowHeight,
		// "继续"按钮回调
		func() {
			log.Printf("[MainMenuScene] 用户选择继续游戏，从存档加载")
			m.currentDialog = 0
			m.startGameFromBattleSave()
		},
		// "重玩关卡"按钮回调
		func() {
			log.Printf("[MainMenuScene] 用户选择重玩关卡，删除存档")
			m.currentDialog = 0
			m.deleteBattleSaveAndStartNew()
		},
		// "取消"按钮回调
		func() {
			log.Printf("[MainMenuScene] 用户选择取消，关闭对话框")
			m.currentDialog = 0
			// 关闭对话框后保持在主菜单，不进入游戏
		},
	)

	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to create continue game dialog: %v", err)
		// 如果创建对话框失败，直接进入游戏
		m.triggerZombieHandAnimation()
		return
	}

	m.currentDialog = dialogEntity
	log.Printf("[MainMenuScene] 继续游戏对话框已显示 (三按钮两行布局)")
}

// startGameFromBattleSave 从战斗存档开始游戏
//
// Story 18.2: 从存档恢复战斗
//
// 步骤：
//  1. 触发僵尸手动画
//  2. 动画完成后创建 GameScene，传入 fromBattleSave=true
func (m *MainMenuScene) startGameFromBattleSave() {
	// 设置标记，表示需要从存档加载
	m.pendingScene = "GameSceneFromSave"
	// 触发僵尸手动画
	m.triggerZombieHandAnimation()
}

// deleteBattleSaveAndStartNew 删除战斗存档并开始新游戏
//
// Story 18.2: 删除存档开始新游戏
//
// 步骤：
//  1. 删除当前用户的战斗存档
//  2. 清除本地存档状态
//  3. 触发僵尸手动画进入新游戏
func (m *MainMenuScene) deleteBattleSaveAndStartNew() {
	// 删除战斗存档
	currentUser := m.saveManager.GetCurrentUser()
	if currentUser != "" {
		if err := m.saveManager.DeleteBattleSave(currentUser); err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to delete battle save: %v", err)
		} else {
			log.Printf("[MainMenuScene] 战斗存档已删除: user=%s", currentUser)
		}
	}

	// 清除本地存档状态
	m.hasBattleSave = false
	m.battleSaveInfo = nil

	// 触发僵尸手动画进入新游戏
	m.triggerZombieHandAnimation()
}

// onExitClicked handles the "Exit Game" button click.
// It terminates the application.
func (m *MainMenuScene) onExitClicked() {
	log.Println("Exit Game button clicked")
	os.Exit(0)
}

// exitGame exits the game
func exitGame() {
	log.Println("Exiting game")
	os.Exit(0)
}

// getGameState returns the global game state
func getGameState() *game.GameState {
	return game.GetGameState()
}

// getMapKeys returns the keys of a map (helper for debugging)
func getMapKeys(m map[string][]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// onStartAdventureClicked handles the "Start Adventure" button click.
// It switches the current scene to the GameScene.
func (m *MainMenuScene) onStartAdventureClicked() {
	log.Println("Start Adventure button clicked")

	// Story 12.1 Task 6: 首次点击"开始冒险吧"时，标记用户已开始游戏
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	if err := saveManager.Load(); err == nil {
		if !saveManager.GetHasStartedGame() {
			log.Println("[MainMenuScene] 首次开始游戏，设置 hasStartedGame = true")
			saveManager.SetHasStartedGame()
			if err := saveManager.Save(); err != nil {
				log.Printf("[MainMenuScene] ⚠️ 保存 hasStartedGame 失败: %v", err)
			}
		}
	}

	// Story 8.6: Load level from save file or default to 1-1
	levelToLoad := "1-1" // Default to first level
	if err := saveManager.Load(); err == nil {
		// Save file exists, get highest level
		highestLevel := saveManager.GetHighestLevel()
		if highestLevel != "" {
			levelToLoad = highestLevel
			log.Printf("[MainMenu] Loading from save: highest level = %s", highestLevel)
		}
	}

	// Story 18.2: 检查是否需要从战斗存档加载
	if m.pendingScene == "GameSceneFromSave" {
		// 从存档加载，使用存档中的关卡ID
		if m.battleSaveInfo != nil {
			levelToLoad = m.battleSaveInfo.LevelID
			log.Printf("[MainMenuScene] 从战斗存档加载: level=%s", levelToLoad)
		}
		// 创建 GameScene 并标记需要从存档恢复
		gameScene := NewGameSceneFromBattleSave(m.resourceManager, m.sceneManager, levelToLoad)
		m.sceneManager.SwitchTo(gameScene)
		return
	}

	// Pass ResourceManager, SceneManager, and levelID to GameScene
	gameScene := NewGameScene(m.resourceManager, m.sceneManager, levelToLoad)
	m.sceneManager.SwitchTo(gameScene)
}
