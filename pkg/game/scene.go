package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene represents a game scene (e.g., main menu, gameplay, pause menu).
// Each scene has its own update and rendering logic.
type Scene interface {
	// Update updates the scene logic based on the elapsed time.
	// deltaTime is the time elapsed since the last update in seconds.
	Update(deltaTime float64)

	// Draw renders the scene to the provided screen.
	// screen is the target image where the scene should be drawn.
	Draw(screen *ebiten.Image)
}

// Saveable 是一个可选接口，用于支持场景在退出时保存状态
//
// Bug Fix: 支持游戏关闭时自动存档
//
// 实现此接口的场景会在以下时机被调用 SaveOnExit()：
//   - 游戏窗口关闭
//   - 用户通过 OS 命令关闭程序
type Saveable interface {
	// SaveOnExit 在场景退出时保存状态
	// 返回 true 表示保存成功或无需保存
	// 返回 false 表示保存失败（但程序仍会正常退出）
	SaveOnExit() bool
}
