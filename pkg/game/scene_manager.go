package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// SceneFactory 场景工厂函数类型
// 用于创建指定ID的关卡场景，避免循环依赖
type SceneFactory func(levelID string) Scene

// MainMenuFactory 主菜单工厂函数类型
// 用于创建主菜单场景，避免循环依赖
type MainMenuFactory func() Scene

// SceneManager manages the game's high-level state by controlling which scene is active.
// It ensures only one scene's Update and Draw methods are called at any given time.
type SceneManager struct {
	currentScene          Scene
	sceneFactory          SceneFactory    // 场景工厂函数，用于创建新场景
	mainMenuFactory       MainMenuFactory // 主菜单工厂函数，用于返回主菜单
	pendingLevelID        string          // 待加载的关卡ID（延迟加载机制）
	hasPendingLevelChange bool            // 是否有待处理的关卡切换
	hasPendingMainMenu    bool            // 是否有待处理的主菜单切换
}

// NewSceneManager creates and returns a new SceneManager instance.
// The manager starts with no active scene; use SwitchTo to set the initial scene.
func NewSceneManager() *SceneManager {
	return &SceneManager{
		currentScene:          nil,
		sceneFactory:          nil,
		mainMenuFactory:       nil,
		pendingLevelID:        "",
		hasPendingLevelChange: false,
		hasPendingMainMenu:    false,
	}
}

// SetSceneFactory 设置场景工厂函数
func (sm *SceneManager) SetSceneFactory(factory SceneFactory) {
	sm.sceneFactory = factory
}

// SetMainMenuFactory 设置主菜单工厂函数
func (sm *SceneManager) SetMainMenuFactory(factory MainMenuFactory) {
	sm.mainMenuFactory = factory
}

// SwitchTo changes the active scene to the provided scene.
// The new scene's Update and Draw methods will be called on subsequent game loop iterations.
func (sm *SceneManager) SwitchTo(scene Scene) {
	sm.currentScene = scene
}

// GetCurrentScene 返回当前活动的场景
//
// Bug Fix: 用于游戏关闭时检查当前场景是否需要保存状态
//
// 返回：
//   - Scene: 当前场景，如果没有活动场景则返回 nil
func (sm *SceneManager) GetCurrentScene() Scene {
	return sm.currentScene
}

// LoadLevel 加载指定ID的关卡场景
// levelID: 关卡ID，如 "1-1", "1-2"
//
// 注意：此方法现在使用延迟加载机制，场景切换会在下一帧开始时执行，
// 避免在当前场景的 Update() 执行期间切换场景导致的状态混乱。
func (sm *SceneManager) LoadLevel(levelID string) {
	log.Printf("[SceneManager] 请求加载关卡: %s (延迟到下一帧执行)", levelID)
	sm.pendingLevelID = levelID
	sm.hasPendingLevelChange = true
}

// SwitchToMainMenu 切换到主菜单场景
//
// 注意：此方法使用延迟加载机制，场景切换会在下一帧开始时执行，
// 避免在当前场景的 Update() 执行期间切换场景导致的状态混乱。
func (sm *SceneManager) SwitchToMainMenu() {
	log.Printf("[SceneManager] 请求切换到主菜单 (延迟到下一帧执行)")
	sm.hasPendingMainMenu = true
}

// loadLevelImmediate 立即加载关卡（内部方法，仅在安全时机调用）
func (sm *SceneManager) loadLevelImmediate(levelID string) {
	log.Printf("[SceneManager] 执行关卡加载: %s", levelID)

	if sm.sceneFactory == nil {
		log.Printf("[SceneManager] 错误: SceneFactory 未设置")
		return
	}

	// 使用工厂函数创建新场景
	newScene := sm.sceneFactory(levelID)
	if newScene != nil {
		sm.SwitchTo(newScene)
		log.Printf("[SceneManager] 成功切换到关卡: %s", levelID)
	} else {
		log.Printf("[SceneManager] 错误: 无法创建关卡场景: %s", levelID)
	}
}

// loadMainMenuImmediate 立即加载主菜单（内部方法，仅在安全时机调用）
func (sm *SceneManager) loadMainMenuImmediate() {
	log.Printf("[SceneManager] 执行主菜单加载")

	if sm.mainMenuFactory == nil {
		log.Printf("[SceneManager] 错误: MainMenuFactory 未设置")
		return
	}

	// 使用工厂函数创建主菜单场景
	newScene := sm.mainMenuFactory()
	if newScene != nil {
		sm.SwitchTo(newScene)
		log.Printf("[SceneManager] 成功切换到主菜单")
	} else {
		log.Printf("[SceneManager] 错误: 无法创建主菜单场景")
	}
}

// Update updates the currently active scene.
// If no scene is active, this method does nothing.
// deltaTime is the time elapsed since the last update in seconds.
func (sm *SceneManager) Update(deltaTime float64) {
	// 在场景 Update 之前处理待加载的关卡
	// 这确保场景切换发生在帧边界，而不是在当前场景的 Update 中间
	if sm.hasPendingLevelChange {
		sm.hasPendingLevelChange = false
		levelID := sm.pendingLevelID
		sm.pendingLevelID = ""
		sm.loadLevelImmediate(levelID)
		// 新场景已加载，继续执行新场景的 Update
	}

	// 处理待加载的主菜单
	if sm.hasPendingMainMenu {
		sm.hasPendingMainMenu = false
		sm.loadMainMenuImmediate()
		// 新场景已加载，继续执行新场景的 Update
	}

	if sm.currentScene != nil {
		sm.currentScene.Update(deltaTime)
	}
}

// Draw renders the currently active scene to the provided screen.
// If no scene is active, this method does nothing.
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	// 如果有待切换的场景，跳过当前帧的渲染，避免闪现旧场景
	if sm.hasPendingLevelChange || sm.hasPendingMainMenu {
		return
	}
	if sm.currentScene != nil {
		sm.currentScene.Draw(screen)
	}
}
