package scenes

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// triggerZombieHandAnimation triggers the zombie hand rising animation and blocks interaction.
// Story 12.6 Task 2.3: Trigger zombie hand animation on Adventure button click
func (m *MainMenuScene) triggerZombieHandAnimation() {
	if m.zombieHandEntity == 0 {
		// Zombie hand entity not created, fall back to直接跳转
		log.Printf("[MainMenuScene] Warning: Zombie hand entity not found, skipping animation")
		m.onStartAdventureClicked()
		return
	}

	// Get ReanimComponent
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.zombieHandEntity)
	if !ok {
		log.Printf("[MainMenuScene] Warning: Zombie hand entity has no ReanimComponent, skipping animation")
		m.onStartAdventureClicked()
		return
	}

	// Debug: Print animation info
	log.Printf("[MainMenuScene] 🧟 Zombie hand animation info:")
	log.Printf("  - CurrentAnimations: %v", reanimComp.CurrentAnimations)
	log.Printf("  - AnimVisiblesMap keys: %v", getMapKeys(reanimComp.AnimVisiblesMap))
	if visibles, ok := reanimComp.AnimVisiblesMap["_root"]; ok {
		log.Printf("  - _root visibles length: %d", len(visibles))
		if len(visibles) > 0 {
			log.Printf("  - _root visibles first 5: %v", visibles[:min(5, len(visibles))])
		}
	}
	log.Printf("  - FPS: %.1f", reanimComp.AnimationFPS)
	log.Printf("  - IsLooping: %v", reanimComp.IsLooping)
	log.Printf("  - IsPaused: %v", reanimComp.IsPaused)
	log.Printf("  - IsFinished: %v", reanimComp.IsFinished)

	// Unpause the animation
	reanimComp.IsPaused = false
	reanimComp.CurrentFrame = 0       // Reset to first frame
	reanimComp.FrameAccumulator = 0.0 // Reset accumulator
	reanimComp.IsFinished = false     // Reset finished flag

	// ✅ 修复：重置所有动画的帧索引，确保动画能从头播放
	if reanimComp.AnimationFrameIndices != nil {
		for k := range reanimComp.AnimationFrameIndices {
			reanimComp.AnimationFrameIndices[k] = 0.0
		}
	}

	// Set menu state to block interaction
	log.Printf("[MainMenuScene] 🧟 Setting menuState from %d to %d", m.menuState, MainMenuStateZombieHandPlaying)
	m.menuState = MainMenuStateZombieHandPlaying
	m.pendingScene = "GameScene"
	log.Printf("[MainMenuScene] 🧟 menuState is now: %d", m.menuState)

	// Disable all buttons to prevent clicks during animation
	m.disableAllButtons()

	log.Printf("[MainMenuScene] Zombie hand animation started (FPS=%.1f, total frames≈25)",
		reanimComp.AnimationFPS)
}

// checkZombieHandAnimationFinished checks if zombie hand animation has finished and switches scene.
// Story 12.6 Task 2.4 & 2.5: Detect animation completion and switch to game scene
func (m *MainMenuScene) checkZombieHandAnimationFinished() {
	if m.zombieHandEntity == 0 {
		return
	}

	// Get ReanimComponent
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.zombieHandEntity)
	if !ok {
		log.Printf("[MainMenuScene] Warning: Zombie hand entity has no ReanimComponent")
		return
	}

	// Debug: Print animation state every frame
	log.Printf("[MainMenuScene] 🧟 Checking animation: CurrentFrame=%d, IsFinished=%v, IsPaused=%v",
		reanimComp.CurrentFrame, reanimComp.IsFinished, reanimComp.IsPaused)
	if reanimComp.AnimationFrameIndices != nil {
		if frameIdx, ok := reanimComp.AnimationFrameIndices["_root"]; ok {
			log.Printf("[MainMenuScene] 🧟   _root frame index: %.2f", frameIdx)
		}
	}

	// Check if animation finished
	if !reanimComp.IsFinished {
		return
	}

	// Animation finished, switch to game scene
	log.Printf("[MainMenuScene] Zombie hand animation finished, switching to game scene")

	// Story 8.6: Load level from save file or default to 1-1
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()

	// Story 12.1 Task 6: 首次点击"开始冒险吧"时，标记用户已开始游戏
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
