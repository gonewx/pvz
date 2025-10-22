package entities

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// MockLawnmowerResourceLoader 用于测试的资源加载器
type MockLawnmowerResourceLoader struct{}

func (m *MockLawnmowerResourceLoader) LoadImage(path string) (*ebiten.Image, error) {
	// 返回测试图像
	return ebiten.NewImage(10, 10), nil
}

func (m *MockLawnmowerResourceLoader) GetReanimXML(name string) *reanim.ReanimXML {
	if name == "LawnMower" {
		return &reanim.ReanimXML{
			FPS: 24,
			Tracks: []reanim.Track{
				{Name: "LawnMower_body"},
			},
		}
	}
	return nil
}

func (m *MockLawnmowerResourceLoader) GetReanimPartImages(name string) map[string]*ebiten.Image {
	if name == "LawnMower" {
		return map[string]*ebiten.Image{
			"LawnMower_body": ebiten.NewImage(1, 1),
		}
	}
	return nil
}

// MockLawnmowerReanimSystem 用于测试的 Reanim 系统
type MockLawnmowerReanimSystem struct {
	PlayedAnimations map[ecs.EntityID]string
}

func (m *MockLawnmowerReanimSystem) PlayAnimation(entityID ecs.EntityID, animName string) error {
	if m.PlayedAnimations == nil {
		m.PlayedAnimations = make(map[ecs.EntityID]string)
	}
	m.PlayedAnimations[entityID] = animName
	return nil
}

func (m *MockLawnmowerReanimSystem) PlayAnimationNoLoop(entityID ecs.EntityID, animName string) error {
	if m.PlayedAnimations == nil {
		m.PlayedAnimations = make(map[ecs.EntityID]string)
	}
	m.PlayedAnimations[entityID] = animName
	return nil
}

func (m *MockLawnmowerReanimSystem) RenderToTexture(entityID ecs.EntityID, target *ebiten.Image) error {
	// Mock implementation for testing
	return nil
}

// TestNewLawnmowerEntity 测试除草车实体创建
func TestNewLawnmowerEntity(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &MockLawnmowerResourceLoader{}
	rs := &MockLawnmowerReanimSystem{}

	// 测试创建第3行的除草车
	lane := 3
	entityID, err := NewLawnmowerEntity(em, rm, rs, lane)

	// 验证创建成功
	if err != nil {
		t.Fatalf("Failed to create lawnmower entity: %v", err)
	}
	if entityID == 0 {
		t.Fatal("Expected non-zero entity ID")
	}

	// 验证位置组件
	pos, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		t.Fatal("Lawnmower entity should have PositionComponent")
	}
	expectedX := config.LawnmowerStartX
	expectedY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0
	if pos.X != expectedX {
		t.Errorf("Expected X=%.1f, got %.1f", expectedX, pos.X)
	}
	if pos.Y != expectedY {
		t.Errorf("Expected Y=%.1f, got %.1f", expectedY, pos.Y)
	}

	// 验证 ReanimComponent
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		t.Fatal("Lawnmower entity should have ReanimComponent")
	}
	if reanimComp.Reanim == nil {
		t.Error("ReanimComponent should have Reanim data")
	}
	if reanimComp.PartImages == nil {
		t.Error("ReanimComponent should have PartImages")
	}
	if !reanimComp.IsLooping {
		t.Error("Lawnmower animation should be looping")
	}

	// 验证除草车组件
	lawnmower, ok := ecs.GetComponent[*components.LawnmowerComponent](em, entityID)
	if !ok {
		t.Fatal("Lawnmower entity should have LawnmowerComponent")
	}
	if lawnmower.Lane != lane {
		t.Errorf("Expected Lane=%d, got %d", lane, lawnmower.Lane)
	}
	if lawnmower.IsTriggered {
		t.Error("Lawnmower should not be triggered initially")
	}
	if lawnmower.IsMoving {
		t.Error("Lawnmower should not be moving initially")
	}
	if lawnmower.Speed != config.LawnmowerSpeed {
		t.Errorf("Expected Speed=%.1f, got %.1f", config.LawnmowerSpeed, lawnmower.Speed)
	}

	// 验证动画初始化
	// LawnMower.reanim 包含两个动画：anim_normal（静止）和 anim_tricked（移动）
	if rs.PlayedAnimations[entityID] != "anim_normal" {
		t.Errorf("Expected anim_normal, got %s", rs.PlayedAnimations[entityID])
	}
}

// TestNewLawnmowerEntityAllLanes 测试创建所有行的除草车
func TestNewLawnmowerEntityAllLanes(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &MockLawnmowerResourceLoader{}
	rs := &MockLawnmowerReanimSystem{}

	// 创建所有5行的除草车
	for lane := 1; lane <= 5; lane++ {
		entityID, err := NewLawnmowerEntity(em, rm, rs, lane)
		if err != nil {
			t.Fatalf("Failed to create lawnmower for lane %d: %v", lane, err)
		}

		// 验证行号正确
		lawnmower, ok := ecs.GetComponent[*components.LawnmowerComponent](em, entityID)
		if !ok {
			t.Fatalf("Lane %d lawnmower should have LawnmowerComponent", lane)
		}
		if lawnmower.Lane != lane {
			t.Errorf("Lane %d: Expected Lane=%d, got %d", lane, lane, lawnmower.Lane)
		}
	}
}

// TestNewLawnmowerEntityInvalidLane 测试无效行号
func TestNewLawnmowerEntityInvalidLane(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &MockLawnmowerResourceLoader{}
	rs := &MockLawnmowerReanimSystem{}

	// 测试无效行号
	invalidLanes := []int{0, 6, -1, 10}
	for _, lane := range invalidLanes {
		_, err := NewLawnmowerEntity(em, rm, rs, lane)
		if err == nil {
			t.Errorf("Expected error for invalid lane %d, got nil", lane)
		}
	}
}

// TestNewLawnmowerEntityNilParameters 测试 nil 参数
func TestNewLawnmowerEntityNilParameters(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &MockLawnmowerResourceLoader{}
	rs := &MockLawnmowerReanimSystem{}

	// 测试 nil EntityManager
	_, err := NewLawnmowerEntity(nil, rm, rs, 3)
	if err == nil {
		t.Error("Expected error for nil EntityManager")
	}

	// 测试 nil ResourceLoader
	_, err = NewLawnmowerEntity(em, nil, rs, 3)
	if err == nil {
		t.Error("Expected error for nil ResourceLoader")
	}

	// 测试 nil ReanimSystem
	_, err = NewLawnmowerEntity(em, rm, nil, 3)
	if err == nil {
		t.Error("Expected error for nil ReanimSystem")
	}
}
