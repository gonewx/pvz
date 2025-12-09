package components

import (
	"testing"

	"github.com/gonewx/pvz/pkg/ecs"
)

// TestGuidedTutorialComponent_Initialization 测试组件初始化
func TestGuidedTutorialComponent_Initialization(t *testing.T) {
	comp := &GuidedTutorialComponent{
		IsActive:             false,
		AllowedActions:       []string{"click_shovel", "click_plant", "click_screen"},
		IdleTimer:            0,
		IdleThreshold:        5.0,
		ShowArrow:            false,
		ArrowTarget:          "shovel",
		ArrowEntityID:        0,
		LastPlantCount:       0,
		TransitionReady:      false,
		OnTransitionCallback: nil,
	}

	// 验证默认值
	if comp.IsActive {
		t.Error("Expected IsActive to be false by default")
	}
	if comp.IdleThreshold != 5.0 {
		t.Errorf("Expected IdleThreshold to be 5.0, got %v", comp.IdleThreshold)
	}
	if len(comp.AllowedActions) != 3 {
		t.Errorf("Expected 3 allowed actions, got %d", len(comp.AllowedActions))
	}
	if comp.ShowArrow {
		t.Error("Expected ShowArrow to be false by default")
	}
	if comp.TransitionReady {
		t.Error("Expected TransitionReady to be false by default")
	}
}

// TestGuidedTutorialComponent_AllowedActions 测试白名单操作
func TestGuidedTutorialComponent_AllowedActions(t *testing.T) {
	comp := &GuidedTutorialComponent{
		AllowedActions: []string{"click_shovel", "click_plant", "click_screen"},
	}

	// 检查白名单中的操作
	allowedOps := map[string]bool{
		"click_shovel": true,
		"click_plant":  true,
		"click_screen": true,
	}

	for _, action := range comp.AllowedActions {
		if !allowedOps[action] {
			t.Errorf("Unexpected action in AllowedActions: %s", action)
		}
	}

	// 验证白名单长度
	if len(comp.AllowedActions) != len(allowedOps) {
		t.Errorf("Expected %d actions, got %d", len(allowedOps), len(comp.AllowedActions))
	}
}

// TestGuidedTutorialComponent_TransitionCallback 测试转场回调
func TestGuidedTutorialComponent_TransitionCallback(t *testing.T) {
	callbackCalled := false
	comp := &GuidedTutorialComponent{
		OnTransitionCallback: func() {
			callbackCalled = true
		},
	}

	// 调用回调
	if comp.OnTransitionCallback != nil {
		comp.OnTransitionCallback()
	}

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}
}

// TestGuidedTutorialComponent_ArrowEntityID 测试箭头实体ID
func TestGuidedTutorialComponent_ArrowEntityID(t *testing.T) {
	comp := &GuidedTutorialComponent{
		ArrowEntityID: 0,
	}

	// 验证初始值为0
	if comp.ArrowEntityID != 0 {
		t.Errorf("Expected ArrowEntityID to be 0, got %d", comp.ArrowEntityID)
	}

	// 设置新值
	comp.ArrowEntityID = ecs.EntityID(123)
	if comp.ArrowEntityID != 123 {
		t.Errorf("Expected ArrowEntityID to be 123, got %d", comp.ArrowEntityID)
	}

	// 重置为0
	comp.ArrowEntityID = 0
	if comp.ArrowEntityID != 0 {
		t.Errorf("Expected ArrowEntityID to be reset to 0, got %d", comp.ArrowEntityID)
	}
}

// TestGuidedTutorialComponent_IdleTimer 测试空闲计时器
func TestGuidedTutorialComponent_IdleTimer(t *testing.T) {
	comp := &GuidedTutorialComponent{
		IdleTimer:     0,
		IdleThreshold: 5.0,
	}

	// 模拟时间流逝
	comp.IdleTimer += 1.0
	if comp.IdleTimer != 1.0 {
		t.Errorf("Expected IdleTimer to be 1.0, got %v", comp.IdleTimer)
	}

	// 检查是否超过阈值
	comp.IdleTimer = 5.0
	if comp.IdleTimer < comp.IdleThreshold {
		t.Error("Expected IdleTimer to reach threshold")
	}

	// 重置计时器
	comp.IdleTimer = 0
	if comp.IdleTimer != 0 {
		t.Errorf("Expected IdleTimer to be reset to 0, got %v", comp.IdleTimer)
	}
}

// TestGuidedTutorialComponent_StateTransitions 测试状态转换
func TestGuidedTutorialComponent_StateTransitions(t *testing.T) {
	comp := &GuidedTutorialComponent{
		IsActive:        false,
		ShowArrow:       false,
		TransitionReady: false,
		LastPlantCount:  3,
	}

	// 激活强引导模式
	comp.IsActive = true
	if !comp.IsActive {
		t.Error("Expected IsActive to be true after activation")
	}

	// 模拟显示箭头
	comp.ShowArrow = true
	comp.ArrowTarget = "shovel"
	if !comp.ShowArrow {
		t.Error("Expected ShowArrow to be true")
	}
	if comp.ArrowTarget != "shovel" {
		t.Errorf("Expected ArrowTarget to be 'shovel', got '%s'", comp.ArrowTarget)
	}

	// 模拟植物被移除
	comp.LastPlantCount = 2
	if comp.LastPlantCount != 2 {
		t.Errorf("Expected LastPlantCount to be 2, got %d", comp.LastPlantCount)
	}

	// 模拟所有植物被移除，触发转场
	comp.LastPlantCount = 0
	comp.TransitionReady = true
	if !comp.TransitionReady {
		t.Error("Expected TransitionReady to be true when no plants remain")
	}

	// 停用强引导模式
	comp.IsActive = false
	comp.ShowArrow = false
	if comp.IsActive {
		t.Error("Expected IsActive to be false after deactivation")
	}
}
