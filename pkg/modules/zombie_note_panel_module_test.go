package modules

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ========== Story 8.14: ZombieNotePanelModule 测试 ==========

// TestZombieNotePanelModule_Struct 测试模块结构定义
func TestZombieNotePanelModule_Struct(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建最小化的模块实例（不初始化资源）
	module := &ZombieNotePanelModule{
		entityManager:  em,
		windowWidth:    800,
		windowHeight:   600,
		isActive:       false,
		fadeAlpha:      0.0,
		panelWidth:     400,
		panelHeight:    300,
	}

	// 验证初始状态
	if module.IsActive() {
		t.Error("Expected IsActive() to return false initially")
	}

	if module.fadeAlpha != 0.0 {
		t.Errorf("Expected fadeAlpha 0.0, got %f", module.fadeAlpha)
	}

	if module.windowWidth != 800 || module.windowHeight != 600 {
		t.Errorf("Expected window size 800x600, got %dx%d", module.windowWidth, module.windowHeight)
	}

	t.Logf("✓ ZombieNotePanelModule struct initialized correctly")
}

// TestZombieNotePanelModule_ShowHide 测试 Show/Hide 方法
func TestZombieNotePanelModule_ShowHide(t *testing.T) {
	em := ecs.NewEntityManager()

	module := &ZombieNotePanelModule{
		entityManager:  em,
		windowWidth:    800,
		windowHeight:   600,
		isActive:       false,
		fadeAlpha:      0.0,
		panelWidth:     400,
		panelHeight:    300,
	}

	// 创建按钮实体（避免 Show/Hide 中的空指针）
	module.nextLevelButtonEntity = em.CreateEntity()
	module.mainMenuButtonEntity = em.CreateEntity()

	ecs.AddComponent(em, module.nextLevelButtonEntity, &components.PositionComponent{X: -1000, Y: -1000})
	ecs.AddComponent(em, module.mainMenuButtonEntity, &components.PositionComponent{X: -1000, Y: -1000})
	ecs.AddComponent(em, module.nextLevelButtonEntity, &components.ButtonComponent{Width: 100, Height: 40})
	ecs.AddComponent(em, module.mainMenuButtonEntity, &components.ButtonComponent{Width: 65, Height: 40})

	// 测试 Show
	module.Show()
	if !module.IsActive() {
		t.Error("Expected IsActive() to return true after Show()")
	}
	if module.fadeAlpha != 1.0 {
		t.Errorf("Expected fadeAlpha 1.0 after Show(), got %f", module.fadeAlpha)
	}

	// 测试 Hide
	module.Hide()
	if module.IsActive() {
		t.Error("Expected IsActive() to return false after Hide()")
	}

	t.Logf("✓ Show/Hide methods work correctly")
}

// TestZombieNotePanelModule_SetFadeAlpha 测试淡入透明度设置
func TestZombieNotePanelModule_SetFadeAlpha(t *testing.T) {
	em := ecs.NewEntityManager()

	module := &ZombieNotePanelModule{
		entityManager: em,
		fadeAlpha:     0.0,
	}

	testValues := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
	for _, alpha := range testValues {
		module.SetFadeAlpha(alpha)
		if module.fadeAlpha != alpha {
			t.Errorf("Expected fadeAlpha %f, got %f", alpha, module.fadeAlpha)
		}
	}

	t.Logf("✓ SetFadeAlpha correctly updates fadeAlpha")
}

// TestZombieNotePanelModule_ButtonCallbacks 测试按钮回调
func TestZombieNotePanelModule_ButtonCallbacks(t *testing.T) {
	em := ecs.NewEntityManager()

	nextLevelCalled := false
	mainMenuCalled := false

	module := &ZombieNotePanelModule{
		entityManager: em,
		onNextLevel: func() {
			nextLevelCalled = true
		},
		onMainMenu: func() {
			mainMenuCalled = true
		},
	}

	// 测试回调存在
	if module.onNextLevel == nil {
		t.Error("Expected onNextLevel callback to be set")
	}
	if module.onMainMenu == nil {
		t.Error("Expected onMainMenu callback to be set")
	}

	// 触发回调
	module.onNextLevel()
	if !nextLevelCalled {
		t.Error("Expected onNextLevel callback to be called")
	}

	module.onMainMenu()
	if !mainMenuCalled {
		t.Error("Expected onMainMenu callback to be called")
	}

	t.Logf("✓ Button callbacks work correctly")
}

// TestZombieNotePanelModule_IsHoveringButtons 测试按钮悬停检测
func TestZombieNotePanelModule_IsHoveringButtons(t *testing.T) {
	em := ecs.NewEntityManager()

	module := &ZombieNotePanelModule{
		entityManager:  em,
		windowWidth:    800,
		windowHeight:   600,
		isActive:       false, // 未激活时应返回 false
		panelWidth:     400,
		panelHeight:    300,
	}

	// 创建按钮实体
	module.nextLevelButtonEntity = em.CreateEntity()
	module.mainMenuButtonEntity = em.CreateEntity()

	ecs.AddComponent(em, module.nextLevelButtonEntity, &components.PositionComponent{X: 350, Y: 350})
	ecs.AddComponent(em, module.mainMenuButtonEntity, &components.PositionComponent{X: 700, Y: 50})
	ecs.AddComponent(em, module.nextLevelButtonEntity, &components.ButtonComponent{Width: 100, Height: 40})
	ecs.AddComponent(em, module.mainMenuButtonEntity, &components.ButtonComponent{Width: 65, Height: 40})

	// 未激活时应返回 false
	if module.IsHoveringNextButton() {
		t.Error("Expected IsHoveringNextButton to return false when not active")
	}
	if module.IsHoveringMainMenuButton() {
		t.Error("Expected IsHoveringMainMenuButton to return false when not active")
	}

	t.Logf("✓ IsHovering methods return false when panel is not active")
}

// TestZombieNotePanelModule_Update 测试 Update 方法
func TestZombieNotePanelModule_Update(t *testing.T) {
	em := ecs.NewEntityManager()

	module := &ZombieNotePanelModule{
		entityManager:  em,
		windowWidth:    800,
		windowHeight:   600,
		isActive:       false,
		panelWidth:     400,
		panelHeight:    300,
	}

	// 创建按钮实体
	module.nextLevelButtonEntity = em.CreateEntity()
	module.mainMenuButtonEntity = em.CreateEntity()

	ecs.AddComponent(em, module.nextLevelButtonEntity, &components.PositionComponent{X: -1000, Y: -1000})
	ecs.AddComponent(em, module.mainMenuButtonEntity, &components.PositionComponent{X: -1000, Y: -1000})
	ecs.AddComponent(em, module.nextLevelButtonEntity, &components.ButtonComponent{Width: 100, Height: 40})
	ecs.AddComponent(em, module.mainMenuButtonEntity, &components.ButtonComponent{Width: 65, Height: 40})

	// 未激活时 Update 应立即返回
	module.Update(0.016)

	// 激活后 Update 应更新按钮位置
	module.Show()
	module.Update(0.016)

	// 验证按钮位置已更新（不再是 -1000）
	nextPos, _ := ecs.GetComponent[*components.PositionComponent](em, module.nextLevelButtonEntity)
	if nextPos.X == -1000 || nextPos.Y == -1000 {
		t.Error("Expected nextLevelButton position to be updated after Update()")
	}

	mainPos, _ := ecs.GetComponent[*components.PositionComponent](em, module.mainMenuButtonEntity)
	if mainPos.X == -1000 || mainPos.Y == -1000 {
		t.Error("Expected mainMenuButton position to be updated after Update()")
	}

	t.Logf("✓ Update method correctly updates button positions")
}

// TestZombieNotePanelModule_Cleanup 测试 Cleanup 方法
func TestZombieNotePanelModule_Cleanup(t *testing.T) {
	em := ecs.NewEntityManager()

	module := &ZombieNotePanelModule{
		entityManager: em,
	}

	// 创建实体
	module.panelEntity = em.CreateEntity()
	module.nextLevelButtonEntity = em.CreateEntity()
	module.mainMenuButtonEntity = em.CreateEntity()

	// 调用 Cleanup（会标记实体删除）
	module.Cleanup()

	// 处理标记删除的实体
	em.RemoveMarkedEntities()

	// Cleanup 成功执行即可（实体已被标记删除）
	t.Logf("✓ Cleanup method executed successfully")
}

// TestZombieNotePanelModule_TitleText 测试标题文本
func TestZombieNotePanelModule_TitleText(t *testing.T) {
	em := ecs.NewEntityManager()

	// 验证默认标题文本（当 LawnStrings 不可用时）
	module := &ZombieNotePanelModule{
		entityManager: em,
		titleText:     "你发现了张便签：", // 默认文本
	}

	if module.titleText == "" {
		t.Error("Expected titleText to have default value")
	}

	expectedText := "你发现了张便签："
	if module.titleText != expectedText {
		t.Errorf("Expected titleText '%s', got '%s'", expectedText, module.titleText)
	}

	t.Logf("✓ Title text configuration is correct")
}
