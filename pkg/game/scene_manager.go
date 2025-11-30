package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// SceneFactory 场景工厂函数类型
// 用于创建指定ID的关卡场景，避免循环依赖
type SceneFactory func(levelID string) Scene

// SceneManager manages the game's high-level state by controlling which scene is active.
// It ensures only one scene's Update and Draw methods are called at any given time.
type SceneManager struct {
	currentScene Scene
	sceneFactory SceneFactory // 场景工厂函数，用于创建新场景
}

// NewSceneManager creates and returns a new SceneManager instance.
// The manager starts with no active scene; use SwitchTo to set the initial scene.
func NewSceneManager() *SceneManager {
	return &SceneManager{
		currentScene: nil,
		sceneFactory: nil,
	}
}

// SetSceneFactory 设置场景工厂函数
func (sm *SceneManager) SetSceneFactory(factory SceneFactory) {
	sm.sceneFactory = factory
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
func (sm *SceneManager) LoadLevel(levelID string) {
	log.Printf("[SceneManager] 加载关卡: %s", levelID)

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

// Update updates the currently active scene.
// If no scene is active, this method does nothing.
// deltaTime is the time elapsed since the last update in seconds.
func (sm *SceneManager) Update(deltaTime float64) {
	if sm.currentScene != nil {
		sm.currentScene.Update(deltaTime)
	}
}

// Draw renders the currently active scene to the provided screen.
// If no scene is active, this method does nothing.
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	if sm.currentScene != nil {
		sm.currentScene.Draw(screen)
	}
}
