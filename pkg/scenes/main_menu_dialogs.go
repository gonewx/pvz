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

// showComingSoonDialog 显示"敬请期待"对话框
// 当玩家通关 1-5 后点击冒险模式按钮时显示此对话框
// 按钮功能：
//   - 确定：将当前关卡重置为 1-1，重新加载主菜单界面
//   - 取消：关闭对话框
func (m *MainMenuScene) showComingSoonDialog() {
	// Close existing dialog (if any)
	if m.currentDialog != 0 {
		m.entityManager.DestroyEntity(m.currentDialog)
		m.currentDialog = 0
	}

	// Create dialog with callback
	dialogEntity, err := entities.NewDialogEntityWithCallback(
		m.entityManager,
		m.resourceManager,
		"敬请期待",
		"后续关卡还未实现，点击确定可以从头开玩",
		[]string{"确定", "取消"},
		WindowWidth,
		WindowHeight,
		func(buttonIndex int) {
			log.Printf("[MainMenuScene] Coming soon dialog button clicked: %d", buttonIndex)
			if buttonIndex == 0 {
				// "确定" 按钮：重置关卡到 1-1，重新加载界面
				m.resetToFirstLevel()
			}
			// buttonIndex == 1 是"取消"按钮，对话框会自动关闭，无需额外操作
		},
	)

	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to create coming soon dialog: %v", err)
		return
	}

	m.currentDialog = dialogEntity
	log.Printf("[MainMenuScene] Coming soon dialog created")
}

// resetToFirstLevel 重置游戏进度到 1-1 并重新加载主菜单
func (m *MainMenuScene) resetToFirstLevel() {
	log.Printf("[MainMenuScene] Resetting game progress to level 1-1")

	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()

	// 重置最高关卡为空（相当于新开始）
	saveManager.ResetHighestLevel()

	// 保存存档
	if err := saveManager.Save(); err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to save after reset: %v", err)
	}

	// 更新主菜单场景的当前关卡显示
	m.currentLevel = "1-1"

	// 重新加载主菜单场景以刷新界面
	newScene := NewMainMenuScene(m.resourceManager, m.sceneManager)
	m.sceneManager.SwitchTo(newScene)

	log.Printf("[MainMenuScene] Game progress reset complete, reloading main menu")
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

	// Bug Fix: 优先使用战斗存档中的 LevelID
	// 如果有战斗存档，必须使用存档中的关卡ID，否则会导致关卡配置与存档数据不匹配
	levelToLoad := ""
	currentUser := saveManager.GetCurrentUser()
	if currentUser != "" && saveManager.HasBattleSave(currentUser) {
		if battleInfo, err := saveManager.GetBattleSaveInfo(currentUser); err == nil && battleInfo != nil {
			levelToLoad = battleInfo.LevelID
			log.Printf("[MainMenu] Found battle save for level %s, using it", levelToLoad)
		}
	}

	// 如果没有战斗存档，使用 GetNextLevelToPlay
	if levelToLoad == "" {
		levelToLoad = saveManager.GetNextLevelToPlay()
		log.Printf("[MainMenu] No battle save, loading next level: %s (highest completed: %s)",
			levelToLoad, saveManager.GetHighestLevel())
	}

	// Pass ResourceManager, SceneManager, and levelID to GameScene
	gameScene := NewGameScene(m.resourceManager, m.sceneManager, levelToLoad)
	m.sceneManager.SwitchTo(gameScene)
}
