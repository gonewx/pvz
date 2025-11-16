package modules

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestPauseMenuModule_IsActive 测试 IsActive 方法
func TestPauseMenuModule_IsActive(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{}

	// 创建模块（最小化配置，不需要完整的资源管理器）
	module := &PauseMenuModule{
		entityManager:  em,
		gameState:      gs,
		buttonEntities: []ecs.EntityID{},
	}

	// 初始状态应该是 false（未激活）
	gs.IsPaused = false
	if module.IsActive() {
		t.Error("Expected IsActive() to return false when not paused")
	}

	// 设置为暂停
	gs.IsPaused = true
	if !module.IsActive() {
		t.Error("Expected IsActive() to return true when paused")
	}

	// 恢复
	gs.IsPaused = false
	if module.IsActive() {
		t.Error("Expected IsActive() to return false after resume")
	}
}

// TestPauseMenuModule_Show 测试 Show 方法
func TestPauseMenuModule_Show(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:  em,
		gameState:      gs,
		buttonEntities: []ecs.EntityID{},
		wasActive:      false,
	}

	// 调用 Show
	module.Show()

	// 验证状态
	if !gs.IsPaused {
		t.Error("Expected gameState.IsPaused to be true after Show()")
	}

	if !module.wasActive {
		t.Error("Expected module.wasActive to be true after Show()")
	}
}

// TestPauseMenuModule_Hide 测试 Hide 方法
func TestPauseMenuModule_Hide(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{IsPaused: true}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:  em,
		gameState:      gs,
		buttonEntities: []ecs.EntityID{},
		wasActive:      true,
	}

	// 调用 Hide
	module.Hide()

	// 验证状态
	if gs.IsPaused {
		t.Error("Expected gameState.IsPaused to be false after Hide()")
	}

	if module.wasActive {
		t.Error("Expected module.wasActive to be false after Hide()")
	}
}

// TestPauseMenuModule_Toggle 测试 Toggle 方法
func TestPauseMenuModule_Toggle(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{IsPaused: false}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:  em,
		gameState:      gs,
		buttonEntities: []ecs.EntityID{},
		wasActive:      false,
	}

	// 第一次切换：显示
	module.Toggle()
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true after first toggle")
	}
	if !module.wasActive {
		t.Error("Expected wasActive to be true after first toggle")
	}

	// 第二次切换：隐藏
	module.Toggle()
	if gs.IsPaused {
		t.Error("Expected IsPaused to be false after second toggle")
	}
	if module.wasActive {
		t.Error("Expected wasActive to be false after second toggle")
	}

	// 第三次切换：再次显示
	module.Toggle()
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true after third toggle")
	}
	if !module.wasActive {
		t.Error("Expected wasActive to be true after third toggle")
	}
}

// TestPauseMenuModule_Update_StateSync 测试 Update 方法的状态同步
func TestPauseMenuModule_Update_StateSync(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{IsPaused: false}

	// 创建暂停菜单实体和组件
	pauseMenuEntity := em.CreateEntity()
	ecs.AddComponent(em, pauseMenuEntity, &components.PauseMenuComponent{
		IsActive: false,
	})

	// 创建 SettingsPanelModule（模拟）
	settingsPanelModule := &SettingsPanelModule{
		entityManager:       em,
		settingsPanelEntity: pauseMenuEntity,
	}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:       em,
		gameState:           gs,
		settingsPanelModule: settingsPanelModule,
		buttonEntities:      []ecs.EntityID{},
		wasActive:           false,
	}

	// 场景1：外部状态变化（如 ESC 键）- 未激活 -> 激活
	gs.IsPaused = true // 外部修改
	module.Update(0.016)

	pauseMenu, _ := ecs.GetComponent[*components.PauseMenuComponent](em, pauseMenuEntity)
	if !pauseMenu.IsActive {
		t.Error("Expected PauseMenuComponent.IsActive to be true after external state change")
	}
	if !module.wasActive {
		t.Error("Expected module.wasActive to be synced after Update()")
	}

	// 场景2：外部状态变化 - 激活 -> 未激活
	gs.IsPaused = false // 外部修改
	module.Update(0.016)

	pauseMenu, _ = ecs.GetComponent[*components.PauseMenuComponent](em, pauseMenuEntity)
	if pauseMenu.IsActive {
		t.Error("Expected PauseMenuComponent.IsActive to be false after external state change")
	}
	if module.wasActive {
		t.Error("Expected module.wasActive to be false after Update()")
	}
}

// TestPauseMenuModule_Update_NoStateChange 测试 Update 在状态不变时不触发重复操作
func TestPauseMenuModule_Update_NoStateChange(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{IsPaused: true}

	// 创建暂停菜单实体和组件
	pauseMenuEntity := em.CreateEntity()
	ecs.AddComponent(em, pauseMenuEntity, &components.PauseMenuComponent{
		IsActive: true,
	})

	// 创建 SettingsPanelModule（模拟）
	settingsPanelModule := &SettingsPanelModule{
		entityManager:       em,
		settingsPanelEntity: pauseMenuEntity,
	}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:       em,
		gameState:           gs,
		settingsPanelModule: settingsPanelModule,
		buttonEntities:      []ecs.EntityID{},
		wasActive:           true, // 已经同步
	}

	// 多次调用 Update，状态不变
	for i := 0; i < 5; i++ {
		module.Update(0.016)

		// 验证状态仍然一致
		pauseMenu, _ := ecs.GetComponent[*components.PauseMenuComponent](em, pauseMenuEntity)
		if !pauseMenu.IsActive {
			t.Error("Expected PauseMenuComponent.IsActive to remain true")
		}
		if !gs.IsPaused {
			t.Error("Expected gameState.IsPaused to remain true")
		}
		if !module.wasActive {
			t.Error("Expected module.wasActive to remain true")
		}
	}
}

// TestPauseMenuModule_ShowAndUpdate 测试 Show 和 Update 的协作
func TestPauseMenuModule_ShowAndUpdate(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{IsPaused: false}

	// 创建暂停菜单实体和组件
	pauseMenuEntity := em.CreateEntity()
	ecs.AddComponent(em, pauseMenuEntity, &components.PauseMenuComponent{
		IsActive: false,
	})

	// 创建 SettingsPanelModule（模拟）
	settingsPanelModule := &SettingsPanelModule{
		entityManager:       em,
		settingsPanelEntity: pauseMenuEntity,
	}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:       em,
		gameState:           gs,
		settingsPanelModule: settingsPanelModule,
		buttonEntities:      []ecs.EntityID{},
		wasActive:           false,
	}

	// 调用 Show
	module.Show()

	// 验证状态立即更新
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true after Show()")
	}
	if !module.wasActive {
		t.Error("Expected wasActive to be true after Show()")
	}

	// 调用 Update（状态已同步，不应触发重复操作）
	module.Update(0.016)

	pauseMenu, _ := ecs.GetComponent[*components.PauseMenuComponent](em, pauseMenuEntity)
	if !pauseMenu.IsActive {
		t.Error("Expected PauseMenuComponent.IsActive to remain true after Update()")
	}
}

// TestPauseMenuModule_HideAndUpdate 测试 Hide 和 Update 的协作
func TestPauseMenuModule_HideAndUpdate(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := &game.GameState{IsPaused: true}

	// 创建暂停菜单实体和组件
	pauseMenuEntity := em.CreateEntity()
	ecs.AddComponent(em, pauseMenuEntity, &components.PauseMenuComponent{
		IsActive: true,
	})

	// 创建 SettingsPanelModule（模拟）
	settingsPanelModule := &SettingsPanelModule{
		entityManager:       em,
		settingsPanelEntity: pauseMenuEntity,
	}

	// 创建模块
	module := &PauseMenuModule{
		entityManager:       em,
		gameState:           gs,
		settingsPanelModule: settingsPanelModule,
		buttonEntities:      []ecs.EntityID{},
		wasActive:           true,
	}

	// 调用 Hide
	module.Hide()

	// 验证状态立即更新
	if gs.IsPaused {
		t.Error("Expected IsPaused to be false after Hide()")
	}
	if module.wasActive {
		t.Error("Expected wasActive to be false after Hide()")
	}

	// 调用 Update（状态已同步，不应触发重复操作）
	module.Update(0.016)

	pauseMenu, _ := ecs.GetComponent[*components.PauseMenuComponent](em, pauseMenuEntity)
	if pauseMenu.IsActive {
		t.Error("Expected PauseMenuComponent.IsActive to remain false after Update()")
	}
}
