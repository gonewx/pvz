package entities

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestNewCrazyDaveEntity_Success 测试成功创建 Dave 实体
func TestNewCrazyDaveEntity_Success(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := newMockResourceManager()

	dialogueKeys := []string{"CRAZY_DAVE_2400", "CRAZY_DAVE_2401", "CRAZY_DAVE_2402"}
	callbackCalled := false
	onComplete := func() {
		callbackCalled = true
	}

	entityID, err := NewCrazyDaveEntity(em, rm, dialogueKeys, onComplete)
	if err != nil {
		t.Fatalf("NewCrazyDaveEntity failed: %v", err)
	}

	if entityID == 0 {
		t.Fatal("Expected non-zero entity ID")
	}

	// 验证 PositionComponent
	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		t.Fatal("Expected PositionComponent to be present")
	}

	if posComp.X != config.DaveEnterStartX {
		t.Errorf("Expected X position %f, got %f", config.DaveEnterStartX, posComp.X)
	}

	if posComp.Y != config.DaveTargetY {
		t.Errorf("Expected Y position %f, got %f", config.DaveTargetY, posComp.Y)
	}

	// 验证 DaveDialogueComponent
	dialogueComp, ok := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if !ok {
		t.Fatal("Expected DaveDialogueComponent to be present")
	}

	if len(dialogueComp.DialogueKeys) != 3 {
		t.Errorf("Expected 3 dialogue keys, got %d", len(dialogueComp.DialogueKeys))
	}

	if dialogueComp.State != components.DaveStateEntering {
		t.Errorf("Expected state Entering, got %v", dialogueComp.State)
	}

	if dialogueComp.IsVisible {
		t.Error("Expected IsVisible to be false initially")
	}

	// 验证回调
	if dialogueComp.OnCompleteCallback == nil {
		t.Error("Expected OnCompleteCallback to be set")
	}

	dialogueComp.OnCompleteCallback()
	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	// 验证 ReanimComponent
	_, hasReanimComp := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !hasReanimComp {
		t.Fatal("Expected ReanimComponent to be present")
	}

	// 验证 AnimationCommandComponent
	animCmd, hasAnimCmd := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !hasAnimCmd {
		t.Fatal("Expected AnimationCommandComponent to be present")
	}

	if animCmd.UnitID != "crazydave" {
		t.Errorf("Expected UnitID 'crazydave', got '%s'", animCmd.UnitID)
	}

	if animCmd.ComboName != "anim_enter" {
		t.Errorf("Expected ComboName 'anim_enter', got '%s'", animCmd.ComboName)
	}
}

// TestNewCrazyDaveEntity_NilEntityManager 测试空 EntityManager
func TestNewCrazyDaveEntity_NilEntityManager(t *testing.T) {
	rm := newMockResourceManager()
	dialogueKeys := []string{"TEST_KEY"}

	_, err := NewCrazyDaveEntity(nil, rm, dialogueKeys, nil)
	if err == nil {
		t.Error("Expected error for nil entity manager")
	}
}

// TestNewCrazyDaveEntity_NilResourceManager 测试空 ResourceManager
func TestNewCrazyDaveEntity_NilResourceManager(t *testing.T) {
	em := ecs.NewEntityManager()
	dialogueKeys := []string{"TEST_KEY"}

	_, err := NewCrazyDaveEntity(em, nil, dialogueKeys, nil)
	if err == nil {
		t.Error("Expected error for nil resource manager")
	}
}

// TestNewCrazyDaveEntity_EmptyDialogueKeys 测试空对话键列表
func TestNewCrazyDaveEntity_EmptyDialogueKeys(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := newMockResourceManager()

	_, err := NewCrazyDaveEntity(em, rm, []string{}, nil)
	if err == nil {
		t.Error("Expected error for empty dialogue keys")
	}
}

// TestNewCrazyDaveEntity_NilCallback 测试空回调（允许）
func TestNewCrazyDaveEntity_NilCallback(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := newMockResourceManager()
	dialogueKeys := []string{"TEST_KEY"}

	entityID, err := NewCrazyDaveEntity(em, rm, dialogueKeys, nil)
	if err != nil {
		t.Fatalf("NewCrazyDaveEntity failed with nil callback: %v", err)
	}

	if entityID == 0 {
		t.Fatal("Expected non-zero entity ID")
	}

	// 验证回调为 nil 是允许的
	dialogueComp, ok := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if !ok {
		t.Fatal("Expected DaveDialogueComponent to be present")
	}

	if dialogueComp.OnCompleteCallback != nil {
		t.Error("Expected OnCompleteCallback to be nil")
	}
}

// TestNewCrazyDaveEntity_BubbleOffset 测试气泡偏移配置
func TestNewCrazyDaveEntity_BubbleOffset(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := newMockResourceManager()
	dialogueKeys := []string{"TEST_KEY"}

	entityID, err := NewCrazyDaveEntity(em, rm, dialogueKeys, nil)
	if err != nil {
		t.Fatalf("NewCrazyDaveEntity failed: %v", err)
	}

	dialogueComp, ok := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if !ok {
		t.Fatal("Expected DaveDialogueComponent to be present")
	}

	if dialogueComp.BubbleOffsetX != config.DaveBubbleOffsetX {
		t.Errorf("Expected BubbleOffsetX %f, got %f", config.DaveBubbleOffsetX, dialogueComp.BubbleOffsetX)
	}

	if dialogueComp.BubbleOffsetY != config.DaveBubbleOffsetY {
		t.Errorf("Expected BubbleOffsetY %f, got %f", config.DaveBubbleOffsetY, dialogueComp.BubbleOffsetY)
	}
}
