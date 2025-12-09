package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// createTestGuidedTutorialSystem 创建测试用的强引导教学系统
// 注意：不使用 ResourceManager，因为粒子效果测试需要真实资源
func createTestGuidedTutorialSystem() (*GuidedTutorialSystem, *ecs.EntityManager) {
	em := ecs.NewEntityManager()

	// 创建系统（不使用 ResourceManager，部分功能受限）
	system := &GuidedTutorialSystem{
		entityManager:   em,
		gameState:       nil, // 测试中不需要
		resourceManager: nil, // 测试中不需要（箭头显示功能受限）
		lastPlantCount:  0,
		initialized:     false,
		lastShovelMode:  false,
		totalTime:       0,
	}

	// 创建强引导教学实体
	system.guidedEntity = em.CreateEntity()
	guidedComp := &components.GuidedTutorialComponent{
		IsActive: false,
		AllowedActions: []string{
			"click_shovel",
			"click_plant",
			"click_screen",
		},
		IdleTimer:            0,
		IdleThreshold:        config.GuidedTutorialIdleThreshold,
		ShowArrow:            false,
		ArrowTarget:          "shovel",
		ArrowEntityID:        0,
		LastPlantCount:       0,
		TransitionReady:      false,
		OnTransitionCallback: nil,
	}
	em.AddComponent(system.guidedEntity, guidedComp)

	return system, em
}

// createTestPlant 创建测试植物实体
func createTestPlant(em *ecs.EntityManager, row, col int) ecs.EntityID {
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType: components.PlantPeashooter,
		GridRow:   row,
		GridCol:   col,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: float64(col) * 80,
		Y: float64(row) * 100,
	})
	return entityID
}

// TestGuidedTutorialSystem_IsOperationAllowed 测试白名单操作检查
func TestGuidedTutorialSystem_IsOperationAllowed(t *testing.T) {
	system, _ := createTestGuidedTutorialSystem()

	// 未激活时，所有操作都允许
	if !system.IsOperationAllowed("click_shovel") {
		t.Error("Expected click_shovel to be allowed when inactive")
	}
	if !system.IsOperationAllowed("click_plant_card") {
		t.Error("Expected click_plant_card to be allowed when inactive")
	}

	// 激活强引导模式
	system.SetActive(true)

	// 允许的操作
	if !system.IsOperationAllowed("click_shovel") {
		t.Error("Expected click_shovel to be allowed")
	}
	if !system.IsOperationAllowed("click_plant") {
		t.Error("Expected click_plant to be allowed")
	}
	if !system.IsOperationAllowed("click_screen") {
		t.Error("Expected click_screen to be allowed")
	}

	// 不允许的操作
	if system.IsOperationAllowed("click_plant_card") {
		t.Error("Expected click_plant_card to be blocked")
	}
	if system.IsOperationAllowed("click_menu") {
		t.Error("Expected click_menu to be blocked")
	}
	if system.IsOperationAllowed("click_lawn_empty") {
		t.Error("Expected click_lawn_empty to be blocked")
	}
}

// TestGuidedTutorialSystem_IdleTimer 测试空闲计时器逻辑
func TestGuidedTutorialSystem_IdleTimer(t *testing.T) {
	system, _ := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 获取组件
	guidedComp, _ := ecs.GetComponent[*components.GuidedTutorialComponent](system.entityManager, system.guidedEntity)

	// 初始状态
	if guidedComp.IdleTimer != 0 {
		t.Errorf("Expected initial IdleTimer to be 0, got %v", guidedComp.IdleTimer)
	}

	// 模拟时间流逝（每次 0.1 秒，共 51 次 = 5.1 秒，略大于阈值避免浮点精度问题）
	for i := 0; i < 51; i++ {
		system.Update(0.1)
	}

	// 检查空闲时间是否超过阈值（使用容差比较避免浮点精度问题）
	tolerance := 0.01
	if guidedComp.IdleTimer < guidedComp.IdleThreshold-tolerance {
		t.Errorf("Expected IdleTimer >= %.1f (with tolerance), got %v", guidedComp.IdleThreshold, guidedComp.IdleTimer)
	}

	// 注意：由于没有设置 ShovelSlotBoundsProvider，箭头不会显示
	// 但 ShowArrow 逻辑在 updateArrowDisplay 中会被调用
}

// TestGuidedTutorialSystem_ResetIdleTimer 测试空闲计时器重置
func TestGuidedTutorialSystem_ResetIdleTimer(t *testing.T) {
	system, _ := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 获取组件
	guidedComp, _ := ecs.GetComponent[*components.GuidedTutorialComponent](system.entityManager, system.guidedEntity)

	// 模拟时间流逝
	for i := 0; i < 30; i++ {
		system.Update(0.1) // 3 秒
	}

	if guidedComp.IdleTimer < 2.9 {
		t.Errorf("Expected IdleTimer >= 2.9, got %v", guidedComp.IdleTimer)
	}

	// 重置计时器
	system.ResetIdleTimer()

	if guidedComp.IdleTimer != 0 {
		t.Errorf("Expected IdleTimer to be reset to 0, got %v", guidedComp.IdleTimer)
	}
}

// TestGuidedTutorialSystem_NotifyOperation 测试操作通知
func TestGuidedTutorialSystem_NotifyOperation(t *testing.T) {
	system, _ := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 获取组件
	guidedComp, _ := ecs.GetComponent[*components.GuidedTutorialComponent](system.entityManager, system.guidedEntity)

	// 模拟时间流逝
	for i := 0; i < 30; i++ {
		system.Update(0.1) // 3 秒
	}

	initialTimer := guidedComp.IdleTimer

	// 通知有效操作
	system.NotifyOperation("click_shovel")

	// 计时器应该被重置
	if guidedComp.IdleTimer != 0 {
		t.Errorf("Expected IdleTimer to be reset after valid operation, got %v (was %v)", guidedComp.IdleTimer, initialTimer)
	}

	// 通知无效操作不应该重置计时器
	for i := 0; i < 20; i++ {
		system.Update(0.1) // 2 秒
	}
	timerBeforeInvalid := guidedComp.IdleTimer

	system.NotifyOperation("invalid_operation")

	// 无效操作不重置计时器
	if guidedComp.IdleTimer < timerBeforeInvalid {
		t.Error("Invalid operation should not reset idle timer")
	}
}

// TestGuidedTutorialSystem_PlantCountMonitoring 测试植物数量监控
func TestGuidedTutorialSystem_PlantCountMonitoring(t *testing.T) {
	system, em := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 获取组件
	guidedComp, _ := ecs.GetComponent[*components.GuidedTutorialComponent](system.entityManager, system.guidedEntity)

	// 创建 3 个植物
	plant1 := createTestPlant(em, 2, 6)
	plant2 := createTestPlant(em, 3, 8)
	plant3 := createTestPlant(em, 4, 7)

	// 更新系统（初始化植物数量）
	system.Update(0.016)

	if system.lastPlantCount != 3 {
		t.Errorf("Expected lastPlantCount to be 3, got %d", system.lastPlantCount)
	}
	if guidedComp.TransitionReady {
		t.Error("Expected TransitionReady to be false when plants exist")
	}

	// 移除一个植物
	em.DestroyEntity(plant1)
	em.RemoveMarkedEntities()

	// 模拟时间流逝使空闲计时器增加
	for i := 0; i < 30; i++ {
		system.Update(0.1)
	}

	// 移除植物应该重置空闲计时器
	// 再移除一个植物
	em.DestroyEntity(plant2)
	em.RemoveMarkedEntities()
	system.Update(0.016)

	if system.lastPlantCount != 1 {
		t.Errorf("Expected lastPlantCount to be 1 after removing 2 plants, got %d", system.lastPlantCount)
	}

	// 移除最后一个植物
	em.DestroyEntity(plant3)
	em.RemoveMarkedEntities()
	system.Update(0.016)

	if system.lastPlantCount != 0 {
		t.Errorf("Expected lastPlantCount to be 0, got %d", system.lastPlantCount)
	}
	if !guidedComp.TransitionReady {
		t.Error("Expected TransitionReady to be true when all plants removed")
	}
}

// TestGuidedTutorialSystem_TransitionCallback 测试转场回调
func TestGuidedTutorialSystem_TransitionCallback(t *testing.T) {
	system, em := createTestGuidedTutorialSystem()

	callbackCalled := false
	system.SetTransitionCallback(func() {
		callbackCalled = true
	})
	system.SetActive(true)

	// 创建并移除植物
	plant := createTestPlant(em, 2, 6)
	system.Update(0.016) // 初始化植物数量

	em.DestroyEntity(plant)
	em.RemoveMarkedEntities()
	system.Update(0.016) // 检测植物移除

	if !callbackCalled {
		t.Error("Expected transition callback to be called when all plants removed")
	}
}

// TestGuidedTutorialSystem_Activation 测试激活/停用
func TestGuidedTutorialSystem_Activation(t *testing.T) {
	system, _ := createTestGuidedTutorialSystem()

	// 初始状态：未激活
	if system.IsActive() {
		t.Error("Expected system to be inactive by default")
	}

	// 激活
	system.SetActive(true)
	if !system.IsActive() {
		t.Error("Expected system to be active after SetActive(true)")
	}

	// 获取组件验证状态重置
	guidedComp, _ := ecs.GetComponent[*components.GuidedTutorialComponent](system.entityManager, system.guidedEntity)
	if guidedComp.IdleTimer != 0 {
		t.Error("Expected IdleTimer to be reset on activation")
	}
	if guidedComp.TransitionReady {
		t.Error("Expected TransitionReady to be false on activation")
	}

	// 停用
	system.SetActive(false)
	if system.IsActive() {
		t.Error("Expected system to be inactive after SetActive(false)")
	}
}

// TestGuidedTutorialSystem_SetAllowedActions 测试设置允许的操作
func TestGuidedTutorialSystem_SetAllowedActions(t *testing.T) {
	system, _ := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 初始白名单
	if !system.IsOperationAllowed("click_shovel") {
		t.Error("Expected click_shovel to be allowed initially")
	}

	// 修改白名单（只允许 click_screen）
	system.SetAllowedActions([]string{"click_screen"})

	// 验证新白名单
	if !system.IsOperationAllowed("click_screen") {
		t.Error("Expected click_screen to be allowed after update")
	}
	if system.IsOperationAllowed("click_shovel") {
		t.Error("Expected click_shovel to be blocked after update")
	}
	if system.IsOperationAllowed("click_plant") {
		t.Error("Expected click_plant to be blocked after update")
	}
}

// TestGuidedTutorialSystem_IsTransitionReady 测试转场就绪状态
func TestGuidedTutorialSystem_IsTransitionReady(t *testing.T) {
	system, em := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 初始状态：未就绪
	if system.IsTransitionReady() {
		t.Error("Expected IsTransitionReady to be false initially")
	}

	// 创建植物
	plant := createTestPlant(em, 1, 1)
	system.Update(0.016)

	if system.IsTransitionReady() {
		t.Error("Expected IsTransitionReady to be false with plants")
	}

	// 移除所有植物
	em.DestroyEntity(plant)
	em.RemoveMarkedEntities()
	system.Update(0.016)

	if !system.IsTransitionReady() {
		t.Error("Expected IsTransitionReady to be true after all plants removed")
	}
}

// TestGuidedTutorialSystem_GetPlantCount 测试获取植物数量
func TestGuidedTutorialSystem_GetPlantCount(t *testing.T) {
	system, em := createTestGuidedTutorialSystem()
	system.SetActive(true)

	// 初始状态：0 个植物
	system.Update(0.016)
	if system.GetPlantCount() != 0 {
		t.Errorf("Expected 0 plants initially, got %d", system.GetPlantCount())
	}

	// 创建 2 个植物
	plant1 := createTestPlant(em, 1, 1)
	plant2 := createTestPlant(em, 2, 2)
	system.Update(0.016)

	if system.GetPlantCount() != 2 {
		t.Errorf("Expected 2 plants, got %d", system.GetPlantCount())
	}

	// 移除 1 个植物
	em.DestroyEntity(plant1)
	em.RemoveMarkedEntities()
	system.Update(0.016)

	if system.GetPlantCount() != 1 {
		t.Errorf("Expected 1 plant after removal, got %d", system.GetPlantCount())
	}

	// 清理
	em.DestroyEntity(plant2)
}

// TestIsGuidedTutorialBlocking 测试全局阻止函数
func TestIsGuidedTutorialBlocking(t *testing.T) {
	// 没有设置提供者时，不阻止任何操作
	guidedTutorialStateProvider = nil
	if IsGuidedTutorialBlocking("click_plant_card") {
		t.Error("Expected no blocking when provider is nil")
	}

	// 设置 mock 提供者
	mockProvider := &mockGuidedTutorialStateProvider{
		isActive:       true,
		allowedActions: map[string]bool{"click_shovel": true},
	}
	guidedTutorialStateProvider = mockProvider

	// 测试阻止
	if !IsGuidedTutorialBlocking("click_plant_card") {
		t.Error("Expected click_plant_card to be blocked")
	}
	if IsGuidedTutorialBlocking("click_shovel") {
		t.Error("Expected click_shovel to not be blocked")
	}

	// 未激活时不阻止
	mockProvider.isActive = false
	if IsGuidedTutorialBlocking("click_plant_card") {
		t.Error("Expected no blocking when inactive")
	}

	// 清理
	guidedTutorialStateProvider = nil
}

// mockGuidedTutorialStateProvider 测试用的 mock 提供者
type mockGuidedTutorialStateProvider struct {
	isActive       bool
	allowedActions map[string]bool
	notifiedOps    []string
}

func (m *mockGuidedTutorialStateProvider) IsGuidedTutorialActive() bool {
	return m.isActive
}

func (m *mockGuidedTutorialStateProvider) IsOperationAllowed(operation string) bool {
	return m.allowedActions[operation]
}

func (m *mockGuidedTutorialStateProvider) NotifyOperation(operation string) {
	m.notifiedOps = append(m.notifiedOps, operation)
}
